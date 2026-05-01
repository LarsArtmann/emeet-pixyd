# emeet-pixyd

Auto-activation daemon for the EMEET PIXY dual-camera AI webcam (USB `328f:00c0`). Linux-only, x86_64.

**Updated:** 2026-05-01

---

## Commands

```bash
# Build
nix build                          # production build (preferred)
go build -o emeet-pixyd .          # manual build (needs `templ generate` first for template changes)

# Test (IMPORTANT: use GOWORK=off if a parent go.work exists)
GOWORK=off go test -race -count=1 ./...  # CI runs this
GOWORK=off go test ./...                 # without race detector
GOWORK=off go test -run TestName ./...   # single test

# Lint (IMPORTANT: use GOWORK=off)
GOWORK=off golangci-lint run --timeout 2m ./...  # CI runs this
GOWORK=off golangci-lint run ./...              # without timeout
GOWORK=off golangci-lint run handlers.go        # single file

# Generate templ templates
templ generate                      # required after editing templates.templ

# Format (Nix)
nix fmt                            # alejandra for .nix files

# Run daemon
nix run                            # or ./emeet-pixyd
emeet-pixyd status                  # send command via unix socket
```

### CI

GitHub Actions: `go vet ./...`, `golangci-lint run --timeout 2m`, then `go test -race -count=1 ./...` on ubuntu-latest. All steps use `GOWORK: off`.

---

## Architecture

Linux-only daemon (`//go:build linux` on all source files). Single binary, no subcommands — running with arguments sends a command to an already-running daemon via Unix socket; running without arguments starts the daemon.

### Configuration

Daemon reads environment variables via `pixy.ConfigFromEnv()`, falling back to defaults for unset/invalid values:

| Environment Variable        | Config Field    | Default              |
| --------------------------- | --------------- | -------------------- |
| `EMEET_PIXYD_STATE_DIR`    | `StateDir`      | `/run/emeet-pixyd`   |
| `EMEET_PIXYD_WEB_ADDR`     | `WebAddr`       | `127.0.0.1:8090`     |
| `EMEET_PIXYD_POLL_INTERVAL`| `PollInterval`  | `2s`                 |
| `EMEET_PIXYD_DEBOUNCE_COUNT`| `DebounceCount`| `3`                  |
| `EMEET_PIXYD_DEBUG`        | `Debug`         | `false`              |

NixOS module passes `EMEET_PIXYD_DEBUG=true` when `hardware.emeet-pixy.debug` is enabled. Why env vars, not CLI flags: `os.Args` is used for socket commands (`emeet-pixyd status`), so flag parsing would conflict.

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

| File              | Purpose                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------ |
| `main.go`         | `Daemon` struct, lifecycle, signal handling, status/waybar output, socket server           |
| `commands.go`     | Command routing for both Unix socket and CLI (`handleCommand` switch)                      |
| `handlers.go`     | HTTP handlers, web UI, OTel/Prometheus metrics, MJPEG streaming, security middleware       |
| `hid.go`          | HID bidirectional communication over hidraw — config writes + response parsing             |
| `v4l2.go`         | V4L2 pan/tilt/zoom control via `v4l2-ctl` subprocess                                       |
| `process.go`      | `/proc/*/fd` scanning for call detection, PipeWire source switching, desktop notifications |
| `uevent.go`       | Netlink uevent listener for device hotplug                                                 |
| `uevent_linux.go` | Low-level `unix.Socket` call for netlink                                                   |
| `auto.go`         | Auto-manage loop, call start/end handling, debounce logic                                  |
| `state.go`        | State persistence (JSON load/save, atomic write)                                           |
| `probe.go`        | Device probing (sysfs walks for video4linux + hidraw)                                      |
| `web_types.go`    | `webStatus` struct shared between handlers and templates                                   |
| `templates.templ` | HTML templates (compiled via `templ generate`)                                             |
| `errors.go`       | `CommandError` type, exported sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`) |
| `internal/pixy/`  | Shared types: `Config`, `State`, `CameraState`, `AudioMode`, constants, `SendCommand`      |
| `static/`         | Frontend assets (HTMX, app.js, style.css) — embedded via `//go:embed`                      |
| `auto_test.go`       | Tests for `handleCallStart`, `handleCallEnd`, `autoManage` state transitions                 |
| `process_test.go`    | Tests for `ppidOf`, `isDescendantOf`, `isCameraInUse`                                       |

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

### Dependency Injection

Daemon uses function fields for external dependencies, enabling test injectability:

| Field              | Default              | Purpose                          |
| ------------------ | -------------------- | -------------------------------- |
| `isCameraInUseFn`  | `isCameraInUse`      | `/proc/*/fd` scanning            |
| `findSourceFn`     | `findPixySource`     | `wpctl status` PipeWire lookup   |
| `setSourceFn`      | `setDefaultSource`   | `wpctl set-default`              |
| `notifyFn`         | `notify`             | `notify-send` desktop notifs     |

`NewDaemon()` wires real implementations. Tests override via functional options (`testDaemonOption`).

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
- Metrics use OpenTelemetry SDK (`go.opentelemetry.io/otel/exporters/prometheus`) with `promhttp.Handler()` for the `/metrics` endpoint. `prometheus/client_golang` kept only for `promhttp`.

### Testing

- Standard `testing` package only (no testify)
- **`newTestDaemon(camera, videoDev, hidrawDev, opts...)`** is the canonical builder — use `withAudio()`, `withInCall()`, or custom `testDaemonOption` to inject mock deps
- Function fields (`isCameraInUseFn`, `findSourceFn`, `setSourceFn`, `notifyFn`) default to no-op stubs in tests
- `testDaemonNoDevice()` and `testDaemonWithDevice(camera)` are convenience wrappers
- `ptr[T any](v T) *T` generic helper for pointer literals (not Go's `new()` literal syntax)
- `sendSC(t, socketPath, cmd)` consolidates `pixy.SendCommand` + error handling in tests
- `assertPtrEqual[T]` generic helper for optional field comparison in tests
- `requireGaugeValue(t, name, want, attrs...)` asserts OTel gauge values via `promExporter.Collect()` + `metricdata.Gauge[float64]` matching
- `matchAttrs(set, wanted)` checks attribute set contains wanted key-values using `attribute.Set.Value()`
- Fake sysfs trees for device probing tests (`createFakeVideo4linux`, `createFakeHidraw`)
- `newTestWebServer` returns `*httptest.Server` only (unparam fix)
- `t.Parallel()` used consistently
- Fuzz tests: `handlers_fuzz_test.go`, `hid_fuzz_test.go`
- Integration tests: `integration_test.go` (web server + socket command tests)

---

## Gotchas

- **GOWORK=off required**: Parent directory has a `go.work` that doesn't include this project. Always use `GOWORK=off` for `go build`/`go test`. CI is configured with `GOWORK: off` env var.
- **Audio mode shorthand**: CLI accepts `"org"` but the stored/displayed value is `"original"`. `ParseAudioMode` maps both.
- **PTZ units**: V4L2 uses 1/3600-degree units internally (`v4l2DegreesPerUnit = 3600`). The daemon presents user-facing degrees but multiplies before sending to `v4l2-ctl`. Zoom is not multiplied.
- **`webStatus` struct lives in `web_types.go`**, not in `templates.templ` (was moved out for cleaner separation).
- **Generated file**: `templates.templ` must be compiled with `templ generate` before `go build`. The generated `_templ.go` file is gitignored.
- **Build tags**: All `.go` files in the root use `//go:build linux`. Tests that test Linux-specific code naturally require a Linux host.
- **`flaky` test awareness**: Some tests probe real sysfs (e.g., `TestProbeDevices_SetsStateToOfflineWhenNoVideo`), so they may pass or fail depending on whether a PIXY is physically connected. These tests handle both outcomes gracefully.
- **Nix filter excludes tests**: The `flake.nix` `srcFilter` excludes `*_test.go` and `*_fuzz_test.go` from the build source.
- **WebAddr default**: `127.0.0.1:8090` (localhost only, not `:8090`)
- **StateDir default**: `/run/emeet-pixyd` (tmpfiles.d rule in NixOS module creates this)
- **Binary symlink**: `package.nix` creates `emeet-pixy` symlink pointing to `emeet-pixyd` for CLI usage
- **Gosec exclusions are intentional**: `.golangci.yml` excludes G304 (file inclusion), G204 (subprocess launch), G706 (log injection), G115 (integer overflow) because this hardware daemon inherently opens `/dev/hidraw*`, `/dev/video*`, and launches `ffmpeg`/`v4l2-ctl`/`wpctl`. These are not fixable — suppressing in config is cleaner than per-site `//nolint` comments.
- **`linters.enable` blocks `issues.exclude-rules`** in golangci-lint v2.11.4. Use `linters.disable` + `issues.exclude-rules` together; the former enables all other linters while the latter can suppress specific issues.
- **Lint is clean (0 issues)**: Linters that produced only false positives (`exhaustruct`, `paralleltest`, `contextcheck`, `gochecknoglobals`) have been removed from `.golangci.yml` `linters.enable`. Go 1.22+ eliminates the need for `tc := tc` loop variable capture (`copyloopvar` handles it). Gosec excludes cover hardware-daemon patterns (G104, G107, G115, G204, G301, G304, G306, G702, G706). Code-level fixes include extracted helpers (`ffmpegStreamCmd`, `cleanupFFmpeg`, `newHIDResponse`, `sendSC`).
- **OTel metrics migration**: Replaced `prometheus/client_golang/prometheus` direct usage with `go.opentelemetry.io/otel/exporters/prometheus`. Metrics (`metricInCall`, `metricAutoMode`, `metricCameraState`) are now `metric.Float64Gauge` instruments created via OTel MeterProvider. `promhttp.Handler()` still serves the `/metrics` endpoint. Test assertions use `promExporter.Collect()` with `metricdata.Gauge[float64]` instead of `testutil.ToFloat64()`.
- **Error consolidation**: Exported sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`) live in `errors.go`. The unexported duplicates in `commands.go` were removed. All code references the exported versions.
- **pprof gated behind `Debug` config**: Pprof endpoints (`/debug/pprof/*`) are only registered when `Config.Debug` is `true`. Default is `false`. The NixOS module exposes `hardware.emeet-pixy.debug` option.
- **`t.Parallel()` in all tests**: All test functions in `integration_test.go` and subtests call `t.Parallel()`. No `tc := tc` captures needed (Go 1.22+ loop variable semantics).
- **Test coverage for auto/process**: `auto_test.go` tests `handleCallStart`, `handleCallEnd`, `autoManage` state transitions. `process_test.go` tests `ppidOf`, `isDescendantOf`, `isCameraInUse` using real `/proc` filesystem.
- **`gochecknoinits` removed from linters**: Valid `init()` in `handlers.go` for `registerMetrics()` — must run before any parallel test can call `updateMetrics()`
- **`TestUpdateMetrics` is NOT parallel**: Tests global mutable metrics state, must run serially to avoid interference from parallel tests calling `updateMetrics()`
- **errcheck `exclude-rules` don't work**: golangci-lint v2.11.4's `issues.exclude-rules` doesn't suppress errcheck issues. Use `//nolint:errcheck` inline instead (see integration_test.go pattern).
- **errcheck in test cleanup**: `resp.Body.Close()` and `os.RemoveAll()` in test code use `//nolint:errcheck` — these errors are harmless and intentionally ignored.

---

## NixOS Module

`modules/nixos.nix` provides:

- `hardware.emeet-pixy.enable` option
- udev rules for PIXY hidraw + video4linux access
- systemd user service (runs after pipewire + graphical-session)
- Installs `v4l-utils`, `wireplumber`, `libnotify` in service PATH
- Creates `/run/emeet-pixyd` tmpfiles.d entry
