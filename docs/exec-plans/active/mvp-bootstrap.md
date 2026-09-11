# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `8b57acc7e68fa0b9f2678c2735a7d3804325a392` (CP-069)
Most recent verified code CI: `34649134519` PASS
Current branch checkpoint: `8b57acc7e68fa0b9f2678c2735a7d3804325a392` (CP-069 candidate / forward repair)
Current branch CI: `34649134519` PASS

Historical execution detail is preserved without deletion:
- through CP-059 at `docs/exec-plans/archive/mvp-bootstrap-through-cp059.md`;
- CP-060 through the fully scoped Stage 11W pre-implementation state at
  `docs/exec-plans/archive/mvp-bootstrap-through-cp068.md`, using the prior active-plan
  blob `546cd16b5985af5518fc7459cb5a159c14d85e34` byte-for-byte.

## Recovery contract

Resume from `Latest verified code checkpoint`, then inspect every later branch commit,
file diff, and CI result before writing. Repair an active partial milestone first.
Never reset, clean, force-push, or overwrite unrelated work.

If interrupted during connector-side staging before an atomic branch move, re-read
the current branch HEAD and this plan, then reconstruct the last intended complete
file replacement from current HEAD plus the verified intended diff. Unreachable
blob/tree/commit staging objects are not branch state and are never proof that a file
was published.

A failed candidate remains historical evidence. Repair forward only. A checkpoint is
verified only after every established CI gate passes on the exact code SHA.

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
  `f2378e70dc4028fa40f9d1bd2a5c540a1f67bac0`, CI `34647334507` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and Telemt
  E2E/rerun.
- CP-068 promotion docs:
  `73d518047cb5365c13b0ea116e5d08068099c2bf`, CI `34647743764` PASS.
- Stage 11W scope docs:
  `9a77d08293f25685dbf8e8f351fafe355cfd1e28`, CI `34648117631` PASS.
- Stage 11W first candidate:
  `711e70016aa4603f2ffb023b62f57012cddd248e`, CI `34648993231` FAILED at Vet.
  Self-review found the page regression test typo `countHTPRows` instead of the existing
  `countHTTPRows`; SQLite migration tests, full Go tests, and installer were skipped by
  that failed run. The failure was repaired forward without reset or force-push.
- CP-069 User inventory exact provisioning phase filter:
  `8b57acc7e68fa0b9f2678c2735a7d3804325a392`, CI `34649134519` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and Telemt
  E2E/rerun.

The CP-069 forward repair restored `internal/httpapi/users_page_test.go` byte-for-byte
and moved the new Web Panel filter regression coverage into the dedicated
`internal/httpapi/users_page_provisioning_phase_filter_test.go`. Runtime semantics did
not change in the repair.

## Explicit blockers

### Stage 11H — Bot Content runtime delivery wiring — BLOCKED

Runtime composition/fallback, button/emoji relationship, and missing-slot behavior
are not defined. Do not invent them.

### Stage 9D — Proxy Node test/health/status — BLOCKED

Per-Node credential source and `internal_api_endpoint` runtime meaning are not
established. Do not probe Nodes or invent credential storage/transport.

### Stage 7D2C — Referral reward issuance — BLOCKED

Reward recipient is unresolved: inviter, invitee, or both. Do not infer a recipient.

### Stage 7D3B — Referral anti-abuse — BLOCKED

Cap scope/defaults, cooldown semantics/default, blacklist subject, and suspicious
score inputs/threshold are unresolved. Do not invent them.

Also do not invent Node/Sponsor assignment/routing, user-scoped referral-tree/history
semantics, audit mutation/redaction wiring, Administrator RBAC enforcement,
backup/restore, update/restart/log/version-source semantics, Dashboard metrics, new
Telemt topology, or secret persistence.

## Remaining CI roadmap — REVIEWED / NOT SCOPED

Current CI explicitly covers gofmt, go vet, full Go tests, SQLite migration tests,
ShellCheck, Docker build, Compose config validation, and installer/Telemt E2E
install/rerun.

Do not scope the remaining roadmap CI items until their repository contracts exist:
- targeted integration tests have no separate tag/suite/harness;
- secret scanning has no selected scanner, ruleset, suppression policy, or pinned
  version contract;
- dependency vulnerability enforcement has no selected scanner/configuration/version
  policy or concrete threshold;
- target-distro smoke has no pinned Ubuntu/Debian matrix or target-OS harness;
- upgrade/rollback smoke depends on unresolved update/rollback/version-source semantics.

## Stage 11W — User inventory exact provisioning phase filter — COMPLETED AT CP-069

CP-069 extends only the existing authenticated `GET /api/users` and `/users` inventory
reads with an optional exact `provisioning_phase` filter over the already-authoritative
stored `proxy_user_provisioning.phase` field.

Accepted values are exactly the established `prepared`, `owned`, and `collision`
`proxyprovision.Phase` constants. The domain query validates the phase and applies exact
`pp.phase = ?` alongside optional Telegram ID, proxy username, `before_id`, and bounded
limit semantics using logical AND. Rows without a provisioning record naturally do not
match an active phase filter. No synthetic `not_provisioned` query value and no generic
secret-status category were added.

The HTTP parser accepts exactly one non-empty `provisioning_phase` value. Empty,
duplicate, unknown, or differently-cased values return the existing
`USER_INVENTORY_INVALID` Problem contract. The Web Panel exposes
All/prepared/owned/collision; All omits the query parameter. Active phase, existing
Telegram/proxy filters, and explicit limit survive Older users pagination.

The final code diff from Stage 11W scope contains exactly:
- `internal/useradmin/list.go`
- `internal/useradmin/list_test.go`
- `internal/httpapi/users.go`
- `internal/httpapi/users_test.go`
- `internal/httpapi/users_page.go`
- `internal/httpapi/users_page_provisioning_phase_filter_test.go`

The scoped intent originally named `internal/httpapi/users_page_test.go` for the new page
coverage. The first candidate edited that file and introduced a test-only typo. The
forward repair restored the file exactly and placed the Stage 11W page coverage in the
dedicated test file above. This is a test-file placement correction only; the product
scope and runtime diff remain the exact read-only filter described here.

Regression coverage verifies exact prepared/collision filtering, an empty owned result,
composition with existing filters, exclusion of absent provisioning rows, invalid-query
rejection, filter-preserving pagination, selector state, no secret digest exposure, and
no mutation of authoritative tables.

No secret digest or plaintext secret is selected, serialized, rendered, logged, or
otherwise exposed. No migration, write path, provisioning transition, Telemt request,
secret rotation semantic, Node/Sponsor assignment/routing, audit wiring, Bot behavior,
RBAC enforcement, backup/update/restart/log behavior, Dashboard metric, or lifecycle
boundary changed.

Stage 11W scope docs `9a77d08293f25685dbf8e8f351fafe355cfd1e28`
passed CI `34648117631`. First candidate `711e70016aa4603f2ffb023b62f57012cddd248e`
failed CI `34648993231` at Vet because of the test typo described above. Forward repair
and final CP-069 checkpoint `8b57acc7e68fa0b9f2678c2735a7d3804325a392`
passed CI `34649134519` across every established gate.

## Current next action

Promote CP-069 documentation/archive only and require full CI PASS on the promotion
commit. Then inspect the remaining roadmap and current repository contracts for another
independent milestone whose semantics are already established. If no such milestone
exists, record that recovery-safe state rather than inventing product/runtime semantics.
Preserve every explicit blocker, Control/Proxy lifecycle separation, and Credit Buckets
as authoritative quota/reward state.
