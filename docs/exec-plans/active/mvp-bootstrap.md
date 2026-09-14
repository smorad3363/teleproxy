# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `65972e0ca743eac5eba2cd753677a39c7c47c932`
Most recent verified code CI: `34829668347` PASS on `main`; branch-equivalent code CI `34828049841` PASS
Current branch checkpoint: `65972e0ca743eac5eba2cd753677a39c7c47c932`
Current branch CI: `34828049841` PASS

Historical execution detail is preserved without deletion:
- through CP-059 at `docs/exec-plans/archive/mvp-bootstrap-through-cp059.md`;
- CP-060 through the fully scoped Stage 11W pre-implementation state at
  `docs/exec-plans/archive/mvp-bootstrap-through-cp068.md`, using prior active-plan blob
  `546cd16b5985af5518fc7459cb5a159c14d85e34` byte-for-byte.

## Recovery contract

Resume from `Latest verified code checkpoint`, then inspect every later branch commit,
file diff, and CI result before writing. Repair an active partial milestone first.
Never reset, clean, force-push, or overwrite unrelated work. If connector-side staging
is interrupted before an atomic branch move, re-read HEAD and this plan and reconstruct
from current HEAD plus the verified intended diff. Unreachable Git objects are not
published branch state. Failed or superseded candidates remain historical evidence and
are repaired forward only.

## Non-negotiable architecture

- SQLite WAL/NORMAL is authoritative Control Plane state.
- Credit Buckets are authoritative quota/reward state.
- Telemt quota/expiry is only an enforcement projection.
- Control Plane and Telemt lifecycles remain independent.
- Plaintext admin/API/Bot/webhook/MTProto secrets are never logged or persisted in
  SQLite. Bot token/webhook-secret persistence is permitted only as owner-readable secret
  files outside the database.
- Migrations remain additive/backward-compatible.

## Current verified checkpoints

- CP-069 User inventory exact provisioning phase filter:
  `8b57acc7e68fa0b9f2678c2735a7d3804325a392`, CI `34649134519` PASS; promotion
  `a41dd6232c01e6ed738d3496d525ad55f23f01dd`, CI `34649427482` PASS.
- Stage 12F scope docs:
  `0ec8a0ca48fe1b5254e978f5fc11b4f6da16d185`, CI `34649791579` PASS.
- CP-070 Control image Docker readiness healthcheck:
  `fa88c94b6ec654035a7fa086319542322ad5003e`, CI `34650289404` PASS; promotion
  `59e34aa2d565ecf2cb7cae7a7f5fd286a4457f44`, CI `34650530913` PASS.
- Post-CP-070 recovery docs:
  `ff8fde54441bc630ca964fd2dce493f8c3486ee2`, CI `34653782213` PASS.
- Stage 12G scope docs:
  `488bfd52ee7dd4f3c3f6dd326f946eae66ff250e`, CI `34657412890` PASS.
- Stage 12G first candidate:
  `a304b7758566284b2d9fdcbe7462b945dfed6e1d`, CI `34657664813` PASS, but self-review
  found one accidental out-of-scope persisted proxy-bind assignment regression; repaired
  forward without reset or force-push.
- CP-071 Installer gates success on Control Docker health:
  `ca9fcf4f605cfb10eef2db8766f14df8d3b5443b`, CI `34657708764` PASS; promotion
  `63fc78dee63f02265b76dcf3c9522fa8b1c11a49`, CI `34657897001` PASS.
- Stage 12H scope docs:
  `935c9bb3ce2824a5b4dcb37e5aa1cb9bbfaafd48`, CI `34658114344` PASS.
- CP-072 `tproxy doctor` Control Docker-health gate:
  `229368ba6bde81f449ed1c0e91deb0c4a9d3ca46`, CI `34658384988` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and installer/
  Telemt E2E install/rerun.
- CP-072 promotion docs:
  `f0042698589747d228e7de1a9a8612548cdbb8de`, CI `34658572418` PASS across every
  established gate.
- Post-CP-072 blocked recovery docs:
  `d1109a83211903dcf5dd0602b9154f37cf3dc8cf`, CI `34658771157` PASS.
- Stage 11X initial scope docs:
  `5706006fb59bc3d4012aaf1cbb4a7266e5790bbf`, CI `34665535932` PASS.
- Stage 11X contract refinement:
  `958f038bd006de39f5cdb5426a06d441fd8fe3d7`, CI `34665685922` PASS.
- Stage 11X implementation candidate:
  `0962113fcdb4a85f626c7b9281a1b92cc79db3d7`, CI `34666450151` FAILED at SQLite
  migration tests; repaired forward.
- Referral migration-count repair:
  `7cd4cdd6a264d506e176a97592b770f3789f0512`, CI `34666509982` FAILED only at installer
  E2E after Go/format/vet/migration gates passed; repaired forward.
- Stage 11X final runtime/settings milestone:
  `45ca6f94bf81656fd33a483863d0cd56a16ed0f3`, CI `34826989386` PASS.
- Explicit panel-bind rerun override:
  `65972e0ca743eac5eba2cd753677a39c7c47c932`, branch CI `34828049841` PASS.
- `main` was fast-forwarded without force to the same `65972e0...` checkpoint; CI
  `34829668347` PASS.

## Explicit blockers still unresolved

### Stage 9D — Proxy Node test/health/status — BLOCKED
Per-Node credential source and `internal_api_endpoint` runtime meaning are not
established. Do not probe Nodes or invent credential storage/transport.

### Stage 7D2C — Referral reward issuance — BLOCKED
Reward recipient is unresolved: inviter, invitee, or both. Do not infer a recipient.

### Stage 7D3B — Referral anti-abuse — BLOCKED
Cap scope/defaults, cooldown semantics/default, blacklist subject, and suspicious score
inputs/threshold are unresolved. Do not invent them.

Also do not invent Node/Sponsor assignment/routing, user-scoped referral-tree/history
semantics, audit mutation/redaction wiring, Administrator RBAC enforcement,
backup/restore, update/rollback, watchdog policy, Dashboard metrics, new Telemt topology,
or new plaintext-secret persistence.

## Remaining CI roadmap — REVIEWED / NOT SCOPED

Current CI explicitly covers gofmt, go vet, full Go tests, SQLite migration tests,
ShellCheck, Docker build, Compose config validation, and installer/Telemt E2E
install/rerun. Do not invent a targeted-integration selector, secret scanner contract,
dependency scanner/version policy, target-distro harness, or upgrade/rollback smoke
before those contracts exist.

## Completed recovery/runtime milestones

### Stage 12F / CP-070 — Control Docker readiness healthcheck

The Control image has a first-class Docker health contract using the existing
unauthenticated `/readyz` database-readiness endpoint and the built-in
`teleproxy-control healthcheck <http-url>` process mode. Control and Telemt keep
independent restart/lifecycle policies.

### Stage 12G / CP-071 — Installer gates success on Control Docker health

After direct Control `/readyz` succeeds, the host installer waits boundedly for the
Control container Docker health status to become exactly `healthy` before the independent
Telemt health gate and before installation success.

### Stage 12H / CP-072 — `tproxy doctor` requires Control Docker health

`tproxy doctor` reports `Control Docker health: <status>` from the existing Docker health
object and succeeds on that check only for exactly `healthy`.

### Stage 11X — Panel-managed Telegram Bot runtime settings — COMPLETE

The Web Panel manages Bot Username, Admin Chat ID, enabled state and write-only Bot Token.
Bot token/webhook secret remain secret-file only. Test Bot sends a fixed message to the
configured Admin Chat ID. Control reads enabled settings at startup. Public TLS/domain
exposure and webhook registration were intentionally not automated by Stage 11X.

## Stage 11Y — Telegram polling + Bot Content runtime delivery — SCOPED

The user has now established the missing product behavior for the immediate Bot path:
`/start` must work on a normal single-server install without requiring the operator to
first configure a public HTTPS webhook, and panel-managed Bot Content must affect the
messages users actually receive.

Implementation contract:
- An enabled Bot uses Telegram Bot API long polling from Control as the default inbound
  update transport. It requires no public domain, TLS certificate, firewall opening or
  Telegram webhook registration.
- On polling startup, Control calls `deleteWebhook` with `drop_pending_updates=false` so
  a stale/external webhook registration does not keep `getUpdates` in conflict and queued
  updates are not intentionally discarded.
- The existing authenticated `/telegram/webhook` handler stays available as a compatibility
  endpoint, but an enabled runtime’s polling loop owns inbound Telegram delivery and clears
  any registered webhook before polling.
- Polling accepts only `message` and `callback_query` updates, uses bounded long-poll and
  retry intervals, advances the offset only after an update is safely handled, and never
  logs or surfaces token-bearing Bot API URLs.
- Existing `/start`, Forced Join, idempotent start-gift, proxy provisioning, quota and
  referral behavior remain the application source of truth.
- Existing Bot Content slots become request-time runtime overrides without restart:
  `welcome` prefixes normal start replies; `forced_join` replaces the default join prompt;
  `proxy` prefixes the proxy section; `referral` prefixes the referral section. Missing
  slots preserve the existing built-in fallback text.
- Custom Bot Content is bounded when composed so dynamic account/proxy/referral data is
  not allowed to produce a Telegram message over 4096 characters.
- A successful normal `/start` reply gets one-tap inline buttons when data exists:
  `Connect Proxy` uses the already validated `tg://proxy` link and `Invite Friends` uses
  the generated `https://t.me/<bot>?start=<code>` link. Forced Join continues to render
  one channel button per missing required channel plus the existing Recheck action.
- `expired`, `no_credit` and `support` remain stored slots but are not newly wired until a
  corresponding established runtime state/action exists; do not invent those triggers.

First Stage 11Y code candidate:
- `5729360d59f507a408a1f7eabfec08ce26afb8fe`, CI `34833483776` FAILED only in the
  full Go test step. Format, vet and SQLite migration gates passed. The failure was one
  pre-existing webhook recheck test that still expected a plain ready message even though
  Stage 11Y intentionally adds the `Connect Proxy` inline action to that same ready reply.
- Forward repair candidate `75a89b63cdd85580c9713a6d56c9b9122e0d0069` changes only that legacy
  test expectation to assert the new one-tap proxy button. No runtime behavior was weakened
  to satisfy the test.

Verification scope:
- focused Bot API long-poll/deleteWebhook tests;
- polling dispatch/offset behavior;
- request-time Bot Content override tests;
- inline proxy/referral button validation;
- existing Telegram/Forced Join/provisioning tests;
- full repository CI once the whole staged batch is published.

## Panel UX review / next redesign — DESIGN READY, IMPLEMENTATION DEFERRED

The current panel is a collection of independently styled server-rendered pages with
repeated headers/navigation/CSS. The dashboard is mostly a vertical list of links rather
than an operational dashboard. Proxy Nodes exposes low-level future metadata that is not
connected to current runtime routing, so it reads like an unfinished infrastructure form.

The next redesign should use one consistent application shell and borrow the useful
interaction pattern from `Sir-MmD/vpn-ui` without copying its protocol complexity:
- persistent sidebar/navigation, compact top bar, responsive content area;
- dashboard summary cards for active users, proxy health, bot state, required channels,
  credit/referral summaries and recent actionable problems once those metrics have real
  contracts;
- primary `Create Proxy` / `New Inbound` action visible from dashboard and proxy page;
- a short wizard/modal with a live summary rail: name, listen/public address, MTProto port,
  enabled state, then optional sponsor/required-channel selection;
- advanced infrastructure fields hidden behind an Advanced section rather than shown in
  the default path;
- existing Proxy Node metadata remains separate from the current single Telemt inbound
  until multi-node routing semantics are explicitly contracted;
- sponsor-channel selection in the quick-create UX must reuse established Sponsor/Forced
  Join records; it must not silently invent Node/Sponsor routing semantics.

No panel redesign code is part of Stage 11Y. This section is the approved design target
for the next user-requested implementation stage.

## Current next action

Publish the Stage 11Y forward repair on top of the failed candidate and require complete
GitHub CI before calling it PASS. The first candidate already proved format, vet and
SQLite migration gates; the repair must still pass the full workflow including installer
E2E. After Stage 11Y is green, begin the panel redesign only when explicitly requested.
