package httpapi

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/useradmin"
)

type userPageData struct {
	Username            string
	CSRF                string
	Users               []useradmin.Entry
	TelegramIDFilter    string
	ProxyUsernameFilter string
	Limit               int
	HasFilters          bool
	NextPageURL         string
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
	"userProvisioningPhase": func(value *proxyprovision.Phase) string {
		if value == nil {
			return "not provisioned"
		}
		return string(*value)
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Teleproxy Users</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.muted,.empty{color:#9aa8c4}.filters{display:flex;gap:12px;align-items:end;flex-wrap:wrap;margin:18px 0}.filters label{display:grid;gap:6px;font-size:12px;color:#aebbd6}.filters input{min-width:210px;border:1px solid #3a4a70;border-radius:8px;background:#0c1324;color:#eef2ff;padding:8px 10px}.filters button{border:1px solid #3a4a70;border-radius:8px;background:#18223a;color:#eef2ff;padding:8px 12px;cursor:pointer}.filters a{padding:8px 0}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:1510px}th,td{text-align:left;padding:10px;border-bottom:1px solid #26324f;vertical-align:top}th{font-size:12px;color:#aebbd6}td{font-size:13px}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.pager{margin-top:16px}.actions{display:flex;gap:8px;flex-wrap:wrap;min-width:190px}.actions button{border:1px solid #3a4a70;border-radius:8px;background:#18223a;color:#eef2ff;padding:7px 10px;cursor:pointer}.actions button:disabled{cursor:wait;opacity:.55}.action-status{min-height:18px;margin:8px 0 0;color:#b8c6e6;max-width:320px;overflow-wrap:anywhere}.action-status.error{color:#ffb8b8}.secret-reveal{margin-top:8px;padding:9px;border:1px solid #56698f;border-radius:8px;background:#0c1324;max-width:320px;overflow-wrap:anywhere}.secret-reveal code{display:block;margin-top:5px;white-space:pre-wrap;word-break:break-all}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/users" aria-current="page">Users</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals">Referrals</a><a href="/forced-join">Forced Join</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main id="user-admin" data-csrf="{{.CSRF}}"><section class="panel">
<h1>Users</h1>
<p class="muted">Authoritative Control Plane inventory. Provisioning phase is the exact durable lifecycle phase when present. Enable/Disable and Rotate secret use the established proxy lifecycle API. Telegram username, traffic, Node/Sponsor assignment, generic secret status and general last activity are not inferred.</p>
<form class="filters" method="get" action="/users" data-user-filter-form>
<label>Telegram ID<input type="text" name="telegram_id" inputmode="numeric" autocomplete="off" value="{{.TelegramIDFilter}}"></label>
<label>Proxy username<input type="text" name="proxy_username" autocomplete="off" value="{{.ProxyUsernameFilter}}"></label>
{{if .Limit}}<input type="hidden" name="limit" value="{{.Limit}}">{{end}}
<button type="submit">Filter</button>{{if .HasFilters}}<a href="/users">Clear filters</a>{{end}}
</form>
{{if .Users}}
<div class="table-wrap"><table>
<thead><tr><th>Telegram ID</th><th>Proxy username</th><th>Enabled</th><th>Sync state</th><th>Provisioning phase</th><th>Last error</th><th>Available bytes</th><th>Nearest expiry</th><th>Referrals</th><th>Created at</th><th>Updated at</th><th>Actions</th></tr></thead>
<tbody>
{{range .Users}}
<tr data-proxy-username="{{.ProxyUsername}}"><td class="tag">{{.TelegramID}}</td><td class="tag">{{.ProxyUsername}}</td><td>{{.DesiredEnabled}}</td><td>{{.SyncState}}</td><td class="tag">{{userProvisioningPhase .ProvisioningPhase}}</td><td class="tag">{{userErrorCode .LastErrorCode}}</td><td class="tag">{{.AvailableBytes}}</td><td>{{userOptionalTime .NearestExpiry}}</td><td class="tag">{{.ReferralCount}}</td><td>{{userTime .CreatedAt}}</td><td>{{userTime .UpdatedAt}}</td><td>{{if .ProxyUsername}}<div class="actions">{{if .DesiredEnabled}}<button type="button" data-proxy-action="disable">Disable</button>{{else}}<button type="button" data-proxy-action="enable">Enable</button>{{end}}<button type="button" data-proxy-action="rotate-secret">Rotate secret</button><button type="button" data-proxy-action="reconcile">Reconcile quota</button></div><p class="action-status" data-action-status role="status" aria-live="polite"></p><div class="secret-reveal" data-secret-reveal hidden><strong>Shown once. Save it now; it will not be shown again.</strong><code data-secret-value></code></div>{{else}}<span class="muted" data-proxy-actions-unavailable>Unavailable</span>{{end}}</td></tr>
{{end}}
</tbody></table></div>
{{else}}{{if .HasFilters}}<p class="empty">No users match the current filters.</p>{{else}}<p class="empty">No Telegram users yet.</p>{{end}}{{end}}
{{if .NextPageURL}}<p class="pager"><a href="{{.NextPageURL}}">Older users</a></p>{{end}}
</section></main>
<script>
(function () {
  "use strict";
  const root = document.getElementById("user-admin");
  if (!root) return;
  const csrf = root.dataset.csrf || "";
  const allowedActions = new Set(["enable", "disable", "rotate-secret", "reconcile"]);
  const filterForm = document.querySelector("[data-user-filter-form]");
  if (filterForm) {
    filterForm.addEventListener("submit", function () {
      filterForm.querySelectorAll('input[name="telegram_id"], input[name="proxy_username"]').forEach(function (input) {
        if (input.value === "") input.disabled = true;
      });
    });
  }

  function setBusy(row, busy) {
    row.querySelectorAll("button[data-proxy-action]").forEach(function (button) {
      button.disabled = busy;
    });
  }

  function setStatus(row, message, isError) {
    const status = row.querySelector("[data-action-status]");
    if (!status) return;
    status.textContent = String(message || "").slice(0, 256);
    status.classList.toggle("error", Boolean(isError));
  }

  async function problemText(response) {
    let payload = null;
    try {
      payload = await response.json();
    } catch (_) {
      payload = null;
    }
    const code = payload && typeof payload.code === "string" ? payload.code : "REQUEST_FAILED";
    const message = payload && typeof payload.message === "string" ? payload.message : "The proxy lifecycle request failed.";
    return (code + ": " + message).slice(0, 256);
  }

  document.addEventListener("click", async function (event) {
    const button = event.target.closest("button[data-proxy-action]");
    if (!button) return;
    const row = button.closest("tr[data-proxy-username]");
    if (!row) return;
    const username = row.dataset.proxyUsername || "";
    const action = button.dataset.proxyAction || "";
    if (!username || !allowedActions.has(action)) {
      setStatus(row, "Lifecycle action is unavailable for this row.", true);
      return;
    }
    if (action === "rotate-secret" && !window.confirm("Rotate this proxy secret? The previous secret will stop working.")) {
      return;
    }

    setBusy(row, true);
    setStatus(row, "Applying lifecycle action…", false);
    try {
      const endpoint = "/api/proxy/users/" + encodeURIComponent(username) + "/" + action;
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          "Accept": "application/json",
          "X-CSRF-Token": csrf
        }
      });
      if (!response.ok) {
        setStatus(row, await problemText(response), true);
        return;
      }
      if (action === "reconcile") {
        setStatus(row, "Quota reconciliation queued.", false);
        return;
      }
      if (action === "rotate-secret") {
        const payload = await response.json();
        const secret = payload && typeof payload.secret === "string" ? payload.secret : "";
        if (!secret) {
          setStatus(row, "Secret rotation returned an invalid response.", true);
          return;
        }
        const reveal = row.querySelector("[data-secret-reveal]");
        const value = row.querySelector("[data-secret-value]");
        if (!reveal || !value) {
          setStatus(row, "Secret rotation succeeded but the reveal area is unavailable.", true);
          return;
        }
        value.textContent = secret;
        reveal.hidden = false;
        setStatus(row, "Secret rotated. The new secret is shown once below.", false);
        return;
      }
      window.location.reload();
    } catch (_) {
      setStatus(row, "REQUEST_FAILED: The proxy lifecycle request could not be completed.", true);
    } finally {
      setBusy(row, false);
    }
  });
}());
</script>
</body>
</html>`))

func (s *Server) handleUserAdminPage(w http.ResponseWriter, r *http.Request) {
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
	query, ok := parseUserAdminQuery(w, r)
	if !ok {
		return
	}
	page, err := useradmin.List(r.Context(), s.db, query, time.Now().UTC())
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	telegramIDFilter := ""
	if query.TelegramID > 0 {
		telegramIDFilter = strconv.FormatInt(query.TelegramID, 10)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = userPageTemplate.Execute(w, userPageData{
		Username:            session.Admin.Username,
		CSRF:                sessionCSRF(token),
		Users:               page.Items,
		TelegramIDFilter:    telegramIDFilter,
		ProxyUsernameFilter: query.ProxyUsername,
		Limit:               query.Limit,
		HasFilters:          query.TelegramID > 0 || query.ProxyUsername != "",
		NextPageURL:         userPageNextURL(query, page.NextBeforeID),
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
	if query.TelegramID > 0 {
		values.Set("telegram_id", strconv.FormatInt(query.TelegramID, 10))
	}
	if query.ProxyUsername != "" {
		values.Set("proxy_username", query.ProxyUsername)
	}
	return "/users?" + values.Encode()
}
