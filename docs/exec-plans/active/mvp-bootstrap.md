# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `523d13000bd26f1eeb1f4bc3ad6a40a3ca0ea57b`

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

### CP-028 implemented

- missing-channel Bot responses now use inline keyboards with one validated Telegram join URL button per missing channel and a fixed `forced_join_recheck` callback button; callback data contains no user/resource/entitlement or secret data and stays below Telegram's 64-byte limit.
- Bot client supports bounded `sendMessage` with inline keyboard and `answerCallbackQuery`; transport/network/API failures reuse sanitized error codes and never expose token-bearing Bot API URLs or upstream response descriptions.
- `Update` parses only the callback-query fields required for manual recheck.
- recheck accepts only the fixed callback token from a non-bot user, with an existing Teleproxy identity and a private callback message whose `chat.id` equals `callback_query.from.id`; forged/malformed/group/inline-only/unknown-user contexts perform no gift/provisioning mutation.
- recheck reuses the CP-026 Forced Join gate and the idempotent `EnsureStartGift`/CP-023 provisioning path; repeated callbacks cannot create a second start gift or duplicate provisioning ownership.
- handled callbacks are answered on success, still-missing membership, and membership/application-error paths so Telegram clients do not remain loading; callback application errors return webhook success and remain fail-closed for entitlement so users retry explicitly rather than through ambiguous Telegram update replay.
- ordinary blocked `/start` also renders join/recheck controls; legacy senders without inline-keyboard capability retain the prior plain-text fallback.

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

### Stage 7D1A — Referral identity + pending attribution domain — ACTIVE

Roadmap basis: `/start` resolves a referral code after creating a new Telegram user and before Forced Join; a valid referral requires a new invitee, no prior reward for that invitee, no self-referral, anti-abuse approval, and completed Forced Join. The roadmap also requires a unique constraint per invitee, idempotent reward creation, referral history, and a user-visible referral link/count.

Scope only:
- additive migration for one stable opaque referral code per Telegram user and at most one referral attribution per invitee
- generate referral codes with cryptographic randomness, bounded URL-safe characters, and a DB unique constraint; never derive security-sensitive authority from the code
- attribution records inviter/invitee and starts as `pending`; this milestone creates no Credit Bucket and does not decide reward eligibility
- accept attribution only for a newly resolved invitee; reject self-referral and unknown/invalid codes with typed safe outcomes; replay must never change an existing invitee attribution
- preserve referral history/status fields needed for later `pending -> rewarded/rejected` transitions without implementing those transitions yet
- expose read primitives for referral code, attribution, and inviter referral count suitable for later Bot response wiring
- no reward issuance, Forced Join integration, caps/cooldowns/blacklist scoring, admin UI/API, Bot callback changes, or sponsor work in D1A

Acceptance:
- one code per inviter and code uniqueness are enforced under replay/concurrency
- one attribution per invitee is enforced by schema/domain; self-referral cannot be persisted
- only `ResolveResult.Created=true` callers can create a new attribution through the public API
- invalid/unknown referral payloads cannot mutate attribution state
- pending attribution survives the Forced Join gap so a later callback can finalize eligibility without carrying referral data in callback_data
- referral count excludes pending/rejected records until later reward completion semantics are implemented
- migration upgrade/rerun, format/vet/test and Docker/Telemt E2E remain green

### Stage 7D1B — `/start` referral resolution wiring — PENDING
Parse the roadmap referral payload before Forced Join, persist pending attribution only for a newly-created invitee, keep callback data fixed/non-secret, and surface the user's stable referral link/count without issuing a reward yet.

### Stage 7D2 — Referral eligibility + idempotent reward Credit Bucket — PENDING
Finalize pending referral only after Forced Join/anti-abuse checks. Implement exactly-once reward Credit Bucket, configurable byte amount/expiry and typed rejected reasons. The supplied roadmap states a default `2GB / 14 days` referral reward but does not explicitly say whether that bucket belongs to inviter, invitee, or both; resolve this from repository/product evidence before issuing credits rather than silently guessing.

### Stage 7D3 — Minimum anti-abuse controls + admin configuration/history — PENDING
Add the roadmap-required configurable caps/cooldowns/blacklist/history surfaces in small verified milestones. Suspicious scoring remains explicit and server-side, never a client-only decision.

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
- C2B isolated local tests covered interactive transport/webhook behavior; production compile used type-compatible stubs because local Go is 1.23.2 while repository `go.mod` is Go 1.26. GitHub CI is authoritative for the full suite.

## Current next action

Implement only Stage 7D1A from CP-028. Do not wire referral payloads into `/start` or issue any referral reward until D1A is separately verified.
