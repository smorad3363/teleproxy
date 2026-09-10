package httpapi

import "html/template"

var loginTemplate = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Teleproxy Login</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;background:radial-gradient(circle at 20% 10%,#172554 0,#0b1020 42%)}main{width:min(92vw,420px);padding:32px;border:1px solid #26324f;border-radius:18px;background:#11182a;box-shadow:0 24px 70px #0007}h1{margin:0 0 6px;font-size:28px}p{color:#aebbd6;margin:0 0 24px}label{display:block;margin:14px 0 7px;font-size:14px}input{width:100%;border:1px solid #34415f;border-radius:10px;padding:12px 13px;background:#0c1323;color:#fff;font:inherit}input:focus{outline:2px solid #7c9cff;outline-offset:1px}button{width:100%;margin-top:20px;border:0;border-radius:10px;padding:12px 14px;background:#dbe6ff;color:#111827;font-weight:700;cursor:pointer}.error{padding:10px 12px;border-radius:10px;background:#35171d;color:#ffc8d0;margin-bottom:14px}small{display:block;margin-top:18px;color:#7483a4}</style>
</head>
<body><main><h1>Teleproxy</h1><p>Administrator sign in</p>{{if .Error}}<div class="error" role="alert">{{.Error}}</div>{{end}}<form method="post" action="/login"><input type="hidden" name="csrf" value="{{.CSRF}}"><label for="username">Username</label><input id="username" name="username" autocomplete="username" required autofocus><label for="password">Password</label><input id="password" name="password" type="password" autocomplete="current-password" required><button type="submit">Sign in</button></form><small>Control Plane access</small></main></body></html>`))

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Teleproxy</title><style>:root{font-family:Inter,ui-sans-serif,system-ui,sans-serif;color-scheme:dark;background:#0b1020;color:#eef2ff}body{margin:0}header{display:flex;justify-content:space-between;align-items:center;padding:18px 5vw;border-bottom:1px solid #24304c;background:#10172a}main{padding:36px 5vw}.card{max-width:720px;padding:26px;border:1px solid #26324f;border-radius:16px;background:#11182a}button{border:1px solid #52617d;border-radius:9px;padding:9px 12px;background:#18223a;color:#fff;cursor:pointer}a{color:#c9d7ff}.muted{color:#9aa8c4}</style></head>
<body><header><strong>Teleproxy</strong><form method="post" action="/logout"><input type="hidden" name="csrf" value="{{.CSRF}}"><button type="submit">Sign out</button></form></header><main><section class="card"><h1>Control Plane</h1><p>Signed in as <strong>{{.Username}}</strong>.</p><p><a href="/users">View Users</a></p><p><a href="/sponsors">Manage Sponsor Profiles</a></p><p><a href="/nodes">Manage Proxy Nodes</a></p><p><a href="/referrals">Manage Referrals</a></p><p><a href="/forced-join">Manage Forced Join</a></p><p><a href="/settings">Manage Settings</a></p><p class="muted">Core login is active. Additional Proxy, Bot and plan management will be added in later stages.</p></section></main></body></html>`))
