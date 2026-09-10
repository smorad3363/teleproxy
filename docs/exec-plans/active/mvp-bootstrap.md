# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `2cc686fa1da66b8cf3b37c1a6565d0bbb94520bf`

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
- Credit Buckets with independent expiry are the quota business source of truth; Telemt quota/expiry is only an enforcement projection.

## Recovery protocol

On interruption:
1. Read this plan and relevant architecture/security/reliability docs.
2. Compare branch head with `Latest verified checkpoint`.
3. Inspect every file/commit after that checkpoint.
4. Re-run/inspect validation for the active partial milestone.
5. Repair that milestone before starting another one.
6. Never reset/clean/overwrite unrelated work; no force update unless explicitly authorized.

## Stage policy

Keep milestones small, buildable, independently understandable, and independently verifiable. Promote a checkpoint only after targeted tests and required regression CI pass.

## Completed stages

- Stage 1 — Repository foundation — PARTIAL. Execution plan, architecture/security/reliability docs, isolated branch and CI exist. Remaining exact-byte mirror of supplied root `AGENTS.md`, `ROADMAP_FA.md`, `ROADMAP_EN.md`; never commit partial mirrors.
- Stage 2 — Minimal Control Plane — COMPLETE. CI `34419759826` PASS.
- Stage 3 — Persistence/admin bootstrap — COMPLETE. `5798075de40d2f966d8546a18e6fa450d7142f90`; CI `34420104043` PASS.
- Stage 4 — Secure Web login — COMPLETE. `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`; CI `34420655852` PASS.
- Stage 5 — Recoverable Docker installer — COMPLETE. `457f52783f9b2962c55be00beb801f0f2534958c`; CI `34427021157` PASS.
- Stage 6A — Pinned Telemt 3.5.7 data plane — COMPLETE. `47a95349922ba5be37cf0ed8416b482de08580ef`; CI `34438212903` PASS.
- Stage 6B — Telemt health client — COMPLETE. `c67874de95b2ca4ff1786ffbd349fb091f270647`; CI `34438961960` PASS.
- Stage 6C1 — Proxy-user persistence + Telemt lifecycle client — COMPLETE. `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`; CI `34445848656` PASS.
- Stage 6C2 — Authenticated lifecycle API + reconciliation — COMPLETE. `24cec6f875fb5f56bfb97d8159d8fa2ac3b8e533`; CI `34446509254` PASS.
- Stage 6D1 — Telemt quota/expiry contract — COMPLETE. `907d818300f9cdb01473fecacf0cd76bd8db1438`; CI `34447125904` PASS.
- Stage 6D2A — Credit Bucket transactional ledger — COMPLETE. `50baa6572c01bf320ac475339cc82a6710438de8`; CI `34448520864` PASS.
- Stage 6D2B1 — Telemt quota usage read + pure projection — COMPLETE. `1013aca4ea01456de043b3e98a74be4686532632`; CI `34449339437` PASS.
- Stage 6D2B2A — Durable reconciliation journal + projection snapshot — COMPLETE. `2cc686fa1da66b8cf3b37c1a6565d0bbb94520bf`; CI `34452322294` PASS.

### Stage 6D2B2A implemented

- additive `004_quota_reconciliation.sql`
- per-user generation/phase, Telemt reset epoch and used baseline, projection timestamp/quota, enforced expiry and next-start boundary
- exact ordered member snapshot with bucket ID and allowance
- `PrepareProjection` persists `phase=applying`; preparation is not falsely treated as successful Telemt application
- generation compare-and-swap prevents stale replacement
- projection/member replacement is one SQLite transaction and rolls back completely on member failure
- member uniqueness/order and same-user bucket ownership enforced by schema
- unsigned Telemt values outside SQLite signed range rejected
- projection preparation does not mutate Credit Bucket consumption
- proxy-user cascade remains valid through deferred composite bucket FK

## Supplied source hashes

Rechecked from mounted originals on 2026-09-10:
- AGENTS: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- ROADMAP_EN: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- ROADMAP_FA: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

## Pinned Telemt

Telemt 3.5.7:
- amd64 musl SHA-256 `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl SHA-256 `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`

## Active stage

### Stage 6D2B2B — Atomic usage accounting against projection members — ACTIVE

Purpose: account observed Telemt quota usage exactly once against the exact buckets authorized by the active projection, even if wall clock expiry or administrative revocation happened after projection.

Scope only:
- add backward-compatible persisted per-member accounted bytes
- add a transaction that accepts one Telemt observation: reset epoch + used bytes
- require an active projection and matching Telemt reset epoch
- calculate delta from the persisted Telemt used baseline
- charge delta in saved projection-member order, bounded by each saved allowance
- detect member/bucket drift rather than silently double-charge or reassign historical traffic
- atomically update Credit Bucket consumed bytes, member accounted bytes, and reconciliation used baseline
- no Telemt network mutation, no HTTP endpoint, no background loop in this milestone

Acceptance:
- same observation is idempotent: second accounting consumes zero bytes
- partial crash cannot commit bucket changes without the new baseline, or baseline without bucket changes
- reset-epoch mismatch and used-byte regression fail without mutation
- inactive/applying projection cannot account traffic
- expired or revoked member buckets remain chargeable for traffic authorized by that snapshot
- accounting never exceeds a member's snapshotted allowance or a bucket's original bytes
- any post-projection unexpected bucket consumption is detected as projection drift
- insufficient saved allowance rolls the transaction back
- unsigned Telemt values outside SQLite signed range are rejected
- migration rerun and existing data remain valid
- Go format/vet/test plus existing Docker/Telemt E2E remain green

### Stage 6D2B2C — Crash-safe Telemt block/reset/apply state machine — PENDING
Only after B2B is verified: fail-closed block, stable usage observation, ledger accounting, durable reset detection, new projection application, and resume-by-phase behavior.

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
- CP-013 durable projection journal: `2cc686fa1da66b8cf3b37c1a6565d0bbb94520bf`, CI `34452322294`

Recovery point: CP-013. If interrupted during B2B, inspect every commit/file after CP-013 and repair only atomic usage accounting before any Telemt mutation.

## Important decisions/discoveries

- Telemt source is not copied; integration is via authenticated API and pinned release artifacts.
- Telemt API remains unexposed on host and authenticated by protected Bearer token.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- Random Panel port is convenience/conflict avoidance, not a security boundary; production TLS hardening remains required.
- Credit Bucket ledger is authoritative; wall-clock expiry needs no background write to become effective.
- Telemt quota counter is persistent `used_bytes` since reset and admission rejects when `used_bytes >= configured quota`; quota `0` blocks immediately.
- A projection needs both next finite expiry and next future-start boundary.
- Telemt has one quota/expiry while the ledger has multiple independent windows, so each applied projection must retain exact member allowances/order for later historical charging.
- Cross-system reconciliation must be fail-closed and resumable by persisted phase; never pretend SQLite + Telemt form one transaction.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode 0640 rejected; kept strict validation and fixed ownership/mode.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E test path bug; test harness fixed only.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token read without sudo; harness fixed, token stayed 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; gofmt repair verified at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found future-start boundary missing; repaired at CP-012.
- Stage 6D2B2A intermediate `44b3dc76...`, CI `34452143936`: schema was published before migration-count test update, so CI failed on expected migration count; no rollback/force used. Final candidate `2cc686fa...`, CI `34452322294`: PASS; CP-013.
- Local B2A SQL/FK cascade was additionally exercised with SQLite. Repository-wide Go verification remains GitHub CI authoritative.

## Current next action

Implement only Stage 6D2B2B from CP-013: migration for per-member accounted bytes and one transactional usage-accounting primitive with focused tests. Do not mutate Telemt until B2B is separately verified.
