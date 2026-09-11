package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
)

func TestAdminInventoryAPIRequiresAuthenticationAndIsNoStore(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/admins", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized admins = %d %s", unauthorized.Code, unauthorized.Body.String())
	}

	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cookie := []*http.Cookie{{Name: sessionCookieName, Value: token}}
	authorized := performJSON(server.Handler(), http.MethodGet, "/api/admins", "", cookie, "")
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized admins = %d %s", authorized.Code, authorized.Body.String())
	}
	if got := authorized.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("admins Cache-Control = %q", got)
	}
	if strings.Contains(authorized.Body.String(), "password_hash") || strings.Contains(authorized.Body.String(), token) {
		t.Fatalf("administrator inventory leaked secret material: %s", authorized.Body.String())
	}
}

func TestAdminInventoryAPIListsDeterministicallyWithoutMutation(t *testing.T) {
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
	result, err := db.ExecContext(ctx, `
		INSERT INTO admins(username, password_hash, role, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "support", "sentinel-password-hash", "support", 0, createdAt.Unix(), updatedAt.Unix())
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	beforeAdmins := countHTTPRows(t, db, "admins")
	beforeSessions := countHTTPRows(t, db, "admin_sessions")

	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	response := performJSON(server.Handler(), http.MethodGet, "/api/admins", "", []*http.Cookie{{Name: sessionCookieName, Value: token}}, "")
	if response.Code != http.StatusOK {
		t.Fatalf("admin inventory = %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Admins []admin.InventoryEntry `json:"admins"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Admins) != 2 || payload.Admins[0].ID != owner.ID || payload.Admins[1].ID != secondID {
		t.Fatalf("administrator inventory = %#v", payload.Admins)
	}
	if payload.Admins[1].Role != "support" || payload.Admins[1].Enabled || !payload.Admins[1].CreatedAt.Equal(createdAt) || !payload.Admins[1].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("second administrator = %#v", payload.Admins[1])
	}
	body := response.Body.String()
	for _, leaked := range []string{"sentinel-password-hash", "password_hash", "token_hash"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("administrator inventory leaked %q: %s", leaked, body)
		}
	}
	if countHTTPRows(t, db, "admins") != beforeAdmins || countHTTPRows(t, db, "admin_sessions") != beforeSessions {
		t.Fatal("administrator inventory API mutated authoritative state")
	}
}

func TestAdminInventoryAPIRejectsQueryWithoutMutation(t *testing.T) {
	db := testDB(t)
	owner, _, err := admin.BootstrapOwner(context.Background(), db, "admin", "generated-admin-password-123")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := admin.CreateSession(context.Background(), db, owner.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cookie := []*http.Cookie{{Name: sessionCookieName, Value: token}}
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)
	beforeAdmins := countHTTPRows(t, db, "admins")
	beforeSessions := countHTTPRows(t, db, "admin_sessions")

	for _, path := range []string{"/api/admins?limit=1", "/api/admins?unknown=1"} {
		invalid := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), `"code":"ADMIN_INVENTORY_INVALID"`) {
			t.Fatalf("invalid admin query %q = %d %s", path, invalid.Code, invalid.Body.String())
		}
	}
	if countHTTPRows(t, db, "admins") != beforeAdmins || countHTTPRows(t, db, "admin_sessions") != beforeSessions {
		t.Fatal("invalid administrator inventory query mutated authoritative state")
	}
}
