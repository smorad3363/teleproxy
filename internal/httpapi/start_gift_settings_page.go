package httpapi

import (
	"html/template"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/settings"
)

type startGiftPageData struct {
	Username string
	CSRF     string
	Bytes    int64
}

var startGiftPageTemplate = template.Must(template.New("settings").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Settings</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px}.panel{max-width:760px;border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#dbe6ff;color:#111827;font-weight:700;cursor:pointer}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/users">Users</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals">Referrals</a><a href="/forced-join">Forced Join</a><a href="/settings" aria-current="page">Settings</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main><section class="panel">
<h1>Settings</h1>
<h2>Start gift</h2>
<p class="muted">This amount applies only when a user's exactly-once start gift has not yet been created. Existing Credit Buckets are not rewritten.</p>
<form id="start-gift-settings">
<label>Start gift bytes<input name="bytes" type="text" inputmode="numeric" pattern="[1-9][0-9]*" value="{{.Bytes}}" required></label>
<div class="actions"><button type="submit">Save start gift</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section></main>
<script>
(() => {
  const form = document.getElementById('start-gift-settings');
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';
  const status = form.querySelector('.status');

  form.addEventListener('submit', async event => {
    event.preventDefault();
    status.textContent = '';
    const bytes = form.elements.bytes.value.trim();
    if (!/^[1-9][0-9]*$/.test(bytes)) {
      status.textContent = 'Start gift bytes must be a positive whole number.';
      return;
    }
    try {
      const response = await fetch('/api/settings/start-gift', {
        method: 'PUT',
        credentials: 'same-origin',
        headers: {'Content-Type': 'application/json', 'X-CSRF-Token': csrf},
        body: '{"bytes":' + bytes + '}'
      });
      if (!response.ok) {
        let message = 'Request failed.';
        try {
          const problem = await response.json();
          if (problem && problem.message) message = problem.message;
        } catch (_) {}
        status.textContent = message;
        return;
      }
      location.reload();
    } catch (_) {
      status.textContent = 'Request failed.';
    }
  });
})();
</script>
</body>
</html>`))

func (s *Server) handleStartGiftSettingsPage(w http.ResponseWriter, r *http.Request) {
	session, token, ok, err := s.currentSession(r)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	if !ok {
		s.clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	value, err := settings.StartGiftBytes(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = startGiftPageTemplate.Execute(w, startGiftPageData{
		Username: session.Admin.Username,
		CSRF:     sessionCSRF(token),
		Bytes:    value,
	})
}
