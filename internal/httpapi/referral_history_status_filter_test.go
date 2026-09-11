package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/referral"
)

func TestReferralHistoryAPIExactStatusFilter(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)
	inviter := resolveHTTPReferralUser(t, ctx, db, 9110)
	pendingInvitee := resolveHTTPReferralUser(t, ctx, db, 9111)
	rejectedInvitee := resolveHTTPReferralUser(t, ctx, db, 9112)
	base := time.Unix(1_821_000_000, 0).UTC()
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := referral.AttributeNewInvitee(ctx, db, pendingInvitee, code.Value, base.Add(time.Second))
	if err != nil || pending.Outcome != referral.OutcomeCreated {
		t.Fatalf("create pending attribution = %#v, %v", pending, err)
	}
	rejected, err := referral.AttributeNewInvitee(ctx, db, rejectedInvitee, code.Value, base.Add(2*time.Second))
	if err != nil || rejected.Outcome != referral.OutcomeCreated {
		t.Fatalf("create rejected attribution = %#v, %v", rejected, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, rejectedInvitee.User.ID, referral.RejectionBlacklist, base.Add(3*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	for _, tc := range []struct {
		path   string
		wantID int64
		status referral.Status
	}{
		{path: "/api/referral/history?status=pending", wantID: pending.Attribution.ID, status: referral.StatusPending},
		{path: "/api/referral/history?status=rejected", wantID: rejected.Attribution.ID, status: referral.StatusRejected},
	} {
		response := performJSON(server.Handler(), http.MethodGet, tc.path, "", cookie, "")
		if response.Code != http.StatusOK {
			t.Fatalf("%s = %d %s", tc.path, response.Code, response.Body.String())
		}
		var page struct {
			History []referral.HistoryEntry `json:"history"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.History) != 1 || page.History[0].ID != tc.wantID || page.History[0].Status != tc.status {
			t.Fatalf("%s history = %#v", tc.path, page.History)
		}
	}

	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("status-filtered referral history API mutated authoritative state")
	}
}

func TestReferralHistoryAPIRejectsInvalidStatusFilter(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)
	for _, path := range []string{
		"/api/referral/history?status=",
		"/api/referral/history?status=Rejected",
		"/api/referral/history?status=unknown",
		"/api/referral/history?status=pending&status=rejected",
	} {
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"REFERRAL_HISTORY_INVALID"`) {
			t.Fatalf("invalid status query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}
