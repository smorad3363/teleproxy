# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `b16c08a5adcfdcf9a3ddaba9b6d9e5d85ab862c0`

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

### CP-037 implemented

- authenticated `/sponsors` is registered through the production Sponsor route set and redirects unauthenticated Web requests to `/login` using the existing session behavior.
- the server-rendered page lists canonical Sponsor Profiles, safely escapes user-controlled fields with `html/template`, uses `Cache-Control: no-store`, and exposes the session-derived CSRF token only inside the authenticated page.
- create/edit/delete controls call the existing CP-036 `/api/sponsors` endpoints with same-origin credentials and `X-CSRF-Token`; Sponsor validation/business rules are not duplicated in page handlers.
- the existing dashboard links to Sponsor management; no frontend framework was introduced.
- the final 8C net diff from the CP-036 docs head contains exactly four files: one Sponsor page handler/template, its tests, one route-registration line, and the dashboard link/template adjustment.
- no Node assignment, weighted/sticky allocation, Campaign routing, Telemt ad-tag projection, Node compatibility/Middle Proxy enforcement, Sponsor statistics, Bot Admin Sponsor UI or Sponsor audit mutation was added.
- candidate `b16c08a5adcfdcf9a3ddaba9b6d9e5d85ab862c0` passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and Telemt E2E/rerun in CI `34518089493`.

## Supplied source hashes

Rechecked from mounted originals on 2026-09-10:
- AGENTS: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- ROADMAP_EN: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- ROADMAP_FA: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

Stage 1 exact-byte root mirrors remain PARTIAL because prior GitHub transport could not safely preserve the large supplied files. Do not commit partial/corrupted mirrors.

## Pinned Telemt

Telemt `3.5.7`, upstream commit `4ca7418442478cd92f9e861c21977a81b249efc8`.
- amd64 musl SHA-256 `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl SHA-256 `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`

## Active stage

### Stage 9A — Proxy Node identity/config persistence primitives — ACTIVE

Roadmap basis: Web Panel explicitly includes Nodes; each Node has identity/network/configuration fields; the schema should be Multi-Node/Relay-ready from day one; Sponsor Profiles later require Node assignment. The current runtime and compose topology still configure one global Telemt endpoint, so this milestone introduces authoritative Node metadata only and does not switch runtime routing.

Scope only:
- add the smallest additive `proxy_nodes` persistence/domain model for stable Node identity/configuration: ID, node type (`proxy`/`relay`), name, region, host/IP, public host, MTProto port, internal API endpoint, enabled, and creation/update timestamps;
- validate/canonicalize names/regions/hosts/ports/API endpoint without making network calls and without persisting API credentials or other secrets;
- support create/get/list with deterministic ordering and safe duplicate identity handling;
- keep live health/version/heartbeat/active-user/bandwidth fields out of authoritative static configuration until a telemetry contract exists;
- do not change `TPROXY_TELEMT_API_URL`, compose topology, quota reconciliation routing, installer behavior, Relay parent/tunnel runtime, Sponsor assignment, FakeTLS/connection-mode/session-limit semantics, Web/Bot Node UI or Node lifecycle actions in 9A.

Acceptance:
- multiple valid Proxy/Relay metadata records can coexist and round-trip deterministically;
- malformed host/public-host/port/internal API endpoint, invalid type/name/region and duplicate stable identity fail before unintended mutation;
- internal API endpoint rejects embedded credentials and fragments; no secret/token field exists in Node persistence;
- migration is additive/backward-compatible and prior data/migrations remain intact/idempotent;
- current single-Telemt runtime behavior remains unchanged;
- format/vet/test and Docker/Telemt E2E remain green.

### Stage 7D2C — Exactly-once referral reward issuance — BLOCKED ON PRODUCT SEMANTICS

The supplied English and Persian roadmaps define reward amount/expiry and invitee eligibility conditions but do not specify who receives the reward Credit Bucket: inviter, invitee, or both. Repository evidence has not resolved this. Do not infer a recipient.

### Stage 7D3B — Remaining minimum anti-abuse controls — BLOCKED ON PRODUCT SEMANTICS

The roadmap requires configurable daily/weekly reward caps, cooldowns, blacklist, suspicious-score mechanism and admin mutation audit log, but does not specify cap scope/default values, cooldown semantics/default, blacklist subject, or suspicious-score inputs/threshold. Do not invent those contracts.

## Important decisions

- Credit Buckets are source of truth; Telemt is only enforcement projection.
- Telemt disable cancels active sessions; quota `0` blocks traffic.
- Bot token and webhook secret are protected runtime files, never ordinary SQLite settings.
- Telemt user view reconstructs proxy links from Telemt-managed secret; Control Plane does not persist MTProto plaintext.
- Telegram `getChatMember` failures fail closed because membership for another user depends on correct Bot/channel permissions.
- Forced Join gates gift and provisioning, not just link visibility; identity may exist with zero credit.
- Manual recheck callback data is fixed/non-secret; authoritative identity always comes from the Telegram update, not callback data.
- Referral attribution must be durable before the Forced Join gap because recheck callback data intentionally carries no referral identity.
- Production referral links reuse configured `TPROXY_BOT_USERNAME`; no extra token or runtime identity source is needed.
- Eligibility approval is distinct from reward settlement: `pending + eligible_at` is approved but not rewarded.
- Referral credit recipient semantics remain unresolved; do not issue referral rewards until repository/product evidence resolves inviter versus invitee versus both.
- Sponsor Profile persistence is independent of assignment routing; assignment modes, sticky allocation, Telemt projection, compatibility and Middle Proxy enforcement remain separate later milestones.
- The existing Web Panel remains server-rendered/minimal; no frontend framework is introduced for isolated management surfaces.
- CP-037 still has no Node domain/table. Current config/compose use one global Telemt API endpoint/service, so Node metadata must land before multi-node runtime routing or Sponsor-to-Node assignment.

## Validation/failure log

- Prior format/test transfer failures were repaired without weakening production validation; no reset/force-push was used.
- D2B2 initial candidate `e50d7b299bd0793ed10f8764f8113f77eae09215` failed only Format because a transferred test raw string gained extra backslashes; repair `7fd1143723532a9d9a7ff5591c914b966b7e129f` restored the exact gofmt-clean blob and passed full CI.
- D3A final candidate `80b02d1183edfb65163b7b7a5dfe1b3834a2de9a` passed full CI `34508958312`.
- CP-034 docs promotion `ba131d11b62e394bedc3e03b1af267981a14d5a0` passed full CI `34509372354`.
- 8A candidate `e1df28fd16d59b3c08eff86ee6b23d579eaf199f` passed full CI `34510107359` on the first candidate.
- CP-035 docs promotion `55ee6d8128f4a63925ce54744c0cf7c98a73d3b8` passed full CI `34510460750`.
- 8B candidate `2abc393bf144b0e269d166a544f1df3277649ffa` passed full CI `34510993603` on the first candidate.
- CP-036 docs promotion `c1d93e7202a5afb34faf6f307ad92761e7f541b7` passed full CI `34517423137`.
- 8C candidate `b16c08a5adcfdcf9a3ddaba9b6d9e5d85ab862c0` passed full CI `34518089493` on the first candidate.

## Current next action

Resume from CP-037. Verify the CP-037 docs-head CI, then implement only Stage 9A additive Proxy Node identity/config persistence/domain primitives. Do not change current single-Telemt runtime routing, installer/compose, Sponsor assignment, Node telemetry/lifecycle actions, or blocked referral reward semantics in this milestone.
