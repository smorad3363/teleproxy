# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `0dafb84228775b30434a0d0de6470dfa139b328d` (Stage 11Y polling/content runtime)
Most recent verified code CI: `34833772706` PASS
Current branch checkpoint: `0dafb84228775b30434a0d0de6470dfa139b328d`
Current branch CI: `34833772706` PASS

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
- Stage 11Y first polling/content candidate:
  `5729360d59f507a408a1f7eabfec08ce26afb8fe`, CI `34833483776` FAILED only in one
  legacy webhook-recheck expectation after format, vet and migration gates passed.
- Stage 11Y forward repair:
  `75a89b63cdd85580c9713a6d56c9b9122e0d0069` updated only that legacy test to the
  intentional one-tap `Connect Proxy` response behavior.
- Stage 11Y recovery checkpoint:
  `0dafb84228775b30434a0d0de6470dfa139b328d`, CI `34833772706` PASS across all
  established Go, migration, shell, Docker, Compose and installer E2E gates.

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
configured Admin Chat ID. Control reads enabled settings at startup.

### Stage 11Y — Telegram polling + Bot Content runtime delivery — COMPLETE

An enabled Bot now uses bounded Telegram long polling, clears stale webhook registration
without dropping pending updates, and no longer requires a public domain/TLS setup for
normal `/start` delivery. Welcome, Forced Join, Proxy and Referral content slots are read
at request time. Normal ready replies may render one-tap `Connect Proxy` and `Invite
Friends` buttons. CI `34833772706` is the verified full PASS checkpoint.

## Stage 13A — Panel shell + Quick Proxy UX — SCOPED

The user has approved the previously prepared panel redesign and requested implementation.
The goal is to make normal administration look and behave like one product instead of a
collection of unrelated forms, and to make adding a Telegram Proxy fast without exposing
undefined infrastructure semantics.

Implementation contract:
- Established authenticated panel pages gain one consistent responsive application shell:
  fixed desktop sidebar, mobile navigation, consistent surfaces/forms/spacing and active
  navigation. Login, health/readiness and JSON APIs are not visually wrapped or altered.
- The dashboard becomes an operational overview using only already-established system
  state. It may present Control, database, global proxy/read-only status and navigation
  cards, but must not fabricate user/traffic/revenue/health metrics whose contracts do not
  exist.
- Proxy Nodes is presented as `Proxies` in the UI while preserving its existing static
  metadata model and API compatibility.
- Quick Create asks by default only for Name, Region, one Public address and MTProto port.
  When `host` is omitted, Control mirrors the validated public address into the legacy
  `host` metadata field. Existing callers that send both fields keep their current behavior.
- The existing schema requires non-empty `internal_api_endpoint`; no migration or runtime
  probing is introduced. When Quick Create omits it, Control stores the existing single-
  node compatibility value `http://telemt:9091`. The field remains editable under
  Advanced and remains metadata only until Stage 9D is explicitly contracted.
- Advanced contains Node type, optional distinct internal-host override and the existing
  internal API endpoint. No Node health/test/lifecycle behavior is inferred.
- Quick Create may optionally create/reuse an established Forced Join Required Channel
  from Telegram chat reference, display name and Telegram join URL. The UI must state
  clearly that this is a global Bot Required Channel, not Node/Sponsor assignment.
- Existing Sponsor Profiles remain a separate management surface. No Node/Sponsor routing,
  sticky assignment, or sponsor selection algorithm is introduced.
- Existing CRUD, CSRF, escaping, no-store behavior and no-probe guarantees remain intact.

Verification scope:
- shared shell appears on authenticated panel GET pages and not on login/API/health;
- dashboard retains established state and exposes direct Create Proxy action;
- Quick Create mirrors one public address into legacy host metadata;
- omitted internal API endpoint gets only the established compatibility metadata default;
- invalid public address is rejected without mutation;
- Proxy page retains escaping, deterministic order and zero endpoint probes;
- optional Required Channel flow uses existing Forced Join APIs and is labeled global;
- full repository CI is required after the batched branch publication.

## Current next action

Publish Stage 13A as one batched forward-only branch move after final diff review. Require
complete GitHub CI before calling it PASS. Do not begin Node runtime routing, health,
Sponsor assignment, new Telemt topology or invented dashboard metrics as part of this
stage.
