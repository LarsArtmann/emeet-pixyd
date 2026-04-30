# emeet-pixyd

Auto-activation daemon for the EMEET PIXY dual-camera AI webcam (USB `328f:00c0`). Linux-only, x86_64.

**Updated:** 2026-04-30

---

## Commands

```bash
# Build
nix build                          # production build (preferred)
go build -o emeet-pixyd .          # manual build (needs `templ generate` first for template changes)

# Test
go test -race -count=1 ./...       # CI runs this exact command
go test ./...                       # without race detector (faster)
go test -run TestName ./...         # single test

# Lint
golangci-lint run                   # uses .golangci.yml config

# Generate templ templates
templ generate                      # required after editing templates.templ

# Format (Nix)
nix fmt                            # alejandra for .nix files

# Run daemon
nix run                            # or ./emeet-pixyd
emeet-pixyd status                  # send command via unix socket
```

### CI

GitHub Actions: `go vet ./...` then `go test -race -count=1 ./...` on ubuntu-latest.

---

## Architecture

Linux-only daemon (`//go:build linux` on all source files). Single binary, no subcommands — running with arguments sends a command to an already-running daemon via Unix socket; running without arguments starts the daemon.

### Control Flow

```
main() → NewDaemon() → Run()
  ├── Unix socket listener (commands.go routing)
  ├── HTTP server (handlers.go → HTMX web UI)
  ├── Polling ticker (2s) → autoManage() → /proc scanning for call detection
  ├── Netlink uevent listener (hotplug detection)
  └── systemd sd_notify (READY=1, WATCHDOG=1)
```

### File Responsibilities

| File | Purpose |
|---|---|
| `main.go` | `Daemon` struct, lifecycle, device probing, state persistence, call management, auto-manage loop |
| `commands.go` | Command routing for both Unix socket and CLI (`handleCommand` switch) |
| `handlers.go` | HTTP handlers, web UI, Prometheus metrics, MJPEG streaming, security middleware |
| `hid.go` | HID bidirectional communication over hidraw — config writes + response parsing |
| `v4l2.go` | V4L2 pan/tilt/zoom control via `v4l2-ctl` subprocess |
| `process.go` | `/proc/*/fd` scanning for call detection, PipeWire source switching, desktop notifications |
| `uevent.go` | Netlink uevent listener for device hotplug |
| `uevent_linux.go` | Low-level `unix.Socket` call for netlink |
| `templates.templ` | HTML templates (compiled via `templ generate`) — defines `webStatus` struct and all UI |
| `internal/pixy/` | Shared types: `Config`, `State`, `CameraState`, `AudioMode`, constants, `SendCommand` |
| `static/` | Frontend assets (HTMX, app.js, style.css) — embedded via `//go:embed` |

### Key Interactions

- **HID protocol**: Commands are 9-byte config reports followed by a commit report, with a 200ms sleep between them. Responses are 64-byte reads parsed by byte position.
- **State persistence**: JSON file at `{StateDir}/state.json`, atomic write via `.tmp` + rename. State dir defaults to `/run/emeet-pixyd`.
- **Call detection**: Scans `/proc/*/fd` for processes holding the video device open, excluding self and descendants. Debounced (default 3 cycles).
- **Device probing**: Walks `/sys/class/video4linux` and `/sys/class/hidraw` matching vendor `328f` product `00c0`.

### External Dependencies at Runtime

- `v4l2-ctl` — PTZ control (must be in PATH)
- `ffmpeg` — MJPEG streaming in web UI
- `wpctl` — PipeWire default source switching
- `notify-send` — desktop notifications

---

## Code Patterns

### Concurrency Model

- `Daemon.mu` (`sync.RWMutex`) — protects `state`, `videoDev`, `hidrawDev`, debounce counters
- `Daemon.cmdMu` (`sync.Mutex`) — serializes commands (prevents concurrent HID writes)
- `Daemon.streamSema` (chan, cap 1) — limits to one MJPEG stream
- `Daemon.lastFrame` — has its own `sync.RWMutex`
- `Daemon.ptzCache` — has its own `sync.RWMutex`, 2-second TTL

All lock acquisitions follow a consistent pattern: acquire, copy values, release, then act on copies.

### Error Handling

- Errors are returned as `"error: ..."` strings from commands (both socket and HTTP)
- `fmt.Errorf("label: %w", err)` for wrapping
- HID send failures trigger `probeDevices()` re-scan
- State save failures are logged but non-fatal

### Type Design

- `CameraState` and `AudioMode` are string-based types with `Valid()` and `String()` methods
- `ParseAudioMode("org")` maps to `AudioOriginal` (value `"original"`) — the CLI shorthand differs from the stored value
- Generic `queryHIDState[T]` in `hid.go` for type-safe HID queries

### Testing

- Standard `testing` package only (no testify)
- Tests construct `Daemon` structs directly (no DI framework)
- Fake sysfs trees for device probing tests (`createFakeVideo4linux`, `createFakeHidraw`)
- `t.Parallel()` used consistently
- Fuzz tests exist: `handlers_fuzz_test.go`, `hid_fuzz_test.go`
- Integration test: `integration_test.go` (tests CLI ↔ daemon via socket)

---

## Gotchas

- **Audio mode shorthand**: CLI accepts `"org"` but the stored/displayed value is `"original"`. `ParseAudioMode` maps both.
- **PTZ units**: V4L2 uses 1/3600-degree units internally (`v4l2DegreesPerUnit = 3600`). The daemon presents user-facing degrees but multiplies before sending to `v4l2-ctl`. Zoom is not multiplied.
- **Generated file**: `templates.templ` must be compiled with `templ generate` before `go build`. The generated `_templ.go` file is gitignored.
- **Build tags**: All `.go` files in the root use `//go:build linux`. Tests that test Linux-specific code naturally require a Linux host.
- **`flaky` test awareness**: Some tests probe real sysfs (e.g., `TestProbeDevices_SetsStateToOfflineWhenNoVideo`), so they may pass or fail depending on whether a PIXY is physically connected. These tests handle both outcomes gracefully.
- **Nix filter excludes tests**: The `flake.nix` `srcFilter` excludes `*_test.go` and `*_fuzz_test.go` from the build source.
- **WebAddr default**: `127.0.0.1:8090` (localhost only, not `:8090`)
- **StateDir default**: `/run/emeet-pixyd` (tmpfiles.d rule in NixOS module creates this)
- **Binary symlink**: `package.nix` creates `emeet-pixy` symlink pointing to `emeet-pixyd` for CLI usage

---

## NixOS Module

`modules/nixos.nix` provides:
- `hardware.emeet-pixy.enable` option
- udev rules for PIXY hidraw + video4linux access
- systemd user service (runs after pipewire + graphical-session)
- Installs `v4l-utils`, `wireplumber`, `libnotify` in service PATH
- Creates `/run/emeet-pixyd` tmpfiles.d entry
