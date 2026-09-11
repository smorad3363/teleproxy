package httpapi

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/smorad3363/teleproxy/internal/telemt"
)

type systemPageData struct {
	Username           string
	PanelState         string
	DatabaseState      string
	ProxyState         telemt.State
	ProxyReadOnly      bool
	ProxyReadOnlyKnown bool
}

var systemPageTemplate = template.Must(template.New("system").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Teleproxy System</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px}.panel{max-width:820px;border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.muted{color:#9aa8c4}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:14px;margin-top:20px}.status{border:1px solid #26324f;border-radius:12px;padding:16px;background:#0d1425}.label{display:block;color:#9aa8c4;font-size:12px;margin-bottom:7px}.value{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;overflow-wrap:anywhere}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/system" aria-current="page">System</a><a href="/audit-log">Audit Log</a><a href="/users">Users</a><a href="/nodes">Proxy Nodes</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main><section class="panel"><h1>System</h1><p class="muted">Read-only status from established Control Plane, database readiness and global proxy health checks.</p><div class="grid"><div class="status"><span class="label">Control Plane</span><strong class="value">{{.PanelState}}</strong></div><div class="status"><span class="label">Database</span><strong class="value">{{.DatabaseState}}</strong></div><div class="status"><span class="label">Global Proxy</span><strong class="value">{{.ProxyState}}</strong></div><div class="status"><span class="label">Proxy read-only mode</span><strong class="value">{{if .ProxyReadOnlyKnown}}{{if .ProxyReadOnly}}yes{{else}}no{{end}}{{else}}not applicable{{end}}</strong></div></div></section></main>
</body>
</html>`))

func (s *Server) handleSystemPage(checker telemt.Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		proxyHealth := telemt.Health{State: telemt.StateNotConfigured}
		proxyReadOnlyKnown := checker != nil
		if checker != nil {
			proxyHealth = checker.Health(r.Context())
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = systemPageTemplate.Execute(w, systemPageData{
			Username:           session.Admin.Username,
			PanelState:         "online",
			DatabaseState:      s.systemDatabaseState(),
			ProxyState:         proxyHealth.State,
			ProxyReadOnly:      proxyHealth.ReadOnly,
			ProxyReadOnlyKnown: proxyReadOnlyKnown,
		})
	}
}

func (s *Server) systemDatabaseState() string {
	if s.db == nil {
		return "ready"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		return "unavailable"
	}
	return "ready"
}
