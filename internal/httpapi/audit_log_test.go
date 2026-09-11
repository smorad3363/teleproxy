package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/auditlog"
)

func TestAuditLogAPIRequiresAuthenticationAndIsNoStore(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedAuditLogAPI(t, db)

	unauthorized := performJSON(server.Handler(), http.MethodGet, "/api/audit-log", "", nil, "")
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Fatalf("unauthorized audit log = %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	authorized := performJSON(server.Handler(), http.MethodGet, "/api/audit-log", "", cookie, "")
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized audit log = %d %s", authorized.Code, authorized.Body.String())
	}
	if got := authorized.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("audit log Cache-Control = %q", got)
	}
	if !strings.Contains(authorized.Body.String(), `"entries":[]`) {
		t.Fatalf("empty audit log = %s", authorized.Body.String())
	}
}

func TestAuditLogAPIPaginatesNewestFirstWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedAuditLogAPI(t, db)
	base := time.Unix(1_850_000_000, 0).UTC()

	first, err := auditlog.Append(ctx, db, auditlog.AppendInput{Actor: "admin:owner", Action: "settings.update", Target: "settings:start_gift", RequestID: "req-a"}, base)
	if err != nil {
		t.Fatal(err)
	}
	before := `{"enabled":false}`
	after := `{"enabled":true}`
	second, err := auditlog.Append(ctx, db, auditlog.AppendInput{Actor: "admin:owner", Action: "node.update", Target: "node:2", BeforeSnapshot: &before, AfterSnapshot: &after, RequestID: "req-b"}, base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	beforeAudit := countHTTPRows(t, db, "audit_log")
	beforeSettings := countHTTPRows(t, db, "settings")

	firstResponse := performJSON(server.Handler(), http.MethodGet, "/api/audit-log?limit=1", "", cookie, "")
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first audit log page = %d %s", firstResponse.Code, firstResponse.Body.String())
	}
	var firstPage struct {
		Entries      []auditlog.Entry `json:"entries"`
		NextBeforeID *int64           `json:"next_before_id"`
	}
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Entries) != 1 || firstPage.Entries[0].ID != second.ID || firstPage.Entries[0].RequestID != "req-b" {
		t.Fatalf("first audit log page = %#v", firstPage)
	}
	if firstPage.Entries[0].BeforeSnapshot == nil || *firstPage.Entries[0].BeforeSnapshot != before || firstPage.Entries[0].AfterSnapshot == nil || *firstPage.Entries[0].AfterSnapshot != after {
		t.Fatalf("first audit log snapshots = %#v", firstPage.Entries[0])
	}
	if firstPage.NextBeforeID == nil || *firstPage.NextBeforeID != second.ID {
		t.Fatalf("first next_before_id = %v", firstPage.NextBeforeID)
	}

	secondResponse := performJSON(server.Handler(), http.MethodGet, "/api/audit-log?limit=1&before_id="+strconv.FormatInt(*firstPage.NextBeforeID, 10), "", cookie, "")
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second audit log page = %d %s", secondResponse.Code, secondResponse.Body.String())
	}
	var secondPage struct {
		Entries      []auditlog.Entry `json:"entries"`
		NextBeforeID *int64           `json:"next_before_id"`
	}
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Entries) != 1 || secondPage.Entries[0].ID != first.ID || secondPage.Entries[0].RequestID != "req-a" || secondPage.NextBeforeID != nil {
		t.Fatalf("second audit log page = %#v", secondPage)
	}
	if countHTTPRows(t, db, "audit_log") != beforeAudit || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("audit log GET mutated authoritative state")
	}
}

func TestAuditLogAPIRejectsInvalidPagination(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedAuditLogAPI(t, db)
	for _, path := range []string{
		"/api/audit-log?before_id=0",
		"/api/audit-log?before_id=-1",
		"/api/audit-log?before_id=nope",
		"/api/audit-log?before_id=1&before_id=2",
		"/api/audit-log?limit=0",
		"/api/audit-log?limit=101",
		"/api/audit-log?limit=1&limit=2",
		"/api/audit-log?unknown=1",
	} {
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"AUDIT_LOG_INVALID"`) {
			t.Fatalf("invalid audit log query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func authenticatedAuditLogAPI(t *testing.T, db *sql.DB) (*Server, []*http.Cookie) {
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
	return server, []*http.Cookie{{Name: sessionCookieName, Value: token}}
}
