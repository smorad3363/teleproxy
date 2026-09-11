# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `8b57acc7e68fa0b9f2678c2735a7d3804325a392` (CP-069)
Most recent verified code CI: `34649134519` PASS
Current branch checkpoint: `a41dd6232c01e6ed738d3496d525ad55f23f01dd` (CP-069 promotion docs)
Current branch CI: `34649427482` PASS

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
  `8b57acc7e68fa0b9f2678c2735a7d3804325a392`, CI `34649134519` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and Telemt
  E2E/rerun.
- CP-069 promotion docs/archive compaction:
  `a41dd6232c01e6ed738d3496d525ad55f23f01dd`, CI `34649427482` PASS across every
  established gate. The prior active plan was preserved byte-for-byte in the CP-068
  archive before compaction.

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

The final code diff from Stage 11W scope contains exactly:
- `internal/useradmin/list.go`
- `internal/useradmin/list_test.go`
- `internal/httpapi/users.go`
- `internal/httpapi/users_test.go`
- `internal/httpapi/users_page.go`
- `internal/httpapi/users_page_provisioning_phase_filter_test.go`

The forward repair restored `internal/httpapi/users_page_test.go` exactly and moved the
new page-filter coverage to the dedicated test file above. No runtime scope changed.

## Stage 12F — Control image Docker readiness healthcheck — SCOPED

The roadmap explicitly requires Docker healthchecks for recovery. The current Compose
services already use `restart: unless-stopped`; the Telemt image already has a native
Docker `HEALTHCHECK`. The Control Plane already exposes unauthenticated `GET /readyz`,
which performs a bounded database ping and returns non-2xx when the authoritative SQLite
state is not ready. The Control image currently has no Docker `HEALTHCHECK` and includes
no curl/wget runtime dependency.

Stage 12F closes only that established gap. Intended code diff is exactly:
- `cmd/control/main.go`
- `cmd/control/healthcheck.go`
- `cmd/control/healthcheck_test.go`
- `Dockerfile`
- `tests/installer_e2e.sh`

The Control binary will gain one local process mode:
`teleproxy-control healthcheck <http-url>`. It performs one bounded HTTP GET with a
3-second client timeout, does not follow redirects, accepts only HTTP 200, closes a
small bounded response body without printing it, and returns non-zero on any transport,
timeout, redirect, or non-200 result. The normal server mode remains unchanged when no
`healthcheck` command is supplied. Unknown command/argument shapes fail non-zero.

The production Control image will add a Docker `HEALTHCHECK` that invokes exactly:
`/usr/local/bin/teleproxy-control healthcheck http://127.0.0.1:8080/readyz`.
No shell, curl, wget, package, secret, token, external endpoint, or new network exposure
is added. The healthcheck is Control readiness only; it does not probe Telemt, Bot,
Proxy Nodes, Sponsors, or other product state and does not couple their lifecycles.

Installer E2E will require the installed Control container to reach Docker health status
`healthy` after initial install and after rerun, using a bounded wait and diagnostic that
prints only the final health status/container state rather than health-log output.
Existing `tproxy doctor` readiness and Proxy Plane checks remain intact and separate.

No watchdog, automatic restart action beyond the already-existing Docker restart policy,
restart-loop detection, admin notification, update/rollback, backup, Node health,
Telemt image/topology, endpoint semantics, migration, database schema, or product feature
is in scope.

Acceptance requires this scope-doc CI to PASS before code change. The candidate must
then PASS Format, Vet, explicit SQLite migration tests, full Go tests, ShellCheck,
installer syntax/unit tests, Docker prerequisites, Docker build, Compose config
validation, and Telemt E2E/rerun including the new Control healthy assertion before a
new verified code checkpoint is promoted.

## Current next action

Require PASS for the Stage 12F scope-doc CI. Then implement exactly the five-file
Control image readiness-healthcheck scope above, self-review the complete net diff,
require full candidate CI PASS, promote the checkpoint, require promotion CI PASS, and
only then re-check the roadmap. Do not expand Stage 12F into watchdog/restart/update or
cross-plane health semantics.
