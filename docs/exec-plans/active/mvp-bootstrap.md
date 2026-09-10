# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `5798075de40d2f966d8546a18e6fa450d7142f90`

## Purpose

Build the first recoverable foundation of Teleproxy from an empty repository, following the supplied engineering contract and project roadmap.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane remain lifecycle-independent.
- Go + lightweight HTTP + embedded UI + SQLite for the Control Plane.
- `telemt` is an external data-plane component; do not copy its source into this project.
- Installation is Docker-first and safe to rerun.
- First installation selects a random available high port for the Web Panel, persists it, and does not silently change it on rerun.
- Installer final output shows Panel URL, Panel port, admin username, generated initial credential/setup secret when needed, component health, version, data/config paths, and `tproxy` management command.
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

Implemented:
- validated HTTP config with loopback default
- HTTP server with timeouts and graceful shutdown
- `/healthz` and `/readyz`
- shared Problem Details representation
- config and HTTP tests

Validation:
- local gofmt/test/vet PASS
- GitHub Actions CI run `34419759826` PASS

### Stage 3 — Core persistence + admin bootstrap — COMPLETE

Verified commit: `5798075de40d2f966d8546a18e6fa450d7142f90`
CI run: `34420104043` -> PASS (format, vet, test)

Implemented:
- SQLite driver pinned to `github.com/mattn/go-sqlite3 v1.14.52`
- Go minimum 1.26 for standard-library `crypto/pbkdf2`
- single-connection SQLite invariant for connection-scoped PRAGMAs
- WAL, `synchronous=NORMAL`, foreign keys, 5000 ms busy timeout
- embedded ordered migration runner and `schema_migrations`
- `admins` and `admin_sessions` schema
- PBKDF2-SHA256 password hashes with random 128-bit salts and bounded verifier parameters
- 256-bit random session tokens with SHA-256 persistence digest
- idempotent one-time owner bootstrap
- tests for DB invariants/migration rerun, password/session hashing, bootstrap idempotency, and plaintext avoidance

### Stage 4 — Minimal Web login — ACTIVE

Scope:
- wire SQLite into the Control Plane process
- first-admin bootstrap from a root-owned/read-only password file rather than a plaintext CLI argument
- admin authentication against stored hash
- persisted session create/lookup/revoke with expiry
- login page and authenticated dashboard shell
- `HttpOnly` + `SameSite` session cookies and configurable `Secure` flag
- CSRF protection for login/logout state changes
- bounded in-memory login rate limiting keyed by direct peer IP
- generic invalid-login response

Acceptance:
- unauthenticated dashboard redirects to login
- valid login creates a DB-backed session and reaches dashboard
- wrong credentials return one generic response
- repeated failed login is rate-limited
- logout requires CSRF and revokes the DB session
- session token is never persisted plaintext
- first-admin bootstrap is rerun-safe and does not require the bootstrap password file once an admin exists
- CI format/vet/test passes

### Stage 5 — Installer foundation — PENDING

Scope:
- Docker Compose
- random available high Panel port selection with conflict check
- persistent install state
- generated admin bootstrap credential
- final install summary
- safe rerun behavior

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
- Recovery point: if Stage 4 is interrupted, inspect every commit/file after CP-003 before editing and repair Stage 4 first.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt remains an external dependency integrated via authenticated Control API.
- Pin Telemt versions/checksums; do not follow unbounded `latest`.
- Do not expose Telemt management API as a public host port.
- No Telemt source is copied into Teleproxy; preserve upstream license/branding requirements where applicable.
- SQLite remains single-connection in the MVP so connection-scoped PRAGMAs cannot silently disappear.
- Initial admin password will be passed through a protected bootstrap file in deployment, not command-line flags or normal logs.

## Validation log

- 2026-09-10: repository initialized and isolated branch created.
- 2026-09-10: Stage 2 local format/test/vet PASS.
- 2026-09-10: CI run `34419759826` PASS.
- 2026-09-10: Stage 3 candidate `5798075...` created atomically.
- 2026-09-10: CI run `34420104043` PASS: format, vet and tests all successful.
- 2026-09-10: Stage 3 promoted to CP-003; Stage 4 started.

## Current next action

Implement Stage 4 as one coherent candidate. If interrupted before its CI passes, resume from CP-003, inspect branch head and Stage 4 files, then finish or repair Stage 4 before Stage 5.
