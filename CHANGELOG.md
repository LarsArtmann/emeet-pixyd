# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Official-app research deliverable** (`docs/emeet-studio-official-app-comparison.md`): full reverse-engineering comparison of EMEET STUDIO 2.0.3 (Windows EXE + macOS PKG) against emeet-pixyd — architecture table, supported-device matrix, HID command surface overlap, parity/official-only/pixyd-only tables, ranked gap recommendations (PTZ speed, battery status, tracking variants, motor presets → TODO #138–#141), and reusable Inno Setup 6.6.1 format notes. Includes a Zoom-specific section: the official app has zero call-app awareness; our `/proc/*/fd` detection is genuine differentiation.
- **Inno Setup 6.6.1 extractor** (`tools/inno661/`): pure-Python toolchain (extract, parse, extract-payload, verify) reverse-engineered from `jrsoftware/issrc@is-6_6_1` because innoextract 1.10-dev lacks 6.6.1 support. Discovers the loader table, de-chunks setup-0, parses all entry streams, extracts the solid LZMA chunk, and decodes Inno's CALL/JMP rel32 call-instruction optimization so output equals installed files. **All 2,211 payload files SHA256-verified against Inno's own recorded digests** — extraction proven byte-perfect. Format spec + committed parse data survive without the installer (TODO #142 done).
- **HID protocol official map** (`docs/hid-protocol-official-map.md`): official `CMD_*` surface ↔ our `hid.go`/`commands.go` implementation mapping with implemented/mappable/needs-bytes/unknown classification and official-implementation notes — the groundwork that gates feature TODOs #138–#141 (TODO #143 done).
- **HID command-ID table** (`tools/emhid/cmdtable.json` + `extract_cmdtable.py`): 162 official command IDs + payload layouts extracted from the Mac binary's `EMHidCmdV2Head` static initializers (arm64 disassembly + chained-fixup binds). Head format `[0x09, iface, category, cmd]`; payload shapes decoded for `SetMotorSpeed` (`[motorType:u8][speed:f32]`), `SetTargetTrack`, `SetDeviceMode`, and the bare 4-byte query heads (battery `09 00 00 02`, charge `09 00 00 06`, motor speed `09 03 01 13`, …). Our commit `[09 01 01 01]` is confirmed as `SET_DEVICE_MODE`. This breaks open the V2Head gate that blocked TODO #138–#141 implementation (TODO #149 folds it into the map doc).
- **Startup permission warning**: a PIXY present in sysfs but denied in `/dev` (missing udev rules, wrong group) now produces an actionable warning naming the udev fix at daemon startup, instead of a cryptic "Permission denied" on the first HID command (`warnInaccessibleDevices` in `probe.go`). The same hint now fires when the device appears on the hotplug path, rate-limited per node.
- **PTZ motor speed over HID (`speed` command, TODO #138)**: the first new HID command family built on the extracted official V2 protocol — `speed <pan|tilt|zoom> <value>` sends the official `SetMotorSpeed` single-report command (head `09 63 01 03` + `[motorType:u8][speed:f32]`) through the PIXY simulator's byte-faithful validation, with CLI dispatch, `POST /api/speed/{axis}`, and three web-UI speed sliders. Values are sanity-bounded (0–10000) and transmitted verbatim; the physical unit is pending hardware verification.
- **Camera model in the web panel and Waybar**: the detected model (PIXY vs PIXY 2K) shows as a footer badge in the web UI, in the Waybar tooltip, and as an additive `model` Waybar JSON field.
- **V2Head protocol vocabulary** (`internal/pixy/v2head.go`): typed command heads, motor types, and payload builders shared by the daemon and the simulator.

### Fixed

- Nothing yet.

## [0.4.0] - 2026-09-17

### Added

- **Model-aware probing and output**: `probeResult` carries the detected model (`pixy.Model`: `PIXY` / `PIXY 2K`, derived from the USB product ID via `pixy.ModelFromProductID`), `probeDevices()` logs it on every find, and both the `device` CLI command (`/dev/video0 /dev/hidraw7 PIXY 2K`) and the web status (`webStatus.Model`) surface it — so issue reports and logs say which hardware was actually found.
- **Device-reappear reconcile (`TODO_LIST.md` #137, issue #6 follow-up)**: when the camera (re)appears — daemon startup or USB hotplug — the daemon now reconciles belief and hardware instead of blindly adopting. Fresh installs (no state file) adopt the hardware's camera mode, so a first run reports the truth instead of assuming privacy with the lens open. Once a state file exists, the persisted camera mode is treated as user intent and re-asserted onto hardware whenever they differ (power cycles and replugs reset the camera to its boot default), so privacy mode physically re-blocks the lens after a power cut and manual modes survive reboots. Audio and gesture settings still adopt from hardware. Every failure path logs and keeps the current belief. Pinned by `reconcile_test.go` (fresh adopt, re-assert, no-op write-free, query-failure) plus `auto=off` persistence tests (3-tick survival + state.json round-trip).
- **EMEET PIXY 2K support (`328f:0118`)**: device probing (`isPixyProductID()` in `probe.go`) and the NixOS udev rules (`ATTRS{idProduct}=="00c0|0118"`, valid alternation since systemd 217) now accept the 2K variant next to the original `00c0` — no more patching `probe.go` locally. Reported working identically over HID/V4L2 in issue #6. Probe tests cover the 2K product ID for video4linux, hidraw, and uevent parsing; README + website docs updated.
- **"Manual Control" documentation** (issue #6): README and the website auto-modes guide now spell out that `auto = off` disables the `/proc` call-detection monitor entirely — manual `track`/`idle`/`privacy` persist until changed and no app needs to hold the camera open (the headless/Home Assistant snapshot-ready setup). `TestAutoManage_AutoOff_NoAction` now asserts `isCameraInUse` is never invoked, pinning the guarantee.
- **Website CI build workflow** (`website.yml`): corepack pnpm install with `--frozen-lockfile` (doubling as the lockfile-drift guard), `astro build` + CSP patch, and an assertion that ≥19 pages were produced — runs on every `website/**` change so MDX/build breakage is caught in CI instead of at manual deploy time (workflow run 35233004615 green).
- **Website relaunch** (`emeet-pixyd.lars.software`): landing pitch rewritten around the origin story — hero badge "Reverse-engineered for Linux", H1 "A great AI webcam, dumb on Linux. Until now.", metrics row led by ±150° pan + MIT license. New `WhySection` (three story cards: the hardware / the problem / the fix) and `ShowcaseSection` embedding a 25s HyperFrames-rendered demo video (`/demo.mp4`, 1920×1080) plus three real web-UI screenshots captured headless against the running daemon. New `camera` Lucide icon. Site title/description updated to match. `firebase.json` now long-caches `mp4`/`webm`/`mov` and serves HTML as `max-age=0, must-revalidate` via a catch-all header rule. The HTTPS custom-domain certificate reached `CERT_ACTIVE` (Google Trust Services) — the ACME TXT record staged in terraform turned out to be unnecessary (CNAME validation sufficed). Screenshots show the offline UI state; retake with the camera connected is tracked as `TODO_LIST.md` #129.
- **Website docs retrofit** (`emeet-pixyd.lars.software`): added curated "Where to go next" sections to all 17 docs pages, converted the installation blockquote to a Starlight callout, expanded the `related-tools` comparison matrix, enabled `lastUpdated` + `editLink` in Starlight, and added `.htmlvalidate.json` (0 HTML-validation errors). Deployed and live.
- **Error classification via `go-error-family` (v0.10.0)**: `errorfamily.go` registers 18 daemon sentinels + stdlib defaults into Infrastructure/Rejection/Transient families. `errorfamily.HTTPStatus(err)` derives HTTP status codes, `errorfamily.ExitCode(err)` derives BSD sysexits CLI exit codes, and `errorfamily.LogError()` adds family/code/retryable structured log fields at the daemon-init failure path — replacing per-call-site hardcoded status/exit codes. Scoped by design: HTMX action handlers keep returning 200+HTML toast (correct `outerHTML` swap), and the HID circuit breaker stays untouched. See `FEATURES.md` → Error Handling.
- **Gesture toggle in the web UI**: a Gesture Control toggle (posts to `/api/gesture`) joins the mode cards — gesture control was previously CLI/socket-only.
- **Web UI overhaul**: camera preview is now a full-width hero above the status panel (does not re-render on HTMX swaps). Camera mode cards (Track/Idle/Privacy) with inline SVG icons, descriptions, per-mode color glow, and keyboard-shortcut badges replace the old button group. Audio selector is a pill-style segmented control. PTZ radar indicator (120px circular position display with crosshair, zoom ring, and glowing position dot) added. Snapshot button overlays the preview. Keyboard shortcut legend (FAB + `?` key + `Escape`) lists all shortcuts. Preset UI (save input + chips with load/delete, delegated events that survive HTMX panel swaps). All UI icons are now inline SVG (Lucide-style) — no emoji anywhere. Responsive breakpoints at 860px/640px/400px + `prefers-reduced-motion` + `hover:none`.
- **Pre-commit lint gate**: `scripts/pre-commit` runs whole-module golangci-lint on staged `.go`/`.templ` changes, installed by the devShell `shellHook` (repo uses `core.hooksPath=.githooks`). Ends the "ungated auto-commit turns master red" failure class.
- Camera preset web UI: save/load/delete named PTZ positions via `/api/preset/{save,load,delete}/{name}` with preset count display (N/16).
- State schema versioning: `pixy.CurrentSchemaVersion = 1` + `State.SchemaVersion` field (`"v"` in JSON). `loadState` logs a warning on version mismatch and loads old files as version 0 (best-effort backward compatibility).
- Preset name validation: `pixy.ValidatePresetName()` — non-empty, ≤32 runes, no path separators, no control chars. Wired at both CLI and HTTP save boundaries (security/data-integrity gap closed).
- `pixy.PresetMap` domain type with `SortedNames()`, `Get()`, `IsFull()` methods. `State.Presets` is now typed. `MaxPresets` constant moved from package main to `internal/pixy`.
- Accessibility pass: WCAG 2.1 AA code-level audit with fixes (aria-labels, `aria-live` toasts, `aria-current` mode cards, focus-visible rings, placeholder contrast), SSE connection status indicator (green/amber/red dot wired to DataStar fetch events), offline banner, focus management across panel swaps, and preset-name `<datalist>` autocomplete. Screen-reader and mobile test matrices documented in `docs/accessibility-audit.md`.
- Test infrastructure: golden-file tests for the web panel and Waybar JSON, property-based tests for `Range.Clamp` + `ValidatePresetName`, auto-manage lifecycle integration tests, wpctl mock + PipeWire tests, `docs/hid-protocol.md` protocol documentation.

### Changed

- **BREAKING**: Replaced HTMX v2.0.9 with DataStar v1.0.2 (`datastar-go` SDK v1.2.2). All `hx-*` attributes converted to `data-*` attributes. Action handlers now return SSE patches (`PatchElementTempl`) instead of HTML fragments — DataStar morphs elements by ID automatically. Eliminated ~275 lines of custom JS (SSE bridge, HTMX lifecycle, focus preservation, PTZ helpers, toast rendering) from `app.js` (510→235 lines). Deleted 82 KB `htmx.js`, added 34 KB `datastar.js`. PTZ radar is now reactive via DataStar signals (`data-style` CSS custom properties). CSP updated to include `'unsafe-eval'` for DataStar expression evaluation. Audio endpoint changed from `POST /api/audio` (form value) to `POST /api/audio/{mode}` (path value).
- **BREAKING**: Removed `cqrs-htmx/v2` dependency entirely (~30 transitive dependencies eliminated, including private `go-cqrs-lite`). SSE broadcasting, middleware, and HTTP helpers reimplemented locally in `sse.go` (~290 lines). HTMX JS embedded directly in `static/htmx.js` instead of served via library handler.
- **BREAKING**: PTZ values are now always absolute by default. `emeet-pixyd tilt -90` sets tilt to -90° instead of "go -90° from current position". Relative mode requires an explicit `rel` prefix: `tilt rel-5`, `pan rel+10`.
- **BREAKING**: PTZ limits corrected to match hardware reality: pan ±150° (was ±170°), tilt ±90° (was ±30°), zoom 100-150× (was 100-400×). Verified empirically via `v4l2-ctl --list-ctrls`.
- **BREAKING**: PTZ limit constants replaced with `Range` struct type: `pixy.PanRange`, `pixy.TiltRange`, `pixy.ZoomRange` (was separate `PanMin`/`PanMax`/etc constants). Includes `Range.Clamp(v int) int` method.
- **BREAKING**: PTZ axis names are now a branded `pixy.Axis` type instead of raw strings. Prevents accidental substitution of arbitrary strings into axis-keyed maps and functions.
- `pixyCommit` now uses named constants instead of raw hex bytes. `respAutoModePrefix` extracted from 3 repetitions.
- `getWebStatus` PTZ initialization fixed: explicit `PTZValues{Pan:0, Tilt:0, Zoom:ZoomDefault}` (was zoom-only init leaving Pan/Tilt implicit).
- Duplicate `ci` devShell in `flake.nix` removed.
- `queryHIDState` errors now include the device path for debugging (was generic "queryHIDState: ...").
- Uevent channel send now uses `select` with `ctx.Done()` to prevent goroutine leak on shutdown.
- Subprocess calls (`v4l2-ctl`, `wpctl`, `notify-send`) now route through `CommandRunner` interface with centralized slog logging of command, args, and duration.
- PTZ cache updated with set values instead of invalidated after a successful PTZ set, providing immediate accurate readback (avoids stale hardware values while motor is still moving).
- All 9 GitHub Actions pinned to commit SHAs; govulncheck + pnpm audit clean (0 known vulns in Go and website deps).
- SSE live updates replace 3-second polling: `/api/events` endpoint with `Broadcaster` fan-out, DataStar `data-init` persistent connection, PTZ slider updates sent as ~20-byte `PatchSignals` instead of ~4KB panel HTML.
- Relative-time formatting ("last synced") switched to `go-humanize` (`humanize.Time`) — natural English output instead of abbreviated/HH:MM fallbacks.
- MJPEG streaming: explicit `WriteHeader` + `Flush` before frames (browser shows the stream immediately), write-deadline opt-out via `http.ResponseController` (`Unwrap()` on `statusRecorder`), ffmpeg stderr routed to debug logging.

### Fixed

- **Second master CI breakage (ungated dependency bumps)**: a parallel session bumped `go.mod` to go 1.27.1 (plus go-branded-id v0.6.0, go-error-family v0.10.1, prometheus bumps) and tightened `.golangci.yml` without re-running the gates, turning the `Lint` and `Nix` jobs red. Fixed: nix now pins go via `buildGoModule.override { go = go_1_27; }` (this nixpkgs revision reads the toolchain from the module parameter, not a derivation attr — the go-modules FOD ran 1.26.7 against a 1.27.1 go.mod), vendorHash refreshed in both `flake.nix` + `package.nix`, the obsolete go-branded-id committed-binary workaround removed (v0.6.0 ships no binary — closes the `TODO_LIST.md` #124 follow-up), the now-stable `GOEXPERIMENT=jsonv2` flag dropped everywhere, 18 lint findings resolved (16 stale `//nolint` directives after test-file exclusions widened, `gocognit` on `extractJPEGFrame` fixed by extracting `scanForSOI`, `modernize` embedlit rewrite), and the deprecated `exhaustruct` linter migrated to `exhaustruct_v5` (zero-value-first types listed under `ignore-patterns`).
- **First master CI breakage**: `ratelimit.go` (warn-rate-limiter, auto-committed) shipped with 9 golangci-lint failures (`exhaustruct`, `gochecknoglobals`, `unparam`, `paralleltest`, `wsl_v5`), turning the CI `Lint` job red on master. Fixed via targeted `//nolint` directives (repo convention), wsl_v5 whitespace, and a genuinely varied `newWarnLimiter` interval in tests. Lint back to 0 issues.
- **7 dependabot vulnerabilities in website build deps** (js-yaml, svgo, fast-uri — build-time only, static site): pinned patched versions via `pnpm-workspace.yaml` `overrides` (the previous top-level npm-style `overrides` key in `package.json` was silently ignored by pnpm); lockfile regenerated, site builds 19/19 pages with CSP intact.
- **3 genuine HTTP status bugs in `stream.go`**: `stream.not_supported`, `stream.pipe_error`, and `stream.start_error` returned 500 (Internal Server Error) for infrastructure failures; they now return 503 (Service Unavailable) via `errorfamily.HTTPStatus(err)`.
- **`nix build` FOD failure** from `go-branded-id@v0.5.0` shipping a committed compiled `namer` binary that embeds nix store paths (`/nix/store/.../go-1.26.5`): added an in-sandbox-only `replace` (`goBrandedSrc` + `replaceBrandedId` in `flake.nix`/`package.nix`) so the poisoned module is never downloaded. Committed `go.mod`/`go.sum` stay canonical (GitHub Actions `go test` against the real proxy is unaffected). **TEMPORARY** — until `go-branded-id` publishes a binary-free version (tracked as `TODO_LIST.md` #124; resolved in this release).
- **`nix flake check` failure** (broken since project inception): the go-modules FOD inherited `preBuild` with an empty-file validation guard that killed the FOD when `templ generate` produced empty output (no modules available). Simplified `preBuild` to bare `templ generate`; reordered `HOME=$TMPDIR` before `runHook preBuild`.
- **PIXY 2K master CI lint breakage**: the auto-committed warn-rate-limiter (9 findings) plus a duplicate `dupword` on the 2K HID name — see the two CI breakage entries above for the full story; both repaired to 0 issues.
- Race condition in parallel tests: `newTestDaemon` previously shared a fixed state directory under `/tmp`, causing data races when tests ran with `-race -count=N`. Now takes `testing.TB` and uses `t.TempDir()` so each test gets an isolated state file. Verified with `-race -count=10`.
- `makezero` linter false-positives on every `make([]byte, N)` I/O buffer. Set `always: false` (default mode) — still catches the real bug (`make([]T, n)` + `append`) without flagging legitimate pre-sized buffer allocations across 9 call sites (hid, socket, uevent, stream, cache, ipc).
- Empty `templates_templ.go` flake: `templ generate` can intermittently emit a zero-byte file, breaking the build. Root cause not fully resolved (tracked as TODO); CI now regenerates after checkout.
- Dead `SSEEvent.ID` and `SSEEvent.Retry` fields removed (never set in production — only in one test). Dead `writeSSEEvent` branches removed; `strconv` import dropped.
- Dead CSS removed: `.htmx-indicator` class was never used (project uses DataStar morphs).
- **Misleading zoom copy**: hero terminal and quick-start docs said `zoom 120 # Zoom to 120x` — zoom is a percentage (100–150), not a multiplier. Corrected to "Zoom to 120%" everywhere.
- Firebase deploy cache (`website/.firebase/`) was accidentally committed; removed from the index and gitignored.
- MJPEG streams died after 30s: the server-level `WriteTimeout` killed them; `setupStream` now clears the write deadline via `http.ResponseController`.
- Flaky PTZ tests that read real `/dev/video0` state via `parsePTZValues` — now use `withNoopParsePTZ()` stub for deterministic behavior.
- Socket bind failure root cause: `ProtectSystem=strict` in NixOS module made `/run/emeet-pixyd` read-only. Added `ReadWritePaths` to allow socket creation.
- `hidrawDevice.String()` receiver consistency (pointer, matching other methods).

### Dependencies

- Added `github.com/larsartmann/go-error-family` (now at v0.10.0; adopted at v0.8.0, bumped in `ca41926`). Direct require. `vendorHash` synced between `flake.nix` and `package.nix`.
- Added `github.com/starfederation/datastar-go` v1.2.2 (DataStar SDK; JS runtime v1.0.2 self-hosted at `static/datastar.js`).
- Added `github.com/dustin/go-humanize` (relative-time formatting in the web panel).
- `github.com/larsartmann/go-branded-id` at v0.5.0 during the FOD incident, v0.6.0 at release (the version that ships no committed binary — see the TEMPORARY nix workaround above, now removed).
- Updated `vendorHash` for Go 1.26.4 module cache. Required sync between `flake.nix` and `package.nix` (both share the hash under `proxyVendor = true`).

## [0.3.1] - 2026-06-12

Backfilled 2026-09-17 (the tag predates the changelog discipline for tags; reconstructed from `git log v0.3.0..v0.3.1`).

### Fixed

- `nix build` vendorHash refreshed for changed dependencies (`flake.nix` + `package.nix`).
- Version plumbing switched from git shortRev to semver `0.3.1` for stable `--version` output.
- `checks.test` added so `nix flake check` verifies tests; docs table alignment and YAML indentation normalized.

## [0.3.0] - 2026-05-21

Backfilled 2026-09-17 (reconstructed from `git log v0.3.0`).

### Added

- `Name()` methods on the branded ID types (`PID`, `SourceID`) for debug-visible IDs, plus `docs/DOMAIN_LANGUAGE.md`.
- Auto-tag GitHub workflow (`auto-tag.yml`) tagging from the nix version string.

### Fixed

- Improved error context for invalid audio-mode arguments and in the tracking/audio command handlers; multi-line argument formatting normalized across the codebase.

## [0.2.0] - 2026-06-07

> Note (added 2026-09-18): this version was never tagged — the tag list jumps from v0.3.0 (2026-05-21) to v0.3.1 (2026-06-12). The section was written on 2026-06-07 as a release-cut of accumulated work, which is why it is dated after v0.3.0 in this file.

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
