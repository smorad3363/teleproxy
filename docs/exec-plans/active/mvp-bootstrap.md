# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `1013aca4ea01456de043b3e98a74be4686532632`

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
- Roadmap quota business model uses Credit Buckets with independent expiry; Telemt quota/expiry values are enforcement projections, never the reward-ledger source of truth.

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
Verified `24cec6f875fb5f56bfb97d8159d8fa2ac3b8e533`; CI `34446509254` PASS. Authenticated lifecycle API, DB-first desired state, session-bound CSRF, safe `TELEMT_*` reconciliation errors, reveal-once secrets.

### Stage 6D1 — Telemt quota/expiry enforcement contract — COMPLETE
Verified `907d818300f9cdb01473fecacf0cd76bd8db1438`; CI `34447125904` PASS. Typed quota/expiry PATCH with exact merge-patch semantics, RFC3339 validation, quota reset primitive, bounded safe responses.

### Stage 6D2A — Credit Bucket transactional ledger — COMPLETE
Verified `50baa6572c01bf320ac475339cc82a6710438de8`; CI `34448520864` PASS. Additive Credit Bucket schema, integer accounting, derived wall-clock states, earliest-expiry-first transactional consumption and rollback tests.

### Stage 6D2B1 — Telemt quota usage read + pure credit projection — COMPLETE
Verified `1013aca4ea01456de043b3e98a74be4686532632`; CI `34449339437` PASS.

Implemented:
- typed authenticated `GET /v1/stats/users/quota` reader using existing bounded response machinery
- safe target-user lookup with explicit clean absence and malformed/duplicate output rejection
- no upstream response-body leakage
- pure Credit Bucket projection of active remaining bytes
- earliest active finite expiry boundary
- earliest future credit start boundary, added after self-review showed expiry-only scheduling would miss future activation
- non-expiring-only and zero-credit projections without inventing unlimited semantics
- overflow and malformed bucket rejection
- no DB, HTTP, background loop, reset, PATCH, or control-plane wiring in this milestone

## Active stage

### Stage 6D2B2A — Durable reconciliation journal + projection snapshot — ACTIVE

Purpose: persist enough local state to make the later Telemt mutation sequence restartable and to account traffic against the exact Credit Buckets that were part of the last applied projection.

Scope only:
- add backward-compatible SQLite tables for per-user quota reconciliation state and current projection members
- persist generation, phase, Telemt reset epoch/baseline usage, projection timestamp, projected quota, enforced expiry, and next-start boundary
- snapshot each participating bucket with its allowance at projection time and deterministic consumption order
- support atomic create/replace/load of a projection snapshot
- no Telemt network mutation, no HTTP endpoint, no background loop in this milestone

Acceptance:
- projection replacement and member snapshot commit atomically
- members reference existing Credit Buckets belonging to the same proxy user
- duplicate members/order are rejected
- signed SQLite byte limits cannot silently accept Telemt values outside supported range
- load after restart reproduces the exact projection generation and member allowances
- replacing a projection does not mutate Credit Bucket consumed bytes
- migration rerun and existing data remain valid
- Go format/vet/test plus existing Docker/Telemt E2E remain green

### Stage 6D2B2B — Atomic usage accounting against projection members — PENDING
Consume Telemt usage delta transactionally against the saved projection members and Credit Buckets, updating the reconciliation baseline exactly once. Historical traffic must remain attributable even when a member bucket has since expired or been revoked.

### Stage 6D2B2C — Crash-safe Telemt block/reset/apply state machine — PENDING
Only after the journal/accounting primitives are verified: fail-closed block, stable usage observation, ledger accounting, durable reset detection, new projection apply, and resume-by-phase behavior.

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
- CP-011 Credit Bucket ledger: `50baa6572c01bf320ac475339cc82a6710438de8`, CI `34448520864`
- CP-012 quota usage + pure projection: `1013aca4ea01456de043b3e98a74be4686532632`, CI `34449339437`

Recovery point: CP-012. If interrupted during 6D2B2A, inspect all commits/files after CP-012 and repair only the reconciliation journal/snapshot before usage accounting or any Telemt mutation.

## Important decisions/discoveries

- Telemt source is not copied; integration is via authenticated API and pinned release artifacts.
- Telemt API remains unexposed on host and authenticated by protected Bearer token.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- Random Panel port is convenience/conflict avoidance, not a security boundary; production TLS hardening remains required.
- Roadmap requires Credit Buckets and nearest-expiry-first consumption; Credit Bucket ledger is authoritative business state.
- Credit expiry is effective wall-clock state and does not depend on a background job to become non-consumable.
- Telemt quota counter is persistent `used_bytes` since reset; admission rejects when `used_bytes >= configured quota`.
- Telemt quota value `0` blocks admission immediately; it is not unlimited.
- Telemt `GET /v1/stats/users/quota` reports positive configured quota users with `data_quota_bytes`, `used_bytes`, and `last_reset_epoch_secs`.
- A projection needs both the next finite expiry and next future start boundary; either can change available business credit without traffic.
- Telemt has one quota/expiry per user while the ledger has multiple independent credit windows. The last applied projection must therefore snapshot member bucket IDs/allowances so historical traffic can be charged to what was actually authorized, even after wall-clock expiry.
- Later reconciliation should be fail-closed and resumable by persisted phase; do not rely on a cross-system transaction between SQLite and Telemt.

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
- Stage 6D1 `907d8183...`, CI `34447125904`: PASS; CP-010.
- Stage 6D2A `50baa657...`, CI `34448520864`: PASS; CP-011.
- Stage 6D2B1 initial `86090aba...`, CI `34449178722`: PASS but superseded after self-review found future-start boundary was missing.
- Stage 6D2B1 repair `1013aca4...`, CI `34449339437`: PASS for format/vet/test plus installer/Telemt E2E; CP-012.
- Local 6D2A test attempt could not resolve github.com from the container; no local-pass claim was made. GitHub CI is authoritative for repository-wide verification.

## Current next action

Implement only Stage 6D2B2A from CP-012: additive reconciliation journal/projection-member schema and atomic store/tests. Do not mutate Telemt until this persistence layer is separately verified.
