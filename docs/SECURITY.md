# Security Baseline

## Authentication and sessions

- Store administrator passwords only as a modern password hash with per-password salt; never store plaintext or reversible password values.
- Initial install credentials are generated from a cryptographically secure random source.
- Login failures use generic responses and are rate-limited.
- Authenticated browser sessions use secure random opaque session identifiers or equivalently protected server-side sessions.
- Cookies are `HttpOnly`; use `Secure` when HTTPS is active and an appropriate `SameSite` policy.
- Rotate session identity on successful login and privilege changes.
- State-changing browser requests require CSRF protection.
- Authorization is enforced server-side for every privileged action.

## Secrets

Never log or return unnecessarily:

- administrator passwords;
- Telegram bot tokens;
- session secrets/IDs;
- telemt Control API authorization values;
- MTProto user secrets;
- private keys;
- webhook secrets.

Use restrictive file permissions for generated secret-bearing configuration. Sensitive values should be passed through environment/files with minimal scope rather than command-line arguments when practical.

## Network boundaries

- Publish only the Web Panel port and the intended MTProto ingress port(s).
- Do not publish the telemt Control API to the public network.
- Keep the telemt Control API on the internal Compose network and require its Authorization header.
- Do not mount `/var/run/docker.sock` into the Web/Control container.
- Host lifecycle operations go through the `tproxy` manager or another narrowly allowlisted host mechanism.
- Use restrictive CORS; the MVP same-origin Web Panel should not require broad cross-origin access.

## HTTP/API

- Validate all untrusted inputs at the boundary.
- Use parameterized SQL exclusively.
- Apply request size limits and timeouts.
- Use one structured Problem Details error contract. Client responses must not contain stack traces, SQL text, internal file paths, tokens or infrastructure secrets.
- Generate/request a trace or request ID for unexpected failures and log only safe diagnostic context.

## Installer

- Require root only for host operations that need it.
- Detect existing installs and never overwrite persistent data blindly.
- Verify downloaded release artifacts using pinned checksums/signatures when available.
- Pin telemt version and checksum; do not silently install an unbounded latest artifact.
- Choose the Panel port only after an availability/conflict check and persist it before activation.
- Print the initial administrator credential only when necessary. Do not write that plaintext credential into normal service logs.

## Database

- Enable foreign keys.
- Use WAL mode and bounded busy timeout.
- Use transactions for authentication/bootstrap, credit/referral and other sensitive mutations.
- Back up before destructive migrations/restores/updates.

## Deferred hardening

TOTP, optional Panel IP allowlists, external TLS termination and encrypted-at-rest application secrets can be added after the core secure login/install path exists. Their absence must not justify weakening the baseline above.
