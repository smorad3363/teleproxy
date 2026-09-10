package referral

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"
)

func TestApproveEligibilityIsIdempotentAndCreatesNoCredit(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 7301)
	invitee := resolveTestUser(t, ctx, db, 7302)
	code, err := EnsureCode(ctx, db, inviter.User.ID, time.Unix(1_800_000_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	created, err := AttributeNewInvitee(ctx, db, invitee, code.Value, time.Unix(1_800_000_010, 0).UTC())
	if err != nil || created.Outcome != OutcomeCreated {
		t.Fatalf("create attribution = %#v, %v", created, err)
	}

	approvedAt := time.Unix(1_800_000_020, 0).UTC()
	approved, err := ApproveEligibility(ctx, db, invitee.User.ID, approvedAt)
	if err != nil || approved.Outcome != EligibilityApproved {
		t.Fatalf("ApproveEligibility() = %#v, %v", approved, err)
	}
	if approved.Attribution.Status != StatusPending || approved.Attribution.EligibleAt == nil || !approved.Attribution.EligibleAt.Equal(approvedAt) || approved.Attribution.FinalizedAt != nil || approved.Attribution.RejectionReason != nil {
		t.Fatalf("approved attribution = %#v", approved.Attribution)
	}

	replayed, err := ApproveEligibility(ctx, db, invitee.User.ID, approvedAt.Add(time.Hour))
	if err != nil || replayed.Outcome != EligibilityAlreadyApproved {
		t.Fatalf("replayed approval = %#v, %v", replayed, err)
	}
	if replayed.Attribution.EligibleAt == nil || !replayed.Attribution.EligibleAt.Equal(approvedAt) {
		t.Fatalf("replay changed eligible_at: %#v", replayed.Attribution)
	}
	if countCreditBuckets(t, db) != 0 {
		t.Fatal("eligibility approval created a credit bucket")
	}
	count, err := RewardedCount(ctx, db, inviter.User.ID)
	if err != nil || count != 0 {
		t.Fatalf("RewardedCount() = %d, %v, want 0", count, err)
	}
}

func TestRejectEligibilityIsTerminalAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 7401)
	invitee := resolveTestUser(t, ctx, db, 7402)
	code, err := EnsureCode(ctx, db, inviter.User.ID, time.Unix(1_800_001_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if result, err := AttributeNewInvitee(ctx, db, invitee, code.Value, time.Unix(1_800_001_010, 0).UTC()); err != nil || result.Outcome != OutcomeCreated {
		t.Fatalf("create attribution = %#v, %v", result, err)
	}

	rejectedAt := time.Unix(1_800_001_020, 0).UTC()
	rejected, err := RejectEligibility(ctx, db, invitee.User.ID, RejectionBlacklist, rejectedAt)
	if err != nil || rejected.Outcome != EligibilityRejected {
		t.Fatalf("RejectEligibility() = %#v, %v", rejected, err)
	}
	if rejected.Attribution.Status != StatusRejected || rejected.Attribution.EligibleAt != nil || rejected.Attribution.FinalizedAt == nil || !rejected.Attribution.FinalizedAt.Equal(rejectedAt) || rejected.Attribution.RejectionReason == nil || *rejected.Attribution.RejectionReason != string(RejectionBlacklist) {
		t.Fatalf("rejected attribution = %#v", rejected.Attribution)
	}

	replayed, err := RejectEligibility(ctx, db, invitee.User.ID, RejectionCooldown, rejectedAt.Add(time.Hour))
	if err != nil || replayed.Outcome != EligibilityAlreadyRejected {
		t.Fatalf("replayed rejection = %#v, %v", replayed, err)
	}
	if replayed.Attribution.RejectionReason == nil || *replayed.Attribution.RejectionReason != string(RejectionBlacklist) || replayed.Attribution.FinalizedAt == nil || !replayed.Attribution.FinalizedAt.Equal(rejectedAt) {
		t.Fatalf("replayed rejection changed terminal state: %#v", replayed.Attribution)
	}
	approved, err := ApproveEligibility(ctx, db, invitee.User.ID, rejectedAt.Add(2*time.Hour))
	if err != nil || approved.Outcome != EligibilityAlreadyRejected {
		t.Fatalf("approval after rejection = %#v, %v", approved, err)
	}
	if countCreditBuckets(t, db) != 0 {
		t.Fatal("eligibility rejection created a credit bucket")
	}
}

func TestRejectEligibilityRejectsUnknownReasonWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 7501)
	invitee := resolveTestUser(t, ctx, db, 7502)
	code, err := EnsureCode(ctx, db, inviter.User.ID, time.Unix(1_800_002_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if result, err := AttributeNewInvitee(ctx, db, invitee, code.Value, time.Unix(1_800_002_010, 0).UTC()); err != nil || result.Outcome != OutcomeCreated {
		t.Fatalf("create attribution = %#v, %v", result, err)
	}
	if _, err := RejectEligibility(ctx, db, invitee.User.ID, RejectionReason("invented"), time.Unix(1_800_002_020, 0).UTC()); err == nil {
		t.Fatal("RejectEligibility() accepted an unknown reason")
	}
	attribution, err := GetAttribution(ctx, db, invitee.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if attribution.Status != StatusPending || attribution.EligibleAt != nil || attribution.RejectionReason != nil || attribution.FinalizedAt != nil {
		t.Fatalf("invalid rejection mutated attribution: %#v", attribution)
	}
}

func TestApproveRejectRaceLeavesOneSafeEligibilityState(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 7601)
	invitee := resolveTestUser(t, ctx, db, 7602)
	code, err := EnsureCode(ctx, db, inviter.User.ID, time.Unix(1_800_003_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if result, err := AttributeNewInvitee(ctx, db, invitee, code.Value, time.Unix(1_800_003_010, 0).UTC()); err != nil || result.Outcome != OutcomeCreated {
		t.Fatalf("create attribution = %#v, %v", result, err)
	}

	now := time.Unix(1_800_003_020, 0).UTC()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := ApproveEligibility(ctx, db, invitee.User.ID, now)
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := RejectEligibility(ctx, db, invitee.User.ID, RejectionSuspicious, now)
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent eligibility mutation error = %v", err)
		}
	}

	attribution, err := GetAttribution(ctx, db, invitee.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	approved := attribution.Status == StatusPending && attribution.EligibleAt != nil && attribution.RejectionReason == nil && attribution.FinalizedAt == nil
	rejected := attribution.Status == StatusRejected && attribution.EligibleAt == nil && attribution.RejectionReason != nil && attribution.FinalizedAt != nil
	if !approved && !rejected {
		t.Fatalf("race left unsafe attribution state: %#v", attribution)
	}
	if countCreditBuckets(t, db) != 0 {
		t.Fatal("eligibility race created a credit bucket")
	}
}

func countCreditBuckets(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM credit_buckets").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
