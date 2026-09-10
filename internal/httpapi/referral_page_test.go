package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/referral"
	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestReferralPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/referrals", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated referral page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "csrf-token") {
		t.Fatal("unauthenticated referral page exposed CSRF markup")
	}
}

func TestReferralPageRendersSettingsHistoryAndPaginationWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	if err := settings.SetReferralReward(ctx, db, 3_500_000_000, 21); err != nil {
		t.Fatal(err)
	}
	inviter := resolveHTTPReferralUser(t, ctx, db, 9201)
	invitee1 := resolveHTTPReferralUser(t, ctx, db, 9202)
	invitee2 := resolveHTTPReferralUser(t, ctx, db, 9203)
	base := time.Unix(1_830_000_000, 0).UTC()
	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, base)
	if err != nil {
		t.Fatal(err)
	}
	first, err := referral.AttributeNewInvitee(ctx, db, invitee1, code.Value, base.Add(time.Second))
	if err != nil || first.Outcome != referral.OutcomeCreated {
		t.Fatalf("create first attribution = %#v, %v", first, err)
	}
	second, err := referral.AttributeNewInvitee(ctx, db, invitee2, code.Value, base.Add(2*time.Second))
	if err != nil || second.Outcome != referral.OutcomeCreated {
		t.Fatalf("create second attribution = %#v, %v", second, err)
	}
	if result, err := referral.RejectEligibility(ctx, db, invitee2.User.ID, referral.RejectionCooldown, base.Add(3*time.Second)); err != nil || result.Outcome != referral.EligibilityRejected {
		t.Fatalf("reject second attribution = %#v, %v", result, err)
	}

	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeSettings := countHTTPRows(t, db, "settings")

	response := perform(server.Handler(), http.MethodGet, "/referrals?limit=1", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("referral page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Referral reward settings",
		`value="3500000000"`,
		`value="21"`,
		">9201<",
		">9203<",
		">rejected<",
		string(referral.RejectionCooldown),
		`content="` + csrf + `"`,
		`/api/referral/reward-settings`,
		`X-CSRF-Token`,
		`href="/referrals?before_id=` + strconv.FormatInt(second.Attribution.ID, 10) + `&amp;limit=1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("referral page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, ">9202<") {
		t.Fatalf("first page unexpectedly rendered older invitee: %s", body)
	}
	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("referral page rendering mutated authoritative state")
	}

	olderPath := "/referrals?before_id=" + strconv.FormatInt(second.Attribution.ID, 10) + "&limit=1"
	older := perform(server.Handler(), http.MethodGet, olderPath, nil, cookie)
	if older.Code != http.StatusOK {
		t.Fatalf("older referral page = %d %s", older.Code, older.Body.String())
	}
	olderBody := older.Body.String()
	if !strings.Contains(olderBody, ">9202<") || strings.Contains(olderBody, ">9203<") {
		t.Fatalf("older referral page pagination mismatch: %s", olderBody)
	}
	if countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("referral pagination created reward credit")
	}
}

func TestReferralPageRejectsInvalidPaginationWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)
	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeSettings := countHTTPRows(t, db, "settings")

	for _, path := range []string{
		"/referrals?before_id=0",
		"/referrals?limit=0",
		"/referrals?unknown=1",
	} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"REFERRAL_HISTORY_INVALID"`) {
			t.Fatalf("invalid referral page query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
	if countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("invalid referral page query mutated authoritative state")
	}
}

func TestDashboardLinksToReferralPage(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	response := perform(server.Handler(), http.MethodGet, "/", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `href="/referrals"`) {
		t.Fatalf("dashboard does not link to referrals: %s", response.Body.String())
	}
}
