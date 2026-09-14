package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestPanelThemeWrapsAuthenticatedPanelPagesOnly(t *testing.T) {
	db := testDB(t)
	server, cookies, _ := authenticatedProxyNodeAPI(t, db)

	for _, path := range []string{"/", "/nodes", "/settings", "/forced-join"} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookies)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, response.Code, response.Body.String())
		}
		body := response.Body.String()
		for _, want := range []string{`id="tp-panel-theme"`, `id="tp-panel-shell"`, "tp-nav-link", "Teleproxy"} {
			if !strings.Contains(body, want) {
				t.Fatalf("GET %s missing %q: %s", path, want, body)
			}
		}
	}

	login := perform(server.Handler(), http.MethodGet, "/login", nil, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d %s", login.Code, login.Body.String())
	}
	if strings.Contains(login.Body.String(), `id="tp-panel-theme"`) || strings.Contains(login.Body.String(), `id="tp-panel-shell"`) {
		t.Fatalf("login unexpectedly received panel shell: %s", login.Body.String())
	}

	health := perform(server.Handler(), http.MethodGet, "/healthz", nil, nil)
	if health.Code != http.StatusOK || !strings.Contains(health.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("health = %d headers=%v body=%s", health.Code, health.Header(), health.Body.String())
	}
	if strings.Contains(health.Body.String(), "tp-panel") {
		t.Fatalf("health response was modified by panel theme: %s", health.Body.String())
	}
}

func TestPanelThemePreservesAuthenticationRedirect(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	response := perform(server.Handler(), http.MethodGet, "/nodes", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated themed page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "tp-panel") {
		t.Fatalf("redirect body was themed: %s", response.Body.String())
	}
}
