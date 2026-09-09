# Reliability and Recovery

## Goal

A failed or interrupted operation must leave enough durable information to diagnose, retry or roll back safely. The repository, persistent install state and database are the recovery sources of truth; chat history is not.

## Development recovery

Substantial work is tracked in `docs/exec-plans/active/`.

After each coherent milestone:

1. update the active execution plan;
2. record the exact checkpoint commit;
3. run targeted validation;
4. inspect the changed files/diff;
5. record failures and the next concrete action.

When resuming, inspect the branch head and active plan before modifying files. Repair a partial milestone before starting another.

## Runtime independence

- Control Plane liveness must not depend on telemt availability.
- telemt must run separately with an independent restart policy.
- A degraded telemt dependency affects readiness/dependency status, not the Control Plane process liveness endpoint.
- Bot failures must not terminate either the Web service or telemt.

## Idempotent host operations

Installer/update/repair operations use an install lock and persistent state directory. Every operation must distinguish:

- not started;
- partially applied;
- applied and unverified;
- verified healthy;
- rollback required.

Rerunning installation must preserve existing data, selected Panel port and established credentials unless an explicit rotate/reset operation is requested.

## Updates

Update sequence:

1. acquire lock;
2. preflight checks;
3. database/config backup;
4. stage new artifacts/config;
5. verify checksums;
6. activate;
7. run health verification;
8. mark checkpoint healthy;
9. otherwise restore the previous known-good state.

Never report update success before post-activation health verification.

## Backups

Backups include at least:

- SQLite database using a safe SQLite-compatible snapshot method;
- installation state/config excluding avoidable plaintext transient secrets;
- version/deployment metadata required for restore compatibility.

Restore must back up current state first and verify migration compatibility and database integrity.

## Health model

Expose separate health concepts:

- liveness: process is serving and event loop is healthy;
- readiness: Control Plane can serve normal authenticated requests and required local state is ready;
- dependency health: telemt, Telegram API, disk and other external dependencies;
- installation health: expected containers/listeners/configuration match persistent install state.

## Watchdog

The watchdog/Compose restart policy may restart a crashed or unhealthy component, but it must not create a restart loop that destroys diagnostic state. After repeated failures, surface a failed/degraded status for `tproxy doctor` and logs.
