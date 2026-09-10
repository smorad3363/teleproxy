#!/usr/bin/env bash
set -Eeuo pipefail

TPROXY_PORT_MIN_DEFAULT=20000
TPROXY_PORT_MAX_DEFAULT=60000

is_uint() {
  [[ ${1:-} =~ ^[0-9]+$ ]]
}

validate_ipv4() {
  local ip=${1:-}
  [[ $ip =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || return 1
  local IFS=. octet
  read -r -a octets <<<"$ip"
  for octet in "${octets[@]}"; do
    ((10#$octet >= 0 && 10#$octet <= 255)) || return 1
  done
}

validate_port() {
  local value=${1:-}
  is_uint "$value" || return 1
  (( value >= 1 && value <= 65535 ))
}

validate_hostname() {
  local host=${1:-}
  [[ ${#host} -ge 1 && ${#host} -le 253 ]] || return 1
  [[ $host =~ ^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$ ]] || return 1
  [[ $host == *.* ]] || return 1
  [[ $host != *..* ]] || return 1
}

random_u32() {
  local hex
  hex=$(od -An -N4 -tx4 /dev/urandom | tr -d '[:space:]')
  printf '%u\n' "$((16#$hex))"
}

random_hex() {
  local bytes=${1:?byte count required}
  is_uint "$bytes" || return 2
  ((bytes > 0 && bytes <= 128)) || return 2
  od -An -N"$bytes" -tx1 /dev/urandom | tr -d ' \n'
  printf '\n'
}

port_in_use() {
  local port=${1:?port required}
  validate_port "$port" || return 2

  if command -v ss >/dev/null 2>&1; then
    ss -H -ltn 2>/dev/null | awk -v p=":${port}" '$4 ~ p"$" {found=1} END {exit found ? 0 : 1}'
    return $?
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 - "$port" <<'PY'
import socket, sys
port = int(sys.argv[1])
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 0)
try:
    s.bind(("0.0.0.0", port))
except OSError:
    raise SystemExit(0)
else:
    raise SystemExit(1)
finally:
    s.close()
PY
    return $?
  fi

  if (echo >/dev/tcp/127.0.0.1/"$port") >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

choose_free_port() {
  local min=${1:-$TPROXY_PORT_MIN_DEFAULT}
  local max=${2:-$TPROXY_PORT_MAX_DEFAULT}
  local attempts=${3:-128}

  validate_port "$min" || return 2
  validate_port "$max" || return 2
  is_uint "$attempts" || return 2
  (( min < max && attempts > 0 )) || return 2

  local span=$((max - min + 1))
  local i candidate r
  for ((i = 0; i < attempts; i++)); do
    r=$(random_u32)
    candidate=$((min + (r % span)))
    if ! port_in_use "$candidate"; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  return 1
}

read_state_value() {
  local file=${1:?state file required}
  local key=${2:?key required}
  [[ -f "$file" ]] || return 1
  awk -F= -v k="$key" '$1 == k {sub(/^[^=]*=/, ""); print; found=1; exit} END {exit found ? 0 : 1}' "$file"
}

resolve_panel_port() {
  local state_file=${1:?state file required}
  local persisted=""
  if persisted=$(read_state_value "$state_file" TPROXY_PANEL_PORT 2>/dev/null); then
    validate_port "$persisted" || return 2
    printf '%s\n' "$persisted"
    return 0
  fi
  choose_free_port "${TPROXY_PORT_MIN:-$TPROXY_PORT_MIN_DEFAULT}" "${TPROXY_PORT_MAX:-$TPROXY_PORT_MAX_DEFAULT}"
}

render_telemt_config() {
  local template=${1:?template required}
  local output=${2:?output required}
  local api_token=${3:?api token required}
  local bootstrap_secret=${4:?bootstrap secret required}
  local tls_domain=${5:?tls domain required}

  [[ -f "$template" ]] || return 2
  [[ $api_token =~ ^[0-9a-f]{64}$ ]] || return 2
  [[ $bootstrap_secret =~ ^[0-9a-f]{32}$ ]] || return 2
  validate_hostname "$tls_domain" || return 2

  local tmp="${output}.tmp.$$"
  sed \
    -e "s/__TELEMT_API_TOKEN__/${api_token}/g" \
    -e "s/__TELEMT_BOOTSTRAP_SECRET__/${bootstrap_secret}/g" \
    -e "s/__TELEMT_TLS_DOMAIN__/${tls_domain}/g" \
    "$template" >"$tmp"
  chmod 0600 "$tmp"
  mv -f "$tmp" "$output"
}

write_install_state() {
  local file=${1:?state file required}
  local phase=${2:?phase required}
  local panel_port=${3:?panel port required}
  local panel_bind=${4:?panel bind required}
  local admin_user=${5:?admin user required}
  local source_ref=${6:?source ref required}
  local credential_printed=${7:?credential flag required}
  local data_dir=${8:?data dir required}
  local secrets_dir=${9:?secrets dir required}
  local control_image=${10:?control image required}
  local proxy_port=${11:-443}
  local proxy_bind=${12:-0.0.0.0}
  local proxy_data_dir=${13:-${data_dir%/}/telemt}
  local config_dir=${14:-${data_dir%/}/config}
  local telemt_config=${15:-${config_dir%/}/telemt.toml}
  local telemt_image=${16:-teleproxy/telemt:3.5.7}
  local tls_domain=${17:-www.cloudflare.com}

  validate_port "$panel_port" || return 2
  validate_port "$proxy_port" || return 2
  [[ "$phase" =~ ^[a-z_]+$ ]] || return 2
  [[ "$credential_printed" == "0" || "$credential_printed" == "1" ]] || return 2
  [[ "$admin_user" =~ ^[A-Za-z0-9_.-]{3,64}$ ]] || return 2
  validate_ipv4 "$panel_bind" || return 2
  validate_ipv4 "$proxy_bind" || return 2
  validate_hostname "$tls_domain" || return 2
  [[ "$source_ref" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]{0,199}$ ]] || return 2
  [[ "$data_dir" != *$'\n'* && "$secrets_dir" != *$'\n'* && "$control_image" != *$'\n'* ]] || return 2
  [[ "$proxy_data_dir" != *$'\n'* && "$config_dir" != *$'\n'* && "$telemt_config" != *$'\n'* && "$telemt_image" != *$'\n'* ]] || return 2

  local tmp="${file}.tmp.$$"
  umask 077
  cat >"$tmp" <<STATE
TPROXY_INSTALL_PHASE=$phase
TPROXY_PANEL_PORT=$panel_port
TPROXY_PANEL_BIND=$panel_bind
TPROXY_ADMIN_USER=$admin_user
TPROXY_SOURCE_REF=$source_ref
TPROXY_CREDENTIAL_PRINTED=$credential_printed
TPROXY_DATA_DIR=$data_dir
TPROXY_SECRETS_DIR=$secrets_dir
TPROXY_CONTROL_IMAGE=$control_image
TPROXY_PROXY_PORT=$proxy_port
TPROXY_PROXY_BIND=$proxy_bind
TPROXY_PROXY_DATA_DIR=$proxy_data_dir
TPROXY_CONFIG_DIR=$config_dir
TPROXY_TELEMT_CONFIG=$telemt_config
TPROXY_TELEMT_IMAGE=$telemt_image
TPROXY_TELEMT_VERSION=3.5.7
TPROXY_TELEMT_TLS_DOMAIN=$tls_domain
TPROXY_COOKIE_SECURE=false
STATE
  chmod 0644 "$tmp"
  mv -f "$tmp" "$file"
}
