# Teleproxy

Teleproxy is a lightweight control plane for managing Telegram MTProto proxy access.

The project is being implemented incrementally from the supplied engineering roadmap. Current work lives on `agent/mvp-bootstrap` until the bootstrap/MVP milestones are verified.

## Development

Run the Go test suite with:

```bash
go test ./...
```

The Docker-first installer and integration checks run in CI. The Control Plane intentionally does not mount the Docker socket.

## Install (development branch)

```bash
curl -fsSL https://raw.githubusercontent.com/smorad3363/teleproxy/agent/mvp-bootstrap/install.sh | sudo env TPROXY_REF=agent/mvp-bootstrap bash
```

After merge to `main`, the production command will be:

```bash
curl -fsSL https://raw.githubusercontent.com/smorad3363/teleproxy/main/install.sh | sudo bash
```
