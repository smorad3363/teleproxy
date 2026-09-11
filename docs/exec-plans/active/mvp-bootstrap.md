# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `ca9fcf4f605cfb10eef2db8766f14df8d3b5443b` (CP-071)
Most recent verified code CI: `34657708764` PASS
Current branch checkpoint: `ca9fcf4f605cfb10eef2db8766f14df8d3b5443b` (CP-071 candidate)
Current branch CI: `34657708764` PASS

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

- CP-068 User inventory authoritative provisioning phase:
  `f2378e70dc4028fa40f9d1bd2a5c540a1f67bac0`, CI `34647334507` PASS.
- CP-068 promotion docs:
  `73d518047cb5365c13b0ea116e5d08068099c2bf`, CI `34647743764` PASS.
- Stage 11W scope docs:
  `9a77d08293f25685dbf8e8f351fafe355cfd1e28`, CI `34648117631` PASS.
- Stage 11W first candidate:
  `711e70016aa4603f2ffb023b62f57012cddd248e`, CI `34648993231` FAILED at Vet due
  the test-only `countHTPRows` typo; later gates were skipped and the branch was repaired
  forward without reset or force-push.
- CP-069 User inventory exact provisioning phase filter:
  `8b57acc7e68fa0b9f2678c2735a7d3804325a392`, CI `34649134519` PASS across every
  established gate.
- CP-069 promotion docs/archive compaction:
  `a41dd6232c01e6ed738d3496d525ad55f23f01dd`, CI `34649427482` PASS across every
  established gate.
- Stage 12F scope docs:
  `0ec8a0ca48fe1b5254e978f5fc11b4f6da16d185`, CI `34649791579` PASS.
- CP-070 Control image Docker readiness healthcheck:
  `fa88c94b6ec654035a7fa086319542322ad5003e`, CI `34650289404` PASS across every
  established gate.
- CP-070 promotion docs:
  `59e34aa2d565ecf2cb7cae7a7f5fd286a4457f44`, CI `34650530913` PASS.
- Post-CP-070 recovery docs:
  `ff8fde54441bc630ca964fd2dce493f8c3486ee2`, CI `34653782213` PASS.
- Stage 12G scope docs:
  `488bfd52ee7dd4f3c3f6dd326f946eae66ff250e`, CI `34657412890` PASS.
- Stage 12G first candidate:
  `a304b7758566284b2d9fdcbe7462b945dfed6e1d`, CI `34657664813` PASS, but self-review
  found one accidental out-of-scope reconstruction regression in persisted proxy-bind
  assignment. The branch was repaired forward without reset or force-push.
- CP-071 Installer gates success on Control Docker health:
  `ca9fcf4f605cfb10eef2db8766f14df8d3b5443b`, CI `34657708764` PASS across Format,
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

## Stage 11W — User inventory exact provisioning phase filter — COMPLETED AT CP-069

CP-069 adds only an optional exact `provisioning_phase` filter to authenticated
`GET /api/users` and `/users`, accepting exactly the established `prepared`, `owned`,
and `collision` values. It composes by logical AND with existing filters and pagination,
excludes absent provisioning rows naturally, preserves no-store/read-only behavior, and
adds no synthetic `not_provisioned` query token or generic secret-status inference.

The final code diff contains exactly:
- `internal/useradmin/list.go`
- `internal/useradmin/list_test.go`
- `internal/httpapi/users.go`
- `internal/httpapi/users_test.go`
- `internal/httpapi/users_page.go`
- `internal/httpapi/users_page_provisioning_phase_filter_test.go`

## Stage 12F — Control image Docker readiness healthcheck — COMPLETED AT CP-070

CP-070 established the Control image's first-class Docker health contract using the
existing unauthenticated `/readyz` database-readiness endpoint and a built-in
`teleproxy-control healthcheck <http-url>` process mode. Installer E2E verifies the
installed Control container becomes Docker `healthy` after first install and rerun.

No watchdog action, restart-loop detection, admin notification, update/rollback, backup,
Node health, Telemt image/topology, migration/schema, product endpoint, or cross-plane
lifecycle coupling was added.

## Stage 12G — Installer gates success on Control Docker health — COMPLETED AT CP-071

CP-071 closes the remaining installer-side gap in the roadmap's Docker-healthcheck
requirement. After the existing direct Control `/readyz` check succeeds, the host
installer now resolves the Control container and waits up to the existing bounded
60-attempt window for Docker health status to become exactly `healthy` before continuing
to the independently existing Telemt health gate and before reporting a successful
installation.

If the Control container is missing, has no health status, remains non-healthy, or times
out, the installer fails while state is still `prepared`. The failure path prints only the
final container/health status and ordinary `compose ps`; it does not print Docker health
logs, requested URLs, tokens, or secrets. The existing direct `/readyz` failure behavior
and the existing Telemt health gate are unchanged.

The final scoped code diff relative to Stage 12G scope contains exactly:
- `scripts/install-host.sh` (+24 lines, no unrelated final diff).

No Compose `depends_on`, schema/migration, product/API, port/topology, credential,
secret-persistence, watchdog, update/rollback, backup, Node health, or Control/Telemt
lifecycle-coupling change was added.

Stage 12G scope docs `488bfd52ee7dd4f3c3f6dd326f946eae66ff250e` passed CI
`34657412890`. The first code candidate `a304b7758566284b2d9fdcbe7462b945dfed6e1d`
passed CI `34657664813`, but self-review found an accidental persisted-proxy-bind
assignment regression introduced while reconstructing the full shell file. Forward repair
`ca9fcf4f605cfb10eef2db8766f14df8d3b5443b` restores that line, leaves only the intended
24-line Control-health gate diff, and passed CI `34657708764` across every established
gate.

## Post-CP-071 roadmap review — BLOCKED ITEMS PRESERVED

The remaining product/reliability work below is not safely implementable from current
repository contracts without inventing behavior:
- watchdog behavior still lacks a concrete repeated-failure/restart-loop threshold,
  durable degraded-state semantics, and admin-notification transport/contract;
- Bot Content runtime delivery still lacks composition/fallback/missing-slot semantics;
- per-Node health/test still lacks its credential/runtime endpoint contract;
- referral reward issuance still lacks the reward recipient contract;
- remaining referral anti-abuse still lacks cap/cooldown/blacklist/suspicious defaults;
- Node/Sponsor routing, RBAC enforcement, backup/restore, update/rollback/version source,
  Dashboard metrics, secret persistence, and new Telemt topology remain explicitly
  unresolved;
- remaining CI recommendations lack repository-selected tooling/acceptance contracts as
  recorded above.

CP-071 completes the second independently contract-defined recovery/health primitive
found in the roadmap review: installer success is now gated on the Control Docker health
contract established by CP-070.

## Current next action

Require full CI PASS on this CP-071 promotion docs commit. Then re-check the roadmap and
repository for another independent milestone whose semantics are already established.
If none remains, record a recovery-safe blocked state rather than invent product/runtime
contracts. Preserve every architecture invariant, explicit blocker, and lifecycle
boundary above.
