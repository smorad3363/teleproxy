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
<title>Teleproxy Proxy Nodes</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:24px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input,select{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}.check{display:flex;align-items:center;gap:8px}.check input{width:auto}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}button.primary{background:#dbe6ff;color:#111827;border-color:#dbe6ff;font-weight:700}button.danger{border-color:#7f1d1d;background:#35171d;color:#fecdd3}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}.nodes{display:grid;gap:14px}.node h2{margin:0 0 14px;font-size:18px}.empty{color:#9aa8c4}.topline{display:flex;justify-content:space-between;gap:16px;align-items:start}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px;color:#a5b4fc;overflow-wrap:anywhere}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}.topline{flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/sponsors">Sponsors</a><a href="/nodes" aria-current="page">Proxy Nodes</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel">
<h1>Proxy Nodes</h1>
<p class="muted">Manage static Proxy/Relay Node metadata. Runtime health, lifecycle actions, routing and Sponsor assignment are intentionally separate.</p>
<form id="create-node">
<div class="grid">
<label>Node type<select name="node_type" required><option value="proxy">Proxy</option><option value="relay">Relay</option></select></label>
<label>Name<input name="name" maxlength="128" required></label>
<label>Region<input name="region" maxlength="128" required></label>
<label>Host / IP<input name="host" maxlength="255" required></label>
<label>Public host / IP<input name="public_host" maxlength="255" required></label>
<label>MTProto port<input name="mtproto_port" type="number" min="1" max="65535" step="1" required></label>
<label>Internal API endpoint<input name="internal_api_endpoint" maxlength="1024" placeholder="http://telemt:9091" required></label>
<label class="check"><input name="enabled" type="checkbox" checked> Enabled metadata state</label>
</div>
<div class="actions"><button class="primary" type="submit">Create Node</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section class="nodes" aria-label="Existing Proxy Nodes">
{{if .Nodes}}
{{range .Nodes}}
<article class="panel node">
<div class="topline"><h2>{{.Name}}</h2><span class="tag">{{.Type}} · {{.Region}}</span></div>
<form class="edit-node" data-id="{{.ID}}">
<div class="grid">
<label>Node type<select name="node_type" required><option value="proxy" {{if nodeTypeIs .Type "proxy"}}selected{{end}}>Proxy</option><option value="relay" {{if nodeTypeIs .Type "relay"}}selected{{end}}>Relay</option></select></label>
<label>Name<input name="name" maxlength="128" value="{{.Name}}" required></label>
<label>Region<input name="region" maxlength="128" value="{{.Region}}" required></label>
<label>Host / IP<input name="host" maxlength="255" value="{{.Host}}" required></label>
<label>Public host / IP<input name="public_host" maxlength="255" value="{{.PublicHost}}" required></label>
<label>MTProto port<input name="mtproto_port" type="number" min="1" max="65535" step="1" value="{{.MTProtoPort}}" required></label>
<label>Internal API endpoint<input name="internal_api_endpoint" maxlength="1024" value="{{.InternalAPIEndpoint}}" required></label>
<label class="check"><input name="enabled" type="checkbox" {{if .Enabled}}checked{{end}}> Enabled metadata state</label>
</div>
<div class="actions"><button class="primary" type="submit">Save</button><button class="danger delete-node" type="button" data-id="{{.ID}}">Delete</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</article>
{{end}}
{{else}}<div class="panel empty">No Proxy Nodes configured.</div>{{end}}
</section>
</main>
<script>
(() => {
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';

  function payload(form) {
    return {
      node_type: form.elements.node_type.value,
      name: form.elements.name.value,
      region: form.elements.region.value,
      host: form.elements.host.value,
      public_host: form.elements.public_host.value,
      mtproto_port: Number(form.elements.mtproto_port.value),
      internal_api_endpoint: form.elements.internal_api_endpoint.value,
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
  }

  function show(form, message) {
    form.querySelector('.status').textContent = message;
  }

  document.getElementById('create-node').addEventListener('submit', async event => {
    event.preventDefault();
    const form = event.currentTarget;
    show(form, '');
    try {
      await mutate('/api/nodes', 'POST', payload(form));
      location.reload();
    } catch (error) {
      show(form, error.message);
    }
  });

  document.querySelectorAll('.edit-node').forEach(form => {
    form.addEventListener('submit', async event => {
      event.preventDefault();
      show(form, '');
      try {
        await mutate('/api/nodes/' + encodeURIComponent(form.dataset.id), 'PUT', payload(form));
        location.reload();
      } catch (error) {
        show(form, error.message);
      }
    });
  });

  document.querySelectorAll('.delete-node').forEach(button => {
    button.addEventListener('click', async () => {
      const form = button.closest('form');
      show(form, '');
      if (!confirm('Delete this Proxy Node metadata record?')) return;
      try {
        await mutate('/api/nodes/' + encodeURIComponent(button.dataset.id), 'DELETE');
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
