package credit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReconciliationPhaseTransitionsUseGenerationAndPhaseCAS(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 4, 1, 12, 0, 0, 0, time.UTC)
	grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})

	snapshot, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseApplying, PhaseEnabling); err != nil {
		t.Fatal(err)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseEnabling, PhaseActive); err != nil {
		t.Fatal(err)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseApplying, PhaseActive); !errors.Is(err, ErrReconciliationPhaseConflict) {
		t.Fatalf("stale phase transition error = %v", err)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation+1, PhaseActive, PhaseBlocking); !errors.Is(err, ErrReconciliationPhaseConflict) {
		t.Fatalf("stale generation transition error = %v", err)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseActive, PhaseBlocked); !errors.Is(err, ErrReconciliationTransition) {
		t.Fatalf("invalid transition error = %v", err)
	}

	for _, transition := range []struct {
		from ReconciliationPhase
		to   ReconciliationPhase
	}{
		{PhaseActive, PhaseBlocking},
		{PhaseBlocking, PhaseBlocked},
		{PhaseBlocked, PhaseResetting},
	} {
		if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, transition.from, transition.to); err != nil {
			t.Fatalf("transition %s -> %s: %v", transition.from, transition.to, err)
		}
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != PhaseResetting {
		t.Fatalf("phase = %q, want %q", loaded.Phase, PhaseResetting)
	}
}

func TestPrepareProjectionReplacementRequiresPreApplyOrResettingPhase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 4, 2, 12, 0, 0, 0, time.UTC)
	grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})

	first, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	preApplyReplacement, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ExpectedGeneration: first.Generation, ProjectedAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if preApplyReplacement.Generation != first.Generation+1 || preApplyReplacement.Phase != PhaseApplying {
		t.Fatalf("pre-apply replacement = %#v", preApplyReplacement)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", preApplyReplacement.Generation, PhaseApplying, PhaseActive); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ExpectedGeneration: preApplyReplacement.Generation, ProjectedAt: now.Add(2 * time.Minute)}); !errors.Is(err, ErrReconciliationPhaseConflict) {
		t.Fatalf("active replacement error = %v", err)
	}
	for _, transition := range []struct {
		from ReconciliationPhase
		to   ReconciliationPhase
	}{
		{PhaseActive, PhaseBlocking},
		{PhaseBlocking, PhaseBlocked},
		{PhaseBlocked, PhaseResetting},
	} {
		if err := TransitionReconciliationPhase(ctx, db, "alice", preApplyReplacement.Generation, transition.from, transition.to); err != nil {
			t.Fatal(err)
		}
	}
	resetReplacement, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ExpectedGeneration: preApplyReplacement.Generation, ProjectedAt: now.Add(3 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if resetReplacement.Generation != preApplyReplacement.Generation+1 || resetReplacement.Phase != PhaseApplying {
		t.Fatalf("reset replacement = %#v", resetReplacement)
	}
}

func TestAccountObservedUsageAllowsBlockedButRejectsOtherTransientPhases(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 4, 3, 12, 0, 0, 0, time.UTC)
	bucket := grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})
	snapshot := prepareAndActivate(t, db, "alice", PrepareProjectionInput{TelemtResetEpochSecs: 50, ProjectedAt: now})

	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseActive, PhaseBlocking); err != nil {
		t.Fatal(err)
	}
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 50, UsedBytes: 10}); !errors.Is(err, ErrReconciliationNotActive) {
		t.Fatalf("blocking accounting error = %v", err)
	}
	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseBlocking, PhaseBlocked); err != nil {
		t.Fatal(err)
	}
	accounted, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 50, UsedBytes: 40})
	if err != nil {
		t.Fatal(err)
	}
	if accounted.AccountedBytes != 40 {
		t.Fatalf("blocked accounted bytes = %d, want 40", accounted.AccountedBytes)
	}
	loaded, err := LoadReconciliation(ctx, db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Phase != PhaseBlocked || loaded.TelemtBaselineUsedBytes != 40 {
		t.Fatalf("blocked snapshot after accounting = %#v", loaded)
	}
	assertBucketConsumed(t, db, "alice", now, bucket.ID, 40)

	if err := TransitionReconciliationPhase(ctx, db, "alice", snapshot.Generation, PhaseBlocked, PhaseResetting); err != nil {
		t.Fatal(err)
	}
	if _, err := AccountObservedUsage(ctx, db, "alice", QuotaUsageObservation{ResetEpochSecs: 50, UsedBytes: 50}); !errors.Is(err, ErrReconciliationNotActive) {
		t.Fatalf("resetting accounting error = %v", err)
	}
}

func TestLoadReconciliationRejectsUnknownPhase(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createProxyUser(t, db, "alice")
	now := time.Date(2030, 4, 4, 12, 0, 0, 0, time.UTC)
	grant(t, db, "alice", Grant{OriginalBytes: 100, StartsAt: now.Add(-time.Hour), RewardType: "base", Source: "grant"})
	snapshot, err := PrepareProjection(ctx, db, "alice", PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE quota_reconciliations SET phase = 'mystery' WHERE proxy_user_id = ? AND generation = ?`, snapshot.ProxyUserID, snapshot.Generation); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReconciliation(ctx, db, "alice"); err == nil {
		t.Fatal("LoadReconciliation() accepted unknown phase")
	}
}
