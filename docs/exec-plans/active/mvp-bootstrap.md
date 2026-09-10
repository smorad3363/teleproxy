# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`

## Purpose

Build the first recoverable foundation of Teleproxy from an empty repository, following the supplied engineering contract and project roadmap.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane remain lifecycle-independent.
- Go + lightweight HTTP + embedded UI + SQLite for the Control Plane.
- `telemt` is an external data-plane component; do not copy its source into this project.
- Installation is Docker-first and safe to rerun.
- First installation selects a random available high port for the Web Panel, persists it, and does not silently change it on rerun.
- Installer final output shows Panel URL, Panel port, admin username, generated initial credential when needed, component health, version/ref, data/config paths, and `tproxy` command.
- Telemt management API must not be published publicly; keep it on a private project network and authenticate requests.
- Never mount `/var/run/docker.sock` into the Web App.
- No plaintext passwords, bot tokens, session secrets, MTProto secrets, API bearer values, or private keys in logs/state.
- Substantial work must be recoverable from this file plus Git state without relying on chat history.

## Recovery protocol

When resuming after interruption:

1. Read `AGENTS.md` when present, `docs/ARCHITECTURE.md`, the relevant subsystem document, and this execution plan.
2. Inspect branch head and compare it with `Latest verified checkpoint` and any active stage candidate.
3. Inspect every file changed after the verified checkpoint.
4. Re-run or inspect targeted validation for the partially completed milestone.
5. If the last write was partial, failed CI, or inconsistent, repair that milestone before starting a new one.
6. Do not restart from scratch and do not overwrite unrelated changes.

## Stage policy

Each stage is intentionally small and independently verifiable. Do not broaden a stage while implementing it.

### Stage 1 — Repository foundation — PARTIAL

Completed:
- active repository-local recovery plan
- architecture/security/reliability documents
- isolated task branch
- CI with Go and installer E2E
- exact source files re-materialized locally and SHA-256 re-verified

Remaining:
- commit the supplied canonical `AGENTS.md`, `ROADMAP_FA.md`, and `ROADMAP_EN.md` at repository root

Verified source hashes:
- `AGENTS.md`: `4a0c4156f14c3a40fbf2c6da8937f36c9f7f15f694bc3895181b13ccc0984483`
- `ROADMAP_EN.md`: `90605c0e08bd960b02d4569e49995dd55c2ece1fb37ca6a09d37c66350114009`
- `ROADMAP_FA.md`: `a9219b597eac4a2d9c73f5ae1013b25a3e15a862266665fcf5de3e2aae174fab`

Current GitHub connector does not accept a local file argument for blob writes and large base64 output is truncated by the tool response. Do not create partial mirrors; the conversation source files remain authoritative until an exact-byte upload path is available.

### Stage 2 — Minimal Control Plane — COMPLETE

Implemented validated HTTP config, graceful HTTP server, `/healthz`, `/readyz`, Problem Details and tests.
CI `34419759826` PASS.

### Stage 3 — Core persistence + admin bootstrap — COMPLETE

Verified commit: `5798075de40d2f966d8546a18e6fa450d7142f90`
CI `34420104043` PASS.

Implemented SQLite WAL/foreign keys/busy timeout/NORMAL sync, migrations, owner bootstrap, PBKDF2-SHA256 password hashing and hashed session-token persistence.

### Stage 4 — Minimal Web login — COMPLETE

Verified commit: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`
CI `34420655852` PASS.

Implemented protected one-time owner bootstrap, DB-backed sessions, login/dashboard, secure cookie flags, CSRF, generic credential failures, login rate limiting and DB readiness.

### Stage 5 — Installer foundation — COMPLETE

Verified commit: `457f52783f9b2962c55be00beb801f0f2534958c`
CI `34427021157` PASS.

Implemented hardened Docker Control Plane, installer lock/state, random persistent Panel port, one-time admin password, safe rerun, final install summary, `tproxy`, and Docker E2E.

### Stage 6A — Pinned Telemt data plane — COMPLETE

Verified commit: `47a95349922ba5be37cf0ed8416b482de08580ef`
CI `34438212903` PASS.

Implemented Telemt `3.5.7` checksum-pinned non-root container, configurable public MTProto port, internal authenticated API with no host publication, protected generated config/token, disabled bootstrap user, proxy state/management, and E2E isolation/rerun checks.

### Stage 6B — Control Plane Telemt client adapter — COMPLETE

Verified commit: `c67874de95b2ca4ff1786ffbd349fb091f270647`
CI `34438961960` PASS.

Implemented protected token-file client construction, bounded HTTP handling, typed safe health states, config validation, authenticated `/api/system/proxy`, and Control Plane readiness independence from Telemt health.

### Stage 6C1 — Proxy user persistence + Telemt lifecycle client — COMPLETE

Verified commit: `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`
CI `34445848656` PASS.

Implemented:
- migration `002_proxy_users.sql` with desired enabled state and reconciliation state
- Control Plane proxy-user store separate from administrator identities
- no secret-bearing column in `proxy_users`
- pending/synced/error state transitions with bounded machine-only error codes
- typed Telemt create/list/enable/disable/rotate-secret client calls matching Telemt 3.5.7 API
- bounded response bodies and safe HTTP failure classification without upstream body leakage
- reveal-once returned MTProto secret validation; plaintext secrets are not persisted
- store/migration/client tests and existing installer/Telemt E2E regression checks

### Stage 6C2 — Admin proxy-user API + reconciliation — ACTIVE

Scope:
- authenticated JSON endpoints for Control Plane proxy-user create/list/enable/disable/rotate
- create writes desired DB record first, then provisions Telemt
- enable/disable writes desired state first, then reconciles Telemt
- Telemt failures mark only a bounded safe machine code in DB; desired state remains recoverable
- rotate is allowed only for an existing Control Plane user and returns the new secret once
- list is DB-first and never exposes secret material or the Telemt bootstrap user
- mutating JSON endpoints require session-bound CSRF protection

Acceptance:
- unauthenticated lifecycle requests return 401
- missing/invalid CSRF on mutations returns 403
- successful create returns one secret plus a synced DB user
- Telemt-unavailable create leaves a pending/error desired DB record without a secret and returns safe 503 metadata
- list contains no secret field/material
- enable/disable preserve desired DB state if Telemt is unavailable and mark safe sync error
- rotate never stores returned secret
- Go format/vet/test and existing Docker/Telemt E2E remain green

### Stage 6D — Quota/expiry reconciliation — PENDING

After lifecycle endpoints are verified, add expiry/quota policy and reconciliation separately before broader referral/bot features.

## Checkpoints

### CP-000 — repository initialized
- Commit: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`

### CP-001 — minimal Control Plane verified locally
- Commit: `b98507bbb3ef0a74e6e147286c22ab7d626f9e72`

### CP-002 — CI foundation verified
- Commit: `084d4a1256a6b28412d9457f6568a4138e09c83c`
- CI: `34419759826` PASS

### CP-003 — persistence/admin bootstrap verified
- Commit: `5798075de40d2f966d8546a18e6fa450d7142f90`
- CI: `34420104043` PASS

### CP-004 — secure admin web login verified
- Commit: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`
- CI: `34420655852` PASS

### CP-005 — Docker installer and rerun verified
- Commit: `457f52783f9b2962c55be00beb801f0f2534958c`
- CI: `34427021157` PASS

### CP-006 — pinned Telemt data plane verified
- Commit: `47a95349922ba5be37cf0ed8416b482de08580ef`
- CI: `34438212903` PASS

### CP-007 — authenticated Telemt health adapter verified
- Commit: `c67874de95b2ca4ff1786ffbd349fb091f270647`
- CI: `34438961960` PASS

### CP-008 — proxy-user persistence and lifecycle client verified
- Commit: `281f91afca202d0cc9a61e64fe88fe7ebbea5aab`
- CI: `34445848656` PASS
- Recovery point: inspect every commit/file after CP-008. If Stage 6C2 has started, repair/finish the admin lifecycle API before quota, referral, bot, or UI expansion.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt remains an external dependency integrated via authenticated Control API; its source is not copied into Teleproxy.
- Pin Telemt versions/checksums; never follow unbounded `latest` for production install.
- Telemt 3.5.7 rejects empty `[access.users]`; generated disabled bootstrap user remains internal until real lifecycle users exist.
- Telemt API must not be host-published.
- Telemt outage must not take down Control Plane readiness.
- SQLite remains single-connection in MVP so connection-scoped PRAGMAs cannot silently disappear.
- Initial admin password uses a protected bootstrap file and is displayed only once after verified install.
- Random Panel port is not a security boundary; TLS hardening remains required for production public exposure.
- Proxy user secrets are reveal-once credentials; desired metadata belongs in SQLite, plaintext secrets do not.
- Reconciliation is desired-state-first: a data-plane outage may delay convergence but must not lose the administrator's intended state.

## Validation / failure log

- 2026-09-10: Stage 2 CI `34419759826` PASS.
- 2026-09-10: Stage 3 CI `34420104043` PASS.
- 2026-09-10: Stage 4 CI `34420655852` PASS; CP-004.
- 2026-09-10: Stage 5 `c1cde407...`, CI `34426475546` failed because bootstrap secret was `0640`; strict validation retained, ownership/mode fixed.
- 2026-09-10: Stage 5 `34b455b0...`, CI `34426870772` failed because E2E omitted temporary `TPROXY_INSTALL_DIR`; production unchanged.
- 2026-09-10: Stage 5 `457f5278...`, CI `34427021157` PASS; CP-005.
- 2026-09-10: Stage 6A `39349dcf...`, CI `34437891406` failed on an unreliable no-binding test; replaced with direct Docker PortBindings inspection.
- 2026-09-10: Stage 6A `91722c95...`, CI `34438040249` failed because shell redirection read a protected token before sudo; token remained `0600`, harness changed to `sudo cat`.
- 2026-09-10: Stage 6A `47a95349...`, CI `34438212903` PASS; CP-006.
- 2026-09-10: canonical source hashes were re-verified; exact repository mirror remains blocked by connector large-file transport and no partial mirrors were committed.
- 2026-09-10: Stage 6B `0fbb8d72...`, CI `34438839369` failed only at format check.
- 2026-09-10: gofmt-only repair `c67874de...`, CI `34438961960` PASS; CP-007.
- 2026-09-10: Stage 6C1 Telemt client isolated tests PASS locally; repository Go minimum remains validated in CI Go 1.27.1.
- 2026-09-10: Stage 6C1 `281f91af...`, CI `34445848656` PASS for format/vet/test plus installer/Telemt E2E; CP-008.

## Current next action

Implement Stage 6C2 from CP-008 as a coherent authenticated API/reconciliation slice. If interrupted, inspect all commits/files after CP-008 and repair 6C2 before starting quota/expiry work.
