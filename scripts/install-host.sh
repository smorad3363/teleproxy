#!/usr/bin/env bash
set -Eeuo pipefail

SOURCE_DIR=""
while (($#)); do
  case "$1" in
    --source-dir)
      SOURCE_DIR=${2:-}
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

[[ -n "$SOURCE_DIR" && -d "$SOURCE_DIR" ]] || { echo "--source-dir is required" >&2; exit 2; }
# shellcheck source=scripts/install_lib.sh
source "$SOURCE_DIR/scripts/install_lib.sh"
[[ ${EUID:-$(id -u)} -eq 0 ]] || { echo "run installer as root (for example: sudo bash)" >&2; exit 1; }

for cmd in docker curl awk od tar cp mv mkdir chmod chown flock hostname install seq grep tr rm sleep sed; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "required command not found: $cmd" >&2; exit 1; }
done

docker compose version >/dev/null 2>&1 || { echo "Docker Compose v2 is required" >&2; exit 1; }
docker info >/dev/null 2>&1 || { echo "Docker daemon is not available" >&2; exit 1; }

INSTALL_DIR=${TPROXY_INSTALL_DIR:-/opt/teleproxy}
STATE_DIR="$INSTALL_DIR/state"
STATE_FILE="$STATE_DIR/install.env"
RUNTIME_SOURCE="$INSTALL_DIR/source"
DATA_DIR="$INSTALL_DIR/data"
PROXY_DATA_DIR="$INSTALL_DIR/proxy-data"
CONFIG_DIR="$INSTALL_DIR/config"
SECRETS_DIR="$INSTALL_DIR/secrets"
LOCK_FILE="$STATE_DIR/install.lock"
PANEL_BIND=${TPROXY_PANEL_BIND:-0.0.0.0}
PROXY_BIND=${TPROXY_PROXY_BIND:-0.0.0.0}
PROXY_PORT=${TPROXY_PROXY_PORT:-443}
REQUESTED_SOURCE_REF=${TPROXY_REF:-}
SOURCE_REF=${REQUESTED_SOURCE_REF:-main}
CONTROL_IMAGE=${TPROXY_CONTROL_IMAGE:-teleproxy/control:local}
TELEMT_IMAGE=${TPROXY_TELEMT_IMAGE:-teleproxy/telemt:3.5.7}
TELEMT_TLS_DOMAIN=${TPROXY_TELEMT_TLS_DOMAIN:-www.cloudflare.com}
ADMIN_USER=${TPROXY_ADMIN_USER:-admin}
TPROXY_BIN=${TPROXY_TPROXY_BIN:-/usr/local/bin/tproxy}
TELEMT_CONFIG="$CONFIG_DIR/telemt.toml"

mkdir -p "$STATE_DIR" "$DATA_DIR" "$PROXY_DATA_DIR" "$CONFIG_DIR" "$SECRETS_DIR"
chmod 0755 "$STATE_DIR"
exec 9>"$LOCK_FILE"
flock -n 9 || { echo "another Teleproxy installation is already running" >&2; exit 1; }

# Preserve installation choices unless an explicit first-class override is supported.
if [[ -f "$STATE_FILE" ]]; then
  persisted_admin=$(read_state_value "$STATE_FILE" TPROXY_ADMIN_USER 2>/dev/null || true)
  persisted_bind=$(read_state_value "$STATE_FILE" TPROXY_PANEL_BIND 2>/dev/null || true)
  persisted_image=$(read_state_value "$STATE_FILE" TPROXY_CONTROL_IMAGE 2>/dev/null || true)
  persisted_proxy_port=$(read_state_value "$STATE_FILE" TPROXY_PROXY_PORT 2>/dev/null || true)
  persisted_proxy_bind=$(read_state_value "$STATE_FILE" TPROXY_PROXY_BIND 2>/dev/null || true)
  persisted_telemt_image=$(read_state_value "$STATE_FILE" TPROXY_TELEMT_IMAGE 2>/dev/null || true)
  persisted_tls_domain=$(read_state_value "$STATE_FILE" TPROXY_TELEMT_TLS_DOMAIN 2>/dev/null || true)
  persisted_source_ref=$(read_state_value "$STATE_FILE" TPROXY_SOURCE_REF 2>/dev/null || true)
  [[ -n "$persisted_admin" ]] && ADMIN_USER=$persisted_admin
  [[ -n "$persisted_bind" ]] && PANEL_BIND=$persisted_bind
  [[ -n "$persisted_image" ]] && CONTROL_IMAGE=$persisted_image
  [[ -n "$persisted_proxy_port" ]] && PROXY_PORT=$persisted_proxy_port
  [[ -n "$persisted_proxy_bind" ]] && PROXY_BIND=$persisted_proxy_bind
  [[ -n "$persisted_telemt_image" ]] && TELEMT_IMAGE=$persisted_telemt_image
  [[ -n "$persisted_tls_domain" ]] && TELEMT_TLS_DOMAIN=$persisted_tls_domain
  [[ -z "$REQUESTED_SOURCE_REF" && -n "$persisted_source_ref" ]] && SOURCE_REF=$persisted_source_ref
fi

[[ "$ADMIN_USER" =~ ^[A-Za-z0-9_.-]{3,64}$ ]] || { echo "invalid administrator username" >&2; exit 1; }
validate_ipv4 "$PANEL_BIND" || { echo "TPROXY_PANEL_BIND must be an IPv4 address" >&2; exit 1; }
validate_ipv4 "$PROXY_BIND" || { echo "TPROXY_PROXY_BIND must be an IPv4 address" >&2; exit 1; }
validate_port "$PROXY_PORT" || { echo "TPROXY_PROXY_PORT must be between 1 and 65535" >&2; exit 1; }
validate_hostname "$TELEMT_TLS_DOMAIN" || { echo "invalid TPROXY_TELEMT_TLS_DOMAIN" >&2; exit 1; }
[[ "$SOURCE_REF" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]{0,199}$ ]] || { echo "invalid source ref" >&2; exit 1; }

# Stage source atomically without touching persistent data or state.
new_source="$INSTALL_DIR/.source.new.$$"
old_source="$INSTALL_DIR/.source.previous"
rm -rf "$new_source"
mkdir -p "$new_source"
cp -a "$SOURCE_DIR"/. "$new_source"/
rm -rf "$old_source"
if [[ -d "$RUNTIME_SOURCE" ]]; then
  mv "$RUNTIME_SOURCE" "$old_source"
fi
mv "$new_source" "$RUNTIME_SOURCE"

persisted_port=""
if [[ -f "$STATE_FILE" ]]; then
  persisted_port=$(read_state_value "$STATE_FILE" TPROXY_PANEL_PORT 2>/dev/null || true)
fi
if [[ -n "$persisted_port" ]]; then
  validate_port "$persisted_port" || { echo "invalid persisted panel port" >&2; exit 1; }
  PANEL_PORT=$persisted_port
  FRESH_PORT=0
else
  PANEL_PORT=$(choose_free_port)
  FRESH_PORT=1
fi

if [[ "$PANEL_BIND" == "$PROXY_BIND" && "$PANEL_PORT" == "$PROXY_PORT" ]]; then
  if [[ "$FRESH_PORT" == "1" ]]; then
    while [[ "$PANEL_PORT" == "$PROXY_PORT" ]]; do PANEL_PORT=$(choose_free_port); done
  else
    echo "persisted panel port conflicts with proxy port $PROXY_PORT" >&2
    exit 1
  fi
fi

credential_printed=$(read_state_value "$STATE_FILE" TPROXY_CREDENTIAL_PRINTED 2>/dev/null || printf '0')
[[ "$credential_printed" == "0" || "$credential_printed" == "1" ]] || credential_printed=0

password_file="$SECRETS_DIR/admin-bootstrap-password"
api_token_file="$SECRETS_DIR/telemt-api-token"
bootstrap_proxy_secret_file="$SECRETS_DIR/telemt-bootstrap-user-secret"
initial_password=""
if [[ "$credential_printed" == "0" ]]; then
  if [[ -f "$password_file" ]]; then
    initial_password=$(tr -d '\r\n' <"$password_file")
  else
    initial_password=$(random_hex 24)
    umask 077
    printf '%s\n' "$initial_password" >"$password_file"
  fi
fi

if [[ ! -f "$api_token_file" ]]; then
  umask 077
  random_hex 32 >"$api_token_file"
fi
if [[ ! -f "$bootstrap_proxy_secret_file" ]]; then
  umask 077
  random_hex 16 >"$bootstrap_proxy_secret_file"
fi
api_token=$(tr -d '\r\n' <"$api_token_file")
bootstrap_proxy_secret=$(tr -d '\r\n' <"$bootstrap_proxy_secret_file")
[[ $api_token =~ ^[0-9a-f]{64}$ ]] || { echo "invalid persisted Telemt API token" >&2; exit 1; }
[[ $bootstrap_proxy_secret =~ ^[0-9a-f]{32}$ ]] || { echo "invalid persisted Telemt bootstrap secret" >&2; exit 1; }

render_telemt_config \
  "$RUNTIME_SOURCE/config/telemt.toml.template" \
  "$TELEMT_CONFIG" \
  "$api_token" \
  "$bootstrap_proxy_secret" \
  "$TELEMT_TLS_DOMAIN"
unset api_token bootstrap_proxy_secret

# UID 10001 is the Control Plane user; distroless nonroot Telemt uses UID 65532.
chown -R 10001:10001 "$DATA_DIR"
chmod 0700 "$DATA_DIR"
chown -R 65532:65532 "$PROXY_DATA_DIR"
chmod 0700 "$PROXY_DATA_DIR"
chown 0:65532 "$CONFIG_DIR"
chmod 0710 "$CONFIG_DIR"
chown 65532:65532 "$TELEMT_CONFIG"
chmod 0600 "$TELEMT_CONFIG"
chown 0:10001 "$SECRETS_DIR"
chmod 0710 "$SECRETS_DIR"
if [[ -f "$password_file" ]]; then
  chown 10001:10001 "$password_file"
  chmod 0600 "$password_file"
fi
chown 10001:10001 "$api_token_file"
chmod 0600 "$api_token_file"
chown 0:0 "$bootstrap_proxy_secret_file"
chmod 0600 "$bootstrap_proxy_secret_file"

save_state() {
  local phase=${1:?phase required}
  local printed=${2:?credential flag required}
  write_install_state "$STATE_FILE" "$phase" "$PANEL_PORT" "$PANEL_BIND" "$ADMIN_USER" "$SOURCE_REF" "$printed" \
    "$DATA_DIR" "$SECRETS_DIR" "$CONTROL_IMAGE" "$PROXY_PORT" "$PROXY_BIND" "$PROXY_DATA_DIR" "$CONFIG_DIR" \
    "$TELEMT_CONFIG" "$TELEMT_IMAGE" "$TELEMT_TLS_DOMAIN"
}
save_state prepared "$credential_printed"

compose() {
  docker compose --project-name teleproxy --env-file "$STATE_FILE" -f "$RUNTIME_SOURCE/compose.yaml" "$@"
}

existing_control_id=$(compose ps -q control 2>/dev/null || true)
existing_proxy_id=$(compose ps -q telemt 2>/dev/null || true)
if [[ -n "$persisted_port" && -z "$existing_control_id" ]] && port_in_use "$PANEL_PORT"; then
  echo "persisted panel port $PANEL_PORT is already in use; refusing to silently change it" >&2
  exit 1
fi
if [[ -z "$existing_proxy_id" ]] && port_in_use "$PROXY_PORT"; then
  echo "proxy port $PROXY_PORT is already in use; choose TPROXY_PROXY_PORT on the first install" >&2
  exit 1
fi

start_output=""
start_ok=0
for _ in 1 2 3 4 5; do
  if start_output=$(compose up -d --build 2>&1); then
    start_ok=1
    break
  fi

  if [[ "$FRESH_PORT" == "1" ]] && grep -Eqi 'port is already allocated|address already in use|bind:.*failed' <<<"$start_output"; then
    PANEL_PORT=$(choose_free_port)
    [[ "$PANEL_PORT" == "$PROXY_PORT" ]] && continue
    save_state prepared "$credential_printed"
    continue
  fi
  break
done

if [[ "$start_ok" != "1" ]]; then
  printf '%s\n' "$start_output" >&2
  if [[ -d "$old_source" ]]; then
    echo "new source failed to start; previous source retained at $old_source" >&2
  fi
  exit 1
fi

control_ready=0
for _ in $(seq 1 60); do
  if curl -fsS --max-time 2 "http://127.0.0.1:${PANEL_PORT}/readyz" >/dev/null 2>&1; then
    control_ready=1
    break
  fi
  sleep 1
done
if [[ "$control_ready" != "1" ]]; then
  echo "control plane did not become ready" >&2
  compose ps >&2 || true
  compose logs --tail=80 control >&2 || true
  exit 1
fi

control_healthy=0
control_health=""
control_state=""
control_id=""
for _ in $(seq 1 60); do
  control_id=$(compose ps -q control 2>/dev/null || true)
  if [[ -n "$control_id" ]]; then
    control_health=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$control_id" 2>/dev/null || true)
    if [[ "$control_health" == "healthy" ]]; then
      control_healthy=1
      break
    fi
  fi
  sleep 1
done
if [[ "$control_healthy" != "1" ]]; then
  if [[ -n "$control_id" ]]; then
    control_state=$(docker inspect -f '{{.State.Status}}' "$control_id" 2>/dev/null || true)
  fi
  echo "Control container did not become healthy: container=${control_state:-missing} health=${control_health:-missing}" >&2
  compose ps >&2 || true
  exit 1
fi

proxy_healthy=0
for _ in $(seq 1 60); do
  proxy_id=$(compose ps -q telemt 2>/dev/null || true)
  if [[ -n "$proxy_id" ]]; then
    proxy_health=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$proxy_id" 2>/dev/null || true)
    if [[ "$proxy_health" == "healthy" ]]; then
      proxy_healthy=1
      break
    fi
  fi
  sleep 1
done
if [[ "$proxy_healthy" != "1" ]]; then
  echo "Telemt proxy did not become healthy" >&2
  compose ps >&2 || true
  compose logs --tail=120 telemt >&2 || true
  exit 1
fi

host_ip=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
if [[ "$PANEL_BIND" == "127.0.0.1" ]]; then
  panel_host=127.0.0.1
elif [[ -n "$host_ip" ]]; then
  panel_host=$host_ip
else
  panel_host='<server-ip>'
fi

printf '\nTeleproxy installation complete\n'
printf '%s\n' '----------------------------------------'
printf 'Panel URL:        http://%s:%s\n' "$panel_host" "$PANEL_PORT"
printf 'Panel Port:       %s\n' "$PANEL_PORT"
printf 'Admin User:       %s\n' "$ADMIN_USER"
if [[ "$credential_printed" == "0" && -n "$initial_password" ]]; then
  printf 'Initial Password: %s\n' "$initial_password"
else
  printf 'Initial Password: (already displayed on first successful install)\n'
fi
printf 'Control Plane:    ready\n'
printf 'Proxy Plane:      healthy (Telemt 3.5.7)\n'
printf 'Proxy Port:       %s\n' "$PROXY_PORT"
printf 'Telemt API:       internal-only, authenticated\n'
printf 'Source Ref:       %s\n' "$SOURCE_REF"
printf 'Install Path:     %s\n' "$INSTALL_DIR"
printf 'Data Path:        %s\n' "$DATA_DIR"
printf 'Config Path:      %s\n' "$CONFIG_DIR"
printf 'Management:       tproxy status | tproxy doctor | tproxy proxy status | tproxy panel\n'
printf '%s\n' '----------------------------------------'
printf '%s\n' 'Security note: the Web Panel is HTTP-only at this stage; expose it only on a trusted interface until TLS hardening.'

if [[ "$credential_printed" == "0" ]]; then
  credential_printed=1
fi
save_state installed "$credential_printed"
rm -f "$password_file"

install -m 0755 "$RUNTIME_SOURCE/bin/tproxy" "$TPROXY_BIN"
rm -rf "$old_source"
