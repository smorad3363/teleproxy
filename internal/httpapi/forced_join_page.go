package httpapi

import (
	"html/template"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/forcedjoin"
)

type forcedJoinPageData struct {
	Username string
	CSRF     string
	Channels []forcedjoin.Channel
}

var forcedJoinPageTemplate = template.Must(template.New("forced-join").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Forced Join</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:24px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input,textarea{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}textarea{min-height:72px;resize:vertical}.check{display:flex;align-items:center;gap:8px}.check input{width:auto}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}button.primary{background:#dbe6ff;color:#111827;border-color:#dbe6ff;font-weight:700}button.danger{border-color:#7f1d1d;background:#35171d;color:#fecdd3}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}.channels{display:grid;gap:14px}.channel h2{margin:0 0 14px;font-size:18px}.empty{color:#9aa8c4}.topline{display:flex;justify-content:space-between;gap:16px;align-items:start}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px;color:#a5b4fc;overflow-wrap:anywhere}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}.topline{flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals">Referrals</a><a href="/forced-join" aria-current="page">Forced Join</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel">
<h1>Forced Join channels</h1>
<p class="muted">Manage channel configuration only. Membership checks and manual recheck behavior remain in the existing Telegram flow.</p>
<form id="create-forced-join">
<div class="grid">
<label>Telegram chat reference<input name="chat_ref" maxlength="128" placeholder="@channel or numeric chat ID" required></label>
<label>Display name<input name="display_name" maxlength="128" required></label>
<label>Join URL<input name="join_url" type="url" maxlength="512" placeholder="https://t.me/channel" required></label>
<label>Position<input name="position" type="number" min="0" max="1000000" step="1" value="0" required></label>
<label class="check"><input name="enabled" type="checkbox" checked> Enabled</label>
<label class="check"><input name="required" type="checkbox" checked> Required</label>
</div>
<label>Custom text<textarea name="custom_text" maxlength="1024"></textarea></label>
<div class="actions"><button class="primary" type="submit">Create channel</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section class="channels" aria-label="Existing Forced Join channels">
{{if .Channels}}
{{range .Channels}}
<article class="panel channel">
<div class="topline"><h2>{{.DisplayName}}</h2><span class="tag">{{.ChatRef}}</span></div>
<form class="edit-forced-join" data-id="{{.ID}}">
<div class="grid">
<label>Telegram chat reference<input name="chat_ref" maxlength="128" value="{{.ChatRef}}" required></label>
<label>Display name<input name="display_name" maxlength="128" value="{{.DisplayName}}" required></label>
<label>Join URL<input name="join_url" type="url" maxlength="512" value="{{.JoinURL}}" required></label>
<label>Position<input name="position" type="number" min="0" max="1000000" step="1" value="{{.Position}}" required></label>
<label class="check"><input name="enabled" type="checkbox" {{if .Enabled}}checked{{end}}> Enabled</label>
<label class="check"><input name="required" type="checkbox" {{if .Required}}checked{{end}}> Required</label>
</div>
<label>Custom text<textarea name="custom_text" maxlength="1024">{{.CustomText}}</textarea></label>
<div class="actions"><button class="primary" type="submit">Save</button><button class="danger delete-forced-join" type="button" data-id="{{.ID}}">Delete</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</article>
{{end}}
{{else}}<div class="panel empty">No Forced Join channels configured.</div>{{end}}
</section>
</main>
<script>
(() => {
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';

  function payload(form) {
    return {
      chat_ref: form.elements.chat_ref.value,
      display_name: form.elements.display_name.value,
      join_url: form.elements.join_url.value,
      enabled: form.elements.enabled.checked,
      required: form.elements.required.checked,
      position: Number(form.elements.position.value),
      custom_text: form.elements.custom_text.value
    };
  }

  async function mutate(url, method, body) {
    const response = await fetch(url, {
      method,
      credentials: 'same-origin',
      headers: {'Content-Type': 'application/json', 'X-CSRF-Token': csrf},
      body: body === undefined ? undefined : JSON.stringify(body)
    });
    if (!response.ok) {
      let message = 'Request failed.';
      try {
        const problem = await response.json();
        if (problem && problem.message) message = problem.message;
      } catch (_) {}
      throw new Error(message);
    }
  }

  function show(form, message) {
    form.querySelector('.status').textContent = message;
  }

  document.getElementById('create-forced-join').addEventListener('submit', async event => {
    event.preventDefault();
    const form = event.currentTarget;
    show(form, '');
    try {
      await mutate('/api/forced-join/channels', 'POST', payload(form));
      location.reload();
    } catch (error) {
      show(form, error.message);
    }
  });

  document.querySelectorAll('.edit-forced-join').forEach(form => {
    form.addEventListener('submit', async event => {
      event.preventDefault();
      show(form, '');
      try {
        await mutate('/api/forced-join/channels/' + encodeURIComponent(form.dataset.id), 'PUT', payload(form));
        location.reload();
      } catch (error) {
        show(form, error.message);
      }
    });
  });

  document.querySelectorAll('.delete-forced-join').forEach(button => {
    button.addEventListener('click', async () => {
      const form = button.closest('form');
      show(form, '');
      if (!confirm('Delete this Forced Join channel?')) return;
      try {
        await mutate('/api/forced-join/channels/' + encodeURIComponent(button.dataset.id), 'DELETE');
        location.reload();
      } catch (error) {
        show(form, error.message);
      }
    });
  });
})();
</script>
</body>
</html>`))

func (s *Server) handleForcedJoinPage(w http.ResponseWriter, r *http.Request) {
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
	channels, err := forcedjoin.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := forcedJoinPageTemplate.Execute(w, forcedJoinPageData{
		Username: session.Admin.Username,
		CSRF:     sessionCSRF(token),
		Channels: channels,
	}); err != nil {
		return
	}
}
