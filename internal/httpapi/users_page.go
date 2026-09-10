package httpapi

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/useradmin"
)

type userPageData struct {
	Username    string
	Users       []useradmin.Entry
	NextPageURL string
}

var userPageTemplate = template.Must(template.New("users").Funcs(template.FuncMap{
	"userTime": func(value time.Time) string {
		return value.UTC().Format(time.RFC3339)
	},
	"userOptionalTime": func(value *time.Time) string {
		if value == nil {
			return "—"
		}
		return value.UTC().Format(time.RFC3339)
	},
	"userErrorCode": func(value string) string {
		if value == "" {
			return "—"
		}
		return value
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Teleproxy Users</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.muted,.empty{color:#9aa8c4}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:1180px}th,td{text-align:left;padding:10px;border-bottom:1px solid #26324f;vertical-align:top}th{font-size:12px;color:#aebbd6}td{font-size:13px}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.pager{margin-top:16px}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/users" aria-current="page">Users</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals">Referrals</a><a href="/forced-join">Forced Join</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main><section class="panel">
<h1>Users</h1>
<p class="muted">Read-only authoritative Control Plane inventory. Telegram username, traffic, Node/Sponsor assignment, secret status and general last activity are not inferred.</p>
{{if .Users}}
<div class="table-wrap"><table>
<thead><tr><th>Telegram ID</th><th>Proxy username</th><th>Enabled</th><th>Sync state</th><th>Last error</th><th>Available bytes</th><th>Nearest expiry</th><th>Referrals</th><th>Created at</th><th>Updated at</th></tr></thead>
<tbody>
{{range .Users}}
<tr><td class="tag">{{.TelegramID}}</td><td class="tag">{{.ProxyUsername}}</td><td>{{.DesiredEnabled}}</td><td>{{.SyncState}}</td><td class="tag">{{userErrorCode .LastErrorCode}}</td><td class="tag">{{.AvailableBytes}}</td><td>{{userOptionalTime .NearestExpiry}}</td><td class="tag">{{.ReferralCount}}</td><td>{{userTime .CreatedAt}}</td><td>{{userTime .UpdatedAt}}</td></tr>
{{end}}
</tbody></table></div>
{{else}}<p class="empty">No Telegram users yet.</p>{{end}}
{{if .NextPageURL}}<p class="pager"><a href="{{.NextPageURL}}">Older users</a></p>{{end}}
</section></main>
</body>
</html>`))

func (s *Server) handleUserAdminPage(w http.ResponseWriter, r *http.Request) {
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
	query, ok := parseUserAdminQuery(w, r)
	if !ok {
		return
	}
	page, err := useradmin.List(r.Context(), s.db, query, time.Now().UTC())
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = userPageTemplate.Execute(w, userPageData{
		Username:    session.Admin.Username,
		Users:       page.Items,
		NextPageURL: userPageNextURL(query, page.NextBeforeID),
	})
}

func userPageNextURL(query useradmin.ListQuery, next *int64) string {
	if next == nil {
		return ""
	}
	values := url.Values{"before_id": []string{strconv.FormatInt(*next, 10)}}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return "/users?" + values.Encode()
}
