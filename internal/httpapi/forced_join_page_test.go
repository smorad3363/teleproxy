package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/forcedjoin"
)

func TestForcedJoinPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/forced-join", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated forced join page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "csrf-token") {
		t.Fatal("unauthenticated forced join page exposed CSRF markup")
	}
}

func TestForcedJoinPageRendersOrderedEscapedChannelsWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie, csrf := authenticatedForcedJoinAPI(t, db)
	base := time.Unix(1_840_000_000, 0).UTC()
	later, err := forcedjoin.Create(ctx, db, forcedjoin.CreateChannel{
		ChatRef: "@later", DisplayName: "Later", JoinURL: "https://t.me/later",
		Enabled: false, Required: false, Position: 20, CustomText: "Optional",
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	first, err := forcedjoin.Create(ctx, db, forcedjoin.CreateChannel{
		ChatRef: "@first", DisplayName: `First <script>alert("x")</script>`, JoinURL: "https://t.me/first",
		Enabled: true, Required: true, Position: 10, CustomText: `<img src=x onerror="alert(1)">`,
	}, base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	beforeChannels := countHTTPRows(t, db, "forced_join_channels")
	beforeAttributions := countHTTPRows(t, db, "referral_attributions")
	beforeCredits := countHTTPRows(t, db, "credit_buckets")

	response := perform(server.Handler(), http.MethodGet, "/forced-join", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("forced join page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Forced Join channels",
		"@first",
		"@later",
		`https://t.me/first`,
		`content="` + csrf + `"`,
		`/api/forced-join/channels`,
		`X-CSRF-Token`,
		`enabled: form.elements.enabled.checked`,
		`required: form.elements.required.checked`,
		`position: Number(form.elements.position.value)`,
		`data-id="` + strconv.FormatInt(first.ID, 10) + `"`,
		`data-id="` + strconv.FormatInt(later.ID, 10) + `"`,
		`First &lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`,
		`&lt;img src=x onerror=&#34;alert(1)&#34;&gt;`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("forced join page missing %q: %s", want, body)
		}
	}
	if strings.Index(body, "@first") > strings.Index(body, "@later") {
		t.Fatalf("forced join page order is not position,id: %s", body)
	}
	if strings.Contains(body, `<script>alert("x")</script>`) || strings.Contains(body, `<img src=x onerror="alert(1)">`) {
		t.Fatalf("forced join page rendered unescaped channel content: %s", body)
	}
	if countHTTPRows(t, db, "forced_join_channels") != beforeChannels || countHTTPRows(t, db, "referral_attributions") != beforeAttributions || countHTTPRows(t, db, "credit_buckets") != beforeCredits {
		t.Fatal("forced join page rendering mutated authoritative state")
	}
}

func TestForcedJoinPageEmptyStateAndDashboardLink(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedForcedJoinAPI(t, db)

	page := perform(server.Handler(), http.MethodGet, "/forced-join", nil, cookie)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "No Forced Join channels configured.") {
		t.Fatalf("empty forced join page = %d %s", page.Code, page.Body.String())
	}
	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, cookie)
	if dashboard.Code != http.StatusOK {
		t.Fatalf("dashboard = %d %s", dashboard.Code, dashboard.Body.String())
	}
	if !strings.Contains(dashboard.Body.String(), `href="/forced-join"`) {
		t.Fatalf("dashboard does not link to Forced Join: %s", dashboard.Body.String())
	}
}
