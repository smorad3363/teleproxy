package referral

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestHistoryNewestFirstPaginatesAndPreservesState(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 8101)
	invitee1 := resolveTestUser(t, ctx, db, 8102)
	invitee2 := resolveTestUser(t, ctx, db, 8103)
	invitee3 := resolveTestUser(t, ctx, db, 8104)
	base := time.Unix(1_810_000_000, 0).UTC()
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

	eligibleAt := base.Add(10 * time.Second)
	if result, err := ApproveEligibility(ctx, db, invitee2.User.ID, eligibleAt); err != nil || result.Outcome != EligibilityApproved {
		t.Fatalf("approve second attribution = %#v, %v", result, err)
	}
	rejectedAt := base.Add(11 * time.Second)
	if result, err := RejectEligibility(ctx, db, invitee3.User.ID, RejectionBlacklist, rejectedAt); err != nil || result.Outcome != EligibilityRejected {
		t.Fatalf("reject third attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHistoryRows(t, db, "referral_attributions")
	beforeCredits := countHistoryRows(t, db, "credit_buckets")

	first, err := History(ctx, db, HistoryQuery{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != created3.Attribution.ID || first.Items[1].ID != created2.Attribution.ID {
		t.Fatalf("first page order = %#v", first.Items)
	}
	if first.NextBeforeID == nil || *first.NextBeforeID != created2.Attribution.ID {
		t.Fatalf("first page next_before_id = %v", first.NextBeforeID)
	}
	if first.Items[0].InviterTelegramID != 8101 || first.Items[0].InviteeTelegramID != 8104 {
		t.Fatalf("history Telegram identities = %#v", first.Items[0])
	}
	if first.Items[0].Status != StatusRejected || first.Items[0].RejectionReason == nil || *first.Items[0].RejectionReason != string(RejectionBlacklist) || first.Items[0].FinalizedAt == nil || !first.Items[0].FinalizedAt.Equal(rejectedAt) {
		t.Fatalf("rejected history state = %#v", first.Items[0])
	}
	if first.Items[1].Status != StatusPending || first.Items[1].EligibleAt == nil || !first.Items[1].EligibleAt.Equal(eligibleAt) || first.Items[1].FinalizedAt != nil {
		t.Fatalf("eligible history state = %#v", first.Items[1])
	}

	second, err := History(ctx, db, HistoryQuery{BeforeID: *first.NextBeforeID, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != created1.Attribution.ID || second.NextBeforeID != nil {
		t.Fatalf("second page = %#v", second)
	}
	if countHistoryRows(t, db, "referral_attributions") != beforeAttributions || countHistoryRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("history read mutated referral or credit state")
	}
}

func TestHistoryEmptyAndRejectsUnboundedOptions(t *testing.T) {
	db := openTestDB(t)
	page, err := History(context.Background(), db, HistoryQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.NextBeforeID != nil {
		t.Fatalf("empty history page = %#v", page)
	}
	for _, query := range []HistoryQuery{
		{BeforeID: -1},
		{Limit: -1},
		{Limit: MaxHistoryLimit + 1},
	} {
		if _, err := History(context.Background(), db, query); err == nil {
			t.Fatalf("History(%#v) unexpectedly succeeded", query)
		}
	}
}

func countHistoryRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
