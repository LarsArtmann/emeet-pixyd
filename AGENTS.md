# emeet-pixyd

Auto-activation daemon for the EMEET PIXY dual-camera AI webcam (USB `328f:00c0`). Linux-only, x86_64.
---

## Commands

```bash
# Build
nix build                          # production build (preferred)
go build -o emeet-pixyd .          # manual build (needs `templ generate` first for template changes)

# Test (IMPORTANT: use GOWORK=off if a parent go.work exists; GOEXPERIMENT=jsonv2 for encoding/json/v2)
GOEXPERIMENT=jsonv2 GOWORK=off go test -race -count=1 ./...  # CI runs this
GOEXPERIMENT=jsonv2 GOWORK=off go test ./...                 # without race detector
GOEXPERIMENT=jsonv2 GOWORK=off go test -run TestName ./...   # single test

# Lint (IMPORTANT: use GOWORK=off)
GOEXPERIMENT=jsonv2 GOWORK=off golangci-lint run --timeout 2m ./...  # CI runs this
GOEXPERIMENT=jsonv2 GOWORK=off golangci-lint run ./...              # without timeout
GOEXPERIMENT=jsonv2 GOWORK=off golangci-lint run handlers.go        # single file

# Generate templ templates
templ generate                      # required after editing templates.templ

# Format (Nix)
nix fmt                            # alejandra for .nix files

# Run daemon
nix run                            # or ./emeet-pixyd
emeet-pixyd status                  # send command via unix socket
emeet-pixyd --help                  # show CLI usage
```

### CI

GitHub Actions (`go-test.yml`): `go vet`, `templ generate`, `golangci-lint run --timeout 2m`, `govulncheck`, then `go test -race -count=1 -coverprofile=coverage.out`, `nix flake check`, and fuzz targets (`FuzzExtractJPEGFrame`, `FuzzParseHIDResponse`, `FuzzParsePTZValue`) on ubuntu-latest. All Go steps use `GOWORK: off` and `GOEXPERIMENT: jsonv2`. Generated `_templ.go` files are gitignored — CI runs `templ generate` before lint/test. All dependencies are on the public Go module proxy — no `GOPRIVATE` needed.

---

## Architecture

Linux-only daemon (`//go:build linux` on all source files). Single binary, no subcommands — running with arguments sends a command to an already-running daemon via Unix socket; running without arguments starts the daemon.

### Configuration

Daemon reads environment variables via `pixy.ConfigFromEnv()`, falling back to defaults for unset/invalid values:

| Environment Variable | Config Field | Default |
| ---------------------------- | --------------- | ------------------ | ---------------------------------------------------------------- |
| `EMEET_PIXYD_STATE_DIR` | `StateDir` | `/run/emeet-pixyd` |
| `EMEET_PIXYD_WEB_ADDR` | `WebAddr` | `127.0.0.1:8090` |
| `EMEET_PIXYD_POLL_INTERVAL` | `PollInterval` | `2s` |
| `EMEET_PIXYD_DEBOUNCE_COUNT` | `DebounceCount` | `3` |
| `EMEET_PIXYD_DEBUG` | `Debug` | `false` |
| `EMEET_PIXYD_AUTO` | `AutoMode` | `full` | off, full, tracking-only, privacy-only (legacy: true/1, false/0) |
| `EMEET_PIXYD_DEFAULT_AUDIO` | `DefaultAudio` | `nc` | nc, live, org (shorthand for original) |

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

| File               | Purpose                                                                                                                                                                                                                                                                                                                                                                                                                         |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.go`          | `Daemon` struct, lifecycle (`Run` → `startHTTPServer` + `eventLoop` + `handleShutdown`), signal handling, `main()` entry point (335 lines)                                                                                                                                                                                                                                                                                      |
| `commands.go`      | Command routing for both Unix socket and CLI (`handleCommand` switch), extracted `handleQueryCommand` and `handleTogglePrivacy`                                                                                                                                                                                                                                                                                                 |
| `handlers.go`      | HTTP routing, web handlers, HTMX response rendering                                                                                                                                                                                                                                                                                                                                                                             |
| `ptz.go`           | PTZ logic: `ptzAxes` map (stores `pixy.Range`), `handlePTZCommand`, `parsePTZValue`, `ptzAxisValid`, V4L2 control name constants (extracted from handlers.go + v4l2.go)                                                                                                                                                                                                                                                         |
| `metrics.go`       | `daemonMetrics` struct encapsulating all metric vars, DRY `mustFloat64Gauge`/`mustInt64Counter`/`mustFloat64Histogram` helpers, `recordProbe`/`recordStreamDuration`/`recordFrame`/`recordUevent` functions, `updateMetrics`, `recordCommandMetric` (all with nil-safe guards)                                                                                                                                                  |
| `stream.go`        | MJPEG streaming, snapshot, JPEG frame extraction, `checkDevice` guard                                                                                                                                                                                                                                                                                                                                                           |
| `middleware.go`    | Assigns middleware variables (`securityHeaderMiddleware`, `requestIDMW`) — all middleware implementations live in `http.go`                                                                                                                                                                                                                                                                                                     |
| `sse.go`           | SSE-only: `Broadcaster` (thread-safe fan-out), `sseStream` (per-client SSE), `writeSSEEvent`/`splitSSELines` (~190 lines)                                                                                                                                                                                                                                                                                                       |
| `http.go`          | HTTP helpers: `writeJSON`, `chain` (middleware chaining), `statusRecorder` (status capture + Flusher/Pusher), `securityHeadersMiddleware`, `requestIDMiddleware`, `loggingMiddlewareFactory` (~115 lines)                                                                                                                                                                                                                       |
| `hid.go`           | HID bidirectional communication over hidraw — config writes + response parsing                                                                                                                                                                                                                                                                                                                                                  |
| `device.go`        | HID device state management: `setDeviceState`, `setTracking`/`Audio`/`Gesture`, `centerCamera`, query methods, `syncState`, `getStatus`                                                                                                                                                                                                                                                                                         |
| `process.go`       | `/proc/*/fd` scanning for call detection, PipeWire source switching, desktop notifications                                                                                                                                                                                                                                                                                                                                      |
| `uevent.go`        | Netlink uevent listener for device hotplug (context-cancellable, fd closed on shutdown)                                                                                                                                                                                                                                                                                                                                         |
| `uevent_linux.go`  | Low-level `unix.Socket` call for netlink                                                                                                                                                                                                                                                                                                                                                                                        |
| `auto.go`          | Auto-manage loop, call start/end handling, debounce logic                                                                                                                                                                                                                                                                                                                                                                       |
| `state.go`         | State persistence (JSON load/save, atomic write)                                                                                                                                                                                                                                                                                                                                                                                |
| `probe.go`         | Device probing — pure `probeDevices()` returns `probeResult`, `applyProbeResult()` applies to Daemon                                                                                                                                                                                                                                                                                                                            |
| `web_types.go`     | `webStatus` struct shared between handlers and templates                                                                                                                                                                                                                                                                                                                                                                        |
| `templates.templ`  | HTML templates: `page()` (full-width preview hero + header), `statusPanel()` (mode cards + controls grid + footer), `cameraModeCard()` (icon + name + desc), `audioSegment()` (segmented control), `ptzRadar()` (spatial position indicator), `ptzSlider()` (self-contained HTMX response), `shortcutLegend()` (keyboard help), inline SVG icon templates (`iconLens`, `iconTarget`, `iconStandby`, `iconEyeOff`, `iconCamera`) |
| `socket.go`        | Unix socket server: listener setup, `SendCommand` client, connection handling                                                                                                                                                                                                                                                                                                                                                   |
| `deps.go`          | `Dependencies` struct (with `CommandRunner` interface + 10+ DI function pointers), `noopDependencies()` factory                                                                                                                                                                                                                                                                                                                 |
| `commander.go`     | `CommandRunner` interface, `realCommandRunner` (with subprocess logging), `noopCommandRunner` — abstracts all exec.Command calls                                                                                                                                                                                                                                                                                                |
| `waybar.go`        | Waybar integration: `waybarJSON` struct, `waybarCameraStates` map, tooltip builder                                                                                                                                                                                                                                                                                                                                              |
| `errors.go`        | `CommandError` type, `CommandResult`, `errStr` helper, exported sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`)                                                                                                                                                                                                                                                                                                   |
| `errorfamily.go`   | Sentinel classification registration (`registerErrorFamilies` via `sync.Once`) — maps all daemon sentinels to go-error-family Families + stdlib defaults. Called from `NewDaemon()` and `newTestDaemon()`.                                                                                                                                                                                                                      |
| `cache.go`         | Named cache types: `lastFrameCache` (Get/Set), `ptzCache` (Get/Set/Invalidate) with encapsulated mutex access                                                                                                                                                                                                                                                                                                                   |
| `internal/pixy/`   | Shared types: `Config`, `State`, `CameraState`, `AudioMode`, `PID`, `SourceID`, constants, `SendCommand`                                                                                                                                                                                                                                                                                                                        |
| `website/`         | Astro + Starlight documentation website (landing page + 16 docs pages), deployed to `emeet-pixyd.lars.software` via Firebase Hosting                                                                                                                                                                                                                                                                                            |
| `static/`          | Frontend assets (`app.js`, `style.css`, `htmx.js`) — embedded via `//go:embed`. HTMX v2.0.9 served as static file at `/static/htmx.js` (was previously via `cqrshtmx.HTMXScriptHandler()`)                                                                                                                                                                                                                                      |
| `behavior_test.go` | BDD-style behavioral tests: full auto lifecycle, debounce flip-flop, PTZ clamping, waybar tooltip, privacy toggle, audio cycle, state restart                                                                                                                                                                                                                                                                                   |
| `commands_test.go` | Unit tests for PTZ, auto, gesture, audio, tracking commands, actionToast, applyResponseToStatus                                                                                                                                                                                                                                                                                                                                 |
| `main_test.go`     | Core test helpers (`newTestDaemon`, `testDaemonNoDevice`), state persistence, HID parsing, device probing, config tests                                                                                                                                                                                                                                                                                                         |
| `handlers_test.go` | Handler tests, `requireGaugeValue`, `matchAttrs` metric assertions, JPEG frame extraction, middleware tests                                                                                                                                                                                                                                                                                                                     |
| `metrics_test.go`  | Test-only `collectMetrics` helper for OTel metric verification                                                                                                                                                                                                                                                                                                                                                                  |
| `uevent_test.go`   | Uevent parsing tests                                                                                                                                                                                                                                                                                                                                                                                                            |
| `auto_test.go`     | Tests for `handleCallStart`, `handleCallEnd`, `autoManage` state transitions, debounce, metrics                                                                                                                                                                                                                                                                                                                                 |
| `process_test.go`  | Tests for `ppidOf`, `isDescendantOf`, `isCameraInUse` using real `/proc`                                                                                                                                                                                                                                                                                                                                                        |

### Key Interactions

- **HID protocol**: Commands are 9-byte config reports followed by a commit report, with a 200ms sleep between them. Responses are 64-byte reads parsed by byte position.
- **State persistence**: JSON file at `{StateDir}/state.json`, atomic write via `.tmp` + rename. State dir defaults to `/run/emeet-pixyd`. State carries a `SchemaVersion` field (`"v"` in JSON) — `loadState` logs a warning when the on-disk version differs from `pixy.CurrentSchemaVersion`, but still loads the data (best-effort backward compatibility). Old state files missing `"v"` load as version 0.
- **Call detection**: Scans `/proc/*/fd` for processes holding the video device open, excluding self and descendants. Debounced (default 3 cycles).
- **Device probing**: Walks `/sys/class/video4linux` and `/sys/class/hidraw` matching vendor `328f` product `00c0`. Device name matching uses shared `isPixyName()` helper in `probe.go`.
- **Uevent listener**: `UeventListener` interface (`uevent.go`) abstracts netlink. Production impl `netlinkUeventListener`; test impl `noopUeventListener`. Context-cancellable; a goroutine closes the netlink fd on cancellation.
- **Camera presets**: PTZ positions can be saved/recalled by name (CLI: `preset save home`, web: `/api/preset/save/{name}`). Max `pixy.MaxPresets` (16) presets, persisted in `state.json` under `Presets` (`pixy.PresetMap`). Names are validated by `pixy.ValidatePresetName` (non-empty, ≤32 runes, no path separators, no control chars) at both CLI and HTTP save boundaries. `PresetMap.SortedNames()` centralizes the sort-iterate pattern.
- **PTZ readback**: After each PTZ set, `schedulePTZReadback(ctx, videoDev)` runs a delayed (500ms) hardware readback via `context.WithoutCancel` to correct the cache with the actual motor position.
- **Integration tests**: `integration_hardware_test.go` (`//go:build integration`) tests real HID/V4L2 paths. Run with: `go test -tags=integration ./...`.
- **Fake device harness**: `fake_device_test.go` provides `fakeHIDDevice`, `fakeProcInspector`, `fakeUeventListener`, and `withFakeDevices()` option builder for hardware-free testing.

### External Dependencies at Runtime

- `v4l2-ctl` — PTZ control (must be in PATH)
- `ffmpeg` — MJPEG streaming in web UI
- `wpctl` — PipeWire default source switching
- `notify-send` — desktop notifications

### Dependency Injection

Daemon uses function fields and interfaces for external dependencies, enabling test injectability:

| Field            | Default                   | Purpose                                                   |
| ---------------- | ------------------------- | --------------------------------------------------------- |
| `commander`      | `realCommandRunner`       | Subprocess execution (`v4l2-ctl`, `wpctl`, `notify-send`) |
| `procInspector`  | `procInspector{}`         | `/proc` traversal (`ProcessInspector` interface)          |
| `ueventListener` | `netlinkUeventListener{}` | Netlink uevents (`UeventListener` interface)              |
| `isCameraInUse`  | `isCameraInUse`           | `/proc/*/fd` scanning                                     |
| `findSource`     | `d.findPixySource`        | `wpctl status` PipeWire lookup (returns `pixy.SourceID`)  |
| `setSource`      | `d.setDefaultSource`      | `wpctl set-default` (takes `pixy.SourceID`)               |
| `notify`         | `d.notifyCmd`             | `notify-send` desktop notifs                              |
| `setTracking`    | `d.setTracking`           | Camera state changes via HID                              |
| `setAudio`       | `d.setAudio`              | Audio mode changes via HID                                |
| `setGesture`     | `d.setGesture`            | Gesture toggle via HID                                    |
| `centerCamera`   | `d.centerCamera`          | PTZ centering via v4l2-ctl                                |
| `v4l2Set`        | `d.v4l2Set`               | Arbitrary V4L2 control setting                            |
| `parsePTZ`       | `d.parsePTZValues`        | V4L2 PTZ readback parsing                                 |

`NewDaemon()` wires real implementations. Tests override via functional options (`testDaemonOption`).

---

## Code Patterns

### Concurrency Model

- `Daemon.mu` (`sync.RWMutex`) — protects `state`, `videoDev`, `hidrawDev`, debounce counters
- `Daemon.cmdMu` (`sync.Mutex`) — **REPLACED** by `hidMu` + `v4l2Mu` (see below). Was serializing ALL mutating commands; now split for concurrency.
- `Daemon.hidMu` (`sync.Mutex`) — serializes HID device access (tracking, audio, gesture). The 200ms HID protocol sleep no longer blocks V4L2 commands.
- `Daemon.v4l2Mu` (`sync.Mutex`) — serializes V4L2 subprocess access (PTZ, center, preset save/load). v4l2-ctl subprocess no longer blocks HID commands.
- State-only commands (auto mode, preset delete/list) use no I/O lock — only `d.mu`.
- `Daemon.streamSema` (chan, cap 1) — limits to one MJPEG stream
- `Daemon.lastFrame` — has its own `sync.RWMutex`
- `Daemon.ptzCache` — has its own `sync.RWMutex`, 2-second TTL
- `Daemon.broadcaster` (`*Broadcaster`) — local thread-safe SSE fan-out with its own internal `sync.RWMutex`; non-blocking sends drop events for slow clients. No daemon-level lock needed for broadcasts.

All lock acquisitions follow a consistent pattern: acquire, copy values, release, then act on copies.

### Error Handling

- Errors are returned as `"error: ..."` strings from commands (both socket and HTTP)
- `fmt.Errorf("label: %w", err)` for wrapping
- HID send failures trigger `probeDevices()` re-scan
- State save failures are logged but non-fatal
- **go-error-family adoption**: `errorfamily.go` registers 18 daemon sentinels + stdlib defaults via `registerErrorFamilies()` (`sync.Once`, called from `NewDaemon()` + `newTestDaemon()`). Three Families: **Infrastructure** (device/HID/stream errors → HTTP 503, exit 69), **Rejection** (config validation + invalid input → HTTP 400, exit 1), **Transient** (HID timeouts → exit 75). Two adoption patterns: (1) **Sentinel registration** via `errorfamily.RegisterClassification()` for existing `errors.New()` sentinels (preserves `fmt.Errorf("%w")` wrapping without touching call sites); (2) **Constructor-born errors** via `errorfamily.NewInfrastructure()` for new errors (stream.go's 7 stream errors). Library APIs used: `HTTPStatus(err)`, `ExitCode(err)`, `LogError(err, slog.Default())`. Scoped: HTMX handlers intentionally return 200+HTML toast (not classified JSON); hardcoded `http.StatusBadRequest` guard clauses (missing axis/value) stay as-is since they have no error object to classify.

### Type Design

- `CameraState` and `AudioMode` are string-based types with `Valid()` and `String()` methods
- `ParseAudioMode("org")` maps to `AudioOriginal` (value `"original"`) — the CLI shorthand differs from the stored value
- Generic `queryHIDState[T]` in `hid.go` for type-safe HID queries
- `PTZValues.Get(axis)` and `PTZValues.Set(axis, val)` for axis-agnostic PTZ access — eliminates switch statements on axis names
- `PID` and `SourceID` are branded types via `github.com/larsartmann/go-branded-id` (phantom typing) — prevent mixing process IDs and PipeWire source IDs at compile time. Defined in `internal/pixy/ids.go`.
- Metrics use OpenTelemetry SDK (`go.opentelemetry.io/otel/exporters/prometheus`) with `promhttp.Handler()` for the `/metrics` endpoint. `prometheus/client_golang` kept only for `promhttp`.

### Testing

- Standard `testing` package only (no testify)
- **`newTestDaemon(t, camera, videoDev, hidrawDev, opts...)`** is the canonical builder — uses `t.TempDir()` for `StateDir` so parallel tests never race on a shared state file, use `withInCall()` or custom `testDaemonOption` to inject mock deps
- Function fields (`isCameraInUseFn`, `findSourceFn`, `setSourceFn`, `notifyFn`, `setTrackingFn`, `setAudioFn`, `setGestureFn`, `centerCameraFn`, `v4l2SetFn`) default to no-op stubs or real implementations in tests
- Predefined test options: `withInCall()`, `withAutoOff()`, `withCameraInUse()`, `withNotifyCalled()`, `withNotifyMessages()`, `withFindSource()`, `withCaptureTracking()`, `withCaptureAudio()`, `withCaptureGesture()`, `withCaptureGestureArg()`, `withCaptureCenter()`, `withNoopV4L2()`, `withNoopTracking()`, `withNoopAudio()`, `withDebounceCount()`
- `testDaemonNoDevice(t)`, `testDaemonWithDevice(t, camera)`, and `testDaemonWithState(t, camera, inCall)` are convenience wrappers
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
- **GOEXPERIMENT=jsonv2 required**: The project uses `encoding/json/v2` (the new experimental JSON API). All `go build`/`go test`/`golangci-lint` commands must set `GOEXPERIMENT=jsonv2`. Set in CI (`go-test.yml`), nix build (`package.nix`), lint derivation and devShell (`flake.nix`). When Go 1.27 ships, `encoding/json/v2` becomes stable and the flag can be removed.
- **Audio mode shorthand**: CLI accepts `"org"` but the stored/displayed value is `"original"`. `ParseAudioMode` maps both.
- **PTZ units**: V4L2 uses 1/3600-degree units internally (`v4l2UnitsPerDegree = 3600`). The daemon presents user-facing degrees but multiplies before sending to `v4l2-ctl`. Zoom is not multiplied.
- **PTZ limits are hardware-verified**: `PanRange = Range{Min: -150, Max: 150}`, `TiltRange = Range{Min: -90, Max: 90}`, `ZoomRange = Range{Min: 100, Max: 150}` in `internal/pixy/pixy.go`. These were verified empirically against the EMEET PIXY via `v4l2-ctl --list-ctrls` (`pan_absolute`/`tilt_absolute` range ±540000/±324000 at 3600 units/°, `zoom_absolute` 100–150). Tilt sign convention: **positive = up** everywhere (V4L2 read, V4L2 write, web slider, keyboard arrows) — no inversion anywhere.
- **`Axis` branded type**: `pixy.Axis` (`type Axis string`) replaces raw string params for PTZ axis names. Constants: `pixy.AxisPan`, `pixy.AxisTilt`, `pixy.AxisZoom`. `PTZValues.Get(axis Axis)` and `Set(axis Axis, val int)` are type-safe. At boundaries (commands.go, handlers.go), convert with `pixy.Axis(parts[0])` or `pixy.Axis(request.PathValue("axis"))`.
- **PTZ value parsing**: `parsePTZValue` in `ptz.go` treats bare numbers as **absolute** (including negatives like `-90`). Relative mode requires an explicit `rel` prefix (e.g. `rel+10`, `rel-5`). Relative commands read current position via `d.deps.parsePTZ`. Tests exercising relative PTZ MUST inject `withNoopParsePTZ()` (deterministic `{0,0,0}`) — otherwise they read real `/dev/video0` and flake on hardware state. `newPTZDaemon` wires this by default; tests using `newTestDaemon` directly must add it.
- **`Range` struct type**: `pixy.Range{Min, Max int}` with `Clamp(v int) int` method replaces separate PanMin/PanMax/TiltMin/TiltMax/ZoomMin/ZoomMax constants. `PTZValues.Clamp()` uses `PanRange.Clamp()`, etc.
- **`HIDDevice` embeds `fmt.Stringer`**: Every HID implementation must provide `String() string` for error context. `hidrawDevice.String()` returns the device path; `failingHID.String()` returns `"failing-hid"`. `queryHIDState` errors include `dev=%s`.
- **Socket bind requires `ReadWritePaths`**: `ProtectSystem=strict` in the NixOS module makes `/run` read-only. The module sets `ReadWritePaths = ["/run/emeet-pixyd"]` to allow socket file creation.
- **Named noop stubs**: Test noop functions (`noopV4L2Set`, `noopSetTracking`, `noopSetAudio`, `noopSetGesture`, `noopCenterCamera`, `noopParsePTZ`) are package-level functions shared by `withNoop*` option builders and `noopDependencies()`. No closure duplication.
- **`webStatus` struct lives in `web_types.go`**, not in `templates.templ`. Fields `Camera`, `Audio`, `Auto` use typed `pixy.CameraState`/`pixy.AudioMode`/`pixy.AutoMode` for compile-time safety. Templates use typed comparisons (`s.Camera == pixy.StateTracking`) instead of raw string literals.
- **Generated file**: `templates.templ` must be compiled with `templ generate` before `go build`. The nix build runs `templ generate` automatically in `preBuild`. The generated `_templ.go` file is gitignored. Use `go generate ./...` to rebuild locally. Templates import `pixy` for typed constants (`pixy.StateTracking`, `pixy.AudioNC`, etc.).
- **Template `load` trigger was removed**: The `#status-panel` div uses `hx-trigger="every 3s, refresh from:body"` (no `load`). The `load` trigger caused an infinite loop with `outerHTML` swap because each swapped element re-triggered `load`. Initial page renders the panel server-side via `@statusPanel(s)` in `page()`.
- **Scripts at end of body**: HTMX and `app.js` are loaded at the end of `<body>` (not in `<head>`) because `app.js` accesses `document.body` at line 12 which would be `null` in `<head>`.
- **Build tags**: All `.go` files in the root use `//go:build linux`. Tests that test Linux-specific code naturally require a Linux host.
- **`flaky` test awareness**: Some tests probe real sysfs (e.g., `TestProbeDevices_SetsStateToOfflineWhenNoVideo`), so they may pass or fail depending on whether a PIXY is physically connected. These tests handle both outcomes gracefully.
- **Branded ID `String()` includes prefix**: `go-branded-id` v0.3.0 changed `String()` output to include the brand name prefix (e.g., `"PID:42"` instead of `"42"`). Use `.Get()` for operational paths (`/proc/{PID}`, `wpctl set-default {ID}`) and `.String()` only for logging/display. Tests should assert `.Get()` for value comparisons.
- **Nix build uses `proxyVendor = true`**: The standard FOD-based vendor approach fails because `templ generate` (in `preBuild`) produces imports that weren't visible when the FOD ran `go mod vendor`. `proxyVendor = true` downloads deps during the build via Go module proxy, after `templ generate` has run. The `vendorHash` is a SHA of the downloaded module tarballs, not of the vendored source tree.
- **`nix flake check` passes**: The go-modules FOD inherits `preBuild`, so `templ generate` runs before modules are available and produces empty output. The `preBuild` must NOT contain an empty-file validation guard (it kills the FOD). Simple `preBuild = "templ generate"` works because the FOD doesn't compile — the main build regenerates correctly with modules present. The lint derivation must set `HOME=$TMPDIR` and `GOCACHE` BEFORE `runHook preBuild`.
- **WebAddr default**: `127.0.0.1:8090` (localhost only, not `:8090`)
- **StateDir default**: `/run/emeet-pixyd` (tmpfiles.d rule in NixOS module creates this)
- **Binary symlink**: `package.nix` creates `emeet-pixy` symlink pointing to `emeet-pixyd` for CLI usage
- **Git-derived version**: `package.nix` receives `version` from `flake.nix` as `self.shortRev or self.dirtyRev or "dev"`. No hardcoded version string. `doCheck = false` because nix sandbox lacks `/dev/hidraw*` and `/dev/video*` devices needed by integration tests. CI runs tests via GitHub Actions (`go test`), not nix.
- **Gosec exclusions are intentional**: `.golangci.yml` excludes G304 (file inclusion), G204 (subprocess launch), G706 (log injection), G115 (integer overflow) because this hardware daemon inherently opens `/dev/hidraw*`, `/dev/video*`, and launches `ffmpeg`/`v4l2-ctl`/`wpctl`. These are not fixable — suppressing in config is cleaner than per-site `//nolint` comments.
- **golangci-lint v2 exclusions**: In v2, `exclusions` is a top-level key (not under `issues`). It works alongside `linters.enable` — no conflict. Test files are excluded from `exhaustruct`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `err113`, `noinlineerr`, and `unused` via `exclusions.rules`.
- **Lint is clean (0 issues)**: ALL golangci-lint v2 linters are enabled in `.golangci.yml`. Specific false positives are suppressed via `//nolint` directives at call sites (e.g., `//nolint:exhaustruct` for intentional partial struct initialization, `//nolint:gochecknoglobals` for OTel metric vars and package-level middleware vars). Gosec excludes cover hardware-daemon patterns (G104, G107, G115, G204, G301, G304, G306, G702, G703, G706). A targeted `.golangci.yml` exclusion rule suppresses `godoclint` on `main.go` — the generated `templates_templ.go` has a comment above `package main` that godoclint's cross-file analysis counts as a second package doc, but path exclusions (`_templ\.go$`) are applied AFTER the analysis, so the false positive fires on `main.go`.
- **`makezero` uses `always: false`**: The `always: true` mode false-positives on every `make([]byte, N)` I/O read/`copy()` buffer (9 sites: hid, socket, uevent, stream, cache, ipc). The default `always: false` mode still catches the real bug (`make([]T, n)` + `append`) without flagging legitimate buffer allocations. Do NOT set `always: true`.
- **`init()` eliminated from `metrics.go`**: OTel metrics registration is now lazy via `sync.Once` in `registerMetrics()`, called from `NewDaemon()` and `updateMetrics()`. No `init()` functions exist anywhere in the codebase.
- **`errorPrefix` constant**: `"error: "` is a named constant in `errors.go` used by `CommandError.Error()`, `IsCommandErrorResponse()`, and all `fmt.Errorf` error formatting in `commands.go`.
- **pprof gated behind `Debug` config**: Pprof endpoints (`/debug/pprof/*`) are only registered when `Config.Debug` is `true`. Default is `false`. The NixOS module exposes `hardware.emeet-pixy.debug` option.
- **`t.Parallel()` in all tests**: All test functions in `integration_test.go` and subtests call `t.Parallel()`. No `tc := tc` captures needed (Go 1.22+ loop variable semantics).
- **Test coverage for auto/process**: `auto_test.go` tests `handleCallStart`, `handleCallEnd`, `autoManage` state transitions. `process_test.go` tests `ppidOf`, `isDescendantOf`, `isCameraInUse` using real `/proc` filesystem.
- **`webStatus` uses typed fields**: `Camera pixy.CameraState`, `Audio pixy.AudioMode`, `Auto pixy.AutoMode`. Templates use typed comparisons (`s.Camera == pixy.StateTracking`) instead of raw string literals, eliminating the previous split brain.
- **Unified toast/error pattern**: All HTTP handlers use `applyResponseToStatus()` to set either `status.Error` (on command error) or `status.Toast`+`status.ToastType` (on success). The toast type (`success`, `info`, `error`) is propagated from `actionToast()` through `applyResponseToStatus()` — not hardcoded.
- **Handler extraction**: `handlers.go` (was 624 lines) extracted into `metrics.go` (OTel registration + updateMetrics), `stream.go` (MJPEG streaming + JPEG parsing), `ptz.go` (PTZ logic extracted from handlers + v4l2), `sse.go` (SSE broadcasting + HTTP middleware + helpers, ~315 lines), `middleware.go` (28 lines — just middleware variable assignments). Handlers.go is now ~310 lines focused on HTTP routing and web handlers.
- **Command constants**: All command strings are named constants in `commands.go`: `cmdTrack`, `cmdIdle`, `cmdPrivacy`, `cmdTogglePrivacy`, `cmdCenter`, `cmdSync`, `cmdProbe`, `cmdAudio`, `cmdAuto`, `cmdAutoOn`, `cmdAutoOff`, `cmdToggleAuto`, `cmdGestureOn`, `cmdGestureOff`, `cmdToggleGesture`, `cmdStatus`, `cmdWaybar`, `cmdDevice`. No raw string literals for command names anywhere in the codebase.
- **`waybarJSON` struct**: Waybar output uses a typed struct with json tags (`waybarJSON`) and `strings.Builder` for tooltip construction. Optimized from 23 allocs/860ns to 7 allocs/334ns.
- **`TestUpdateMetrics` is NOT parallel**: Tests global mutable metrics state, must run serially to avoid interference from parallel tests calling `updateMetrics()`. `TestSendCommand_EndToEnd` and `TestConfigFromEnv_DefaultsWhenUnset` both use `t.Parallel()` (temp socket dir and env reads respectively are safe to parallelize).
- **Auto-manage uses DI functions**: `handleCallStart` and `handleCallEnd` call `d.setTrackingFn()` and `d.setAudioFn()` (not the direct methods). This ensures the auto-manage path is mockable in tests.
- **Toast type propagation**: `applyResponseToStatus()` accepts a `toastType` parameter. `actionToast()` returns both message and type (`success`/`info`). All callers propagate the type correctly.
- **Response string constants**: `respTrackingOn`, `respPrivacyOn`, `respTrackingOff`, `respAutoModeOff`, `respAudioUsage`, `respAutoUsage`, `respDeviceNotFound` in `commands.go`. All command handlers return these constants instead of inline strings.
- **`TestHandleCommandTogglePrivacy` tests mock behavior**: The test uses `withCaptureTrackingSlice` to verify the DI function was called with the correct argument, and asserts the exact response string (`respTrackingOn`). It does NOT assert `d.state.Camera` because the mock bypasses `setDeviceState` (which is the only path that mutates daemon state via the `stateMutator` callback).
- **PTZ axis constants**: `pixy.AxisPan`, `pixy.AxisTilt`, `pixy.AxisZoom` in `internal/pixy/pixy.go`. Used by `handlers.go` (PTZ lookup table), `commands.go` (command routing), and `v4l2.go` (V4L2 control names).
- **Middleware reimplemented locally**: `securityHeadersMiddleware`, `requestIDMiddleware` (generates short request IDs), `loggingMiddlewareFactory(slog.Logger)` are all in `http.go`. The chain is applied via local `chain()` function in `main.go`. Request IDs are short hex strings generated via `crypto/rand` (8 chars) — ULID-based IDs from cqrs-htmx were removed.
- **PTZ axis lookup table**: `ptzAxes` map in `handlers.go` replaces 4 scattered switch-based functions (`ptzLimits`, `ptzAxisLabel`, `ptzAxisUnit`, `ptzAxisValid`). Each entry is a `ptzAxisInfo{Min, Max, Label, Unit}`. Keys use `pixy.AxisPan`/`pixy.AxisTilt`/`pixy.AxisZoom`.
- **`PTZValues.Clamp()`**: Domain method in `internal/pixy/pixy.go` clamps all three axes in one call. Returns a new `PTZValues` without mutating the receiver.
- **`probeResult` struct**: `probeDevices()` in `probe.go` is a pure function returning `probeResult{VideoDev, HidrawDev}`. `Daemon.applyProbeResult()` applies it under lock.
- **Named cache types**: `lastFrameCache` and `ptzCache` in `cache.go` are named types with `Get()`/`Set()`/`Invalidate()` methods, replacing anonymous embedded structs with direct mutex access.
- **State validation**: `loadState()` calls `loaded.Valid()` to reject garbage enum values. `pixy.State.Valid()` checks Camera, Audio, and Auto fields.
- **Stream constants**: `streamBufSize` and `ffmpegShutdownTimeout` live in `stream.go`, not `handlers.go`.
- **Toast constants**: `toastAudioChanged`, `toastGestureToggled`, `toastAutoToggled` in `handlers.go`.
- **NixOS systemd hardening**: `ProtectSystem=strict`, `PrivateTmp=true`, `NoNewPrivileges=true`, `RestrictAddressFamilies=[AF_UNIX AF_NETLINK AF_INET]`, `MemoryMax=256M`.
- **JPEG max-iterations**: `extractJPEGFrame` has a 10M iteration guard to prevent infinite loops on corrupt streams.
- **Uevent retry**: Transient read errors use `continue` instead of `return` to keep the hotplug listener alive.
- **Debounce counters capped**: `autoManage` caps `debounceInUse` and `debounceIdle` at `debounceCount` to prevent unbounded growth.
- **`centerCamera` uses DI**: `centerCamera()` calls `d.v4l2SetFn()` per-axis instead of `v4l2SetMultiple` directly — fully testable via `withNoopV4L2()`.
- **`cleanupFFmpeg` nil guard**: Checks `cmd.Process == nil` before signaling to prevent panic when ffmpeg failed to start.
- **`matchesPixyID` unified probe helper**: Single function replaces `hasPixyProduct` and `hasPixyVendorProduct`. Takes prefix, separator, and vendor/product index parameters to handle both `PRODUCT=vendor/product/version` and `HID_ID=bus:vendor:product`.
- **Partial device match logging**: `probeDevices()` logs a warning when only video or only hidraw is found (partial match).
- **`v4l2SetMultiple` removed**: Center camera now uses per-axis `v4l2SetFn` calls. The batch function is no longer needed.
- **Benchmarks**: 9 established — `BenchmarkExtractJPEGFrame`, `BenchmarkFormatLastSynced`, `BenchmarkParseHIDResponse`, `BenchmarkWaybarOutput`, `BenchmarkHandleCommand_Query`, `BenchmarkHandleCommand_Mutating`, `BenchmarkGetWebStatus`, `BenchmarkWriteSSEEvent`, `BenchmarkBroadcasterBroadcast`.
- **`ParseAudioMode` accepts full names**: Both `org` (shorthand) and `original` (full name) are accepted. This lets users type `audio original` on the CLI.
- **`Config.Validate()` checks enum fields**: Validates `AutoMode` and `DefaultAudio` in addition to StateDir, PollInterval, DebounceCount, and WebAddr. Invalid enum values prevent the daemon from starting.
- **Bare `auto` command shows current mode**: `auto` without arguments reports the current auto mode instead of silently setting it to full. Use `auto-on`, `auto-off`, or `auto <mode>` to change.
- **`--version` and `--help` flags**: `emeet-pixyd --version` prints version, `emeet-pixyd --help` prints usage. Both handled by `handleFlag()` before CLI command dispatch.
- **`/api/health` endpoint**: Returns JSON with `status`, `camera`, and `version`. Returns 503 when device is offline, 200 when online.
- **`AutoMode.Toggle()` method**: Domain type method encapsulates toggle logic (off→full, on→off). Used by `handleAutoCommand`.
- **`handleCommand` refactored**: Query commands (waybar, version, sync, probe, device) extracted into `handleQueryCommand()`. Toggle-privacy extracted into `handleTogglePrivacy()`. Reduces cyclomatic complexity.
- **`device` command shows both paths**: Output now includes both `/dev/videoX` and `/dev/hidrawY`.
- **Audio toast shows mode name**: Web UI shows "Audio: nc" instead of generic "Audio mode changed".
- **Temp state file cleanup**: `loadState()` removes leftover `.tmp` files from crashed writes.
- **justfile removed**: Deprecated in favor of flake.nix. No justfile in the project.
- **PTZ success toasts suppressed**: `handlePTZ` passes empty toast on success to avoid visual spam during slider drag. Error toasts still shown.
- **`TestAutoManage_NoDevice_Returns` skips when device present**: Detects real hardware via `probeVideo4linux()` and `t.Skip()` if found.
- **`newTestDaemon` wires REAL HID/V4L2 impls by default**: `setTracking`, `setAudio`, `setGesture`, `centerCamera`, `v4l2Set` default to the real `d.*` methods (which open `/dev/hidraw7` / run `v4l2-ctl`). Tests that assert on `autoError` or other side effects WITHOUT exercising HID must inject no-op stubs (`withNoopTracking()`, `withNoopAudio()`, `withNoopV4L2()`), otherwise they fail on hardware-bearing machines where `/dev/hidraw7` is inaccessible (permission denied) and pass only by accident on hardware-less CI.
- **No `init()` functions**: All `init()` functions eliminated. Metrics registration is lazy via `sync.Once` in `registerMetrics()`, called from `NewDaemon()` and `updateMetrics()`.
- **`handleHealth` uses typed struct**: `healthResponse` struct with `json.Marshal` instead of `fmt.Fprintf` JSON template — proper escaping.
- **`PTZValues.Get/Set` for axis-agnostic access**: All PTZ code uses `Get(axis)`/`Set(axis, val)` instead of switch-on-axis. `status.Get(axis)`, relative PTZ, and `parsePTZValues` all use these methods. The former `ptzAxisValue` wrapper was removed — callers use the embedded `pixy.PTZValues.Get` directly.
- **V4L2 control names centralized**: `ptzAxes` map has `V4L2Ctrl` and `Multiplier` fields. `v4l2CtrlToAxis` reverse map converts V4L2 output back to axis names. `v4l2GetCtrlList()` builds the `--get-ctrl` argument from the map. Zero hardcoded `"pan_absolute"` etc. strings in production code.
- **Map-based lookups replace switches**: `actionToasts` map in `handlers.go`, `waybarCameraStates` map in `waybar.go` — new entries only require adding to the map.
- **External binary constants**: `ffmpegBin`, `wpctl`, `notifySend`, `v4l2ctl` — no raw binary name strings in production code.
- **`parsePTZValues` fully DI-wired**: All callers route through `d.deps.parsePTZ` (commands.go, device.go `getStatus`, handlers.go `getWebStatusWithPTZ`). No direct calls to `parsePTZValues` in production code — the DI seam is consistent and testable.
- **`metricsInstance` nil-safe**: All `record*` functions and `updateMetrics` guard against nil `metricsInstance` (if OTel exporter init fails, metric recording silently no-ops instead of panicking). `collectMetrics` (test-only, in `metrics_test.go`) returns an error if nil.
- **`checkDevice` lives in `stream.go`**: Its only caller is `setupStream`. Moved from `handlers.go` for cohesion.
- **`findSource` errors propagate to `autoError`**: `handleCallStart` in `auto.go` now appends PipeWire source failures to `errs` (visible in web UI) and logs them. Previously silently swallowed.
- **`ConfigFromEnv` logs invalid values**: `slog.Warn` for unparseable `EMEET_PIXYD_POLL_INTERVAL`, `DEBOUNCE_COUNT`, `AUTO`, `DEFAULT_AUDIO`. Users now know when their env config is wrong instead of silently getting defaults.
- **Live UI updates via SSE (local implementation)**: `/api/events` endpoint uses `newSSEStream(w, r)` + `Broadcaster` for thread-safe fan-out. The handler sends a `connected` event on join, then forwards `refresh` events from the broadcaster. `app.js` connects on load, reconnects with exponential backoff, and closes the connection when the tab is hidden. HTMX v2.0.9 served as static file at `/static/htmx.js` (embedded in `static/htmx.js`).
- **State-change broadcasting**: `broadcastStateChanged()` calls `d.broadcaster.Broadcast(SSEEvent{...})` after every state mutation (`setDeviceState`, `syncState`, `handleCallStart`, `handleCallEnd`, `applyProbeResultLocked`). The local `Broadcaster` is a thread-safe fan-out hub — non-blocking sends drop events for slow clients (buffered channel per subscriber). The `Daemon.broadcaster` field (`*Broadcaster`) is initialized in both `NewDaemon()` and `newTestDaemon()`.
- **Log level conventions**: `Warn` for degraded functionality (hotplug disabled, max JPEG iterations reached); `Info` for normal lifecycle; `Error` for failures requiring attention; `Debug` for optional/verbose detail.
- **`vendorHash` is shared between `flake.nix` and `package.nix`**: Both files must be kept in sync when dependencies change. The hash is a SHA of module tarballs downloaded via `proxyVendor = true`, not of vendored source. All dependencies are on the public Go module proxy (cqrs-htmx removed — no private repos).
- **UI layout — preview as hero**: The camera preview is full-width above the status panel (not in a 2-column grid). The preview is rendered once in `page()` and does NOT re-render on HTMX swaps (only `#status-panel` swaps). The preview online/offline state is handled by `app.js` error recovery, not server-side swaps. When offline, a placeholder card replaces the PTZ section inside the panel.
- **Camera mode cards**: Three selectable cards (`cameraModeCard` template) replace the old button group. Each has an SVG icon, name, description, and keyboard shortcut badge. Active state uses per-mode color glow: `mode-track.active` (green), `mode-idle.active` (yellow), `mode-privacy.active` (red). Icons are inline SVG (Lucide-style stroke) defined as templ sub-templates.
- **Audio segmented control**: Three `audioSegment` buttons inside a `.segmented` container (was `audioBtn`). Active segment highlighted with accent color. Posts `hx-vals='{"mode":"<mode>"}'` to `/api/audio`.
- **PTZ radar**: `ptzRadar(pan, tilt, zoom)` template renders a 120px circular position indicator with crosshair, inner ring, zoom ring, center dot, and a glowing position dot. Dot position set via CSS custom properties `--pan-x` and `--pan-y` (percentages computed from pan/tilt ranges). Zoom ring size set via `--zoom-pct` (scales 30%→80% of radar diameter). The radar lives inside the PTZ card in the panel. `app.js` `updateRadar()` updates all three axes live as sliders move. Server re-renders correct position on panel refresh.
- **Snapshot button**: Overlay button on the preview (only when online). `app.js` fetches `/api/snapshot`, converts response to blob, triggers download as `pixy-YYYY-MM-DD-HH-MM-SS.jpg`, shows toast on success/failure.
- **Keyboard shortcut legend**: Fixed-position panel (bottom-left), toggled via `?` key, FAB button click, or `Escape` to close. Lists all shortcuts: T/I/P/C (modes + center), arrow keys (pan/tilt), +/- (zoom), ? (help). FAB has `aria-expanded` attribute.
- **Removed templates**: `stateIndicator`, `cameraBtn`, `audioBtn` were replaced by `cameraModeCard`, `audioSegment`, and inline state display in mode cards. No external code references them.
- **Preset UI**: `presetSection(names)` template renders inside the PTZ card when online. Shows save input + button, preset count (N/16), and preset chips. Each chip has a load button (click name → POST `/api/preset/load/{name}`) and delete button (× → POST `/api/preset/delete/{name}`). Save handler uses delegated events in `app.js` (survives HTMX panel swaps) — Enter key or button click POSTs to `/api/preset/save/{name}` with trimmed input value. `webStatus.PresetNames []string` is sorted in `getWebStatus()` from `state.Presets` map.
- **Preview placeholders use SVG**: Both offline and fallback states use `iconCameraOff()` (camera with diagonal slash) instead of emoji. All icons in the UI are now inline SVG — no emoji anywhere.

### External Libraries Considered

- **`go-error-family`** (same author, `github.com/larsartmann/go-error-family` v0.8.0): **ADOPTED** for behavioral error classification. Registers all daemon sentinels with Families (Infrastructure/Rejection/Transient) so `errorfamily.HTTPStatus(err)` derives correct HTTP status codes, `errorfamily.ExitCode(err)` derives BSD sysexits exit codes, and `errorfamily.LogError()` adds family/code/retryable structured log fields with correct severity. Used in `stream.go` (7 typed stream errors via `NewInfrastructure()`, fixed 3x 500→503 bugs), `handlers.go` (preset validation HTTP status), `main.go` (CLI exit codes + structured logging at daemon init failure). Scoped adoption: HTMX action handlers intentionally return 200+HTML toast (correct HTMX pattern), NOT classified JSON errors. Circuit breaker logic untouched (more sophisticated than binary `IsRetryable`).
- **`cqrs-htmx/v2`** (same author): HTMX-aware web infrastructure library. **Evaluated and REMOVED** — was briefly adopted for SSE, middleware, embedded HTMX JS, and WriteJSON, but pulled in ~30 transitive deps including a private repo (`go-cqrs-lite`) that broke `nix build`. The 8 used features were reimplemented locally in `sse.go` (~290 lines). All deps are now on the public Go module proxy.
- **`templ-components`** (same author): Tailwind CSS component library for templ. Not adopted because emeet-pixyd uses hand-crafted custom CSS (glass morphism dark theme with CSS variables), not Tailwind. The UI is a purpose-built hardware control panel with 6 cards — generic component library overhead is unjustified.

- **License**: MIT. The `flake.nix` (`license` block), `package.nix` (`lib.licenses.mit`), `LICENSE` file, `README.md`, and website (`package.json`, JSON-LD, footer) are all consistent. Previously was Proprietary but changed to MIT on 2026-07-14.

---

## NixOS Module

`modules/nixos.nix` provides:

- `hardware.emeet-pixy.enable` option
- udev rules for PIXY hidraw + video4linux access
- systemd user service (runs after pipewire + graphical-session)
- Installs `v4l-utils`, `wireplumber`, `libnotify` in service PATH
- Creates `/run/emeet-pixyd` tmpfiles.d entry

---

## Website

The `website/` directory contains an Astro + Starlight documentation site deployed to Firebase Hosting at `emeet-pixyd.lars.software`.

| File                                      | Purpose                                                                                |
| ----------------------------------------- | -------------------------------------------------------------------------------------- |
| `website/astro.config.mjs`                | Astro config: Starlight sidebar, sitemap, CSP, Tailwind, fonts                         |
| `website/firebase.json`                   | Firebase Hosting config (target: `emeet-pixyd`, cache headers, CSP)                    |
| `website/.firebaserc`                     | Firebase project mapping (project: `lars-software`, target: `emeet-pixyd`)             |
| `website/flake.nix`                       | Nix flake with `dev`, `build`, `preview`, `deploy` apps                                |
| `website/package.json`                    | Dependencies: Astro 7, Starlight, Tailwind v4, astro-og-canvas                         |
| `website/scripts/fix-csp.mjs`             | Post-build script that hashes inline scripts and injects CSP SHA-256 hashes            |
| `website/src/pages/index.astro`           | Landing page composition                                                               |
| `website/src/pages/og/[...slug].ts`       | OG image generation via astro-og-canvas (violet border)                                |
| `website/src/layouts/LandingLayout.astro` | HTML shell (SEO, OG, JSON-LD, CSP)                                                     |
| `website/src/components/`                 | 14 Astro components (Hero, Features, HowItWorks, Comparison, etc.)                     |
| `website/src/data/`                       | TypeScript data modules (config, features, sections, types, hero-code)                 |
| `website/src/styles/`                     | global.css (Tailwind v4 + violet theme) + starlight.css (Starlight overrides)          |
| `website/src/content/docs/`               | 16 MDX documentation pages                                                             |
| `website/public/`                         | favicon.svg, manifest.json, robots.txt, JS (theme-init, header, copy-code, animations) |

### Website Conventions

- **Accent color**: Violet (`#8b5cf6`) — distinct from go-atomic-write (emerald) and gogenfilter (cyan)
- **CSP enabled** via Astro's `security.csp` config + post-build `fix-csp.mjs` script
- **OG images** generated per-page via astro-og-canvas with violet border
- **Build**: `npm run build` runs `astro build && node scripts/fix-csp.mjs`
- **Deploy**: `nix run .#deploy` runs `npm run build && firebase deploy --only hosting`
- **Node 24** (`.node-version`)
- **TypeScript strict** mode — clean typecheck expected

---
