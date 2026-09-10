# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `80b02d1183edfb65163b7b7a5dfe1b3834a2de9a`

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

### CP-034 implemented

- authoritative referral history is exposed through a bounded domain query over `referral_attributions`, joined only to existing Telegram identities for administrator-facing inviter/invitee identification.
- history preserves stored attribution status, rejection reason, eligibility/finalization timestamps and created/updated timestamps; it derives no abuse score and exposes no referral code authority or secrets.
- pagination is deterministic newest-first by attribution ID using optional positive `before_id`; default page size is 50, maximum is 100, and the query reads at most `limit + 1` rows to determine whether another page exists.
- authenticated `GET /api/referral/history` reuses the existing Admin session contract, returns `Cache-Control: no-store`, and rejects invalid/repeated/unknown pagination parameters with a safe typed problem.
- history reads do not mutate referral attributions, settings, or Credit Buckets.
- the final D3A diff from the CP-033 docs head contains exactly five files: two referral history domain/test files, two HTTP history/test files, and one route-registration line in `referral_settings.go`.
- all four new source/test blobs were byte-verified against locally gofmt-clean Git blobs before CI; no transfer corruption occurred.
- candidate `80b02d1183edfb65163b7b7a5dfe1b3834a2de9a` passed Format, Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and Telemt E2E/rerun in CI `34508958312`.

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

### Stage 8A — Sponsor Profile persistence/domain primitives — ACTIVE

Roadmap-confirmed Sponsor Profile fields are ID, name, channel username/link, `ad_tag`, enabled, weight, assigned nodes, start time, end time, notes and statistics metadata. Multiple Sponsor Profiles must be supported concurrently. `ad_tag` must be exactly 32 hexadecimal characters and duplicate/invalid configurations must be prevented.

Scope only:
- inspect existing node/schema/domain patterns first, then add the smallest additive Sponsor Profile persistence/domain primitive with explicit validation for name/channel reference/ad tag/enabled/weight/time window/notes;
- preserve future assignment modes without prematurely implementing Global/Per Node/Per User/Weighted/Manual/Campaign routing semantics;
- do not wire Telemt per-user ad tags, node compatibility, Middle Proxy enforcement, sticky assignment, statistics aggregation, Web/Bot Admin UI, or Sponsor mutation audit in this first milestone;
- schema must remain additive/backward-compatible and support multiple profiles.

Acceptance:
- valid Sponsor Profiles can be created/read/listed with deterministic ordering;
- `ad_tag` is normalized/validated as exactly 32 hexadecimal characters and duplicates fail safely;
- invalid time windows/weights/channel references fail before mutation;
- no existing proxy/referral/quota behavior changes;
- format/vet/test and Docker/Telemt E2E remain green.

### Stage 7D2C — Exactly-once referral reward issuance — BLOCKED ON PRODUCT SEMANTICS

Unresolved product contract:
- the supplied English and Persian roadmaps define reward amount/expiry and invitee eligibility conditions but do not specify who receives the reward Credit Bucket: inviter, invitee, or both.
- repository issue/code search found no product decision resolving that recipient as of CP-034.
- do not infer a recipient from common referral conventions. Reward issuance remains blocked until this contract is explicit.

### Stage 7D3B — Remaining minimum anti-abuse controls — BLOCKED ON PRODUCT SEMANTICS

The roadmap requires configurable daily/weekly reward caps, cooldowns, blacklist, suspicious-score mechanism and admin mutation audit log. It does not specify cap scope/default values, cooldown semantics/default, blacklist subject, or suspicious-score inputs/threshold. Do not silently invent those product semantics. Referral history is complete at CP-034; remaining policy-bearing controls stay blocked until their contracts are explicit.

## Important decisions

- Credit Buckets are source of truth; Telemt is only enforcement projection.
- Telemt disable cancels active sessions; quota `0` blocks traffic.
- Bot token and webhook secret are protected runtime files, never ordinary SQLite settings.
- Telemt user view reconstructs proxy links from Telemt-managed secret; Control Plane does not persist MTProto plaintext.
- Telegram `getChatMember` failures fail closed because membership for another user depends on correct Bot/channel permissions.
- Forced Join gates gift and provisioning, not just link visibility; identity may exist with zero credit.
- Forced Join management reuses authoritative domain validation.
- Manual recheck callback data is fixed/non-secret; authoritative identity always comes from the Telegram update, not callback data.
- Referral attribution must be durable before the Forced Join gap because recheck callback data intentionally carries no referral identity.
- Production referral links reuse configured `TPROXY_BOT_USERNAME`; no extra token or runtime identity source is needed.
- Eligibility approval is distinct from reward settlement: `pending + eligible_at` is approved but not rewarded.
- Referral credit recipient semantics remain unresolved; do not issue referral rewards until repository/product evidence resolves inviter versus invitee versus both.
- Anti-abuse controls are roadmap-required, but unspecified scope/default semantics are not inferred.
- Sponsor Profile persistence may be implemented before assignment routing; assignment modes, sticky allocation, Telemt projection, compatibility and Middle Proxy enforcement remain separate later milestones.

## Validation/failure log

- Stage 5 secret-mode/E2E harness failures repaired without weakening production validation.
- Stage 6A port assertion/protected-token harness failures repaired.
- Stage 6B format-only failure repaired at CP-007.
- CP-012 superseded an otherwise-passing candidate after self-review found a missing future-start boundary.
- CP-013 repaired a partial migration-count publication without reset/force.
- C3A in-memory SQLite test was incompatible with required WAL; production unchanged.
- C3B misspelled HTTP constant caused vet failure; repaired.
- 7A accidental README intermediate commit was neutralized by restoring the exact prior blob in the next fast-forward, no reset/force.
- 7B2 local test caught body-cap ambiguity before publish; oversized bodies now deterministically return 413.
- 7B3A format-only transfer failure and 7B3B unused test import were repaired; production semantics unchanged.
- CP-024 hardened Telegram join URLs against explicit ports before publish.
- CP-025 pre-publish compile caught unsupported `testing.T.Context`; repaired before branch publication.
- C2A produced two unattached mismatched Git blobs during transfer checks; neither was referenced by a tree/branch. Final six published SHAs matched local staging and CI `34482353102` passed.
- C2B was published through seven coherent sequential fast-forward commits after an accidental direct `create_file` publication of `interactive.go`; no reset/force was used. Net diff from CP-027 contains exactly seven Telegram Bot files, and final CI `34490204497` passed.
- C2B isolated local tests covered interactive transport/webhook behavior; production compile used type-compatible stubs because local Go is 1.23.2 while repository CI is authoritative for the full suite.
- D1A intermediate schema commit `1efded4b6b583678637b3949b152bc2ab38c82ec` had Format/Vet PASS but `go test` failure in CI `34497018395`; subsequent migration-test update and D1A tests produced final candidate `989c32c234127e93879719a566046144201e6d18`, whose full CI `34497165310` passed.
- D1B candidate `caf9bffd275ed2418b540773042875b8fe2d23fa` passed full CI `34497967471`.
- CP-030 docs promotion `2ad01cb8bce7b6d489c62e8594cebe99ea1774cf` passed full CI `34498484861`.
- D2A commit `5f96316f7ea58581dce0d8f8a312518fe595a42b` passed full CI `34505475099` on the first candidate.
- CP-031 docs promotion `9f35a3ece67dd32717af1803a7b3c0fac03624d5` passed full CI `34505782176`.
- D2B candidate `06dc77e0b18517179e0dea937a0612ed54bc356e` passed full CI `34505999351` on the first candidate; CP-032 docs promotion `8f268d0b6cfc336e91742383d991251ac49e1e9a` passed full CI `34506301685`.
- D2B2 initial candidate `e50d7b299bd0793ed10f8764f8113f77eae09215` failed only the Format check in CI `34506528551`; Vet/Test and installer were skipped. Byte-level comparison showed `internal/httpapi/referral_settings_test.go` did not match the locally gofmt-clean source because raw JSON literals had been transferred with extra backslashes. No production source semantics were implicated. Fast-forward repair `7fd1143723532a9d9a7ff5591c914b966b7e129f` restored the exact local test blob SHA and passed full CI `34506809233`.
- CP-033 docs promotion `09fe0924194b644306d49f3685ccb0d638af2690` passed full CI `34507182842`.
- D3A was published as five coherent fast-forward commits ending at `80b02d1183edfb65163b7b7a5dfe1b3834a2de9a`; the net diff contains exactly the referral history domain/API scope, all new blobs matched local gofmt-clean Git hashes, and final full CI `34508958312` passed including Docker/Telemt E2E/rerun.

## Current next action

Resume from CP-034. First inspect existing node/schema/domain patterns on the actual branch HEAD, then implement only Stage 8A Sponsor Profile persistence/domain primitives from the supplied roadmap. Keep referral reward issuance and policy-bearing anti-abuse controls blocked until their product semantics are explicit.