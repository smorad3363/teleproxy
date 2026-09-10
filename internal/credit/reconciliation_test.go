package credit

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"testing"
	"time"
)

func TestPrepareProjectionPersistsExactRestartableSnapshot(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 2, 1, 12, 0, 0, 0, time.UTC)
	earlyExpiry := now.Add(time.Hour)
	lateExpiry := now.Add(2 * time.Hour)
	futureStart := now.Add(30 * time.Minute)
	futureExpiry := now.Add(3 * time.Hour)

	late := grant(t, db, "alice", Grant{OriginalBytes: 200, StartsAt: now.Add(-time.Hour), ExpiresAt: &lateExpiry, RewardType: "reward", Source: "late"})
	nonExpiring := grant(t, db, "alice", Grant{OriginalBytes: 300, StartsAt: now.Add(-time.Hour), RewardType: "admin", Source: "permanent"})
	early := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), ExpiresAt: &earlyExpiry, RewardType: "base", Source: "early"})
	_ = grant(t, db, "alice", Grant{OriginalBytes: 400, StartsAt: futureStart, ExpiresAt: &futureExpiry, RewardType: "future", Source: "scheduled"})

	if _, err := Consume(ctx, db, "alice", 25, now); err != nil {
		t.Fatal(err)
	}

	snapshot, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{
		ExpectedGeneration:      0,
		TelemtResetEpochSecs:    1234,
		TelemtBaselineUsedBytes: 50,
		ProjectedAt:             now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation != 1 || snapshot.Phase != PhaseApplying {
		t.Fatalf("snapshot generation/phase = %d/%q", snapshot.Generation, snapshot.Phase)
	}
	if snapshot.ProjectedQuotaBytes != 625 {
		t.Fatalf("projected quota = %d, want 625", snapshot.ProjectedQuotaBytes)
	}
	if snapshot.EnforcedExpiry == nil || !snapshot.EnforcedExpiry.Equal(earlyExpiry) {
		t.Fatalf("enforced expiry = %v, want %v", snapshot.EnforcedExpiry, earlyExpiry)
	}
	if snapshot.NextStart == nil || !snapshot.NextStart.Equal(futureStart) {
		t.Fatalf("next start = %v, want %v", snapshot.NextStart, futureStart)
	}
	wantMembers := []ProjectionMemberSnapshot{
		{Position: 0, BucketID: early.ID, AllowanceBytes: 75},
		{Position: 1, BucketID: late.ID, AllowanceBytes: 200},
		{Position: 2, BucketID: nonExpiring.ID, AllowanceBytes: 300},
	}
	assertMembers(t, snapshot.Members, wantMembers)

	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Generation != snapshot.Generation || loaded.Phase != snapshot.Phase || loaded.TelemtResetEpochSecs != 1234 || loaded.TelemtBaselineUsedBytes != 50 || loaded.ProjectedQuotaBytes != 625 || !loaded.ProjectedAt.Equal(now) {
		t.Fatalf("loaded snapshot differs: %#v", loaded)
	}
	if loaded.EnforcedExpiry == nil || !loaded.EnforcedExpiry.Equal(earlyExpiry) || loaded.NextStart == nil || !loaded.NextStart.Equal(futureStart) {
		t.Fatalf("loaded boundaries differ: expiry=%v next=%v", loaded.EnforcedExpiry, loaded.NextStart)
	}
	assertMembers(t, loaded.Members, wantMembers)
}

func TestPrepareProjectionUsesGenerationCASAndDoesNotConsumeCredits(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 2, 2, 12, 0, 0, 0, time.UTC)
	bucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})

	first, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ExpectedGeneration: 0, ProjectedAt: now}); !errors.Is(err, ErrReconciliationConflict) {
		t.Fatalf("stale PrepareProjection() error = %v", err)
	}
	second, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ExpectedGeneration: first.Generation, ProjectedAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if second.Generation != 2 {
		t.Fatalf("generation = %d, want 2", second.Generation)
	}

	buckets, err := List(ctx, db, "alice", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 || buckets[0].ID != bucket.ID || buckets[0].ConsumedBytes != 0 {
		t.Fatalf("projection mutated credit bucket: %#v", buckets)
	}
}

func TestPrepareProjectionReplacementIsAtomicOnMemberFailure(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 2, 3, 12, 0, 0, 0, time.UTC)
	grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})

	first, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TRIGGER reject_projection_member
BEFORE INSERT ON quota_projection_members
BEGIN
    SELECT RAISE(ABORT, 'forced member failure');
END`); err != nil {
		t.Fatal(err)
	}

	if _, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ExpectedGeneration: first.Generation, ProjectedAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("PrepareProjection() error = nil, want forced member failure")
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Generation != first.Generation || len(loaded.Members) != len(first.Members) || loaded.Members[0] != first.Members[0] {
		t.Fatalf("failed replacement changed snapshot: %#v", loaded)
	}
}

func TestProjectionMemberConstraintsRejectDuplicatesAndWrongUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	createProxyUser(t, db, "bob")
	now := time.Date(2030, 2, 4, 12, 0, 0, 0, time.UTC)
	aliceBucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "alice"})
	future := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(time.Hour), RewardType: "future", Source: "alice"})
	bobBucket := grant(t, db, "bob", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "bob"})
	snapshot, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO quota_projection_members(proxy_user_id, generation, position, bucket_id, allowance_bytes) VALUES (?, ?, ?, ?, ?)`, snapshot.ProxyUserID, snapshot.Generation, 1, aliceBucket.ID, 1); err == nil {
		t.Fatal("duplicate member was accepted")
	}
	if _, err := db.Exec(`INSERT INTO quota_projection_members(proxy_user_id, generation, position, bucket_id, allowance_bytes) VALUES (?, ?, ?, ?, ?)`, snapshot.ProxyUserID, snapshot.Generation, 0, future.ID, 1); err == nil {
		t.Fatal("duplicate position was accepted")
	}
	if _, err := db.Exec(`INSERT INTO quota_projection_members(proxy_user_id, generation, position, bucket_id, allowance_bytes) VALUES (?, ?, ?, ?, ?)`, snapshot.ProxyUserID, snapshot.Generation, 1, bobBucket.ID, 1); err == nil {
		t.Fatal("cross-user bucket was accepted")
	}
}

func TestPrepareProjectionRejectsUnsignedValuesOutsideSQLiteRange(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 2, 5, 12, 0, 0, 0, time.UTC)
	tooLarge := uint64(math.MaxInt64) + 1

	if _, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now, TelemtResetEpochSecs: tooLarge}); err == nil {
		t.Fatal("oversized reset epoch was accepted")
	}
	if _, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now, TelemtBaselineUsedBytes: tooLarge}); err == nil {
		t.Fatal("oversized baseline usage was accepted")
	}
}

func TestPrepareProjectionAllowsBlockedSnapshotWithFutureBoundary(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 2, 6, 12, 0, 0, 0, time.UTC)
	futureStart := now.Add(time.Hour)
	grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: futureStart, RewardType: "scheduled", Source: "future"})

	snapshot, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ProjectedQuotaBytes != 0 || len(snapshot.Members) != 0 {
		t.Fatalf("blocked snapshot = %#v", snapshot)
	}
	if snapshot.NextStart == nil || !snapshot.NextStart.Equal(futureStart) {
		t.Fatalf("next start = %v, want %v", snapshot.NextStart, futureStart)
	}
}

func TestReconciliationDoesNotBreakProxyUserCascade(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 2, 7, 12, 0, 0, 0, time.UTC)
	grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})
	if _, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("DELETE FROM proxy_users WHERE username = ?", "alice"); err != nil {
		t.Fatalf("delete proxy user with reconciliation: %v", err)
	}
	for _, table := range []string{"credit_buckets", "quota_reconciliations", "quota_projection_members"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows after proxy user delete = %d, want 0", table, count)
		}
	}
}

func TestLoadReconciliationErrors(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	if _, err := LoadReconciliation(ctx, db, "alice"); !errors.Is(err, ErrReconciliationNotFound) {
		t.Fatalf("LoadReconciliation() error = %v", err)
	}
	if _, err := LoadReconciliation(ctx, db, "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing user error = %v", err)
	}
}

func createProxyUser(t *testing.T, db *sql.DB, username string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO proxy_users(username, desired_enabled, sync_state, created_at, updated_at) VALUES (?, 1, 'pending', 1, 1)`, username); err != nil {
		t.Fatal(err)
	}
}

func grant(t *testing.T, db *sql.DB, username string, input Grant) Bucket {
	t.Helper()
	bucket, err := GrantBucket(context.Background(), db, username, input)
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func assertMembers(t *testing.T, got, want []ProjectionMemberSnapshot) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("members len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("member[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
