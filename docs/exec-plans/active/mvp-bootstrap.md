# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `084d4a1256a6b28412d9457f6568a4138e09c83c`
Stage 3 candidate: `5798075de40d2f966d8546a18e6fa450d7142f90`

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
2. Inspect branch head and compare it with `Latest verified checkpoint` and any stage candidate recorded above.
3. Inspect every file changed after the verified checkpoint.
4. Re-run or inspect the targeted validation recorded for the partially completed milestone.
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
- Go module
- validated `TPROXY_HTTP_ADDR` configuration with loopback default
- HTTP server with timeouts and graceful shutdown
- `/healthz` and `/readyz`
- shared RFC 9457-style Problem Details representation
- config and HTTP tests

Validation:
- local `gofmt -l cmd internal` -> clean
- local `GOTOOLCHAIN=local go test -v ./...` -> PASS
- local `GOTOOLCHAIN=local go vet ./...` -> PASS
- GitHub Actions CI run `34419759826` at `084d4a1256a6b28412d9457f6568a4138e09c83c` -> SUCCESS

### Stage 3 — Core persistence + admin bootstrap — VALIDATING

Candidate commit: `5798075de40d2f966d8546a18e6fa450d7142f90`

Implemented in candidate:
- SQLite driver pinned to `github.com/mattn/go-sqlite3 v1.14.52`
- Go minimum raised to 1.26 for standard-library `crypto/pbkdf2`
- single-connection SQLite invariant so connection-scoped PRAGMAs remain enforced
- WAL, `synchronous=NORMAL`, foreign keys, 5000 ms busy timeout
- embedded ordered migration runner with `schema_migrations`
- `admins` and `admin_sessions` schema
- PBKDF2-SHA256 password hashes with random 128-bit salt and bounded verifier parameters
- 256-bit random session tokens with only SHA-256 token digests intended for persistence
- idempotent one-time owner bootstrap
- tests for SQLite invariants/migration rerun, password hashing, session token hashing, and bootstrap idempotency/plaintext avoidance

Local validation:
- `gofmt` applied to all Stage 3 Go sources
- compile/test not claimed locally because the container Go toolchain is 1.23 and Stage 3 intentionally requires Go 1.26+

Acceptance pending:
- GitHub Actions Go 1.27.1 format/vet/test must pass for the branch containing this candidate.

### Stage 4 — Minimal Web login — PENDING

Scope:
- authenticate admin from the persisted password hash
- session persistence/lookup/revocation
- login page and authenticated dashboard shell
- secure cookie attributes
- CSRF protection for state-changing requests
- login rate limiting / brute-force delay

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
- Validation: GitHub repository write succeeded.

### CP-001 — minimal Control Plane verified locally

- Commit: `b98507bbb3ef0a74e6e147286c22ab7d626f9e72`
- Validation: gofmt clean, local tests pass, local vet pass.

### CP-002 — CI foundation verified

- Commit: `084d4a1256a6b28412d9457f6568a4138e09c83c`
- CI run: `34419759826`
- Result: SUCCESS.
- This is the current safe recovery checkpoint until Stage 3 CI passes.

## Decisions / discoveries

- Repository was empty at task start.
- Telemt is treated as an external dependency and integrated via its authenticated Control API boundary.
- Pin Telemt versions/checksums instead of following an unbounded `latest` tag.
- Do not expose the Telemt management API as a public host port.
- Upstream Telemt license/branding conditions must be preserved where applicable; no Telemt source is copied into Teleproxy.
- Stage 3 uses the supported `v1.14.x` go-sqlite3 line and standard-library PBKDF2 to keep new dependency count small.
- SQLite is intentionally single-connection in the MVP because `foreign_keys` and `busy_timeout` are connection-scoped; scale-out can revisit this with per-connection hooks rather than silently losing PRAGMAs.

## Validation log

- 2026-09-10: repository confirmed empty before initialization.
- 2026-09-10: branch `agent/mvp-bootstrap` created.
- 2026-09-10: Stage 2 local format/test/vet passed after a tool-transport timeout was retried separately.
- 2026-09-10: CI run `34419759826` completed successfully at CP-002.
- 2026-09-10: Stage 3 candidate created atomically as commit `5798075de40d2f966d8546a18e6fa450d7142f90`; local gofmt completed; CI validation is required before promotion to checkpoint.

## Current next action

Inspect the CI run triggered after this plan write. If Stage 3 fails, inspect the failed job/log and repair Stage 3 before any Stage 4 work. If it succeeds, promote the resulting branch head to the next verified checkpoint and immediately start Stage 4.
