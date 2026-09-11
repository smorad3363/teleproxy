package httpapi

import (
	"html/template"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/botcontent"
)

type botContentPageData struct {
	Username string
	CSRF     string
	Slots    []botContentPageSlot
}

type botContentPageSlot struct {
	Slot       botcontent.Slot
	Label      string
	Text       string
	Configured bool
}

var botContentPageTemplate = template.Must(template.New("bot-content").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Bot Content</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:18px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:20px}.slot-head{display:flex;justify-content:space-between;gap:12px;align-items:center;flex-wrap:wrap}h1,h2{margin-top:0}.muted{color:#9aa8c4}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#b8c7e8}textarea{width:100%;min-height:130px;resize:vertical;border:1px solid #34415f;border-radius:9px;padding:11px;background:#0c1323;color:#fff;font:inherit;line-height:1.45}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:12px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}button.primary{background:#dbe6ff;color:#111827;border-color:#dbe6ff;font-weight:700}button:disabled{opacity:.45;cursor:not-allowed}.status{min-height:1.3em;color:#fda4af;margin:10px 0 0}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/users">Users</a><a href="/settings">Settings</a><a href="/bot-content" aria-current="page">Bot Content</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel"><h1>Bot Content</h1><p class="muted">Configure literal text overrides only. Empty slots have no configured override; runtime Bot wiring, formatting, buttons and emoji are intentionally separate.</p></section>
{{range .Slots}}
<section class="panel" data-bot-slot="{{.Slot}}">
<div class="slot-head"><div><h2>{{.Label}}</h2><div class="tag">{{.Slot}}</div></div>{{if .Configured}}<span>Configured</span>{{else}}<span class="muted">Not configured</span>{{end}}</div>
<textarea aria-label="{{.Label}} text">{{.Text}}</textarea>
<div class="actions"><button class="primary save" type="button">Save override</button><button class="clear" type="button" {{if not .Configured}}disabled{{end}}>Clear override</button></div>
<p class="status" role="status" aria-live="polite"></p>
</section>
{{end}}
</main>
<script>
(() => {
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';
  async function problemMessage(response) {
    try {
      const problem = await response.json();
      if (problem && problem.message) return problem.message;
    } catch (_) {}
    return 'Request failed.';
  }
  document.querySelectorAll('[data-bot-slot]').forEach(panel => {
    const slot = panel.dataset.botSlot;
    const textarea = panel.querySelector('textarea');
    const status = panel.querySelector('.status');
    panel.querySelector('.save').addEventListener('click', async () => {
      status.textContent = '';
      try {
        const response = await fetch('/api/bot-content/' + encodeURIComponent(slot), {
          method: 'PUT', credentials: 'same-origin',
          headers: {'Content-Type': 'application/json', 'X-CSRF-Token': csrf},
          body: JSON.stringify({text: textarea.value})
        });
        if (!response.ok) { status.textContent = await problemMessage(response); return; }
        location.reload();
      } catch (_) { status.textContent = 'Request failed.'; }
    });
    panel.querySelector('.clear').addEventListener('click', async () => {
      status.textContent = '';
      try {
        const response = await fetch('/api/bot-content/' + encodeURIComponent(slot), {
          method: 'DELETE', credentials: 'same-origin',
          headers: {'X-CSRF-Token': csrf}
        });
        if (!response.ok) { status.textContent = await problemMessage(response); return; }
        location.reload();
      } catch (_) { status.textContent = 'Request failed.'; }
    });
  });
})();
</script>
</body>
</html>`))

func (s *Server) handleBotContentPage(w http.ResponseWriter, r *http.Request) {
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
	entries, err := botcontent.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := botContentPageTemplate.Execute(w, botContentPageData{
		Username: session.Admin.Username,
		CSRF:     sessionCSRF(token),
		Slots:    botContentPageSlots(entries),
	}); err != nil {
		return
	}
}

func botContentPageSlots(entries []botcontent.Entry) []botContentPageSlot {
	configured := make(map[botcontent.Slot]botcontent.Entry, len(entries))
	for _, entry := range entries {
		configured[entry.Slot] = entry
	}
	slots := []botContentPageSlot{
		{Slot: botcontent.SlotWelcome, Label: "Welcome"},
		{Slot: botcontent.SlotForcedJoin, Label: "Forced Join"},
		{Slot: botcontent.SlotReferral, Label: "Referral"},
		{Slot: botcontent.SlotProxy, Label: "Proxy"},
		{Slot: botcontent.SlotExpired, Label: "Expired"},
		{Slot: botcontent.SlotNoCredit, Label: "No Credit"},
		{Slot: botcontent.SlotSupport, Label: "Support"},
	}
	for index := range slots {
		if entry, ok := configured[slots[index].Slot]; ok {
			slots[index].Configured = true
			slots[index].Text = entry.Text
		}
	}
	return slots
}
