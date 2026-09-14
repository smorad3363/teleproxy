package httpapi

import (
	"html/template"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/proxynode"
)

type proxyNodePageData struct {
	Username string
	CSRF     string
	Nodes    []proxynode.Node
}

var proxyNodePageTemplate = template.Must(template.New("proxy-nodes").Funcs(template.FuncMap{
	"nodeTypeIs": func(value proxynode.Type, want string) bool { return string(value) == want },
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Proxies</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:24px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.hero{display:flex;justify-content:space-between;gap:20px;align-items:end}.hero h1{margin:0 0 7px}.hero p{margin:0}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input,select{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}.check{display:flex;align-items:center;gap:8px}.check input{width:auto}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}button.primary{background:#dbe6ff;color:#111827;border-color:#dbe6ff;font-weight:700}button.danger{border-color:#7f1d1d;background:#35171d;color:#fecdd3}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}.ok{color:#86efac}.nodes{display:grid;gap:14px}.node h2{margin:0;font-size:18px}.empty{color:#9aa8c4}.topline{display:flex;justify-content:space-between;gap:16px;align-items:start}.node-meta{display:flex;gap:8px;flex-wrap:wrap}.tag,.badge{display:inline-flex;align-items:center;border:1px solid #354765;border-radius:999px;padding:4px 8px;font-size:12px}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;color:#a5b4fc}.badge.enabled{color:#86efac}.badge.disabled{color:#fda4af}details{margin-top:14px;border:1px solid #26324f;border-radius:12px;background:#0d1425;padding:12px 14px}summary{cursor:pointer;font-weight:700;color:#d9e5fb}details .grid{margin-top:14px}.callout{margin-top:12px;padding:12px 14px;border:1px solid #2d4264;border-radius:12px;background:#0b1729;color:#aebbd6;font-size:13px;line-height:1.5}.quick-title{margin:0 0 16px}.existing-title{margin:0 0 12px}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}.topline,.hero{flex-direction:column;align-items:flex-start}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/sponsors">Sponsors</a><a href="/nodes" aria-current="page">Proxies</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel hero"><div><h1>Proxies</h1><p class="muted">Fast setup for Proxy/Relay metadata. Runtime routing and per-node health stay separate until their contracts are defined.</p></div><a href="#quick-create">Quick Add</a></section>
<section class="panel" id="quick-create">
<h2 class="quick-title">Create Proxy</h2>
<form id="create-node">
<div class="grid">
<label>Name<input name="name" maxlength="128" placeholder="Germany 1" required></label>
<label>Region<input name="region" maxlength="128" placeholder="DE / Frankfurt" required></label>
<label>Public address<input name="public_host" maxlength="255" placeholder="proxy.example.com or 203.0.113.10" required></label>
<label>MTProto port<input name="mtproto_port" type="number" min="1" max="65535" step="1" value="443" required></label>
</div>
<label class="check"><input name="enabled" type="checkbox" checked> Enabled metadata state</label>
<details>
<summary>Advanced</summary>
<div class="grid">
<label>Node type<select name="node_type" required><option value="proxy">Proxy</option><option value="relay">Relay</option></select></label>
<label>Internal host override<input name="host_override" maxlength="255" placeholder="Leave blank to use Public address"></label>
<label>Internal API endpoint <span class="muted">(optional metadata)</span><input name="internal_api_endpoint" maxlength="1024" placeholder="http://telemt:9091"></label>
</div>
</details>
<details>
<summary>Add sponsor / required channel (optional)</summary>
<div class="callout">If filled, this creates or reuses a global Required Channel for Bot users. It is not a node-specific Sponsor routing rule.</div>
<div class="grid">
<label>Telegram chat ref<input name="sponsor_chat_ref" maxlength="128" placeholder="@channel"></label>
<label>Display name<input name="sponsor_display_name" maxlength="128" placeholder="Our Sponsor"></label>
<label>Join URL<input name="sponsor_join_url" maxlength="512" placeholder="https://t.me/channel"></label>
</div>
</details>
<div class="actions"><button class="primary" type="submit">Create Proxy</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section aria-label="Existing Proxies">
<h2 class="existing-title">Existing Proxies</h2>
<div class="nodes">
{{if .Nodes}}
{{range .Nodes}}
<article class="panel node">
<div class="topline"><div><h2>{{.Name}}</h2><div class="muted">{{.PublicHost}}:{{.MTProtoPort}}</div></div><div class="node-meta"><span class="tag">{{.Type}} · {{.Region}}</span>{{if .Enabled}}<span class="badge enabled">Enabled</span>{{else}}<span class="badge disabled">Disabled</span>{{end}}</div></div>
<form class="edit-node" data-id="{{.ID}}">
<div class="grid">
<label>Name<input name="name" maxlength="128" value="{{.Name}}" required></label>
<label>Region<input name="region" maxlength="128" value="{{.Region}}" required></label>
<label>Public address<input name="public_host" maxlength="255" value="{{.PublicHost}}" required></label>
<label>MTProto port<input name="mtproto_port" type="number" min="1" max="65535" step="1" value="{{.MTProtoPort}}" required></label>
</div>
<label class="check"><input name="enabled" type="checkbox" {{if .Enabled}}checked{{end}}> Enabled metadata state</label>
<details>
<summary>Advanced</summary>
<div class="grid">
<label>Node type<select name="node_type" required><option value="proxy" {{if nodeTypeIs .Type "proxy"}}selected{{end}}>Proxy</option><option value="relay" {{if nodeTypeIs .Type "relay"}}selected{{end}}>Relay</option></select></label>
<label>Internal host override<input name="host_override" maxlength="255" value="{{.Host}}"></label>
<label>Internal API endpoint <span class="muted">(optional metadata)</span><input name="internal_api_endpoint" maxlength="1024" value="{{.InternalAPIEndpoint}}"></label>
</div>
</details>
<div class="actions"><button class="primary" type="submit">Save</button><button class="danger delete-node" type="button" data-id="{{.ID}}">Delete</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</article>
{{end}}
{{else}}<div class="panel empty">No proxies added yet. Use Quick Add above.</div>{{end}}
</div>
</section>
</main>
<script>
(() => {
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';

  function payload(form) {
    const publicHost = form.elements.public_host.value.trim();
    const hostOverride = form.elements.host_override ? form.elements.host_override.value.trim() : '';
    return {
      node_type: form.elements.node_type.value,
      name: form.elements.name.value,
      region: form.elements.region.value,
      host: hostOverride || publicHost,
      public_host: publicHost,
      mtproto_port: Number(form.elements.mtproto_port.value),
      internal_api_endpoint: form.elements.internal_api_endpoint ? form.elements.internal_api_endpoint.value.trim() : '',
      enabled: form.elements.enabled.checked
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
    return response;
  }

  function show(form, message, ok) {
    const node = form.querySelector('.status');
    node.textContent = message;
    node.classList.toggle('ok', Boolean(ok));
  }

  function sponsorInput(form) {
    const chatRef = form.elements.sponsor_chat_ref.value.trim();
    const displayName = form.elements.sponsor_display_name.value.trim();
    const joinURL = form.elements.sponsor_join_url.value.trim();
    if (!chatRef && !displayName && !joinURL) return null;
    if (!chatRef || !displayName || !joinURL) throw new Error('Complete all Sponsor channel fields or leave all of them empty.');
    return {chat_ref: chatRef, display_name: displayName, join_url: joinURL};
  }

  async function ensureRequiredChannel(input) {
    if (!input) return false;
    const response = await fetch('/api/forced-join/channels', {credentials: 'same-origin', headers: {'Accept': 'application/json'}});
    if (!response.ok) throw new Error('Could not read Required Channels.');
    const body = await response.json();
    const channels = Array.isArray(body.channels) ? body.channels : [];
    if (channels.some(channel => channel.chat_ref === input.chat_ref)) return false;
    const maxPosition = channels.reduce((max, channel) => Math.max(max, Number(channel.position) || 0), -1);
    await mutate('/api/forced-join/channels', 'POST', {
      chat_ref: input.chat_ref,
      display_name: input.display_name,
      join_url: input.join_url,
      enabled: true,
      required: true,
      position: Math.min(maxPosition + 1, 1000000),
      custom_text: ''
    });
    return true;
  }

  document.getElementById('create-node').addEventListener('submit', async event => {
    event.preventDefault();
    const form = event.currentTarget;
    show(form, '', false);
    let channelChanged = false;
    try {
      const sponsor = sponsorInput(form);
      channelChanged = await ensureRequiredChannel(sponsor);
      await mutate('/api/nodes', 'POST', payload(form));
      show(form, channelChanged ? 'Proxy and Required Channel saved.' : 'Proxy saved.', true);
      location.reload();
    } catch (error) {
      show(form, channelChanged ? 'Required Channel was saved, but Proxy creation failed: ' + error.message : error.message, false);
    }
  });

  document.querySelectorAll('.edit-node').forEach(form => {
    form.addEventListener('submit', async event => {
      event.preventDefault();
      show(form, '', false);
      try {
        await mutate('/api/nodes/' + encodeURIComponent(form.dataset.id), 'PUT', payload(form));
        show(form, 'Saved.', true);
        location.reload();
      } catch (error) {
        show(form, error.message, false);
      }
    });
  });

  document.querySelectorAll('.delete-node').forEach(button => {
    button.addEventListener('click', async () => {
      const form = button.closest('form');
      show(form, '', false);
      if (!confirm('Delete this Proxy metadata record?')) return;
      try {
        await mutate('/api/nodes/' + encodeURIComponent(button.dataset.id), 'DELETE');
        location.reload();
      } catch (error) {
        show(form, error.message, false);
      }
    });
  });
})();
</script>
</body>
</html>`))

func (s *Server) handleProxyNodePage(w http.ResponseWriter, r *http.Request) {
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
	nodes, err := proxynode.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := proxyNodePageTemplate.Execute(w, proxyNodePageData{
		Username: session.Admin.Username,
		CSRF:     sessionCSRF(token),
		Nodes:    nodes,
	}); err != nil {
		return
	}
}
