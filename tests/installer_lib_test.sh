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
validate_hostname proxy.example.com
! validate_hostname 'bad host'
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

api_token=$(printf 'a%.0s' {1..64})
bootstrap_secret=$(printf 'b%.0s' {1..32})
rendered="$tmp/telemt.toml"
render_telemt_config "$ROOT/config/telemt.toml.template" "$rendered" "$api_token" "$bootstrap_secret" proxy.example.com
grep -F "Bearer $api_token" "$rendered" >/dev/null
grep -F "teleproxy_bootstrap = \"$bootstrap_secret\"" "$rendered" >/dev/null
if grep -F '__TELEMT_' "$rendered" >/dev/null; then
  echo "Telemt template placeholder remained after render" >&2
  exit 1
fi
if command -v python3 >/dev/null 2>&1; then
  python3 - "$rendered" <<'PY'
import pathlib, sys, tomllib
with pathlib.Path(sys.argv[1]).open('rb') as f:
    cfg = tomllib.load(f)
assert cfg['server']['api']['enabled'] is True
assert cfg['access']['user_enabled']['teleproxy_bootstrap'] is False
PY
fi

state="$tmp/install.env"
write_install_state "$state" prepared "$port" 0.0.0.0 admin agent/mvp-bootstrap 0 "$tmp/data" "$tmp/secrets" teleproxy/control:test 443 0.0.0.0 "$tmp/proxy-data" "$tmp/config" "$rendered" teleproxy/telemt:3.5.7 proxy.example.com
[[ $(read_state_value "$state" TPROXY_PANEL_PORT) == "$port" ]]
[[ $(resolve_panel_port "$state") == "$port" ]]
[[ $(read_state_value "$state" TPROXY_CREDENTIAL_PRINTED) == 0 ]]
[[ $(read_state_value "$state" TPROXY_PROXY_PORT) == 443 ]]
[[ $(read_state_value "$state" TPROXY_TELEMT_VERSION) == 3.5.7 ]]

write_install_state "$state" installed "$port" 0.0.0.0 admin agent/mvp-bootstrap 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test 443 0.0.0.0 "$tmp/proxy-data" "$tmp/config" "$rendered" teleproxy/telemt:3.5.7 proxy.example.com
[[ $(resolve_panel_port "$state") == "$port" ]]
[[ $(read_state_value "$state" TPROXY_INSTALL_PHASE) == installed ]]
[[ $(read_state_value "$state" TPROXY_CREDENTIAL_PRINTED) == 1 ]]

if write_install_state "$state" installed 70000 0.0.0.0 admin main 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test 443 0.0.0.0 "$tmp/proxy-data" "$tmp/config" "$rendered" teleproxy/telemt:3.5.7 proxy.example.com 2>/dev/null; then
  echo "invalid panel port was accepted" >&2
  exit 1
fi
if write_install_state "$state" installed "$port" 0.0.0.0 admin main 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test 70000 0.0.0.0 "$tmp/proxy-data" "$tmp/config" "$rendered" teleproxy/telemt:3.5.7 proxy.example.com 2>/dev/null; then
  echo "invalid proxy port was accepted" >&2
  exit 1
fi
if write_install_state "$state" installed "$port" 0.0.0.0 admin 'bad ref with spaces' 1 "$tmp/data" "$tmp/secrets" teleproxy/control:test 443 0.0.0.0 "$tmp/proxy-data" "$tmp/config" "$rendered" teleproxy/telemt:3.5.7 proxy.example.com 2>/dev/null; then
  echo "unsafe source ref was accepted" >&2
  exit 1
fi

echo "installer library tests: PASS"
