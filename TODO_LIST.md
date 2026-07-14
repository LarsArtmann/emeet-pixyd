# emeet-pixyd — TODO List

**Updated:** 2026-07-13 (Docs-health audit — open work extracted from 2026-07-04 self-review status report)
**Source docs verified:** all June–July 2026 `.md` + `.html` (status, planning, ADR, roadmap, design), cross-referenced against actual code

---

## Status Legend

- ✅ DONE — Verified in code
- 🔶 PARTIAL — Started but incomplete
- ⬜ TODO — Not started
- 🟢 DECIDED (won't-do) — Evaluated and rejected with rationale
- ❌ SKIP — Blocked by external constraints

---

## Open Work (2026-07-13)

Actionable items surfaced by the 2026-07-04 self-review status report
(`docs/status/2026-07-04_16-29_self-review-and-cleanup-status.md`). Ranked by
impact/effort. Vague or long-term ideas live in the "Low-value enhancements"
brainstorm section below and in `docs/SUPERB_ROADMAP.md`.

### UX / Accessibility

| #   | Status  | Task                                                                                       | Impact | Effort | Evidence             |
| --- | ------- | ------------------------------------------------------------------------------------------ | ------ | ------ | -------------------- |
| 106 | ⬜ TODO | Add `hx-on::after-swap` focus management for keyboard users (HTMX `outerHTML` loses focus) | MED    | LOW    | status report §e.8   |
| 107 | ⬜ TODO | Add SSE connection status indicator (green/red dot) in the UI                              | MED    | LOW    | status report §e.9   |
| 108 | ⬜ TODO | Add gesture toggle button to web UI mode cards (backend supports it, CLI works)            | MED    | LOW    | status report §e.10  |
| 109 | ⬜ TODO | Screen reader test pass (manual, document findings)                                        | MED    | LOW    | status report §f.3   |
| 110 | ⬜ TODO | WCAG 2.1 AA audit                                                                          | MED    | MED    | status report §f.15  |
| 111 | ⬜ TODO | Preset name autocomplete in web UI save input                                              | LOW    | MED    | status report §f.21  |
| 112 | ⬜ TODO | Mobile device testing pass (real phone/iPad/landscape)                                     | MED    | MED    | FEATURES.md (Mobile) |

### Architecture

| #   | Status  | Task                                                                    | Impact | Effort | Evidence                                        |
| --- | ------- | ----------------------------------------------------------------------- | ------ | ------ | ----------------------------------------------- |
| 113 | ⬜ TODO | Wire `errors.Is` checks for the 9 sentinel errors in production callers | MED    | LOW    | status report §e.1                              |
| 114 | ⬜ TODO | Consolidate `commandMsgError` into the `CommandError` pattern           | LOW    | MED    | status report §e.2                              |
| 115 | ⬜ TODO | Add `DisallowUnknownFields` to state JSON decoder (strict schema)       | LOW    | LOW    | status report §f.17                             |
| 116 | ⬜ TODO | Structured command types instead of `strings.Fields` string dispatch    | HIGH   | HIGH   | status report §e.4, §g (design decision needed) |

### Testing

| #   | Status  | Task                                                            | Impact | Effort | Evidence            |
| --- | ------- | --------------------------------------------------------------- | ------ | ------ | ------------------- |
| 117 | ⬜ TODO | Snapshot testing for web panel HTML (`go-snaps`)                | MED    | MED    | status report §f.8  |
| 118 | ⬜ TODO | Property-based tests for `ValidatePresetName` and `Range.Clamp` | LOW    | MED    | status report §f.9  |
| 119 | ⬜ TODO | `go-snaps` snapshot test for waybar JSON output                 | LOW    | LOW    | status report §f.18 |
| 120 | ⬜ TODO | Integration test: full auto-manage lifecycle with fake devices  | MED    | MED    | status report §f.20 |
| 121 | ⬜ TODO | Add `wpctl` mock for PipeWire integration tests                 | LOW    | MED    | status report §f.24 |

### Docs

| #   | Status  | Task                                               | Impact | Effort | Evidence            |
| --- | ------- | -------------------------------------------------- | ------ | ------ | ------------------- |
| 122 | ⬜ TODO | Document HID protocol reverse-engineering findings | LOW    | LOW    | status report §f.23 |

### Open design question

| #   | Status  | Question                                                                                                                                                                                                                          |
| --- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 123 | ❓ OPEN | How to handle multi-word preset names through CLI `strings.Fields` dispatch? Web UI works; CLI silently truncates. Options: quote support / join-remaining-parts / structured commands / accept limitation. See status report §g. |

---

## Resolved History (archive)

All items below are completed, decided, or skipped. Kept as a historical record
of resolved work. New open work is tracked in the "Open Work" section above.

## Phase 1: Quick Wins (P0)

| #   | Status  | Task                                                                                                            | Source        |
| --- | ------- | --------------------------------------------------------------------------------------------------------------- | ------------- |
| 1   | ✅ DONE | `.golangci.yml` centralized configuration                                                                       | Roadmap 2.3   |
| 2   | ✅ DONE | Fix linter suppressions (nlreturn, whitespace, goconst, perfsprint, modernize)                                  | Roadmap 7.1   |
| 3   | ✅ DONE | `CommandError` structured error type                                                                            | Roadmap 3.4   |
| 4   | ✅ DONE | `String()` method test coverage — `CameraState.String()`, `AudioMode.String()` exist but no explicit test calls | Roadmap 7.2   |
| 5   | ✅ DONE | `t.Parallel()` in all tests (only 2 justified serial tests)                                                     | Quality sweep |

## Phase 2: Decomposition (P1)

| #   | Status  | Task                                                                              | Source        |
| --- | ------- | --------------------------------------------------------------------------------- | ------------- |
| 6   | ✅ DONE | Decompose `Run()` into focused helpers                                            | Roadmap 2.1   |
| 7   | ✅ DONE | pprof endpoint gated behind `Config.Debug`                                        | Roadmap 4.3   |
| 8   | ✅ DONE | Keyboard shortcuts in web UI (T/I/P/C)                                            | Roadmap 5.2   |
| 9   | ✅ DONE | `AutoMode`/`DefaultAudio` from env vars                                           | Quality sweep |
| 10  | ✅ DONE | Uevent context cancellation (goroutine leak fix)                                  | Quality sweep |
| 11  | ✅ DONE | Device name matching shared `isPixyName()` helper                                 | Quality sweep |
| 12  | ✅ DONE | Error var consolidation (no duplicates)                                           | Quality sweep |
| 13  | ✅ DONE | Eliminate `init()` for Prometheus metrics — lazy registration via `sync.Once`     | Roadmap 2.2   |
| 14  | ✅ DONE | Structured log levels audit (standardize Debug/Info/Warn/Error usage)             | Roadmap 4.2   |
| 15  | ✅ DONE | Graceful degradation for missing optional deps (`checkExternalDeps()` at startup) | Roadmap 3.1   |

## Phase 3: Observability (P1-P2)

| #   | Status  | Task                                                                                                                  | Source      |
| --- | ------- | --------------------------------------------------------------------------------------------------------------------- | ----------- |
| 16  | ✅ DONE | Additional Prometheus metrics (command_total, probe_total, uevent_total, hid_failures, stream_duration, frames_total) | Roadmap 4.1 |
| 17  | ✅ DONE | Circuit breaker for HID failures (`hidFailCount` + `hidCircuitBreakerThreshold`)                                      | Roadmap 3.2 |
| 18  | ✅ DONE | Stream health monitoring (`metricStreamDuration` histogram + `metricFramesTotal` counter)                             | Roadmap 3.3 |
| 19  | ✅ DONE | Benchmark suite (7 benchmarks: JPEG, HID, Waybar, HandleCommand, GetWebStatus, FormatLastSynced)                      | Roadmap 6.3 |
| 20  | ✅ DONE | Continuous fuzz in CI (60s per test, store corpus, fail on crash)                                                     | Roadmap 6.2 |

## Phase 4: Architecture (P2-P3)

| #   | Status  | Task                                                                                                                                         | Source            |
| --- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- |
| 21  | ✅ DONE | Extract `CommandRunner` interface for shell commands (`commander.go`: Run/Output/LookPath; ffmpeg intentionally excluded — needs StdoutPipe) | Roadmap 1.1       |
| 22  | ✅ DONE | Extract `HIDDevice` interface for HID I/O (`Send`/`SendRecv` methods, `hidrawDevice` impl)                                                   | Roadmap 1.2       |
| 23  | ✅ DONE | Extract `ProcessInspector` interface for /proc traversal (`process.go`: `ProcessInspector` interface + `procInspector` impl)                 | Roadmap 1.3       |
| 24  | ✅ DONE | Extract `UeventListener` interface for netlink (`uevent.go`: `UeventListener` interface + `netlinkUeventListener` impl)                      | Roadmap 1.4       |
| 25  | ✅ DONE | `probeDevices()` — pure function returning `probeResult`, applied via `applyProbeResult`                                                     | Quality sweep 4.1 |

## Phase 5: Web UI (P2-P3)

| #   | Status  | Task                                                                | Source      |
| --- | ------- | ------------------------------------------------------------------- | ----------- |
| 26  | ✅ DONE | Mobile-responsive layout (720px + 480px breakpoints, touch targets) | Roadmap 5.3 |
| 27  | ✅ DONE | SSE for live state updates (replace 3s HTMX polling)                | Roadmap 5.1 |
| 28  | ✅ DONE | Keyboard shortcuts for PTZ (arrow keys, +/- for zoom)               | Status E.12 |
| 29  | ✅ DONE | PTZ relative mode (`pan+10`, `tilt-5`) via `parsePTZValue`          | Status E.8  |
| 30  | ✅ DONE | Camera preset support (save/recall PTZ positions)                   | Status F.9  |

## Phase 6: Testing (P3-P4)

| #   | Status  | Task                                                                      | Source      |
| --- | ------- | ------------------------------------------------------------------------- | ----------- |
| 31  | ✅ DONE | Integration test harness with fake devices (`fake_device_test.go`)        | Roadmap 6.1 |
| 32  | ✅ DONE | Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths | Status E.2  |
| 33  | ✅ DONE | Surface auto-manage errors to web UI (`autoError` field + `errors.Join`)  | Status E.3  |
| 34  | ✅ DONE | Improve MJPEG stream reconnection (exponential backoff + max-retry cap)   | Status E.4  |
| 35  | ✅ DONE | Integration test with real hardware (build tag guarded)                   | Status F.15 |

## Phase 7: Code Nits (from this review)

| #   | Status  | Task                                                                                  | Source     |
| --- | ------- | ------------------------------------------------------------------------------------- | ---------- |
| 36  | ✅ DONE | Extract toast type from `actionToast`, propagate through `applyResponseToStatus`      | Review L2  |
| 37  | ✅ DONE | Extract `lastFrame`/`ptzCache` into named types in `cache.go`                         | Review M1  |
| 38  | ✅ DONE | Moved `streamBufSize`/`ffmpegShutdownTimeout` constants from handlers.go to stream.go | Review L12 |
| 39  | ✅ DONE | Removed decorative blank lines in stream.go select/case blocks                        | Review M5  |
| 40  | ✅ DONE | Update `SUPERB_ROADMAP.md` — completion status table added, marked archived           | Review M4  |
| 41  | ✅ DONE | Consolidate PTZ axis dispatch into `ptzAxes` lookup table                             | Review     |
| 42  | ✅ DONE | PTZ readback accuracy — delayed hardware readback via `schedulePTZReadback` (500ms)   | Status E.1 |

## Phase 8: From 15-Skill Comprehensive Audit (2026-05-12)

|     | #       | Status                                                                                                         | Task            | Source |
| --- | ------- | -------------------------------------------------------------------------------------------------------------- | --------------- | ------ |
| 43  | ✅ DONE | Fix `hidSendRecv` nil error wrapping bug (zero-write produces `%!w(<nil>)`)                                    | Code Review C1  |
| 44  | ✅ DONE | Fix `hasPixyVendorProduct` — `return false` → `continue` on malformed HID_ID                                   | Code Review C4  |
| 45  | ✅ DONE | Fix `flake.nix` — remove invalid `env` attribute from app definition                                           | Nix Review      |
| 46  | ✅ DONE | Fix `package.nix` — deduplicate version string via `let version` binding                                       | Nix Review      |
| 47  | ✅ DONE | Fix `autoManage` — only call `saveState` when state actually changed                                           | Self-Review 4   |
| 48  | ✅ DONE | Validate loaded state in `loadState()` — reject garbage CameraState/AudioMode/AutoMode values                  | Code Review C2  |
| 49  | ✅ DONE | Fix `uevent.go` — transient read errors permanently disable hotplug, added retry with continue                 | Code Review C5  |
| 50  | ✅ DONE | Moved PTZ limits to shared constants in `internal/pixy/` (eliminated split brain with templates)               | Self-Review S1  |
| 51  | ✅ DONE | Consolidate 10 DI function pointers into `Dependencies` struct in `deps.go`                                    | Architecture 3  |
| 52  | ✅ DONE | Replace `handleCommand(string) string` with typed `CommandResult` struct                                       | Architecture 2  |
| 53  | ✅ DONE | Consolidate PTZ logic into `ptz.go` (extracted from handlers.go + v4l2.go, v4l2.go deleted)                    | Architecture 3  |
| 54  | ✅ DONE | Added systemd hardening to NixOS module (ProtectSystem, PrivateTmp, NoNewPrivileges, MemoryMax)                | Nix Review H2   |
| 55  | ✅ DONE | Fixed false-positive tests — proper assertions for sync/toggle-privacy commands                                | BDD Review P0   |
| 56  | ✅ DONE | Removed `, change` from PTZ slider hx-trigger (was doubling requests)                                          | Frontend Review |
| 57  | ✅ DONE | Suppress toast spam during PTZ slider drag (empty toast on success)                                            | Frontend Review |
| 58  | ✅ DONE | Added `role="alert"` to error banners for screen reader announcement                                           | Frontend A11y   |
| 59  | ✅ DONE | Migrated to `encoding/json/v2` via `GOEXPERIMENT=jsonv2` (5 files: http.go, state.go, waybar.go, 2 test files) | How-to-Go       |
| 60  | ✅ DONE | Added `extractJPEGFrame` max-iterations guard (10M) to prevent infinite loop on corrupt stream                 | Self-Review 4.8 |
| 62  | ✅ DONE | Enrich `PTZValues` with `Get(axis)`/`Set(axis, val)` methods, eliminate all hardcoded V4L2 control names       | Session 7       |

## Phase 9: Post-cqrs-htmx Removal Hardening (2026-06-21)

|     | #          | Status                                                                                                                                                                                                                                                                                                               | Task        | Source |
| --- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ------ |
| 63  | ✅ DONE    | Removed `cqrs-htmx/v2` dependency (~30 transitive deps eliminated)                                                                                                                                                                                                                                                   | Session 9   |
| 64  | ✅ DONE    | Reimplemented SSE `Broadcaster` + `sseStream` + middleware locally in `sse.go`/`http.go`                                                                                                                                                                                                                             | Session 9   |
| 65  | ✅ DONE    | Fixed nix build: updated `vendorHash` in `package.nix` + `flake.nix`                                                                                                                                                                                                                                                 | Session 9   |
| 66  | ✅ DONE    | Embedded HTMX v2.0.9 JS in `static/htmx.js` (was via `cqrshtmx.HTMXScriptHandler`)                                                                                                                                                                                                                                   | Session 9   |
| 67  | ✅ DONE    | Split `sse.go` → `sse.go` (SSE-only) + `http.go` (HTTP helpers + middleware)                                                                                                                                                                                                                                         | Session 9   |
| 68  | ✅ DONE    | Added 9 SSE unit tests (`writeSSEEvent`, `Broadcaster`, `splitSSELines`)                                                                                                                                                                                                                                             | Session 9   |
| 69  | ✅ DONE    | CI cleanup: removed `GOPRIVATE`, added `FuzzParsePTZValue`, added `nix flake check`                                                                                                                                                                                                                                  | Session 9   |
| 70  | ✅ DONE    | Updated CHANGELOG + AGENTS.md with cqrs-htmx removal docs                                                                                                                                                                                                                                                            | Session 9   |
| 71  | ✅ DONE    | `CommandRunner` interface wired (`commander.go`) — centralized subprocess logging via `realCommandRunner`, `noopCommandRunner` for tests. ffmpeg excluded by design (streaming API). NOTE: a broader "unified Commander facade" was mooted but the testability+logging goal is fully achieved — see `deps.go` wiring | Session 9   |
| 72  | ✅ DONE    | Define + wire `ProcessInspector` interface (`process.go`)                                                                                                                                                                                                                                                            | Roadmap 1.3 |
| 73  | ✅ DONE    | Define + wire `UeventListener` interface (`uevent.go`)                                                                                                                                                                                                                                                               | Roadmap 1.4 |
| 74  | ✅ DONE    | Camera preset support (CLI + web API + state.json persistence)                                                                                                                                                                                                                                                       | Status F.9  |
| 75  | ✅ DONE    | Mobile-responsive layout (720px + 480px breakpoints)                                                                                                                                                                                                                                                                 | Roadmap 5.3 |
| 76  | ✅ DONE    | Fake device test infrastructure (`fake_device_test.go`)                                                                                                                                                                                                                                                              | Roadmap 6.1 |
| 77  | ✅ DONE    | PTZ readback accuracy — delayed hardware readback (500ms)                                                                                                                                                                                                                                                            | Status E.1  |
| 78  | ✅ DONE    | MJPEG stream reconnection with backoff (max 10 retries, exponential)                                                                                                                                                                                                                                                 | Status E.4  |
| 79  | 🟢 DECIDED | `prometheus/client_golang` cannot be removed — OTel exporter depends on it transitively; `promhttp.Handler()` is required for `/metrics`                                                                                                                                                                             | Session 9   |
| 80  | 🟡 PARTIAL | Move `SSEEvent` + `toastType` to `internal/pixy` (domain types) — `toastType` already moved to `web_types.go`; `SSEEvent` kept in `sse.go` by design decision (transport-layer DTO, not domain)                                                                                                                      | Session 9   |
| 81  | ✅ DONE    | Integration test with real hardware (`integration_hardware_test.go`, `//go:build integration`)                                                                                                                                                                                                                       | Status F.15 |

---

## Phase 10: June 2026 Comprehensive Audit (verified against code)

All items below were extracted by reading **every `.md`/`.html` doc from June 2026**, then **verified against the actual code** on 2026-06-30. Stale items found during verification are noted. Build confirmed green: `go test -race -count=1 ./...` passes, `golangci-lint` reports 0 issues.

### Stale items corrected in this audit (were TODO, actually DONE)

| Was                                                                                        | Now        | Evidence                                                                                                        |
| ------------------------------------------------------------------------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------- |
| #21 / #71 Commander interface                                                              | ✅ DONE    | `commander.go`: `CommandRunner` interface (Run/Output/LookPath), wired in `deps.go`. ffmpeg excluded by design. |
| #80 toastType move                                                                         | 🟡 PARTIAL | `toastType` moved to `web_types.go`; `SSEEvent` kept in `sse.go` by design decision (transport DTO).            |
| `setSource`/`findSource` error surfacing (old session-10 net-new)                          | ✅ DONE    | `auto.go:44-55` appends source errors to `errs`, joins into `autoError`, surfaced in web UI (`handlers.go:95`). |
| `TestHandleStream_NoFFmpeg` hang on hardware (old session-11 bug)                          | ✅ FIXED   | `stream_test.go:66-86` accepts 503 or 200, bounded by 2s context timeout — no longer hangs.                     |
| Continuous fuzz in CI / `nix flake check` in CI / govulncheck in CI (old net-new CI items) | ✅ DONE    | `.github/workflows/go-test.yml`: nix flake check (L45), govulncheck (L35), 4 fuzz targets 60s each (L59).       |
| State-race in parallel tests (`newTestDaemon` shared `/tmp` state file)                    | ✅ DONE    | commit `241348f` — `newTestDaemon` takes `testing.TB`, uses `tb.TempDir()`. Stress-tested `-race -count=10`.    |

### Resolved — Architecture & testability

|     | #                     | Status                                                                                                        | Task                                   | Source / Evidence |
| --- | --------------------- | ------------------------------------------------------------------------------------------------------------- | -------------------------------------- | ----------------- |
| 82  | ✅ DONE               | Extract `ProcessInspector` interface for `/proc` traversal (`process.go`)                                     | Roadmap 1.3                            |
| 83  | ✅ DONE               | Extract `UeventListener` interface for netlink (`uevent.go`)                                                  | Roadmap 1.4                            |
| 84  | ✅ DONE               | Split `cmdMu` into `hidMu`+`v4l2Mu` — HID and V4L2 commands now run concurrently                              | Sessions 10-11; resolved Session 12    |
| 85  | 🟢 DECIDED (won't-do) | Move `main.go` → `cmd/emeet-pixyd/main.go` — contradicts single-binary architecture; no subcommands by design | Sessions 9-11; AGENTS.md §Architecture |
| 86  | 🟢 DECIDED (won't-do) | `Daemon` struct decomposition — 17 fields is manageable for a single-binary hardware daemon; accepted as-is   | Session 9; verified 17 fields          |
| 87  | ✅ DONE               | `PTZValues.Get(axis) int` → `(int, bool)` — unknown axes no longer silently return zero                       | Session 10; `pixy.go`                  |

### Resolved — Features & UX

|     | #       | Status                                                                                      | Task        | Source / Evidence |
| --- | ------- | ------------------------------------------------------------------------------------------- | ----------- | ----------------- |
| 88  | ✅ DONE | Camera preset support (CLI + web API + state.json persistence, max 16 presets)              | Roadmap     |
| 89  | ✅ DONE | Mobile-responsive layout (720px + 480px breakpoints, larger touch targets, stacked buttons) | Roadmap 5.3 |
| 90  | ✅ DONE | MJPEG stream reconnection (exponential backoff, max 10 retries, retry counter display)      | `app.js`    |

### Resolved — Quality & correctness

|     | #       | Status                                                                                                                           | Task                 | Source / Evidence |
| --- | ------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------- | ----------------- |
| 91  | ✅ DONE | PTZ readback accuracy — `schedulePTZReadback` does delayed (500ms) hardware readback to correct cache with actual motor position | Status E.1; `ptz.go` |
| 92  | ✅ DONE | Structured log-levels audit — uevent socket failures promoted Warn→Error, per-event uevent demoted Info→Debug                    | Roadmap 4.2          |

### Resolved — Testing

|     | #       | Status                                                                                                                   | Task                     | Source / Evidence |
| --- | ------- | ------------------------------------------------------------------------------------------------------------------------ | ------------------------ | ----------------- |
| 93  | ✅ DONE | Fake device test harness (`fake_device_test.go`: fakeHIDDevice, fakeProcInspector, fakeUeventListener, withFakeDevices)  | Roadmap 6.1              |
| 94  | ✅ DONE | Integration test with real hardware (`integration_hardware_test.go`, `//go:build integration`, 4 tests verified on PIXY) | Status F.15              |
| 95  | ✅ DONE | Coverage threshold enforcement in CI (70% minimum via `bc` comparison)                                                   | Session 9; `go-test.yml` |

### Resolved — Dependencies / modernization

|     | #                     | Status                                                                                                  | Task                        | Source / Evidence |
| --- | --------------------- | ------------------------------------------------------------------------------------------------------- | --------------------------- | ----------------- |
| 96  | 🟢 DECIDED (won't-do) | `prometheus/client_golang` cannot be removed — OTel exporter depends on it transitively                 | `go mod graph` verified     |
| 97  | ✅ DONE               | Migrated to `encoding/json/v2` via `GOEXPERIMENT=jsonv2` — enabled in nix build, lint, devShell, and CI | How-to-Go; verified working |
| 98  | 🟢 DECIDED (won't-do) | Move `SSEEvent` to `internal/pixy` — author decision: keep as transport DTO in `sse.go`                 | Session 9 §g                |

### Resolved — Build / CI hardening

|     | #       | Status                                                                                                         | Task              | Source / Evidence |
| --- | ------- | -------------------------------------------------------------------------------------------------------------- | ----------------- | ----------------- |
| 99  | ✅ DONE | Post-`templ generate` validation — detects empty `*_templ.go` with retry (CI + nix preBuild)                   | 2026-06-28 status |
| 100 | ✅ DONE | `templ` CLI version alignment — CI pins from `go.mod` instead of `@latest`                                     | 2026-06-20 status |
| 101 | ✅ DONE | `go-error-family` warning resolved — already absent from go.mod/go.sum (fixed by go-branded-id v0.3.1 upgrade) | Session 9         |

### Resolved — Docs / cleanup drift

|     | #       | Status                                                                                                                                         | Task                | Source / Evidence |
| --- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------- | ----------------- |
| 102 | ✅ DONE | `DESIGN.md` inaccuracies fixed — middleware described as local (not httputil), promhttp.Handler noted                                          | `DESIGN.md:107-108` |
| 103 | ✅ DONE | `README.md` source layout fixed — removed deleted `v4l2.go`, added `ptz.go`/`commander.go`/`deps.go`/`cache.go`/`waybar.go`/`sse.go`/`http.go` | `README.md:199`     |
| 104 | ✅ DONE | `CHANGELOG.md` `[Unreleased]` updated with makezero fix, vendorHash sync, state-race fix, templ regeneration note                              | `CHANGELOG.md`      |
| 105 | ✅ DONE | NixOS module nits fixed — `RestartSec=3` (integer not string), removed redundant `v4l-utils` from `systemPackages`                             | `modules/nixos.nix` |

### Low-value enhancements (brainstormed, consider only)

These are genuinely optional ideas surfaced across June status docs. Listed for completeness, not as committed work.

- Add `pan`/`tilt` to Waybar output (`waybar.go` currently only camera/audio/auto in tooltip)
- PTZ patrol/sweep mode (automatic periodic pan)
- `koanf` layered config (file + env, replacing env-only)
- OpenTelemetry tracing (not just metrics) — trace PTZ command latency
- SSE heartbeat (prevent proxy idle kills) + `LastEventID` replay after reconnect
- HTTP panic-recovery middleware
- Camera diagnostics endpoint (full V4L2 control dump)
- PTZ movement speed control; configurable home position
- Property-based test for `Range.Clamp` (gopter)
- Benchmark PTZ and auto-manage paths in CI

---

## Docs Verified

| File                                          | Status                                           |
| --------------------------------------------- | ------------------------------------------------ |
| AGENTS.md                                     | ✅ Current as of 2026-07-01                      |
| FEATURES.md                                   | ✅ Re-audited against code 2026-06-30            |
| docs/SUPERB_ROADMAP.md                        | ✅ Archived — completion status added 2026-06-05 |
| README.md                                     | ✅ Source layout fixed 2026-07-01                |
| CHANGELOG.md                                  | ✅ Updated with 2026-06-28/29 entries 2026-07-01 |
| DESIGN.md                                     | ✅ Inaccuracies fixed 2026-07-01                 |
| All June 2026 status/planning `.md` + `.html` | ✅ Read & cross-referenced in this audit         |

## Summary

> The "Open Work (2026-07-13)" section above supersedes the resolved history below. The history (items #1–#105) is complete; counts reflect the resolved archive only.

**Resolved history (items #1–#105):**

|                       | Status                                                                                   | Count |
| --------------------- | ---------------------------------------------------------------------------------------- | ----- |
| ✅ DONE               | 78                                                                                       |
| 🟡 PARTIAL            | 1 (#80)                                                                                  |
| 🟢 DECIDED (won't-do) | 4 (#85, #86, #96, #98)                                                                   |
| ❌ SKIP               | 1 (#97)                                                                                  |
| **Total resolved**    | 84 unique actionable (items #1–#105, with #61 omitted) + ~10 unnumbered brainstorm ideas |

**Open work (items #106–#123):** 17 actionable TODO items + 1 open design question, ranked in the "Open Work" section above.
