# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `c67874de95b2ca4ff1786ffbd349fb091f270647`

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

Implemented:
- Telemt `3.5.7` exact release artifacts with checksum verification
- amd64 musl digest `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl digest `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`
- non-root Telemt container and liveness healthcheck
- public/configurable MTProto port but no host publication of API `9091`
- persistent 256-bit Telemt Bearer token outside install state/output
- generated protected Telemt config
- disabled internal bootstrap user to satisfy Telemt's non-empty-users invariant
- Proxy state/port/config handling and `tproxy proxy` management
- E2E checks API host isolation, auth, token non-leakage, proxy health and rerun persistence

Architecture note: the project bridge is not Docker `internal:true` because Telemt needs outbound Telegram connectivity. API isolation is no host publication plus Bearer authentication.

### Stage 6B — Control Plane Telemt client adapter — COMPLETE

Verified commit: `c67874de95b2ca4ff1786ffbd349fb091f270647`
CI `34438961960` PASS.

Implemented:
- focused `internal/telemt` Go client
- protected Bearer-token file loading without token logging
- bounded HTTP timeout and response-body limit
- safe typed health states: `healthy`, `unauthorized`, `unavailable`, `invalid_response`, `not_configured`
- Telemt URL/token-file pair validation
- client wiring without startup network dependency
- authenticated `GET /api/system/proxy` admin status surface
- `/readyz` remains independent of Telemt health
- auth, timeout, malformed-response and non-leakage tests

### Stage 6C — Proxy user lifecycle slice — ACTIVE

Scope for this milestone only:
- define persistent Control Plane proxy-user records separate from admin users
- extend `internal/telemt` with typed create/list/enable/disable/rotate-secret operations
- expose authenticated admin JSON endpoints for create/list/enable/disable/rotate
- never persist or log plaintext MTProto secrets after initial return
- represent reconciliation state so a Telemt outage does not corrupt the desired DB state
- keep disabled Telemt bootstrap user internal and excluded from Control Plane user listings

Acceptance:
- creating a proxy user stores desired metadata and provisions Telemt when available
- list returns Control Plane users without secrets
- enable/disable updates desired state and Telemt idempotently
- rotate returns a new secret once and does not persist plaintext
- Telemt unavailable paths return a safe classification while preserving recoverable desired state
- unauthenticated lifecycle requests are rejected
- Go format/vet/test and existing Docker/Telemt installer E2E stay green

### Stage 6D — Quota/expiry reconciliation — PENDING

After the lifecycle slice is verified, add expiry/quota policy and reconciliation separately before broader referral/bot features.

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
- Recovery point: inspect every commit/file after CP-007. If Stage 6C has started, repair/finish it before quota, referral, bot, or UI expansion work.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt remains an external dependency integrated via authenticated Control API; its source is not copied into Teleproxy.
- Pin Telemt versions/checksums; never follow unbounded `latest` for production install.
- Telemt 3.5.7 rejects empty `[access.users]`; generated disabled bootstrap user is temporary until real user lifecycle exists.
- Telemt API must not be host-published.
- Telemt outage must not take down Control Plane readiness.
- SQLite remains single-connection in MVP so connection-scoped PRAGMAs cannot silently disappear.
- Initial admin password uses a protected bootstrap file and is displayed only once after verified install.
- Random Panel port is not a security boundary; TLS hardening remains required for production public exposure.
- Proxy user secrets must be treated as reveal-once credentials; desired metadata belongs in SQLite, plaintext secrets do not.

## Validation / failure log

- 2026-09-10: Stage 2 CI `34419759826` PASS.
- 2026-09-10: Stage 3 CI `34420104043` PASS.
- 2026-09-10: Stage 4 CI `34420655852` PASS; CP-004.
- 2026-09-10: Stage 5 `c1cde407...`, CI `34426475546` failed because bootstrap secret was `0640`; kept strict backend validation and changed it to UID 10001 + `0600`.
- 2026-09-10: Stage 5 `34b455b0...`, CI `34426870772` failed because E2E omitted temporary `TPROXY_INSTALL_DIR`; production unchanged.
- 2026-09-10: Stage 5 `457f5278...`, CI `34427021157` PASS; CP-005.
- 2026-09-10: Stage 6A `39349dcf...`, CI `34437891406` reached healthy Telemt but E2E used an unreliable `docker compose port` no-binding assertion; changed to direct Docker `HostConfig.PortBindings` inspection.
- 2026-09-10: Stage 6A `91722c95...`, CI `34438040249` then failed because shell redirection read protected token before `sudo`; token stayed `0600`, harness changed to `sudo cat`.
- 2026-09-10: Stage 6A `47a95349...`, CI `34438212903` PASS; CP-006.
- 2026-09-10: canonical source files re-materialized and SHA-256 values matched the original recorded hashes. Exact repository mirror still blocked by connector large-file transport; no partial files committed.
- 2026-09-10: Stage 6B candidate `0fbb8d72...`, CI `34438839369` failed only at format check; no tests ran.
- 2026-09-10: gofmt-only repair `c67874de...`, CI `34438961960` PASS for Go and installer/Telemt E2E; CP-007.

## Current next action

Implement Stage 6C as a bounded lifecycle slice from CP-007. If interrupted, inspect every file after CP-007, repair Stage 6C first, and only promote after Go plus existing Docker/Telemt E2E are green.
