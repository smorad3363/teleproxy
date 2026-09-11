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

func TestReferralHistoryAPIExactRejectionReasonFilter(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)
	inviter := resolveHTTPReferralUser(t, ctx, db, 9120)
	suspiciousInvitee := resolveHTTPReferralUser(t, ctx, db, 9121)
	cooldownInvitee := resolveHTTPReferralUser(t, ctx, db, 9122)
	base := time.Unix(1_822_000_000, 0).UTC()
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	suspicious, err := referral.AttributeNewInvitee(ctx, db, suspiciousInvitee, code.Value, base.Add(time.Second))
	if err != nil || suspicious.Outcome != referral.OutcomeCreated {
		t.Fatalf("create suspicious attribution = %#v, %v", suspicious, err)
	}
	cooldown, err := referral.AttributeNewInvitee(ctx, db, cooldownInvitee, code.Value, base.Add(2*time.Second))
	if err != nil || cooldown.Outcome != referral.OutcomeCreated {
		t.Fatalf("create cooldown attribution = %#v, %v", cooldown, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, suspiciousInvitee.User.ID, referral.RejectionSuspicious, base.Add(3*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject suspicious attribution = %#v, %v", result, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, cooldownInvitee.User.ID, referral.RejectionCooldown, base.Add(4*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject cooldown attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeSettings := countHTTPRows(t, db, "settings")

	response := performJSON(server.Handler(), http.MethodGet, "/api/referral/history?status=rejected&rejection_reason=suspicious", "", cookie, "")
	if response.Code != http.StatusOK {
		t.Fatalf("filtered history = %d %s", response.Code, response.Body.String())
	}
	var page struct {
		History []referral.HistoryEntry `json:"history"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.History) != 1 || page.History[0].ID != suspicious.Attribution.ID || page.History[0].RejectionReason == nil || *page.History[0].RejectionReason != string(referral.RejectionSuspicious) {
		t.Fatalf("suspicious history = %#v", page.History)
	}

	contradictory := performJSON(server.Handler(), http.MethodGet, "/api/referral/history?status=pending&rejection_reason=suspicious", "", cookie, "")
	if contradictory.Code != http.StatusOK {
		t.Fatalf("contradictory history = %d %s", contradictory.Code, contradictory.Body.String())
	}
	var empty struct {
		History []referral.HistoryEntry `json:"history"`
	}
	if err := json.Unmarshal(contradictory.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if len(empty.History) != 0 {
		t.Fatalf("contradictory filter inferred status: %#v", empty.History)
	}

	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("rejection-reason history API mutated authoritative state")
	}
}

func TestReferralHistoryAPIRejectsInvalidRejectionReasonFilter(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedReferralHistoryAPI(t, db)
	for _, path := range []string{
		"/api/referral/history?rejection_reason=",
		"/api/referral/history?rejection_reason=Suspicious",
		"/api/referral/history?rejection_reason=unknown",
		"/api/referral/history?rejection_reason=suspicious&rejection_reason=cooldown",
	} {
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"REFERRAL_HISTORY_INVALID"`) {
			t.Fatalf("invalid rejection reason query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}
