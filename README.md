# retropie-controller

`retropie-controller` turns a phone browser into a RetroPie controller. A Raspberry Pi running RetroPie boots the Go binary, prints a QR code, and exposes up to 8 simultaneous browser-backed gamepads as Linux virtual input devices.

## Features

- Single Go binary with no Node.js or npm dependency
- HTTP controller page plus WebSocket input channel
- HMAC-signed join token
- LAN subnet check and in-memory join rate limiting
- 8 pre-created virtual gamepads named `BrowserPad 1` through `BrowserPad 8`
- QR code output both as `/tmp/controller-qr.png` and terminal ASCII
- Graceful cleanup on disconnect, heartbeat timeout, and shutdown
- macOS-safe build path with non-Linux gamepad stubs

## Requirements

- Raspberry Pi running RetroPie on Linux
- `/dev/uinput` available and writable
- Go 1.22+ for local development
- `pi` user in the `input` group:

```bash
sudo usermod -a -G input pi
```

## Project Layout

```text
retropie-controller/
├── go.mod
├── go.sum
├── main.go
├── internal/
│   ├── gamepad/
│   │   ├── manager.go
│   │   ├── manager_linux.go
│   │   └── manager_stub.go
│   ├── server/
│   │   ├── server.go
│   │   └── server_test.go
│   └── token/
│       ├── token.go
│       └── token_test.go
├── public/
│   └── controller.html
├── setup/
│   ├── install.sh
│   ├── retropie-controller.service
│   └── retroarch-autoconfig/
│       └── BrowserPad.cfg
└── Makefile
```

## Build

Build for the current machine:

```bash
make build
```

Cross-compile for Raspberry Pi (32-bit ARMv7):

```bash
make build-pi
```

That produces `retropie-controller-arm` using:

```bash
GOOS=linux GOARCH=arm GOARM=7 go build -o retropie-controller-arm .
```

Run locally:

```bash
make run
```

## Install on RetroPie

1. Build the Pi binary with `make build-pi`.
2. Copy the repository to `/home/pi/retropie-controller` on the Pi.
3. Run the installer:

```bash
chmod +x setup/install.sh
./setup/install.sh
```

4. Reboot or restart RetroArch if autoconfig files were not picked up immediately.

The installer:

- copies `retropie-controller-arm` into `/home/pi/retropie-controller`
- installs the systemd service
- generates RetroArch autoconfig files for `BrowserPad 1` through `BrowserPad 8`

## Usage

1. Start the service or run the binary manually.
2. The binary detects a LAN IP and starts HTTP on port `80`, falling back to `3000` when it cannot bind privileged port `80`.
3. Scan the QR code printed in the terminal or open the generated `/tmp/controller-qr.png`.
4. Each phone joins the same session URL and is assigned the next free player slot.

The browser UI uses full-state input messages, 100ms heartbeats, and automatically releases all buttons on disconnect or missed heartbeats.

## Multiplayer

- Up to 8 players can connect at once
- Player slots are assigned in connection order
- Disconnects immediately free the slot for the next phone
- All 8 virtual controllers are created when the process starts

## Security Notes

- Session join tokens are HMAC-SHA256 signed and generated at process start
- WebSocket joins are rejected if the client is not on the same `/16` subnet as the Pi
- Join attempts are rate-limited to 10 per IP per minute

## Troubleshooting

`permission denied` opening `/dev/uinput`:

- confirm the `input` group membership for `pi`
- confirm `/dev/uinput` exists and the udev permissions allow group write access

Phones can load the page but never join:

- confirm they are on the same LAN as the Pi
- confirm the Pi’s chosen IP is reachable from the phone
- if port `80` is unavailable, use the fallback `:3000` URL shown in the logs

RetroArch does not recognize the controllers:

- confirm the generated files exist under `/opt/retropie/configs/all/retroarch/autoconfig`
- restart RetroArch after installing the configs
- inspect `evtest` or `/proc/bus/input/devices` to verify `BrowserPad N` devices were created

## Development Notes

- Linux uses `github.com/bendahl/uinput` for virtual gamepads
- WebSockets are handled with `github.com/gorilla/websocket`
- QR code generation uses `github.com/skip2/go-qrcode`
- Non-Linux builds use a no-op stub so the project still builds cleanly on macOS for the HTTP, token, and UI layers
