# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `89c147f0a34ccc62f8014c030beef2a443c9cc15`

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
- CP-045 Authenticated Start Gift settings API: `e7ac0ccdd4747667dec8681452b3ace4e4539fd0`, CI `34542385124` PASS.
- CP-046 Web Panel Start Gift settings surface: `198100a015195c5ac64c13ddb4eea90e1c653eb6`, CI `34543281630` PASS.
- CP-047 Bot Content persistence primitives: `88465adede77f118c9507b016bd38274bd2408f0`, CI `34543883064` PASS.
- CP-048 Authenticated Bot Content Admin API: `10dbde635dc8909e9bcbbe633ff659fc9934480e`, CI `34544396136` PASS.
- CP-049 Web Panel Bot Content management surface: `8588193e98d027c16f45b3f77149acf18c65cb9f`, CI `34544809255` PASS.
- CP-050 Audit Log persistence primitives: `4a52a8864f1f348c5020fbfff7d76b3dbeaa59e8`, CI `34567969934` PASS.
- CP-051 Authenticated read-only Audit Log Admin API: `89c147f0a34ccc62f8014c030beef2a443c9cc15`, CI `34573865558` PASS.

### Recent implemented checkpoints

- CP-043 added authenticated read-only User inventory API from authoritative SQLite only; candidate `5f513acd...`, CI `34541388012` PASS.
- CP-044 added read-only `/users` Web Panel; candidate `64dece67...`, CI `34541946964` PASS.
- CP-045 added authenticated Start Gift settings API preserving exactly-once Credit Bucket history; candidate `e7ac0ccd...`, CI `34542385124` PASS.
- CP-046 added `/settings` Start Gift Web Panel with exact int64 browser handling; candidate `198100a0...`, CI `34543281630` PASS.
- CP-047 added migration `013_bot_content.sql` plus literal-text Bot Content persistence for seven fixed roadmap slots. Initial `f9b5ad04...` CI failed only because legacy migration-count tests expected 12; forward repair ended at `88465ade...`, CI `34543883064` PASS.
- CP-048 added authenticated `GET /api/bot-content`, CSRF-protected per-slot `PUT` and `DELETE`, bounded strict JSON, typed Problems, deterministic configured-only list and empty `[]`. No Bot runtime wiring or network calls. Candidate `10dbde63...`, CI `34544396136` PASS.
- CP-049 added authenticated `/bot-content` Web Panel showing all seven fixed slots, explicit not-configured state, escaped literal text and same-origin CSRF PUT/DELETE wiring over CP-048. No fallback copy or Bot runtime wiring. Candidate `8588193e...`, CI `34544809255` PASS.
- CP-050 added additive migration `014_audit_log.sql` plus `internal/auditlog` append/get/list primitives for roadmap actor/action/target/before/after/timestamp/request-ID fields. The table is database-level append-only via update/delete rejection triggers; required identifiers and snapshots are bounded/UTF-8 validated; snapshots remain opaque caller-owned text, no existing mutation path is wired, and sensitive state is never implicitly copied. Migration compatibility tests now expect 14. Candidate `4a52a886...`, CI `34567969934` PASS.
- CP-051 added authenticated read-only `GET /api/audit-log` with bounded `before_id`/`limit` pagination, newest-first SQLite reads, exact stored audit fields, explicit empty `[]`, `Cache-Control: no-store`, stable `AUDIT_LOG_INVALID` Problems, and no mutation endpoint or automatic mutation wiring. Candidate `89c147f0...`, CI `34573865558` PASS.

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

### Stage 11K — Web Panel read-only Audit Log surface — ACTIVE

CP-051 exposes the roadmap Audit Log fields through an authenticated read-only API. This milestone adds only the corresponding authenticated Web Panel surface; it does not create audit events or change mutation behavior.

Scope only:
- add authenticated `GET /audit-log` Web Panel route using the existing session/page-auth pattern;
- render newest-first actor, action, target, request ID and timestamp plus optional before/after snapshots from authoritative SQLite;
- render snapshots as escaped literal text, preserving absence explicitly and never interpreting snapshot contents as HTML;
- provide bounded read-only pagination consistent with CP-051; no client-side write operation or audit mutation endpoint;
- add a minimal Dashboard navigation link for Audit Log;
- no migration, mutation logging, automatic snapshot serialization/redaction, Telegram/Telemt call, Node/Sponsor routing, reward issuance, secret handling change or installer/compose change.

Acceptance:
- unauthenticated page follows the existing authenticated Web Panel behavior;
- empty state is explicit and safe;
- actor/action/target/request ID/timestamp and optional before/after snapshots are rendered faithfully with HTML escaping;
- pagination remains bounded and deterministic newest-first;
- page load does not alter `audit_log` or other authoritative state;
- format/vet/test and Docker/Telemt E2E remain green.

### Stage 11H — Bot Content runtime delivery wiring — BLOCKED ON PRODUCT SEMANTICS

Roadmap Web Panel Bot Content names seven content slots — welcome, forced join, referral, proxy, expired, no-credit and support — but does not define runtime composition/fallback semantics. Current Telegram `webhook.go` emits one combined account/proxy/referral start response, has a separate Forced Join formatter/keyboard, and has no established expired/no-credit/support delivery events. The roadmap also lists button labels, Premium/Custom Emoji and fallback emoji separately without defining their relationship to literal text overrides. Do not map or concatenate stored slots into live Bot messages, invent absent-slot fallback behavior, or create new delivery events until that contract is explicit.

### Stage 9D — Proxy Node test/health/status — BLOCKED ON RUNTIME CREDENTIAL CONTRACT

The roadmap requires Node test/status/health, but CP-038/039 intentionally persist no Node token/secret/password and current Telemt authentication comes from a protected runtime token file for the single global Telemt target. Repository/product evidence does not yet define the per-Node credential source or whether `internal_api_endpoint` is specifically a Telemt API base URL. Do not probe Nodes or invent credential storage/transport semantics until that contract is explicit.

### Stage 7D2C — Exactly-once referral reward issuance — BLOCKED ON PRODUCT SEMANTICS

Roadmaps define reward amount/expiry and invitee eligibility conditions but do not specify who receives the Credit Bucket: inviter, invitee, or both. Do not infer a recipient.

### Stage 7D3B — Remaining minimum anti-abuse controls — BLOCKED ON PRODUCT SEMANTICS

Roadmap requires configurable daily/weekly caps, cooldowns, blacklist and suspicious score, but does not define cap scope/defaults, cooldown semantics/default, blacklist subject or score inputs/threshold. Do not invent those contracts.

## Important decisions

- Credit Buckets are source of truth; Telemt is only enforcement projection.
- Bot token/webhook secret remain protected runtime files, never ordinary SQLite settings.
- Forced Join gates gift and provisioning; referral reward recipient remains unresolved.
- Sponsor persistence remains independent of assignment routing; current global Telemt topology remains unchanged.
- User surfaces do not infer schema-absent Telegram username, Node/Sponsor assignment or general last activity.
- Start Gift configuration affects only future exactly-once grants.
- Bot Content remains literal text overrides only. Absent slots stay absent; runtime defaults, templates/formatting, buttons/emoji and Bot delivery wiring are separate contracts.
- Audit Log primitives remain caller-driven and append-only; snapshot serialization/redaction is not guessed by the persistence layer.
- Audit Log Admin API is read-only; mutation logging/redaction/actor propagation is a separate contract and must not be inferred.

## Validation/failure log

- No reset/force-push/clean was used for repository recovery or milestone repair.
- CP-046 promotion docs `fc7eae47...` CI `34543545636` PASS.
- CP-047 initial `f9b5ad04...` CI `34543744898` FAIL only in stale migration-count assertions; forward repair `74071dfa...` → `88465ade...`; final CI `34543883064` PASS.
- CP-047 promotion docs `b4468969...` CI `34544137155` PASS.
- CP-048 candidate `10dbde63...` CI `34544396136` PASS across full Go and installer/Docker/Telemt E2E validation.
- CP-048 promotion docs `b6e7dc81...` CI `34544624549` PASS.
- CP-049 candidate `8588193e...` CI `34544809255` PASS across full Go and installer/Docker/Telemt E2E validation.
- CP-049 promotion docs `d2cef37c...` CI `34567507147` PASS.
- CP-050 candidate `4a52a886...` CI `34567969934` PASS across Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun.
- CP-050 promotion docs `e2927d63...` CI `34568207035` PASS.
- CP-051 candidate `89c147f0...` CI `34573865558` PASS across Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites and Telemt E2E/rerun.
- Local clone for CP-050 targeted tests was unavailable because the container could not resolve github.com; staged Go files were gofmt-clean and production `store.go` compile-checked locally before atomic publication, then full GitHub CI supplied the authoritative verification.
- One initial 11F `create_tree` connector call was tool-blocked before any branch move; retry succeeded with the same three staged blobs. No repository state was changed by the blocked call.

## Current next action

Verify the CP-051 promotion docs-head CI. Then implement only Stage 11K Web Panel read-only Audit Log surface. Keep audit write endpoints, automatic mutation logging/snapshot serialization/redaction, Bot runtime delivery/wiring, fallback/default copy, templates/placeholders/formatting, button labels/emoji, secrets, reward issuance, unresolved anti-abuse policy, Node runtime probing, multi-Telemt routing and installer/compose behavior unchanged.
