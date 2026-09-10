#!/usr/bin/env bash
set -Eeuo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
# shellcheck source=../scripts/install_lib.sh
source "$ROOT/scripts/install_lib.sh"

tmp=$(mktemp -d)
cleanup() {
  if [[ -n ${listener_pid:-} ]]; then kill "$listener_pid" 2>/dev/null || true; fi
  rm -rf "$tmp"
}
trap cleanup EXIT

port=$(choose_free_port 30000 45000 64)
validate_port "$port"
validate_ipv4 127.0.0.1
! validate_ipv4 999.0.0.1
((port >= 30000 && port <= 45000))
if port_in_use "$port"; then
  echo "chosen port unexpectedly reported in use" >&2
  exit 1
fi

if command -v python3 >/dev/null 2>&1; then
  python3 -m http.server "$port" --bind 127.0.0.1 >/dev/null 2>&1 &
  listener_pid=$!
  for _ in $(seq 1 20); do
    if port_in_use "$port"; then break; fi
    sleep 0.05
  done
  port_in_use "$port"
  kill "$listener_pid"
  wait "$listener_pid" 2>/dev/null || true
  listener_pid=""
fi

state="$tmp/install.env"
write_install_state "$state" prepared "$port" 0.0.0.0 admin agent/mvp-bootstrap 0 "$tmp/data" "$tmp/secrets" teleproxy/control:test
[[ $(read_state_value "$state" TPROXY_PANEL_PORT) == "$port" ]]
[[ $(resolve_panel_port "$state") == "$port" ]]
[[ $(read_state_value "$state" TPROXY_CREDENTIAL_PRINTED) == 0 ]]

write_install_state "$state" installed "$port" 0.0.0.0 admin agent/mvp-bootstrap 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test
[[ $(resolve_panel_port "$state") == "$port" ]]
[[ $(read_state_value "$state" TPROXY_INSTALL_PHASE) == installed ]]
[[ $(read_state_value "$state" TPROXY_CREDENTIAL_PRINTED) == 1 ]]

if write_install_state "$state" installed 70000 0.0.0.0 admin main 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test 2>/dev/null; then
  echo "invalid port was accepted" >&2
  exit 1
fi

if write_install_state "$state" installed "$port" 0.0.0.0 admin 'bad ref with spaces' 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test 2>/dev/null; then
  echo "unsafe source ref was accepted" >&2
  exit 1
fi

echo "installer library tests: PASS"
