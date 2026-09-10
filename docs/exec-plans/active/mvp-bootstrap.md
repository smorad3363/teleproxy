# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `47a95349922ba5be37cf0ed8416b482de08580ef`

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
2. Inspect branch head and compare it with `Latest verified checkpoint` and any stage candidate recorded below.
3. Inspect every file changed after the verified checkpoint.
4. Re-run or inspect targeted validation for the partially completed milestone.
5. If the last write was partial, failed CI, or inconsistent, repair that milestone before starting a new one.
6. Do not restart from scratch and do not overwrite unrelated changes.

## Stage policy

Each stage is intentionally small and independently verifiable. Do not broaden a stage while implementing it.

### Stage 1 — Repository foundation — PARTIAL

Completed:
- active repository-local recovery plan
- architecture boundaries
- security baseline
- reliability/recovery invariants
- isolated task branch
- CI with Go and installer checks

Remaining:
- commit the supplied canonical `AGENTS.md` at repository root
- mirror supplied `ROADMAP_FA.md` and `ROADMAP_EN.md` at repository root

The supplied conversation files are currently materialized locally and remain authoritative until mirrored exactly into the branch.

### Stage 2 — Minimal Control Plane — COMPLETE

Implemented validated HTTP config, graceful HTTP server, `/healthz`, `/readyz`, shared Problem Details and tests.
Validation: local gofmt/test/vet PASS; GitHub Actions run `34419759826` PASS.

### Stage 3 — Core persistence + admin bootstrap — COMPLETE

Verified commit: `5798075de40d2f966d8546a18e6fa450d7142f90`
CI run: `34420104043` PASS.

Implemented SQLite WAL/foreign keys/busy timeout/NORMAL sync, migrations, owner bootstrap, PBKDF2-SHA256 password hashing, hashed session-token persistence and tests.

### Stage 4 — Minimal Web login — COMPLETE

Verified commit: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`
CI run: `34420655852` PASS.

Implemented DB startup/migration, protected one-time owner bootstrap, DB-backed expiring sessions, login/dashboard, `HttpOnly`/`SameSite=Strict` cookie handling, CSRF, generic credential failures, direct-peer login rate limiting and database readiness.

### Stage 5 — Installer foundation — COMPLETE

Verified commit: `457f52783f9b2962c55be00beb801f0f2534958c`
Verified CI run: `34427021157` PASS.

Implemented multi-stage non-root Control Plane image, hardened Compose service, exclusive installer lock/state, random persistent Panel port, one-time admin password, safe rerun, final install summary, `tproxy` management and real Docker E2E.

### Stage 6A — Pinned Telemt data plane — COMPLETE

Verified commit: `47a95349922ba5be37cf0ed8416b482de08580ef`
Verified CI run: `34438212903` PASS.

Implemented:
- Telemt `3.5.7` pinned to exact release artifacts
- SHA-256 verification before extracting the Telemt binary
- amd64 musl digest `db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41`
- arm64 musl digest `8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1`
- non-root distroless Telemt runtime with liveness healthcheck
- Telemt service on the project bridge; MTProto listener published, Control API `9091` not host-published
- cryptographically random persistent Telemt API Bearer token, stored outside install state/log output
- generated Telemt config with strict file permissions
- disabled internal bootstrap proxy user solely to satisfy Telemt's non-empty-users startup invariant
- persistent Proxy port/config/data state and conflict checks
- `tproxy proxy status|restart|logs` and Telemt health in `tproxy doctor`
- installer waits for both Control readiness and Telemt health before success output
- E2E checks API host isolation, unauthorized rejection, authorized health, token non-leakage, proxy health and rerun port persistence

Architecture note: the project bridge is intentionally not Docker `internal:true` because Telemt itself needs outbound connectivity to Telegram. API isolation is enforced by no host port publication plus Bearer authentication.

### Stage 6B — Control Plane Telemt client adapter — NEXT

Scope:
- add focused `internal/telemt` Go client
- load API URL and Bearer token from config/file without logging the token
- bounded HTTP client timeouts
- typed health result and dependency errors
- tests for healthy, unauthorized, malformed response, unavailable and timeout behavior
- wire adapter into Control Plane without making `/readyz` depend on Telemt
- expose proxy dependency state only through authenticated/admin-safe status surface

Acceptance:
- token is never emitted in logs/errors/client-visible responses
- `GET /v1/health` is called with required Authorization header
- timeout/unavailable Telemt does not stop Control Plane startup/readiness
- tests cover healthy/auth/unavailable/timeout paths
- Go format/vet/test and existing installer E2E remain green

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
- Recovery point: if interrupted, inspect every commit/file after CP-006. Finish the repository-source mirror checkpoint if it is the active diff; otherwise repair/continue Stage 6B before starting user lifecycle work.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt remains an external dependency integrated via authenticated Control API; no Telemt source is copied into Teleproxy.
- Pin Telemt versions/checksums; never follow unbounded `latest` in production installation.
- Telemt 3.5.7 rejects an empty `[access.users]`; use a generated disabled internal bootstrap user until real user lifecycle exists.
- Telemt Control API must not be published as a host port.
- Project bridge cannot be Docker-internal because Telemt requires outbound Telegram connectivity.
- SQLite remains single-connection in the MVP so connection-scoped PRAGMAs cannot silently disappear.
- Initial admin password is passed through a protected bootstrap file, never a plaintext command-line argument or normal log field.
- Random Panel port is not a security boundary; TLS hardening is still required before treating public Panel exposure as production-ready.

## Validation / failure log

- 2026-09-10: repository initialized and isolated branch created.
- 2026-09-10: Stage 2 local format/test/vet PASS; CI `34419759826` PASS.
- 2026-09-10: Stage 3 CI `34420104043` PASS.
- 2026-09-10: Stage 4 CI `34420655852` PASS; promoted to CP-004.
- 2026-09-10: Stage 5 candidate `c1cde407...`; CI `34426475546` failed because bootstrap secret was `0640`; backend correctly rejected it. Kept backend validation and changed secret to UID 10001 + `0600`.
- 2026-09-10: Stage 5 fix `34b455b0...`; CI `34426870772` failed because the E2E harness omitted temporary `TPROXY_INSTALL_DIR`; production unchanged.
- 2026-09-10: Stage 5 final `457f5278...`; CI `34427021157` PASS; promoted to CP-005.
- 2026-09-10: Stage 6A candidate `39349dcf...`; CI `34437891406` reached healthy Telemt but the E2E used `docker compose port telemt 9091` as an unreliable no-binding assertion. Production Compose already had no `9091` mapping; changed test to inspect Docker `HostConfig.PortBindings` directly.
- 2026-09-10: Stage 6A test fix `91722c95...`; CI `34438040249` passed installation/isolation up to reading the protected token, then failed because shell input redirection happened before `sudo`. Production token permissions remained `0600`; test changed to read via `sudo cat`.
- 2026-09-10: Stage 6A final `47a95349...`; CI `34438212903` PASS including Go, shell/unit checks, real Docker Telemt build/install, API isolation/authentication, secret non-leakage, proxy health and rerun. Promoted to CP-006.

## Current next action

Mirror the exact supplied `AGENTS.md`, `ROADMAP_FA.md`, and `ROADMAP_EN.md` into repository root as a documentation-only checkpoint now that the source files are materialized locally. Then implement Stage 6B from CP-006 without coupling Control Plane readiness to Telemt availability.
