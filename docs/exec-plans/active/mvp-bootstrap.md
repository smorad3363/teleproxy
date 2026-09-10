# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `3664c7ce93264b4036ee87a1862aa1ae3634c41a`

## Purpose

Build Teleproxy incrementally from the supplied roadmap while keeping every milestone secure, independently verifiable, and recoverable from Git + this file without relying on chat history.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane have independent lifecycles.
- Go + lightweight HTTP + SQLite for Control Plane; Telemt remains an external data-plane component.
- Docker-first installer is rerun-safe and does not expose the Telemt API on the host.
- No Docker socket in the Web App.
- No plaintext passwords, session/API/MTProto secrets, bot tokens, or private keys in logs/state.
- Proxy-user secrets are reveal-once and never stored by Control Plane.
- Desired state is persisted before data-plane reconciliation.
- Credit Buckets with independent expiry are authoritative business state; Telemt quota/expiry is only an enforcement projection.

## Recovery protocol

On interruption: read this plan; compare branch head with `Latest verified checkpoint`; inspect every later commit/file and CI result; repair the active partial milestone before starting another one; never reset/clean/force over unrelated work.

## Completed checkpoints

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
- CP-018 Control Plane reconciliation wiring: `3664c7ce93264b4036ee87a1862aa1ae3634c41a`, CI `34462985548` PASS.

### CP-018 implemented

- configurable bounded reconciliation concurrency is wired through config and Compose
- startup queues existing proxy users without making Control Plane startup depend on Telemt convergence
- authenticated + CSRF-protected manual reconciliation endpoint is available
- create/enable paths remain fail-closed: when the runner exists, Telemt stays disabled until quota projection has converged
- create preserves the reveal-once secret even if later queueing fails
- enable/disable queue reconciliation; rotate-secret remains independent
- runner shutdown cancels timers/work and is waited with a bounded context before DB close
- sync failures persist only narrow codes

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

### Stage 7A — Telegram identity + idempotent start-gift domain — ACTIVE

Roadmap basis: Phase 3 begins with `/start`; the flow resolves Telegram identity, creates a user only when new, creates the initial gift exactly once, then later resolves Forced Join/referrals/proxy links. Start gift must be configurable; default is 100MB.

Scope only:
- add durable Telegram-user identity mapped one-to-one to a proxy user
- add a typed non-secret setting for configurable `start_gift_bytes` with safe default 100MB
- add an idempotent transactional start/bootstrap application service: repeated `/start` for the same Telegram ID never creates another user or gift
- create the initial Credit Bucket with an explicit idempotency key/source suitable for audit/recovery
- generate a deterministic safe proxy username from Telegram ID; never use Telegram display names as identity
- retain existing Credit Bucket ledger as the single quota/reward truth
- no Telegram network transport/token/webhook, referral reward, Forced Join, sponsor, Bot Admin, or broad Web UI in this milestone

Acceptance:
- concurrent/repeated bootstrap for one Telegram ID yields one Telegram user, one proxy user and one start-gift bucket
- different Telegram IDs cannot map to the same proxy user
- invalid/non-positive Telegram IDs and invalid gift settings are rejected
- changing the configured start gift affects only future new users, never re-grants existing users
- no secret material is added to SQLite
- migrations are additive/rerun-safe; old database migration tests are updated
- Go format/vet/test and existing Docker/Telemt E2E remain green

### Stage 7B — Telegram Bot transport + `/start` response — PENDING
After 7A verification: add a narrow Bot API client/transport around the verified start service, token from a protected secret file only, safe update parsing/rate limits, and a user-facing start/menu response. Forced Join/referrals stay separate.

### Stage 7C — Forced Join — PENDING
Configurable required channels, membership checks, manual recheck, and fail-safe user messaging.

### Stage 7D — Referrals + rewards — PENDING
Unique-per-invitee referral attribution, self-referral protection, idempotent reward creation and configurable reward/expiry/caps; integrate with Credit Buckets.

## Important decisions/discoveries

- Credit Bucket ledger is authoritative; Telemt is only an enforcement projection.
- Telemt disable cancels active sessions and blocks new admission, so it is the fail-closed freeze primitive.
- Quota `0` means blocked, not unlimited.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- Roadmap Phase 3 order is Bot start/user menu, Forced Join, referrals/rewards, proxy links; implementation is split into smaller independently verified milestones.
- Bot token is a secret and will not be stored as an ordinary SQLite setting in 7A.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode issue; validation kept strict and ownership/mode fixed.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E path harness bug; production unchanged.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token harness read fixed; token remained 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; repaired at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found missing future-start boundary; repaired at CP-012.
- Stage 6D2B2A `44b3dc76...`, CI `34452143936`: partial migration publication caused migration-count failure; final passed at CP-013; no force/reset.
- C3A `8040d4e3...`, CI `34458741101`: tests used in-memory SQLite incompatible with required WAL; production validation unchanged; fixed at CP-017.
- C3B `470ecb81...`, CI `34459855904`: two misspelled `http.StatusServiceUnavailable` constants caused vet failure; repaired without behavior change.
- C3B repair `3664c7ce...`, CI `34462985548`: Go + installer/Telemt E2E PASS; CP-018.
- Full local Go suite remains unavailable in the container because external module DNS is unavailable; GitHub CI is authoritative.

## Current next action

Implement only Stage 7A from CP-018: additive Telegram identity/settings schema plus an idempotent transactional start-gift service and focused concurrency/replay tests. Do not add Telegram network transport until 7A is separately verified.
