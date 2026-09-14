# teleproxy

Lightweight, Docker-first Telegram MTProto proxy management platform built around a separate Control Plane and `telemt` data plane.

> Active implementation and recovery state is tracked under `docs/exec-plans/active/`.

## Current bootstrap

Stage 5 provides the first Docker-first Control Plane installer. It selects a random available high TCP port for the panel, persists that port across reruns, creates the initial administrator credential, starts the Control Plane, waits for readiness, and prints the panel URL and login details.

After this work reaches `main`:

```bash
curl -fsSL https://raw.githubusercontent.com/smorad3363/teleproxy/main/install.sh | sudo bash
```

For the current development branch, with the Web Panel reachable directly from the server's network interfaces:

```bash
curl -fsSL https://raw.githubusercontent.com/smorad3363/teleproxy/agent/mvp-bootstrap/install.sh \
  | sudo env TPROXY_REF=agent/mvp-bootstrap TPROXY_PANEL_BIND=0.0.0.0 bash
```

An explicit `TPROXY_PANEL_BIND` is a first-class override on reruns. This means the command above can change an existing loopback-only installation from `127.0.0.1` to `0.0.0.0` without changing its persisted panel port. If `TPROXY_PANEL_BIND` is omitted, the installer preserves the previously persisted bind address.

The installer does not silently change an already-persisted panel port. If that port becomes occupied by another process while Teleproxy is stopped, installation fails with an explicit conflict instead.

Management commands after installation:

```bash
tproxy status
tproxy logs
tproxy restart
tproxy panel
```

The Stage 5 panel endpoint is HTTP-only. Binding it to `0.0.0.0` makes it reachable on network interfaces but does not configure TLS or a host/cloud firewall. Restrict the panel to trusted networks until TLS/public-exposure hardening is implemented.
