# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `7ef2b3b1083670ddb4bc303df06d6a9002856539` (Bot polling delivery/diagnostic repair)
Most recent verified code CI: `34838255090` PASS
Current branch checkpoint: `7ef2b3b1083670ddb4bc303df06d6a9002856539` (latest verified code before this documentation synchronization)
Current branch CI: `34838255090` PASS

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

The commit containing this plan synchronization is documentation-only and follows
`7ef2b3b...`; before any later code change, inspect that docs commit and its CI as part of
normal recovery. Promotion to `main` must remain a non-force fast-forward and must receive
its own complete CI before being described as published/final.

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
- Stage 11Y first polling/content candidate:
  `5729360d59f507a408a1f7eabfec08ce26afb8fe`, CI `34833483776` FAILED only in one
  legacy webhook-recheck expectation after format, vet and migration gates passed.
- Stage 11Y forward repair:
  `75a89b63cdd85580c9713a6d56c9b9122e0d0069` updated only that legacy test to the
  intentional one-tap `Connect Proxy` response behavior.
- Stage 11Y recovery checkpoint:
  `0dafb84228775b30434a0d0de6470dfa139b328d`, CI `34833772706` PASS across all
  established Go, migration, shell, Docker, Compose and installer E2E gates.
- Stage 13A scope docs:
  `1b7746ebc8f29fd053371eb851f69efabbafd809`.
- Stage 13A first implementation candidate:
  `7c9451c2fbf67413f18d39e0df96d638968a35b3`, CI `34834970777` FAILED only at the
  initial gofmt gate. Vet, migrations, tests and installer were therefore not run.
- Stage 13A format repair:
  `75a0ce259eda189f9f887f7b9b7f8aa20cfdd4c8`, CI `34835197825` FAILED only at full
  Go tests after format, vet and SQLite migration gates passed. The dashboard rewrite had
  removed the established server-rendered Administrator and Audit Log links; repaired
  forward without reverting the shell/Quick Proxy work.
- Stage 13A final panel/navigation repair:
  `866a4868e7932566b799978c74ef3ed1e6336ec2`, CI `34835431754` PASS across every
  established Go, migration, shell, Docker, Compose and installer E2E gate.
- Bot `/start` provisioning fallback runtime repair:
  `f53a519cee10320c82b2945dd5cbe3b6ad055fcb` keeps the account/credit response available
  when Telemt proxy provisioning temporarily fails, leaving the existing proxy-pending
  presentation to the response formatter instead of aborting the handled update.
- Bot `/start` fallback regression candidate:
  `ff3f9ab2fa48058da85c7e566bdbecc3d375e45e`, CI `34836483075` FAILED only at full
  Go tests because one older provisioning test still asserted the superseded error-return
  behavior; format, vet and SQLite migration gates passed and installer was skipped.
- Bot `/start` fallback test repair:
  `d9ac31a10dad5dffdeb040b0ae06006771db3470`, CI `34836627620` PASS across every
  established Go, migration, shell, Docker, Compose and installer E2E gate. The legacy
  test now verifies that repeated temporary provisioning failures keep exactly one start
  gift while returning a handled pending response rather than silence.
- Plan synchronization after Stage 13A/Bot fallback:
  `5156cb78add3679580c2f28013c4aca9888e0129`, branch CI `34836963977` PASS; `main` was
  then fast-forwarded without force to the same checkpoint and CI `34837240023` PASS.
- Bot polling delivery/diagnostic repair:
  `7ef2b3b1083670ddb4bc303df06d6a9002856539`, CI `34838255090` PASS across all
  established Go, migration, shell, Docker, Compose and installer E2E gates. Polling now
  treats a start reply as delivered only after Telegram accepts a message; an inline-
  keyboard send failure falls back once to the same plain text, and a total send failure
  keeps the update unacknowledged for retry. Terminal polling API failures such as
  unauthorized/rejected/invalid responses now escape the polling loop so Control logs a
  safe runtime failure instead of looping silently.

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
configured Admin Chat ID. Control reads enabled settings at startup. Runtime settings are
loaded on Control startup, so a settings change that affects the enabled runtime still
requires a Control restart before polling uses the new configuration.

### Stage 11Y — Telegram polling + Bot Content runtime delivery — COMPLETE

An enabled Bot uses bounded Telegram long polling, clears stale webhook registration
without dropping pending updates, and does not require a public domain/TLS setup for normal
`/start` delivery. Welcome, Forced Join, Proxy and Referral content slots are read at
request time. Normal ready replies may render one-tap `Connect Proxy` and `Invite Friends`
buttons. CI `34833772706` is the verified Stage 11Y recovery checkpoint.

### Stage 13A — Panel shell + Quick Proxy UX — COMPLETE

Established authenticated panel pages use one consistent responsive application shell
with desktop/mobile navigation. The dashboard is an operational overview based only on
established state. Proxy Nodes is presented as `Proxies`, with Quick Create requiring only
Name, Region, Public address and MTProto port by default. Legacy host metadata mirrors the
public address when omitted, and the existing single-node compatibility metadata endpoint
`http://telemt:9091` remains under Advanced. Optional Required Channel creation reuses the
existing global Forced Join contract and is explicitly not Node/Sponsor routing. No Node
health probing, lifecycle behavior or Sponsor assignment was invented. Final Stage 13A CI
`34835431754` PASS.

### Bot `/start` proxy-provisioning fallback — COMPLETE

The Start transaction remains authoritative for user creation and credit gift. A temporary
Telemt/proxy provisioning failure no longer turns an otherwise handled `/start` into a
silent retry loop. The Bot returns the established account/credit response with no proxy
link, which renders as proxy provisioning pending; a later `/start` can retry provisioning.
Start-gift idempotency remains intact. CI `34836627620` PASS.

### Bot polling delivery reliability — COMPLETE

Polling no longer advances the Telegram update offset when the outbound start reply fails.
If an inline keyboard cannot be sent, the same bounded response is retried once as plain
text. If that also fails, the update remains pending for the existing bounded retry loop.
After webhook cleanup succeeds, terminal Bot API failures during polling are returned to
the Control runtime logger rather than being swallowed indefinitely. No Bot token or
Telegram error payload is logged. CI `34838255090` PASS.

## Current next action

Require the documentation synchronization commit containing this plan to complete all
established CI gates. If green, compare `main` to `agent/mvp-bootstrap`, require a strict
fast-forward, move `main` without force to the same verified HEAD, and require the resulting
`main` CI to complete successfully before describing the repair as finally published.
