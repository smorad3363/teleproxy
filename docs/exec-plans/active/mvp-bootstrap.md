# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `66353eee172412c270b8b50be3be2c5b18f04caf`

## Purpose

Build Teleproxy incrementally from the supplied roadmap while keeping every milestone secure, independently verifiable, and recoverable from Git + this file without relying on chat history.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane have independent lifecycles.
- Go + lightweight HTTP + SQLite for Control Plane; Telemt remains an external data-plane component.
- Docker-first installer is rerun-safe and does not expose the Telemt API on the host.
- No Docker socket in the Web App.
- No plaintext passwords, session/API/MTProto secrets, bot tokens, webhook secrets, or private keys in logs/state.
- Proxy-user plaintext secrets are never persisted by Control Plane.
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
- CP-019 Telegram identity + idempotent start-gift: `a7f83236a797120d2f5f34d8201a7de73b0b959f`, CI `34463953470` PASS.
- CP-020 Telegram Bot API + safe start core: `0ee6a2c0651c1db150b6939341664b7ede203c38`, CI `34465039914` PASS.
- CP-021 authenticated Telegram webhook wiring: `1c7068420ac34a9c5182878e7befeeb5629d8ffc`, CI `34468311092` PASS.
- CP-022 durable Telemt provisioning ownership proof: `66353eee172412c270b8b50be3be2c5b18f04caf`, CI `34469442781` PASS.

### CP-022 implemented

- additive migration 007 stores one provisioning journal per proxy user with phase, SHA-256 secret digest, narrow error code and timestamps; plaintext MTProto secrets are impossible to persist through this schema
- provisioning attempts generate a cryptographically random 16-byte/32-hex secret in memory and persist only its digest before network create
- Telemt client can create a user with a caller-supplied validated secret while legacy admin create behavior remains unchanged
- validated Telemt links normalize classic/secure/TLS encodings back to the effective raw 32-hex secret for ownership comparison
- ownership comparison is constant-time and produces an unforgeable package-private-backed `OwnershipProof`; `MarkOwned` requires that proof
- prepared-attempt digest may be CAS-replaced only after Telemt non-existence is established, enabling crash-before-network recovery without adopting a pre-existing same-name user
- migration 007 has upgrade/rerun/cascade coverage

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

### Stage 7B3B — Bot proxy provisioning + link response — ACTIVE

Purpose: compose CP-019 Telegram identity, CP-022 ownership proof and CP-017 quota runner into a crash-safe `/start` provisioning flow without persisting MTProto secrets.

Scope only:
- add a testable Bot provisioning application service; serialize one proxy username per process while DB/CAS journal remains the durable cross-retry guard
- for an owned journal, verify the Telemt user exists and continue without requiring the historical plaintext secret
- for a prepared journal, GET Telemt first: if found, prove ownership from links before marking owned; if absent, CAS-replace stale digest with a new in-memory attempt, create disabled with caller secret, verify links, then mark owned
- ambiguous create failures remain `prepared`; retry must GET+prove before any further create
- a pre-existing Telemt user without a matching prepared digest is collision/fail-closed, never adopted
- after ownership is established, trigger quota reconciliation; do not directly enable the user
- retrieve validated Telemt links only after reconciliation has been triggered; expose status/link text through the Bot response without storing link or plaintext secret in SQLite
- wire the service into the authenticated webhook while keeping unsupported commands ignored and outbound Bot API ambiguity non-destructive
- no Forced Join, referrals/rewards, sponsor/Admin UI or webhook registration in this milestone

Acceptance:
- first `/start` creates Control Plane identity/gift, creates Telemt user disabled, establishes ownership proof, queues reconciliation and returns a safe status/link response
- replay after success does not rotate/recreate secret
- timeout after Telemt create is recoverable by GET+ownership proof without a second create
- crash before network create can safely rotate the prepared digest only after GET proves absence
- same-name Telemt collision never gets adopted
- runner trigger failure leaves owned state recoverable and no direct enable occurs
- concurrent same-user provisioning is serialized and converges to one Telemt user
- Go format/vet/test and Docker/Telemt installer E2E remain green

### Stage 7C — Forced Join — PENDING
Configurable required channels, membership checks, manual recheck, and fail-safe user messaging.

### Stage 7D — Referrals + rewards — PENDING
Unique-per-invitee referral attribution, self-referral protection, idempotent reward creation and configurable reward/expiry/caps; integrate with Credit Buckets.

## Important decisions/discoveries

- Credit Bucket ledger is authoritative; Telemt is only an enforcement projection.
- Telemt disable cancels active sessions and blocks new admission; quota `0` means blocked.
- SQLite stays single-connection in MVP so connection-scoped PRAGMAs remain reliable.
- Bot token and webhook secret are protected runtime files, not ordinary SQLite settings.
- Telemt user views reconstruct `tg://proxy` links from Telemt-managed user secrets.
- Upstream CreateUser accepts an optional caller-provided 32-hex secret. Teleproxy can choose a random secret in memory, persist only its digest, and prove an ambiguous create later from Telemt-generated links.
- A provisioning ownership journal proves creation identity; it must not be treated as a permanent assertion about the current secret after an intentional admin secret rotation. Once phase is `owned`, later normal operations identify the user by Control Plane mapping rather than re-validating the historical digest.

## Validation/failure log

- Stage 5 `c1cde407...`, CI `34426475546`: bootstrap secret mode issue; strict validation kept and ownership/mode fixed.
- Stage 5 `34b455b0...`, CI `34426870772`: E2E path harness bug; production unchanged.
- Stage 6A `39349dcf...`, CI `34437891406`: unreliable port assertion; replaced with Docker HostConfig inspection.
- Stage 6A `91722c95...`, CI `34438040249`: protected token harness read fixed; token remained 0600.
- Stage 6B `0fbb8d72...`, CI `34438839369`: format-only failure; repaired at CP-007.
- Stage 6D2B1 `86090aba...`, CI `34449178722`: PASS but superseded after self-review found missing future-start boundary; repaired at CP-012.
- Stage 6D2B2A `44b3dc76...`, CI `34452143936`: partial migration publication caused migration-count failure; final passed at CP-013; no force/reset.
- C3A `8040d4e3...`, CI `34458741101`: test helper used in-memory SQLite incompatible with required WAL; production validation unchanged.
- C3B `470ecb81...`, CI `34459855904`: two misspelled HTTP status constants caused vet failure; repaired without behavior change.
- 7A publish intermediate `bcac80e9...`: accidental README commit while intending ref move; no reset/force; next commit restored exact prior README blob and net feature diff was clean.
- 7A `a7f83236...`, CI `34463953470`: PASS; CP-019.
- 7B1 `0ee6a2c0...`, CI `34465039914`: PASS; CP-020.
- 7B2 local pre-publish test caught body-cap status ambiguity; changed to read `limit+1` first so oversized bodies deterministically return 413.
- 7B2 `1c706842...`, CI `34468311092`: PASS; CP-021.
- 7B3A `f52aa0b6...`, CI `34469352217`: format-only failure in manually transferred `create_with_secret_test.go`; production files were unchanged.
- 7B3A repair `66353eee...`, CI `34469442781`: Go + installer/Telemt E2E PASS; CP-022.
- Full local Go suite remains unavailable in the container because external module DNS is unavailable; GitHub CI is authoritative. Standard-library-only slices are locally tested when possible.

## Current next action

Implement only Stage 7B3B from CP-022: crash-safe provisioning application orchestration, quota-runner trigger, validated link/status response, and narrow webhook wiring. Do not start Forced Join or referral logic until B3B is separately verified.
