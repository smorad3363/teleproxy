# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `7560b7b17c288b9789e98d8f790bbb10dd79a21d`

## Purpose and recovery

Build Teleproxy incrementally from the supplied roadmap. On interruption, start from this file, compare branch head with `Latest verified checkpoint`, inspect every later commit/file/CI result, repair the active partial milestone, then continue. Never reset/clean/force over unrelated work.

Non-negotiable: Control Plane and Telemt lifecycles remain independent; SQLite WAL/NORMAL is authoritative Control Plane state; Credit Buckets are authoritative quota/reward state; Telemt quota/expiry is only an enforcement projection; plaintext admin/API/Bot/webhook/MTProto secrets are never logged or persisted by Control Plane; DB changes are additive/backward compatible.

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

### CP-023 implemented

- `/start` provisions the mapped Telemt user disabled, never directly enables it, then queues the verified quota reconciler.
- Provisioning is serialized per username in-process and guarded durably by CP-022 CAS journal state.
- A pre-existing same-name Telemt user is adopted only when a prepared secret digest is proved from validated Telemt links; mismatches become durable collision state.
- Ambiguous create failures recover by GET+proof without a second create. A definitively deleted previously-owned user can be safely reprovisioned with a new in-memory secret/digest.
- Historical provisioning digest is not revalidated after intentional admin secret rotation; owned users continue by Control Plane identity and current Telemt view.
- Only validated proxy links are returned transiently to Telegram; no plaintext secret/link is stored in SQLite.
- Bot-enabled startup requires configured Telemt + quota runner; default installation without Bot remains unaffected.

## Supplied source hashes

Rechecked from mounted originals on 2026-09-10:
- AGENTS: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- ROADMAP_EN: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- ROADMAP_FA: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

Stage 1 root exact-byte mirrors remain PARTIAL because prior GitHub transport could not safely preserve the large supplied files. Do not commit partial/corrupted mirrors.

## Pinned Telemt

Telemt `3.5.7`, upstream commit `4ca7418442478cd92f9e861c21977a81b249efc8`.
- amd64 musl SHA-256 `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl SHA-256 `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`

## Active stage

### Stage 7C1 — Forced Join domain + membership-gated start — ACTIVE

Roadmap basis: `/start` resolves Telegram identity, checks Forced Join before initial gift/proxy assignment, and Forced Join can contain one or more configurable Telegram channels.

Scope only:
- additive required-channel schema/store with deterministic ordering, enabled state, Telegram chat reference, display name and safe join URL; no Bot token/secrets in DB
- add bounded `getChatMember` support to the existing Telegram client; accept `creator`, `administrator`, `member`, and `restricted` only when `is_member=true`; treat `left`/`kicked` as not joined; malformed/unknown status is fail-closed
- because Telegram guarantees `getChatMember` for other users only when the Bot is an administrator, membership API errors must never bypass Forced Join
- refactor Telegram start domain compatibly into `Resolve identity` and idempotent `EnsureStartGift`; existing `telegramuser.Start` remains a compatibility wrapper
- gated Bot flow: resolve identity -> check every enabled required channel -> if any missing/unverifiable, do not grant initial gift and do not provision proxy -> after successful recheck, grant the gift exactly once and continue CP-023 provisioning
- return structured missing-channel data for user messaging; no broad Admin UI or referral logic in C1

Acceptance:
- a new non-member can create durable Telegram identity/proxy mapping but receives no Credit Bucket and no Telemt provisioning
- after joining, replay grants exactly one start gift and provisions normally
- member/admin/creator and restricted+is_member pass; left/kicked/restricted+false fail
- Telegram API timeout/unauthorized/malformed result fails closed and does not grant/provision
- multiple required channels all must pass; disabled channels are ignored
- existing pre-Forced-Join users/gifts remain valid and migrations upgrade cleanly
- format/vet/test and Docker/Telemt E2E remain green

### Stage 7C2 — Forced Join configuration + manual recheck UX — PENDING
Add authenticated admin CRUD for required channels plus Telegram join/recheck controls/messages. Keep callbacks authenticated by Telegram webhook context and make repeated recheck idempotent.

### Stage 7D — Referrals + rewards — PENDING
Unique-per-invitee attribution, self-referral protection, idempotent reward Credit Buckets, configurable reward/expiry/caps, and Forced-Join eligibility integration.

## Important decisions

- Credit Buckets are the source of truth; Telemt is only enforcement projection.
- Telemt disable cancels active sessions and quota `0` blocks traffic.
- Bot token and webhook secret are protected runtime files, never ordinary SQLite settings.
- Telemt user view reconstructs proxy links from Telemt-managed secret; Control Plane does not persist MTProto plaintext.
- Telegram `getChatMember` for another user is only guaranteed when the Bot is an administrator in the target chat/channel; Forced Join therefore fails closed on API/configuration errors.
- Forced Join must gate gift/provisioning, not merely hide the final link. Existing `telegramuser.Start` will remain as a compatibility wrapper while the Bot path uses the split gated flow.

## Validation/failure log

- Stage 5 `c1cde407...` secret mode and `34b455b0...` E2E harness failures repaired without weakening production validation.
- Stage 6A `39349dcf...` port assertion and `91722c95...` protected-token harness repaired.
- Stage 6B `0fbb8d72...` format-only failure repaired at CP-007.
- Stage 6D2B1 `86090aba...` passed but was superseded after self-review found missing future-start boundary; CP-012 repaired it.
- Stage 6D2B2A `44b3dc76...` partial migration publication caused migration-count failure; final CP-013 used no reset/force.
- C3A `8040d4e3...` test used in-memory SQLite incompatible with required WAL; production unchanged.
- C3B `470ecb81...` misspelled HTTP constant caused vet failure; repaired.
- 7A intermediate `bcac80e9...` accidentally touched README; exact prior blob restored in next fast-forward, no reset/force.
- 7B2 local test caught body-cap ambiguity before publish; oversized bodies now deterministically return 413.
- 7B3A `f52aa0b6...`, CI `34469352217`: format-only test transfer failure; repair `66353eee...` passed.
- 7B3B `264c1f1f...`, CI `34472413695`: unused test import after safe test-file split caused vet failure; production unchanged. Repair `7560b7b1...`, CI `34472531024`, PASS.
- An unattached malformed binary Git blob and an unattached dry webhook-diff commit were created during safe publishing checks; neither was ever referenced by the branch and neither affects repository state.
- Full local Go suite remains unavailable because external module DNS is blocked in the container; GitHub CI is authoritative.

## Current next action

Implement only Stage 7C1 from CP-023. Do not start admin configuration UX, callbacks, or referrals until C1 is separately verified.
