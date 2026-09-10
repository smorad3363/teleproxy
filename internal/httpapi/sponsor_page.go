package httpapi

import (
	"html/template"
	"net/http"
	"time"

	"github.com/smorad3363/teleproxy/internal/sponsor"
)

type sponsorPageData struct {
	Username string
	CSRF     string
	Profiles []sponsor.Profile
}

var sponsorPageTemplate = template.Must(template.New("sponsors").Funcs(template.FuncMap{
	"sponsorTime": sponsorPageTime,
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Sponsors</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:24px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input,textarea{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}textarea{min-height:72px;resize:vertical}.check{display:flex;align-items:center;gap:8px}.check input{width:auto}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}button.primary{background:#dbe6ff;color:#111827;border-color:#dbe6ff;font-weight:700}button.danger{border-color:#7f1d1d;background:#35171d;color:#fecdd3}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}.profiles{display:grid;gap:14px}.profile h2{margin:0 0 14px;font-size:18px}.empty{color:#9aa8c4}.topline{display:flex;justify-content:space-between;gap:16px;align-items:start}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px;color:#a5b4fc;overflow-wrap:anywhere}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}.topline{flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/sponsors" aria-current="page">Sponsors</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel">
<h1>Sponsor Profiles</h1>
<p class="muted">Manage Sponsor metadata. Assignment and Telemt projection are intentionally separate.</p>
<form id="create-sponsor">
<div class="grid">
<label>Name<input name="name" maxlength="128" required></label>
<label>Channel username/link<input name="channel_ref" maxlength="512" placeholder="@channel or https://t.me/channel" required></label>
<label>AdTag (32 hex characters)<input name="ad_tag" maxlength="32" minlength="32" pattern="[0-9A-Fa-f]{32}" required></label>
<label>Weight<input name="weight" type="number" min="1" step="1" value="1" required></label>
<label>Start time (optional RFC3339)<input name="starts_at" placeholder="2030-01-01T00:00:00Z"></label>
<label>End time (optional RFC3339)<input name="ends_at" placeholder="2030-01-02T00:00:00Z"></label>
<label class="check"><input name="enabled" type="checkbox" checked> Enabled</label>
</div>
<label>Notes<textarea name="notes" maxlength="4096"></textarea></label>
<div class="actions"><button class="primary" type="submit">Create Sponsor</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section class="profiles" aria-label="Existing Sponsor Profiles">
{{if .Profiles}}
{{range .Profiles}}
<article class="panel profile">
<div class="topline"><h2>{{.Name}}</h2><span class="tag">{{.AdTag}}</span></div>
<form class="edit-sponsor" data-id="{{.ID}}">
<div class="grid">
<label>Name<input name="name" maxlength="128" value="{{.Name}}" required></label>
<label>Channel username/link<input name="channel_ref" maxlength="512" value="{{.ChannelRef}}" required></label>
<label>AdTag (32 hex characters)<input name="ad_tag" maxlength="32" minlength="32" pattern="[0-9A-Fa-f]{32}" value="{{.AdTag}}" required></label>
<label>Weight<input name="weight" type="number" min="1" step="1" value="{{.Weight}}" required></label>
<label>Start time (optional RFC3339)<input name="starts_at" value="{{sponsorTime .StartsAt}}" placeholder="2030-01-01T00:00:00Z"></label>
<label>End time (optional RFC3339)<input name="ends_at" value="{{sponsorTime .EndsAt}}" placeholder="2030-01-02T00:00:00Z"></label>
<label class="check"><input name="enabled" type="checkbox" {{if .Enabled}}checked{{end}}> Enabled</label>
</div>
<label>Notes<textarea name="notes" maxlength="4096">{{.Notes}}</textarea></label>
<div class="actions"><button class="primary" type="submit">Save</button><button class="danger delete-sponsor" type="button" data-id="{{.ID}}">Delete</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</article>
{{end}}
{{else}}<div class="panel empty">No Sponsor Profiles configured.</div>{{end}}
</section>
</main>
<script>
(() => {
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';

  function optionalTime(value) {
    const trimmed = value.trim();
    return trimmed === '' ? null : trimmed;
  }

  function payload(form) {
    return {
      name: form.elements.name.value,
      channel_ref: form.elements.channel_ref.value,
      ad_tag: form.elements.ad_tag.value,
      enabled: form.elements.enabled.checked,
      weight: Number(form.elements.weight.value),
      starts_at: optionalTime(form.elements.starts_at.value),
      ends_at: optionalTime(form.elements.ends_at.value),
      notes: form.elements.notes.value
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

  document.getElementById('create-sponsor').addEventListener('submit', async event => {
    event.preventDefault();
    const form = event.currentTarget;
    show(form, '');
    try {
      await mutate('/api/sponsors', 'POST', payload(form));
      location.reload();
    } catch (error) {
      show(form, error.message);
    }
  });

  document.querySelectorAll('.edit-sponsor').forEach(form => {
    form.addEventListener('submit', async event => {
      event.preventDefault();
      show(form, '');
      try {
        await mutate('/api/sponsors/' + encodeURIComponent(form.dataset.id), 'PUT', payload(form));
        location.reload();
      } catch (error) {
        show(form, error.message);
      }
    });
  });

  document.querySelectorAll('.delete-sponsor').forEach(button => {
    button.addEventListener('click', async () => {
      const form = button.closest('form');
      show(form, '');
      if (!confirm('Delete this Sponsor Profile?')) return;
      try {
        await mutate('/api/sponsors/' + encodeURIComponent(button.dataset.id), 'DELETE');
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

func sponsorPageTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *Server) handleSponsorPage(w http.ResponseWriter, r *http.Request) {
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
	profiles, err := sponsor.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sponsorPageTemplate.Execute(w, sponsorPageData{
		Username: session.Admin.Username,
		CSRF:     sessionCSRF(token),
		Profiles: profiles,
	}); err != nil {
		return
	}
}
