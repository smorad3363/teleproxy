# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `bb6d6f98e8cf8e8806e7e5210dcbbb9a2fe31228`

## Recovery contract

Resume from `Latest verified checkpoint`, then inspect every later branch commit/file and its CI result before writing. Repair the active partial milestone first. Never reset, clean, force-push, or overwrite unrelated work.

Non-negotiable architecture: SQLite WAL/NORMAL is authoritative Control Plane state; Credit Buckets are authoritative quota/reward state; Telemt quota/expiry is only an enforcement projection; Control Plane and Telemt lifecycles remain independent; plaintext admin/API/Bot/webhook/MTProto secrets are never logged or persisted by Control Plane; DB migrations are additive/backward-compatible.

## Verified checkpoints

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
- CP-012 quota usage + projection boundaries: `1013aca4ea01456de043b3e98a74be4686532632`, CI `34449339437` PASS.
- CP-013 durable projection journal: `2cc686fa1da66b8cf3b37c1a6565d0bbb94520bf`, CI `34452322294` PASS.
- CP-014 atomic usage accounting: `cdde3bfb7a953d3be619e7d82204790dd2b6182e`, CI `34453311029` PASS.
- CP-015 durable reconciliation phases: `4b8ea6ebd16d8499f0a3409d1962cddc4b014bac`, CI `34454816438` PASS.
- CP-016 callable crash-safe reconciler: `23a9eb92fade84b66aa6ec7f4cce96f37de21325`, CI `34456452874` PASS.
- CP-017 bounded reconciliation runner: `5505a1e981270f38474cfbaccb40a8abc23e50e9`, CI `34458886839` PASS.
- CP-018 reconciliation wiring: `3664c7ce93264b4036ee87a1862aa1ae3634c41a`, CI `34462985548` PASS.
- CP-019 Telegram identity + idempotent start gift: `a7f83236a797120d2f5f34d8201a7de73b0b959f`, CI `34463953470` PASS.
- CP-020 Telegram Bot API + safe start core: `0ee6a2c0651c1db150b6939341664b7ede203c38`, CI `34465039914` PASS.
- CP-021 authenticated Telegram webhook: `1c7068420ac34a9c5182878e7befeeb5629d8ffc`, CI `34468311092` PASS.
- CP-022 durable Telemt provisioning ownership proof: `66353eee172412c270b8b50be3be2c5b18f04caf`, CI `34469442781` PASS.
- CP-023 crash-safe Bot provisioning + link response: `7560b7b17c288b9789e98d8f790bbb10dd79a21d`, CI `34472531024` PASS.
- CP-024 Forced Join channel domain: `fbc825783409aa01908528d29afca9320edc7916`, CI `34475645129` PASS.
- CP-025 Telegram membership + split start primitives: `8e172246c3f7c4c656ead73424b1fcc9a3330d77`, CI `34477880363` PASS.
- CP-026 membership-gated Telegram start: `dd0656d8ea1725ab4a9de68899ae474031590ee2`, CI `34480254706` PASS.
- CP-027 Forced Join authenticated Admin CRUD API: `bad15c07f439ff3d90945b08fb75eb9690160c58`, CI `34482353102` PASS.
- CP-028 Forced Join manual recheck UX: `523d13000bd26f1eeb1f4bc3ad6a40a3ca0ea57b`, CI `34490204497` PASS.
- CP-029 Referral identity + pending attribution domain: `989c32c234127e93879719a566046144201e6d18`, CI `34497165310` PASS.
- CP-030 `/start` referral resolution wiring: `caf9bffd275ed2418b540773042875b8fe2d23fa`, CI `34497967471` PASS.
- CP-031 Referral eligibility persistence primitives: `5f96316f7ea58581dce0d8f8a312518fe595a42b`, CI `34505475099` PASS.
- CP-032 Referral reward configuration primitives: `06dc77e0b18517179e0dea937a0612ed54bc356e`, CI `34505999351` PASS.
- CP-033 Authenticated Web Admin referral reward settings API: `7fd1143723532a9d9a7ff5591c914b966b7e129f`, CI `34506809233` PASS.
- CP-034 Referral history read model + authenticated Admin API: `80b02d1183edfb65163b7b7a5dfe1b3834a2de9a`, CI `34508958312` PASS.
- CP-035 Sponsor Profile persistence/domain primitives: `e1df28fd16d59b3c08eff86ee6b23d579eaf199f`, CI `34510107359` PASS.
- CP-036 Authenticated Sponsor Profile Admin CRUD API: `2abc393bf144b0e269d166a544f1df3277649ffa`, CI `34510993603` PASS.
- CP-037 Web Panel Sponsor management surface: `b16c08a5adcfdcf9a3ddaba9b6d9e5d85ab862c0`, CI `34518089493` PASS.
- CP-038 Proxy Node identity/config persistence primitives: `c2599bdedaffad413892769131ed925db0bec139`, CI `34519237959` PASS.
- CP-039 Authenticated Proxy Node Admin CRUD API: `23d43eba64af94efcf2259b3a24b67a065c8db44`, CI `34524258262` PASS.
- CP-040 Web Panel Proxy Node metadata management surface: `5c9c07eab510634490b9d70fe55295f6633500bd`, CI `34525730547` PASS.
- CP-041 Web Panel referral reward settings + history surface: `bb6d6f98e8cf8e8806e7e5210dcbbb9a2fe31228`, CI `34529186492` PASS.

### CP-041 implemented

- authenticated server-rendered `/referrals` exposes only the already-verified CP-033 reward settings and CP-034 referral history contracts through the existing minimal Web Panel pattern.
- settings updates go only to same-origin `PUT /api/referral/reward-settings` with the existing session-derived CSRF token; the page does not create Credit Buckets or choose any referral reward recipient.
- reward byte/day values are submitted as exact positive integer JSON text instead of lossily converting potentially large int64 values through JavaScript `Number`.
- referral history renders inviter/invitee Telegram IDs, status, rejection reason and relevant timestamps, and reuses the existing typed CP-034 pagination parser plus `before_id` cursor semantics.
- tests prove unauthenticated redirect, no-store rendering, typed invalid pagination, deterministic bounded pagination, Dashboard reachability and no mutation of referral attribution/settings/Credit Bucket state during rendering/pagination.
- no reward issuance, anti-abuse policy, Telegram/proxy provisioning behavior, Node runtime probing, Sponsor assignment, multi-Telemt routing or installer/compose behavior changed.
- final 7D4 diff is one atomic four-file commit and candidate `bb6d6f98e8cf8e8806e7e5210dcbbb9a2fe31228` passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun in CI `34529186492`.

## Supplied source hashes

- AGENTS: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- ROADMAP_EN: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- ROADMAP_FA: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

Stage 1 exact-byte root mirrors remain PARTIAL because prior GitHub transport could not safely preserve the large supplied files. Do not commit partial/corrupted mirrors.

## Pinned Telemt

Telemt `3.5.7`, upstream commit `4ca7418442478cd92f9e861c21977a81b249efc8`.
- amd64 musl SHA-256 `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl SHA-256 `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`

## Active stage

### Stage 7C2C — Web Panel Forced Join management surface — ACTIVE

Roadmap basis: Web Panel Forced Join explicitly requires multiple channels, enabled/disabled state, required/optional campaign state, ordering and custom text. CP-027 already provides the authenticated typed CRUD API and CP-028 already provides the Telegram manual recheck UX. This milestone adds only the missing Web Panel configuration surface.

Scope only:
- add authenticated server-rendered `/forced-join` using the existing minimal Web Panel pattern and no frontend framework;
- render all Forced Join channel records from the authoritative domain in deterministic `position,id` order, including disabled/optional records;
- allow create/update/delete only through the existing same-origin `/api/forced-join/channels` CRUD API with session-derived CSRF;
- expose only existing fields: chat reference, display name, Telegram join URL, enabled, required, position and custom text;
- surface safe returned Problem messages and preserve existing CP-024/027 validation/canonicalization;
- add the smallest Dashboard/navigation links needed to reach Forced Join management;
- do not call Telegram membership APIs, alter `/start` gating/manual-recheck behavior, issue rewards, change provisioning, add new secrets, or introduce broad content-management/audit changes.

Acceptance:
- unauthenticated `/forced-join` redirects to `/login` without CSRF markup;
- authenticated page renders all channels in deterministic management order and safely escapes stored display/custom text;
- create/edit/delete controls target only the existing typed API and include valid session-derived CSRF;
- enabled/required toggles and position/custom text round-trip through the existing API without alternate validation rules;
- page rendering itself does not call Telegram or mutate Forced Join/referral/Credit Bucket state;
- existing Sponsor/Node/referral/proxy/quota behavior remains unchanged;
- format/vet/test and Docker/Telemt E2E remain green.

### Stage 9D — Proxy Node test/health/status — BLOCKED ON RUNTIME CREDENTIAL CONTRACT

The roadmap requires Node test/status/health, but CP-038/039 intentionally persist no Node token/secret/password and current Telemt authentication comes from a protected runtime token file for the single global Telemt target. Repository/product evidence does not yet define the per-Node credential source or whether `internal_api_endpoint` is specifically a Telemt API base URL. Do not probe Nodes or invent credential storage/transport semantics until that contract is explicit.

### Stage 7D2C — Exactly-once referral reward issuance — BLOCKED ON PRODUCT SEMANTICS

Roadmaps define reward amount/expiry and invitee eligibility conditions but do not specify who receives the Credit Bucket: inviter, invitee, or both. Do not infer a recipient.

### Stage 7D3B — Remaining minimum anti-abuse controls — BLOCKED ON PRODUCT SEMANTICS

Roadmap requires configurable daily/weekly caps, cooldowns, blacklist and suspicious score, but does not define cap scope/defaults, cooldown semantics/default, blacklist subject or score inputs/threshold. Do not invent those contracts.

## Important decisions

- Credit Buckets are source of truth; Telemt is only enforcement projection.
- Telemt disable cancels active sessions; quota `0` blocks traffic.
- Bot token/webhook secret are protected runtime files, never ordinary SQLite settings.
- Telemt user view reconstructs proxy links from Telemt-managed secret; Control Plane does not persist MTProto plaintext.
- Forced Join gates gift and provisioning; referral attribution is durable before the Forced Join recheck gap.
- Referral credit recipient semantics remain unresolved; do not issue referral rewards.
- Sponsor Profile persistence is independent of assignment routing; sticky/weighted assignment and Telemt projection remain separate later milestones.
- CP-041 still leaves `TPROXY_TELEMT_API_URL`, current compose topology and quota reconciliation targeting one global Telemt service.
- Relay records are schema-readiness metadata only; no Iran Relay tunnel runtime exists yet.
- Existing Web Panel remains server-rendered/minimal; do not introduce a frontend framework for isolated management surfaces.

## Validation/failure log

- Prior format/test transport issues were repaired without weakening production validation; no reset/force-push was used.
- D2B2 transfer corruption was repaired at CP-033 and final CI passed.
- CP-034 docs `ba131d11...` CI `34509372354` PASS.
- CP-035 candidate `e1df28fd...` CI `34510107359` PASS; docs `55ee6d81...` CI `34510460750` PASS.
- CP-036 candidate `2abc393b...` CI `34510993603` PASS; docs `c1d93e72...` CI `34517423137` PASS.
- CP-037 candidate `b16c08a5...` CI `34518089493` PASS; docs `bb0e2308...` CI `34518442647` PASS.
- CP-038 candidate `c2599bde...` CI `34519237959` PASS on the first candidate. One initial unattached `create_commit` call was tool-blocked before any branch move; the same atomic tree was then committed normally and fast-forwarded. No repository state was lost or rewritten.
- CP-038 promotion commit `b930b1f3...` contained a documentation-only typo in the historical CP-010 SHA; it was immediately repaired in the next fast-forward docs commit before any 9B code publication.
- CP-039 candidate `23d43eba...` CI `34524258262` PASS on the first candidate; full Go and installer/Docker/Telemt E2E validation succeeded with no repair commit required.
- CP-039 promotion docs commit `890e4a90...` CI `34524760453` PASS.
- CP-040 candidate `5c9c07ea...` CI `34525730547` PASS on the first candidate; full Go and installer/Docker/Telemt E2E validation succeeded with no repair commit required.
- CP-040 promotion docs commit `82fb13d4...` CI `34528554637` PASS.
- CP-041 candidate `bb6d6f98...` CI `34529186492` PASS on the first candidate; full Go and installer/Docker/Telemt E2E validation succeeded with no repair commit required.

## Current next action

Verify the CP-041 promotion docs-head CI. Then implement only Stage 7C2C: authenticated server-rendered Forced Join channel management backed by the existing CP-027 CRUD API. Keep Telegram membership/recheck behavior, reward issuance, unresolved anti-abuse policy, Node runtime probing, Sponsor assignment, multi-Telemt routing and installer/compose behavior unchanged.
