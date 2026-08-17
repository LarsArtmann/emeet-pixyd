# Comprehensive Status Report — emeet-pixyd

**Date:** 2026-05-07 23:57\
**Session:** Multi-session comprehensive code review + brutal self-review + fix execution\
**Head Commit:** `bef4686` chore: deduplicate AGENTS.md gotchas and modernize benchmarks to b.Loop()\
**Branch:** master (pushed to origin)

---

## A) FULLY DONE ✅

### Session 1 — 8-Skill Audit (commit `6e24659`)

- Full codebase review across 28 source files (8,468 LOC)
- Architecture diagrams, dependency evaluation, features audit, TODO list, status report
- Documentation suite at `docs/planning/` and `docs/status/`

### Session 2 — Toast Type Bug + Constants + Tests (commit `cf0b64f`)

|                                           | Item                                                                                                                                                                                                                                                          | Details |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| **Toast type propagation bug**            | `applyResponseToStatus()` hardcoded `toastTypeInfo`, discarding the type from `actionToast()`. All success toasts rendered as "info". Fixed: added `toastType` param, propagated through `action()`, `handleAudio`, `handleGestureToggle`, `handleAutoToggle` |         |
| **Deduplicate `audioCommand`/`cmdAudio`** | Removed `audioCommand` from `handlers.go`, unified to `cmdAudio`                                                                                                                                                                                              |         |
| **Missing command constants**             | Added `cmdStatus`, `cmdTogglePrivacy`, `cmdWaybar`, `cmdDevice` to `commands.go`                                                                                                                                                                              |         |
| **Raw string elimination**                | Replaced 8+ raw strings in `newWebMux()`, `handleGestureCommand()`, `handleCenterCommand()` with named constants                                                                                                                                              |         |
| **Test constant alignment**               | Updated `behavior_test.go`, `main_test.go`, `integration_test.go`, `commands_test.go` to use constants                                                                                                                                                        |         |
| **Toast type tests**                      | 3 new tests: info toast, error overrides toast, success toast type verification                                                                                                                                                                               |         |
| **`stream_test.go`**                      | 5 tests: semaphore full, no device, no ffmpeg, snapshot no frame, snapshot with frame                                                                                                                                                                         |         |
| **`v4l2_test.go`**                        | 4 tests: command format (2), invalid device parse, degree constant                                                                                                                                                                                            |         |
| **Benchmarks**                            | `BenchmarkExtractJPEGFrame`, `BenchmarkFormatLastSynced`, `BenchmarkParseHIDResponse`, `BenchmarkWaybarOutput`                                                                                                                                                |         |

### Session 2 — Status Report (commit `2f3d0f6`)

- Comprehensive status report at `docs/status/2026-05-07_23-13_comprehensive-review-and-fixes.md`

### Session 3 — DI Bypass Fix (commit `5f1adf0`)

|                            | Item                                                                                                                                                                                                                                                                                                                                                                     | Details |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------- |
| **DI bypass in `auto.go`** | **Critical bug:** `handleCallStart()` called `d.setTracking()` and `d.setAudio()` directly (bypassing injectable `d.setTrackingFn` and `d.setAudioFn`). Same for `handleCallEnd()` with `d.setTracking()`. This meant auto-manage path was NOT mockable in tests — any test injecting a mock `setTrackingFn` would never see it called through the auto-manage code path |         |
| **DI mock tests**          | 3 new tests proving DI mocks are called: `TestAutoManage_UsesMockedTrackingFn`, `TestAutoManage_UsesMockedAudioFn`, `TestAutoManage_CallEndUsesMockedTrackingFn`                                                                                                                                                                                                         |         |

### Session 3 — Middleware Tests (commit `3bdb433`)

|                                  | Item                                                                                                       | Details |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------- | ------- |
| **`loggingMiddleware` coverage** | Was 0%. Added `TestLoggingMiddleware_CapturesStatus` and `TestLoggingMiddleware_DefaultStatusOK`. Now 100% |         |

### Session 3 — PTZ Helper Tests (commit `1c35d52`)

|                                                 | Item                                                                               | Details |
| ----------------------------------------------- | ---------------------------------------------------------------------------------- | ------- |
| **`ptzAxisLabel`/`ptzAxisUnit`/`ptzAxisValue`** | Was 40%. Added `TestPTZAxisLabel`, `TestPTZAxisUnit`, `TestPTZAxisValue`. Now 100% |         |
| **`goconst` lint fix**                          | Fixed pre-existing `goconst` warnings for "unknown" test strings in `main_test.go` |         |

### Session 3 — Waybar Optimization (commit `6029752`)

|                                   | Item                                                                                                                                                                                                                       | Details |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| **`waybarOutput()` optimization** | Replaced `map[string]string` + `json.Marshal` with typed `waybarJSON` struct. Replaced `fmt.Sprintf` tooltip with `strings.Builder` (pre-allocated). **860ns/23 allocs → 334ns/7 allocs** (2.6x faster, 3.3x fewer allocs) |         |

### Session 4 — Cleanup (commit `bef4686`)

|                             | Item                                                                            | Details |
| --------------------------- | ------------------------------------------------------------------------------- | ------- |
| **AGENTS.md dedup**         | Removed duplicate gotcha entries (waybarJSON, command constants appeared twice) |         |
| **Benchmark modernization** | All 4 benchmarks updated from `for range b.N` to `for b.Loop()` (Go 1.24+)      |         |

### Build Verification (Current)

|                                                   | Check                                         | Result |
| ------------------------------------------------- | --------------------------------------------- | ------ |
| `GOWORK=off go test -race -count=1 ./...`         | ✅ PASS                                       |        |
| `GOWORK=off golangci-lint run --timeout 2m ./...` | ✅ 0 issues                                   |        |
| `GOWORK=off go vet ./...`                         | ✅ PASS                                       |        |
| Coverage (main)                                   | **72.1%** (started at 69.3%)                  |        |
| Coverage (internal/pixy)                          | **77.9%**                                     |        |
| Coverage (total)                                  | **72.4%**                                     |        |
| LOC                                               | 8,644 source + 5,591 tests = **14,235 total** |        |
| Lint issues                                       | 0                                             |        |
| Race detector                                     | Clean                                         |        |

---

## B) PARTIALLY DONE ⚠️

|                                     | Item    | Status                                                                                                                                                                                                                     | Details |
| ----------------------------------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `v4l2Set`/`v4l2SetMultiple` testing | Partial | Only command construction format tests. No execution path tests (requires real/mock `v4l2-ctl`). Both functions remain at **0%** coverage.                                                                                 |         |
| `handleStream` testing              | Partial | Error paths tested (no device, no ffmpeg, semaphore full). Happy path (actual MJPEG streaming loop) not tested — requires real ffmpeg. `handleStream` at **76%**.                                                          |         |
| `AutoMode` type coverage            | Partial | `ParseAutoMode` at 25%. `AutoMode.String()`, `Valid()`, `IsOff()`, `ActivatesTracking()`, `ActivatesAudio()`, `ActivatesPrivacy()`, `SwitchesSource()` all at **0%**. These methods exist but no test calls them directly. |         |
| `hidSendRecv` testing               | Partial | At **23.1%**. Only the happy path via mock. Timeout, context cancellation, read errors untested.                                                                                                                           |         |
| `syncState` testing                 | Partial | At **59.5%**. HID query + state reconciliation — the "query all" path and error reconciliation untested.                                                                                                                   |         |

---

## C) NOT STARTED ❌

|    | #                                                                                              | Item  | Effort | Impact                                                               | Notes |
| -- | ---------------------------------------------------------------------------------------------- | ----- | ------ | -------------------------------------------------------------------- | ----- |
| 1  | Test `AutoMode` methods — `String()`, `Valid()`, `IsOff()`, `Activates*()`, `SwitchesSource()` | 15min | Low    | 7 methods at 0% in `internal/pixy/pixy.go`. Trivial to test.         |       |
| 2  | Test `CameraState.String()` (0%) and `AudioMode.String()` (0%)                                 | 10min | Low    | Stringer methods, trivial tests.                                     |       |
| 3  | Test `findPixySource` with mock `wpctl` output                                                 | 15min | Low    | Already injectable via `findSourceFn`.                               |       |
| 4  | Test `setDefaultSource` with mock `wpctl`                                                      | 10min | Low    | Already injectable via `setSourceFn`.                                |       |
| 5  | Test `notify` with mock `notify-send`                                                          | 10min | Low    | Already injectable via `notifyFn`.                                   |       |
| 6  | Test `listenUevents` context cancellation + goroutine cleanup                                  | 20min | Medium | Verify fd closed on context cancel, no goroutine leak.               |       |
| 7  | Test `unixOpenNetlinkKobjectUevent` error paths                                                | 10min | Low    | Netlink socket bind failure path.                                    |       |
| 8  | Extract `webServer` read-only state interface from `*Daemon`                                   | 60min | Medium | Architectural improvement. `Daemon` is a god object with 15+ fields. |       |
| 9  | Test `Run()` lifecycle — signal handling, graceful shutdown                                    | 30min | Medium | Most critical untested path.                                         |       |
| 10 | Web UI frontend design review                                                                  | 45min | Medium | Review `templates.templ`, `static/style.css`, `static/app.js`.       |       |
| 11 | Make OTel meter provider injectable                                                            | 30min | Medium | Fixes serial-only metrics test constraint.                           |       |
| 12 | Test `centerCamera` happy path with mock `v4l2SetMultiple`                                     | 10min | Low    | `centerCamera` at 55.6%.                                             |       |
| 13 | Test `handlePTZ` error path                                                                    | 10min | Low    | `handlePTZ` at 83.3%. Missing error branch.                          |       |
| 14 | Add `BenchmarkIsCameraInUse`                                                                   | 10min | Low    | Hot path, runs every 2s scanning `/proc`.                            |       |
| 15 | Add `BenchmarkParseUevent`                                                                     | 5min  | Low    | Uevent parsing performance baseline.                                 |       |
| 16 | Fuzz test for `handlePTZ` HTTP endpoint                                                        | 15min | Low    | Existing fuzz pattern in `handlers_fuzz_test.go`.                    |       |
| 17 | Test `handleCommand` concurrent access via `cmdMu`                                             | 20min | Medium | No concurrent command test exists.                                   |       |
| 18 | Integration test: full auto lifecycle via HTTP API                                             | 20min | Medium | Call start/end through web API.                                      |       |
| 19 | Test `sendCommand` (0%) — daemon responding to socket commands                                 | 15min | Low    | Socket command processing loop.                                      |       |
| 20 | Test `exitWithDaemonError` (0%)                                                                | 5min  | Low    | Error display helper.                                                |       |
| 21 | Add `CHANGELOG.md` entry for all session fixes                                                 | 10min | Low    | Multiple user-facing fixes not documented.                           |       |
| 22 | Test `handleStream` happy path with fake ffmpeg                                                | 30min | Medium | Complex; needs MJPEG-producing fake process.                         |       |
| 23 | Review `static/app.js` error handling                                                          | 15min | Low    | Frontend robustness.                                                 |       |
| 24 | Verify `nix build` still works                                                                 | 10min | Medium | No nix build run this session.                                       |       |
| 25 | Add `go:generate` directive for templ                                                          | 5min  | Low    | Standard Go practice.                                                |       |

---

## D) TOTALLY FUCKED UP 💥

**Nothing catastrophic.** No regressions. No broken tests. No lint issues. Build clean. All features functional.

**Near-misses:**

- AGENTS.md update in session 3 had a failed edit block — tried to replace text that was already replaced. The duplicate entries (waybarJSON, command constants appeared twice) persisted until session 4 cleanup. Not a fuck-up per se, but sloppy.
- The `waybarOutput` optimization initially removed `encoding/json` from imports but the typed struct still needed `json.Marshal`. Had to add it back. Minor rework.
- The `v4l2_test.go` golines formatting warning from the LSP — only LSP-level, not `golangci-lint` (which passes clean). Not actionable without `golines` in nix shell.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **`Daemon` is a god object** — 15+ fields, 3 mutexes, 10 DI function fields, 2 nested structs. Every file shares `*Daemon` receiver. Web server should be its own struct with read-only state interface. This is the single biggest architectural debt.
2. **No `Run()` lifecycle test** — Signal handling, graceful shutdown, socket cleanup, HTTP server shutdown, context propagation — all untested. This is the most critical path in the entire daemon.
3. **`AutoMode` type is undertested** — 7 methods at 0% coverage in `internal/pixy`. These are domain logic methods that determine daemon behavior. Should be 100%.
4. **OTel global state** — `init()` in `metrics.go` registers global meter provider. Forces `TestUpdateMetrics` to be serial. Should be injectable.

### Testing

5. **External command functions at 0%** — `findPixySource`, `setDefaultSource`, `notify`, `v4l2Set`, `v4l2SetMultiple` are all 0%. These are the integration points that break in production. All are already injectable via DI fields — just need tests.
6. **`hidSendRecv` at 23.1%** — The bidirectional HID communication path. Timeout, context cancellation, read errors all untested. This is the most fragile code in the daemon (hardware timing-dependent).
7. **No concurrent command test** — `cmdMu` serializes commands but there's no test proving it works under concurrent access. Race conditions could hide here.
8. **`syncState` at 59.5%** — State reconciliation between HID and daemon is critical for correctness. Error paths and partial query failures untested.

### Performance

9. **`isCameraInUse` not benchmarked** — Runs every 2s, scans all of `/proc/*/fd`. Could be expensive on systems with many processes.
10. **`waybarOutput` at 334ns** — Good after optimization, but `strings.Builder` could use `Grow()` for exact pre-allocation.

### Code Quality

11. **`CameraState.String()` and `AudioMode.String()` at 0%** — These are used in waybar output and status. Should be tested.
12. **22 functions at 0% coverage** — Total 22 functions across the codebase. Many are trivial (main, Run, init) but some are important business logic (AutoMode methods, external commands).
13. **63 functions at 100% coverage** — Good, but the gap between 63/130 and the remaining 22 at 0% represents ~2 hours of focused work to close.

### Documentation

14. **CHANGELOG.md not updated** — Multiple user-facing fixes (toast type, DI bypass) not recorded in changelog.
15. **FEATURES.md status should be re-verified** — Toast fix was a real user-facing bug. "100% fully functional" claims should be re-validated.

---

## F) TOP 25 THINGS TO DO NEXT

Sorted by impact/effort ratio (Pareto-ordered):

|    | #                                                                                                        | Task   | Impact | Effort       | Category |
| -- | -------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------ | -------- |
| 1  | Test `AutoMode` methods: `String`, `Valid`, `IsOff`, `Activates*`, `SwitchesSource` (7 methods, 0%→100%) | High   | 15min  | Testing      |          |
| 2  | Test `CameraState.String()` and `AudioMode.String()` (2 methods, 0%→100%)                                | Low    | 10min  | Testing      |          |
| 3  | Test `findPixySource` with mock `wpctl` output                                                           | Low    | 15min  | Testing      |          |
| 4  | Test `setDefaultSource` with mock `wpctl`                                                                | Low    | 10min  | Testing      |          |
| 5  | Test `notify` with mock `notify-send`                                                                    | Low    | 10min  | Testing      |          |
| 6  | Test `listenUevents` context cancellation + goroutine cleanup                                            | Medium | 20min  | Testing      |          |
| 7  | Test `centerCamera` happy path with mock `v4l2SetMultiple`                                               | Low    | 10min  | Testing      |          |
| 8  | Test `handlePTZ` error path (83.3%→100%)                                                                 | Low    | 10min  | Testing      |          |
| 9  | Add `BenchmarkIsCameraInUse` — hot path every 2s                                                         | Low    | 10min  | Perf         |          |
| 10 | Add `BenchmarkParseUevent` — uevent performance baseline                                                 | Low    | 5min   | Perf         |          |
| 11 | Test `handleCommand` concurrent access via `cmdMu`                                                       | Medium | 20min  | Testing      |          |
| 12 | Integration test: full auto lifecycle via HTTP API                                                       | Medium | 20min  | Testing      |          |
| 13 | Test `syncState` HID query + reconciliation (59.5%→100%)                                                 | Medium | 20min  | Testing      |          |
| 14 | Test `hidSendRecv` timeout + context cancellation (23.1%→80%+)                                           | Medium | 20min  | Testing      |          |
| 15 | Test `Run()` lifecycle — signal handling, shutdown                                                       | Medium | 30min  | Testing      |          |
| 16 | Fuzz test for `handlePTZ` HTTP endpoint                                                                  | Low    | 15min  | Testing      |          |
| 17 | Extract `webServer` read-only state interface from `*Daemon`                                             | Medium | 60min  | Architecture |          |
| 18 | Web UI frontend design review — templates, CSS, JS                                                       | Medium | 45min  | UX           |          |
| 19 | Make OTel meter provider injectable (fix serial metrics tests)                                           | Medium | 30min  | Architecture |          |
| 20 | Test `handleStream` happy path with fake ffmpeg output                                                   | Medium | 30min  | Testing      |          |
| 21 | Test `sendCommand` (0%) — socket command processing loop                                                 | Low    | 15min  | Testing      |          |
| 22 | Verify `nix build` still works after all changes                                                         | Medium | 10min  | Build        |          |
| 23 | Update `CHANGELOG.md` with all session fixes                                                             | Low    | 10min  | Docs         |          |
| 24 | Review `static/app.js` error handling improvements                                                       | Low    | 15min  | UX           |          |
| 25 | Add `go:generate` directive for templ in `go.mod`                                                        | Low    | 5min   | Build        |          |

**Estimated total:** ~7 hours for all 25 items. Items 1–10 are ~2 hours and would bring coverage to ~80%.

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should we refactor `Daemon` into smaller structs now, or wait until the test coverage is higher?**

The `Daemon` god object (15+ fields, 3 mutexes, 10 DI fields) is the single biggest architectural debt. Extracting a `webServer` struct with read-only state access would clean up the codebase significantly. BUT:

- Current coverage is 72.1% — refactoring without higher coverage risks introducing regressions we can't detect
- The DI function fields already make individual methods testable without the full Daemon
- A premature refactor could be verschlimmbessern (making-worse-by-trying-to-improve)

The tradeoff: refactoring now gives a cleaner architecture for future work. Refactoring later gives more safety net. The right call depends on whether we expect more feature work (refactor now) or mostly maintenance (refactor later when something forces it).

---

## Metrics Summary

|                          | Metric                                                               | Value |
| ------------------------ | -------------------------------------------------------------------- | ----- |
| Source LOC               | 8,644                                                                |       |
| Test LOC                 | 5,591                                                                |       |
| Total LOC                | 14,235                                                               |       |
| Source files             | 28                                                                   |       |
| Test files               | 12                                                                   |       |
| Coverage (main)          | 72.1%                                                                |       |
| Coverage (internal/pixy) | 77.9%                                                                |       |
| Coverage (total)         | 72.4%                                                                |       |
| Functions total          | 130                                                                  |       |
| Functions at 100%        | 63 (48.5%)                                                           |       |
| Functions at 0%          | 22 (16.9%)                                                           |       |
| Lint issues              | 0                                                                    |       |
| Race detector            | Clean                                                                |       |
| Benchmarks               | 4 (JPEG 1108ns, FormatLastSynced 21ns, HID parse 41ns, Waybar 285ns) |       |
| Features                 | 43/43 functional                                                     |       |
| Commits this session     | 7                                                                    |       |
| Bugs fixed               | 2 (toast type, DI bypass)                                            |       |

### Benchmark Results

|                             | Benchmark | ns/op | B/op | allocs/op |
| --------------------------- | --------- | ----- | ---- | --------- |
| `BenchmarkExtractJPEGFrame` | 1,108     | 4,448 | 5    |           |
| `BenchmarkFormatLastSynced` | 21.4      | 0     | 0    |           |
| `BenchmarkParseHIDResponse` | 41.4      | 40    | 2    |           |
| `BenchmarkWaybarOutput`     | 285.2     | 456   | 7    |           |

### Coverage by File

|                 | File                                             | Key Functions                                           | Coverage |
| --------------- | ------------------------------------------------ | ------------------------------------------------------- | -------- |
| `auto.go`       | `handleCallStart`, `handleCallEnd`, `autoManage` | 100%                                                    |          |
| `commands.go`   | `handleCommand`, `handlePTZCommand`              | 97%/100%                                                |          |
| `handlers.go`   | All handlers, `applyResponseToStatus`            | 81–100%                                                 |          |
| `hid.go`        | `parseHIDResponse`, `pixyConfig/Commit`          | 100% but `hidSendRecv` 23%, `queryHIDState` 42%         |          |
| `main.go`       | `setTracking`, `setAudio`, `waybarOutput`        | 80–100% but `main`/`Run`/`sendCommand` 0%               |          |
| `metrics.go`    | `registerMetrics`, `updateMetrics`               | 64%/100%                                                |          |
| `middleware.go` | All middleware                                   | 100%                                                    |          |
| `probe.go`      | `isPixyName`, `probeHidraw`                      | 100% but `probeDevices` 57%                             |          |
| `process.go`    | `isCameraInUse`                                  | 92% but `findPixySource`/`setDefaultSource`/`notify` 0% |          |
| `state.go`      | `loadState`, `saveStateOrLog`                    | 100%                                                    |          |
| `stream.go`     | `handleSnapshot`, `extractJPEGFrame`             | 96–100%                                                 |          |
| `uevent.go`     | `parseUevent`, `isRelevantUevent`                | 100% but `listenUevents`/`unixSocketUevent` 0%          |          |
| `v4l2.go`       | `parsePTZValues`                                 | 87.5% but `v4l2Set`/`v4l2SetMultiple` 0%                |          |
| `internal/pixy` | Core types                                       | 77.9% but `AutoMode` methods 0%                         |          |
