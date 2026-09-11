package httpapi

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/auditlog"
)

type auditLogPageData struct {
	Username    string
	Entries     []auditlog.Entry
	NextPageURL string
}

var auditLogPageTemplate = template.Must(template.New("audit-log").Funcs(template.FuncMap{
	"auditTime": func(value time.Time) string {
		return value.UTC().Format(time.RFC3339)
	},
	"auditSnapshotPresent": func(value *string) bool {
		return value != nil
	},
	"auditSnapshotValue": func(value *string) string {
		if value == nil {
			return ""
		}
		return *value
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Teleproxy Audit Log</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.muted,.empty{color:#9aa8c4}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:1280px}th,td{text-align:left;padding:10px;border-bottom:1px solid #26324f;vertical-align:top}th{font-size:12px;color:#aebbd6}td{font-size:13px}.tag,.snapshot{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.snapshot{margin:0;max-width:420px;white-space:pre-wrap;overflow-wrap:anywhere}.pager{margin-top:16px}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/audit-log" aria-current="page">Audit Log</a><a href="/users">Users</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals">Referrals</a><a href="/forced-join">Forced Join</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main><section class="panel">
<h1>Audit Log</h1>
<p class="muted">Read-only append-only Control Plane audit entries, newest first. Snapshot text is displayed literally and is never interpreted as HTML.</p>
{{if .Entries}}
<div class="table-wrap"><table>
<thead><tr><th>Actor</th><th>Action</th><th>Target</th><th>Request ID</th><th>Timestamp</th><th>Before</th><th>After</th></tr></thead>
<tbody>
{{range .Entries}}
<tr><td class="tag">{{.Actor}}</td><td class="tag">{{.Action}}</td><td class="tag">{{.Target}}</td><td class="tag">{{.RequestID}}</td><td>{{auditTime .CreatedAt}}</td><td>{{if auditSnapshotPresent .BeforeSnapshot}}<pre class="snapshot">{{auditSnapshotValue .BeforeSnapshot}}</pre>{{else}}<span class="muted">Not present</span>{{end}}</td><td>{{if auditSnapshotPresent .AfterSnapshot}}<pre class="snapshot">{{auditSnapshotValue .AfterSnapshot}}</pre>{{else}}<span class="muted">Not present</span>{{end}}</td></tr>
{{end}}
</tbody></table></div>
{{else}}<p class="empty">No audit log entries yet.</p>{{end}}
{{if .NextPageURL}}<p class="pager"><a href="{{.NextPageURL}}">Older audit entries</a></p>{{end}}
</section></main>
</body>
</html>`))

func (s *Server) handleAuditLogPage(w http.ResponseWriter, r *http.Request) {
	session, _, ok, err := s.currentSession(r)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	if !ok {
		s.clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	beforeID, limit, ok := parseAuditLogQuery(w, r)
	if !ok {
		return
	}
	entries, err := auditlog.List(r.Context(), s.db, beforeID, limit)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	var nextBeforeID *int64
	if len(entries) == limit {
		lastID := entries[len(entries)-1].ID
		more, err := auditlog.List(r.Context(), s.db, lastID, 1)
		if err != nil {
			s.writeInternalError(w, r)
			return
		}
		if len(more) > 0 {
			nextBeforeID = &lastID
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = auditLogPageTemplate.Execute(w, auditLogPageData{
		Username:    session.Admin.Username,
		Entries:     entries,
		NextPageURL: auditLogPageNextURL(limit, nextBeforeID),
	})
}

func auditLogPageNextURL(limit int, next *int64) string {
	if next == nil {
		return ""
	}
	values := url.Values{
		"before_id": []string{strconv.FormatInt(*next, 10)},
		"limit":     []string{strconv.Itoa(limit)},
	}
	return "/audit-log?" + values.Encode()
}
