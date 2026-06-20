# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.2.0] - 2026-06-07

### Added

- Full auto-management: call detection via `/proc/*/fd` scanning with debounced state transitions
- Three auto modes: `full` (tracking + audio + source + privacy), `tracking-only`, `privacy-only`
- HID bidirectional protocol for camera control (tracking, idle, privacy) and audio mode switching
- V4L2 PTZ control via `v4l2-ctl` subprocess (pan ±150°, tilt ±90°, zoom 100–150×)
- HTMX web UI with dark glassmorphism theme, MJPEG preview, PTZ sliders, toast notifications
- Keyboard shortcuts: T (track), I (idle), P (privacy), C (center)
- Waybar integration with JSON output (icon, class, tooltip with full status)
- Netlink uevent listener for USB hotplug detection and auto re-probe
- State persistence via JSON with atomic write (tmp + rename)
- Prometheus metrics via OTel SDK (`emeet_pixyd_in_call`, `emeet_pixyd_auto_mode`, `emeet_pixyd_camera_state`)
- Desktop notifications via `notify-send` for call start/end events
- PipeWire default source switching via `wpctl`
- Gesture control toggle via HID
- Unix socket control interface (CLI and daemon share same binary)
- NixOS module: `hardware.emeet-pixy` with udev rules, systemd user service, tmpfiles.d
- Nix flake build with `proxyVendor = true` for templ compatibility
- Security middleware: CSP, X-Frame-Options, Referrer-Policy, X-Content-Type-Options, request ID
- Pprof debug endpoints gated behind `EMEET_PIXYD_DEBUG=true`
- systemd `sd_notify` integration (READY=1, WATCHDOG=1)
- SIGHUP for state save without shutdown
- Comprehensive test suite: unit, integration, fuzz, and BDD behavioral tests
- `behavior_test.go`: 14 end-to-end user scenario tests
- `/api/health` endpoint with JSON status, camera state, and version
- `--version` and `--help` CLI flags
- HID circuit breaker: 3 consecutive failures triggers re-probe
- Stream health monitoring (duration histogram + frame counter)
- `device` command shows both video and hidraw paths
- Lint check in `flake.nix` (`nix build .#checks.x86_64-linux.lint`)

### Changed

- Handler extraction: `handlers.go` (624 lines) split into `metrics.go`, `stream.go`, `middleware.go`
- OTel metrics migration from direct `prometheus/client_golang` to OTel SDK
- Metrics encapsulated in `daemonMetrics` struct with DRY `mustFloat64Gauge`/`mustInt64Counter`/`mustFloat64Histogram` helpers
- Error consolidation: exported sentinel errors in `errors.go`
- Auto mode type changed from boolean to `AutoMode` string enum (`off`/`full`/`tracking-only`/`privacy-only`)
- Branded types for PID and SourceID via `go-branded-id`
- PTZ limits moved to shared `internal/pixy` constants (eliminated template split brain)
- `PTZValues.Get/Set` for axis-agnostic PTZ access (eliminated switch statements)
- `PTZValues.Clamp()` domain method for safe range clamping
- Auto-manage only persists state when a state change actually occurs
- State validation on load rejects garbage CameraState/AudioMode/AutoMode values
- JPEG frame extraction guarded against infinite loops on corrupt streams (10M iteration cap)
- Uevent listener retries on transient read errors instead of permanently dying
- PTZ slider hx-trigger fixed (removed `, change` that doubled requests)
- Toast response constants extracted (`respTrackingOn`, `respPrivacyOn`, `respTrackingOff`)
- NixOS systemd hardening: `ProtectSystem=strict`, `PrivateTmp`, `NoNewPrivileges`, `MemoryMax=256M`
- HID byte protocol uses map lookups instead of switch statements
- V4L2 control names centralized in `ptzAxes` map with reverse lookup
- External binary names extracted as constants (`ffmpegBin`, `wpctlBin`, `notifySendBin`, `v4l2ctlBin`)
- Contextual logging via `slog.With` in `device.go` and `auto.go`
- CSS variables for all hover/border/background colors (10 hardcoded values replaced)
- `app.js`: XSS-safe `createElement`/`textContent`, URL validation in doAction, PTZ helpers, named constants
- `setupStream` 4-tuple return replaced with named `streamResult` struct
- `lastFrameCache.Get()` returns defensive copy to prevent data race
- Unused linters and invalid build tags removed from `.golangci.yml`
- Docs archived into `docs/status/archive/` and `docs/planning/archive/`

### Fixed

- Fixed `hid.go` nil error wrapping bug in `hidSendRecv` zero-write path
- Fixed `probe.go` malformed HID_ID handling (`return false` → `continue`)
- Fixed `flake.nix` invalid `env` attribute in app definition
- Fixed `package.nix` version string duplication via `let version` binding
- Fixed error banner `role="alert"` for screen reader accessibility
- Fixed false-positive tests with proper assertions

## [0.1.0] - 2026-01-01

### Added

- Initial release
