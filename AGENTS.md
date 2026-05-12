# emeet-pixyd

Auto-activation daemon for the EMEET PIXY dual-camera AI webcam (USB `328f:00c0`). Linux-only, x86_64.

**Updated:** 2026-05-12

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

| Environment Variable         | Config Field    | Default            |
| ---------------------------- | --------------- | ------------------ | ---------------------------------------------------------------- |
| `EMEET_PIXYD_STATE_DIR`      | `StateDir`      | `/run/emeet-pixyd` |
| `EMEET_PIXYD_WEB_ADDR`       | `WebAddr`       | `127.0.0.1:8090`   |
| `EMEET_PIXYD_POLL_INTERVAL`  | `PollInterval`  | `2s`               |
| `EMEET_PIXYD_DEBOUNCE_COUNT` | `DebounceCount` | `3`                |
| `EMEET_PIXYD_DEBUG`          | `Debug`         | `false`            |
| `EMEET_PIXYD_AUTO`           | `AutoMode`      | `full`             | off, full, tracking-only, privacy-only (legacy: true/1, false/0) |
| `EMEET_PIXYD_DEFAULT_AUDIO`  | `DefaultAudio`  | `nc`               | nc, live, org (shorthand for original)                           |

NixOS module passes all options as `Environment=` vars — `auto` (enum: off/full/tracking-only/privacy-only) maps directly to `EMEET_PIXYD_AUTO`, `defaultAudio` maps directly, `debug` sets `EMEET_PIXYD_DEBUG`. `NewDaemon()` applies `Config.AutoMode` and `Config.DefaultAudio` to initial state before `loadState()` (persisted state wins). Why env vars, not CLI flags: `os.Args` is used for socket commands (`emeet-pixyd status`), so flag parsing would conflict.

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

| File               | Purpose                                                                                                                                       |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.go`          | `Daemon` struct, lifecycle, signal handling, status/waybar output, socket server                                                              |
| `commands.go`      | Command routing for both Unix socket and CLI (`handleCommand` switch)                                                                         |
| `handlers.go`      | HTTP routing, web handlers, web UI                                                                                                            |
| `metrics.go`       | OTel metrics registration, `updateMetrics()`, `init()`                                                                                        |
| `stream.go`        | MJPEG streaming, snapshot, JPEG frame extraction                                                                                              |
| `middleware.go`    | Security headers, request ID, caching FS, PTZ axis validation                                                                                 |
| `hid.go`           | HID bidirectional communication over hidraw — config writes + response parsing                                                                |
| `v4l2.go`          | V4L2 pan/tilt/zoom control via `v4l2-ctl` subprocess                                                                                          |
| `process.go`       | `/proc/*/fd` scanning for call detection, PipeWire source switching, desktop notifications                                                    |
| `uevent.go`        | Netlink uevent listener for device hotplug (context-cancellable, fd closed on shutdown)                                                       |
| `uevent_linux.go`  | Low-level `unix.Socket` call for netlink                                                                                                      |
| `auto.go`          | Auto-manage loop, call start/end handling, debounce logic                                                                                     |
| `state.go`         | State persistence (JSON load/save, atomic write)                                                                                              |
| `probe.go`         | Device probing (sysfs walks for video4linux + hidraw)                                                                                         |
| `web_types.go`     | `webStatus` struct shared between handlers and templates                                                                                      |
| `templates.templ`  | HTML templates (compiled via `templ generate`)                                                                                                |
| `errors.go`        | `CommandError` type, exported sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`)                                                   |
| `internal/pixy/`   | Shared types: `Config`, `State`, `CameraState`, `AudioMode`, `PID`, `SourceID`, constants, `SendCommand`                                      |
| `static/`          | Frontend assets (HTMX, app.js, style.css) — embedded via `//go:embed`                                                                         |
| `behavior_test.go` | BDD-style behavioral tests: full auto lifecycle, debounce flip-flop, PTZ clamping, waybar tooltip, privacy toggle, audio cycle, state restart |
| `commands_test.go` | Unit tests for PTZ, auto, gesture, audio, tracking commands, actionToast, applyResponseToStatus                                               |
| `main_test.go`     | Core test helpers (`newTestDaemon`, `testDaemonNoDevice`), state persistence, HID parsing, device probing, config tests                       |
| `handlers_test.go` | Handler tests, `requireGaugeValue`, `matchAttrs` metric assertions, JPEG frame extraction, middleware tests                                   |
| `uevent_test.go`   | Uevent parsing tests                                                                                                                          |
| `auto_test.go`     | Tests for `handleCallStart`, `handleCallEnd`, `autoManage` state transitions, debounce, metrics                                               |
| `process_test.go`  | Tests for `ppidOf`, `isDescendantOf`, `isCameraInUse` using real `/proc`                                                                      |

### Key Interactions

- **HID protocol**: Commands are 9-byte config reports followed by a commit report, with a 200ms sleep between them. Responses are 64-byte reads parsed by byte position.
- **State persistence**: JSON file at `{StateDir}/state.json`, atomic write via `.tmp` + rename. State dir defaults to `/run/emeet-pixyd`.
- **Call detection**: Scans `/proc/*/fd` for processes holding the video device open, excluding self and descendants. Debounced (default 3 cycles).
- **Device probing**: Walks `/sys/class/video4linux` and `/sys/class/hidraw` matching vendor `328f` product `00c0`. Device name matching uses shared `isPixyName()` helper in `probe.go`.
- **Uevent listener**: `listenUevents(ctx, ch)` is context-cancellable. A goroutine closes the netlink fd on context cancellation to unblock `fd.Read()` on shutdown.

### External Dependencies at Runtime

- `v4l2-ctl` — PTZ control (must be in PATH)
- `ffmpeg` — MJPEG streaming in web UI
- `wpctl` — PipeWire default source switching
- `notify-send` — desktop notifications

### Dependency Injection

Daemon uses function fields for external dependencies, enabling test injectability:

| Field             | Default            | Purpose                                                  |
| ----------------- | ------------------ | -------------------------------------------------------- |
| `isCameraInUseFn` | `isCameraInUse`    | `/proc/*/fd` scanning                                    |
| `findSourceFn`    | `findPixySource`   | `wpctl status` PipeWire lookup (returns `pixy.SourceID`) |
| `setSourceFn`     | `setDefaultSource` | `wpctl set-default` (takes `pixy.SourceID`)              |
| `notifyFn`        | `notify`           | `notify-send` desktop notifs                             |
| `setTrackingFn`   | `d.setTracking`    | Camera state changes via HID                             |
| `setAudioFn`      | `d.setAudio`       | Audio mode changes via HID                               |
| `setGestureFn`    | `d.setGesture`     | Gesture toggle via HID                                   |
| `centerCameraFn`  | `d.centerCamera`   | PTZ centering via v4l2-ctl                               |
| `v4l2SetFn`       | `v4l2Set`          | Arbitrary V4L2 control setting                           |

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
- `PID` and `SourceID` are branded types via `github.com/larsartmann/go-branded-id` (phantom typing) — prevent mixing process IDs and PipeWire source IDs at compile time. Defined in `internal/pixy/ids.go`.
- Metrics use OpenTelemetry SDK (`go.opentelemetry.io/otel/exporters/prometheus`) with `promhttp.Handler()` for the `/metrics` endpoint. `prometheus/client_golang` kept only for `promhttp`.

### Testing

- Standard `testing` package only (no testify)
- **`newTestDaemon(camera, videoDev, hidrawDev, opts...)`** is the canonical builder — use `withInCall()` or custom `testDaemonOption` to inject mock deps
- Function fields (`isCameraInUseFn`, `findSourceFn`, `setSourceFn`, `notifyFn`, `setTrackingFn`, `setAudioFn`, `setGestureFn`, `centerCameraFn`, `v4l2SetFn`) default to no-op stubs or real implementations in tests
- Predefined test options: `withInCall()`, `withAutoOff()`, `withCameraInUse()`, `withNotifyCalled()`, `withNotifyMessages()`, `withFindSource()`, `withCaptureTracking()`, `withCaptureAudio()`, `withCaptureGesture()`, `withCaptureGestureArg()`, `withCaptureCenter()`, `withNoopV4L2()`, `withDebounceCount()`
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
- **`webStatus` struct lives in `web_types.go`**, not in `templates.templ`. Fields `Camera`, `Audio`, `Auto` use typed `pixy.CameraState`/`pixy.AudioMode`/`pixy.AutoMode` for compile-time safety. Templates use typed comparisons (`s.Camera == pixy.StateTracking`) instead of raw string literals.
- **Generated file**: `templates.templ` must be compiled with `templ generate` before `go build`. The nix build runs `templ generate` automatically in `preBuild`. The generated `_templ.go` file is gitignored. Use `go generate ./...` to rebuild locally. Templates import `pixy` for typed constants (`pixy.StateTracking`, `pixy.AudioNC`, etc.).
- **Template `load` trigger was removed**: The `#status-panel` div uses `hx-trigger="every 3s, refresh from:body"` (no `load`). The `load` trigger caused an infinite loop with `outerHTML` swap because each swapped element re-triggered `load`. Initial page renders the panel server-side via `@statusPanel(s)` in `page()`.
- **Scripts at end of body**: HTMX and `app.js` are loaded at the end of `<body>` (not in `<head>`) because `app.js` accesses `document.body` at line 12 which would be `null` in `<head>`.
- **Build tags**: All `.go` files in the root use `//go:build linux`. Tests that test Linux-specific code naturally require a Linux host.
- **`flaky` test awareness**: Some tests probe real sysfs (e.g., `TestProbeDevices_SetsStateToOfflineWhenNoVideo`), so they may pass or fail depending on whether a PIXY is physically connected. These tests handle both outcomes gracefully.
- **Branded ID types**: `pixy.PID` (branded `int`) and `pixy.SourceID` (branded `string`) in `internal/pixy/ids.go` use `github.com/larsartmann/go-branded-id` for phantom type branding. `ppidOf`/`isDescendantOf` use `pixy.PID`; `findPixySource`/`setDefaultSource` use `pixy.SourceID`. Zero value is `pixy.PID{}`/`pixy.SourceID{}` (check with `.IsZero()`).
- **Nix build uses `proxyVendor = true`**: The standard FOD-based vendor approach fails because `templ generate` (in `preBuild`) produces imports that weren't visible when the FOD ran `go mod vendor`. `proxyVendor = true` downloads deps during the build via Go module proxy, after `templ generate` has run. The `vendorHash` is a SHA of the downloaded module tarballs, not of the vendored source tree.
- **WebAddr default**: `127.0.0.1:8090` (localhost only, not `:8090`)
- **StateDir default**: `/run/emeet-pixyd` (tmpfiles.d rule in NixOS module creates this)
- **Binary symlink**: `package.nix` creates `emeet-pixy` symlink pointing to `emeet-pixyd` for CLI usage
- **Gosec exclusions are intentional**: `.golangci.yml` excludes G304 (file inclusion), G204 (subprocess launch), G706 (log injection), G115 (integer overflow) because this hardware daemon inherently opens `/dev/hidraw*`, `/dev/video*`, and launches `ffmpeg`/`v4l2-ctl`/`wpctl`. These are not fixable — suppressing in config is cleaner than per-site `//nolint` comments.
- **`linters.enable` blocks `issues.exclude-rules`** in golangci-lint v2.11.4. Use `linters.disable` + `issues.exclude-rules` together; the former enables all other linters while the latter can suppress specific issues.
- **Lint is clean (0 issues)**: Linters that produced only false positives (`contextcheck`, `exhaustruct`, `gochecknoinits`, `gochecknoglobals`, `paralleltest`) have been removed from `.golangci.yml` `linters.enable`. `contextcheck` flagged templ-generated `ServeHTTP` calls and `updateMetrics`; `exhaustruct` flagged intentional partial struct initializations throughout; `gochecknoinits` flagged the required OTel `init()` in `handlers.go`; `gochecknoglobals` flagged the OTel metric variables; `paralleltest` flagged `TestUpdateMetrics` which must be serial (global metrics state). Go 1.22+ eliminates the need for `tc := tc` loop variable capture (`copyloopvar` handles it). Gosec excludes cover hardware-daemon patterns (G104, G107, G115, G204, G301, G304, G306, G702, G706).
- **`init()` is in `metrics.go`**: The OTel metrics `init()` was moved from `handlers.go` to `metrics.go` during handler extraction. `gochecknoinits` flags it there.
- **Error consolidation**: Exported sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`) live in `errors.go`. The unexported duplicates in `commands.go` were removed. All code references the exported versions.
- **pprof gated behind `Debug` config**: Pprof endpoints (`/debug/pprof/*`) are only registered when `Config.Debug` is `true`. Default is `false`. The NixOS module exposes `hardware.emeet-pixy.debug` option.
- **`t.Parallel()` in all tests**: All test functions in `integration_test.go` and subtests call `t.Parallel()`. No `tc := tc` captures needed (Go 1.22+ loop variable semantics).
- **Test coverage for auto/process**: `auto_test.go` tests `handleCallStart`, `handleCallEnd`, `autoManage` state transitions. `process_test.go` tests `ppidOf`, `isDescendantOf`, `isCameraInUse` using real `/proc` filesystem.
- **`webStatus` uses typed fields**: `Camera pixy.CameraState`, `Audio pixy.AudioMode`, `Auto pixy.AutoMode`. Templates use typed comparisons (`s.Camera == pixy.StateTracking`) instead of raw string literals, eliminating the previous split brain.
- **Unified toast/error pattern**: All HTTP handlers use `applyResponseToStatus()` to set either `status.Error` (on command error) or `status.Toast`+`status.ToastType` (on success). The toast type (`success`, `info`, `error`) is propagated from `actionToast()` through `applyResponseToStatus()` — not hardcoded.
- **Handler extraction**: `handlers.go` (was 624 lines) extracted into `metrics.go` (OTel registration + updateMetrics), `stream.go` (MJPEG streaming + JPEG parsing), `middleware.go` (security headers, request ID, caching, PTZ validation). Handlers.go is now ~310 lines focused on HTTP routing and web handlers.
- **Command constants**: All command strings are named constants in `commands.go`: `cmdTrack`, `cmdIdle`, `cmdPrivacy`, `cmdTogglePrivacy`, `cmdCenter`, `cmdSync`, `cmdProbe`, `cmdAudio`, `cmdAuto`, `cmdAutoOn`, `cmdAutoOff`, `cmdToggleAuto`, `cmdGestureOn`, `cmdGestureOff`, `cmdToggleGesture`, `cmdStatus`, `cmdWaybar`, `cmdDevice`. No raw string literals for command names anywhere in the codebase.
- **`waybarJSON` struct**: Waybar output uses a typed struct with json tags (`waybarJSON`) and `strings.Builder` for tooltip construction. Optimized from 23 allocs/860ns to 7 allocs/334ns.
- **`TestUpdateMetrics` is NOT parallel**: Tests global mutable metrics state, must run serially to avoid interference from parallel tests calling `updateMetrics()`. `TestSendCommand_EndToEnd` and `TestConfigFromEnv_DefaultsWhenUnset` both use `t.Parallel()` (temp socket dir and env reads respectively are safe to parallelize).
- **Auto-manage uses DI functions**: `handleCallStart` and `handleCallEnd` call `d.setTrackingFn()` and `d.setAudioFn()` (not the direct methods). This ensures the auto-manage path is mockable in tests.
- **Toast type propagation**: `applyResponseToStatus()` accepts a `toastType` parameter. `actionToast()` returns both message and type (`success`/`info`). All callers propagate the type correctly.
- **Response string constants**: `respTrackingOn`, `respPrivacyOn`, `respTrackingOff`, `respAutoModeOff`, `respAudioUsage`, `respAutoUsage`, `respDeviceNotFound` in `commands.go`. All command handlers return these constants instead of inline strings.
- **`TestHandleCommandTogglePrivacy` tests mock behavior**: The test uses `withCaptureTrackingSlice` to verify the DI function was called with the correct argument, and asserts the exact response string (`respTrackingOn`). It does NOT assert `d.state.Camera` because the mock bypasses `setDeviceState` (which is the only path that mutates daemon state via the `stateSetter` callback).
- **PTZ shared constants**: `pixy.PanMin`, `pixy.PanMax`, `pixy.TiltMin`, `pixy.TiltMax`, `pixy.ZoomMin`, `pixy.ZoomMax`, `pixy.ZoomDefault` in `internal/pixy/pixy.go`. Used by `handlers.go` and `templates.templ`.
- **State validation**: `loadState()` calls `loaded.Valid()` to reject garbage enum values. `pixy.State.Valid()` checks Camera, Audio, and Auto fields.
- **Stream constants**: `streamBufSize` and `ffmpegShutdownTimeout` live in `stream.go`, not `handlers.go`.
- **Toast constants**: `toastAudioChanged`, `toastGestureToggled`, `toastAutoToggled` in `handlers.go`.
- **NixOS systemd hardening**: `ProtectSystem=strict`, `PrivateTmp=true`, `NoNewPrivileges=true`, `RestrictAddressFamilies=[AF_UNIX AF_NETLINK AF_INET]`, `MemoryMax=256M`.
- **JPEG max-iterations**: `extractJPEGFrame` has a 10M iteration guard to prevent infinite loops on corrupt streams.
- **Uevent retry**: Transient read errors use `continue` instead of `return` to keep the hotplug listener alive.
- **Auto-manage conditional save**: `autoManage` only calls `saveStateOrLog` when `handleCallStart` or `handleCallEnd` actually triggered a state change.
- **Benchmarks**: 4 established — `BenchmarkExtractJPEGFrame`, `BenchmarkFormatLastSynced`, `BenchmarkParseHIDResponse`, `BenchmarkWaybarOutput`.

---

## NixOS Module

`modules/nixos.nix` provides:

- `hardware.emeet-pixy.enable` option
- udev rules for PIXY hidraw + video4linux access
- systemd user service (runs after pipewire + graphical-session)
- Installs `v4l-utils`, `wireplumber`, `libnotify` in service PATH
- Creates `/run/emeet-pixyd` tmpfiles.d entry
