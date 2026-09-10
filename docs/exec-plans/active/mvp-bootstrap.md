# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `4b28df5b30e1686a0f43400df7425e400b5951b6`

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
- CP-042 Web Panel Forced Join management surface: `4b28df5b30e1686a0f43400df7425e400b5951b6`, CI `34530056110` PASS.

### CP-042 implemented

- authenticated server-rendered `/forced-join` lists every authoritative Forced Join channel in existing deterministic `position,id` order, including disabled and optional records.
- create/edit/delete controls send only the existing CP-027 fields to same-origin `/api/forced-join/channels` with session-derived CSRF; existing validation/canonicalization remains the only mutation contract.
- the page exposes chat reference, display name, Telegram join URL, enabled/required toggles, position and custom text, with `html/template` escaping and a clear empty state.
- Dashboard/navigation gained only the smallest links needed to reach Forced Join management; no frontend framework or unrelated UI refactor was introduced.
- tests cover unauthenticated redirect, no-store HTML, deterministic ordering, escaped stored text, exact API/CSRF wiring, empty state/Dashboard reachability and no mutation of Forced Join/referral/Credit Bucket state during rendering.
- no Telegram membership API call, `/start` or manual-recheck behavior change, reward issuance, provisioning change, new secret, Node runtime probe, Sponsor assignment, multi-Telemt routing or installer/compose change was added.
- final 7C2C diff is one atomic four-file commit and candidate `4b28df5b30e1686a0f43400df7425e400b5951b6` passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun in CI `34530056110`.

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

### Stage 11A — Authenticated read-only User inventory API — ACTIVE

Roadmap basis: Web Panel Users requires Telegram identity, status, credits, expiry, referrals and created/activity data. Current repository state already has authoritative Telegram ID ↔ Proxy User identity, Proxy User desired/sync state, Credit Buckets, referral attribution and timestamps. It does not yet persist Telegram username, Node/Sponsor assignment or a general user last-activity field, so those must not be fabricated in this milestone.

Scope only:
- add a read-only user-admin domain/read model over existing tables; no migration;
- expose authenticated `GET /api/users` using the existing Admin API auth contract, `Cache-Control: no-store` and bounded deterministic cursor pagination;
- return only authoritative existing fields: Telegram user ID, Telegram ID, Proxy User ID/username, desired enabled state, sync state and safe last error code, currently available Credit Bucket bytes, nearest active credit expiry if any, referral count, created_at and updated_at;
- compute credit availability/expiry from Credit Buckets as source of truth at one request timestamp; do not read Telemt quota as authoritative state;
- referral count is the count of durable attributions where the user is inviter, independent of unresolved reward recipient semantics;
- invalid pagination returns a typed/safe problem without mutation;
- do not expose Telemt/MTProto secrets, call Telemt/Telegram, add user mutations, infer Telegram username, Node/Sponsor assignment, traffic/last-activity semantics, or issue/revoke Credit Buckets.

Acceptance:
- authenticated list is deterministic and paginates without duplicates/skips under stable data;
- balances and nearest expiry reflect authoritative active Credit Buckets and do not use Telemt as source of truth;
- referral count reflects durable inviter attributions regardless of pending/rewarded/rejected state;
- unauthenticated calls fail under existing Admin API behavior and malformed pagination returns typed safe errors;
- listing is read-only and does not mutate Telegram users, Proxy Users, referrals, Credit Buckets or Telemt state;
- existing Sponsor/Node/referral/Forced Join/proxy/quota behavior remains unchanged;
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
- CP-042 still leaves `TPROXY_TELEMT_API_URL`, current compose topology and quota reconciliation targeting one global Telemt service.
- Relay records are schema-readiness metadata only; no Iran Relay tunnel runtime exists yet.
- Existing Web Panel remains server-rendered/minimal; do not introduce a frontend framework for isolated management surfaces.
- The current user identity schema does not store Telegram username, Node/Sponsor assignment or general last activity; Stage 11A must expose absence rather than infer those roadmap fields.

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
- CP-041 promotion docs commit `d78e526e...` CI `34529587053` PASS.
- CP-042 candidate `4b28df5b...` CI `34530056110` PASS on the first candidate; full Go and installer/Docker/Telemt E2E validation succeeded with no repair commit required.

## Current next action

Verify the CP-042 promotion docs-head CI. Then implement only Stage 11A: authenticated read-only user inventory from existing authoritative SQLite state. Keep user mutation actions, Telegram username persistence, Node/Sponsor assignment, traffic/last-activity semantics, reward issuance, unresolved anti-abuse policy, Node runtime probing, multi-Telemt routing and installer/compose behavior unchanged.
