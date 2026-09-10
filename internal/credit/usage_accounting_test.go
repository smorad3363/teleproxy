package credit

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"
)

func TestAccountObservedUsageIsOrderedAtomicAndIdempotent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	projectedAt := time.Date(2030, 3, 1, 12, 0, 0, 0, time.UTC)
	expiry := projectedAt.Add(time.Hour)
	early := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), ExpiresAt: &expiry, RewardType: "base", Source: "early"})
	later := grant(t, db, "alice", Grant{OriginalBytes: 200, StartsAt: projectedAt.Add(-time.Hour), RewardType: "admin", Source: "later"})

	snapshot := prepareAndActivate(t, db, "alice", PrepareProjectionInput{
		TelemtResetEpochSecs:    7,
		TelemtBaselineUsedBytes: 10,
		ProjectedAt:             projectedAt,
	})
	accounted, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 7, UsedBytes: 160})
	if err != nil {
		t.Fatal(err)
	}
	if accounted.AccountedBytes != 150 || accounted.PreviousUsedBytes != 10 || accounted.ObservedUsedBytes != 160 || accounted.Generation != snapshot.Generation {
		t.Fatalf("accounting = %#v", accounted)
	}
	wantAllocations := []Allocation{{BucketID: early.ID, Bytes: 100}, {BucketID: later.ID, Bytes: 50}}
	assertAllocations(t, accounted.Allocations, wantAllocations)

	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TelemtBaselineUsedBytes != 160 {
		t.Fatalf("baseline = %d, want 160", loaded.TelemtBaselineUsedBytes)
	}
	if len(loaded.Members) != 2 || loaded.Members[0].AccountedBytes != 100 || loaded.Members[1].AccountedBytes != 50 {
		t.Fatalf("member accounting = %#v", loaded.Members)
	}
	assertBucketConsumed(t, db, "alice", projectedAt, early.ID, 100)
	assertBucketConsumed(t, db, "alice", projectedAt, later.ID, 50)

	again, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 7, UsedBytes: 160})
	if err != nil {
		t.Fatal(err)
	}
	if again.AccountedBytes != 0 || len(again.Allocations) != 0 {
		t.Fatalf("idempotent accounting = %#v", again)
	}
	assertBucketConsumed(t, db, "alice", projectedAt, early.ID, 100)
	assertBucketConsumed(t, db, "alice", projectedAt, later.ID, 50)
}

func TestAccountObservedUsageRejectsInactiveResetMismatchAndRegression(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	projectedAt := time.Date(2030, 3, 2, 12, 0, 0, 0, time.UTC)
	bucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), RewardType: "base", Source: "grant"})
	snapshot, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{TelemtResetEpochSecs: 9, TelemtBaselineUsedBytes: 20, ProjectedAt: projectedAt})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 9, UsedBytes: 30}); !errors.Is(err, ErrReconciliationNotActive) {
		t.Fatalf("inactive accounting error = %v", err)
	}
	activateSnapshot(t, db, snapshot.ProxyUserID, snapshot.Generation)
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 10, UsedBytes: 30}); !errors.Is(err, ErrQuotaResetMismatch) {
		t.Fatalf("reset mismatch error = %v", err)
	}
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 9, UsedBytes: 19}); !errors.Is(err, ErrQuotaUsageRegressed) {
		t.Fatalf("usage regression error = %v", err)
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TelemtBaselineUsedBytes != 20 || loaded.Members[0].AccountedBytes != 0 {
		t.Fatalf("rejected observations mutated reconciliation: %#v", loaded)
	}
	assertBucketConsumed(t, db, "alice", projectedAt, bucket.ID, 0)
}

func TestAccountObservedUsageChargesExpiredAndRevokedProjectionMembers(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	projectedAt := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	expiry := projectedAt.Add(time.Hour)
	expiredNow := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), ExpiresAt: &expiry, RewardType: "base", Source: "expired-after-projection"})
	revokedLater := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), RewardType: "admin", Source: "revoked-after-projection"})
	snapshot := prepareAndActivate(t, db, "alice", PrepareProjectionInput{TelemtResetEpochSecs: 11, ProjectedAt: projectedAt})
	if _, err := Revoke(ctx, db, "alice", revokedLater.ID); err != nil {
		t.Fatal(err)
	}

	accounted, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 11, UsedBytes: 150})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation != accounted.Generation {
		t.Fatalf("generation = %d, want %d", accounted.Generation, snapshot.Generation)
	}
	assertAllocations(t, accounted.Allocations, []Allocation{{BucketID: expiredNow.ID, Bytes: 100}, {BucketID: revokedLater.ID, Bytes: 50}})
	assertBucketConsumed(t, db, "alice", time.Now().UTC(), expiredNow.ID, 100)
	assertBucketConsumed(t, db, "alice", time.Now().UTC(), revokedLater.ID, 50)
}

func TestAccountObservedUsageDetectsProjectionDrift(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	projectedAt := time.Date(2030, 3, 3, 12, 0, 0, 0, time.UTC)
	bucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), RewardType: "base", Source: "grant"})
	prepareAndActivate(t, db, "alice", PrepareProjectionInput{TelemtResetEpochSecs: 12, ProjectedAt: projectedAt})

	if _, err := Consume(ctx, db, "alice", 1, projectedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 12, UsedBytes: 10}); !errors.Is(err, ErrProjectionDrift) {
		t.Fatalf("projection drift error = %v", err)
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TelemtBaselineUsedBytes != 0 || loaded.Members[0].AccountedBytes != 0 {
		t.Fatalf("drift mutated reconciliation: %#v", loaded)
	}
	assertBucketConsumed(t, db, "alice", projectedAt, bucket.ID, 1)
}

func TestAccountObservedUsageInsufficientAllowanceRollsBack(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	projectedAt := time.Date(2030, 3, 4, 12, 0, 0, 0, time.UTC)
	bucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), RewardType: "base", Source: "grant"})
	prepareAndActivate(t, db, "alice", PrepareProjectionInput{TelemtResetEpochSecs: 13, ProjectedAt: projectedAt})

	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 13, UsedBytes: 101}); !errors.Is(err, ErrInsufficientProjectionAllowance) {
		t.Fatalf("insufficient allowance error = %v", err)
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TelemtBaselineUsedBytes != 0 || loaded.Members[0].AccountedBytes != 0 {
		t.Fatalf("insufficient usage mutated reconciliation: %#v", loaded)
	}
	assertBucketConsumed(t, db, "alice", projectedAt, bucket.ID, 0)
}

func TestAccountObservedUsageRollsBackBucketAndMemberWhenBaselineUpdateFails(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	projectedAt := time.Date(2030, 3, 5, 12, 0, 0, 0, time.UTC)
	bucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: projectedAt.Add(-time.Hour), RewardType: "base", Source: "grant"})
	prepareAndActivate(t, db, "alice", PrepareProjectionInput{TelemtResetEpochSecs: 14, ProjectedAt: projectedAt})
	if _, err := db.Exec(`
CREATE TRIGGER reject_baseline_update
BEFORE UPDATE OF telemt_baseline_used_bytes ON quota_reconciliations
BEGIN
    SELECT RAISE(ABORT, 'forced baseline failure');
END`); err != nil {
		t.Fatal(err)
	}

	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 14, UsedBytes: 50}); err == nil {
		t.Fatal("AccountObservedUsage() error = nil, want forced baseline failure")
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TelemtBaselineUsedBytes != 0 || loaded.Members[0].AccountedBytes != 0 {
		t.Fatalf("failed transaction mutated reconciliation: %#v", loaded)
	}
	assertBucketConsumed(t, db, "alice", projectedAt, bucket.ID, 0)
}

func TestAccountObservedUsageRejectsUnsignedValuesOutsideSQLiteRange(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	tooLarge := uint64(math.MaxInt64) + 1
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: tooLarge}); err == nil {
		t.Fatal("oversized reset epoch was accepted")
	}
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{UsedBytes: tooLarge}); err == nil {
		t.Fatal("oversized usage was accepted")
	}
}

func prepareAndActivate(t *testing.T, db *sql.DB, username string, input PrepareProjectionInput) ReconciliationSnapshot {
	t.Helper()
	snapshot, err := PrepareProjection(context.Background(), db, username, input)
	if err != nil {
		t.Fatal(err)
	}
	activateSnapshot(t, db, snapshot.ProxyUserID, snapshot.Generation)
	return snapshot
}

func activateSnapshot(t *testing.T, db *sql.DB, userID, generation int64) {
	t.Helper()
	result, err := db.Exec(`UPDATE quota_reconciliations SET phase = ? WHERE proxy_user_id = ? AND generation = ?`, string(PhaseActive), userID, generation)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		t.Fatalf("activate snapshot rows=%d err=%v", changed, err)
	}
}

func assertBucketConsumed(t *testing.T, db *sql.DB, username string, now time.Time, bucketID, want int64) {
	t.Helper()
	buckets, err := List(context.Background(), db, username, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.ID == bucketID {
			if bucket.ConsumedBytes != want {
				t.Fatalf("bucket %d consumed = %d, want %d", bucketID, bucket.ConsumedBytes, want)
			}
			return
		}
	}
	t.Fatalf("bucket %d not found", bucketID)
}

func assertAllocations(t *testing.T, got, want []Allocation) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("allocations len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("allocation[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
