package quotareconcile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

func TestReconcileInitialBootstrapAppliesCreditAndRestoresEnabled(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	expiry := now.Add(time.Hour)
	bucket := grantCredit(t, db, "alice", 100, now.Add(-time.Hour), &expiry)
	plane := &fakePlane{enabled: true, resetEpoch: 10, used: 30}
	r := newTestReconciler(t, db, plane, now)

	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if !plane.enabled || !plane.quotaSet || plane.quota != 100 {
		t.Fatalf("plane enabled=%v quotaSet=%v quota=%d", plane.enabled, plane.quotaSet, plane.quota)
	}
	if plane.expiry == nil || !plane.expiry.Equal(expiry) {
		t.Fatalf("plane expiry = %v, want %v", plane.expiry, expiry)
	}
	snapshot := loadSnapshot(t, db, "alice")
	if snapshot.Generation != 1 || snapshot.Phase != credit.PhaseActive || snapshot.ProjectedQuotaBytes != 100 || snapshot.TelemtBaselineUsedBytes != 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	assertConsumed(t, db, "alice", now, bucket.ID, 0)
	assertEventOrder(t, plane.events, "disable", "reset", "disable", "apply", "enable")
}

func TestReconcileActiveProjectionFreezesAccountsResetsAndReprojects(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	projectedAt := time.Date(2030, 5, 2, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	bucket := grantCredit(t, db, "alice", 100, projectedAt.Add(-time.Hour), nil)
	first := prepareActive(t, db, "alice", projectedAt, 7)
	plane := &fakePlane{
		enabled:    true,
		quotaSet:   true,
		quota:      100,
		resetEpoch: 7,
		used:       40,
		usageQueue: []QuotaUsage{
			{DataQuotaBytes: 100, UsedBytes: 30, LastResetEpochSecs: 7},
			{DataQuotaBytes: 100, UsedBytes: 40, LastResetEpochSecs: 7},
			{DataQuotaBytes: 100, UsedBytes: 40, LastResetEpochSecs: 7},
		},
	}
	r := newTestReconciler(t, db, plane, projectedAt.Add(10*time.Minute))

	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	assertConsumed(t, db, "alice", projectedAt, bucket.ID, 40)
	snapshot := loadSnapshot(t, db, "alice")
	if snapshot.Generation != first.Generation+1 || snapshot.Phase != credit.PhaseActive || snapshot.ProjectedQuotaBytes != 60 || snapshot.TelemtBaselineUsedBytes != 0 {
		t.Fatalf("reprojected snapshot = %#v", snapshot)
	}
	if !plane.enabled || plane.quota != 60 || plane.used != 0 || plane.resetEpoch != 8 {
		t.Fatalf("plane after reconciliation enabled=%v quota=%d used=%d reset=%d", plane.enabled, plane.quota, plane.used, plane.resetEpoch)
	}
	assertEventOrder(t, plane.events, "disable", "usage", "usage", "usage", "reset", "disable", "apply", "enable")
}

func TestReconcileRetriesAmbiguousDisableBeforeAccounting(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 3, 6, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	bucket := grantCredit(t, db, "alice", 100, now.Add(-time.Hour), nil)
	first := prepareActive(t, db, "alice", now, 19)
	plane := &fakePlane{enabled: true, quotaSet: true, quota: 100, resetEpoch: 19, used: 25, failAfter: map[string]int{"disable": 1}}
	r := newTestReconciler(t, db, plane, now.Add(time.Minute))

	if err := r.Reconcile(ctx, "alice"); err == nil {
		t.Fatal("first Reconcile() error = nil, want ambiguous disable failure")
	}
	snapshot := loadSnapshot(t, db, "alice")
	if snapshot.Generation != first.Generation || snapshot.Phase != credit.PhaseBlocking || snapshot.TelemtBaselineUsedBytes != 0 {
		t.Fatalf("snapshot after disable ambiguity = %#v", snapshot)
	}
	if plane.enabled {
		t.Fatal("ambiguous disable did not freeze fake data plane")
	}
	assertConsumed(t, db, "alice", now, bucket.ID, 0)

	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	snapshot = loadSnapshot(t, db, "alice")
	if snapshot.Generation != first.Generation+1 || snapshot.Phase != credit.PhaseActive || snapshot.ProjectedQuotaBytes != 75 {
		t.Fatalf("snapshot after disable retry = %#v", snapshot)
	}
	assertConsumed(t, db, "alice", now, bucket.ID, 25)
	if countEvent(plane.events, "disable") < 3 {
		t.Fatalf("disable was not retried through fail-closed stages: %#v", plane.events)
	}
}

func TestReconcileRetriesAmbiguousResetWithoutDoubleAccounting(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 3, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	bucket := grantCredit(t, db, "alice", 100, now.Add(-time.Hour), nil)
	first := prepareActive(t, db, "alice", now, 20)
	plane := &fakePlane{enabled: true, quotaSet: true, quota: 100, resetEpoch: 20, used: 25, failAfter: map[string]int{"reset": 1}}
	r := newTestReconciler(t, db, plane, now.Add(time.Minute))

	if err := r.Reconcile(ctx, "alice"); err == nil {
		t.Fatal("first Reconcile() error = nil, want ambiguous reset failure")
	}
	snapshot := loadSnapshot(t, db, "alice")
	if snapshot.Generation != first.Generation || snapshot.Phase != credit.PhaseResetting || snapshot.TelemtBaselineUsedBytes != 25 {
		t.Fatalf("snapshot after reset ambiguity = %#v", snapshot)
	}
	assertConsumed(t, db, "alice", now, bucket.ID, 25)
	if plane.resetEpoch != 21 || plane.used != 0 {
		t.Fatalf("ambiguous reset did not mutate fake plane: epoch=%d used=%d", plane.resetEpoch, plane.used)
	}

	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	snapshot = loadSnapshot(t, db, "alice")
	if snapshot.Generation != first.Generation+1 || snapshot.Phase != credit.PhaseActive || snapshot.ProjectedQuotaBytes != 75 {
		t.Fatalf("snapshot after reset retry = %#v", snapshot)
	}
	assertConsumed(t, db, "alice", now, bucket.ID, 25)
	if countEvent(plane.events, "reset") != 2 {
		t.Fatalf("reset events = %#v", plane.events)
	}
}

func TestReconcileRetriesAmbiguousPolicyApplyWithoutAnotherReset(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 4, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	grantCredit(t, db, "alice", 100, now.Add(-time.Hour), nil)
	plane := &fakePlane{enabled: true, resetEpoch: 30, used: 10, failAfter: map[string]int{"apply": 1}}
	r := newTestReconciler(t, db, plane, now)

	if err := r.Reconcile(ctx, "alice"); err == nil {
		t.Fatal("first Reconcile() error = nil, want ambiguous apply failure")
	}
	snapshot := loadSnapshot(t, db, "alice")
	if snapshot.Phase != credit.PhaseApplying || snapshot.Generation != 1 {
		t.Fatalf("snapshot after apply ambiguity = %#v", snapshot)
	}
	if countEvent(plane.events, "reset") != 1 || !plane.quotaSet || plane.quota != 100 {
		t.Fatalf("plane after apply ambiguity events=%#v quota=%d", plane.events, plane.quota)
	}

	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if countEvent(plane.events, "reset") != 1 || countEvent(plane.events, "apply") != 2 {
		t.Fatalf("retry repeated wrong mutations: %#v", plane.events)
	}
	if loadSnapshot(t, db, "alice").Phase != credit.PhaseActive || !plane.enabled {
		t.Fatalf("reconciliation did not finish after apply retry")
	}
}

func TestReconcileEnablingRereadsDesiredStateAfterAmbiguousEnable(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 5, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	grantCredit(t, db, "alice", 100, now.Add(-time.Hour), nil)
	plane := &fakePlane{enabled: true, resetEpoch: 40, failAfter: map[string]int{"enable": 1}}
	r := newTestReconciler(t, db, plane, now)

	if err := r.Reconcile(ctx, "alice"); err == nil {
		t.Fatal("first Reconcile() error = nil, want ambiguous enable failure")
	}
	if loadSnapshot(t, db, "alice").Phase != credit.PhaseEnabling || !plane.enabled {
		t.Fatalf("enable ambiguity was not persisted safely")
	}
	if _, err := proxyuser.SetDesiredEnabled(ctx, db, "alice", false); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if plane.enabled {
		t.Fatal("desired-disabled user was re-enabled during recovery")
	}
	if loadSnapshot(t, db, "alice").Phase != credit.PhaseActive {
		t.Fatal("reconciliation did not return to active phase")
	}
}

func TestReconcileNoCreditAppliesZeroQuotaAndKeepsDesiredDisabled(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 6, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", false)
	plane := &fakePlane{enabled: true, resetEpoch: 50}
	r := newTestReconciler(t, db, plane, now)

	if err := r.Reconcile(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	snapshot := loadSnapshot(t, db, "alice")
	if snapshot.ProjectedQuotaBytes != 0 || snapshot.Phase != credit.PhaseActive || len(snapshot.Members) != 0 {
		t.Fatalf("zero-credit snapshot = %#v", snapshot)
	}
	if !plane.quotaSet || plane.quota != 0 || plane.enabled {
		t.Fatalf("zero-credit plane enabled=%v quotaSet=%v quota=%d", plane.enabled, plane.quotaSet, plane.quota)
	}
	if countEvent(plane.events, "enable") != 0 {
		t.Fatalf("desired-disabled user was enabled: %#v", plane.events)
	}
}

func TestReconcileUnstableUsageStaysBlockedWithoutConsumption(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 7, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	bucket := grantCredit(t, db, "alice", 100, now.Add(-time.Hour), nil)
	prepareActive(t, db, "alice", now, 60)
	plane := &fakePlane{
		enabled:    true,
		quotaSet:   true,
		quota:      100,
		resetEpoch: 60,
		usageQueue: []QuotaUsage{
			{DataQuotaBytes: 100, UsedBytes: 1, LastResetEpochSecs: 60},
			{DataQuotaBytes: 100, UsedBytes: 2, LastResetEpochSecs: 60},
			{DataQuotaBytes: 100, UsedBytes: 3, LastResetEpochSecs: 60},
			{DataQuotaBytes: 100, UsedBytes: 4, LastResetEpochSecs: 60},
		},
	}
	r := newTestReconcilerWithOptions(t, db, plane, Options{Now: func() time.Time { return now }, StableSamples: 2, MaxUsageReads: 4, StabilityInterval: time.Nanosecond})

	if err := r.Reconcile(ctx, "alice"); !errors.Is(err, ErrQuotaUsageUnstable) {
		t.Fatalf("Reconcile() error = %v, want unstable usage", err)
	}
	if loadSnapshot(t, db, "alice").Phase != credit.PhaseBlocked {
		t.Fatal("unstable usage did not remain fail-closed in blocked phase")
	}
	assertConsumed(t, db, "alice", now, bucket.ID, 0)
	if plane.enabled {
		t.Fatal("unstable usage left data-plane user enabled")
	}
}

func TestReconcilePolicyMismatchStaysApplyingAndDisabled(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 5, 8, 12, 0, 0, 0, time.UTC)
	createUser(t, db, "alice", true)
	grantCredit(t, db, "alice", 100, now.Add(-time.Hour), nil)
	plane := &fakePlane{enabled: true, resetEpoch: 70, policyMismatch: true}
	r := newTestReconciler(t, db, plane, now)

	if err := r.Reconcile(ctx, "alice"); !errors.Is(err, ErrPolicyMismatch) {
		t.Fatalf("Reconcile() error = %v, want policy mismatch", err)
	}
	if loadSnapshot(t, db, "alice").Phase != credit.PhaseApplying || plane.enabled {
		t.Fatalf("policy mismatch was not fail-closed")
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createUser(t *testing.T, db *sql.DB, username string, enabled bool) {
	t.Helper()
	if _, err := proxyuser.Create(context.Background(), db, username, enabled); err != nil {
		t.Fatal(err)
	}
}

func grantCredit(t *testing.T, db *sql.DB, username string, bytes int64, starts time.Time, expires *time.Time) credit.Bucket {
	t.Helper()
	bucket, err := credit.GrantBucket(context.Background(), db, username, credit.Grant{
		OriginalBytes: bytes,
		StartsAt:      starts,
		ExpiresAt:     expires,
		RewardType:    "base",
		Source:        "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func prepareActive(t *testing.T, db *sql.DB, username string, at time.Time, resetEpoch uint64) credit.ReconciliationSnapshot {
	t.Helper()
	snapshot, err := credit.PrepareProjection(context.Background(), db, username, credit.PrepareProjectionInput{
		TelemtResetEpochSecs: resetEpoch,
		ProjectedAt:          at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := credit.TransitionReconciliationPhase(context.Background(), db, username, snapshot.Generation, credit.PhaseApplying, credit.PhaseActive); err != nil {
		t.Fatal(err)
	}
	snapshot.Phase = credit.PhaseActive
	return snapshot
}

func loadSnapshot(t *testing.T, db *sql.DB, username string) credit.ReconciliationSnapshot {
	t.Helper()
	snapshot, err := credit.LoadReconciliation(context.Background(), db, username)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func assertConsumed(t *testing.T, db *sql.DB, username string, now time.Time, bucketID, want int64) {
	t.Helper()
	buckets, err := credit.List(context.Background(), db, username, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.ID == bucketID {
			if bucket.ConsumedBytes != want {
				t.Fatalf("bucket %d consumed=%d want=%d", bucketID, bucket.ConsumedBytes, want)
			}
			return
		}
	}
	t.Fatalf("bucket %d not found", bucketID)
}

func newTestReconciler(t *testing.T, db *sql.DB, plane DataPlane, now time.Time) *Reconciler {
	t.Helper()
	return newTestReconcilerWithOptions(t, db, plane, Options{
		Now:               func() time.Time { return now },
		StableSamples:     2,
		MaxUsageReads:     5,
		StabilityInterval: time.Nanosecond,
	})
}

func newTestReconcilerWithOptions(t *testing.T, db *sql.DB, plane DataPlane, options Options) *Reconciler {
	t.Helper()
	r, err := New(db, plane, options)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type fakePlane struct {
	enabled        bool
	quotaSet       bool
	quota          uint64
	expiry         *time.Time
	resetEpoch     uint64
	used           uint64
	usageQueue     []QuotaUsage
	events         []string
	failBefore     map[string]int
	failAfter      map[string]int
	policyMismatch bool
}

func (p *fakePlane) SetEnabled(_ context.Context, _ string, enabled bool) error {
	op := "disable"
	if enabled {
		op = "enable"
	}
	p.events = append(p.events, op)
	if p.takeFailure(p.failBefore, op) {
		return fmt.Errorf("forced %s failure", op)
	}
	p.enabled = enabled
	if p.takeFailure(p.failAfter, op) {
		return fmt.Errorf("forced ambiguous %s failure", op)
	}
	return nil
}

func (p *fakePlane) QuotaUsage(_ context.Context, _ string) (QuotaUsage, bool, error) {
	p.events = append(p.events, "usage")
	if p.takeFailure(p.failBefore, "usage") {
		return QuotaUsage{}, false, errors.New("forced usage failure")
	}
	if len(p.usageQueue) > 0 {
		usage := p.usageQueue[0]
		p.usageQueue = p.usageQueue[1:]
		p.quotaSet = true
		p.quota = usage.DataQuotaBytes
		p.used = usage.UsedBytes
		p.resetEpoch = usage.LastResetEpochSecs
		return usage, true, nil
	}
	if !p.quotaSet || p.quota == 0 {
		return QuotaUsage{}, false, nil
	}
	return QuotaUsage{DataQuotaBytes: p.quota, UsedBytes: p.used, LastResetEpochSecs: p.resetEpoch}, true, nil
}

func (p *fakePlane) ResetQuota(_ context.Context, _ string) (ResetResult, error) {
	p.events = append(p.events, "reset")
	if p.takeFailure(p.failBefore, "reset") {
		return ResetResult{}, errors.New("forced reset failure")
	}
	p.used = 0
	p.resetEpoch++
	result := ResetResult{UsedBytes: 0, LastResetEpochSecs: p.resetEpoch}
	if p.takeFailure(p.failAfter, "reset") {
		return ResetResult{}, errors.New("forced ambiguous reset failure")
	}
	return result, nil
}

func (p *fakePlane) ApplyPolicy(_ context.Context, _ string, policy Policy) (PolicyState, error) {
	p.events = append(p.events, "apply")
	if p.takeFailure(p.failBefore, "apply") {
		return PolicyState{}, errors.New("forced apply failure")
	}
	p.quotaSet = true
	p.quota = policy.DataQuotaBytes
	p.expiry = cloneTime(policy.ExpiresAt)
	state := PolicyState{DataQuotaBytes: uint64Ptr(p.quota), ExpiresAt: cloneTime(p.expiry)}
	if p.policyMismatch {
		wrong := p.quota + 1
		state.DataQuotaBytes = &wrong
	}
	if p.takeFailure(p.failAfter, "apply") {
		return PolicyState{}, errors.New("forced ambiguous apply failure")
	}
	return state, nil
}

func (p *fakePlane) takeFailure(failures map[string]int, operation string) bool {
	if failures == nil || failures[operation] <= 0 {
		return false
	}
	failures[operation]--
	return true
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func countEvent(events []string, want string) int {
	count := 0
	for _, event := range events {
		if event == want {
			count++
		}
	}
	return count
}

func assertEventOrder(t *testing.T, events []string, want ...string) {
	t.Helper()
	index := 0
	for _, event := range events {
		if index < len(want) && event == want[index] {
			index++
		}
	}
	if index != len(want) {
		t.Fatalf("events %#v do not contain ordered subsequence %#v", events, want)
	}
}
