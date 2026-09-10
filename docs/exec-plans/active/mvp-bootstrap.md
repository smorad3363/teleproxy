# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `5505a1e981270f38474cfbaccb40a8abc23e50e9`

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

On interruption: read this plan; compare branch head with `Latest verified checkpoint`; inspect every later commit/file and CI result; repair the active partial milestone before starting another one; never reset/clean/force over unrelated work.

## Completed stages/checkpoints

- Stage 1 — Repository foundation — PARTIAL. Architecture/security/reliability docs, plan, branch and CI exist. Remaining exact-byte root mirrors of supplied `AGENTS.md`, `ROADMAP_FA.md`, `ROADMAP_EN.md`; never commit partial mirrors.
- CP-002 CI foundation: `084d4a1256a6b28412d9457f6568a4138e09c83c`, CI `34419759826` PASS.
- CP-003 persistence/admin: `5798075de40d2f966d8546a18e6fa450d7142f90`, CI `34420104043` PASS.
- CP-004 Web auth: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`, CI `34420655852` PASS.
- CP-005 Docker installer: `457f52783f9b2962c55be00beb801f0f2534958c`, CI `34427021157` PASS.
- CP-006 Telemt 3.5.7 data plane: `47a95349922ba5be37cf0ed8416b482de08580ef`, CI `34438212903` PASS.
- CP-007 Telemt health client: `c67874de95b2ca4ff1786ffbd349fb091f270647`, CI `34438961960` PASS.
- CP-008 proxy lifecycle core: `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`, CI `34445848656` PASS.
- CP-009 authenticated lifecycle API: `24cec6f875fb5f56bfb97d8159d8fa2ac3b8e533`, CI `34446509254` PASS.
- CP-010 Telemt quota/expiry contract: `907d818300f9cdb01473fecacf0cd76bd8db1438`, CI `34447125904` PASS.
- CP-011 Credit Bucket ledger: `50baa6572c01bf320ac475339cc82a6710438de8`, CI `34448520864` PASS.
- CP-012 quota usage + pure projection: `1013aca4ea01456de043b3e98a74be4686532632`, CI `34449339437` PASS.
- CP-013 durable projection journal: `2cc686fa1da66b8cf3b37c1a6565d0bbb94520bf`, CI `34452322294` PASS.
- CP-014 atomic usage accounting: `cdde3bfb7a953d3be619e7d82204790dd2b6182e`, CI `34453311029` PASS.
- CP-015 durable reconciliation phases: `4b8ea6ebd16d8499f0a3409d1962cddc4b014bac`, CI `34454816438` PASS.
- CP-016 callable crash-safe reconciler: `23a9eb92fade84b66aa6ec7f4cce96f37de21325`, CI `34456452874` PASS.
- CP-017 bounded reconciliation runner core: `5505a1e981270f38474cfbaccb40a8abc23e50e9`, CI `34458886839` PASS.

### CP-017 implemented

- fixed worker pool with configurable/default-safe global concurrency; no goroutine-per-user fan-out
- per-user deduplication: queued duplicates coalesce; duplicates arriving while a run is active request at most one pending rerun
- successful active snapshots schedule the nearest `NextExpiry`/`NextStart` boundary through runner-owned timers
- explicit trigger cancels stale boundary timer for that user
- startup helper can enqueue all existing proxy users without waiting for Telemt network convergence
- runner shutdown cancels timers and in-flight contexts; `Wait` is caller-bounded
- reconciliation failures store only `QUOTA_RECONCILE_FAILED`, never raw upstream error text
- focused tests cover concurrency bound, same-user exclusion/coalescing, startup trigger-all, narrow error persistence, timer retrigger and shutdown

## Supplied source hashes

Rechecked from mounted originals on 2026-09-10:
- AGENTS: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- ROADMAP_EN: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- ROADMAP_FA: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

## Pinned Telemt

Telemt 3.5.7, upstream commit `4ca7418442478cd92f9e861c21977a81b249efc8`:
- amd64 musl SHA-256 `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl SHA-256 `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`

## Active stage

### Stage 6D2B2C3B — Control Plane reconciliation wiring — ACTIVE

Purpose: wire the separately verified runner into startup/admin/lifecycle/shutdown paths without weakening reveal-once secret handling or fail-closed quota bootstrap.

Scope only:
- add `TPROXY_RECONCILE_CONCURRENCY` config, default 2, valid 1..32, and Compose pass-through
- when Telemt is configured, create the verified Telemt reconciler + runner and enqueue existing users on startup
- startup queueing must not wait for Telemt network success; worker failures remain asynchronous/narrow
- add authenticated + CSRF-protected `POST /api/proxy/users/{username}/reconcile`
- trigger runner after create and desired-enable/disable lifecycle mutations; rotate-secret remains unchanged
- when a runner is configured, create a new Telemt user initially disabled even if DB desired state is enabled; this closes the pre-projection unlimited window while preserving DB desired=true and reveal-once secret response
- if post-create trigger queueing fails, still return the reveal-once secret once and mark a narrow sync error; never discard a generated secret because a later queue action failed
- stop runner at shutdown and wait only with a bounded context before DB close
- no bot/referral/sponsor, broad UI redesign, or multi-node scheduler in this milestone

Acceptance:
- old constructors/tests remain compatible when no runner is configured
- manual reconcile endpoint requires admin session + CSRF; returns no secret material
- trigger errors expose/store only narrow codes, not upstream error bodies
- desired-enabled create with runner creates Telemt user disabled until reconciler applies quota and restores desired state
- enable/disable successful lifecycle mutations queue reconciliation; rotate does not
- startup without Telemt remains functional and creates no runner/network work
- config rejects invalid concurrency and Compose defaults it safely
- Go format/vet/test and Docker/Telemt installer E2E remain green

Recovery point: CP-017. If interrupted during C3B, inspect every commit/file after `5505a1e...` and repair only Control Plane reconciliation wiring before starting any later roadmap feature.

## Important decisions/discoveries

- Credit Bucket ledger is authoritative; Telemt is only an enforcement projection.
- Telemt disable cancels active sessions and blocks new admission, so it is the fail-closed freeze primitive.
- Telemt reset writes `used_bytes=0` and advances reset epoch; ambiguous reset is recognized by the persisted state machine.
- Quota `0` means blocked, not unlimited.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- C2 crash-safety and C3A concurrency/timer behavior were verified independently before wiring them into process lifecycle.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode issue; strict validation kept and ownership/mode fixed.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E path harness bug; production unchanged.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token harness read fixed; token remained 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; repaired at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found missing future-start boundary; repaired at CP-012.
- Stage 6D2B2A `44b3dc76...`, CI `34452143936`: migration-count failure from partial publication; final `2cc686fa...` passed at CP-013; no force/reset.
- C3A `8040d4e305a07091bde91136daf4de6e1dea473e`, CI `34458741101`: format/vet PASS, tests failed because test helper used in-memory SQLite while production `database.Open` requires WAL. Production validation was not weakened; test switched to a temp file DB.
- C3A repair `5505a1e981270f38474cfbaccb40a8abc23e50e9`, CI `34458886839`: Go + installer/Telemt E2E PASS; CP-017.
- Full local Go suite remains unavailable in the container because external module DNS is unavailable; GitHub CI is authoritative. Local gofmt/YAML/SQL checks are supplementary.

## Current next action

Publish and verify only C3B Control Plane wiring from CP-017. On success, promote Stage 6D2B2C3 as complete before any bot/referral/sponsor or broader UI work.
