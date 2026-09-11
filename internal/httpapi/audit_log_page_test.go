package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/auditlog"
)

func TestAuditLogPageRequiresAuthentication(t *testing.T) {
	db := testDB(t)
	server := NewWithProxyServicesAndForcedJoin(db, Options{}, nil, nil)

	response := perform(server.Handler(), http.MethodGet, "/audit-log", nil, nil)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated audit log page = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "Audit Log") {
		t.Fatalf("unauthenticated audit log page rendered audit markup: %s", response.Body.String())
	}
}

func TestAuditLogPageRendersNewestFirstEscapedSnapshotsAndPaginationWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedAuditLogAPI(t, db)
	base := time.Unix(1_850_000_000, 0).UTC()

	first, err := auditlog.Append(ctx, db, auditlog.AppendInput{
		Actor: "admin:owner", Action: "settings.update", Target: "settings:start_gift", RequestID: "req-a",
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	before := `<b>before&</b>`
	after := `<script>after&</script>`
	second, err := auditlog.Append(ctx, db, auditlog.AppendInput{
		Actor: "admin:<owner>", Action: "node.update&check", Target: "node:<2>", BeforeSnapshot: &before, AfterSnapshot: &after, RequestID: "req-<b>",
	}, base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	beforeAudit := countHTTPRows(t, db, "audit_log")
	beforeSettings := countHTTPRows(t, db, "settings")

	response := perform(server.Handler(), http.MethodGet, "/audit-log?limit=1", nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("audit log page = %d %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		"admin:&lt;owner&gt;",
		"node.update&amp;check",
		"node:&lt;2&gt;",
		"req-&lt;b&gt;",
		base.Add(time.Second).Format(time.RFC3339),
		"&lt;b&gt;before&amp;&lt;/b&gt;",
		"&lt;script&gt;after&amp;&lt;/script&gt;",
		`href="/audit-log?before_id=` + strconv.FormatInt(second.ID, 10) + `&amp;limit=1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("audit log page missing %q: %s", want, body)
		}
	}
	for _, leaked := range []string{"admin:<owner>", "node.update&check", "node:<2>", "req-<b>", before, after} {
		if strings.Contains(body, leaked) {
			t.Fatalf("audit log page leaked unescaped literal %q: %s", leaked, body)
		}
	}
	if strings.Contains(body, "req-a") {
		t.Fatalf("audit log page included older row on first page: %s", body)
	}
	if countHTTPRows(t, db, "audit_log") != beforeAudit || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("audit log page rendering mutated authoritative state")
	}

	older := perform(server.Handler(), http.MethodGet, "/audit-log?before_id="+strconv.FormatInt(second.ID, 10)+"&limit=1", nil, cookie)
	if older.Code != http.StatusOK || !strings.Contains(older.Body.String(), "req-a") || strings.Contains(older.Body.String(), "req-&lt;b&gt;") {
		t.Fatalf("older audit log page pagination mismatch: %d %s", older.Code, older.Body.String())
	}
	if got := strings.Count(older.Body.String(), "Not present"); got != 2 {
		t.Fatalf("older audit log page absent snapshots = %d, body=%s", got, older.Body.String())
	}
	if strings.Contains(older.Body.String(), `href="/audit-log?`) {
		t.Fatalf("last audit log page unexpectedly has pagination: %s", older.Body.String())
	}
	if first.ID >= second.ID {
		t.Fatalf("audit log fixture IDs are not ordered: first=%d second=%d", first.ID, second.ID)
	}
}

func TestAuditLogPageRejectsInvalidPaginationWithoutMutation(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedAuditLogAPI(t, db)
	beforeAudit := countHTTPRows(t, db, "audit_log")
	beforeSettings := countHTTPRows(t, db, "settings")

	for _, path := range []string{
		"/audit-log?before_id=0",
		"/audit-log?before_id=-1",
		"/audit-log?before_id=nope",
		"/audit-log?before_id=1&before_id=2",
		"/audit-log?limit=0",
		"/audit-log?limit=101",
		"/audit-log?limit=1&limit=2",
		"/audit-log?unknown=1",
	} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"AUDIT_LOG_INVALID"`) {
			t.Fatalf("invalid audit log page query %q = %d %s", path, response.Code, response.Body.String())
		}
	}
	if countHTTPRows(t, db, "audit_log") != beforeAudit || countHTTPRows(t, db, "settings") != beforeSettings {
		t.Fatal("invalid audit log page query mutated authoritative state")
	}
}

func TestAuditLogPageEmptyStateAndDashboardLink(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedAuditLogAPI(t, db)

	auditPage := perform(server.Handler(), http.MethodGet, "/audit-log", nil, cookie)
	if auditPage.Code != http.StatusOK || !strings.Contains(auditPage.Body.String(), "No audit log entries yet.") {
		t.Fatalf("empty audit log page = %d %s", auditPage.Code, auditPage.Body.String())
	}
	dashboard := perform(server.Handler(), http.MethodGet, "/", nil, cookie)
	if dashboard.Code != http.StatusOK || !strings.Contains(dashboard.Body.String(), `href="/audit-log"`) {
		t.Fatalf("dashboard audit log link = %d %s", dashboard.Code, dashboard.Body.String())
	}
}
