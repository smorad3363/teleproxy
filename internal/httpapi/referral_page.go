package httpapi

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/referral"
	"github.com/smorad3363/teleproxy/internal/settings"
)

type referralPageData struct {
	Username    string
	CSRF        string
	Reward      settings.ReferralRewardSettings
	History     []referral.HistoryEntry
	NextPageURL string
}

var referralPageTemplate = template.Must(template.New("referrals").Funcs(template.FuncMap{
	"referralTime": func(value *time.Time) string {
		if value == nil {
			return "—"
		}
		return value.UTC().Format(time.RFC3339)
	},
	"referralRequiredTime": func(value time.Time) string {
		return value.UTC().Format(time.RFC3339)
	},
	"referralReason": func(value *string) string {
		if value == nil || *value == "" {
			return "—"
		}
		return *value
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Referrals</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:24px}.panel{border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}button.primary{background:#dbe6ff;color:#111827;border-color:#dbe6ff;font-weight:700}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}.table-wrap{overflow:auto}table{width:100%;border-collapse:collapse;min-width:980px}th,td{text-align:left;padding:10px;border-bottom:1px solid #26324f;vertical-align:top}th{font-size:12px;color:#aebbd6}td{font-size:13px}.tag{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.empty{color:#9aa8c4}.pager{margin-top:16px}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals" aria-current="page">Referrals</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel">
<h1>Referral reward settings</h1>
<p class="muted">Configure reward amount and expiry only. Reward recipient selection and issuance remain intentionally disabled until the product contract is explicit.</p>
<form id="referral-settings">
<div class="grid">
<label>Reward bytes<input name="bytes" type="number" min="1" step="1" value="{{.Reward.Bytes}}" required></label>
<label>Expiry days<input name="expiry_days" type="number" min="1" step="1" value="{{.Reward.ExpiryDays}}" required></label>
</div>
<div class="actions"><button class="primary" type="submit">Save settings</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section class="panel">
<h1>Referral history</h1>
{{if .History}}
<div class="table-wrap"><table>
<thead><tr><th>ID</th><th>Inviter Telegram ID</th><th>Invitee Telegram ID</th><th>Status</th><th>Rejection</th><th>Eligible at</th><th>Finalized at</th><th>Created at</th><th>Updated at</th></tr></thead>
<tbody>
{{range .History}}
<tr><td class="tag">{{.ID}}</td><td class="tag">{{.InviterTelegramID}}</td><td class="tag">{{.InviteeTelegramID}}</td><td>{{.Status}}</td><td>{{referralReason .RejectionReason}}</td><td>{{referralTime .EligibleAt}}</td><td>{{referralTime .FinalizedAt}}</td><td>{{referralRequiredTime .CreatedAt}}</td><td>{{referralRequiredTime .UpdatedAt}}</td></tr>
{{end}}
</tbody></table></div>
{{else}}<p class="empty">No referral history yet.</p>{{end}}
{{if .NextPageURL}}<p class="pager"><a href="{{.NextPageURL}}">Older referrals</a></p>{{end}}
</section>
</main>
<script>
(() => {
  const form = document.getElementById('referral-settings');
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';
  const status = form.querySelector('.status');

  form.addEventListener('submit', async event => {
    event.preventDefault();
    status.textContent = '';
    const bytes = form.elements.bytes.value.trim();
    const expiryDays = form.elements.expiry_days.value.trim();
    if (!/^[1-9][0-9]*$/.test(bytes) || !/^[1-9][0-9]*$/.test(expiryDays)) {
      status.textContent = 'Reward bytes and expiry days must be positive whole numbers.';
      return;
    }
    try {
      const response = await fetch('/api/referral/reward-settings', {
        method: 'PUT',
        credentials: 'same-origin',
        headers: {'Content-Type': 'application/json', 'X-CSRF-Token': csrf},
        body: '{"bytes":' + bytes + ',"expiry_days":' + expiryDays + '}'
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

func (s *Server) handleReferralPage(w http.ResponseWriter, r *http.Request) {
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
	query, ok := parseReferralHistoryQuery(w, r)
	if !ok {
		return
	}
	reward, err := settings.ReferralReward(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	page, err := referral.History(r.Context(), s.db, query)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := referralPageTemplate.Execute(w, referralPageData{
		Username:    session.Admin.Username,
		CSRF:        sessionCSRF(token),
		Reward:      reward,
		History:     page.Items,
		NextPageURL: referralPageNextURL(query, page.NextBeforeID),
	}); err != nil {
		return
	}
}

func referralPageNextURL(query referral.HistoryQuery, next *int64) string {
	if next == nil {
		return ""
	}
	values := url.Values{"before_id": []string{strconv.FormatInt(*next, 10)}}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return "/referrals?" + values.Encode()
}
