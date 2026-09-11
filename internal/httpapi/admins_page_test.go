package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
)

func TestAdminInventoryPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/admins", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated admins page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "Administrators") {
		t.Fatalf("unauthenticated admins page rendered inventory markup: %s", response.Body.String())
	}
}

func TestAdminInventoryPageRendersAuthoritativeInventoryWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(ctx, db, "owner", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(ctx, db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Unix(1_850_000_000, 0).UTC()
	updatedAt := createdAt.Add(time.Minute)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO admins(username, password_hash, role, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, `<unsafe&admin>`, "sentinel-password-hash", `<support&literal>`, 0, createdAt.Unix(), updatedAt.Unix()); err != nil {
		t.Fatal(err)
	}
	beforeAdmins := countHTTPRows(t, db, "admins")
	beforeSessions := countHTTPRows(t, db, "admin_sessions")
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	cookie := []*http.Cookie{{Name: sessionCookieName, Value: token}}

	response := perform(server.Handler(), http.MethodGet, "/admins", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("admins page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		">owner<",
		"&lt;unsafe&amp;admin&gt;",
		"&lt;support&amp;literal&gt;",
		">false<",
		createdAt.Format(time.RFC3339),
		updatedAt.Format(time.RFC3339),
		`href="/admins" aria-current="page"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("admins page missing %q: %s", want, body)
		}
	}
	if strings.Index(body, ">owner<") > strings.Index(body, "&lt;unsafe&amp;admin&gt;") {
		t.Fatalf("admins page is not ascending by ID: %s", body)
	}
	for _, leaked := range []string{"<unsafe&admin>", "<support&literal>", "sentinel-password-hash", "password_hash", "token_hash", token} {
		if strings.Contains(body, leaked) {
			t.Fatalf("admins page leaked %q: %s", leaked, body)
		}
	}
	if strings.Contains(body, "<form") || strings.Contains(body, "<button") {
		t.Fatalf("admins page unexpectedly contains mutation controls: %s", body)
	}
	if countHTTPRows(t, db, "admins") != beforeAdmins || countHTTPRows(t, db, "admin_sessions") != beforeSessions {
		t.Fatal("administrator page rendering mutated authoritative state")
	}
}

func TestAdminInventoryPageRejectsQueryWithoutMutation(t *testing.T) {
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	cookie := []*http.Cookie{{Name: sessionCookieName, Value: token}}
	beforeAdmins := countHTTPRows(t, db, "admins")
	beforeSessions := countHTTPRows(t, db, "admin_sessions")

	for _, path := range []string{"/admins?limit=1", "/admins?unknown=1"} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"ADMIN_INVENTORY_INVALID"`) {
			t.Fatalf("invalid admins page query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
	if countHTTPRows(t, db, "admins") != beforeAdmins || countHTTPRows(t, db, "admin_sessions") != beforeSessions {
		t.Fatal("invalid administrator page query mutated authoritative state")
	}
}

func TestAdminInventoryPageEmptyStateAndDashboardLink(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := adminInventoryPageTemplate.Execute(recorder, adminInventoryPageData{Username: "admin", Admins: []admin.InventoryEntry{}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recorder.Body.String(), "No administrators yet.") {
		t.Fatalf("empty administrators template = %s", recorder.Body.String())
	}

	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, []*http.Cookie{{Name: sessionCookieName, Value: token}})
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), `href="/admins"`) {
		t.Fatalf("dashboard admins link = %d %s", dashboard.Code, dashboard.Body.String())
	}
}
