# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `a7f83236a797120d2f5f34d8201a7de73b0b959f`

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
- CP-019 Telegram identity + idempotent start-gift foundation: `a7f83236a797120d2f5f34d8201a7de73b0b959f`, CI `34463953470` PASS.

### CP-019 implemented

- additive migration 006 creates `telegram_users` with unique Telegram identity and one-to-one proxy-user mapping
- non-secret typed `start_gift_bytes` setting defaults to decimal 100MB (`100_000_000` bytes)
- `/start` domain bootstrap is one SQLite transaction: proxy user, Telegram mapping and initial Credit Bucket succeed or roll back together
- deterministic proxy usernames are derived from positive Telegram IDs, not display names
- initial gift carries a per-user idempotency key; concurrent/repeated starts produce one user/mapping/gift
- changing start-gift setting affects only future new users; replay returns the originally granted amount
- migration upgrade test proves v5 proxy/credit rows survive migration 006 with legacy `idempotency_key` left NULL
- no Telegram network transport or bot token exists yet

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

### Stage 7B1 — Telegram Bot API client + safe command/update core — ACTIVE

Roadmap basis: Phase 3 requires Telegram Bot `/start` and user menu. Production webhook is preferred later and webhook secret authentication is required; this milestone first verifies transport/client contracts independently from HTTP webhook wiring.

Scope only:
- add protected-file Bot token loader with strict file-permission checks; never store token in SQLite or logs
- add narrow Telegram Bot API client for `sendMessage` using bounded HTTP timeouts/body limits and sanitized typed errors
- add strict update/message structs and `/start` command parser with optional payload extraction
- add a start application handler that invokes the verified `telegramuser.Start` service and produces a safe domain response containing Telegram identity, proxy username and current start-gift state
- add Telemt user-link retrieval client support based on upstream user views so existing users can later receive proxy links without Control Plane storing plaintext MTProto secrets
- no webhook HTTP endpoint, Bot token config wiring, Telemt user creation, Forced Join, referrals, sponsor or Bot Admin in B1

Acceptance:
- token files must be regular, owner-readable only and non-empty; token value never appears in errors
- Telegram API non-2xx/malformed/oversized responses map to narrow errors without response-body leakage
- `/start`, `/start@botname` and optional payload parse deterministically; unrelated messages are ignored
- repeated start handling remains idempotent through CP-019 service
- Telemt link retrieval validates username/output and returns link strings but never exposes/stores raw secret fields
- Go format/vet/test and installer/Telemt E2E stay green

### Stage 7B2 — Authenticated webhook + Control Plane wiring — PENDING
After B1 verification: add Telegram webhook endpoint with Telegram secret-token verification, request-size/rate bounds, config from protected token/secret files, Bot API sending, startup-safe lifecycle wiring and tests. Keep Forced Join/referrals separate.

### Stage 7B3 — Proxy provisioning/link response — PENDING
After webhook transport is verified: create missing Telemt user fail-closed, trigger quota reconciliation, retrieve Telemt-generated links and return user menu/status without storing MTProto secrets.

### Stage 7C — Forced Join — PENDING
Configurable required channels, membership checks, manual recheck, and fail-safe user messaging.

### Stage 7D — Referrals + rewards — PENDING
Unique-per-invitee referral attribution, self-referral protection, idempotent reward creation and configurable reward/expiry/caps; integrate with Credit Buckets.

## Important decisions/discoveries

- Credit Bucket ledger is authoritative; Telemt is only an enforcement projection.
- Telemt disable cancels active sessions and blocks new admission, so it is the fail-closed freeze primitive.
- Quota `0` means blocked, not unlimited.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- Roadmap Phase 3 order is Bot start/user menu, Forced Join, referrals/rewards, proxy links; implementation is split into independently verified milestones.
- Bot token is a secret and is read from a protected file rather than ordinary SQLite settings.
- Pinned Telemt user views reconstruct `tg://proxy` links from Telemt-managed user secrets, so Control Plane can display links later without persisting plaintext MTProto secrets itself.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode issue; validation kept strict and ownership/mode fixed.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E path harness bug; production unchanged.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token harness read fixed; token remained 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; repaired at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found missing future-start boundary; repaired at CP-012.
- Stage 6D2B2A `44b3dc76...`, CI `34452143936`: partial migration publication caused migration-count failure; final passed at CP-013; no force/reset.
- C3A `8040d4e3...`, CI `34458741101`: tests used in-memory SQLite incompatible with required WAL; production validation unchanged; fixed at CP-017.
- C3B `470ecb81...`, CI `34459855904`: two misspelled HTTP status constants caused vet failure; repaired without behavior change.
- 7A publish intermediate `bcac80e9...`: README was accidentally committed while intending to move a ref. No reset/force was used; the next commit restored the exact previous README blob. Net CP-018→7A diff contains only the six intended 7A files.
- 7A `a7f83236...`, CI `34463953470`: Go + installer/Telemt E2E PASS; CP-019.
- Full local Go suite remains unavailable in the container because external module DNS is unavailable; GitHub CI is authoritative.

## Current next action

Implement only Stage 7B1 from CP-019: protected Bot token loader, bounded Bot API client, strict update/start parsing, start application response, and Telemt user-link retrieval with focused tests. Do not wire webhook/network ingress until B1 is separately verified.
