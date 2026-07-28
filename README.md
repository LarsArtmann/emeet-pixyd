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
  <a href="https://emeet-pixyd.lars.software">
    <img alt="Documentation" src="https://img.shields.io/badge/docs-emeet--pixyd.lars.software-8b5cf6?logo=astro">
  </a>
  <br>
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
  <a href="LICENSE">
    <img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-blue.svg">
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

## Who is this for?

- **Linux users with an EMEET PIXY** who want face tracking and privacy to "just work" on every call.
- **Remote workers** tired of manually enabling tracking before meetings and remembering privacy mode after.
- **NixOS users** who want a declarative, reproducible setup via a single module option.
- **Waybar / tiling-WM users** who want live camera status (mode, PTZ, in-call) in their bar.
- **Privacy-conscious users** who want a guaranteed physical lens block whenever no call is active.

## When NOT to use this

Skip this daemon if:

- You don't own an EMEET PIXY (`328f:00c0`) — this is hardware-specific, not a generic webcam tool. Reach for [webcamoid](https://webcamoid.github.io/) instead.
- You're on **macOS or Windows** — this is Linux-only by design (HID hidraw, V4L2, `/proc`, netlink uevents).
- The camera's own tracking toggle is enough for you — if you never forget to enable it, you don't need a daemon.
- You want **cloud or AI features** — this is fully local, no network calls, no telemetry.
- You need **multi-vendor webcam management** — this targets one device family. See [Related Tools](https://emeet-pixyd.lars.software/related-tools/) for general-purpose alternatives.

## Comparison

| Approach                   | Auto call detection | Privacy on end | Audio switching | Hotplug | Linux-native |
| -------------------------- | :-----------------: | :------------: | :-------------: | :-----: | :----------: |
| **emeet-pixyd**            |          ✓          |       ✓        |        ✓        |    ✓    |      ✓       |
| Manual `v4l2-ctl` per call |                     |                |                 |         |      ✓       |
| Vendor Windows/Mac app     |          ✓          |                |        ✓        |         |              |
| webcamoid                  |                     |                |                 |         |      ✓       |

The differentiator is the first column: emeet-pixyd is the only option that detects calls by watching `/proc`, so it works with **any** app that opens the camera — no per-app setup, no browser extension.

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
emeet-pixy track            # Enable face tracking
emeet-pixy idle             # Set camera to idle
emeet-pixy privacy          # Enable privacy mode
emeet-pixy toggle-privacy   # Toggle privacy mode
emeet-pixy center           # Center camera (pan=0, tilt=0, zoom=100)
emeet-pixy audio [mode]     # Cycle or set audio mode (nc, live, org)
emeet-pixy gesture-on       # Enable gesture control
emeet-pixy gesture-off      # Disable gesture control
emeet-pixy toggle-gesture   # Toggle gesture control
emeet-pixy auto [mode]      # Set auto mode (off, full, tracking-only, privacy-only)
emeet-pixy auto-on          # Enable full auto mode
emeet-pixy auto-off         # Disable auto mode
emeet-pixy toggle-auto      # Toggle auto mode
emeet-pixy pan <value>      # Set pan (−150 to 150; or rel+/-N for relative)
emeet-pixy tilt <value>     # Set tilt (−90 to 90; or rel+/-N for relative)
emeet-pixy zoom <value>     # Set zoom (100 to 150; or rel+/-N for relative)
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

- Live MJPEG camera preview (full-width hero)
- Camera mode cards (Track / Idle / Privacy) with SVG icons and keyboard shortcuts (T / I / P / C)
- Audio mode selector (Noise Cancel / Live / Original)
- PTZ sliders (pan ±150°, tilt ±90°, zoom 100–150×) with a spatial position radar
- Snapshot button to capture still frames
- Preset save/load/delete chips
- Gesture control and auto-mode toggles
- Toast notifications for state changes
- Live updates via SSE (Server-Sent Events)

## Configuration

### Environment Variables

All config is via environment variables (no CLI flags — `os.Args` is reserved for socket commands):

| Variable                     | Default            | Description                                                                 |
| ---------------------------- | ------------------ | --------------------------------------------------------------------------- |
| `EMEET_PIXYD_STATE_DIR`      | `/run/emeet-pixyd` | Runtime state directory (socket + state.json)                               |
| `EMEET_PIXYD_WEB_ADDR`       | `127.0.0.1:8090`   | Web UI listen address (localhost only)                                      |
| `EMEET_PIXYD_POLL_INTERVAL`  | `2s`               | Call detection polling interval (Go duration)                               |
| `EMEET_PIXYD_DEBOUNCE_COUNT` | `3`                | Consecutive polls before state change                                       |
| `EMEET_PIXYD_DEBUG`          | `false`            | Enable pprof endpoints at `/debug/pprof/`                                   |
| `EMEET_PIXYD_AUTO`           | `full`             | Auto mode: off, full, tracking-only, privacy-only (legacy: true/1, false/0) |
| `EMEET_PIXYD_DEFAULT_AUDIO`  | `nc`               | Default audio mode: nc, live, org                                           |

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
  ├── Unix socket listener    Command routing (status, track, privacy, PTZ, …)
  ├── HTTP server              Web UI + API + Prometheus /metrics
  ├── Polling ticker (2s)      /proc scanning → call detection + debounce
  ├── Netlink uevent listener  USB hotplug detection
  └── systemd sd_notify        READY=1 + WATCHDOG=1
```

### Source Layout

```
main.go             Entry point, daemon lifecycle, signal handling, status/waybar output
commands.go         Command routing (socket + CLI), PTZ/audio/auto/gesture handlers
handlers.go         HTTP routing, web handlers, toast propagation, PTZ axis lookup table
metrics.go          OTel metrics registration and updates
stream.go           MJPEG streaming, snapshot, JPEG frame extraction, checkDevice guard
middleware.go       Middleware variable assignments (implementations live in http.go)
sse.go              SSE-only: Broadcaster (thread-safe fan-out), per-client SSE, event helpers
http.go             HTTP helpers: writeJSON, middleware chain, security/request/logging middleware
ptz.go              PTZ logic: parsePTZValue, axis dispatch, V4L2 control via v4l2-ctl
hid.go              HID bidirectional communication (config/query over hidraw)
device.go           HID device state management: setDeviceState, setTracking/Audio/Gesture, centerCamera
process.go          /proc scanning for call detection, PipeWire, notifications
uevent.go           Netlink uevent listener for hotplug (context-cancellable)
uevent_linux.go     Low-level unix.Socket for netlink
auto.go             Auto-manage loop, call start/end, debounce logic
state.go            State persistence (JSON, atomic write via tmpfile + rename)
probe.go            Device probing (sysfs walks for video4linux + hidraw)
commander.go        CommandRunner interface (subprocess logging), realCommandRunner, noopCommandRunner
deps.go             Dependencies struct (CommandRunner + DI function pointers)
waybar.go           Waybar integration: waybarJSON struct, tooltip builder
web_types.go        webStatus struct shared between handlers and templates
errors.go           CommandError type, CommandResult, exported sentinel errors
cache.go            Named cache types: lastFrameCache, ptzCache (encapsulated mutex access)
templates.templ     HTML templates (compiled via templ generate)
internal/pixy/      Shared types: Config, State, CameraState, AudioMode, PID, SourceID, PTZ constants
static/             Frontend assets (HTMX, app.js, style.css) — go:embed
```

### Key Design Decisions

- **HID protocol**: Commands are 9-byte config reports followed by a commit report with a 200ms inter-report delay. Responses are 64-byte reads parsed by byte position.
- **State persistence**: JSON file at `{StateDir}/state.json`, atomic write via `.tmp` + rename. Loaded state always wins over defaults.
- **Call detection**: Scans `/proc/*/fd` for processes holding the video device open, excluding self and descendants. Debounced (default 3 cycles × 2s = 6s).
- **Dependency injection**: All external interactions (HID, v4l2, PipeWire, notifications) are function fields on `Daemon`, enabling full test injectability without interfaces.
- **Branded types**: `PID` and `SourceID` use phantom-type branding to prevent mixing process IDs and PipeWire source IDs at compile time.

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

MIT — see [LICENSE](LICENSE).

---

<p align="center">
  <a href="https://emeet-pixyd.lars.software">Full documentation at emeet-pixyd.lars.software</a>
</p>
