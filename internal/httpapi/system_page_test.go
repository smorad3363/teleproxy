package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

func TestSystemPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyHealth(db, Options{}, fakeProxyHealth(func(context.Context) telemt.Health {
		return telemt.Health{State: telemt.StateHealthy}
	}))

	response := perform(server.Handler(), http.MethodGet, "/system", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated system page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "Global Proxy") {
		t.Fatalf("unauthenticated system page rendered status markup: %s", response.Body.String())
	}
}

func TestSystemPageRendersEstablishedHealthWithoutMutation(t *testing.T) {
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyHealth(db, Options{}, fakeProxyHealth(func(context.Context) telemt.Health {
		return telemt.Health{State: telemt.StateHealthy, ReadOnly: true}
	}))
	cookies := []*http.Cookie{{Name: sessionCookieName, Value: token}}
	beforeAudit := countHTTPRows(t, db, "audit_log")
	beforeSettings := countHTTPRows(t, db, "settings")

	response := perform(server.Handler(), http.MethodGet, "/system", nil, cookies)
	if response.Code != http.StatusOK {
		t.Fatalf("system page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{"Control Plane", "online", "Database", "ready", "Global Proxy", "healthy", "Proxy read-only mode", "yes", owner.Username} {
		if !strings.Contains(body, want) {
			t.Fatalf("system page missing %q: %s", want, body)
		}
	}
	if countHTTPRows(t, db, "audit_log") != beforeAudit || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("system page rendering mutated authoritative state")
	}

	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, cookies)
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), `href="/system"`) {
		t.Fatalf("dashboard system link = %d %s", dashboard.Code, dashboard.Body.String())
	}
}

func TestSystemPageRendersNotConfiguredProxySafely(t *testing.T) {
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyHealth(db, Options{}, nil)

	response := perform(server.Handler(), http.MethodGet, "/system", nil, []*http.Cookie{{Name: sessionCookieName, Value: token}})
	if response.Code != http.StatusOK {
		t.Fatalf("system page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "not_configured") || !strings.Contains(body, "not applicable") {
		t.Fatalf("system page missing not-configured proxy state: %s", body)
	}
}

func TestSystemDatabaseStateMatchesReadySemantics(t *testing.T) {
	if got := New(nil, Options{}).systemDatabaseState(); got != "ready" {
		t.Fatalf("nil database state = %q", got)
	}

	db := testDB(t)
	server := New(db, Options{})
	if got := server.systemDatabaseState(); got != "ready" {
		t.Fatalf("open database state = %q", got)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if got := server.systemDatabaseState(); got != "unavailable" {
		t.Fatalf("closed database state = %q", got)
	}
}
