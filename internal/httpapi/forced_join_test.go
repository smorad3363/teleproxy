package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
)

func TestForcedJoinAdminRequiresAuthenticationAndCSRF(t *testing.T) {
	db := testDB(t)
	server, cookie, _ := authenticatedForcedJoinAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/forced-join/channels", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	missingCSRF := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", validForcedJoinJSON("@one", "One", 1), cookie, "")
	if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), `"code":"CSRF_INVALID"`) {
		t.Fatalf("missing CSRF = %d %s", missingCSRF.Code, missingCSRF.Body.String())
	}
}

func TestForcedJoinAdminCRUDAndOrdering(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedForcedJoinAPI(t, db)

	later := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", validForcedJoinJSON("@later", "Later", 20), cookie, csrf)
	if later.Code != http.StatusCreated {
		t.Fatalf("create later = %d %s", later.Code, later.Body.String())
	}
	first := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", `{"chat_ref":"@first","display_name":"First","join_url":"https://t.me/first","enabled":false,"required":false,"position":10,"custom_text":"Optional"}`, cookie, csrf)
	if first.Code != http.StatusCreated {
		t.Fatalf("create first = %d %s", first.Code, first.Body.String())
	}

	listed := performJSON(server.Handler(), http.MethodGet, "/api/forced-join/channels", "", cookie, "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list = %d %s", listed.Code, listed.Body.String())
	}
	body := listed.Body.String()
	if strings.Index(body, `"chat_ref":"@first"`) > strings.Index(body, `"chat_ref":"@later"`) || !strings.Contains(body, `"enabled":false`) || !strings.Contains(body, `"required":false`) {
		t.Fatalf("list ordering/state = %s", body)
	}

	var laterID int64
	if err := db.QueryRow("SELECT id FROM forced_join_channels WHERE chat_ref = '@later'").Scan(&laterID); err != nil {
		t.Fatal(err)
	}
	updated := performJSON(server.Handler(), http.MethodPut, "/api/forced-join/channels/"+forcedJoinIntString(laterID), `{"chat_ref":"@later_new","display_name":"Later new","join_url":"https://t.me/later_new","enabled":true,"required":true,"position":1,"custom_text":"Join now"}`, cookie, csrf)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"chat_ref":"@later_new"`) || !strings.Contains(updated.Body.String(), `"position":1`) {
		t.Fatalf("update = %d %s", updated.Code, updated.Body.String())
	}

	deleted := performJSON(server.Handler(), http.MethodDelete, "/api/forced-join/channels/"+forcedJoinIntString(laterID), "", cookie, csrf)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", deleted.Code, deleted.Body.String())
	}
	listed = performJSON(server.Handler(), http.MethodGet, "/api/forced-join/channels", "", cookie, "")
	if strings.Contains(listed.Body.String(), `"chat_ref":"@later_new"`) || !strings.Contains(listed.Body.String(), `"chat_ref":"@first"`) {
		t.Fatalf("delete affected wrong rows: %s", listed.Body.String())
	}
}

func TestForcedJoinAdminRejectsInvalidConflictNotFoundAndOversized(t *testing.T) {
	db := testDB(t)
	server, cookie, csrf := authenticatedForcedJoinAPI(t, db)

	unknown := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", `{"chat_ref":"@one","display_name":"One","join_url":"https://t.me/one","enabled":true,"required":true,"position":1,"unknown":1}`, cookie, csrf)
	if unknown.Code != http.StatusBadRequest || !strings.Contains(unknown.Body.String(), `"code":"BAD_REQUEST"`) {
		t.Fatalf("unknown field = %d %s", unknown.Code, unknown.Body.String())
	}
	missingState := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", `{"chat_ref":"@one","display_name":"One","join_url":"https://t.me/one"}`, cookie, csrf)
	if missingState.Code != http.StatusBadRequest || !strings.Contains(missingState.Body.String(), `"code":"FORCED_JOIN_INVALID"`) {
		t.Fatalf("missing state = %d %s", missingState.Code, missingState.Body.String())
	}
	unsafe := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", `{"chat_ref":"@one","display_name":"One","join_url":"https://example.com/one","enabled":true,"required":true,"position":1}`, cookie, csrf)
	if unsafe.Code != http.StatusBadRequest || !strings.Contains(unsafe.Body.String(), `"code":"FORCED_JOIN_INVALID"`) {
		t.Fatalf("unsafe = %d %s", unsafe.Code, unsafe.Body.String())
	}

	created := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", validForcedJoinJSON("@one", "One", 1), cookie, csrf)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	conflict := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", validForcedJoinJSON("@one", "Duplicate", 2), cookie, csrf)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"code":"FORCED_JOIN_CONFLICT"`) {
		t.Fatalf("conflict = %d %s", conflict.Code, conflict.Body.String())
	}

	notFound := performJSON(server.Handler(), http.MethodDelete, "/api/forced-join/channels/999999", "", cookie, csrf)
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), `"code":"FORCED_JOIN_NOT_FOUND"`) {
		t.Fatalf("not found = %d %s", notFound.Code, notFound.Body.String())
	}
	badID := performJSON(server.Handler(), http.MethodDelete, "/api/forced-join/channels/nope", "", cookie, csrf)
	if badID.Code != http.StatusBadRequest || !strings.Contains(badID.Body.String(), `"code":"FORCED_JOIN_INVALID_ID"`) {
		t.Fatalf("bad id = %d %s", badID.Code, badID.Body.String())
	}

	oversized := performJSON(server.Handler(), http.MethodPost, "/api/forced-join/channels", `{"chat_ref":"@big","display_name":"`+strings.Repeat("x", int(maxForcedJoinBodyBytes))+`","join_url":"https://t.me/big","enabled":true,"required":true,"position":1}`, cookie, csrf)
	if oversized.Code != http.StatusRequestEntityTooLarge || !strings.Contains(oversized.Body.String(), `"code":"REQUEST_TOO_LARGE"`) {
		t.Fatalf("oversized = %d %s", oversized.Code, oversized.Body.String())
	}
}

func authenticatedForcedJoinAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie, string) {
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

func validForcedJoinJSON(chatRef, displayName string, position int) string {
	return `{"chat_ref":"` + chatRef + `","display_name":"` + displayName + `","join_url":"https://t.me/` + strings.TrimPrefix(chatRef, "@") + `","enabled":true,"required":true,"position":` + forcedJoinIntString(int64(position)) + `,"custom_text":""}`
}

func forcedJoinIntString(value int64) string {
	return strconv.FormatInt(value, 10)
}
