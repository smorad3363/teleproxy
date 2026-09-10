# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `457f52783f9b2962c55be00beb801f0f2534958c`

## Purpose

Build the first recoverable foundation of Teleproxy from an empty repository, following the supplied engineering contract and project roadmap.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane remain lifecycle-independent.
- Go + lightweight HTTP + embedded UI + SQLite for the Control Plane.
- `telemt` is an external data-plane component; do not copy its source into this project.
- Installation is Docker-first and safe to rerun.
- First installation selects a random available high port for the Web Panel, persists it, and does not silently change it on rerun.
- Installer final output shows Panel URL, Panel port, admin username, generated initial credential when needed, component health, version/ref, data/config paths, and `tproxy` command.
- Telemt management API must not be published publicly; keep it on an internal network and authenticate requests.
- Never mount `/var/run/docker.sock` into the Web App.
- No plaintext passwords, bot tokens, session secrets, MTProto secrets, API bearer values, or private keys in logs.
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

Implemented DB startup/migration, protected one-time owner bootstrap, DB-backed expiring sessions, login/dashboard, `HttpOnly`/`SameSite=Strict` cookie handling, CSRF, generic credential failures, direct-peer login rate limiting and database readiness.

### Stage 5 — Installer foundation — COMPLETE

Verified commit: `457f52783f9b2962c55be00beb801f0f2534958c`
Verified CI run: `34427021157` PASS.

Implemented:
- multi-stage non-root Control Plane Docker image
- Compose service with no Docker socket, `no-new-privileges`, dropped capabilities and read-only root filesystem
- exclusive installer lock and persistent install state
- random high Panel port selection and persistence
- explicit conflict refusal for an already-persisted port
- first-install bootstrap password generated from `/dev/urandom`
- bootstrap secret owner-only (`0600`) for container UID 10001 and removed after successful first install
- final install summary with Panel URL/port, admin credential on first success, Control/Proxy status, source ref and paths
- `tproxy` status/logs/restart/start/stop/doctor/config/panel commands
- installer library tests plus real Docker install/rerun E2E

Acceptance verified by CI:
- Go format/vet/test PASS
- installer shell syntax PASS
- installer unit tests PASS
- Docker/Compose available in E2E runner
- first install reaches `/readyz`
- generated password is displayed once and secret file is removed afterward
- second install preserves the exact same Panel port
- second install does not reprint initial plaintext password
- manager resolves the installed panel

### Stage 6 — Telemt integration — PENDING

Scope:
- re-verify and pin upstream Telemt release/version/checksum
- add internal-only Telemt service/API network boundary
- generate Telemt API authorization secret without logging it
- authenticated Telemt client adapter and health probe
- initial proxy-node configuration suitable for later user lifecycle integration
- do not expose Telemt Control API as a host port

Acceptance:
- pinned Telemt artifact is checksum-verified before use
- Telemt API is reachable from Control Plane network but not published on host
- API auth is required
- adapter health test handles healthy, unauthorized and unavailable states
- existing installer E2E and Go CI remain green

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
- Recovery point: if interrupted after this checkpoint, inspect every commit/file after CP-005. Finish the repository-source mirror checkpoint if active; otherwise resume Stage 6. Do not reopen Stage 5 unless evidence shows a regression.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt remains an external dependency integrated via authenticated Control API.
- Pin Telemt versions/checksums; do not follow unbounded `latest`.
- Do not expose Telemt management API as a public host port.
- No Telemt source is copied into Teleproxy; preserve upstream license/branding requirements where applicable.
- SQLite remains single-connection in the MVP so connection-scoped PRAGMAs cannot silently disappear.
- Initial admin password is passed through a protected bootstrap file, never a plaintext command-line argument or normal log field.
- The random panel port is convenience/obscurity only, not a security boundary; Stage 5 remains HTTP-only and later hardening must add TLS before treating public exposure as production-ready.

## Validation / failure log

- 2026-09-10: repository initialized and isolated branch created.
- 2026-09-10: Stage 2 local format/test/vet PASS; CI `34419759826` PASS.
- 2026-09-10: Stage 3 CI `34420104043` PASS.
- 2026-09-10: Stage 4 CI `34420655852` PASS; promoted to CP-004.
- 2026-09-10: Stage 5 candidate `c1cde407...`; CI `34426475546` failed E2E because bootstrap secret was group-readable (`0640`) and backend correctly rejected it. Fixed by keeping backend validation and changing the secret to UID 10001 ownership + `0600`.
- 2026-09-10: Stage 5 fix `34b455b0...`; CI `34426870772` reached post-install manager assertion but failed because the E2E harness omitted its temporary `TPROXY_INSTALL_DIR`. Production behavior was not changed; harness corrected.
- 2026-09-10: Stage 5 final candidate `457f5278...`; CI `34427021157` PASS including Go, shell/unit checks and real Docker first-install + rerun E2E. Promoted to CP-005.

## Current next action

First close the remaining Stage 1 recovery-source gap by committing the exact supplied `AGENTS.md`, `ROADMAP_FA.md`, and `ROADMAP_EN.md` into the repository as one documentation-only checkpoint. Then begin Stage 6 Telemt integration from CP-005 plus that documentation checkpoint.
