# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `8993aee8b9d980990504fe25261eb4c29879316e` (CP-061)
Most recent verified code CI: `34615741067` PASS

Historical execution detail through CP-059 is preserved byte-for-byte at
`docs/exec-plans/archive/mvp-bootstrap-through-cp059.md`, using the prior active-plan
blob `7ee43ba2a7373fd4e359fba08e9e7c5b47ff4817`.

## Recovery contract

Resume from `Latest verified code checkpoint`, then inspect every later branch commit,
file diff, and CI result before writing. Repair an active partial milestone first.
Never reset, clean, force-push, or overwrite unrelated work.

If interrupted during connector-side staging before an atomic branch move, re-read
the current branch HEAD and this plan, then reconstruct the last intended complete
file replacement from current HEAD plus the verified intended diff. Unreachable
blob/tree/commit staging objects are not branch state and are never proof that a file
was published.

## Non-negotiable architecture

- SQLite WAL/NORMAL is authoritative Control Plane state.
- Credit Buckets are authoritative quota/reward state.
- Telemt quota/expiry is only an enforcement projection.
- Control Plane and Telemt lifecycles remain independent.
- Plaintext admin/API/Bot/webhook/MTProto secrets are never logged or persisted by
  Control Plane.
- Migrations remain additive/backward-compatible.

## Current verified checkpoints

- CP-059 Web Panel authoritative User exact filters:
  `c702bec933eed9a86676a36149a246526ffd1753`, CI `34607269536` PASS.
- CP-059 promotion docs:
  `49f9b302b9ad911ccfb503040611d2bd8ea4f0aa`, CI `34608286970` PASS.
- Stage 11S scope docs:
  `6660951628ac191b46199930899b29a0fddb5ce6`, CI `34612807650` PASS.
- CP-060 Settings established referral reward configuration surface:
  `f038c8b266c66b9378d26547c7c4ab4a68e45de6`, CI `34613204935` PASS.
- CP-060 promotion docs:
  `e32525cfdccf64d0bff9b87e489d3fdace2fa7ed`, CI `34613627486` PASS.
- Stage 11T scope docs:
  `ad308e646c3f2a2bf1c3d7e3d378f4ca1e601b88`, CI `34614212190` PASS.
- CP-061 Referral history authoritative exact status filter:
  `8993aee8b9d980990504fe25261eb4c29879316e`, CI `34615741067` PASS.

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

## Stage 11S — Settings established referral reward configuration surface — COMPLETED AT CP-060

CP-060 extends authenticated `GET /settings` with the already-established referral
reward bytes and expiry days and reuses the existing CSRF-protected
`PUT /api/referral/reward-settings` contract. Browser handling preserves exact
positive int64 decimal text and GET remains no-store/read-only. No reward
recipient/issuance/anti-abuse semantics were added.

## Stage 11T — Referral history authoritative exact status filter — COMPLETED AT CP-061

CP-061 extends only existing authenticated `GET /api/referral/history` and
`GET /referrals` reads with one optional exact `status` filter over authoritative
`referral_attributions.status`.

Accepted values are exactly the established constants `pending`, `rewarded`, and
`rejected`. The domain read model validates the filter, applies exact
`ra.status = ?`, composes it with optional `before_id` using logical AND, and
preserves newest-first bounded pagination.

The HTTP parser rejects empty, duplicate, unknown, differently-cased, or otherwise
invalid status values with the existing `REFERRAL_HISTORY_INVALID` Problem contract.
The `/referrals` page offers only All/Pending/Rewarded/Rejected. Selecting All omits
the status query parameter; active status and explicit limit are retained in Older
referrals pagination.

The candidate changes exactly:
- `internal/referral/history.go`
- `internal/httpapi/referral_history.go`
- `internal/httpapi/referral_page.go`
- `internal/referral/history_status_filter_test.go`
- `internal/httpapi/referral_history_status_filter_test.go`
- `internal/httpapi/referral_page_status_filter_test.go`

No endpoint, migration, write path, rejection-reason filter, suspicious-referral
classification, anti-abuse policy, reward recipient/issuance,
eligibility/finalization mutation, referral tree or user-scoped filtering,
assignment/routing, audit wiring, Bot runtime behavior, per-Node credentials/health,
Telemt topology, or secret persistence was added.

Candidate `8993aee8b9d980990504fe25261eb4c29879316e`, CI `34615741067` PASS across
Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and
Telemt E2E/rerun.

## Current next action

Promote CP-061 documentation only and require full CI PASS. Then inspect the roadmap
and current repository contracts for the next semantics-established bounded milestone.
Preserve Credit Buckets as source of truth and every blocker above; do not invent
missing product/runtime semantics.
