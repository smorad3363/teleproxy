#!/usr/bin/env bash
set -Eeuo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TMP=$(mktemp -d -t teleproxy-e2e.XXXXXX)
INSTALL_DIR="$TMP/install"
TPROXY_BIN="$TMP/tproxy"
OUT1="$TMP/first.out"
OUT2="$TMP/second.out"
# Keep the proxy test port outside the panel random range to avoid accidental collision.
PROXY_PORT=$(bash -c "source '$ROOT/scripts/install_lib.sh'; choose_free_port 12000 19000 64")

cleanup() {
  if [[ -f "$INSTALL_DIR/state/install.env" && -f "$INSTALL_DIR/source/compose.yaml" ]]; then
    sudo docker compose --project-name teleproxy \
      --env-file "$INSTALL_DIR/state/install.env" \
      -f "$INSTALL_DIR/source/compose.yaml" down --remove-orphans >/dev/null 2>&1 || true
  fi
  sudo rm -rf "$TMP"
}
trap cleanup EXIT

run_install() {
  local out=$1
  sudo env \
    PATH="$PATH" \
    TPROXY_INSTALL_DIR="$INSTALL_DIR" \
    TPROXY_PANEL_BIND=127.0.0.1 \
    TPROXY_PROXY_BIND=127.0.0.1 \
    TPROXY_PROXY_PORT="$PROXY_PORT" \
    TPROXY_REF=ci \
    TPROXY_CONTROL_IMAGE=teleproxy/control:ci \
    TPROXY_TELEMT_IMAGE=teleproxy/telemt:ci \
    TPROXY_TPROXY_BIN="$TPROXY_BIN" \
    bash "$ROOT/scripts/install-host.sh" --source-dir "$ROOT" >"$out"
}

compose() {
  sudo docker compose --project-name teleproxy \
    --env-file "$INSTALL_DIR/state/install.env" \
    -f "$INSTALL_DIR/source/compose.yaml" "$@"
}

run_install "$OUT1"
state="$INSTALL_DIR/state/install.env"
[[ -f "$state" ]]
port1=$(sudo awk -F= '$1 == "TPROXY_PANEL_PORT" {print $2}' "$state")
phase1=$(sudo awk -F= '$1 == "TPROXY_INSTALL_PHASE" {print $2}' "$state")
printed1=$(sudo awk -F= '$1 == "TPROXY_CREDENTIAL_PRINTED" {print $2}' "$state")
proxy_port1=$(sudo awk -F= '$1 == "TPROXY_PROXY_PORT" {print $2}' "$state")
[[ "$phase1" == installed ]]
[[ "$printed1" == 1 ]]
[[ "$proxy_port1" == "$PROXY_PORT" ]]
grep -Eq '^Initial Password: [0-9a-f]{48}$' "$OUT1"
grep -F 'Proxy Plane:      healthy (Telemt 3.5.7)' "$OUT1" >/dev/null
[[ ! -e "$INSTALL_DIR/secrets/admin-bootstrap-password" ]]
curl -fsS --max-time 3 "http://127.0.0.1:${port1}/readyz" >/dev/null
TPROXY_INSTALL_DIR="$INSTALL_DIR" "$TPROXY_BIN" panel | grep -F "http://127.0.0.1:${port1}" >/dev/null
TPROXY_INSTALL_DIR="$INSTALL_DIR" "$TPROXY_BIN" proxy status | grep -F 'Proxy Plane: healthy' >/dev/null

# The proxy listener is public/configurable, but the Telemt Control API must not be published.
telemt_id=$(compose ps -q telemt)
[[ -n "$telemt_id" ]]
api_bindings=$(sudo docker inspect -f '{{with index .HostConfig.PortBindings "9091/tcp"}}{{json .}}{{end}}' "$telemt_id")
if [[ -n "$api_bindings" && "$api_bindings" != "null" && "$api_bindings" != "[]" ]]; then
  echo "Telemt API port 9091 was published to the host: $api_bindings" >&2
  exit 1
fi
telemt_ip=$(sudo docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$telemt_id")
[[ -n "$telemt_ip" ]]
unauth_code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "http://${telemt_ip}:9091/v1/health")
[[ "$unauth_code" != 200 ]]
api_token=$(sudo cat "$INSTALL_DIR/secrets/telemt-api-token" | tr -d '\r\n')
[[ $api_token =~ ^[0-9a-f]{64}$ ]]
auth_code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -H "Authorization: Bearer ${api_token}" "http://${telemt_ip}:9091/v1/health")
[[ "$auth_code" == 200 ]]
if sudo grep -F "$api_token" "$state" >/dev/null || grep -F "$api_token" "$OUT1" >/dev/null; then
  echo "Telemt API token leaked into state or installer output" >&2
  exit 1
fi
unset api_token

run_install "$OUT2"
port2=$(sudo awk -F= '$1 == "TPROXY_PANEL_PORT" {print $2}' "$state")
proxy_port2=$(sudo awk -F= '$1 == "TPROXY_PROXY_PORT" {print $2}' "$state")
[[ "$port2" == "$port1" ]]
[[ "$proxy_port2" == "$proxy_port1" ]]
grep -F 'Initial Password: (already displayed on first successful install)' "$OUT2" >/dev/null
if grep -Eq '^Initial Password: [0-9a-f]{48}$' "$OUT2"; then
  echo "initial password was reprinted on rerun" >&2
  exit 1
fi
curl -fsS --max-time 3 "http://127.0.0.1:${port2}/readyz" >/dev/null
TPROXY_INSTALL_DIR="$INSTALL_DIR" "$TPROXY_BIN" proxy status | grep -F 'Proxy Plane: healthy' >/dev/null

echo "installer + Telemt end-to-end rerun test: PASS"
