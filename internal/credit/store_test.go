package credit

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

func TestCreditStatusesAndBalance(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if _, err := proxyuser.Create(ctx, db, "alice", true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 1, 10, 12, 0, 0, 0, time.UTC)

	earlyExpiry := now.Add(24 * time.Hour)
	futureStart := now.Add(time.Hour)
	futureExpiry := futureStart.Add(24 * time.Hour)
	expiredAt := now.Add(-time.Hour)
	expiredStart := expiredAt.Add(-24 * time.Hour)

	active, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), ExpiresAt: &earlyExpiry, RewardType: "start", Source: "bootstrap"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 50, StartsAt: futureStart, ExpiresAt: &futureExpiry, RewardType: "admin", Source: "future"}); err != nil {
		t.Fatal(err)
	}
	if _, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 75, StartsAt: expiredStart, ExpiresAt: &expiredAt, RewardType: "referral", Source: "expired"}); err != nil {
		t.Fatal(err)
	}
	revoked, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 25, StartsAt: now.Add(-time.Hour), RewardType: "admin", Source: "revoked"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Revoke(ctx, db, "alice", revoked.ID); err != nil {
		t.Fatal(err)
	}

	buckets, err := List(ctx, db, "alice", now)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]Status{}
	for _, bucket := range buckets {
		got[bucket.ID] = bucket.Status
	}
	if got[active.ID] != StatusActive || got[revoked.ID] != StatusRevoked {
		t.Fatalf("unexpected statuses: %#v", got)
	}
	var pending, expired bool
	for _, status := range got {
		pending = pending || status == StatusPending
		expired = expired || status == StatusExpired
	}
	if !pending || !expired {
		t.Fatalf("missing derived pending/expired state: %#v", got)
	}
	balance, err := Balance(ctx, db, "alice", now)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 100 {
		t.Fatalf("Balance() = %d, want 100", balance)
	}
}

func TestConsumeUsesNearestExpiryAndIsAtomicOnInsufficientCredit(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if _, err := proxyuser.Create(ctx, db, "alice", true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)
	earlyExpiry := now.Add(24 * time.Hour)
	laterExpiry := now.Add(72 * time.Hour)
	early, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), ExpiresAt: &earlyExpiry, RewardType: "referral", Source: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	later, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 200, StartsAt: now.Add(-time.Hour), ExpiresAt: &laterExpiry, RewardType: "referral", Source: "r2"})
	if err != nil {
		t.Fatal(err)
	}
	nonExpiring, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 300, StartsAt: now.Add(-time.Hour), RewardType: "start", Source: "base"})
	if err != nil {
		t.Fatal(err)
	}

	consumed, err := Consume(ctx, db, "alice", 350, now)
	if err != nil {
		t.Fatal(err)
	}
	if consumed.ConsumedBytes != 350 || len(consumed.Allocations) != 3 {
		t.Fatalf("Consume() = %#v", consumed)
	}
	if consumed.Allocations[0] != (Allocation{BucketID: early.ID, Bytes: 100}) || consumed.Allocations[1] != (Allocation{BucketID: later.ID, Bytes: 200}) || consumed.Allocations[2] != (Allocation{BucketID: nonExpiring.ID, Bytes: 50}) {
		t.Fatalf("allocation order = %#v", consumed.Allocations)
	}
	buckets, err := List(ctx, db, "alice", now)
	if err != nil {
		t.Fatal(err)
	}
	byID := bucketMap(buckets)
	if byID[early.ID].ConsumedBytes != 100 || byID[early.ID].Status != StatusExhausted || byID[later.ID].ConsumedBytes != 200 || byID[later.ID].Status != StatusExhausted || byID[nonExpiring.ID].ConsumedBytes != 50 {
		t.Fatalf("unexpected consumption state: %#v", byID)
	}

	before := byID
	if _, err := Consume(ctx, db, "alice", 1000, now); !errors.Is(err, ErrInsufficientCredit) {
		t.Fatalf("insufficient Consume() error = %v", err)
	}
	afterList, err := List(ctx, db, "alice", now)
	if err != nil {
		t.Fatal(err)
	}
	after := bucketMap(afterList)
	for id, bucket := range before {
		if after[id].ConsumedBytes != bucket.ConsumedBytes {
			t.Fatalf("bucket %d changed after failed consume: before=%d after=%d", id, bucket.ConsumedBytes, after[id].ConsumedBytes)
		}
	}
}

func TestCreditValidationAndIneligibleBuckets(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	if _, err := proxyuser.Create(ctx, db, "alice", true); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2030, 3, 1, 0, 0, 0, 0, time.UTC)
	badExpiry := now
	cases := []Grant{
		{OriginalBytes: 0, StartsAt: now, RewardType: "start", Source: "x"},
		{OriginalBytes: 1, StartsAt: time.Time{}, RewardType: "start", Source: "x"},
		{OriginalBytes: 1, StartsAt: now, ExpiresAt: &badExpiry, RewardType: "start", Source: "x"},
		{OriginalBytes: 1, StartsAt: now, RewardType: "", Source: "x"},
		{OriginalBytes: 1, StartsAt: now, RewardType: "start", Source: ""},
	}
	for i, grant := range cases {
		if _, err := GrantBucket(ctx, db, "alice", grant); err == nil {
			t.Fatalf("case %d accepted invalid grant", i)
		}
	}
	if _, err := GrantBucket(ctx, db, "missing", Grant{OriginalBytes: 1, StartsAt: now, RewardType: "start", Source: "x"}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing user error = %v", err)
	}

	futureStart := now.Add(time.Hour)
	futureExpiry := now.Add(2 * time.Hour)
	if _, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 100, StartsAt: futureStart, ExpiresAt: &futureExpiry, RewardType: "admin", Source: "future"}); err != nil {
		t.Fatal(err)
	}
	expiredStart := now.Add(-2 * time.Hour)
	expiredAt := now.Add(-time.Hour)
	if _, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 100, StartsAt: expiredStart, ExpiresAt: &expiredAt, RewardType: "referral", Source: "expired"}); err != nil {
		t.Fatal(err)
	}
	revoked, err := GrantBucket(ctx, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "admin", Source: "revoked"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Revoke(ctx, db, "alice", revoked.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Consume(ctx, db, "alice", 1, now); !errors.Is(err, ErrInsufficientCredit) {
		t.Fatalf("Consume() with only ineligible buckets error = %v", err)
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func bucketMap(buckets []Bucket) map[int64]Bucket {
	result := make(map[int64]Bucket, len(buckets))
	for _, bucket := range buckets {
		result[bucket.ID] = bucket
	}
	return result
}
