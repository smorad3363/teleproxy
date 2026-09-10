package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestReferralRewardSettingsAPIRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/referral/reward-settings", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized GET = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingCSRF := performJSON(server.Handler(), http.MethodPut, "/api/referral/reward-settings", `{"bytes":2000000000,"expiry_days":14}`, cookie, "")
	if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("missing CSRF = %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestReferralRewardSettingsAPIGetAndUpdate(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)

	defaults := performJSON(server.Handler(), http.MethodGet, "/api/referral/reward-settings", "", cookie, "")
	if defaults.Code != http.StatusOK || !strings.Contains(defaults.Body.String(), `"bytes":2000000000`) || !strings.Contains(defaults.Body.String(), `"expiry_days":14`) {
		t.Fatalf("default GET = %d %s", defaults.Code, defaults.Body.String())
	}
	if got := defaults.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("default GET Cache-Control = %q", got)
	}

	updated := performJSON(server.Handler(), http.MethodPut, "/api/referral/reward-settings", `{"bytes":3500000000,"expiry_days":21}`, cookie, csrf)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"bytes":3500000000`) || !strings.Contains(updated.Body.String(), `"expiry_days":21`) {
		t.Fatalf("PUT = %d %s", updated.Code, updated.Body.String())
	}
	stored, err := settings.ReferralReward(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Bytes != 3_500_000_000 || stored.ExpiryDays != 21 {
		t.Fatalf("stored settings = %#v", stored)
	}

	for _, table := range []string{"referral_attributions", "credit_buckets"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("settings API mutated %s: count=%d", table, count)
		}
	}
}

func TestReferralRewardSettingsAPIRejectsBadRequestsWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	if err := settings.SetReferralReward(context.Background(), db, 2_750_000_000, 18); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		body string
		code int
		key  string
	}{
		{name: "malformed", body: `{"bytes":`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "unknown field", body: `{"bytes":2000000000,"expiry_days":14,"unknown":1}`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "zero bytes", body: `{"bytes":0,"expiry_days":14}`, code: http.StatusBadRequest, key: `"code":"REFERRAL_REWARD_INVALID"`},
		{name: "negative expiry", body: `{"bytes":2000000000,"expiry_days":-1}`, code: http.StatusBadRequest, key: `"code":"REFERRAL_REWARD_INVALID"`},
		{name: "oversized", body: strings.Repeat(" ", int(maxReferralRewardSettingsBodyBytes)+1) + `{}`, code: http.StatusRequestEntityTooLarge, key: `"code":"REQUEST_TOO_LARGE"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performJSON(server.Handler(), http.MethodPut, "/api/referral/reward-settings", test.body, cookie, csrf)
			if response.Code != test.code || !strings.Contains(response.Body.String(), test.key) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			stored, err := settings.ReferralReward(context.Background(), db)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Bytes != 2_750_000_000 || stored.ExpiryDays != 18 {
				t.Fatalf("bad request mutated settings: %#v", stored)
			}
		})
	}
}

func authenticatedReferralSettingsAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie, string) {
	t.Helper()
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}, sessionCSRF(token)
}
