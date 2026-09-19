# emeet-pixyd

Auto-activation daemon for the EMEET PIXY dual-camera AI webcam (USB `328f:00c0`; PIXY 2K `328f:0118`). Linux-only, x86_64 + aarch64. Single binary: no arguments = start daemon; arguments = send a command to the running daemon via Unix socket.

Completed work lives in `CHANGELOG.md`; open work in `TODO_LIST.md`; feature inventory in `FEATURES.md`; decisions/raw ideas in `ROADMAP.md`; domain glossary in `docs/DOMAIN_LANGUAGE.md`. This file is **enduring context only** — no changelogs, no TODO status, no incident history.

## Commands

```bash
nix build                                   # production build (preferred)
GOWORK=off go test -race -count=1 ./...     # CI test command
GOWORK=off go test -run TestName ./...      # single test
GOWORK=off golangci-lint run --timeout 2m ./...  # CI lint (0 issues is the bar)
templ generate                              # REQUIRED after editing templates.templ
nix fmt                                     # alejandra for .nix
nix run                                     # run the daemon; emeet-pixyd status sends a socket command
go test -tags=integration ./...             # real-hardware HID/V4L2 tests
```

- **`GOWORK=off` is mandatory** — a parent `go.work` exists that does not include this project.
- **Go ≥ 1.27.1 required** (`go.mod`); the host toolchain may be older with `GOTOOLCHAIN=local` — run all Go tooling inside `nix develop` (devShell pins `go_1_27`).
- CI (`.github/workflows/`): `go-test.yml` (vet, templ generate, lint, govulncheck, race tests, fuzz targets + a list-vs-CI assert), `nix.yml` (`nix flake check --no-build` + build — do NOT duplicate into go-test.yml; a copy without the nix installer silently failed for months), `website.yml` (corepack pnpm, `--frozen-lockfile` drift guard, astro build + CSP patch, ≥19 pages, self-skipping deploy job). Releases are deliberately manual (annotated tag); `auto-tag.yml` is deleted.

## Architecture

```
main() → NewDaemon() → Run()
  ├── Unix socket listener (commands.go routing)
  ├── HTTP server (handlers.go → DataStar web UI)
  ├── Polling ticker (2s) → autoManage() → /proc scanning for call detection
  ├── Netlink uevent listener (hotplug detection)
  └── systemd sd_notify (READY=1, WATCHDOG=1)
```

| File                                      | Purpose                                                                                                                                                                                         |
| ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.go`                                 | `Daemon` struct, lifecycle (`Run` → `startHTTPServer` + `eventLoop` + `handleShutdown`), signals, `main()`                                                                                      |
| `commands.go`                             | Command routing (socket + CLI), named command/response constants, `handleQueryCommand`, `handleTogglePrivacy`                                                                                   |
| `handlers.go`                             | HTTP routing, web handlers, DataStar SSE rendering                                                                                                                                              |
| `ptz.go`                                  | PTZ logic: `ptzAxes` map, `parsePTZValue`, readback scheduling                                                                                                                                  |
| `metrics.go`                              | `daemonMetrics` struct, OTel registration (lazy `sync.Once`, no `init()` anywhere)                                                                                                              |
| `stream.go`                               | MJPEG streaming, snapshot, JPEG extraction, typed stream errors                                                                                                                                 |
| `http.go` / `middleware.go`               | HTTP helpers (`writeJSON`, `chain`, middleware implementations)                                                                                                                                 |
| `sse.go`                                  | `Broadcaster` (thread-safe fan-out); wire format handled by the DataStar SDK                                                                                                                    |
| `hid.go`                                  | HID config/query over hidraw; generic `queryHIDState[T]`                                                                                                                                        |
| `motor.go`                                | V2 single-report writers (`speed`, `preset push`) over hidraw, circuit-breaker accounting                                                                                                       |
| `identity.go`                             | Best-effort identity queries (`sn`/`ver`/`devver`/`func`) for the `device` command                                                                                                              |
| `device.go`                               | Device state mgmt, `reconcileOnDeviceAppear`, `getStatus`, `syncState`                                                                                                                          |
| `process.go`                              | `/proc/*/fd` call detection, PipeWire switching, notifications                                                                                                                                  |
| `uevent.go` / `uevent_linux.go`           | Netlink uevent listener (`UeventListener` interface)                                                                                                                                            |
| `auto.go`                                 | Auto-manage loop, debounce                                                                                                                                                                      |
| `state.go`                                | JSON state persistence (atomic tmp+rename, schema version `"v"`)                                                                                                                                |
| `probe.go`                                | Pure `probeDevices()` → `probeResult{VideoDev, HidrawDev, Model}`; `warnInaccessibleDevices`                                                                                                    |
| `errorfamily.go` / `errors.go`            | Sentinel classification (Infrastructure/Rejection/Transient), `CommandError`, `errorPrefix`                                                                                                     |
| `commander.go` / `deps.go`                | `CommandRunner` + `Dependencies` DI struct (function fields, noop defaults)                                                                                                                     |
| `waybar.go` / `web_types.go` / `cache.go` | Waybar JSON; typed `webStatus`; `lastFrameCache`/`ptzCache`                                                                                                                                     |
| `templates.templ`                         | DataStar UI (compiled via `templ generate`)                                                                                                                                                     |
| `internal/pixy/`                          | Shared domain types: `Config`, `State`, `CameraState`, `AudioMode`, `AutoMode`, `PID`, `SourceID`, `Axis`, `Range`, `PTZValues`, `PresetMap`, `Model`, `V2Head`, `MotorType`, `TargetTrackMode` |
| `tools/inno661/`, `tools/emhid/`          | EMEET STUDIO reverse-engineering artifacts (see Research section)                                                                                                                               |

### Key behaviors

- **HID protocol**: 9-byte config report + 4-byte commit report, 200ms sleep between. Responses are 64-byte reads parsed by byte position. Config-send failures trigger re-probe; only commit failures accumulate to the circuit breaker (threshold 3 → `ErrPIXYNotConnected`).
- **V2 HID command families** (`internal/pixy/v2head.go`, `motor.go`, `identity.go`): official single-report commands — head `[0x09, iface, category, cmd]`; motor sets route the iface byte to 0x63 (motor-MCU sub-device) and send NO commit pair. `speed`, `tracking`, `battery`, `preset push`, and the identity queries ride them; battery answers are TTL-cached (60s) and every surface degrades by omission. EVIDENCE GRADES (v2head.go comments carry them, map doc §3.5a is the reference): head bytes statically evidenced (Mac 2.0.3 cmdtable; 108/162 confirmed on Beta.25 x64 via `tools/emhid/extract_x64.py`); response framing DECODED + implemented (head-echo + reserved dword + payload@8 via shared `pixy.V2ResponsePayloadOffset`, min len 9); `TargetTrackMode` corrected to the official 1-based scheme (0=none, 1=face, 2=halfbody, 3=fullbody — numeric CLI aliases removed); `ChargeStatus` typed ({1,2}=charging). Still ASSUMED: `MotorType` 0/1/2, DefaultPosMode values, speed unit, preset slot count — TODO_LIST #166 pins them on hardware.
- **Call detection**: `/proc/*/fd` scan, excludes self + descendants, debounced (default 3 cycles). `auto = off` disables the monitor entirely (pinned by `TestAutoManage_AutoOff_NoAction`).
- **Device-reappear reconcile** (`reconcileOnDeviceAppear`): fresh install (no persisted state) adopts hardware; otherwise the persisted camera mode is re-asserted when hardware differs (power cycles reset the camera — privacy must survive), audio/gesture adopt from hardware. Every failure logs and keeps belief. Do NOT reintroduce plain adopt-on-appear.
- **PIXY 2K**: probing + udev rules accept both `00c0` and `0118`; `pixy.Model` flows through probe → logs → `device` cmd → `webStatus.Model`.
- **PTZ**: V4L2 uses 1/3600-degree units (`v4l2UnitsPerDegree = 3600`); user-facing degrees. Limits are hardware-verified: pan ±150°, tilt ±90° (positive = up everywhere), zoom 100–150 (percentage, not multiplier). Bare numbers are absolute (including negatives); relative requires `rel` prefix. `PTZValues.Get/Set(axis)` for axis-agnostic access; `Range.Clamp()`; `pixy.Axis` branded type.
- **Presets**: max 16 (`pixy.MaxPresets`), names validated by `pixy.ValidatePresetName`, persisted in `state.json` as `pixy.PresetMap`; `preset push` mirrors them into hardware slots (alphabetical mapping, assumed cap 8). Web UI handles multi-word names; CLI `strings.Fields` truncates at the first space — ADR written (`docs/adr/2026-09-18_multi-word-preset-names.md`), join-remaining lands after Lars approves.
- **State**: `{StateDir}/state.json`, atomic write; `loadState` validates enums, warns on schema-version mismatch, still loads best-effort. Persisted state wins over env defaults (env defaults apply only when no valid state file exists).
- **Error handling**: `go-error-family` classifies sentinels — `HTTPStatus(err)` for JSON endpoints, `ExitCode(err)` for CLI. Scoped by design: DataStar action handlers return 200 + SSE patch + toast (`applyResponseToStatus`, `actionToast` — toast type propagates, never hardcoded); the circuit breaker stays untouched. All command strings are named constants; errors use the `errorPrefix` constant.

### Concurrency model

- `d.mu` (RWMutex): state, devices, debounce counters. `hidMu`: serializes HID access (200ms sleep doesn't block V4L2). `v4l2Mu`: serializes v4l2-ctl. `streamSema` (cap 1): one MJPEG stream. `ptzCache`/`lastFrame`: own mutexes. `Broadcaster`: own internal lock, non-blocking sends.
- **Lock order where the two nest: `v4l2Mu` → `hidMu`, never the reverse** (PTZ move paths hold `v4l2Mu` and take `hidMu` in `reassertSpeeds`; the uevent path takes both explicitly). Pinned by `TestLockOrder_V4L2MoveWithHIDCommands` (deadlock watchdog).
- Pattern everywhere: acquire → copy → release → act on copies.

### Dependency injection

`NewDaemon()` wires real implementations; tests override via functional options. DI fields: `commander`, `procInspector`, `ueventListener`, `isCameraInUse`, `findSource`, `setSource`, `notify`, `setTracking`, `setAudio`, `setGesture`, `centerCamera`, `v4l2Set`, `parsePTZ`. Auto-manage paths call the `*Fn` fields, never the methods directly.

## Testing

- Standard `testing` only (no testify). `newTestDaemon(t, camera, videoDev, hidrawDev, opts...)` is the canonical builder — uses `t.TempDir()` (parallel-safe).
- **`newTestDaemon` wires REAL HID/V4L2 impls by default** — tests asserting side effects without hardware MUST inject noops (`withNoopTracking()`, `withNoopAudio()`, `withNoopV4L2()`, `withNoopParsePTZ()`), otherwise they pass on CI and fail on hardware-bearing machines.
- Options: `withInCall()`, `withAutoOff()`, `withCameraInUse()`, `withNotify*()`, `withFindSource()`, `withCapture*()`, `withDebounceCount()`, `withPixySimulator()` (returns the simulator + option).
- `pixySimulator` (`pixy_simulator_test.go`): protocol-faithful `HIDDevice` that validates every outgoing byte, enforces config→commit sequencing, generates round-trippable responses, and supports failure injection (`sendErr`, `commitErr` — the realistic breaker trigger, `sendRecvErr`, `nilResponse`, `corruptResp`). V2-head families live in `pixy_simulator_v2_test.go` + `pixyProtocolState` (payload validation, response builders — framed via `pixy.V2ResponsePayloadOffset`, failure-injection parity). Prefer it over `fakeHIDDevice` for HID-path tests; `v2Response(head, payload...)` in `test_helpers_test.go` builds framed responses.
- Notable pins: `TestTargetTrackModeWireValues` (enum bytes + no numeric aliases), `TestChargeStatusPredicate`, `TestWaybarBatteryClass`, `TestLockOrder_V4L2MoveWithHIDCommands`, `TestStateRoundTrip_SpeedsOmitZero_TrackModeNone` (on-disk JSON shape).
- `TestIntegration_BatteryProbe` (`integration_hardware_test.go`, `-tags=integration`) sends the exact read-only V2 query heads × iface bytes {3, 0x63} — run it with the PIXY attached; it never mutates state (no config+commit).
- Fuzz targets: `FuzzExtractJPEGFrame`, `FuzzParseHIDResponse`, `FuzzParsePTZValue`, `FuzzReadSignals`, `FuzzHandleConfigAndCommit`, `FuzzParseUevent` — CI asserts the list matches `go test -list '^Fuzz'`, so a stale entry fails the build.
- `t.Parallel()` everywhere except tests of global mutable metrics state (`TestUpdateMetrics` runs serially).
- Benchmarks (9): extract-JPEG, format-last-synced, parse-HID, waybar, handle-command ×2, web-status, broadcaster, simulator round-trip.

## DataStar UI patterns

SDK `datastar-go`, JS runtime self-hosted at `static/datastar.js` (ES module). CSP includes `'unsafe-eval'` (required by DataStar expression evaluation).

| Pattern           | Attribute                                                                                                                                                                          |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Action button     | `data-on:click="@post('/api/track')"`                                                                                                                                              |
| Loading state     | `data-indicator="loading"` + `data-class:btn-loading="$loading"` — the SHARED `$loading` signal is intentional (hardware ops are serialized; nested fetches stack via the counter) |
| Signal init       | `data-signals:pan="0"`                                                                                                                                                             |
| Two-way binding   | `data-bind="$pan"` (slider thumb reflects external changes)                                                                                                                        |
| Reactive text/CSS | `data-text="$pan + '°'"`, `data-style:--pan-x="..."`                                                                                                                               |
| Debounced input   | `data-on:input__debounce.300ms="@post('/api/ptz/pan')"`                                                                                                                            |
| Persistent SSE    | `data-init="@get('/api/events', {openWhenHidden: true})"` on `<body>`                                                                                                              |

Server side: `sse.PatchElementTempl(statusPanel(status))` morphs `#status-panel` by ID (idiomorph); PTZ slider updates use `sse.MarshalAndPatchSignals` (~20 bytes) with full-panel HTML only for errors; toasts via `sse.ExecuteScript("window.__showToast(...)")`; signals read via `datastar.ReadSignals`. SSE indicator + offline banner live OUTSIDE `#status-panel` so morphs don't reset them. Scripts load at end of `<body>` (`app.js` touches `document.body`). Server-rendered `style` attributes back the reactive `data-style` ones (FOUC prevention).

## Gotchas

- `GOWORK=off` for all Go tooling; go ≥ 1.27.1 via `nix develop`. The project `.crushrc` pins LSP toolchains (loads at crush startup only — mid-session `lsp_restart` won't re-read it). Remove it when the host NixOS ships go ≥ 1.27.
- **`vendorHash` is shared between `flake.nix` and `package.nix`** — keep both in sync on any dependency change. It hashes module tarballs downloaded via `proxyVendor = true` (needed because `templ generate` runs in `preBuild` before modules exist; the FOD doesn't compile). Nix pins the toolchain via `pkgs.buildGoModule.override { go = pkgs.go_1_27; }` — a derivation-attr `go =` is silently ignored.
- `pre-commit` lint gate: devShell `shellHook` symlinks `scripts/pre-commit` into `.githooks` (`core.hooksPath`); runs whole-module golangci-lint on staged `.go`/`.templ`.
- Generated `_templ.go` files are committed (nix build compatibility); CI still runs `templ generate`.
- Lint: ALL golangci-lint v2 linters enabled; false positives suppressed at call sites with `//nolint` (remove stale ones — test-file exclusions make many directives rot). Intentional config-level choices: gosec excludes (hardware daemon opens devices, launches subprocesses), `makezero always: false` (I/O buffers), `exhaustruct_v5` suppressions via `ignore-patterns` with FULL-STRING `packagePath.TypeName` regexes (not the old `exclude` key, not short names), `godoclint` excluded on `main.go` (generated-file cross-file false positive).
- Config is env-only (`pixy.ConfigFromEnv`): `EMEET_PIXYD_STATE_DIR` (`/run/emeet-pixyd`), `WEB_ADDR` (`127.0.0.1:8090`), `POLL_INTERVAL` (`2s`), `DEBOUNCE_COUNT` (`3`), `DEBUG`, `AUTO` (`full`; off/full/tracking-only/privacy-only), `DEFAULT_AUDIO` (`nc`). Env vars, not flags: `os.Args` is reserved for socket commands.
- Audio shorthand: CLI accepts `org`, stored value is `original` (`ParseAudioMode` maps both).
- `HIDDevice` embeds `fmt.Stringer` — every implementation must provide `String()` (error context).
- Branded IDs (`PID`, `SourceID`): `String()` includes the brand prefix (`"PID:42"`) — use `.Get()` for operational paths.
- NixOS module: `ProtectSystem=strict` makes `/run` read-only — `ReadWritePaths = ["/run/emeet-pixyd"]` is required for the socket. `hardware.emeet-pixy.{enable,user,auto,defaultAudio,debug}`.
- Overlay uses `self.packages.${system}.emeet-pixyd` (not `callPackage`) — the NixOS module's `mkPackageOption` depends on it.
- `webStatus` (in `web_types.go`) uses typed `pixy.*` fields; templates compare against typed constants, never raw strings.
- Uevent listener: context-cancellable, transient read errors `continue` (never `return`), fd closed on shutdown.
- `extractJPEGFrame` has a 10M-iteration guard; debounce counters cap at `DebounceCount`.
- Simulator V2 routing rule: motor SET heads collide with the commit-report shape, so V2 SETs are validated BEFORE the commit step, and the 0x63 iface substitution applies to motor heads ONLY (applying it to optics heads made `SetTargetTrack` collide with `SetMotorPos` — pinned by the preset-push tests).
- vmTest: `/etc/systemd/user` is a symlink `find` cannot descend — an unguarded `cat $(find …)` blocks on stdin; cat the canonical path.

## Research artifacts (EMEET STUDIO reverse engineering)

- Specimens live durably in `~/specimens/emeet-studio/` (2.0.0-Beta.25 Win installer + full extraction incl. the x64 `EMEET STUDIO 2.exe`); re-acquisition recipe for vendor-pulled files: **Wayback Machine CDX** (`https://web.archive.org/cdx/search/cdx?urlkey=<urlkey>*&output=json&filter=statuscode:200` then fetch `https://web.archive.org/web/<timestamp>id_/<url>` for the original bytes). The 2.0.3 Mac installer and the Beta.25 Mac pkg are NOT re-acquired (first 404; alternate snapshot forms untried) — nothing blocks on them.
- `docs/emeet-studio-official-app-comparison.md` — the deliverable (official Win EXE + Mac PKG vs emeet-pixyd; Inno 6.6.1 RE notes; Zoom call-detection differentiation).
- `docs/hid-protocol-official-map.md` — official `CMD_*` surface ↔ our `hid.go` (§3.5 = the 162-command table; §3.5a = framing + decoded enums with evidence grades; the version-shift model lives there: 2.0.3 = Beta.25 IDs + 1 after SET_REBOOT/GET_MOTOR_SPEED insertions — responses echo the request head, never cmd−1). Remaining gate: hardware verification, TODO_LIST #166.
- `tools/inno661/` — pure-Python Inno Setup 6.6.1 extractor (README = format spec); verified on both the 2.0.3 and Beta.25 Win installers; parse data for both committed under `data/`.
- `tools/emhid/` — `cmdtable.json` (162 Mac 2.0.3 command IDs, `extract_cmdtable.py`) and `extract_x64.py` → `x64_heads.json` (CRT-thunk sweep of the Beta.25 x64 build: 108/162 byte-for-byte matches, zero contradictions; regeneration from the specimen verified deterministic). Analysis scratch (`text.asm`, ~160 MB) is regenerable via `objdump -d -M intel -j .text` and does NOT belong in the repo.
- Failed approaches (do NOT retry): wine/wineWow installer runs (GUI crash), C++ innoextract patch (Python won), `lzma.FORMAT_ALONE` (use `FORMAT_RAW` after stripping 5 props bytes — Inno 6.6.1 blocks are independent LZMA1 streams), EMEETLINK 5.8.5 as a PIXY specimen (different product, zero motor surface).
- Method lesson: bind a log-string → named sender/handler FIRST, then read the surrounding code; pattern-hunting without a name produced the wrong cmd−1 hypothesis and several dead ends.

## Website

`website/` — Astro + Starlight + Tailwind v4, deployed to `emeet-pixyd.lars.software` (Firebase Hosting, project `lars-software`, target `emeet-pixyd`; cert active via CNAME validation). Accent violet `#8b5cf6`. Build: `nix shell nixpkgs#nodejs -c pnpm run build` (runs `astro build && node scripts/fix-csp.mjs`; Node 24, `.node-version`). Deploy: `firebase deploy --only hosting:emeet-pixyd --project lars-software`, or let the `website.yml` deploy job do it once `FIREBASE_SERVICE_ACCOUNT` exists (`FIREBASE_DEPLOY_SETUP.md` = setup checklist). Gotchas: `typescript` is pinned `~6.0.2` because `astro check` crashes on TS7 (`tsc --strict` is the strict gate); `pnpm-lock.yaml` must match `package.json` or the frozen-lockfile CI guard fails the website job; pre-deploy greps must grep for the NEW content, not the old; demo-video source lives in `website/emeet-pixy-demo/` (re-render: `HYPERFRAMES_BROWSER_PATH=<nixpkgs chromium> node node_modules/hyperframes/dist/cli.js render`, renders gitignored); screenshots via headless chromium against the live daemon (current shots show offline state, TODO #129).

## External libraries (adopted)

- `datastar-go` — SSE/UI framework (replaced HTMX + cqrs-htmx; all deps on the public proxy).
- `go-error-family` — error classification (HTTP status / exit codes / structured logs).
- `go-branded-id` — phantom-typed `PID`/`SourceID`.
- `templ` — typed HTML templates. `prometheus/client_golang` kept only for `promhttp` (OTel exporter depends on it; see ROADMAP won't-do). `go-humanize` — relative time in the panel.
- Rejected (see ROADMAP for rationale): templ-components (hand-crafted CSS, not Tailwind), kardianos/service (ADR: zero sd_notify value for a Linux-only daemon).
