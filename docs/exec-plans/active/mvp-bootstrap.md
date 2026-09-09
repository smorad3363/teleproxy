# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`

## Purpose

Build the first recoverable foundation of Teleproxy from an empty repository, following the supplied engineering contract and project roadmap.

## Non-negotiable requirements

- Control Plane and Proxy Data Plane remain lifecycle-independent.
- Go + lightweight HTTP + embedded UI + SQLite for the Control Plane.
- `telemt` is integrated as an external data-plane component; do not copy its source into this project.
- Installation is Docker-first and safe to rerun.
- First installation selects a random available high port for the Web Panel, persists it, and does not silently change it on rerun.
- Installer final output shows Panel URL, Panel port, admin username, generated initial credential/setup secret when needed, component health, version, data/config paths, and `tproxy` management command.
- Telemt management API must not be published publicly; keep it on an internal network and authenticate requests.
- Never mount `/var/run/docker.sock` into the Web App.
- No plaintext passwords, bot tokens, session secrets, MTProto secrets, API bearer values, or private keys in logs.
- Substantial work must be recoverable from this file plus Git state without relying on chat history.

## Recovery protocol

When resuming after interruption:

1. Read `AGENTS.md` when present and this execution plan.
2. Inspect branch head and compare it to the last checkpoint below.
3. Inspect every file changed after that checkpoint.
4. Re-run the targeted validation recorded for the partially completed milestone.
5. If the last write was partial or inconsistent, repair that milestone before starting a new one.
6. Do not restart from scratch and do not overwrite unrelated changes.

## Stage policy

Each stage is intentionally small and independently verifiable. Do not broaden a stage while implementing it.

### Stage 1 — Repository foundation

Scope:
- repository-local recovery plan
- canonical roadmap/instructions
- architecture/security/reliability docs
- CI skeleton and base project layout

Acceptance:
- a new agent can determine architecture, security boundaries, current stage, and next action from repository files alone.

### Stage 2 — Minimal Control Plane

Scope:
- Go module
- config loader with safe defaults
- HTTP server
- `/healthz` liveness and `/readyz` readiness
- structured Problem Details error type
- tests for config and health endpoints

Acceptance:
- `go test ./...` passes locally.
- server starts with a configurable bind address.

### Stage 3 — Core persistence + admin bootstrap

Scope:
- SQLite connection and migrations
- WAL / foreign keys / busy timeout / synchronous NORMAL
- admins table and one-time bootstrap credential
- secure password hashing
- login/session primitives

Acceptance:
- migration test passes on a fresh DB.
- password is never stored plaintext.
- bootstrap is idempotent.

### Stage 4 — Minimal Web login

Scope:
- login page
- secure cookie session
- CSRF protection for state-changing requests
- login rate limiting / brute-force delay
- authenticated dashboard shell

Acceptance:
- unauthenticated dashboard redirects/rejects.
- valid login works.
- invalid login is generic and rate-limited.

### Stage 5 — Installer foundation

Scope:
- Docker Compose
- random available high panel port selection with conflict check
- persistent install state
- generated admin bootstrap credential
- final install summary
- safe rerun behavior

Acceptance:
- shell syntax checks pass.
- port-selection test proves selected port is free at selection time.
- rerun preserves the previously selected panel port.

### Later stages

Continue with telemt integration, Telegram Bot, referrals/credits, sponsors, RBAC, backups/update/rollback, watchdog, and relay support according to the roadmap.

## Checkpoints

### CP-000 — repository initialized

- Commit: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
- State: default branch initialized with README only.
- Validation: GitHub repository write succeeded.
- Known issue: canonical `AGENTS.md` and roadmap files still need to be committed on the task branch.
- Next action: finish Stage 1 foundation files on `agent/mvp-bootstrap`.

## Decisions / discoveries

- Repository was empty at task start.
- Current upstream Telemt exposes a Control API with user management, quota/expiry, per-user ad tag, health and reload-related capabilities; integration should use that API rather than direct uncontrolled config editing where possible.
- Pin Telemt versions/checksums in deployment instead of following an unbounded `latest` tag.
- Upstream Telemt has its own license/branding conditions; treat it as an external dependency and preserve required attribution.

## Validation log

- 2026-09-10: repository metadata inspected; repository confirmed empty before initialization.
- 2026-09-10: default branch initial README commit created.
- 2026-09-10: task branch `agent/mvp-bootstrap` created from baseline.

## Current next action

Finish Stage 1 repository foundation, update this plan with the resulting checkpoint, then immediately start Stage 2.
