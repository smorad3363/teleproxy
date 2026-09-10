#!/usr/bin/env bash
set -Eeuo pipefail

[[ ${EUID:-$(id -u)} -eq 0 ]] || { echo "run as root, e.g. curl ... | sudo bash" >&2; exit 1; }
for cmd in curl tar mktemp find bash awk head rm; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "required command not found: $cmd" >&2; exit 1; }
done

REPO=${TPROXY_REPO:-smorad3363/teleproxy}
INSTALL_DIR=${TPROXY_INSTALL_DIR:-/opt/teleproxy}
STATE_FILE="$INSTALL_DIR/state/install.env"
if [[ -n ${TPROXY_REF:-} ]]; then
  REF=$TPROXY_REF
elif [[ -f "$STATE_FILE" ]]; then
  REF=$(awk -F= '$1 == "TPROXY_SOURCE_REF" {sub(/^[^=]*=/, ""); print; exit}' "$STATE_FILE")
  [[ -n "$REF" ]] || REF=main
else
  REF=main
fi
[[ "$REF" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]{0,199}$ ]] || { echo "invalid TPROXY_REF" >&2; exit 1; }
TMP_DIR=$(mktemp -d -t teleproxy-install.XXXXXX)
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

archive="$TMP_DIR/source.tar.gz"
url="https://codeload.github.com/${REPO}/tar.gz/refs/heads/${REF}"
echo "Downloading Teleproxy source ref: $REF"
curl -fL --retry 3 --connect-timeout 10 "$url" -o "$archive"
tar -xzf "$archive" -C "$TMP_DIR"
source_dir=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d -not -path "$TMP_DIR" | head -n1)
[[ -n "$source_dir" && -x "$source_dir/scripts/install-host.sh" ]] || { echo "installer payload is incomplete" >&2; exit 1; }

TPROXY_REF="$REF" bash "$source_dir/scripts/install-host.sh" --source-dir "$source_dir"
