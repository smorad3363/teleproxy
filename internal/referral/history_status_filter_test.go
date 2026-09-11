package referral

import (
	"context"
	"testing"
	"time"
)

func TestHistoryFiltersExactStatusAndComposesWithCursor(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 8110)
	invitee1 := resolveTestUser(t, ctx, db, 8111)
	invitee2 := resolveTestUser(t, ctx, db, 8112)
	invitee3 := resolveTestUser(t, ctx, db, 8113)
	invitee4 := resolveTestUser(t, ctx, db, 8114)
	base := time.Unix(1_811_000_000, 0).UTC()
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
	if result, err := RejectEligibility(ctx, db, invitee3.User.ID, RejectionBlacklist, base.Add(10*time.Second)); err != nil || result.Outcome != EligibilityRejected {
		t.Fatalf("reject third attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHistoryRows(t, db, "referral_attributions")
	beforeCredits := countHistoryRows(t, db, "credit_buckets")

	first, err := History(ctx, db, HistoryQuery{Status: StatusPending, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != created4.Attribution.ID || first.Items[1].ID != created2.Attribution.ID {
		t.Fatalf("first pending page = %#v", first.Items)
	}
	if first.NextBeforeID == nil || *first.NextBeforeID != created2.Attribution.ID {
		t.Fatalf("pending next_before_id = %v", first.NextBeforeID)
	}
	second, err := History(ctx, db, HistoryQuery{Status: StatusPending, BeforeID: *first.NextBeforeID, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != created1.Attribution.ID || second.NextBeforeID != nil {
		t.Fatalf("second pending page = %#v", second)
	}

	rejected, err := History(ctx, db, HistoryQuery{Status: StatusRejected})
	if err != nil {
		t.Fatal(err)
	}
	if len(rejected.Items) != 1 || rejected.Items[0].ID != created3.Attribution.ID || rejected.Items[0].Status != StatusRejected {
		t.Fatalf("rejected history = %#v", rejected.Items)
	}

	if _, err := db.ExecContext(ctx, `UPDATE referral_attributions SET status = ?, finalized_at = ?, updated_at = ? WHERE id = ?`, string(StatusRewarded), base.Add(20*time.Second).Unix(), base.Add(20*time.Second).Unix(), created2.Attribution.ID); err != nil {
		t.Fatal(err)
	}
	rewarded, err := History(ctx, db, HistoryQuery{Status: StatusRewarded})
	if err != nil {
		t.Fatal(err)
	}
	if len(rewarded.Items) != 1 || rewarded.Items[0].ID != created2.Attribution.ID || rewarded.Items[0].Status != StatusRewarded {
		t.Fatalf("rewarded history = %#v", rewarded.Items)
	}

	if countHistoryRows(t, db, "referral_attributions") != beforeAttributions || countHistoryRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("filtered history read mutated authoritative row counts")
	}
}

func TestHistoryRejectsUnknownStatus(t *testing.T) {
	db := openTestDB(t)
	if _, err := History(context.Background(), db, HistoryQuery{Status: Status("Rejected")}); err == nil {
		t.Fatal("History accepted non-established status")
	}
}
