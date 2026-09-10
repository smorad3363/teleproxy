# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`

## Purpose

Build the first recoverable foundation of Teleproxy from an empty repository, following the supplied engineering contract and project roadmap.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane remain lifecycle-independent.
- Go + lightweight HTTP + embedded UI + SQLite for the Control Plane.
- `telemt` is an external data-plane component; do not copy its source into this project.
- Installation is Docker-first and safe to rerun.
- First installation selects a random available high port for the Web Panel, persists it, and does not silently change it on rerun.
- Installer final output shows Panel URL, Panel port, admin username, generated initial credential/setup secret when needed, component health, version/ref, data/config paths, and `tproxy` management command.
- Telemt management API must not be published publicly; keep it on an internal network and authenticate requests.
- Never mount `/var/run/docker.sock` into the Web App.
- No plaintext passwords, bot tokens, session secrets, MTProto secrets, API bearer values, or private keys in logs.
- Substantial work must be recoverable from this file plus Git state without relying on chat history.

## Recovery protocol

When resuming after interruption:

1. Read `AGENTS.md` when present, `docs/ARCHITECTURE.md`, the subsystem document being changed, and this execution plan.
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
- minimal Go CI workflow with read-only repository permission

Remaining:
- commit the supplied canonical `AGENTS.md` at repository root
- mirror supplied `ROADMAP_FA.md` and `ROADMAP_EN.md` at repository root

The supplied uploaded files remain authoritative until those large source files are mirrored into the branch.

### Stage 2 — Minimal Control Plane — COMPLETE

Implemented validated HTTP config, graceful HTTP server, `/healthz`, `/readyz`, shared Problem Details and tests.
Validation: local gofmt/test/vet PASS; GitHub Actions run `34419759826` PASS.

### Stage 3 — Core persistence + admin bootstrap — COMPLETE

Verified commit: `5798075de40d2f966d8546a18e6fa450d7142f90`
CI run: `34420104043` PASS.

Implemented SQLite WAL/foreign keys/busy timeout/NORMAL sync, migrations, owner bootstrap, PBKDF2-SHA256 password hashing, hashed session-token persistence and tests.

### Stage 4 — Minimal Web login — COMPLETE

Verified commit: `2e83b4770dd8e26fc5c0ebcca8dee6aa51111254`
CI run: `34420655852` PASS (format, vet, test).

Implemented:
- Control Plane opens/migrates SQLite on startup
- first-admin bootstrap from a protected file, only when no admin exists
- no plaintext administrator password in DB/logging
- persisted server-side admin sessions with hashed token storage and expiry/revocation
- login page and authenticated dashboard shell
- `HttpOnly`/`SameSite=Strict` session cookies with configurable `Secure`
- CSRF protection for login/logout
- generic invalid-login error
- direct-peer login rate limiting
- database readiness check
- database file permission hardening to `0600`

### Stage 5 — Installer foundation — ACTIVE

Scope:
- hardened multi-stage Control Plane Docker image
- Docker Compose for the Control Plane with no Docker socket, dropped capabilities and read-only root filesystem where practical
- idempotent host installer with an exclusive lock and persistent install state
- random high Panel port selection, host/Docker conflict check and persistence
- safe retry after interruption: reuse healthy partial install, or reselect an unavailable port only before install is marked complete
- cryptographically random initial admin password delivered through a protected bootstrap file
- do not retain/reprint the initial plaintext password after a verified first install
- final install summary showing Panel URL/port, admin user/password on first successful bootstrap, Control Plane health, Proxy Plane status, ref/version, data/config paths and `tproxy` command
- minimal `tproxy` manager commands for status/logs/restart/doctor/config
- shell/unit tests for installer state and port behavior
- CI shell validation

Acceptance:
- `bash -n` passes for installer/manager/test scripts
- installer unit tests pass without Docker by stubbing host probes
- first install selects and persists a free high Panel port
- rerun after completed install preserves Panel port
- interrupted install with an unavailable unverified port can recover by selecting a new free port
- generated password file has restrictive permissions and is removed after successful bootstrap summary
- Compose does not mount `/var/run/docker.sock`
- Compose publishes only the selected Panel host port for this stage
- Go CI still passes

### Stage 6 — Telemt integration — PENDING

Scope after Stage 5:
- pin upstream Telemt version/checksum
- add internal-only Telemt service/API network boundary
- authenticated Telemt client adapter and health
- initial proxy-node configuration and user lifecycle integration

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
- Recovery point: if Stage 5 is interrupted, inspect every commit/file after CP-004 and repair Stage 5 before starting Telemt integration.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt remains an external dependency integrated via authenticated Control API.
- Pin Telemt versions/checksums; do not follow unbounded `latest`.
- Do not expose Telemt management API as a public host port.
- No Telemt source is copied into Teleproxy; preserve upstream license/branding requirements where applicable.
- SQLite remains single-connection in the MVP so connection-scoped PRAGMAs cannot silently disappear.
- Initial admin password is passed through a protected bootstrap file, never a plaintext command-line argument or normal log field.
- Stage 5 will mount a secrets directory rather than a single required secret file so the bootstrap file can be deleted after first success without breaking later container restarts.
- Stage 5 is Control Plane installation only; the final summary must explicitly report Proxy Plane as not configured until Stage 6 rather than pretending it is healthy.

## Validation log

- 2026-09-10: repository initialized and isolated branch created.
- 2026-09-10: Stage 2 local format/test/vet PASS.
- 2026-09-10: CI run `34419759826` PASS.
- 2026-09-10: Stage 3 CI run `34420104043` PASS.
- 2026-09-10: Stage 4 candidate `2e83b477...` committed atomically.
- 2026-09-10: Stage 4 CI run `34420655852` PASS; Stage 4 promoted to CP-004.
- 2026-09-10: Stage 5 started.

## Current next action

Implement Stage 5 as one coherent candidate, validate shell behavior locally and in CI, then promote it only after Go and installer checks pass. If interrupted, recover from CP-004 and inspect all Stage 5 changes before editing.
