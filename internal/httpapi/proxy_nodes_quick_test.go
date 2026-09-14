package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/smorad3363/teleproxy/internal/proxynode"
)

func TestProxyNodeQuickCreateMirrorsPublicAddressAndUsesLegacyEndpointDefault(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedProxyNodeAPI(t, db)
	body := `{"node_type":"proxy","name":"Quick DE","region":"DE","public_host":"proxy.example.com","mtproto_port":443,"enabled":true}`

	response := performJSON(server.Handler(), http.MethodPost, "/api/nodes", body, cookie, csrf)
	if response.Code != http.StatusCreated {
		t.Fatalf("quick create = %d %s", response.Code, response.Body.String())
	}
	var result struct {
		Node proxynode.Node `json:"node"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Node.Host != "proxy.example.com" || result.Node.PublicHost != "proxy.example.com" {
		t.Fatalf("quick addresses = host=%q public=%q", result.Node.Host, result.Node.PublicHost)
	}
	if result.Node.InternalAPIEndpoint != defaultProxyNodeAPIEndpoint {
		t.Fatalf("endpoint = %q, want %q", result.Node.InternalAPIEndpoint, defaultProxyNodeAPIEndpoint)
	}
}

func TestProxyNodeQuickCreateStillRejectsInvalidPublicAddress(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedProxyNodeAPI(t, db)
	body := `{"node_type":"proxy","name":"Bad","region":"DE","public_host":"bad host","mtproto_port":443,"enabled":true}`

	response := performJSON(server.Handler(), http.MethodPost, "/api/nodes", body, cookie, csrf)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"PROXY_NODE_INVALID"`) {
		t.Fatalf("invalid quick create = %d %s", response.Code, response.Body.String())
	}
	if proxyNodeCountHTTP(t, db) != 0 {
		t.Fatal("invalid quick create mutated state")
	}
}
