package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/botcontent"
)

func TestBotContentPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/bot-content", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated Bot Content page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "csrf-token") {
		t.Fatal("unauthenticated Bot Content page exposed CSRF markup")
	}
}

func TestBotContentPageRendersAllSlotsConfiguredTextAndAPIWiringWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedReferralSettingsAPI(t, db)
	now := time.Unix(1_850_000_000, 0).UTC()
	if _, err := botcontent.Set(ctx, db, botcontent.SlotWelcome, `<script>alert("x")</script> & سلام`, now); err != nil {
		t.Fatal(err)
	}
	beforeContent := countHTTPRows(t, db, "bot_content")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")
	beforeReferrals := countHTTPRows(t, db, "referral_attributions")

	response := perform(server.Handler(), http.MethodGet, "/bot-content", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("Bot Content page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, slot := range []string{"welcome", "forced_join", "referral", "proxy", "expired", "no_credit", "support"} {
		if !strings.Contains(body, `data-bot-slot="`+slot+`"`) {
			t.Fatalf("Bot Content page missing slot %q: %s", slot, body)
		}
	}
	for _, want := range []string{
		`content="` + csrf + `"`,
		`&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt; &amp; سلام`,
		`Not configured`,
		`'/api/bot-content/' + encodeURIComponent(slot)`,
		`method: 'PUT'`,
		`method: 'DELETE'`,
		`X-CSRF-Token`,
		`JSON.stringify({text: textarea.value})`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Bot Content page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, `<script>alert("x")</script>`) {
		t.Fatalf("Bot Content page rendered stored HTML unsafely: %s", body)
	}
	if countHTTPRows(t, db, "bot_content") != beforeContent || countHTTPRows(t, db, "credit_buckets") != beforeCredits || countHTTPRows(t, db, "referral_attributions") != beforeReferrals {
		t.Fatal("Bot Content page rendering mutated authoritative state")
	}
}

func TestBotContentPageEmptyStateDoesNotFabricateCopy(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	response := perform(server.Handler(), http.MethodGet, "/bot-content", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("Bot Content page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if got := strings.Count(body, "Not configured"); got != 7 {
		t.Fatalf("Not configured count = %d, want 7: %s", got, body)
	}
	for _, fabricated := range []string{"Welcome to", "Join the channel", "Your proxy", "No credit remaining"} {
		if strings.Contains(body, fabricated) {
			t.Fatalf("empty Bot Content page fabricated copy %q: %s", fabricated, body)
		}
	}
	if countHTTPRows(t, db, "bot_content") != 0 {
		t.Fatal("empty Bot Content page created overrides")
	}
}

func TestDashboardLinksToBotContentPage(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedReferralSettingsAPI(t, db)

	response := perform(server.Handler(), http.MethodGet, "/", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `href="/bot-content"`) {
		t.Fatalf("dashboard does not link to Bot Content: %s", response.Body.String())
	}
}
