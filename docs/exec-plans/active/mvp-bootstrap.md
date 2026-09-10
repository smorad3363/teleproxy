# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `4b8ea6ebd16d8499f0a3409d1962cddc4b014bac`

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

### Stage 6D2B2C1 implemented

- explicit persisted phases: `applying`, `active`, `blocking`, `blocked`, `resetting`, `enabling`
- strict allowed transition graph with generation+expected-phase compare-and-swap
- stale generation/phase and unsupported transitions cannot advance state
- `PrepareProjection` can replace a pre-apply snapshot or create the post-reset generation, but cannot bypass an `active` projection directly
- unknown stored phases are rejected as invalid state
- usage accounting is allowed in `active` and `blocked` only; its baseline CAS uses the observed phase rather than hard-coding `active`
- focused tests cover transition graph, stale CAS, re-projection gate, blocked accounting and unknown-phase rejection
- pinned Telemt source confirms disabling a user cancels active sessions, making disable suitable for fail-closed freezing before final accounting

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

### Stage 6D2B2C2 — Callable crash-safe Telemt reconciliation orchestrator — ACTIVE

Purpose: combine the separately verified journal, usage accounting and Telemt primitives into one resumable, fail-closed operation without introducing background scheduling yet.

Scope only:
- add a dedicated orchestration package with a narrow Telemt interface and fake-driven tests
- on an existing active projection: persist `blocking`, disable the Telemt user, persist `blocked`, read stable quota usage, account it, persist `resetting`, reset Telemt quota, observe the new reset epoch, prepare the next projection, apply quota/expiry, then restore current desired-enabled state through `enabling`/`active`
- resume safely from `blocking`, `blocked`, `resetting`, `applying`, or `enabling` after a process/network failure
- initial no-journal bootstrap must fail closed: disable first, reset/observe baseline while disabled, prepare generation 1, apply policy, then restore current desired state
- re-read Control Plane desired-enabled state immediately before any re-enable decision
- verify enough Telemt state after reset/apply to distinguish successful previous network mutation from a retry
- no HTTP endpoint, timer, goroutine, startup loop or credit-grant wiring in this milestone

Acceptance:
- every external mutation is preceded/followed by a durable phase boundary sufficient for retry
- a crash after successful disable/reset/apply/enable can resume without double-accounting or granting extra quota
- usage is accounted only while data-plane user is disabled/frozen
- reset epoch change is used to recognize a reset that succeeded before a crash
- policy application is idempotent and verified before progressing
- no-credit projection remains quota `0` (blocked), never unlimited
- desired-disabled users are never re-enabled by reconciliation
- Telemt failures leave a persisted fail-closed/resumable phase and do not alter Credit Bucket truth except through verified usage accounting
- Go format/vet/test plus existing Docker/Telemt E2E remain green

### Stage 6D2B2C3 — Reconciliation trigger/wiring — PENDING
After C2 verification, wire initial/manual/boundary reconciliation into Control Plane lifecycle with bounded concurrency and shutdown behavior. Background scheduling is not part of C2.

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

Recovery point: CP-015. If interrupted during C2, inspect every commit/file after CP-015 and repair only the callable reconciliation orchestrator before adding triggers, timers, bot/referral/sponsor or UI expansion.

## Important decisions/discoveries

- Telemt source is not copied; integration is via authenticated API and pinned artifacts.
- Telemt API is unexposed on host and Bearer-authenticated.
- SQLite remains single-connection in MVP so connection-scoped PRAGMAs stay reliable.
- Credit Bucket ledger is authoritative; Telemt is only an enforcement projection.
- Telemt quota is persistent absolute `used_bytes` since reset; quota `0` blocks admission.
- Each applied projection retains exact member allowances/order for historical charging after expiry/revocation.
- Telemt disable cancels active sessions as well as blocking new admission; reconciliation therefore freezes traffic by disable before final usage observation.
- Cross-system reconciliation is fail-closed and persisted by phase; SQLite + Telemt are never treated as one transaction.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode issue; strict validation kept and ownership/mode fixed.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E path harness bug; production unchanged.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token harness read fixed; token remained 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; repaired at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found missing future-start boundary; repaired at CP-012.
- Stage 6D2B2A intermediate `44b3dc76...`, CI `34452143936`: migration-count failure from partial publication; no reset/force. Final `2cc686fa...`, CI `34452322294`: PASS; CP-013.
- Stage 6D2B2B `cdde3bfb...`, CI `34453311029`: PASS; CP-014.
- Stage 6D2B2C1 `4b8ea6eb...`, CI `34454816438`: PASS for format/vet/test and installer/Telemt E2E; CP-015.
- Full local Go suite remains unavailable in the container because external module DNS is unavailable; GitHub CI is authoritative. Local gofmt/SQL checks are supplementary.

## Current next action

Implement only Stage 6D2B2C2 from CP-015: callable fake-tested crash-safe reconciliation orchestration. Do not add scheduling or lifecycle wiring until C2 is separately verified.
