package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/smorad3363/teleproxy/internal/settings"
)

func TestStartGiftSettingsPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/settings", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated settings page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "csrf-token") || strings.Contains(response.Body.String(), "Start gift") {
		t.Fatalf("unauthenticated settings page exposed settings markup: %s", response.Body.String())
	}
}

func TestStartGiftSettingsPageRendersExactInt64AndAPIWiringWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	const exact = int64(9_007_199_254_740_993)
	if err := settings.SetStartGiftBytes(ctx, db, exact); err != nil {
		t.Fatal(err)
	}
	beforeSettings := countHTTPRows(t, db, "settings")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	response := perform(server.Handler(), http.MethodGet, "/settings", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("settings page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		`content="` + csrf + `"`,
		`value="9007199254740993"`,
		`fetch('/api/settings/start-gift'`,
		`'X-CSRF-Token': csrf`,
		`body: '{"bytes":' + bytes + '}'`,
		`/^[1-9][0-9]*$/`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings page missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{"Number(bytes)", "parseInt(bytes", "parseFloat(bytes"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("settings page uses lossy numeric conversion %q: %s", forbidden, body)
		}
	}
	if countHTTPRows(t, db, "settings") != beforeSettings || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("settings page rendering mutated state")
	}
}

func TestStartGiftSettingsPageDefaultAndDashboardLink(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	settingsPage := perform(server.Handler(), http.MethodGet, "/settings", nil, cookie)
	if settingsPage.Code != http.StatusOK || !strings.Contains(settingsPage.Body.String(), `value="100000000"`) {
		t.Fatalf("default settings page = %d %s", settingsPage.Code, settingsPage.Body.String())
	}
	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, cookie)
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), `href="/settings"`) {
		t.Fatalf("dashboard settings link = %d %s", dashboard.Code, dashboard.Body.String())
	}
}
