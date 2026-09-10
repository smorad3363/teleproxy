# teleproxy

Lightweight, Docker-first Telegram MTProto proxy management platform built around a separate Control Plane and `telemt` data plane.

> Active implementation and recovery state is tracked under `docs/exec-plans/active/`.

## Current bootstrap

Stage 5 provides the first Docker-first Control Plane installer. It selects a random available high TCP port for the panel, persists that port across reruns, creates the initial administrator credential, starts the Control Plane, waits for readiness, and prints the panel URL and login details.

After this work reaches `main`:

```bash
curl -fsSL https://raw.githubusercontent.com/smorad3363/teleproxy/main/install.sh | sudo bash
```

For the current development branch:

```bash
curl -fsSL https://raw.githubusercontent.com/smorad3363/teleproxy/agent/mvp-bootstrap/install.sh \
  | sudo env TPROXY_REF=agent/mvp-bootstrap bash
```

The installer does not silently change an already-persisted panel port. If that port becomes occupied by another process while Teleproxy is stopped, installation fails with an explicit conflict instead.

Management commands after installation:

```bash
tproxy status
tproxy logs
tproxy restart
tproxy panel
```

The Stage 5 panel endpoint is HTTP-only. Do not treat the random port as a security control; TLS/public exposure hardening is a separate stage.
