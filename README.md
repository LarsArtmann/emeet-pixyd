# emeet-pixyd

Auto-activation daemon for the [EMEET PIXY](https://www.emeet.com) dual-camera AI webcam (USB vendor `328f:00c0`).

## What It Does

- **Call detection**: Scans `/proc/*/fd` for processes holding the video device open
- **Auto-activate**: Enables face tracking + noise cancellation when a call starts
- **Auto-privacy**: Enters privacy mode (camera physically disabled) when the call ends
- **Audio switching**: Auto-switches PipeWire default source to PIXY on call start
- **Web UI**: Serves an HTMX-based control panel for manual camera control
- **Waybar integration**: Outputs JSON for a custom status bar module

## Commands

Run without arguments to start the daemon, or pass a command to communicate via Unix socket:

```
emeet-pixy status           # Full status (camera, audio, gesture, PTZ, in-call, auto)
emeet-pixy toggle-privacy   # Toggle privacy mode
emeet-pixy track            # Enable face tracking
emeet-pixy idle             # Set camera to idle
emeet-pixy center           # Center camera (pan=0, tilt=0, zoom=100)
emeet-pixy audio [mode]     # Cycle or set audio mode (nc, live, org)
emeet-pixy gesture [on|off] # Toggle gesture control
emeet-pixy auto [on|off]    # Toggle auto-management mode
emeet-pixy sync             # Sync daemon state from camera hardware
emeet-pixy probe            # Re-detect device (video + hidraw)
emeet-pixy waybar           # Output Waybar JSON
emeet-pixy snapshot         # Capture current frame as JPEG
emeet-pixy ptz              # Read pan/tilt/zoom values
```

## Architecture

```
main.go           Entry point, daemon lifecycle, signal handling
commands.go       Command routing (socket + CLI)
handlers.go       HTTP handlers + web UI (HTMX + templ templates)
hid.go            HID bidirectional communication (config/query)
process.go        /proc scanning for call detection
v4l2.go           V4L2 pan/tilt/zoom control
uevent.go         Netlink uevent listener for hotplug
templates.templ   HTML templates (compiled via `templ`)
internal/pixy/    Shared types, config, constants, state
static/           Frontend assets (JS, CSS, htmx)
```

## Configuration

Defaults via `pixy.DefaultConfig()`. Configurable via the `Config` struct:

| Field | Default | Purpose |
|-------|---------|---------|
| `PollInterval` | 2s | Call detection polling |
| `DebounceCount` | 3 | Confirmations before state change |
| `StateDir` | `~/.local/state/emeet-pixyd` | Persistent state location |
| `WebAddr` | `:8090` | Web UI listen address |
| `SocketPath` | `{StateDir}/emeet-pixyd.sock` | Unix domain socket |

## Development

```bash
just build        # Build daemon
just test         # Run tests (unit + integration)
just lint         # Run golangci-lint
```

## Nix

Build with Nix:

```bash
nix build
```

Use as a NixOS module (adds systemd service + udev rules):

```nix
inputs.emeet-pixyd.url = "github:LarsArtmann/emeet-pixyd";

# In nixosModules:
inputs.emeet-pixyd.nixosModules.default

# In nixpkgs overlays:
inputs.emeet-pixyd.overlays.default
```

## Dependencies

- Go 1.26+
- Linux (uses V4L2, hidraw, netlink uevents)
- `github.com/a-h/templ` — HTML template generation
- `github.com/coreos/go-systemd/v22` — systemd sd_notify + watchdog
- `github.com/prometheus/client_golang` — metrics export

## License

MIT
