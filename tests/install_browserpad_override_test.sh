#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

CONFIG_FILE="$TMP_DIR/retroarch.cfg"
cat >"$CONFIG_FILE" <<'CFG'
config_save_on_exit = "true"
CFG

bash "$ROOT_DIR/setup/install.sh" --write-browserpad-override "$CONFIG_FILE"

grep -F 'all_users_control_menu = "true"' "$CONFIG_FILE"
grep -F 'input_player1_reserved_device = "BrowserPad 1"' "$CONFIG_FILE"
grep -F 'input_player8_device_reservation_type = "2"' "$CONFIG_FILE"

bash "$ROOT_DIR/setup/install.sh" --write-browserpad-override "$CONFIG_FILE"

block_count="$(grep -c '>>> retropie-controller BrowserPad reservations >>>' "$CONFIG_FILE")"
if [[ "$block_count" -ne 1 ]]; then
  echo "expected one managed BrowserPad block, found $block_count"
  exit 1
fi
