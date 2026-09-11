package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/referral"
)

func TestReferralPageExactStatusFilterAndPagination(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)
	inviter := resolveHTTPReferralUser(t, ctx, db, 9210)
	olderRejectedInvitee := resolveHTTPReferralUser(t, ctx, db, 9211)
	pendingInvitee := resolveHTTPReferralUser(t, ctx, db, 9212)
	newerRejectedInvitee := resolveHTTPReferralUser(t, ctx, db, 9213)
	base := time.Unix(1_831_000_000, 0).UTC()
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	olderRejected, err := referral.AttributeNewInvitee(ctx, db, olderRejectedInvitee, code.Value, base.Add(time.Second))
	if err != nil || olderRejected.Outcome != referral.OutcomeCreated {
		t.Fatalf("create older rejected attribution = %#v, %v", olderRejected, err)
	}
	pending, err := referral.AttributeNewInvitee(ctx, db, pendingInvitee, code.Value, base.Add(2*time.Second))
	if err != nil || pending.Outcome != referral.OutcomeCreated {
		t.Fatalf("create pending attribution = %#v, %v", pending, err)
	}
	newerRejected, err := referral.AttributeNewInvitee(ctx, db, newerRejectedInvitee, code.Value, base.Add(3*time.Second))
	if err != nil || newerRejected.Outcome != referral.OutcomeCreated {
		t.Fatalf("create newer rejected attribution = %#v, %v", newerRejected, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, olderRejectedInvitee.User.ID, referral.RejectionCooldown, base.Add(4*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject older attribution = %#v, %v", result, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, newerRejectedInvitee.User.ID, referral.RejectionBlacklist, base.Add(5*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject newer attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeSettings := countHTTPRows(t, db, "settings")

	response := perform(server.Handler(), http.MethodGet, "/referrals?status=rejected&limit=1", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("filtered referral page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`<option value="">All</option>`,
		`<option value="pending">Pending</option>`,
		`<option value="rewarded">Rewarded</option>`,
		`<option value="rejected" selected>Rejected</option>`,
		`name="limit" value="1"`,
		`if (statusFilter.value === '') statusFilter.disabled = true;`,
		">9213<",
		`href="/referrals?before_id=` + strconv.FormatInt(newerRejected.Attribution.ID, 10) + `&amp;limit=1&amp;status=rejected"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("filtered referral page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, ">9211<") || strings.Contains(body, ">9212<") {
		t.Fatalf("filtered first page included non-page/non-status row: %s", body)
	}

	olderPath := "/referrals?before_id=" + strconv.FormatInt(newerRejected.Attribution.ID, 10) + "&limit=1&status=rejected"
	older := perform(server.Handler(), http.MethodGet, olderPath, nil, cookie)
	if older.Code != http.StatusOK {
		t.Fatalf("older filtered referral page = %d %s", older.Code, older.Body.String())
	}
	olderBody := older.Body.String()
	if !strings.Contains(olderBody, ">9211<") || strings.Contains(olderBody, ">9212<") || strings.Contains(olderBody, ">9213<") {
		t.Fatalf("older filtered referral page mismatch: %s", olderBody)
	}

	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("filtered referral page mutated authoritative state")
	}
}
