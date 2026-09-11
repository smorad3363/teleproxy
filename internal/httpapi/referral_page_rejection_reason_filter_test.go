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

func TestReferralPageExactRejectionReasonFilterAndPagination(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)
	inviter := resolveHTTPReferralUser(t, ctx, db, 9220)
	olderSuspiciousInvitee := resolveHTTPReferralUser(t, ctx, db, 9221)
	cooldownInvitee := resolveHTTPReferralUser(t, ctx, db, 9222)
	newerSuspiciousInvitee := resolveHTTPReferralUser(t, ctx, db, 9223)
	base := time.Unix(1_832_000_000, 0).UTC()
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	olderSuspicious, err := referral.AttributeNewInvitee(ctx, db, olderSuspiciousInvitee, code.Value, base.Add(time.Second))
	if err != nil || olderSuspicious.Outcome != referral.OutcomeCreated {
		t.Fatalf("create older suspicious attribution = %#v, %v", olderSuspicious, err)
	}
	cooldown, err := referral.AttributeNewInvitee(ctx, db, cooldownInvitee, code.Value, base.Add(2*time.Second))
	if err != nil || cooldown.Outcome != referral.OutcomeCreated {
		t.Fatalf("create cooldown attribution = %#v, %v", cooldown, err)
	}
	newerSuspicious, err := referral.AttributeNewInvitee(ctx, db, newerSuspiciousInvitee, code.Value, base.Add(3*time.Second))
	if err != nil || newerSuspicious.Outcome != referral.OutcomeCreated {
		t.Fatalf("create newer suspicious attribution = %#v, %v", newerSuspicious, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, olderSuspiciousInvitee.User.ID, referral.RejectionSuspicious, base.Add(4*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject older suspicious attribution = %#v, %v", result, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, cooldownInvitee.User.ID, referral.RejectionCooldown, base.Add(5*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject cooldown attribution = %#v, %v", result, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, newerSuspiciousInvitee.User.ID, referral.RejectionSuspicious, base.Add(6*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject newer suspicious attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeSettings := countHTTPRows(t, db, "settings")

	response := perform(server.Handler(), http.MethodGet, "/referrals?status=rejected&rejection_reason=suspicious&limit=1", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("filtered referral page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`<option value=""` + `>All</option>`,
		`<option value="anti_abuse">Anti abuse</option>`,
		`<option value="daily_cap">Daily cap</option>`,
		`<option value="weekly_cap">Weekly cap</option>`,
		`<option value="cooldown">Cooldown</option>`,
		`<option value="blacklist">Blacklist</option>`,
		`<option value="suspicious" selected>Suspicious</option>`,
		`if (rejectionReasonFilter.value === '') rejectionReasonFilter.disabled = true;`,
		">9223<",
		`href="/referrals?before_id=` + strconv.FormatInt(newerSuspicious.Attribution.ID, 10) + `&amp;limit=1&amp;rejection_reason=suspicious&amp;status=rejected"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("filtered referral page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, ">9221<") || strings.Contains(body, ">9222<") {
		t.Fatalf("filtered first page included non-page/non-reason row: %s", body)
	}

	olderPath := "/referrals?before_id=" + strconv.FormatInt(newerSuspicious.Attribution.ID, 10) + "&limit=1&rejection_reason=suspicious&status=rejected"
	older := perform(server.Handler(), http.MethodGet, olderPath, nil, cookie)
	if older.Code != http.StatusOK {
		t.Fatalf("older filtered referral page = %d %s", older.Code, older.Body.String())
	}
	olderBody := older.Body.String()
	if !strings.Contains(olderBody, ">9221<") || strings.Contains(olderBody, ">9222<") || strings.Contains(olderBody, ">9223<") {
		t.Fatalf("older filtered referral page mismatch: %s", olderBody)
	}

	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("rejection-reason filtered referral page mutated authoritative state")
	}
}
