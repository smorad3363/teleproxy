# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `76681c8c7a5031cb44d3c12e3664d11f8421cc29` (CP-063)
Most recent verified code CI: `34621814385` PASS
Current branch checkpoint: `2ef6317a251990f056fff7d518adc680887d67d3` (CP-063 promotion docs)
Current branch CI: `34622311160` PASS

Historical execution detail through CP-059 is preserved byte-for-byte at
`docs/exec-plans/archive/mvp-bootstrap-through-cp059.md`, using the prior active-plan
blob `7ee43ba2a7373fd4e359fba08e9e7c5b47ff4817`.

## Recovery contract

Resume from `Latest verified code checkpoint`, then inspect every later branch commit,
file diff, and CI result before writing. Repair an active partial milestone first.
Never reset, clean, force-push, or overwrite unrelated work.

If interrupted during connector-side staging before an atomic branch move, re-read
the current branch HEAD and this plan, then reconstruct the last intended complete
file replacement from current HEAD plus the verified intended diff. Unreachable
blob/tree/commit staging objects are not branch state and are never proof that a file
was published.

## Non-negotiable architecture

- SQLite WAL/NORMAL is authoritative Control Plane state.
- Credit Buckets are authoritative quota/reward state.
- Telemt quota/expiry is only an enforcement projection.
- Control Plane and Telemt lifecycles remain independent.
- Plaintext admin/API/Bot/webhook/MTProto secrets are never logged or persisted by
  Control Plane.
- Migrations remain additive/backward-compatible.

## Current verified checkpoints

- CP-060 Settings established referral reward configuration surface:
  `f038c8b266c66b9378d26547c7c4ab4a68e45de6`, CI `34613204935` PASS.
- CP-060 promotion docs:
  `e32525cfdccf64d0bff9b87e489d3fdace2fa7ed`, CI `34613627486` PASS.
- Stage 11T scope docs:
  `ad308e646c3f2a2bf1c3d7e3d378f4ca1e601b88`, CI `34614212190` PASS.
- CP-061 Referral history authoritative exact status filter:
  `8993aee8b9d980990504fe25261eb4c29879316e`, CI `34615741067` PASS.
- CP-061 promotion docs:
  `a440758caf316d41d50e9f9bb8ce6b70d93ef42e`, CI `34616097813` PASS.
- Stage 11U scope docs:
  `08776242d9ec8f83114082d405037a259bfb46cf`, CI `34616505248` PASS.
- Stage 11U first candidate:
  `8bdbdcb3adefc95fdb4000c47fbbebb6560846e8`, CI `34620149594` FAILED at Vet because
  `validRejectionReason` duplicated the already-established package helper in
  `internal/referral/eligibility.go`; Test and installer were skipped. This failure was
  repaired forward without reset or force-push.
- CP-062 Referral history authoritative exact rejection-reason filter:
  `046d487f38ebbbe30ba8dd753ce728f781042b04`, CI `34620300934` PASS across Format,
  Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and Telemt
  E2E/rerun.
- CP-062 promotion docs:
  `a5fcc2db68c38987231e22d453b5106ebfd19ac8`, CI `34620702411` PASS.
- Stage 12A scope docs:
  `e5ae7a0866b13ba68640ddf5d9370787700cc68a`, CI `34621417199` PASS.
- CP-063 Management CLI doctor Compose/container checks:
  `76681c8c7a5031cb44d3c12e3664d11f8421cc29`, CI `34621814385` PASS across Format,
  Vet, full Go tests, installer syntax/unit tests, Docker prerequisites, and Telemt
  E2E/rerun including `tproxy doctor` after initial install and rerun.
- CP-063 promotion docs:
  `2ef6317a251990f056fff7d518adc680887d67d3`, CI `34622311160` PASS.

## Explicit blockers

### Stage 11H — Bot Content runtime delivery wiring — BLOCKED

Runtime composition/fallback, button/emoji relationship, and missing-slot behavior
are not defined. Do not invent them.

### Stage 9D — Proxy Node test/health/status — BLOCKED

Per-Node credential source and `internal_api_endpoint` runtime meaning are not
established. Do not probe Nodes or invent credential storage/transport.

### Stage 7D2C — Referral reward issuance — BLOCKED

Reward recipient is unresolved: inviter, invitee, or both. Do not infer a recipient.

### Stage 7D3B — Referral anti-abuse — BLOCKED

Cap scope/defaults, cooldown semantics/default, blacklist subject, and suspicious
score inputs/threshold are unresolved. Do not invent them.

Also do not invent Node/Sponsor assignment/routing, user-scoped referral-tree/history
semantics, audit mutation/redaction wiring, Administrator RBAC enforcement,
backup/restore, update/restart/log/version-source semantics, Dashboard metrics, new
Telemt topology, or secret persistence.

## Stage 11T — Referral history authoritative exact status filter — COMPLETED AT CP-061

CP-061 extends only existing authenticated referral-history reads with an optional
exact `status` filter accepting the already-established `pending`, `rewarded`, and
`rejected` constants. It preserves no-store/read-only behavior and bounded newest-first
pagination. No reward or anti-abuse write semantics were added.

## Stage 11U — Referral history authoritative exact rejection-reason filter — COMPLETED AT CP-062

CP-062 extends the existing authenticated `GET /api/referral/history` and
`GET /referrals` read contracts with one optional exact `rejection_reason` filter over
the authoritative stored `referral_attributions.rejection_reason` field.

Accepted values are exactly the six established `referral.RejectionReason` constants:
`anti_abuse`, `daily_cap`, `weekly_cap`, `cooldown`, `blacklist`, and `suspicious`.
The domain read model reuses the existing package `validRejectionReason` validator,
applies exact `ra.rejection_reason = ?`, and composes independently with optional
`status` and `before_id` using logical AND. It does not infer `status=rejected`;
contradictory filters return an empty result.

The HTTP parser rejects empty, duplicate, unknown, differently-cased, or otherwise
invalid reason values with the existing `REFERRAL_HISTORY_INVALID` Problem contract.
The `/referrals` page exposes All plus only the six established reasons; All omits the
query parameter, while active status/reason and explicit limit survive Older referrals
pagination.

The final scoped diff contains exactly:
- `internal/referral/history.go`
- `internal/httpapi/referral_history.go`
- `internal/httpapi/referral_page.go`
- `internal/referral/history_rejection_reason_filter_test.go`
- `internal/httpapi/referral_history_rejection_reason_filter_test.go`
- `internal/httpapi/referral_page_rejection_reason_filter_test.go`

No endpoint, migration, write path, anti-abuse policy, suspicious scoring, reward
recipient/issuance, eligibility/finalization mutation, referral tree, inviter/invitee
user-scoped filtering, assignment/routing, audit wiring, Bot behavior, per-Node
credentials/health, Telemt topology, or secret persistence was added. GET remains
no-store/read-only and regression coverage verifies exact filtering, independent
status composition, bounded pagination, invalid-query rejection, and no authoritative
state mutation.

## Stage 12A — Management CLI doctor Compose/container checks — COMPLETED AT CP-063

CP-063 strengthens only the existing host-side `tproxy doctor` path. It preserves the
existing Docker daemon check, Control Plane `/readyz`, and Proxy Plane `proxy_health`
behavior while adding bounded checks for installed Compose configuration validity and
for the expected `control` and `telemt` containers to exist and be running.

`control` container-running state remains separate from `/readyz`; no Docker
healthcheck was invented. `telemt` container-running state remains separate from and
does not weaken the existing authoritative Proxy Plane health check. Doctor failures
remain non-zero and diagnostics are bounded; no state-file contents, secret contents,
tokens, passwords, internal Telemt credentials, or raw Docker inspect dumps are
printed.

The final scoped diff contains exactly:
- `bin/tproxy`
- `tests/installer_e2e.sh`

Both executable modes remain `100755`. Installer E2E now invokes `tproxy doctor` after
the first successful install and after rerun, and verifies only the established bounded
labels: `Compose: valid`, `Control container: running`, `Telemt container: running`,
`Control Plane: ready`, and `Proxy Plane: healthy`.

No daemon, migration, endpoint, backup/update/rollback, watchdog, Node health, Bot
connectivity, DNS, disk, clock, repair semantics, or unrelated CLI command behavior was
added or changed. Candidate `76681c8c7a5031cb44d3c12e3664d11f8421cc29`,
CI `34621814385` PASS all gates.

## Stage 12B — CI ShellCheck gate — ACTIVE

The roadmap explicitly requires ShellCheck in recommended CI. The current workflow
performs Bash syntax checks and shell unit/E2E tests but has no ShellCheck gate. This
stage adds one bounded static-analysis gate for the repository's existing shell entry
points and tests.

The intended first implementation changes only `.github/workflows/ci.yml`, adding a
ShellCheck step in the existing `installer` job after checkout and before syntax/unit
tests. It checks exactly the current shell files already covered by the installer CI
path: `install.sh`, `scripts/install_lib.sh`, `scripts/install-host.sh`, `bin/tproxy`,
`tests/installer_lib_test.sh`, and `tests/installer_e2e.sh`.

Use the ShellCheck binary supplied by the GitHub-hosted Ubuntu runner; do not add a
third-party action, remote install script, new package repository, or downloaded
binary. The existing Bash syntax/unit, Docker prerequisite, and installer/Telemt E2E
gates remain unchanged and still run after ShellCheck.

If the new gate exposes pre-existing ShellCheck findings, repair them forward only
when the fix is demonstrably semantics-preserving and limited to the listed shell
files. Do not suppress broad warning classes, add blanket exclusions, or refactor
unrelated installer/CLI behavior merely to silence the linter. Any such repair must be
reviewed as part of the candidate diff and pass the full existing CI.

No runtime feature, endpoint, migration, Docker topology, installer behavior, CLI
command semantics, secret handling, backup/update/rollback/watchdog behavior, Bot
behavior, Node health, referral semantics, or Control/Proxy lifecycle coupling is in
scope.

### Acceptance

- CI has an explicit ShellCheck step in the existing installer job.
- The gate covers exactly the six existing shell files listed above.
- No third-party action or network-time shell linter installer is introduced.
- Existing Bash syntax/unit, Docker prerequisite, and Telemt E2E/rerun gates remain
  intact.
- Any linter-driven source repair is minimal, forward-only, and semantics-preserving.
- Full CI passes.

## Current next action

Publish this Stage 12B scope as a plan-only commit and require full CI PASS. Then add
only the bounded ShellCheck CI gate above; if it reports existing findings, repair only
minimal semantics-preserving shell issues and require full CI again before checkpoint
promotion. Preserve every explicit blocker and do not broaden the milestone into
runtime or product behavior.
