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
	Reward   settings.ReferralRewardSettings
	Bot      botRuntimeSettingsView
}

var startGiftPageTemplate = template.Must(template.New("settings").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="csrf-token" content="{{.CSRF}}">
<title>Teleproxy Settings</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;background:#0b1020;color:#eef2ff}header{display:flex;gap:18px;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}header nav{display:flex;gap:14px;align-items:center;flex-wrap:wrap}a{color:#c9d7ff}main{padding:30px 5vw 56px;display:grid;gap:24px}.panel{max-width:760px;border:1px solid #26324f;border-radius:16px;background:#11182a;padding:22px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}label{display:grid;gap:6px;font-size:13px;color:#cbd5e1}input{width:100%;border:1px solid #34415f;border-radius:9px;padding:10px;background:#0c1323;color:#fff;font:inherit}.check{display:flex;align-items:center;gap:8px}.check input{width:auto}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#dbe6ff;color:#111827;font-weight:700;cursor:pointer}.muted{color:#9aa8c4}.status{min-height:1.4em;color:#fda4af;margin:8px 0 0}.ok{color:#86efac}@media(max-width:600px){header{align-items:flex-start;flex-direction:column}}
</style>
</head>
<body>
<header><nav><strong>Teleproxy</strong><a href="/">Dashboard</a><a href="/users">Users</a><a href="/sponsors">Sponsors</a><a href="/nodes">Proxy Nodes</a><a href="/referrals">Referrals</a><a href="/forced-join">Forced Join</a><a href="/settings" aria-current="page">Settings</a></nav><span class="muted">Signed in as {{.Username}}</span></header>
<main>
<section class="panel">
<h1>Settings</h1>
<h2>Telegram Bot</h2>
<p class="muted">Configure the Bot runtime here. The token is write-only and is stored only in a protected secret file, never in SQLite. After saving, run <code>tproxy restart</code> on the server to apply incoming webhook activation or username changes.</p>
<form id="bot-settings" autocomplete="off">
<div class="grid">
<label>Bot Username<input name="username" type="text" maxlength="64" value="{{.Bot.Username}}" placeholder="TeleproxyBot" required></label>
<label>Admin Chat ID<input name="admin_chat_id" type="text" inputmode="numeric" pattern="[1-9][0-9]*" value="{{if .Bot.Configured}}{{.Bot.AdminChatID}}{{end}}" placeholder="123456789" required></label>
<label>Bot Token<input name="token" type="password" maxlength="256" value="" autocomplete="new-password" placeholder="{{if .Bot.TokenConfigured}}Leave blank to keep current token{{else}}Paste BotFather token{{end}}"></label>
</div>
<label class="check"><input name="enabled" type="checkbox" {{if .Bot.Enabled}}checked{{end}}>Enable Telegram Bot after next Control restart</label>
<p class="muted">Token: <strong class="{{if .Bot.TokenConfigured}}ok{{end}}">{{if .Bot.TokenConfigured}}configured{{else}}not configured{{end}}</strong>. Admin Chat ID is used by the Test Bot action only; it does not grant panel access.</p>
<div class="actions"><button type="submit">Save Bot settings</button><button id="test-bot" type="button">Test Bot</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section class="panel">
<h2>Start gift</h2>
<p class="muted">This amount applies only when a user's exactly-once start gift has not yet been created. Existing Credit Buckets are not rewritten.</p>
<form id="start-gift-settings">
<label>Start gift bytes<input name="bytes" type="text" inputmode="numeric" pattern="[1-9][0-9]*" value="{{.Bytes}}" required></label>
<div class="actions"><button type="submit">Save start gift</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
<section class="panel">
<h2>Referral reward</h2>
<p class="muted">Configure the established referral reward amount and expiry only. Reward recipient selection and issuance remain outside this settings surface.</p>
<form id="referral-reward-settings">
<div class="grid">
<label>Reward bytes<input name="reward_bytes" type="text" inputmode="numeric" pattern="[1-9][0-9]*" value="{{.Reward.Bytes}}" required></label>
<label>Expiry days<input name="reward_expiry_days" type="text" inputmode="numeric" pattern="[1-9][0-9]*" value="{{.Reward.ExpiryDays}}" required></label>
</div>
<div class="actions"><button type="submit">Save referral reward</button></div>
<p class="status" role="status" aria-live="polite"></p>
</form>
</section>
</main>
<script>
(() => {
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrf = csrfMeta ? csrfMeta.content : '';

  const requestMessage = async response => {
    let message = 'Request failed.';
    try {
      const problem = await response.json();
      if (problem && problem.message) message = problem.message;
    } catch (_) {}
    return message;
  };

  const botForm = document.getElementById('bot-settings');
  const botStatus = botForm.querySelector('.status');
  botForm.addEventListener('submit', async event => {
    event.preventDefault();
    botStatus.textContent = '';
    botStatus.classList.remove('ok');
    const username = botForm.elements.username.value.trim();
    const adminChatID = botForm.elements.admin_chat_id.value.trim();
    const token = botForm.elements.token.value.trim();
    const enabled = botForm.elements.enabled.checked;
    if (!/^[A-Za-z0-9_]{1,64}$/.test(username.replace(/^@/, '')) || !/^[1-9][0-9]*$/.test(adminChatID)) {
      botStatus.textContent = 'Bot username or Admin Chat ID is invalid.';
      return;
    }
    try {
      const body = '{"username":' + JSON.stringify(username) + ',"admin_chat_id":' + adminChatID + ',"enabled":' + (enabled ? 'true' : 'false') + ',"token":' + JSON.stringify(token) + '}';
      const response = await fetch('/api/settings/bot', {
        method: 'PUT',
        credentials: 'same-origin',
        headers: {'Content-Type': 'application/json', 'X-CSRF-Token': csrf},
        body
      });
      if (!response.ok) {
        botStatus.textContent = await requestMessage(response);
        return;
      }
      botForm.elements.token.value = '';
      botStatus.classList.add('ok');
      botStatus.textContent = 'Saved. Run tproxy restart on the server to apply incoming webhook changes.';
    } catch (_) {
      botStatus.textContent = 'Request failed.';
    }
  });

  document.getElementById('test-bot').addEventListener('click', async () => {
    botStatus.textContent = '';
    botStatus.classList.remove('ok');
    try {
      const response = await fetch('/api/settings/bot/test', {
        method: 'POST',
        credentials: 'same-origin',
        headers: {'X-CSRF-Token': csrf}
      });
      if (!response.ok) {
        botStatus.textContent = await requestMessage(response);
        return;
      }
      botStatus.classList.add('ok');
      botStatus.textContent = 'Test message sent to the configured Admin Chat ID.';
    } catch (_) {
      botStatus.textContent = 'Request failed.';
    }
  });

  const startForm = document.getElementById('start-gift-settings');
  const startStatus = startForm.querySelector('.status');
  startForm.addEventListener('submit', async event => {
    event.preventDefault();
    startStatus.textContent = '';
    const bytes = startForm.elements.bytes.value.trim();
    if (!/^[1-9][0-9]*$/.test(bytes)) {
      startStatus.textContent = 'Start gift bytes must be a positive whole number.';
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
        startStatus.textContent = await requestMessage(response);
        return;
      }
      location.reload();
    } catch (_) {
      startStatus.textContent = 'Request failed.';
    }
  });

  const referralForm = document.getElementById('referral-reward-settings');
  const referralStatus = referralForm.querySelector('.status');
  referralForm.addEventListener('submit', async event => {
    event.preventDefault();
    referralStatus.textContent = '';
    const rewardBytes = referralForm.elements.reward_bytes.value.trim();
    const expiryDays = referralForm.elements.reward_expiry_days.value.trim();
    if (!/^[1-9][0-9]*$/.test(rewardBytes) || !/^[1-9][0-9]*$/.test(expiryDays)) {
      referralStatus.textContent = 'Reward bytes and expiry days must be positive whole numbers.';
      return;
    }
    try {
      const response = await fetch('/api/referral/reward-settings', {
        method: 'PUT',
        credentials: 'same-origin',
        headers: {'Content-Type': 'application/json', 'X-CSRF-Token': csrf},
        body: '{"bytes":' + rewardBytes + ',"expiry_days":' + expiryDays + '}'
      });
      if (!response.ok) {
        referralStatus.textContent = await requestMessage(response);
        return;
      }
      location.reload();
    } catch (_) {
      referralStatus.textContent = 'Request failed.';
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
	reward, err := settings.ReferralReward(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	bot, err := readBotRuntimeSettingsView(r.Context(), s.db)
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
		Reward:   reward,
		Bot:      bot,
	})
}
