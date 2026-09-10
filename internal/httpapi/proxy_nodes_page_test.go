package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxynode"
)

func TestProxyNodePageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/nodes", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated Proxy Node page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "csrf-token") {
		t.Fatal("unauthenticated Proxy Node page exposed CSRF markup")
	}
}

func TestProxyNodePageRendersCanonicalNodesEscapedNoStoreAndDoesNotProbe(t *testing.T) {
	db := testDB(t)
	server, cookies, csrf := authenticatedProxyNodeAPI(t, db)
	var endpointHits atomic.Int64
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		endpointHits.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer endpoint.Close()

	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	first, err := proxynode.Create(context.Background(), db, proxynode.CreateNode{
		Type: proxynode.Type(" PROXY "), Name: `Alpha <script>alert("x")</script>`, Region: `DE <West>`,
		Host: "TELEMT.INTERNAL.", PublicHost: "PROXY.EXAMPLE.COM.", MTProtoPort: 443,
		InternalAPIEndpoint: endpoint.URL + "/v1", Enabled: true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := proxynode.Create(context.Background(), db, proxynode.CreateNode{
		Type: proxynode.TypeRelay, Name: "Iran-Relay-1", Region: "Iran", Host: "2001:DB8::10",
		PublicHost: "RELAY.EXAMPLE.COM.", MTProtoPort: 8443, InternalAPIEndpoint: "https://[2001:db8::10]:9443/api", Enabled: false,
	}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	response := perform(server.Handler(), http.MethodGet, "/nodes", nil, cookies)
	if response.Code != http.StatusOK {
		t.Fatalf("Proxy Node page = %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Proxy Nodes",
		"telemt.internal",
		"proxy.example.com",
		"2001:db8::10",
		"relay.example.com",
		`X-CSRF-Token`,
		`/api/nodes`,
		`content="` + csrf + `"`,
		`data-id="` + strconv.FormatInt(first.ID, 10) + `"`,
		`data-id="` + strconv.FormatInt(second.ID, 10) + `"`,
		`value="relay" selected`,
		`id="create-node"`,
		`class="edit-node"`,
		`class="danger delete-node"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Proxy Node page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, `<script>alert("x")</script>`) || strings.Contains(body, `DE <West>`) {
		t.Fatalf("Proxy Node page rendered unescaped metadata: %s", body)
	}
	if !strings.Contains(body, `Alpha &lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`) || !strings.Contains(body, `DE &lt;West&gt;`) {
		t.Fatalf("Proxy Node metadata was not safely escaped: %s", body)
	}
	if strings.Index(body, `data-id="`+strconv.FormatInt(first.ID, 10)+`"`) > strings.Index(body, `data-id="`+strconv.FormatInt(second.ID, 10)+`"`) {
		t.Fatalf("Proxy Node page order is not deterministic by id: %s", body)
	}
	if endpointHits.Load() != 0 {
		t.Fatalf("Proxy Node page contacted configured internal endpoint %d times", endpointHits.Load())
	}
}

func TestProxyNodePageEmptyStateAndDashboardLink(t *testing.T) {
	db := testDB(t)
	server, cookies, _ := authenticatedProxyNodeAPI(t, db)

	page := perform(server.Handler(), http.MethodGet, "/nodes", nil, cookies)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "No Proxy Nodes configured.") {
		t.Fatalf("empty Proxy Node page = %d %s", page.Code, page.Body.String())
	}

	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, cookies)
	if dashboard.Code != http.StatusOK {
		t.Fatalf("dashboard = %d %s", dashboard.Code, dashboard.Body.String())
	}
	if !strings.Contains(dashboard.Body.String(), `href="/nodes"`) {
		t.Fatalf("dashboard does not link to Proxy Nodes: %s", dashboard.Body.String())
	}
}
