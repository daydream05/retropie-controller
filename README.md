# retropie-controller

Turn any phone browser into a game controller — no app install needed. Scan a QR code, open Safari (or any browser), and your phone becomes a gamepad. Supports up to 8 simultaneous players.

Works with **RetroPie** and **standalone RetroArch** on any Linux machine (Raspberry Pi, desktop, mini PC, etc.).

## How It Works

1. Run the binary on your Linux machine
2. It prints a QR code in the terminal and generates `/tmp/controller-qr.png`
3. Players scan the QR → browser opens → instant controller UI
4. Each phone gets assigned a player slot (P1–P8)
5. Inputs flow over WebSocket → Linux virtual gamepad (uinput) → RetroArch reads it like a real controller

## Features

- Single Go binary — no Node.js, no npm, no runtime dependencies
- Up to 8 simultaneous players
- HMAC-signed join tokens baked into the QR (secure, no guessable codes)
- LAN-only: subnet check enforces same-network requirement
- Full-screen controller UI: D-pad, A/B/X/Y, LB/RB, Start/Select
- iOS Safari compatible (Pointer Events, no scroll/zoom, pagehide cleanup)
- Heartbeat timeout: stuck buttons auto-release if phone disconnects
- macOS build support for local development (uinput stubbed out)

## Requirements

- Linux with `/dev/uinput` (Raspberry Pi, Ubuntu, Debian, etc.)
- RetroArch — RetroPie or standalone
- Go 1.22+ (only for building)
- Your user in the `input` group (see below)

## Build

```bash
# Build for current machine
make build

# Cross-compile for Raspberry Pi (32-bit ARMv7)
make build-pi
# → produces retropie-controller-arm

# Run locally (macOS/Linux dev)
make run
```

## Install

### RetroPie

```bash
make build-pi
# Copy repo to Pi, then on the Pi:
chmod +x setup/install.sh
./setup/install.sh
```

The installer auto-detects RetroPie and puts autoconfig files in:
`/opt/retropie/configs/all/retroarch/autoconfig/`

### Standalone RetroArch (any Linux)

Same steps — the installer auto-detects standalone RetroArch and puts autoconfig files in:
`~/.config/retroarch/autoconfig/`

### Input group (required for uinput)

```bash
sudo usermod -a -G input $USER
# Log out and back in, or reboot
```

### Systemd service (auto-start on boot)

The installer sets this up automatically. To manage manually:

```bash
sudo systemctl start retropie-controller
sudo systemctl stop retropie-controller
sudo systemctl status retropie-controller
```

## Usage

1. Start the service (or run `./retropie-controller-arm` manually)
2. It logs the controller URL and prints a QR code to the terminal
3. Scan the QR with your phone — Safari opens the controller page
4. Multiple players scan the same QR — each gets their own slot
5. Start a game in RetroArch — BrowserPad controllers are auto-detected

**Port:** binds to `80` by default, falls back to `3000` if not root. The QR URL reflects whichever port is active.

## Multiplayer

- Each phone that scans = one player slot (P1, P2, ... up to P8)
- Assignment is first-come, first-served
- Disconnecting frees the slot immediately
- All 8 virtual controllers are created at startup — RetroArch sees them as always-present gamepads

## Controller Layout

```
[LB]                        [RB]
         [ Player N ]

   [↑]                  [Y]
[←]   [→]           [X]   [B]
   [↓]                  [A]

        [Select] [Start]
```

## Security

- Join tokens are HMAC-SHA256 signed, generated fresh at each startup
- Tokens expire after 24 hours
- WebSocket joins rejected if client is not on the same `/16` subnet
- Rate limited: max 10 join attempts per IP per minute

## Troubleshooting

**`permission denied` on `/dev/uinput`**
- Make sure your user is in the `input` group and you've logged out/in

**Phone loads the page but can't join**
- Confirm phone and Pi are on the same WiFi network
- Check the URL in the QR matches the Pi's actual LAN IP
- If port 80 is unavailable, the fallback URL (`:3000`) is shown in logs

**RetroArch doesn't recognize the controllers**
- Confirm autoconfig files exist: `ls ~/.config/retroarch/autoconfig/BrowserPad*`
- Restart RetroArch after install
- Check devices exist: `cat /proc/bus/input/devices | grep BrowserPad`

**Works in RetroPie menus but not in games**
- Some cores need controllers configured per-core in RetroArch settings
- Go to: Settings → Input → Port 1 Controls → Set to BrowserPad 1

## Project Layout

```
retropie-controller/
├── main.go
├── go.mod / go.sum
├── internal/
│   ├── gamepad/        # uinput virtual gamepad (Linux) + stub (macOS)
│   ├── server/         # HTTP + WebSocket handler
│   └── token/          # HMAC token gen/validation
├── public/
│   └── controller.html # Phone controller UI
├── setup/
│   ├── install.sh                    # Auto-detects RetroPie vs standalone
│   ├── retropie-controller.service   # systemd unit
│   └── retroarch-autoconfig/
│       └── BrowserPad.cfg            # RetroArch button mapping template
└── Makefile
```
