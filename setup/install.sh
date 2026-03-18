#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${RETROPIE_CONTROLLER_APP_DIR:-/home/pi/retropie-controller}"
SERVICE_NAME="retropie-controller.service"
OVERRIDE_BEGIN="# >>> retropie-controller BrowserPad reservations >>>"
OVERRIDE_END="# <<< retropie-controller BrowserPad reservations <<<"

print_browserpad_override() {
  echo "$OVERRIDE_BEGIN"
  echo 'all_users_control_menu = "true"'
  for player in $(seq 1 8); do
    echo "input_player${player}_reserved_device = \"BrowserPad ${player}\""
    echo "input_player${player}_device_reservation_type = \"2\""
  done
  echo "$OVERRIDE_END"
}

write_browserpad_override() {
  local config_file="$1"
  local tmp_file
  local filtered_file

  mkdir -p "$(dirname "$config_file")"
  tmp_file="$(mktemp)"
  filtered_file="$(mktemp)"
  trap 'rm -f "$tmp_file" "$filtered_file"' RETURN

  if [ -f "$config_file" ]; then
    awk -v begin="$OVERRIDE_BEGIN" -v end="$OVERRIDE_END" '
      $0 == begin { skip = 1; next }
      $0 == end { skip = 0; next }
      !skip { print }
    ' "$config_file" > "$filtered_file"
  fi

  {
    if [ -s "$filtered_file" ]; then
      cat "$filtered_file"
      printf '\n'
    fi
    print_browserpad_override
    printf '\n'
  } > "$tmp_file"

  mv "$tmp_file" "$config_file"
}

detect_autoconfig_dir() {
  if [ -d "/opt/retropie/configs/all/retroarch/autoconfig" ]; then
    echo "/opt/retropie/configs/all/retroarch/autoconfig"
    return
  fi

  if [ -d "$HOME/.config/retroarch/autoconfig" ]; then
    echo "$HOME/.config/retroarch/autoconfig"
    return
  fi

  mkdir -p "$HOME/.config/retroarch/autoconfig"
  echo "$HOME/.config/retroarch/autoconfig"
}

detect_retroarch_config() {
  if [ -d "/opt/retropie/configs/all/retroarch/autoconfig" ]; then
    echo "/opt/retropie/configs/all/retroarch.cfg"
    return
  fi

  echo "$HOME/.config/retroarch/retroarch.cfg"
}

if [ "${1:-}" = "--print-browserpad-override" ]; then
  print_browserpad_override
  exit 0
fi

if [ "${1:-}" = "--write-browserpad-override" ]; then
  if [ $# -ne 2 ]; then
    echo "usage: $0 --write-browserpad-override <retroarch.cfg>" >&2
    exit 1
  fi
  write_browserpad_override "$2"
  exit 0
fi

AUTOCONFIG_KIND="created"
if [ -d "/opt/retropie/configs/all/retroarch/autoconfig" ]; then
  AUTOCONFIG_KIND="retropie"
elif [ -d "$HOME/.config/retroarch/autoconfig" ]; then
  AUTOCONFIG_KIND="standalone"
fi

AUTOCONFIG_DIR="$(detect_autoconfig_dir)"
RETROARCH_CFG="$(detect_retroarch_config)"

if [ "$AUTOCONFIG_KIND" = "retropie" ]; then
  echo "Detected: RetroPie install"
elif [ "$AUTOCONFIG_KIND" = "standalone" ]; then
  echo "Detected: standalone RetroArch install"
else
  echo "No existing RetroArch autoconfig found — created $AUTOCONFIG_DIR"
fi

mkdir -p "$APP_DIR"
cp retropie-controller-arm "$APP_DIR/retropie-controller-arm"
chmod +x "$APP_DIR/retropie-controller-arm"

mkdir -p "$AUTOCONFIG_DIR"
for player in $(seq 1 8); do
  sed "s/BrowserPad/BrowserPad ${player}/g" setup/retroarch-autoconfig/BrowserPad.cfg > "$AUTOCONFIG_DIR/BrowserPad ${player}.cfg"
done
echo "Installed BrowserPad autoconfigs to $AUTOCONFIG_DIR"

write_browserpad_override "$RETROARCH_CFG"
echo "Installed BrowserPad RetroArch reservations to $RETROARCH_CFG"

sudo cp setup/$SERVICE_NAME /etc/systemd/system/$SERVICE_NAME
sudo systemctl daemon-reload
sudo systemctl enable --now $SERVICE_NAME

echo ""
echo "Done! retropie-controller is installed and running."
echo ""
echo "Ensure your user is in the input group:"
echo "  sudo usermod -a -G input \$USER"
echo "  (then log out and back in, or reboot)"
