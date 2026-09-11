# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `fa88c94b6ec654035a7fa086319542322ad5003e` (CP-070)
Most recent verified code CI: `34650289404` PASS
Current branch checkpoint: `fa88c94b6ec654035a7fa086319542322ad5003e` (CP-070 candidate)
Current branch CI: `34650289404` PASS

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
published branch state. Failed candidates remain historical evidence and are repaired
forward only.

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
  `fa88c94b6ec654035a7fa086319542322ad5003e`, CI `34650289404` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and installer/
  Telemt E2E/rerun including the Control Docker healthy assertion after first install and
  rerun.

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

CP-070 closes the roadmap's established Docker-healthcheck gap for the Control image
without adding a shell or HTTP utility to the runtime image. The Control binary now has
a local process mode:
`teleproxy-control healthcheck <http-url>`.

That mode performs one HTTP GET with a 3-second client timeout, refuses non-HTTP URLs
and URL userinfo, does not follow redirects, accepts only HTTP 200, drains at most 4096
response bytes, never prints a response body or requested URL, and returns non-zero for
transport, timeout, redirect, malformed command, or non-200 failure. With no command
arguments, normal Control Plane startup is unchanged.

The Control Docker image now defines:
`HEALTHCHECK ... CMD ["/usr/local/bin/teleproxy-control", "healthcheck", "http://127.0.0.1:8080/readyz"]`.
It reuses the existing unauthenticated `/readyz` database-readiness contract and adds no
curl, wget, shell, token, secret, external request, package, or network exposure.

Installer E2E resolves the installed Control container ID and waits boundedly for Docker
health status `healthy` after first install and rerun. On failure it reports only final
container/health status, not Docker health logs. Existing direct `/readyz`, `tproxy
doctor`, and Proxy Plane checks remain separate and intact.

The final scoped code diff contains exactly:
- `cmd/control/main.go`
- `cmd/control/healthcheck.go`
- `cmd/control/healthcheck_test.go`
- `Dockerfile`
- `tests/installer_e2e.sh`

No watchdog restart action, restart-loop detection, admin notification, update/rollback,
backup, Node health, Telemt image/topology, migration/schema, product endpoint, or
cross-plane lifecycle coupling was added.

Stage 12F scope docs `0ec8a0ca48fe1b5254e978f5fc11b4f6da16d185`
passed CI `34649791579`. Candidate and CP-070 checkpoint
`fa88c94b6ec654035a7fa086319542322ad5003e` passed CI `34650289404` across every
established gate, including the new installed-Control Docker health assertion.

## Current next action

Promote CP-070 documentation only and require full CI PASS on the promotion commit.
Then re-check the roadmap and repository for another independent milestone whose
semantics are already established. If none remains, record a recovery-safe blocked state
rather than invent watchdog thresholds/notifications or other missing product/runtime
contracts. Preserve every explicit blocker and lifecycle boundary.
