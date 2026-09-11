# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `f038c8b266c66b9378d26547c7c4ab4a68e45de6` (CP-060)
Most recent verified code CI: `34613204935` PASS

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

- CP-057 Web Panel established proxy lifecycle actions:
  `c8ba6e27de8153e6907e53e3eb813026728d0203`, CI `34596564418` PASS.
- CP-058 Web Panel manual quota reconciliation:
  `cbb4ac3a22a18332a74972270ae46a74c381f80d`, CI `34601789910` PASS.
- CP-058 promotion docs: `40ce8e433e3f296be210cccfb154d8705c73d323`,
  CI `34602626633` PASS.
- Stage 11R scope docs: `deb5bda3de2e49bcbbb835e4a088d69934b44274`,
  CI `34606209209` PASS.
- CP-059 Web Panel authoritative User exact filters:
  `c702bec933eed9a86676a36149a246526ffd1753`, CI `34607269536` PASS.
- CP-059 promotion docs:
  `49f9b302b9ad911ccfb503040611d2bd8ea4f0aa`, CI `34608286970` PASS.
- Stage 11S scope docs:
  `6660951628ac191b46199930899b29a0fddb5ce6`, CI `34612807650` PASS.
- CP-060 Settings established referral reward configuration surface:
  `f038c8b266c66b9378d26547c7c4ab4a68e45de6`, CI `34613204935` PASS.

CP-059 extends only existing authenticated `GET /api/users` and `GET /users` with
optional exact `telegram_id` and `proxy_username` filters over authoritative SQLite.
The filters compose with logical AND. Malformed, duplicate, non-positive/non-integer
Telegram IDs, invalid proxy usernames, and unknown query keys use the existing
`USER_INVENTORY_INVALID` Problem contract. Matching is exact and case-sensitive;
there is no trimming, fuzzy/substring search, case folding, alternate identity
resolution, migration, endpoint, assignment/routing, reward semantic, or Telemt
topology change.

## Explicit blockers

### Stage 11H — Bot Content runtime delivery wiring — BLOCKED

Roadmap content slots exist, but runtime composition/fallback, button/emoji
relationship, and missing-slot behavior are not defined. Do not invent them.

### Stage 9D — Proxy Node test/health/status — BLOCKED

Per-Node credential source and `internal_api_endpoint` runtime meaning are not
established. Do not probe Nodes or invent credential storage/transport.

### Stage 7D2C — Referral reward issuance — BLOCKED

Reward recipient is unresolved: inviter, invitee, or both. Do not infer a recipient.

### Stage 7D3B — Referral anti-abuse — BLOCKED

Cap scope/defaults, cooldown semantics/default, blacklist subject, and suspicious
score inputs/threshold are unresolved. Do not invent them.

Also do not invent Node/Sponsor assignment/routing, audit mutation/redaction wiring,
Administrator RBAC enforcement, backup/restore, update/restart/log/version-source
semantics, Dashboard metrics, referral-tree/filter semantics, new Telemt topology,
or secret persistence.

## Stage 11S — Settings established referral reward configuration surface — COMPLETED AT CP-060

CP-060 extends only authenticated `GET /settings` with the already-established
referral reward amount and expiry configuration. The page reads
`settings.ReferralReward` from authoritative SQLite and renders referral reward bytes
and expiry days alongside the existing Start Gift surface.

The browser reuses the existing CSRF-protected
`PUT /api/referral/reward-settings` contract. Positive int64 values are kept as exact
decimal text in the browser, including values above JavaScript's safe integer range;
the implementation does not use JavaScript `Number`, `parseInt`, or `parseFloat`.
A successful update reloads `/settings`, so displayed values are read back from
authoritative SQLite. GET remains `Cache-Control: no-store` and mutation-free.

The candidate changes exactly
`internal/httpapi/start_gift_settings_page.go` and
`internal/httpapi/start_gift_settings_page_test.go`. Existing Start Gift behavior and
the `/referrals` settings/history surface remain intact. No endpoint, persistence key,
migration, Credit Bucket issuance, referral recipient/eligibility/finalization or
anti-abuse behavior, assignment/routing, audit wiring, Bot runtime behavior, per-Node
credential/health behavior, Telemt topology, or secret persistence was added.

Candidate `f038c8b266c66b9378d26547c7c4ab4a68e45de6`, CI `34613204935` PASS across
Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and
Telemt E2E/rerun.

## Current next action

Promote CP-060 documentation only and require full CI PASS. Then inspect the roadmap
and current repository contracts for the next semantics-established bounded milestone.
Preserve Credit Buckets as source of truth and every blocker above; do not invent
missing product/runtime semantics.
