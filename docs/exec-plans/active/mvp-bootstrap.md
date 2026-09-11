# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `229368ba6bde81f449ed1c0e91deb0c4a9d3ca46` (CP-072)
Most recent verified code CI: `34658384988` PASS
Current branch checkpoint: `229368ba6bde81f449ed1c0e91deb0c4a9d3ca46` (CP-072 candidate)
Current branch CI: `34658384988` PASS

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
- Plaintext admin/API/Bot/webhook/MTProto secrets are never logged or persisted by
  Control Plane.
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
  `ca9fcf4f605cfb10eef2db8766f14df8d3b5443b`, CI `34657708764` PASS.
- CP-071 promotion docs:
  `63fc78dee63f02265b76dcf3c9522fa8b1c11a49`, CI `34657897001` PASS.
- Stage 12H scope docs:
  `935c9bb3ce2824a5b4dcb37e5aa1cb9bbfaafd48`, CI `34658114344` PASS.
- CP-072 `tproxy doctor` Control Docker-health gate:
  `229368ba6bde81f449ed1c0e91deb0c4a9d3ca46`, CI `34658384988` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and installer/
  Telemt E2E install/rerun.

## Explicit blockers

### Stage 11H — Bot Content runtime delivery wiring — BLOCKED
Runtime composition/fallback, button/emoji relationship, and missing-slot behavior are
not defined. Do not invent them.

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
Telemt topology, or secret persistence.

## Remaining CI roadmap — REVIEWED / NOT SCOPED

Current CI explicitly covers gofmt, go vet, full Go tests, SQLite migration tests,
ShellCheck, Docker build, Compose config validation, and installer/Telemt E2E
install/rerun. Do not invent a targeted-integration selector, secret scanner contract,
dependency scanner/version policy, target-distro harness, or upgrade/rollback smoke
before those contracts exist.

## Stage 12F — Control image Docker readiness healthcheck — COMPLETED AT CP-070

CP-070 established a first-class Control Docker health contract using the existing
unauthenticated `/readyz` database-readiness endpoint and the built-in
`teleproxy-control healthcheck <http-url>` process mode. Compose restart policy remains
independent for Control and Telemt.

## Stage 12G — Installer gates success on Control Docker health — COMPLETED AT CP-071

After direct Control `/readyz` succeeds, the host installer waits boundedly for the
Control container Docker health status to become exactly `healthy` before the independent
Telemt health gate and before installation success. The final scoped diff is exactly
`scripts/install-host.sh` (+24 lines).

## Stage 12H — `tproxy doctor` requires Control Docker health — COMPLETED AT CP-072

CP-072 extends the established installation-health view without adding a new protocol.
`tproxy doctor` now resolves the Control container and reads only its existing Docker
`.State.Health.Status`, printing the stable line `Control Docker health: <status>` and
requiring exactly `healthy` for success. Missing container/health, non-healthy status, or
inspect failure makes doctor fail. Existing Compose validation, both-container running
checks, direct Control `/readyz`, and Telemt Docker-health checks remain unchanged.

Installer E2E now asserts `Control Docker health: healthy` through the existing doctor
path after both first install and rerun. No Docker health logs, URLs, tokens, credentials
or secrets are printed by the new path.

The final scoped code diff contains exactly:
- `bin/tproxy` (+13 lines)
- `tests/installer_e2e.sh` (+1 assertion)

No restart/watchdog action, restart-loop state, notification, Compose `depends_on`,
schema/migration/API/topology, secret-persistence, or Control/Telemt lifecycle-coupling
change was added.

Stage 12H scope `935c9bb3ce2824a5b4dcb37e5aa1cb9bbfaafd48` passed CI
`34658114344`; CP-072 `229368ba6bde81f449ed1c0e91deb0c4a9d3ca46` passed CI
`34658384988` across every established gate.

## Blocked items preserved

The remaining product/reliability work still lacks complete contracts:
- watchdog repeated-failure/restart-loop threshold, durable degraded-state semantics and
  admin-notification transport;
- Bot Content composition/fallback/missing-slot semantics;
- per-Node health/test credential/runtime endpoint contract;
- referral reward recipient and anti-abuse defaults;
- Node/Sponsor routing, RBAC enforcement, backup/restore, update/rollback/version source,
  Dashboard metrics, secret persistence, and new Telemt topology;
- remaining CI scanner/matrix/upgrade tooling and acceptance contracts.

## Current next action

Require full CI PASS on this CP-072 promotion docs commit. Then re-check the roadmap and
repository for another independent milestone whose semantics are already established.
If none remains, record a recovery-safe blocked state rather than invent product/runtime
contracts. Preserve every architecture invariant, explicit blocker and lifecycle
boundary above.
