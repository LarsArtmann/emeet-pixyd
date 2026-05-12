# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Full auto-management: call detection via `/proc/*/fd` scanning with debounced state transitions
- Three auto modes: `full` (tracking + audio + source + privacy), `tracking-only`, `privacy-only`
- HID bidirectional protocol for camera control (tracking, idle, privacy) and audio mode switching
- V4L2 PTZ control via `v4l2-ctl` subprocess (pan ±170°, tilt ±30°, zoom 100–400×)
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

### Changed

- Handler extraction: `handlers.go` (624 lines) split into `metrics.go`, `stream.go`, `middleware.go`
- OTel metrics migration from direct `prometheus/client_golang` to OTel SDK
- Error consolidation: exported sentinel errors in `errors.go`
- Auto mode type changed from boolean to `AutoMode` string enum (`off`/`full`/`tracking-only`/`privacy-only`)
- Branded types for PID and SourceID via `go-branded-id`
- Unified `audioCommand` to `cmdAudio` constant
- Fixed `hid.go` nil error wrapping bug in `hidSendRecv` zero-write path
- Fixed `probe.go` malformed HID_ID handling (`return false` → `continue`)
- Fixed `flake.nix` invalid `env` attribute in app definition
- Fixed `package.nix` version string duplication via `let version` binding

## [0.1.0] - 2026-01-01

### Added

- Initial release
