# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `046d487f38ebbbe30ba8dd753ce728f781042b04` (CP-062)
Most recent verified code CI: `34620300934` PASS
Current branch checkpoint: `046d487f38ebbbe30ba8dd753ce728f781042b04` (CP-062 candidate)
Current branch CI: `34620300934` PASS

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

- CP-060 Settings established referral reward configuration surface:
  `f038c8b266c66b9378d26547c7c4ab4a68e45de6`, CI `34613204935` PASS.
- CP-060 promotion docs:
  `e32525cfdccf64d0bff9b87e489d3fdace2fa7ed`, CI `34613627486` PASS.
- Stage 11T scope docs:
  `ad308e646c3f2a2bf1c3d7e3d378f4ca1e601b88`, CI `34614212190` PASS.
- CP-061 Referral history authoritative exact status filter:
  `8993aee8b9d980990504fe25261eb4c29879316e`, CI `34615741067` PASS.
- CP-061 promotion docs:
  `a440758caf316d41d50e9f9bb8ce6b70d93ef42e`, CI `34616097813` PASS.
- Stage 11U scope docs:
  `08776242d9ec8f83114082d405037a259bfb46cf`, CI `34616505248` PASS.
- Stage 11U first candidate:
  `8bdbdcb3adefc95fdb4000c47fbbebb6560846e8`, CI `34620149594` FAILED at Vet because
  `validRejectionReason` duplicated the already-established package helper in
  `internal/referral/eligibility.go`; Test and installer were skipped. This failure was
  repaired forward without reset or force-push.
- CP-062 Referral history authoritative exact rejection-reason filter:
  `046d487f38ebbbe30ba8dd753ce728f781042b04`, CI `34620300934` PASS across Format,
  Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and Telemt
  E2E/rerun.

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

## Stage 11T — Referral history authoritative exact status filter — COMPLETED AT CP-061

CP-061 extends only existing authenticated referral-history reads with an optional
exact `status` filter accepting the already-established `pending`, `rewarded`, and
`rejected` constants. It preserves no-store/read-only behavior and bounded newest-first
pagination. No reward or anti-abuse write semantics were added.

## Stage 11U — Referral history authoritative exact rejection-reason filter — COMPLETED AT CP-062

CP-062 extends the existing authenticated `GET /api/referral/history` and
`GET /referrals` read contracts with one optional exact `rejection_reason` filter over
the authoritative stored `referral_attributions.rejection_reason` field.

Accepted values are exactly the six established `referral.RejectionReason` constants:
`anti_abuse`, `daily_cap`, `weekly_cap`, `cooldown`, `blacklist`, and `suspicious`.
The domain read model reuses the existing package `validRejectionReason` validator,
applies exact `ra.rejection_reason = ?`, and composes independently with optional
`status` and `before_id` using logical AND. It does not infer `status=rejected`;
contradictory filters return an empty result.

The HTTP parser rejects empty, duplicate, unknown, differently-cased, or otherwise
invalid reason values with the existing `REFERRAL_HISTORY_INVALID` Problem contract.
The `/referrals` page exposes All plus only the six established reasons; All omits the
query parameter, while active status/reason and explicit limit survive Older referrals
pagination.

The final scoped diff contains exactly:
- `internal/referral/history.go`
- `internal/httpapi/referral_history.go`
- `internal/httpapi/referral_page.go`
- `internal/referral/history_rejection_reason_filter_test.go`
- `internal/httpapi/referral_history_rejection_reason_filter_test.go`
- `internal/httpapi/referral_page_rejection_reason_filter_test.go`

No endpoint, migration, write path, anti-abuse policy, suspicious scoring, reward
recipient/issuance, eligibility/finalization mutation, referral tree, inviter/invitee
user-scoped filtering, assignment/routing, audit wiring, Bot behavior, per-Node
credentials/health, Telemt topology, or secret persistence was added. GET remains
no-store/read-only and regression coverage verifies exact filtering, independent
status composition, bounded pagination, invalid-query rejection, and no authoritative
state mutation.

## Current next action

Promote CP-062 documentation only and require full CI PASS. Then inspect the roadmap
and current repository contracts for the next semantics-established bounded milestone.
Preserve Credit Buckets as source of truth and every blocker above; do not invent
missing product/runtime semantics.
