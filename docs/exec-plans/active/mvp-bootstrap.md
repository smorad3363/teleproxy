# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `23a9eb92fade84b66aa6ec7f4cce96f37de21325`

## Purpose

Build Teleproxy incrementally from an empty repository while keeping every milestone secure, independently verifiable, and recoverable from Git + this file without relying on chat history.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane have independent lifecycles.
- Go + lightweight HTTP + SQLite for Control Plane; Telemt remains an external data-plane component.
- Docker-first installer is rerun-safe, chooses/persists a random free high Panel port, and reveals first-install credentials only after health verification.
- Telemt API is authenticated and never host-published; no Docker socket in Web App.
- No plaintext passwords, session/API/MTProto secrets, bot tokens, or private keys in logs/state.
- Proxy-user secrets are reveal-once and never stored by Control Plane.
- Desired state is persisted before data-plane reconciliation so Telemt outages cannot lose admin intent.
- Credit Buckets with independent expiry are authoritative business state; Telemt quota/expiry is only an enforcement projection.

## Recovery protocol

On interruption: read this plan; compare branch head with `Latest verified checkpoint`; inspect every later commit/file and its CI; repair the active partial milestone before starting another one; never reset/clean/force over unrelated work.

## Completed stages

- Stage 1 — Repository foundation — PARTIAL. Architecture/security/reliability docs, plan, branch and CI exist. Remaining exact-byte root mirrors of supplied `AGENTS.md`, `ROADMAP_FA.md`, `ROADMAP_EN.md`; never commit partial mirrors.
- Stage 2 — Minimal Control Plane — COMPLETE. CI `34419759826` PASS.
- Stage 3 — Persistence/admin bootstrap — COMPLETE. `5798075de40d2f966d8546a18e6fa450d7142f90`; CI `34420104043` PASS.
- Stage 4 — Secure Web login — COMPLETE. `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`; CI `34420655852` PASS.
- Stage 5 — Recoverable Docker installer — COMPLETE. `457f52783f9b2962c55be00beb801f0f2534958c`; CI `34427021157` PASS.
- Stage 6A — Pinned Telemt 3.5.7 data plane — COMPLETE. `47a95349922ba5be37cf0ed8416b482de08580ef`; CI `34438212903` PASS.
- Stage 6B — Telemt health client — COMPLETE. `c67874de95b2ca4ff1786ffbd349fb091f270647`; CI `34438961960` PASS.
- Stage 6C1 — Proxy-user persistence + lifecycle client — COMPLETE. `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`; CI `34445848656` PASS.
- Stage 6C2 — Authenticated lifecycle API + desired-state reconciliation — COMPLETE. `24cec6f875fb5f56bfb97d8159d8fa2ac3b8e533`; CI `34446509254` PASS.
- Stage 6D1 — Telemt quota/expiry contract — COMPLETE. `907d818300f9cdb01473fecacf0cd76bd8db1438`; CI `34447125904` PASS.
- Stage 6D2A — Credit Bucket transactional ledger — COMPLETE. `50baa6572c01bf320ac475339cc82a6710438de8`; CI `34448520864` PASS.
- Stage 6D2B1 — quota usage read + pure projection — COMPLETE. `1013aca4ea01456de043b3e98a74be4686532632`; CI `34449339437` PASS.
- Stage 6D2B2A — durable reconciliation journal + member snapshot — COMPLETE. `2cc686fa1da66b8cf3b37c1a6565d0bbb94520bf`; CI `34452322294` PASS.
- Stage 6D2B2B — atomic usage accounting — COMPLETE. `cdde3bfb7a953d3be619e7d82204790dd2b6182e`; CI `34453311029` PASS.
- Stage 6D2B2C1 — durable fail-closed phases — COMPLETE. `4b8ea6ebd16d8499f0a3409d1962cddc4b014bac`; CI `34454816438` PASS.
- Stage 6D2B2C2 — callable crash-safe Telemt reconciliation orchestrator — COMPLETE. `23a9eb92fade84b66aa6ec7f4cce96f37de21325`; CI `34456452874` PASS.

### Stage 6D2B2C2 implemented

- dedicated `internal/quotareconcile` orchestration package keeps Credit Bucket domain separate from Telemt transport
- existing projections freeze traffic by persisted `blocking`, Telemt disable, then persisted `blocked`
- stable quota usage is observed and charged against the saved projection while the data-plane user is disabled
- persisted `resetting` makes ambiguous reset requests retryable; reset epoch is used to recognize a successful prior reset
- the next generation is prepared after reset, policy is applied idempotently, and policy state is verified before progression
- no-credit projection uses quota `0`, never an unlimited/cleared quota
- desired-enabled state is re-read from Control Plane before restoration; desired-disabled users remain disabled
- initial no-journal bootstrap is also fail-closed: disable, reset/observe, prepare generation 1, apply, then conditionally enable
- fake-driven tests cover normal path and ambiguous failures after disable/reset/apply/enable without double-accounting or extra quota
- pinned Telemt source confirms disable cancels active sessions and quota reset writes `used_bytes=0` with a new reset epoch

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

### Stage 6D2B2C3 — Reconciliation trigger/wiring — ACTIVE

Purpose: expose the verified callable reconciler through bounded Control Plane lifecycle triggers without turning reconciliation into an uncontrolled background subsystem.

Scope only:
- add a small reconciliation runner/manager with per-user deduplication and bounded global concurrency
- wire startup reconciliation for existing proxy users when Telemt is configured
- add an authenticated admin manual reconcile endpoint for one user
- schedule the next finite `NextExpiry`/`NextStart` boundary after a successful reconciliation using in-process timers owned by the runner
- cancel timers/work on Control Plane shutdown; do not block HTTP shutdown indefinitely
- trigger reconciliation after desired-enabled lifecycle mutations only after their existing operation completes; do not rewrite reveal-once create/rotate semantics in this milestone
- retain DB-first desired state and fail-closed Telemt behavior
- no bot/referral/sponsor features, no broad UI redesign, no multi-node scheduler in this milestone

Acceptance:
- at most one reconciliation runs per proxy user at a time
- global concurrency is bounded and configurable/default-safe
- duplicate manual/startup/boundary triggers coalesce rather than race generations
- shutdown cancels pending timers and contexts cleanly
- boundary scheduling uses both `NextExpiry` and `NextStart`, selecting the nearest future boundary
- manual endpoint requires admin session + CSRF and never returns Telemt secrets
- Telemt unavailable/reconciliation failure leaves durable phase for retry and is represented by a narrow sync error code, not raw upstream text
- startup with Telemt unconfigured continues without runner/network work
- Go format/vet/test plus existing Docker/Telemt E2E remain green

## Checkpoints

- CP-000 repo initialized: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
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
- CP-014 atomic usage accounting: `cdde3bfb7a953d3be619e7d82204790dd2b6182e`, CI `34453311029`
- CP-015 durable reconciliation phases: `4b8ea6ebd16d8499f0a3409d1962cddc4b014bac`, CI `34454816438`
- CP-016 callable crash-safe reconciler: `23a9eb92fade84b66aa6ec7f4cce96f37de21325`, CI `34456452874`

Recovery point: CP-016. If interrupted during C3, inspect every commit/file after CP-016 and repair only reconciliation trigger/wiring before bot/referral/sponsor/UI expansion.

## Important decisions/discoveries

- Telemt source is not copied; integration is via authenticated API and pinned artifacts.
- Telemt API is unexposed on host and Bearer-authenticated.
- SQLite remains single-connection in MVP so connection-scoped PRAGMAs stay reliable.
- Credit Bucket ledger is authoritative; Telemt is only an enforcement projection.
- Telemt quota is persistent absolute `used_bytes` since reset; quota `0` blocks admission.
- Each applied projection retains exact member allowances/order for historical charging after expiry/revocation.
- Telemt disable cancels active sessions as well as blocking new admission; reconciliation freezes traffic by disable before final usage observation.
- Cross-system reconciliation is fail-closed and persisted by phase; SQLite + Telemt are never treated as one transaction.
- C2 is callable only; scheduling is deliberately deferred to C3 so crash-safety was verified independently from concurrency/timer behavior.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode issue; strict validation kept and ownership/mode fixed.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E path harness bug; production unchanged.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token harness read fixed; token remained 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; repaired at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found missing future-start boundary; repaired at CP-012.
- Stage 6D2B2A intermediate `44b3dc76...`, CI `34452143936`: migration-count failure from partial publication; no reset/force. Final `2cc686fa...`, CI `34452322294`: PASS; CP-013.
- Stage 6D2B2B `cdde3bfb...`, CI `34453311029`: PASS; CP-014.
- Stage 6D2B2C1 `4b8ea6eb...`, CI `34454816438`: PASS; CP-015.
- Stage 6D2B2C2 `23a9eb92...`, CI `34456452874`: PASS for Go and installer/Telemt E2E; CP-016.
- Full local Go suite remains unavailable in the container because external module DNS is unavailable; GitHub CI is authoritative. Local gofmt/SQL checks are supplementary.

## Current next action

Implement only Stage 6D2B2C3 from CP-016: bounded/coalescing runner, startup/manual/boundary triggers, lifecycle/shutdown wiring, focused tests, then full CI. Do not start bot/referral/sponsor or broad UI work until C3 is separately verified.
