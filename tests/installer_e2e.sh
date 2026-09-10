#!/usr/bin/env bash
set -Eeuo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TMP=$(mktemp -d -t teleproxy-e2e.XXXXXX)
INSTALL_DIR="$TMP/install"
TPROXY_BIN="$TMP/tproxy"
OUT1="$TMP/first.out"
OUT2="$TMP/second.out"

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
    TPROXY_REF=ci \
    TPROXY_CONTROL_IMAGE=teleproxy/control:ci \
    TPROXY_TPROXY_BIN="$TPROXY_BIN" \
    bash "$ROOT/scripts/install-host.sh" --source-dir "$ROOT" >"$out"
}

run_install "$OUT1"
state="$INSTALL_DIR/state/install.env"
[[ -f "$state" ]]
port1=$(sudo awk -F= '$1 == "TPROXY_PANEL_PORT" {print $2}' "$state")
phase1=$(sudo awk -F= '$1 == "TPROXY_INSTALL_PHASE" {print $2}' "$state")
printed1=$(sudo awk -F= '$1 == "TPROXY_CREDENTIAL_PRINTED" {print $2}' "$state")
[[ "$phase1" == installed ]]
[[ "$printed1" == 1 ]]
grep -Eq '^Initial Password: [0-9a-f]{48}$' "$OUT1"
[[ ! -e "$INSTALL_DIR/secrets/admin-bootstrap-password" ]]
curl -fsS --max-time 3 "http://127.0.0.1:${port1}/readyz" >/dev/null
TPROXY_INSTALL_DIR="$INSTALL_DIR" "$TPROXY_BIN" panel | grep -F "http://127.0.0.1:${port1}" >/dev/null

run_install "$OUT2"
port2=$(sudo awk -F= '$1 == "TPROXY_PANEL_PORT" {print $2}' "$state")
[[ "$port2" == "$port1" ]]
grep -F 'Initial Password: (already displayed on first successful install)' "$OUT2" >/dev/null
if grep -Eq '^Initial Password: [0-9a-f]{48}$' "$OUT2"; then
  echo "initial password was reprinted on rerun" >&2
  exit 1
fi
curl -fsS --max-time 3 "http://127.0.0.1:${port2}/readyz" >/dev/null

echo "installer end-to-end rerun test: PASS"
