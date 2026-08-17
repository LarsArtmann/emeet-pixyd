# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Changed

- **BREAKING**: Replaced HTMX v2.0.9 with DataStar v1.0.2 (`datastar-go` SDK v1.2.2). All `hx-*` attributes converted to `data-*` attributes. Action handlers now return SSE patches (`PatchElementTempl`) instead of HTML fragments — DataStar morphs elements by ID automatically. Eliminated ~275 lines of custom JS (SSE bridge, HTMX lifecycle, focus preservation, PTZ helpers, toast rendering) from `app.js` (510→235 lines). Deleted 82 KB `htmx.js`, added 34 KB `datastar.js`. PTZ radar is now reactive via DataStar signals (`data-style` CSS custom properties). CSP updated to include `'unsafe-eval'` for DataStar expression evaluation. Audio endpoint changed from `POST /api/audio` (form value) to `POST /api/audio/{mode}` (path value).

### Added

- **Website relaunch** (`emeet-pixyd.lars.software`): landing pitch rewritten around the origin story — hero badge "Reverse-engineered for Linux", H1 "A great AI webcam, dumb on Linux. Until now.", metrics row led by ±150° pan + MIT license. New `WhySection` (three story cards: the hardware / the problem / the fix) and `ShowcaseSection` embedding a 25s HyperFrames-rendered demo video (`/demo.mp4`, 1920×1080) plus three real web-UI screenshots captured headless against the running daemon. New `camera` Lucide icon. Site title/description updated to match. `firebase.json` now long-caches `mp4`/`webm`/`mov`. The HTTPS custom-domain certificate reached `CERT_ACTIVE` (Google Trust Services) — the ACME TXT record staged in terraform turned out to be unnecessary (CNAME validation sufficed). Screenshots show the offline UI state; retake with the camera connected is tracked as `TODO_LIST.md` #129.
- **Error classification via `go-error-family` (v0.10.0)**: `errorfamily.go` registers 18 daemon sentinels + stdlib defaults into Infrastructure/Rejection/Transient families. `errorfamily.HTTPStatus(err)` derives HTTP status codes, `errorfamily.ExitCode(err)` derives BSD sysexits CLI exit codes, and `errorfamily.LogError()` adds family/code/retryable structured log fields at the daemon-init failure path — replacing per-call-site hardcoded status/exit codes. Scoped by design: HTMX action handlers keep returning 200+HTML toast (correct `outerHTML` swap), and the HID circuit breaker stays untouched. See `FEATURES.md` → Error Handling.
- **Website docs retrofit** (`emeet-pixyd.lars.software`): added curated "Where to go next" sections to all 17 docs pages, converted the installation blockquote to a Starlight callout, expanded the `related-tools` comparison matrix, enabled `lastUpdated` + `editLink` in Starlight, and added `.htmlvalidate.json` (0 HTML-validation errors). Deployed and live.
- **Gesture toggle in the web UI**: a Gesture Control toggle (posts to `/api/gesture`) joins the mode cards — gesture control was previously CLI/socket-only.
- **Web UI overhaul**: camera preview is now a full-width hero above the status panel (does not re-render on HTMX swaps). Camera mode cards (Track/Idle/Privacy) with inline SVG icons, descriptions, per-mode color glow, and keyboard-shortcut badges replace the old button group. Audio selector is a pill-style segmented control. PTZ radar indicator (120px circular position display with crosshair, zoom ring, and glowing position dot) added. Snapshot button overlays the preview. Keyboard shortcut legend (FAB + `?` key + `Escape`) lists all shortcuts. Preset UI (save input + chips with load/delete, delegated events that survive HTMX panel swaps). All UI icons are now inline SVG (Lucide-style) — no emoji anywhere. Responsive breakpoints at 860px/640px/400px + `prefers-reduced-motion` + `hover:none`.
- Camera preset web UI: save/load/delete named PTZ positions via `/api/preset/{save,load,delete}/{name}` with preset count display (N/16).
- State schema versioning: `pixy.CurrentSchemaVersion = 1` + `State.SchemaVersion` field (`"v"` in JSON). `loadState` logs a warning on version mismatch and loads old files as version 0 (best-effort backward compatibility).
- Preset name validation: `pixy.ValidatePresetName()` — non-empty, ≤32 runes, no path separators, no control chars. Wired at both CLI and HTTP save boundaries (security/data-integrity gap closed).
- `pixy.PresetMap` domain type with `SortedNames()`, `Get()`, `IsFull()` methods. `State.Presets` is now typed. `MaxPresets` constant moved from package main to `internal/pixy`.

### Changed

- `maxPresets` moved to `internal/pixy` as `pixy.MaxPresets`; template references updated.
- `pixyCommit` now uses named constants instead of raw hex bytes. `respAutoModePrefix` extracted from 3 repetitions.
- `getWebStatus` PTZ initialization fixed: explicit `PTZValues{Pan:0, Tilt:0, Zoom:ZoomDefault}` (was zoom-only init leaving Pan/Tilt implicit).
- Duplicate `ci` devShell in `flake.nix` removed.

### Fixed

- **3 genuine HTTP status bugs in `stream.go`**: `stream.not_supported`, `stream.pipe_error`, and `stream.start_error` returned 500 (Internal Server Error) for infrastructure failures; they now return 503 (Service Unavailable) via `errorfamily.HTTPStatus(err)`.
- **`nix build` FOD failure** from `go-branded-id@v0.5.0` shipping a committed compiled `namer` binary that embeds nix store paths (`/nix/store/.../go-1.26.5`): added an in-sandbox-only `replace` (`goBrandedSrc` + `replaceBrandedId` in `flake.nix`/`package.nix`) so the poisoned module is never downloaded. Committed `go.mod`/`go.sum` stay canonical (GitHub Actions `go test` against the real proxy is unaffected). **TEMPORARY** — until `go-branded-id` publishes a binary-free version (tracked as `TODO_LIST.md` #124).
- **`nix flake check` failure** (broken since project inception): the go-modules FOD inherited `preBuild` with an empty-file validation guard that killed the FOD when `templ generate` produced empty output (no modules available). Simplified `preBuild` to bare `templ generate`; reordered `HOME=$TMPDIR` before `runHook preBuild`.
- Dead `SSEEvent.ID` and `SSEEvent.Retry` fields removed (never set in production — only in one test). Dead `writeSSEEvent` branches removed; `strconv` import dropped.
- Dead CSS removed: `.htmx-indicator` class was never used (project uses `#status-panel.htmx-request`).
- **Misleading zoom copy**: hero terminal and quick-start docs said `zoom 120 # Zoom to 120x` — zoom is a percentage (100–150), not a multiplier. Corrected to "Zoom to 120%" everywhere.
- Firebase deploy cache (`website/.firebase/`) was accidentally committed; removed from the index and gitignored.

### Fixed

- Race condition in parallel tests: `newTestDaemon` previously shared a fixed state directory under `/tmp`, causing data races when tests ran with `-race -count=N`. Now takes `testing.TB` and uses `t.TempDir()` so each test gets an isolated state file. Verified with `-race -count=10`.
- `makezero` linter false-positives on every `make([]byte, N)` I/O buffer. Set `always: false` (default mode) — still catches the real bug (`make([]T, n)` + `append`) without flagging legitimate pre-sized buffer allocations across 9 call sites (hid, socket, uevent, stream, cache, ipc).
- Empty `templates_templ.go` flake: `templ generate` can intermittently emit a zero-byte file, breaking the build. Root cause not fully resolved (tracked as TODO); CI now regenerates after checkout.

### Changed

- `newTestDaemon` signature changed: `func newTestDaemon(t testing.TB, ...)` — tests must pass the `*testing.T` or `*testing.B` as the first argument.

### Dependencies

- Added `github.com/larsartmann/go-error-family` (now at v0.10.0; adopted at v0.8.0, bumped in `ca41926`). Direct require. `vendorHash` synced between `flake.nix` and `package.nix`.
- `github.com/larsartmann/go-branded-id` at v0.5.0 (the version that ships the committed binary — see the TEMPORARY nix workaround above).
- Updated `vendorHash` for Go 1.26.4 module cache. Required sync between `flake.nix` and `package.nix` (both share the hash under `proxyVendor = true`).

### Changed

- **BREAKING**: Removed `cqrs-htmx/v2` dependency entirely (~30 transitive dependencies eliminated, including private `go-cqrs-lite`). SSE broadcasting, middleware, and HTTP helpers reimplemented locally in `sse.go` (~290 lines). HTMX JS embedded directly in `static/htmx.js` instead of served via library handler.
- **BREAKING**: PTZ values are now always absolute by default. `emeet-pixyd tilt -90` sets tilt to -90° instead of "go -90° from current position". Relative mode requires an explicit `rel` prefix: `tilt rel-5`, `pan rel+10`.
- **BREAKING**: PTZ limits corrected to match hardware reality: pan ±150° (was ±170°), tilt ±90° (was ±30°), zoom 100-150× (was 100-400×). Verified empirically via `v4l2-ctl --list-ctrls`.
- **BREAKING**: PTZ limit constants replaced with `Range` struct type: `pixy.PanRange`, `pixy.TiltRange`, `pixy.ZoomRange` (was separate `PanMin`/`PanMax`/etc constants). Includes `Range.Clamp(v int) int` method.
- **BREAKING**: PTZ axis names are now a branded `pixy.Axis` type instead of raw strings. Prevents accidental substitution of arbitrary strings into axis-keyed maps and functions.
- `queryHIDState` errors now include the device path for debugging (was generic "queryHIDState: ...").
- Uevent channel send now uses `select` with `ctx.Done()` to prevent goroutine leak on shutdown.
- Subprocess calls (`v4l2-ctl`, `wpctl`, `notify-send`) now route through `CommandRunner` interface with centralized slog logging of command, args, and duration.
- PTZ cache updated with set values instead of invalidated after a successful PTZ set, providing immediate accurate readback (avoids stale hardware values while motor is still moving).

### Fixed

- Flaky PTZ tests that read real `/dev/video0` state via `parsePTZValues` — now use `withNoopParsePTZ()` stub for deterministic behavior.
- Socket bind failure root cause: `ProtectSystem=strict` in NixOS module made `/run/emeet-pixyd` read-only. Added `ReadWritePaths` to allow socket creation.
- `hidrawDevice.String()` receiver consistency (pointer, matching other methods).

### Added

- `FuzzParsePTZValue` fuzz test for arbitrary CLI input robustness.
- `sse.go`: Local reimplementation of `Broadcaster` (thread-safe SSE fan-out), `sseStream` (per-client SSE), `writeSSEEvent`, `writeJSON`, `chain` (middleware chaining), `securityHeadersMiddleware`, `requestIDMiddleware`, `loggingMiddlewareFactory`.
- `static/htmx.js`: Embedded HTMX v2.0.9 minified JS (was served via `cqrshtmx.HTMXScriptHandler()`).
- `TestBehavior_PTZAbsoluteNegativeTilt`: proves bare negative values are absolute, not relative (seeds non-zero baseline to distinguish).
- `TestBehavior_PTZRelativeMath`: proves `rel` prefix triggers relative mode with correct math.
- Named noop stub functions (`noopV4L2Set`, `noopSetTracking`, etc.) shared by `withNoop*` builders and `noopDependencies()`.
- `assertSingleV4L2Call` helper deduplicates V4L2 call count assertions.
- `doGet` shared GET helper used by both `get` and `getStream` test helpers.
- `fmt.Stringer` embedded in `HIDDevice` interface for better error context.
- `commander.go`: `CommandRunner` interface with `realCommandRunner` (subprocess logging) and `noopCommandRunner`.
- `http.go`: HTTP helpers extracted from `sse.go` (`writeJSON`, `chain`, `statusRecorder`, middleware).
- `FuzzWriteSSEEvent` fuzz test for SSE event serialization robustness.
- `BenchmarkWriteSSEEvent` and `BenchmarkBroadcasterBroadcast` for SSE performance baselines.
- 9 unit tests for SSE internals: `writeSSEEvent` (5 cases), `Broadcaster` (3 cases), `splitSSELines` (6 cases).

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
