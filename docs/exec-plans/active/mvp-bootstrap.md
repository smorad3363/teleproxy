# MVP Bootstrap Execution Plan

Status: ACTIVE
Branch: `agent/mvp-bootstrap`
Baseline: `79bfc2a4f0151719bf3502f74d7acb6b9600e094`
Latest verified code checkpoint: `f2378e70dc4028fa40f9d1bd2a5c540a1f67bac0` (CP-068)
Most recent verified code CI: `34647334507` PASS
Current branch checkpoint: `f2378e70dc4028fa40f9d1bd2a5c540a1f67bac0` (CP-068 candidate)
Current branch CI: `34647334507` PASS

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
- Stage 12B scope docs:
  `26e5ba683a1c26579db5c916ac890dc01864bbc2`, CI `34623293172` PASS.
- Stage 12B first candidate:
  `c22a0cc81013d329ef0381982908e14b87dab827`, CI `34623606814` FAILED at the new
  ShellCheck gate. The gate exposed pre-existing `SC1091`, `SC2034`, `SC2251`, and
  `SC2024` findings; later installer syntax/unit, Docker, and E2E gates were skipped.
- Stage 12B first forward repair:
  `86189982ad860aaa08d12beda8a8d42b724f5595`, CI `34624091531` FAILED only on the
  two remaining `SC1091` source-resolution findings. The unused loop variable, test
  assertions, and intentional redirect finding were already repaired; later installer
  gates were skipped again.
- CP-064 CI ShellCheck gate and minimal shell repairs:
  `e20f3cb2c17489cbf911b3bdd3130bf420978d4d`, CI `34624305188` PASS across Format,
  Vet, full Go tests, ShellCheck, installer syntax/unit tests, Docker prerequisites,
  and Telemt E2E/rerun.
- CP-064 promotion docs:
  `b055fedf4638f3b401ad5e3d9fbc00f48d692cf7`, CI `34624780650` PASS.
- Stage 12C scope docs:
  `5a4469dff041722a4fbc6475f07a25fc787294ac`, CI `34628954182` PASS.
- CP-065 CI Docker Compose config validation gate:
  `e0192fecaa9031af47ba508b0f94178654729dae`, CI `34629239550` PASS across Format,
  Vet, full Go tests, ShellCheck, installer syntax/unit tests, Docker prerequisites,
  explicit Compose config validation, and Telemt E2E/rerun.
- CP-065 promotion docs:
  `14c5f5504d0b729ca4f8df77d68dbb75dc160441`, CI `34629611234` PASS.
- Stage 12D scope docs:
  `099e6580d5977476b78736ccfa4356aa12258fe3`, CI `34644118607` PASS.
- CP-066 CI Docker build gate:
  `fa82c2d1c2646eb727b09f4415c1e3537af448bc`, CI `34644370094` PASS across Format,
  Vet, full Go tests, ShellCheck, installer syntax/unit tests, Docker prerequisites,
  explicit Docker build, Compose config validation, and Telemt E2E/rerun.
- CP-066 promotion docs:
  `e4c8c46b2f231928be19b8cf5d0358c0bcc6720b`, CI `34644717476` PASS.
- Stage 12E scope docs:
  `21a041a3ae8374c69ac607c965b2757f894434f2`, CI `34645182157` PASS.
- CP-067 CI SQLite migration test gate:
  `96727695b3bc5fbe28d0d9739e17d93c1e14632c`, CI `34645419394` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, explicit Docker build, Compose config validation, and
  Telemt E2E/rerun.
- CP-067 promotion docs:
  `ed3bb7b33a5758b0bc5d0ec0bec0f9daf60eea98`, CI `34645693007` PASS.
- Post-CP-067 recovery-state docs:
  `3390e4189bb8ca6d4c7791a7b48b57f2245db84f`, CI `34646117924` PASS across both
  jobs and all established Go, SQLite migration, ShellCheck, installer, Docker build,
  Compose validation, and Telemt E2E/rerun gates.
- Stage 11V scope docs:
  `a37f479658667e618adb351cc2342149b17a12f3`, CI `34646836783` PASS.
- CP-068 User inventory authoritative provisioning phase:
  `f2378e70dc4028fa40f9d1bd2a5c540a1f67bac0`, CI `34647334507` PASS across Format,
  Vet, explicit SQLite migration tests, full Go tests, ShellCheck, installer syntax/unit
  tests, Docker prerequisites, Docker build, Compose config validation, and Telemt
  E2E/rerun.

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

## Stage 12B — CI ShellCheck gate — COMPLETED AT CP-064

CP-064 adds the roadmap-required ShellCheck gate to the existing `installer` CI job
without adding a third-party action, package install, remote script, or downloaded
binary. The gate uses the ShellCheck binary already present on the GitHub-hosted Ubuntu
runner and checks exactly the six existing shell files in the installer CI path:
`install.sh`, `scripts/install_lib.sh`, `scripts/install-host.sh`, `bin/tproxy`,
`tests/installer_lib_test.sh`, and `tests/installer_e2e.sh`.

The first candidate exposed pre-existing lint findings and was repaired forward only.
The final gate runs `shellcheck -x` so the two established `source` directives can be
followed. Their annotations now resolve to the repository path
`scripts/install_lib.sh`. The unused retry loop variable became `_`; the two negative
validator assertions were rewritten as explicit failing `if` blocks; and the E2E
installer redirect keeps one line-local `SC2024` suppression with an explanatory
comment because `sudo` intentionally applies to the installer process while the
redirect remains owned by the invoking CI user. No warning class is disabled broadly.

The final scoped diff from Stage 12B scope contains exactly:
- `.github/workflows/ci.yml`
- `scripts/install-host.sh`
- `tests/installer_lib_test.sh`
- `tests/installer_e2e.sh`

Executable modes remain unchanged for the three shell files. Existing Bash
syntax/unit, Docker prerequisite, and Telemt E2E/rerun gates remain intact and run
after ShellCheck. No runtime feature, endpoint, migration, Docker topology, installer
behavior, CLI command semantics, secret handling, backup/update/rollback/watchdog
behavior, Bot behavior, Node health, referral semantics, or Control/Proxy lifecycle
coupling was added or changed.

Candidate chain and CI evidence:
- `c22a0cc81013d329ef0381982908e14b87dab827`, CI `34623606814` FAILED at ShellCheck.
- `86189982ad860aaa08d12beda8a8d42b724f5595`, CI `34624091531` FAILED only on
  remaining source-resolution `SC1091` findings.
- `e20f3cb2c17489cbf911b3bdd3130bf420978d4d`, CI `34624305188` PASS all gates.

## Stage 12C — CI Docker Compose config validation gate — COMPLETED AT CP-065

CP-065 adds one explicit `Compose config validation` step to the existing `installer`
CI job after Docker prerequisites and before the installer/Telemt E2E gate. It runs
`docker compose config --quiet` against the existing root `compose.yaml` using only
the already-required non-secret interpolation values for panel port and filesystem
paths.

The final scoped code diff contains exactly:
- `.github/workflows/ci.yml`

The new step adds no secret material, build, pull, container start, runtime mutation,
Compose topology/image/Dockerfile change, installer or CLI semantic change, endpoint,
migration, Bot/referral/Sponsor/Node behavior, backup/update/rollback/watchdog behavior,
or lifecycle coupling. Existing Format, Vet, full Go tests, ShellCheck, installer
syntax/unit, Docker prerequisites and Telemt E2E/rerun gates remain intact.

Stage 12C scope docs `5a4469dff041722a4fbc6475f07a25fc787294ac`
passed CI `34628954182`. Candidate `e0192fecaa9031af47ba508b0f94178654729dae`
passed CI `34629239550` across all gates including the new Compose validation step.

## Stage 12D — CI Docker build gate — COMPLETED AT CP-066

CP-066 adds one explicit `Docker build` step to the existing `installer` CI job after
Docker prerequisites and before Compose config validation. It reuses only the same
non-secret required Compose interpolation values and runs `docker compose build`, so it
builds the two already-defined `control` and `telemt` service images without pushing or
starting containers.

The final scoped code diff contains exactly:
- `.github/workflows/ci.yml`

No image tag, Dockerfile, Compose topology, secret, installer/runtime behavior,
endpoint, migration, Bot/referral/Sponsor/Node/Telemt product semantics, or lifecycle
boundary changed. The pre-existing Compose validation and installer/Telemt E2E/rerun
remain after the explicit build gate.

Stage 12D scope docs `099e6580d5977476b78736ccfa4356aa12258fe3`
passed CI `34644118607`. Candidate `fa82c2d1c2646eb727b09f4415c1e3537af448bc`
passed CI `34644370094` across Format, Vet, full Go tests, ShellCheck, installer
syntax/unit tests, Docker prerequisites, explicit Docker build, Compose config
validation, and Telemt E2E/rerun.

## Stage 12E — CI SQLite migration test gate — COMPLETED AT CP-067

CP-067 adds one explicit `SQLite migration tests` step to the existing `go` CI job
after Vet and before the full Go test suite. It runs only the already-established
migration-focused tests with:
`go test ./internal/database -run '^(TestOpenAppliesSQLiteInvariantsAndMigrations|TestMigrate.*)$'`.
This independently attributes regressions in fresh migration application/rerun and
existing upgrade-preservation fixtures while leaving the full `go test ./...` gate in
place immediately afterward.

The final scoped code diff contains exactly:
- `.github/workflows/ci.yml`

No migration SQL, schema, migration count, SQLite pragma, database runtime behavior,
test fixture semantics, persisted data contract, endpoint, installer, Docker, secret,
Bot/referral/Sponsor/Node/Telemt product behavior, or lifecycle boundary changed.

Stage 12E scope docs `21a041a3ae8374c69ac607c965b2757f894434f2`
passed CI `34645182157`. Candidate `96727695b3bc5fbe28d0d9739e17d93c1e14632c`
passed CI `34645419394` across Format, Vet, explicit SQLite migration tests, full Go
tests, ShellCheck, installer syntax/unit tests, Docker prerequisites, explicit Docker
build, Compose config validation, and Telemt E2E/rerun.

## Remaining CI roadmap after CP-067 — REVIEWED / NOT SCOPED

Current CI now explicitly covers gofmt, go vet, full Go tests, SQLite migration tests,
ShellCheck, Docker build, Compose config validation, and the existing installer/Telemt
E2E install/rerun path.

The remaining roadmap CI items were compared against the current repository primitives:

- Targeted integration tests: the repository has no dedicated test tag, named suite, or
  separate harness defining a distinct subset beyond the existing package tests and
  installer/Telemt E2E. Do not invent selection semantics merely to create a second gate.
- Secret scan: no repository-defined scanner, ruleset, suppression policy, or pinned
  scanner contract exists. Do not invent tool/pattern/version policy.
- Dependency vulnerability check: no repository-defined scanner, configuration, or
  version policy exists; the roadmap's "reasonable" threshold does not itself define a
  concrete enforcement contract. Do not invent one.
- Target-distro install smoke: the roadmap names Ubuntu LTS and Debian Stable, but the
  current automation has no pinned distro matrix or Debian installation harness. Do not
  claim support or invent a container/systemd/Docker substitute for actual target-OS
  testing.
- Upgrade/rollback smoke: update/rollback/version-source behavior remains explicitly
  unresolved in this plan, so those checks cannot be defined without inventing product
  semantics.

No additional implementation milestone is scoped from these items until its tooling and
acceptance contract are repository-defined or otherwise explicitly established.

## Stage 11V — User inventory authoritative provisioning phase — COMPLETED AT CP-068

CP-068 extends the existing authenticated User inventory reads with the authoritative
stored `proxy_user_provisioning.phase` value and does not infer a generic secret status.
The read model LEFT JOINs provisioning state by `proxy_user_id`, exposes JSON
`provisioning_phase: null` when no durable row exists, and otherwise returns only one of
the established `prepared`, `owned`, or `collision` phase constants after fail-closed
validation.

The `/users` page adds one `Provisioning phase` column. It renders the exact phase
literally, or the explicit safe text `not provisioned` when absent. Existing
newest-first bounded pagination, exact Telegram/proxy filters, no-store behavior,
lifecycle actions, CSRF behavior, Credit Bucket projection, referral count, and safe
empty states remain unchanged.

The final scoped code diff contains exactly:
- `internal/useradmin/list.go`
- `internal/useradmin/list_test.go`
- `internal/httpapi/users_test.go`
- `internal/httpapi/users_page.go`
- `internal/httpapi/users_page_test.go`

No secret digest or plaintext secret is selected, serialized, rendered, logged, or
otherwise exposed. No migration, write path, Telemt request, provisioning transition,
secret rotation semantic, Node/Sponsor assignment/routing, audit wiring, Bot behavior,
RBAC enforcement, backup/update/restart/log behavior, Dashboard metric, or lifecycle
boundary changed. Regression coverage verifies prepared versus absent provisioning
state in API/page pagination and includes authoritative-state no-mutation checks.

Stage 11V scope docs `a37f479658667e618adb351cc2342149b17a12f3`
passed CI `34646836783`. Candidate `f2378e70dc4028fa40f9d1bd2a5c540a1f67bac0`
passed CI `34647334507` across Format, Vet, explicit SQLite migration tests, full Go
tests, ShellCheck, installer syntax/unit tests, Docker prerequisites, Docker build,
Compose config validation, and Telemt E2E/rerun.

## Current next action

Promote CP-068 documentation only and require full CI PASS. Then inspect the remaining
roadmap and current repository contracts for the next independent milestone whose
semantics and tooling are already established. Preserve every explicit blocker,
lifecycle separation, and Credit Buckets as authoritative quota/reward state; do not
invent missing product/runtime semantics.
