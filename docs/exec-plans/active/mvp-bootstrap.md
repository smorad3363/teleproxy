# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `229368ba6bde81f449ed1c0e91deb0c4a9d3ca46` (CP-072)
Most recent verified code CI: `34658384988` PASS
Current branch checkpoint: `5706006fb59bc3d4012aaf1cbb4a7266e5790bbf` (Stage 11X initial scope docs)
Current branch CI: `34665535932` PASS

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
  SQLite. Stage 11X explicitly permits Bot token/webhook-secret persistence only as
  owner-readable secret files outside the database.
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

## Explicit blockers

### Stage 11H — Bot Content runtime delivery wiring — BLOCKED
Runtime composition/fallback, button/emoji relationship, and missing-slot behavior are
not defined. Do not invent them. Stage 11X below is only Bot runtime configuration and
must not change Bot Content composition semantics.

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
backup/restore, update/restart/log/version-source semantics, Dashboard metrics, new
Telemt topology, or secret persistence outside the narrowly established Stage 11X Bot
secret-file contract.

## Remaining CI roadmap — REVIEWED / NOT SCOPED

Current CI explicitly covers gofmt, go vet, full Go tests, SQLite migration tests,
ShellCheck, Docker build, Compose config validation, and installer/Telemt E2E
install/rerun. Do not invent a targeted-integration selector, secret scanner contract,
dependency scanner/version policy, target-distro harness, or upgrade/rollback smoke
before those contracts exist.

## Recovery/health milestones completed

### Stage 12F / CP-070 — Control Docker readiness healthcheck

The Control image has a first-class Docker health contract using the existing
unauthenticated `/readyz` database-readiness endpoint and the built-in
`teleproxy-control healthcheck <http-url>` process mode. Control and Telemt keep
independent restart/lifecycle policies.

### Stage 12G / CP-071 — Installer gates success on Control Docker health

After direct Control `/readyz` succeeds, the host installer waits boundedly for the
Control container Docker health status to become exactly `healthy` before the independent
Telemt health gate and before installation success. The final scoped diff is exactly
`scripts/install-host.sh` (+24 lines).

### Stage 12H / CP-072 — `tproxy doctor` requires Control Docker health

`tproxy doctor` now reports `Control Docker health: <status>` from the existing Docker
health object and succeeds on that check only for exactly `healthy`. Existing Compose,
running-container, direct `/readyz`, and Telemt health checks remain intact. Installer
E2E asserts the healthy line after first install and rerun.

The final CP-072 code diff contains exactly:
- `bin/tproxy` (+13 lines)
- `tests/installer_e2e.sh` (+1 assertion)

No restart/watchdog action, restart-loop state, notification, Compose `depends_on`,
schema/migration/API/topology, secret-persistence, or Control/Telemt lifecycle-coupling
change was added.

## Post-CP-072 roadmap/repository review

The repository covers the previously contract-defined recovery primitives. Remaining
watchdog, backup/update/rollback, product routing, RBAC, Bot Content composition and CI
tooling contracts stay blocked as recorded above.

The user has explicitly established one previously missing product contract: Telegram
Bot runtime credentials and administrator Chat ID must be configurable inside the Web
Panel rather than by editing an `.env` file or installer command. That decision scopes
Stage 11X below and does not resolve unrelated blockers.

## Stage 11X — Panel-managed Telegram Bot runtime settings — SCOPED

Product/runtime contract established by the user:
- The existing `/settings` Web Panel gains a Telegram Bot section for `Bot Username`,
  `Admin Chat ID`, `Enabled`, and Bot Token replacement.
- Bot Token is write-only in the UI/API. Reads expose only whether a token is configured;
  the plaintext token is never returned, logged, or stored in SQLite.
- Bot Token and Telegram webhook secret persist only as dedicated owner-readable secret
  files (`0600`) under the existing Teleproxy secrets directory. The Control container
  may write those two files; no other plaintext-secret persistence is introduced.
- Webhook secret is generated cryptographically by Control when first enabling/saving Bot
  configuration and is never rendered back to the UI.
- Bot Username, positive administrator private-chat ID, enabled state, and timestamps are
  stored in an additive SQLite settings table. Admin Chat ID is the destination for the
  panel's fixed Bot configuration test message; no broader notification policy is
  inferred.
- Saving settings is CSRF-protected and bounded. Token input may be left empty only when a
  valid token file already exists; an empty token never clears an existing token.
- A `Test Bot` action sends one fixed, non-user-controlled test message to the configured
  Admin Chat ID using the configured token. Telegram/API failure responses remain generic
  and must not leak token-bearing URLs or response bodies.
- Control reads enabled Bot runtime settings from SQLite and the secret files during
  startup. Saving from the panel does not restart containers automatically; inbound
  webhook activation/deactivation or username changes take effect after the operator runs
  `tproxy restart`. The fixed `Test Bot` action uses the newly saved values immediately.
- Disabled or absent Bot settings never prevent Control/Telemt startup or health. Enabled
  settings are validated on startup; invalid/missing secret files fail Bot runtime setup
  without exposing secret contents.
- Existing `/telegram/webhook` authentication, rate limiting, Forced Join/start/
  provisioning behavior and Telemt quota reconciliation remain unchanged once configured.
- This milestone does not register a public Telegram webhook, invent TLS/domain exposure,
  add Bot Content delivery composition, add admin-notification behavior beyond the fixed
  test action, or add Chat-ID-based administrator authorization.

Implementation scope is limited to:
- one additive migration for non-secret Bot runtime settings;
- a small `internal/settings` Bot runtime store/validator and tests;
- Web Panel/API wiring under the existing `/settings` surface plus tests;
- safe Bot secret-file read/write/generation helpers and tests;
- startup Bot runtime wiring using the existing Telegram Bot/provisioning components;
- Compose/installer permissions and fixed secret-file path wiring required for Control to
  write Bot secret files;
- focused installer E2E assertions that do not expose token contents.

No unrelated schema, Sponsor/Node routing, referral issuance/anti-abuse, RBAC, backup,
update/rollback, watchdog, Bot Content composition, new Telemt topology or public webhook
registration is in scope.

## Current next action

Require full CI PASS on this revised Stage 11X scope commit. Then implement the scoped
migration, settings store, secret-file handling, panel/API surface, startup webhook
configuration, and focused tests as one forward-only milestone. Require full code CI PASS
before promotion. Preserve every unrelated blocker and architecture invariant above.
