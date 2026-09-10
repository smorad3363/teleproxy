# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `64dece67e2e0b9e2c2a9e2653b6fee42921c4450`

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
- CP-043 Authenticated read-only User inventory API: `5f513acd1849313beee05cd739640aad4b507e7a`, CI `34541388012` PASS.
- CP-044 Web Panel read-only User inventory surface: `64dece67e2e0b9e2c2a9e2653b6fee42921c4450`, CI `34541946964` PASS.

### CP-042 implemented

- authenticated server-rendered `/forced-join` lists every authoritative Forced Join channel in existing deterministic `position,id` order, including disabled and optional records.
- create/edit/delete controls send only the existing CP-027 fields to same-origin `/api/forced-join/channels` with session-derived CSRF; existing validation/canonicalization remains the only mutation contract.
- the page exposes chat reference, display name, Telegram join URL, enabled/required toggles, position and custom text, with `html/template` escaping and a clear empty state.
- Dashboard/navigation gained only the smallest links needed to reach Forced Join management; no frontend framework or unrelated UI refactor was introduced.
- tests cover unauthenticated redirect, no-store HTML, deterministic ordering, escaped stored text, exact API/CSRF wiring, empty state/Dashboard reachability and no mutation of Forced Join/referral/Credit Bucket state during rendering.
- no Telegram membership API call, `/start` or manual-recheck behavior change, reward issuance, provisioning change, new secret, Node runtime probe, Sponsor assignment, multi-Telemt routing or installer/compose change was added.
- final 7C2C diff is one atomic four-file commit and candidate `4b28df5b30e1686a0f43400df7425e400b5951b6` passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun in CI `34530056110`.

### CP-043 implemented

- added authenticated read-only `GET /api/users` with existing Admin API authentication, `Cache-Control: no-store`, bounded `before_id`/`limit` cursor pagination and typed `USER_INVENTORY_INVALID` errors.
- added a dedicated `internal/useradmin` read model over existing SQLite state only; no migration or write path was introduced.
- user inventory exposes authoritative Telegram ID ↔ Proxy User identity, desired/sync state and validated safe error code, currently available Credit Bucket bytes, nearest active credit expiry, durable inviter-attribution count and persisted timestamps.
- Credit Bucket availability and expiry are computed at one request timestamp from the authoritative ledger; Telemt quota is never used as source of truth.
- referral count includes pending/rewarded/rejected durable attributions and does not depend on unresolved reward-recipient semantics.
- tests cover authentication, no-store response, deterministic pagination, empty `[]`, authoritative credit/referral projection, invalid pagination, fail-closed unsafe stored error codes and no mutation of user/referral/Credit Bucket state.
- no Telegram/Telemt calls, secrets, user mutation actions, Telegram username inference, Node/Sponsor assignment, traffic/last-activity semantics, reward issuance or installer/compose changes were introduced.
- publication temporarily exposed partial fast-forward commits `4991971c3648b0bdec0f7a2a95923f1eac6bf094` and `7f7118cf91296c0f32a0edb094a87e9e118e3107` due contents/tool sequencing; the branch was repaired only by further fast-forward to coherent candidate `5f513acd1849313beee05cd739640aad4b507e7a`, with no reset, force or history rewrite. The final candidate passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun in CI `34541388012`.

### CP-044 implemented

- added authenticated server-rendered `/users` using the existing minimal Web Panel pattern over the CP-043 authoritative read model.
- the page renders only persisted/authoritative Telegram ID, proxy username, desired state, sync state/safe error code, current Credit Bucket bytes/nearest active expiry, referral count and created/updated timestamps.
- the page reuses CP-043 `before_id`/`limit` validation and cursor semantics, is `Cache-Control: no-store`, provides an explicit empty state and uses `html/template` escaping for stored strings.
- Dashboard gained only one Users link; the page remains read-only with no enable/disable, ban, credit, secret, Node, Sponsor or reset controls.
- tests cover unauthenticated redirect, deterministic pagination, escaped tampered stored text, current credit/referral projection, empty state, Dashboard reachability, typed invalid pagination and no mutation of authoritative state.
- no Telegram/Telemt calls, new secrets, migrations, inferred Telegram username/traffic/Node/Sponsor/last-activity fields or installer/compose changes were introduced.
- candidate `64dece67e2e0b9e2c2a9e2653b6fee42921c4450` is one atomic four-file fast-forward and passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun in CI `34541946964`.

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

### Stage 11C — Authenticated Start Gift settings API — ACTIVE

Roadmap basis: Web Panel Settings explicitly includes `start gift`. The repository already has `settings.StartGiftBytes` / `SetStartGiftBytes`, and Telegram `EnsureStartGift` reads that setting transactionally before creating the exactly-once start-gift Credit Bucket. This milestone exposes only that existing contract to authenticated Admin API clients.

Scope only:
- add authenticated `GET` and `PUT` for the existing start-gift byte setting using the current Admin API auth + CSRF contract;
- use a bounded JSON body, reject unknown fields/trailing JSON and keep errors typed/safe;
- accept only a positive int64 byte value, reuse `settings.SetStartGiftBytes` as the mutation primitive and return the effective stored value;
- default GET must return `settings.DefaultStartGiftBytes` when no override exists, matching current Telegram start behavior;
- mutation affects only future users whose exactly-once start gift has not yet been created; do not rewrite or top up existing Credit Buckets;
- no new migration, no reward-recipient semantics, no Telegram/Telemt call, no Bot token/content changes and no user-specific mutation endpoint.

Acceptance:
- unauthenticated GET/PUT follow existing Admin API behavior and PUT requires valid session-derived CSRF;
- GET returns default or configured positive int64 bytes with `Cache-Control: no-store`;
- PUT rejects malformed, unknown-field, trailing, oversized, zero/negative and overflow input with typed safe problems and no setting mutation;
- a successful PUT changes the amount used by a later `EnsureStartGift`, while replay for an already-gifted user remains exactly-once and preserves the original bucket amount;
- no existing Credit Bucket is mutated by changing the setting;
- existing Sponsor/Node/referral/Forced Join/User/proxy/quota behavior remains unchanged;
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
- CP-044 still leaves `TPROXY_TELEMT_API_URL`, current compose topology and quota reconciliation targeting one global Telemt service.
- Relay records are schema-readiness metadata only; no Iran Relay tunnel runtime exists yet.
- Existing Web Panel remains server-rendered/minimal; do not introduce a frontend framework for isolated management surfaces.
- The current user identity schema does not store Telegram username, Node/Sponsor assignment or general last activity; User surfaces must expose absence rather than infer those roadmap fields.
- Start Gift configuration changes only the future exactly-once grant amount; historical Credit Buckets remain authoritative and immutable except through explicit ledger operations.

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
- CP-042 promotion docs commit `461dc9c9...` CI `34530542473` PASS.
- CP-043 publication temporarily exposed partial fast-forward commits `4991971c...` and `7f7118cf...`; final coherent candidate `5f513acd...` repaired the milestone by fast-forward only. CI `34541388012` PASS across full Go and installer/Docker/Telemt E2E validation.
- CP-043 promotion docs commit `d1587d80...` CI `34541753619` PASS.
- CP-044 candidate `64dece67...` CI `34541946964` PASS on the first candidate; full Go and installer/Docker/Telemt E2E validation succeeded with no repair commit required.

## Current next action

Verify the CP-044 promotion docs-head CI. Then implement only Stage 11C: authenticated Admin API for the already-existing Start Gift byte setting. Keep existing Credit Buckets immutable, user-specific mutations, Telegram/Bot content, reward issuance, unresolved anti-abuse policy, Node runtime probing, multi-Telemt routing and installer/compose behavior unchanged.
