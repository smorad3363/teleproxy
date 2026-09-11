package referral

import (
	"context"
	"testing"
	"time"
)

func TestHistoryFiltersExactRejectionReasonAndComposesWithCursorAndStatus(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 8120)
	invitee1 := resolveTestUser(t, ctx, db, 8121)
	invitee2 := resolveTestUser(t, ctx, db, 8122)
	invitee3 := resolveTestUser(t, ctx, db, 8123)
	invitee4 := resolveTestUser(t, ctx, db, 8124)
	base := time.Unix(1_812_000_000, 0).UTC()
	code, err := EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	created1, err := AttributeNewInvitee(ctx, db, invitee1, code.Value, base.Add(time.Second))
	if err != nil || created1.Outcome != OutcomeCreated {
		t.Fatalf("create first attribution = %#v, %v", created1, err)
	}
	created2, err := AttributeNewInvitee(ctx, db, invitee2, code.Value, base.Add(2*time.Second))
	if err != nil || created2.Outcome != OutcomeCreated {
		t.Fatalf("create second attribution = %#v, %v", created2, err)
	}
	created3, err := AttributeNewInvitee(ctx, db, invitee3, code.Value, base.Add(3*time.Second))
	if err != nil || created3.Outcome != OutcomeCreated {
		t.Fatalf("create third attribution = %#v, %v", created3, err)
	}
	created4, err := AttributeNewInvitee(ctx, db, invitee4, code.Value, base.Add(4*time.Second))
	if err != nil || created4.Outcome != OutcomeCreated {
		t.Fatalf("create fourth attribution = %#v, %v", created4, err)
	}
	if result, err := RejectEligibility(ctx, db, invitee1.User.ID, RejectionSuspicious, base.Add(10*time.Second)); err != nil || result.Outcome != EligibilityRejected {
		t.Fatalf("reject first attribution = %#v, %v", result, err)
	}
	if result, err := RejectEligibility(ctx, db, invitee2.User.ID, RejectionCooldown, base.Add(11*time.Second)); err != nil || result.Outcome != EligibilityRejected {
		t.Fatalf("reject second attribution = %#v, %v", result, err)
	}
	if result, err := RejectEligibility(ctx, db, invitee3.User.ID, RejectionSuspicious, base.Add(12*time.Second)); err != nil || result.Outcome != EligibilityRejected {
		t.Fatalf("reject third attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHistoryRows(t, db, "referral_attributions")
	beforeCredits := countHistoryRows(t, db, "credit_buckets")

	first, err := History(ctx, db, HistoryQuery{RejectionReason: RejectionSuspicious, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != created3.Attribution.ID || first.Items[0].RejectionReason == nil || *first.Items[0].RejectionReason != string(RejectionSuspicious) {
		t.Fatalf("first suspicious page = %#v", first.Items)
	}
	if first.NextBeforeID == nil || *first.NextBeforeID != created3.Attribution.ID {
		t.Fatalf("first suspicious next_before_id = %v", first.NextBeforeID)
	}
	second, err := History(ctx, db, HistoryQuery{RejectionReason: RejectionSuspicious, BeforeID: *first.NextBeforeID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != created1.Attribution.ID || second.NextBeforeID != nil {
		t.Fatalf("second suspicious page = %#v", second)
	}

	contradictory, err := History(ctx, db, HistoryQuery{Status: StatusPending, RejectionReason: RejectionSuspicious})
	if err != nil {
		t.Fatal(err)
	}
	if len(contradictory.Items) != 0 || contradictory.NextBeforeID != nil {
		t.Fatalf("contradictory status/reason filter = %#v", contradictory)
	}

	pending, err := History(ctx, db, HistoryQuery{Status: StatusPending})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending.Items) != 1 || pending.Items[0].ID != created4.Attribution.ID {
		t.Fatalf("status-only history changed = %#v", pending.Items)
	}

	if countHistoryRows(t, db, "referral_attributions") != beforeAttributions || countHistoryRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("rejection-reason history read mutated authoritative state")
	}
}

func TestHistoryRejectsUnknownRejectionReason(t *testing.T) {
	db := openTestDB(t)
	if _, err := History(context.Background(), db, HistoryQuery{RejectionReason: RejectionReason("Suspicious")}); err == nil {
		t.Fatal("History accepted non-established rejection reason")
	}
}
