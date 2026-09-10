package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/proxynode"
)

func TestProxyNodeAdminRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedProxyNodeAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/nodes", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingCSRF := performJSON(server.Handler(), http.MethodPost, "/api/nodes", validProxyNodeJSON("proxy", "One", "DE", "telemt", "proxy.example.com", 443, "http://telemt:9091", true), cookie, "")
	if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("missing CSRF = %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestProxyNodeAdminCRUDOrderingAndNoEndpointContact(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedProxyNodeAPI(t, db)
	var endpointHits atomic.Int64
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		endpointHits.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer endpoint.Close()

	empty := performJSON(server.Handler(), http.MethodGet, "/api/nodes", "", cookie, "")
	if empty.Code != http.StatusOK || empty.Header().Get("Cache-Control") != "no-store" || !strings.Contains(empty.Body.String(), `"nodes":[]`) {
		t.Fatalf("empty list = %d %s headers=%v", empty.Code, empty.Body.String(), empty.Header())
	}

	first := performJSON(server.Handler(), http.MethodPost, "/api/nodes", validProxyNodeJSON(" PROXY ", " Germany-1 ", " Germany ", "TELEMT.INTERNAL.", "203.0.113.10", 443, endpoint.URL+"/api", true), cookie, csrf)
	if first.Code != http.StatusCreated {
		t.Fatalf("create first = %d %s", first.Code, first.Body.String())
	}
	second := performJSON(server.Handler(), http.MethodPost, "/api/nodes", validProxyNodeJSON("relay", "Iran-Relay-1", "Iran", "2001:db8::10", "RELAY.EXAMPLE.COM.", 8443, "https://[2001:db8::10]:9443", false), cookie, csrf)
	if second.Code != http.StatusCreated {
		t.Fatalf("create second = %d %s", second.Code, second.Body.String())
	}

	var firstBody struct {
		Node proxynode.Node `json:"node"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}
	var secondBody struct {
		Node proxynode.Node `json:"node"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatal(err)
	}
	if firstBody.Node.Type != proxynode.TypeProxy || firstBody.Node.Name != "Germany-1" || firstBody.Node.Host != "telemt.internal" {
		t.Fatalf("canonical first node = %#v", firstBody.Node)
	}
	if secondBody.Node.PublicHost != "relay.example.com" {
		t.Fatalf("canonical second node = %#v", secondBody.Node)
	}

	listed := performJSON(server.Handler(), http.MethodGet, "/api/nodes", "", cookie, "")
	var listBody struct {
		Nodes []proxynode.Node `json:"nodes"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	if listed.Code != http.StatusOK || len(listBody.Nodes) != 2 || listBody.Nodes[0].ID != firstBody.Node.ID || listBody.Nodes[1].ID != secondBody.Node.ID {
		t.Fatalf("list = %d %#v", listed.Code, listBody.Nodes)
	}

	updated := performJSON(server.Handler(), http.MethodPut, "/api/nodes/"+strconv.FormatInt(firstBody.Node.ID, 10), validProxyNodeJSON("relay", "Germany-Relay", "DE", "relay.internal", "relay-de.example.com", 9443, endpoint.URL+"/v1", false), cookie, csrf)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"name":"Germany-Relay"`) || !strings.Contains(updated.Body.String(), `"enabled":false`) {
		t.Fatalf("update = %d %s", updated.Code, updated.Body.String())
	}

	deleted := performJSON(server.Handler(), http.MethodDelete, "/api/nodes/"+strconv.FormatInt(secondBody.Node.ID, 10), "", cookie, csrf)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", deleted.Code, deleted.Body.String())
	}
	if endpointHits.Load() != 0 {
		t.Fatalf("Node CRUD contacted internal endpoint %d times", endpointHits.Load())
	}
}

func TestProxyNodeAdminRejectsInvalidConflictNotFoundAndOversizedWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedProxyNodeAPI(t, db)
	created := performJSON(server.Handler(), http.MethodPost, "/api/nodes", validProxyNodeJSON("proxy", "One", "DE", "telemt", "proxy.example.com", 443, "http://telemt:9091", true), cookie, csrf)
	if created.Code != http.StatusCreated {
		t.Fatalf("create fixture = %d %s", created.Code, created.Body.String())
	}
	before := proxyNodeCountHTTP(t, db)

	tests := []struct {
		name string
		body string
		code int
		key  string
	}{
		{name: "unknown field", body: `{"node_type":"proxy","name":"Bad","region":"DE","host":"telemt2","public_host":"proxy2.example.com","mtproto_port":443,"internal_api_endpoint":"http://telemt2:9091","enabled":true,"unknown":1}`, code: http.StatusBadRequest, key: `"code":"BAD_REQUEST"`},
		{name: "missing enabled", body: `{"node_type":"proxy","name":"Bad","region":"DE","host":"telemt2","public_host":"proxy2.example.com","mtproto_port":443,"internal_api_endpoint":"http://telemt2:9091"}`, code: http.StatusBadRequest, key: `"code":"PROXY_NODE_INVALID"`},
		{name: "bad type", body: validProxyNodeJSON("unknown", "Bad", "DE", "telemt2", "proxy2.example.com", 443, "http://telemt2:9091", true), code: http.StatusBadRequest, key: `"code":"PROXY_NODE_INVALID"`},
		{name: "bad host", body: validProxyNodeJSON("proxy", "Bad", "DE", "bad host", "proxy2.example.com", 443, "http://telemt2:9091", true), code: http.StatusBadRequest, key: `"code":"PROXY_NODE_INVALID"`},
		{name: "api credentials", body: validProxyNodeJSON("proxy", "Bad", "DE", "telemt2", "proxy2.example.com", 443, "http://user:pass@telemt2:9091", true), code: http.StatusBadRequest, key: `"code":"PROXY_NODE_INVALID"`},
		{name: "oversized", body: `{"node_type":"proxy","name":"` + strings.Repeat("x", int(maxProxyNodeBodyBytes)) + `","region":"DE","host":"telemt2","public_host":"proxy2.example.com","mtproto_port":443,"internal_api_endpoint":"http://telemt2:9091","enabled":true}`, code: http.StatusRequestEntityTooLarge, key: `"code":"REQUEST_TOO_LARGE"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performJSON(server.Handler(), http.MethodPost, "/api/nodes", test.body, cookie, csrf)
			if response.Code != test.code || !strings.Contains(response.Body.String(), test.key) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if proxyNodeCountHTTP(t, db) != before {
				t.Fatal("invalid Node request mutated state")
			}
		})
	}

	conflict := performJSON(server.Handler(), http.MethodPost, "/api/nodes", validProxyNodeJSON("relay", "oNe", "IR", "relay", "relay.example.com", 8443, "http://relay:9091", false), cookie, csrf)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"PROXY_NODE_CONFLICT"`) {
		t.Fatalf("conflict = %d %s", conflict.Code, conflict.Body.String())
	}
	if proxyNodeCountHTTP(t, db) != before {
		t.Fatal("conflict mutated Node count")
	}

	notFound := performJSON(server.Handler(), http.MethodDelete, "/api/nodes/999999", "", cookie, csrf)
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), `"code":"PROXY_NODE_NOT_FOUND"`) {
		t.Fatalf("not found = %d %s", notFound.Code, notFound.Body.String())
	}
	badID := performJSON(server.Handler(), http.MethodDelete, "/api/nodes/nope", "", cookie, csrf)
	if badID.Code != http.StatusBadRequest || !strings.Contains(badID.Body.String(), `"code":"PROXY_NODE_INVALID_ID"`) {
		t.Fatalf("bad id = %d %s", badID.Code, badID.Body.String())
	}
}

func authenticatedProxyNodeAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie, string) {
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

func validProxyNodeJSON(nodeType, name, region, host, publicHost string, port int, endpoint string, enabled bool) string {
	body, err := json.Marshal(map[string]any{
		"node_type": nodeType, "name": name, "region": region, "host": host, "public_host": publicHost,
		"mtproto_port": port, "internal_api_endpoint": endpoint, "enabled": enabled,
	})
	if err != nil {
		panic(err)
	}
	return string(body)
}

func proxyNodeCountHTTP(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM proxy_nodes").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
