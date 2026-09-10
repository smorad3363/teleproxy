# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified checkpoint: `b98507bbb3ef0a74e6e147286c22ab7d626f9e72`

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
2. Inspect branch head and compare it with `Latest verified checkpoint`.
3. Inspect every file changed after that checkpoint.
4. Re-run the targeted validation recorded for the partially completed milestone.
5. If the last write was partial or inconsistent, repair that milestone before starting a new one.
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

Remaining:
- commit the supplied canonical `AGENTS.md` at repository root
- mirror `ROADMAP_FA.md` / `ROADMAP_EN.md` at repository root
- CI skeleton

Note: the connected GitHub text writer requires a complete payload per large file. Until the canonical large files are mirrored, the supplied uploaded files remain the task source and this plan records that gap explicitly.

### Stage 2 — Minimal Control Plane — COMPLETE

Implemented:
- Go module
- validated `TPROXY_HTTP_ADDR` configuration with safe loopback default
- HTTP server with timeouts and graceful shutdown
- `/healthz` and `/readyz`
- shared RFC 9457-style Problem Details representation
- config and HTTP tests

Validation actually run on the local staged source:
- `gofmt -l cmd internal` -> clean
- `GOTOOLCHAIN=local go test -v ./...` -> PASS
- `GOTOOLCHAIN=local go vet ./...` -> PASS

Verified GitHub branch head after the Stage 2 writes:
- `b98507bbb3ef0a74e6e147286c22ab7d626f9e72`

### Stage 3 — Core persistence + admin bootstrap — ACTIVE

Scope:
- SQLite connection and migrations
- WAL / foreign keys / busy timeout / synchronous NORMAL
- admins table and one-time bootstrap credential
- secure password hashing
- session storage primitives

Acceptance:
- migration test passes on a fresh DB
- password is never stored plaintext
- bootstrap is idempotent
- session tokens are not stored plaintext

### Stage 4 — Minimal Web login — PENDING

Scope:
- login page
- secure cookie session
- CSRF protection for state-changing requests
- login rate limiting / brute-force delay
- authenticated dashboard shell

### Stage 5 — Installer foundation — PENDING

Scope:
- Docker Compose
- random available high panel port selection with conflict check
- persistent install state
- generated admin bootstrap credential
- final install summary
- safe rerun behavior

## Checkpoints

### CP-000 — repository initialized

- Commit: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
- Validation: GitHub repository write succeeded.

### CP-001 — minimal Control Plane verified

- Commit: `b98507bbb3ef0a74e6e147286c22ab7d626f9e72`
- Local validation: gofmt clean, `go test ./...` pass, `go vet ./...` pass.
- Files: `go.mod`, `cmd/control/main.go`, `internal/config/*`, `internal/httpapi/*` plus Stage 1 docs.
- Known gap: Stage 1 canonical large source mirrors and CI still pending.
- Next action: implement Stage 3 locally, validate, then commit only the validated files.

## Decisions / discoveries

- Repository was empty at task start.
- Current upstream Telemt exposes Control API capabilities needed for user management, quota/expiry, per-user AdTag, health and reload behavior; use the API integration boundary rather than uncontrolled direct config edits where possible.
- Pin Telemt versions/checksums in deployment instead of following an unbounded `latest` tag.
- Upstream Telemt has its own license/branding conditions; treat it as an external dependency and preserve required attribution.
- The project `go` directive is currently `1.23` as a minimum language baseline for locally verified Stage 2 code; production builder images will be pinned independently to a currently supported Go release.

## Validation log

- 2026-09-10: repository metadata inspected; repository confirmed empty before initialization.
- 2026-09-10: default branch initial README commit created.
- 2026-09-10: task branch `agent/mvp-bootstrap` created from baseline.
- 2026-09-10: Stage 2 combined local command timed out at the tool transport layer; code was then inspected and checks rerun separately.
- 2026-09-10: Stage 2 `go test -v ./...` passed.
- 2026-09-10: Stage 2 `go vet ./...` passed.
- 2026-09-10: GitHub branch head verified at CP-001.

## Current next action

Implement and validate Stage 3 core persistence + admin bootstrap. If interrupted, resume from CP-001 and inspect any commits after it before editing.
