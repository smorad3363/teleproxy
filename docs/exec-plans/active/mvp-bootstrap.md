# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `907d818300f9cdb01473fecacf0cd76bd8db1438`

## Purpose

Build Teleproxy incrementally from an empty repository while keeping every milestone secure, independently verifiable, and recoverable from Git + this file without relying on chat history.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane have independent lifecycles.
- Go + lightweight HTTP + SQLite for Control Plane; Telemt remains an external data-plane component.
- Docker-first installer is rerun-safe, chooses/persists a random free high Panel port, and shows first-install Panel credentials/settings only after health verification.
- Telemt API is authenticated and never host-published; no Docker socket in Web App.
- No plaintext passwords, session/API/MTProto secrets, bot tokens, or private keys in logs/state.
- Proxy-user secrets are reveal-once and are never stored by Control Plane.
- Desired state is persisted before data-plane reconciliation so Telemt outages cannot lose admin intent.
- Roadmap quota business model uses Credit Buckets with independent expiry; do not collapse it into a single business `traffic_limit` field. Telemt quota/expiry values are enforcement projections, not the reward-ledger source of truth.

## Recovery protocol

On interruption:
1. Read this plan and relevant architecture/security/reliability docs.
2. Compare branch head with `Latest verified checkpoint`.
3. Inspect every file/commit after that checkpoint.
4. Re-run/inspect validation for the active partial milestone.
5. Repair the partial milestone before starting another one.
6. Never reset/clean/overwrite unrelated work; no force update unless explicitly authorized.

## Stage policy

Keep milestones small, buildable, and independently verifiable. Promote a checkpoint only after targeted tests and required regression CI pass.

## Completed stages

### Stage 1 — Repository foundation — PARTIAL
Completed execution plan, architecture/security/reliability docs, isolated branch and CI. Remaining exact-byte mirror of supplied root `AGENTS.md`, `ROADMAP_FA.md`, `ROADMAP_EN.md`; do not commit partial mirrors.

Verified source SHA-256, rechecked from mounted originals on 2026-09-10:
- AGENTS: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- ROADMAP_EN: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- ROADMAP_FA: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

### Stage 2 — Minimal Control Plane — COMPLETE
Health/readiness, HTTP timeouts, graceful shutdown, Problem Details and CI. CI `34419759826` PASS.

### Stage 3 — Persistence/admin bootstrap — COMPLETE
Verified `5798075de40d2f966d8546a18e6fa450d7142f90`; CI `34420104043` PASS. SQLite WAL/foreign keys/busy timeout, migrations, PBKDF2-SHA256 admin hashes, hashed session tokens.

### Stage 4 — Secure Web login — COMPLETE
Verified `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`; CI `34420655852` PASS. Protected first-admin bootstrap, DB sessions, CSRF, secure cookie flags, login rate limiting.

### Stage 5 — Recoverable Docker installer — COMPLETE
Verified `457f52783f9b2962c55be00beb801f0f2534958c`; CI `34427021157` PASS. Random persistent Panel port, exclusive install lock/state, one-time admin password, `tproxy`, real Docker install/readiness/rerun E2E.

### Stage 6A — Pinned Telemt data plane — COMPLETE
Verified `47a95349922ba5be37cf0ed8416b482de08580ef`; CI `34438212903` PASS. Telemt 3.5.7 pinned/checksummed, non-root container, public MTProto port, no host API 9091, protected Bearer token/config, disabled internal bootstrap user, E2E isolation/rerun.

Pinned Telemt 3.5.7 artifacts:
- amd64 musl SHA-256 `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl SHA-256 `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`

### Stage 6B — Telemt health client — COMPLETE
Verified `c67874de95b2ca4ff1786ffbd349fb091f270647`; CI `34438961960` PASS. Protected token-file loading, bounded HTTP client, safe health classifications, `/api/system/proxy`; Control Plane readiness remains independent from Telemt.

### Stage 6C1 — Proxy-user persistence + Telemt lifecycle client — COMPLETE
Verified `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`; CI `34445848656` PASS. `proxy_users` desired/sync state, no secret column, typed Telemt create/list/enable/disable/rotate calls, bounded upstream error classification.

### Stage 6C2 — Authenticated proxy-user API + reconciliation — COMPLETE
Verified `24cec6f875fb5f56bfb97d8159d8fa2ac3b8e533`; CI `34446509254` PASS.

Implemented authenticated proxy-user create/list/enable/disable/rotate, DB-first desired state, session-bound CSRF on mutations, safe `TELEMT_*` reconciliation errors, reveal-once secrets, and integration coverage.

### Stage 6D1 — Telemt quota/expiry enforcement contract — COMPLETE
Verified `907d818300f9cdb01473fecacf0cd76bd8db1438`; CI `34447125904` PASS.

Implemented:
- typed Telemt per-user quota/expiry PATCH contract
- exact JSON Merge Patch semantics: omitted = unchanged, `null` = remove, numeric/string value = set
- explicit preservation of quota value `0` instead of treating it as removal
- RFC3339 expiry validation before network calls
- typed quota reset primitive against `/v1/users/{username}/reset-quota`
- bounded upstream response handling and safe failure classification
- focused tests plus existing Docker/Telemt installer E2E regression coverage

## Active stage

### Stage 6D2A — Credit Bucket transactional ledger — ACTIVE

Roadmap contract for each Credit Bucket:
- original amount
- consumed amount
- start timestamp
- expiry timestamp
- reward type
- reward source
- status
- consumption prioritizes buckets with the nearest expiry

Scope only:
- add backward-compatible SQLite migration for credit buckets linked to Control Plane proxy users
- add a focused credit domain/store with grant/list/balance/consume primitives
- make consumption transactional and all-or-nothing when available credit is insufficient
- consume nearest-expiry eligible buckets first, with non-expiring buckets last
- derive effective pending/active/exhausted/expired/revoked state without requiring background jobs for wall-clock expiry
- preserve integer byte accounting; no floating-point traffic math
- no HTTP, bot, referral, or Telemt projection changes in this milestone

Acceptance:
- migration is rerunnable through existing migration framework and preserves existing data
- invalid amount/time/source/type input is rejected before insert
- future-start, expired, revoked, and exhausted buckets are not consumable
- finite expiries are consumed before non-expiring buckets; earlier expiry before later expiry
- insufficient credit leaves every bucket unchanged
- successful multi-bucket consume commits exact byte totals atomically
- Go format/vet/test plus existing Docker/Telemt E2E remain green

### Stage 6D2B — Credit projection/reconciliation to Telemt — PENDING
After ledger verification, project effective remaining bytes/expiry into Telemt and reconcile reset/patch behavior separately. Credit Buckets remain business source of truth.

## Checkpoints

- CP-000 repo initialized: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
- CP-001 minimal Control Plane local: `b98507bbb3ef0a74e6e147286c22ab7d626f9e72`
- CP-002 CI foundation: `084d4a1256a6b28412d9457f6568a4138e09c83c`, CI `34419759826`
- CP-003 persistence/admin: `5798075de40d2f966d8546a18e6fa450d7142f90`, CI `34420104043`
- CP-004 Web auth: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`, CI `34420655852`
- CP-005 Docker installer: `457f52783f9b2962c55be00beb801f0f2534958c`, CI `34427021157`
- CP-006 Telemt data plane: `47a95349922ba5be37cf0ed8416b482de08580ef`, CI `34438212903`
- CP-007 Telemt health client: `c67874de95b2ca4ff1786ffbd349fb091f270647`, CI `34438961960`
- CP-008 proxy lifecycle core: `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`, CI `34445848656`
- CP-009 authenticated lifecycle API: `24cec6f875fb5f56bfb97d8159d8fa2ac3b8e533`, CI `34446509254`
- CP-010 Telemt quota/expiry contract: `907d818300f9cdb01473fecacf0cd76bd8db1438`, CI `34447125904`

Recovery point: CP-010. If interrupted during 6D2A, inspect all commits/files after CP-010 and repair/finish only the Credit Bucket ledger before any projection, bot, referral, sponsor, or UI expansion.

## Important decisions/discoveries

- Telemt source is not copied; integration is via authenticated API and pinned release artifacts.
- Telemt 3.5.7 rejects empty `[access.users]`; internal disabled bootstrap user satisfies that invariant until real users exist.
- Project Docker bridge cannot be `internal:true` because Telemt needs outbound Telegram connectivity; API isolation is no host publication + Bearer auth.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- Random Panel port is convenience/conflict avoidance, not a security boundary; production TLS hardening remains required.
- Roadmap requires Credit Buckets and nearest-expiry-first consumption; temporary referral rewards remain independent from other quota.
- Credit expiry is effective wall-clock state and should not depend on a background task merely to become non-consumable.
- Telemt quota/expiry is an enforcement projection; the Credit Bucket ledger remains authoritative business state.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode 0640 rejected; strict validation retained, fixed to UID 10001 + 0600.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E omitted temporary install path; test harness fixed only.
- Stage 5 `457f5278...`, CI `34427021157`: PASS; CP-005.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable no-port-binding assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token read before sudo; token stayed 0600, harness changed to `sudo cat`.
- Stage 6A `47a95349...`, CI `34438212903`: PASS; CP-006.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; gofmt repair `c67874de...`, CI `34438961960` PASS; CP-007.
- Stage 6C1 `281f91af...`, CI `34445848656`: PASS; CP-008.
- Stage 6C2 `24cec6f8...`, CI `34446509254`: PASS; CP-009.
- Stage 6D1 `907d8183...`, CI `34447125904`: PASS for Go and installer/Telemt E2E; CP-010.
- 2026-09-10: source SHA-256 values rechecked from mounted originals; a transient plan typo for ROADMAP_EN was corrected immediately before 6D1 code changes.

## Current next action

Implement only Stage 6D2A from CP-010: additive credit-bucket migration, transactional domain/store, and focused migration/ledger tests. Promote only after Go and existing Docker/Telemt E2E are green.
