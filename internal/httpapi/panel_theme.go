package httpapi

import (
	"bytes"
	"net/http"
	"strings"
)

var themedPanelPaths = map[string]struct{}{
	"/":            {},
	"/system":      {},
	"/audit-log":   {},
	"/users":       {},
	"/admins":      {},
	"/sponsors":    {},
	"/nodes":       {},
	"/referrals":   {},
	"/forced-join": {},
	"/settings":    {},
	"/bot-content": {},
}

const panelThemeStyle = `<style id="tp-panel-theme">
:root{--tp-bg:#080d18;--tp-surface:#0f1728;--tp-surface-2:#141f34;--tp-border:#263650;--tp-text:#eef4ff;--tp-muted:#8ea0bd;--tp-accent:#7aa2ff;--tp-good:#6ee7b7;--tp-danger:#fda4af;--tp-sidebar:244px;color-scheme:dark}*{box-sizing:border-box}html,body{min-height:100%;background:var(--tp-bg)!important;color:var(--tp-text)!important}body{margin:0!important;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif!important}body>header{position:fixed!important;inset:0 auto 0 0!important;width:var(--tp-sidebar)!important;height:100vh!important;z-index:40!important;display:flex!important;flex-direction:column!important;align-items:stretch!important;justify-content:flex-start!important;gap:14px!important;padding:22px 16px!important;border:0!important;border-right:1px solid var(--tp-border)!important;background:linear-gradient(180deg,#101a2d 0%,#0b1220 100%)!important;overflow:auto!important}body>header nav{display:flex!important;flex-direction:column!important;align-items:stretch!important;gap:4px!important;flex:0 0 auto!important}body>header nav>strong,body>header>strong{display:none!important}.tp-brand{display:flex;align-items:center;gap:10px;padding:4px 8px 16px;color:#fff!important;text-decoration:none!important;font-size:19px;font-weight:800;letter-spacing:.2px}.tp-brand::before{content:"T";display:grid;place-items:center;width:34px;height:34px;border-radius:11px;background:linear-gradient(135deg,#8ab4ff,#6366f1);color:#07101f;font-weight:900;box-shadow:0 8px 24px #3556c833}.tp-nav-link{display:flex;align-items:center;min-height:40px;padding:9px 11px;border-radius:10px;color:#b8c6de!important;text-decoration:none!important;font-size:14px;font-weight:600}.tp-nav-link:hover{background:#17243b;color:#fff!important}.tp-nav-link[aria-current="page"]{background:#1b2b47;color:#fff!important;box-shadow:inset 3px 0 0 var(--tp-accent)}body>header>.muted,body>header>span{margin-top:auto!important;padding:14px 8px 0!important;border-top:1px solid #21314a!important;color:var(--tp-muted)!important;font-size:12px!important;overflow-wrap:anywhere}body>header>form{margin-top:auto!important}body>header>form button{width:100%!important}body>main{margin-left:var(--tp-sidebar)!important;padding:34px clamp(18px,4vw,54px) 64px!important;max-width:none!important;width:auto!important;display:block}.panel,.card{max-width:none!important;border:1px solid var(--tp-border)!important;border-radius:16px!important;background:linear-gradient(180deg,var(--tp-surface) 0%,#0d1525 100%)!important;box-shadow:0 12px 34px #0002!important}.panel+.panel,.card+.card{margin-top:18px}h1{letter-spacing:-.025em}h1,h2,h3{color:#f7faff}.muted{color:var(--tp-muted)!important}a{color:#a8c1ff}button,.primary{border-radius:10px!important;min-height:38px}input,select,textarea{border-radius:10px!important;border-color:#32445f!important;background:#0a1221!important}input:focus,select:focus,textarea:focus{outline:2px solid #6f95ee!important;outline-offset:1px}.grid{gap:14px!important}.status{border-color:#2a3a55!important;background:#0b1424!important}.tp-mobilebar{display:none}.tp-shell-note{font-size:12px;color:var(--tp-muted)}table{width:100%}.empty{border-style:dashed!important}.actions{align-items:center}.tag{display:inline-flex;align-items:center;padding:4px 8px;border:1px solid #314564;border-radius:999px;background:#101c30}.danger{color:var(--tp-danger)!important}.ok{color:var(--tp-good)!important}
@media(max-width:900px){:root{--tp-sidebar:250px}body>header{transform:translateX(-102%);transition:transform .18s ease;box-shadow:20px 0 50px #0008}body.tp-nav-open>header{transform:translateX(0)}body>main{margin-left:0!important;padding:76px 16px 48px!important}.tp-mobilebar{position:fixed;z-index:35;inset:0 0 auto 0;height:58px;display:flex;align-items:center;gap:12px;padding:0 14px;border-bottom:1px solid var(--tp-border);background:#0d1627eF;backdrop-filter:blur(10px)}.tp-mobilebar button{min-height:36px;width:40px;padding:0;font-size:20px}.tp-mobilebar strong{font-size:16px}}
</style>`

const panelThemeScript = `<script id="tp-panel-shell">
(() => {
  const routes = [
    ['/', 'Overview'], ['/users', 'Users'], ['/nodes', 'Proxies'], ['/sponsors', 'Sponsors'],
    ['/forced-join', 'Required Channels'], ['/referrals', 'Referrals'], ['/bot-content', 'Bot Content'],
    ['/settings', 'Settings'], ['/system', 'System'], ['/admins', 'Administrators'], ['/audit-log', 'Audit Log']
  ];
  const header = document.querySelector('body > header');
  if (!header) return;
  let nav = header.querySelector('nav');
  if (!nav) {
    nav = document.createElement('nav');
    header.prepend(nav);
  }
  const brand = document.createElement('a');
  brand.className = 'tp-brand';
  brand.href = '/';
  brand.textContent = 'Teleproxy';
  header.prepend(brand);
  nav.innerHTML = '';
  const path = location.pathname;
  routes.forEach(([href, label]) => {
    const link = document.createElement('a');
    link.className = 'tp-nav-link';
    link.href = href;
    link.textContent = label;
    if (path === href) link.setAttribute('aria-current', 'page');
    nav.appendChild(link);
  });
  const bar = document.createElement('div');
  bar.className = 'tp-mobilebar';
  const toggle = document.createElement('button');
  toggle.type = 'button';
  toggle.setAttribute('aria-label', 'Open navigation');
  toggle.textContent = '☰';
  const title = document.createElement('strong');
  title.textContent = 'Teleproxy';
  bar.append(toggle, title);
  document.body.prepend(bar);
  toggle.addEventListener('click', () => document.body.classList.toggle('tp-nav-open'));
  nav.addEventListener('click', () => document.body.classList.remove('tp-nav-open'));
})();
</script>`

type panelCaptureWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *panelCaptureWriter) Header() http.Header { return w.header }

func (w *panelCaptureWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *panelCaptureWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}

func panelThemeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !isThemedPanelPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		capture := &panelCaptureWriter{header: make(http.Header)}
		next.ServeHTTP(capture, r)
		status := capture.status
		if status == 0 {
			status = http.StatusOK
		}
		body := capture.body.Bytes()
		if status == http.StatusOK && strings.HasPrefix(capture.header.Get("Content-Type"), "text/html") {
			body = injectPanelTheme(body)
			capture.header.Del("Content-Length")
		}
		for key, values := range capture.header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}

func isThemedPanelPath(path string) bool {
	_, ok := themedPanelPaths[path]
	return ok
}

func injectPanelTheme(body []byte) []byte {
	source := string(body)
	if strings.Contains(source, `id="tp-panel-theme"`) {
		return body
	}
	if strings.Contains(source, "</head>") {
		source = strings.Replace(source, "</head>", panelThemeStyle+"</head>", 1)
	}
	if strings.Contains(source, "</body>") {
		source = strings.Replace(source, "</body>", panelThemeScript+"</body>", 1)
	}
	return []byte(source)
}
