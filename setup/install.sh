#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/home/pi/retropie-controller"
SERVICE_NAME="retropie-controller.service"

# Detect RetroArch autoconfig directory
if [ -d "/opt/retropie/configs/all/retroarch/autoconfig" ]; then
  AUTOCONFIG_DIR="/opt/retropie/configs/all/retroarch/autoconfig"
  echo "Detected: RetroPie install"
elif [ -d "$HOME/.config/retroarch/autoconfig" ]; then
  AUTOCONFIG_DIR="$HOME/.config/retroarch/autoconfig"
  echo "Detected: standalone RetroArch install"
else
  # Create standalone RetroArch path as fallback
  AUTOCONFIG_DIR="$HOME/.config/retroarch/autoconfig"
  mkdir -p "$AUTOCONFIG_DIR"
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

sudo cp setup/$SERVICE_NAME /etc/systemd/system/$SERVICE_NAME
sudo systemctl daemon-reload
sudo systemctl enable --now $SERVICE_NAME

echo ""
echo "Done! retropie-controller is installed and running."
echo ""
echo "Ensure your user is in the input group:"
echo "  sudo usermod -a -G input \$USER"
echo "  (then log out and back in, or reboot)"
