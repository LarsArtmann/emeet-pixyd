<h1 align="center">
  emeet-pixyd
  <br>
  <sub>Auto-activation daemon for the EMEET PIXY dual-camera AI webcam</sub>
</h1>

<p align="center">
  <strong>Linux-native daemon that makes your PIXY webcam smart:</strong>
  <br>
  face tracking on call start · privacy mode on call end · audio switching · HTMX web UI
</p>

<p align="center">
  <a href="https://github.com/LarsArtmann/emeet-pixyd/actions/workflows/go-test.yml">
    <img alt="Go tests" src="https://github.com/LarsArtmann/emeet-pixyd/actions/workflows/go-test.yml/badge.svg">
  </a>
  <a href="https://github.com/LarsArtmann/emeet-pixyd/actions/workflows/nix.yml">
    <img alt="Nix build" src="https://github.com/LarsArtmann/emeet-pixyd/actions/workflows/nix.yml/badge.svg">
  </a>
  <img alt="Go 1.26+" src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go">
  <img alt="Linux only" src="https://img.shields.io/badge/platform-Linux-FCC624?logo=linux">
  <a href="https://www.emeet.com">
    <img alt="EMEET PIXY" src="https://img.shields.io/badge/device-EMEET_PIXY-328f00?logo=usb">
  </a>
</p>

---

## Why

The EMEET PIXY is a great dual-camera AI webcam — but on Linux, you're stuck manually enabling face tracking before calls, switching audio sources, and remembering to enable privacy mode when you're done. **emeet-pixyd automates all of that.**

It watches `/proc` to detect when a video call starts (Zoom, Teams, Google Meet, anything that opens the camera), then:

1. **Activates** face tracking + noise cancellation
2. **Switches** your PipeWire audio source to the PIXY microphone
3. When the call ends, enters **privacy mode** (lens physically blocked)

No setup per app. No browser extension. Works with anything that opens `/dev/video*`.

## Features

| Feature                | Description                                                                                  |
| ---------------------- | -------------------------------------------------------------------------------------------- |
| **Call detection**     | Scans `/proc/*/fd` for processes holding the video device — works with any app               |
| **Auto-activate**      | Enables face tracking + noise cancellation when a call starts                                |
| **Auto-privacy**       | Physically blocks the lens when the call ends                                                |
| **Audio switching**    | Auto-switches PipeWire default source to PIXY on call start                                  |
| **Web UI**             | Dark-themed HTMX control panel with live MJPEG preview, PTZ sliders, and toast notifications |
| **Waybar integration** | JSON output for a custom Waybar module                                                       |
| **Hotplug**            | Netlink uevent listener detects USB plug/unplug, auto-re-probes                              |
| **Prometheus metrics** | OTel-based metrics at `/metrics` for monitoring                                              |
| **NixOS module**       | Systemd user service, udev rules, tmpfiles.d — one option to enable                          |

## Quick Start

### NixOS (recommended)

Add the flake input and enable the module:

```nix
# flake.nix
inputs.emeet-pixyd.url = "github:LarsArtmann/emeet-pixyd";

# In your nixosConfiguration:
inputs.emeet-pixyd.nixosModules.default
```

Then in your configuration:

```nix
hardware.emeet-pixy = {
  enable = true;
  auto = "full";           # off | full | tracking-only | privacy-only
  defaultAudio = "nc";     # nc | live | org
  debug = false;
};
```

That's it. The module installs the daemon, udev rules, systemd user service, and all runtime dependencies.

### Build from Source

```bash
git clone https://github.com/LarsArtmann/emeet-pixyd.git
cd emeet-pixyd
nix build    # or: go build -o emeet-pixyd .
```

## Usage

Run without arguments to start the daemon, or pass a command to communicate via Unix socket:

```
emeet-pixy status           # Full status (camera, audio, gesture, PTZ, in-call, auto)
emeet-pixy toggle-privacy   # Toggle privacy mode
emeet-pixy track            # Enable face tracking
emeet-pixy idle             # Set camera to idle
emeet-pixy center           # Center camera (pan=0, tilt=0, zoom=100)
emeet-pixy audio [mode]     # Cycle or set audio mode (nc, live, org)
emeet-pixy gesture-on       # Enable gesture control
emeet-pixy gesture-off      # Disable gesture control
emeet-pixy toggle-gesture   # Toggle gesture control
emeet-pixy auto [mode]      # Set auto mode (off, full, tracking-only, privacy-only)
emeet-pixy auto-on          # Enable full auto mode
emeet-pixy auto-off         # Disable auto mode
emeet-pixy toggle-auto      # Toggle auto mode
emeet-pixy pan <value>      # Set pan (-170 to 170)
emeet-pixy tilt <value>     # Set tilt (-30 to 30)
emeet-pixy zoom <value>     # Set zoom (100 to 400)
emeet-pixy sync             # Sync daemon state from camera hardware
emeet-pixy probe            # Re-detect device (video + hidraw)
emeet-pixy device           # Show current video device path
emeet-pixy waybar           # Output Waybar JSON
```

## Auto Modes

The daemon supports four auto-management strategies:

| Mode             | On call start                                               | On call end  |
| ---------------- | ----------------------------------------------------------- | ------------ |
| `full` (default) | Face tracking + noise cancellation + PipeWire source switch | Privacy mode |
| `tracking-only`  | Face tracking                                               | Privacy mode |
| `privacy-only`   | Nothing                                                     | Privacy mode |
| `off`            | Nothing                                                     | Nothing      |

## Web UI

The daemon serves a dark-themed control panel at `http://127.0.0.1:8090` with:

- Live MJPEG camera preview
- Camera state buttons (Track / Idle / Privacy) with keyboard shortcuts (T / I / P / C)
- Audio mode selector (Noise Cancel / Live / Original)
- PTZ sliders (pan ±170°, tilt ±30°, zoom 100–400×)
- Gesture control and auto-mode toggles
- Toast notifications for state changes
- Auto-refresh every 3 seconds via HTMX

## Configuration

### Environment Variables

All config is via environment variables (no CLI flags — `os.Args` is reserved for socket commands):

| Variable                     | Default            | Description                                       |
| ---------------------------- | ------------------ | ------------------------------------------------- |
| `EMEET_PIXYD_STATE_DIR`      | `/run/emeet-pixyd` | Runtime state directory (socket + state.json)     |
| `EMEET_PIXYD_WEB_ADDR`       | `127.0.0.1:8090`   | Web UI listen address (localhost only)            |
| `EMEET_PIXYD_POLL_INTERVAL`  | `2s`               | Call detection polling interval (Go duration)     |
| `EMEET_PIXYD_DEBOUNCE_COUNT` | `3`                | Consecutive polls before state change             |
| `EMEET_PIXYD_DEBUG`          | `false`            | Enable pprof endpoints at `/debug/pprof/`         |
| `EMEET_PIXYD_AUTO`           | `full`             | Auto mode: off, full, tracking-only, privacy-only |
| `EMEET_PIXYD_DEFAULT_AUDIO`  | `nc`               | Default audio mode: nc, live, org                 |

### NixOS Module Options

| Option                             | Type   | Default  | Description                             |
| ---------------------------------- | ------ | -------- | --------------------------------------- |
| `hardware.emeet-pixy.enable`       | bool   | `false`  | Enable the daemon                       |
| `hardware.emeet-pixy.user`         | string | `"lars"` | User owning the runtime state directory |
| `hardware.emeet-pixy.auto`         | enum   | `"full"` | Auto-management strategy                |
| `hardware.emeet-pixy.defaultAudio` | enum   | `"nc"`   | Default audio mode                      |
| `hardware.emeet-pixy.debug`        | bool   | `false`  | Enable debug mode                       |

## Runtime Dependencies

The daemon integrates with these Linux tools (all provided by the NixOS module):

| Tool          | Purpose                           | Required            |
| ------------- | --------------------------------- | ------------------- |
| `v4l2-ctl`    | Pan/tilt/zoom control via V4L2    | Yes                 |
| `ffmpeg`      | MJPEG streaming in web UI         | For preview only    |
| `wpctl`       | PipeWire default source switching | For audio switching |
| `notify-send` | Desktop notifications             | Optional            |

## Architecture

```
main() → NewDaemon() → Run()
  ├── Unix socket listener    Command routing (status, track, privacy, …)
  ├── HTTP server              Web UI + API + Prometheus /metrics
  ├── Polling ticker (2s)      /proc scanning for call detection
  ├── Netlink uevent listener  USB hotplug detection
  └── systemd sd_notify        READY=1 + WATCHDOG=1
```

```
main.go           Entry point, daemon lifecycle, signal handling
commands.go       Command routing (socket + CLI)
handlers.go       HTTP handlers, web UI
metrics.go        OTel metrics registration and updates
stream.go         MJPEG streaming, snapshot, JPEG frame extraction
middleware.go      Security headers, request ID, caching, PTZ validation
hid.go            HID bidirectional communication (config/query)
v4l2.go           V4L2 pan/tilt/zoom via v4l2-ctl
process.go        /proc scanning for call detection, PipeWire, notifications
uevent.go         Netlink uevent listener for hotplug
auto.go           Auto-manage loop, call start/end, debounce
state.go          State persistence (JSON, atomic write)
probe.go          Device probing (sysfs walks for video4linux + hidraw)
errors.go         CommandError type, sentinel errors
web_types.go      webStatus struct for web UI
templates.templ   HTML templates (compiled via templ)
internal/pixy/    Shared types: Config, State, CameraState, AudioMode, constants
static/           Frontend assets (HTMX, app.js, style.css) — go:embed
```

## Troubleshooting

| Problem                           | Solution                                                                                     |
| --------------------------------- | -------------------------------------------------------------------------------------------- |
| `PIXY not connected`              | Check USB connection, run `lsusb \| grep 328f`. Ensure udev rules are loaded.                |
| `Permission denied` on hidraw     | Verify udev rules match vendor `328f` product `00c0`. Run `udevadm control --reload-rules`.  |
| `v4l2-ctl: command not found`     | Install `v4l-utils` (provided by NixOS module).                                              |
| No audio switching                | Check `wpctl status` shows a PIXY source. `wpctl` must be in PATH.                           |
| Camera not detected after plug-in | Daemon auto-detects via netlink uevents. Check `emeet-pixy probe` output.                    |
| Web UI shows "Camera in use"      | Another process (e.g., OBS, another browser tab) is holding `/dev/video*` open.              |
| Debounce too slow                 | Reduce `EMEET_PIXYD_DEBOUNCE_COUNT` (default 3) or `EMEET_PIXYD_POLL_INTERVAL` (default 2s). |

## Development

```bash
nix develop             # Enter dev shell (go, golangci-lint, templ)
nix build               # Build the daemon
templ generate           # Regenerate HTML templates after editing .templ

# Test (use GOWORK=off — parent go.work doesn't include this project)
GOWORK=off go test -race -count=1 ./...

# Lint
GOWORK=off golangci-lint run --timeout 2m ./...
```

## License

Proprietary — see [LICENSE](LICENSE).
