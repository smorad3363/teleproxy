package httpapi

import (
	"html/template"
	"net/http"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
)

type adminInventoryPageData struct {
	Username string
	Admins   []admin.InventoryEntry
}

var adminInventoryPageTemplate = template.Must(template.New("admins").Funcs(template.FuncMap{
	"adminInventoryTime": func(value time.Time) string {
		return value.UTC().Format(time.RFC3339)
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Teleproxy Administrators</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.muted,.empty{color:#9aa8c4}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:860px}th,td{text-align:left;padding:10px;border-bottom:1px solid #26324f;vertical-align:top}th{font-size:12px;color:#aebbd6}td{font-size:13px}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/admins" aria-current="page">Admins</a><a href="/users">Users</a><a href="/audit-log">Audit Log</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main><section class="panel">
<h1>Administrators</h1>
<p class="muted">Read-only authoritative Control Plane administrator inventory. Stored roles are displayed literally and are not interpreted as authorization policy.</p>
{{if .Admins}}
<div class="table-wrap"><table>
<thead><tr><th>ID</th><th>Username</th><th>Role</th><th>Enabled</th><th>Created at</th><th>Updated at</th></tr></thead>
<tbody>
{{range .Admins}}
<tr><td class="tag">{{.ID}}</td><td class="tag">{{.Username}}</td><td class="tag">{{.Role}}</td><td>{{.Enabled}}</td><td>{{adminInventoryTime .CreatedAt}}</td><td>{{adminInventoryTime .UpdatedAt}}</td></tr>
{{end}}
</tbody></table></div>
{{else}}<p class="empty">No administrators yet.</p>{{end}}
</section></main>
</body>
</html>`))

func (s *Server) handleAdminInventoryPage(w http.ResponseWriter, r *http.Request) {
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
	if !validateAdminInventoryQuery(w, r) {
		return
	}
	entries, err := admin.ListInventory(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = adminInventoryPageTemplate.Execute(w, adminInventoryPageData{
		Username: session.Admin.Username,
		Admins:   entries,
	})
}
