#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/home/pi/retropie-controller"
AUTOCONFIG_DIR="/opt/retropie/configs/all/retroarch/autoconfig"
SERVICE_NAME="retropie-controller.service"

mkdir -p "$APP_DIR"
cp retropie-controller-arm "$APP_DIR/retropie-controller-arm"
chmod +x "$APP_DIR/retropie-controller-arm"

mkdir -p "$AUTOCONFIG_DIR"
for player in $(seq 1 8); do
  sed "s/BrowserPad/BrowserPad ${player}/g" setup/retroarch-autoconfig/BrowserPad.cfg > "$AUTOCONFIG_DIR/BrowserPad ${player}.cfg"
done

sudo cp setup/$SERVICE_NAME /etc/systemd/system/$SERVICE_NAME
sudo systemctl daemon-reload
sudo systemctl enable --now $SERVICE_NAME

echo "Installed ${SERVICE_NAME} and BrowserPad RetroArch autoconfigs."
echo "Ensure the pi user is in the input group:"
echo "  sudo usermod -a -G input pi"
