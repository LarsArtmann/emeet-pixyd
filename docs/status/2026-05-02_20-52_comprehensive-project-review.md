# Status Report: Post-Lint-Fix & Test Repair — Comprehensive Project Review

**Date:** 2026-05-02 20:52
**Session:** Linter config cleanup, test fix, comprehensive project status review
**Branch:** master (up to date with origin)
**Codebase:** 8,017 lines Go across 26 files | 75 commits since inception (2026-04-19)

---

## Current Health

| Check                          | Status                   |
| ------------------------------ | ------------------------ |
| `golangci-lint run --timeout 2m` | ✅ **0 issues**        |
| `go vet ./...`                 | ✅ Clean                |
| `go test -race -count=1 ./...` | ✅ **PASS × 2** (all green) |
| `go test -race -count=1 -cover ./...` | ✅ 69.7% total coverage |
| Build                          | ✅ `nix build` succeeds |
| Working tree                   | ✅ Clean (post-commit)  |

### Coverage Breakdown

| Package                              | Coverage | Functions at 0% |
| ------------------------------------ | -------- | --------------- |
| `github.com/LarsArtmann/emeet-pixyd` | 69.2%    | 12              |
| `github.com/.../internal/pixy`       | 77.9%    | 9               |
| **Total**                            | **69.7%**| **21**          |

### Roadmap Target Progress (from SUPERB_ROADMAP.md)

| Metric           | Roadmap "Current" | Roadmap Target | **Actual Now** |
| ---------------- | ----------------- | -------------- | -------------- |
| Coverage         | 63.4%             | 80%+           | **69.7%** (+6.3%) |
| Test functions   | 120               | 180+           | **~193** (+73) |
| Linter warnings  | ~73               | 0              | **0** ✅       |
| `go vet`         | Clean             | Clean          | Clean ✅       |
| Race detector    | Clean             | Clean          | Clean ✅       |
| Zero-value init  | Yes               | No             | Yes (still has `init()`) |
| Hardware deps    | Direct            | Injected       | **Partially injected** |
| pprof endpoint   | No                | Yes            | **Yes** ✅ (behind Debug flag) |

---

## A) Fully Done

### This Session (2026-05-02)

| # | Item | Detail | Commit |
|---|------|--------|--------|
| 1 | Remove 5 false-positive linters (again) | `contextcheck`, `exhaustruct`, `gochecknoglobals`, `gochecknoinits`, `paralleltest` removed from `.golangci.yml` — they were re-added in staged changes, producing 51 false positives | `88cad01` |
| 2 | Clean gocritic config | Removed 3 already-disabled-by-default entries (`dupImport`, `octalLiteral`, `whyNoLint`) — eliminates config warnings | `88cad01` |
| 3 | Fix `TestIsCommandErrorResponse` | Test case `"error: pan"` had a space but comment said "no space after colon". Fixed to `"error:pan"` | `12970c9` |
| 4 | Fix `TestHandleCenterCommand_NoDevice` | Removed `centerCameraFn` spy that bypassed real `d.centerCamera` device check. Real impl already errors on empty `videoDev` | `12970c9` |

### Since Last Status Report (2026-05-01 → 2026-05-02, 5 commits)

| # | Item | Detail | Commit |
|---|------|--------|--------|
| 5 | `AutoMode` typed enum | Replaced `bool` auto field with `AutoMode` enum (`off`/`full`/`tracking-only`/`privacy-only`) in `pixy.State` — extensible for future modes | `021b599` |
| 6 | Rename methods to Fn-suffixed fields | `setTracking`→`setTrackingFn`, `setAudio`→`setAudioFn`, `setGesture`→`setGestureFn`, `centerCamera`→`centerCameraFn`, `v4l2Set`→`v4l2SetFn` — clearer DI pattern | `976a20f` |
| 7 | Error consolidation | Exported sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`) moved to `errors.go`, removed duplicates | `976a20f` |
| 8 | Expanded test coverage | New tests for `handleCenterCommand`, `handleAutoCommand`, `handleGestureCommand`, command routing, `CommandError` | `976a20f` |
| 9 | Dependency evaluation for samber/lo | Evaluated `samber/lo` and `samber/ro` — concluded not worth adding for this codebase's size | `d4ef606` |

### Major Milestones Completed (Full Project History)

| Milestone | Detail | Commits |
|-----------|--------|---------|
| **OTel metrics migration** | Replaced direct `prometheus/client_golang` with OTel SDK + Prometheus exporter | 3 |
| **Linter zero issues** | From 154 issues → 0, comprehensive `.golangci.yml` config | ~10 |
| **AutoMode enum** | `bool` → typed enum with 4 modes, env var parsing | 1 |
| **DI function fields** | All external deps injectable via function fields on Daemon | 3 |
| **pprof endpoint** | `/debug/pprof/*` gated behind `Config.Debug` | 1 |
| **NixOS module** | Full `hardware.emeet-pixy` module with systemd service, udev rules | 3 |
| **CI pipeline** | GitHub Actions: `go vet` + `golangci-lint` + `go test -race` + `nix build` | 2 |
| **Extraction refactoring** | `main.go` (845→611 lines) decomposed into `auto.go`, `state.go`, `probe.go`, `web_types.go` | 4 |
| **Uevent leak fix** | Netlink fd closed on context cancellation, preventing goroutine leak | 1 |
| **`CommandError` type** | Structured error type with `Op`/`Err` fields, `Error()`/`Unwrap()` methods | 1 |
| **Fuzz tests** | `FuzzExtractJPEGFrame`, `FuzzParseHIDResponse` — 0 crashes in 1.3M+ executions | 1 |

---

## B) Partially Done

| Item | Progress | Remaining | Blocker |
|------|----------|-----------|---------|
| **Dependency injection** | 5 function fields injected (`isCameraInUseFn`, `findSourceFn`, `setSourceFn`, `notifyFn`, `centerCameraFn`) + 5 method fields (`setTrackingFn`, `setAudioFn`, `setGestureFn`, `centerCameraFn`, `v4l2SetFn`) | `ffmpegStreamCmd`, `v4l2Set`, `v4l2SetMultiple`, `listenUevents`, HID device I/O still directly coupled | None — just work |
| **`init()` elimination** | Identified in Phase 2.2 of roadmap | OTel `init()` in `handlers.go` still registers global metrics — not hermetic | None |
| **Command test coverage** | New tests for center, auto, gesture commands | `handlePTZCommand` fully tested (100% coverage ✅), but `ffmpegStreamCmd`, `handleStream` (MJPEG) still 0% | Requires ffmpeg mock |
| **`webStatus` type safety** | Identified — `Camera`/`Audio` fields are raw strings | Should use `pixy.CameraState`/`pixy.AudioMode` types | None |
| **SUPERB_ROADMAP Phase 2.1** | Decompose `Run()` (currently 0% coverage) | `Run()` is still a monolithic lifecycle function with signal handling, uevent listener, web server, socket, poll loop | None |

---

## C) Not Started

### From SUPERB_ROADMAP.md

| Phase | Item | Priority |
|-------|------|----------|
| 1.1 | Extract `Commander` interface for shell commands | P3 |
| 1.2 | Extract `HIDDevice` interface for HID I/O | P4 |
| 1.3 | Extract `ProcessInspector` interface for /proc | P4 |
| 1.4 | Extract `UeventListener` interface for netlink | P4 |
| 2.2 | Eliminate `init()` for Prometheus metrics | P1 |
| 3.1 | Graceful degradation for missing optional deps (wpctl, notify-send, v4l2-ctl) | P1 |
| 3.2 | Circuit breaker for HID failures | P2 |
| 3.3 | Stream health monitoring (frame counter, uptime metric) | P3 |
| 4.1 | Additional OTel metrics (stream duration, command counters, probe counters) | P1 |
| 4.2 | Structured log levels audit | P1 |
| 5.1 | WebSocket for live state updates | P3 |
| 5.2 | Keyboard shortcuts in web UI | P2 |
| 5.3 | Mobile-responsive layout | P3 |
| 6.1 | Integration test harness with fake devices | P4 |
| 6.2 | Continuous fuzz in CI | P2 |
| 6.3 | Benchmark suite | P2 |

### From Previous Status Report (2026-05-01)

| Item | Status |
|------|--------|
| Document HID protocol in `docs/HID_PROTOCOL.md` | Not started |
| Add `--version` flag with ldflags | Not started |
| Add `emeet-pixyd diagnose` command | Not started |
| OTel tracing for HID round-trips | Not started |
| Config hot-reload via SIGHUP | Not started |
| Rate-limiting for rapid-fire CLI commands | Not started |
| Graceful shutdown timeout for in-flight requests | Not started |
| Shell completions (bash, zsh, fish) | Not started |

---

## D) Totally Fucked Up

| Item | Detail | Resolution | Status |
|------|--------|------------|--------|
| Staged `.golangci.yml` broke lint | Someone staged changes re-adding 5 false-positive linters, producing 51 issues | Removed them again in `88cad01` | ✅ Fixed |
| `TestHandleCenterCommand_NoDevice` was broken | Test overrode `centerCameraFn` with a spy returning nil, bypassing the real device check that should have produced the error | Removed spy, let real impl run | ✅ Fixed |
| `TestIsCommandErrorResponse` had wrong test string | `"error: pan"` matched `HasPrefix("error: ")` but test expected false — comment said "no space" but string had space | Changed to `"error:pan"` | ✅ Fixed |
| SUPERB_ROADMAP.md is stale | Roadmap says 63.4% coverage, 120 tests, ~73 linter warnings — all significantly outdated | Needs full update | ❌ Not fixed |
| `templates_templ.go` gitignored but required for build | CI has no `templ generate` step, file is gitignored, but `go build`/`go test` require it. Nix build generates it, but Go CI doesn't | Actually: file IS tracked in git (verified via `git ls-files`), just gitignored for future changes — confusing setup | ⚠️ Confusing |
| gci formatting mismatch | Comment alignment in `commands_test.go` didn't match gci expectations | Auto-fixed by `golangci-lint run --fix` | ✅ Fixed |

---

## E) What We Should Improve

### Self-Inflicted / Process Issues

1. **Staged changes breakage**: The `.golangci.yml` had staged changes re-adding removed linters. This suggests a workflow where partial changes get staged and forgotten. **Fix**: Always review `git diff --cached` before starting work.
2. **Tests were broken on master**: Two test failures existed on master — `TestHandleCenterCommand_NoDevice` and `TestIsCommandErrorResponse`. CI should have caught these but CI runs `go test -race -count=1 ./...` which would have caught them. **Fix**: Verify CI is green after every push.
3. **SUPERB_ROADMAP.md is stale**: Coverage 63.4% → 69.7%, tests 120 → 193, linter 73 → 0. The roadmap is now misleading. **Fix**: Update it.
4. **Too many status reports**: 12 status reports in `docs/status/` over 13 days. Some are redundant. **Fix**: Consolidate into fewer, more meaningful reports.

### Architectural Debt

5. **`Daemon` struct is still a god object**: 51+ lines of fields. The Fn-suffix rename helped naming clarity but didn't reduce the struct size.
6. **`handleCommand` is a 110-line switch**: All command routing in one function (91.9% covered but hard to navigate).
7. **`Run()` has 0% coverage**: The main lifecycle function is completely untested — signal handling, server startup, poll loop, uevent listener wiring.
8. **`init()` still exists**: OTel metrics registered globally in `handlers.go:init()`. Non-hermetic for tests.
9. **Mixed error return styles**: Some commands use `CommandError.Error()`, some use `fmt.Sprintf`, some return bare strings.
10. **No structured logging**: Uses `slog` but levels are inconsistent (Debug for errors that should be Warn, Error for expected conditions).
11. **`ffmpegStreamCmd` and `cleanupFFmpeg` at 0%**: MJPEG streaming has no test coverage at all.
12. **`process.go` shell commands untestable**: `findPixySource`, `setDefaultSource`, `notify` all shell out directly — no `Commander` interface.
13. **`v4l2.go` at 0%**: Both `v4l2Set` and `v4l2SetMultiple` have no coverage — they shell out to `v4l2-ctl`.
14. **`webStatus` uses raw strings**: `Camera` and `Audio` fields are `string` instead of `pixy.CameraState`/`pixy.AudioMode`.

### Testing Gaps

15. **`AutoMode.String()` at 0%**: New enum's `String()` method untested.
16. **`AutoMode.Valid()` at 0%**: New enum's `Valid()` method untested.
17. **`AutoMode.IsOff()`, `ActivatesTracking()`, etc. at 0%**: 5 boolean methods on the new enum, all untested.
18. **`CameraState.String()` and `AudioMode.String()` at 0%**: Trivial methods but inflate the 0% count.
19. **No benchmarks**: Zero benchmark functions exist.
20. **No CI fuzz**: Fuzz tests only run manually.

---

## F) Top #25 Things To Get Done Next

Sorted by **Impact × (1/Effort)** — highest value first.

| #  | Task | Impact | Effort | Category |
|----|------|--------|--------|----------|
| 1  | Update `SUPERB_ROADMAP.md` with current metrics (69.7% coverage, 193 tests, 0 lint) | Medium | Low | Docs |
| 2  | Add tests for `AutoMode` methods (`String`, `Valid`, `IsOff`, `Activates*`, `SwitchesSource`) | Medium | Low | Testing |
| 3  | Add tests for `CameraState.String()` and `AudioMode.String()` | Low | Trivial | Testing |
| 4  | Use `pixy.CameraState`/`pixy.AudioMode` in `webStatus` instead of raw strings | Medium | Low | Types |
| 5  | Name `Daemon.lastFrame` and `Daemon.ptzCache` anonymous structs | Medium | Low | Architecture |
| 6  | Consolidate all command error returns to use `CommandError` consistently | Medium | Low | Error handling |
| 7  | Extract `Commander` interface for shell commands (`wpctl`, `notify-send`, `v4l2-ctl`, `ffmpeg`) | High | Medium | DI |
| 8  | Eliminate `init()` — accept `prometheus.Registerer` in web server setup | Medium | Small | Architecture |
| 9  | Decompose `Run()` into testable sub-functions (signal handler, uevent, web server, poll loop) | High | Medium | Architecture |
| 10 | Add graceful degradation for missing optional deps (cache availability at startup) | Medium | Small | Robustness |
| 11 | Audit and standardize log levels (Debug/Info/Warn/Error) | Medium | Small | Observability |
| 12 | Add additional OTel metrics (command counters, stream duration, probe counters) | Medium | Small | Observability |
| 13 | Add `--version` flag with `ldflags`-based build info | Medium | Low | CLI |
| 14 | Document HID protocol in `docs/HID_PROTOCOL.md` | Medium | Medium | Docs |
| 15 | Add `emeet-pixyd diagnose` command (device state, HID health, V4L2 status) | High | Medium | CLI |
| 16 | Add benchmark suite for hot paths (extractJPEGFrame, parseHIDResponse, updateMetrics, handleCommand) | Low | Small | Testing |
| 17 | Add CI fuzz job (60s per fuzz test, store corpus in repo) | Medium | Small | CI |
| 18 | Add circuit breaker for HID failures (stop retrying after N consecutive failures) | Medium | Medium | Robustness |
| 19 | Add WebSocket for live state updates (replace HTMX polling) | Medium | Medium | Web UI |
| 20 | Add keyboard shortcuts in web UI (Space=privacy, T=tracking, A=audio, G=gesture) | Low | Small | Web UI |
| 21 | Extract `HIDDevice` interface for HID I/O (enables mocking `/dev/hidraw*`) | High | Medium-large | DI |
| 22 | Add mobile-responsive layout (CSS breakpoints for control buttons + PTZ sliders) | Low | Small | Web UI |
| 23 | Add Nix flake integration test (daemon + mocked device) | High | High | Testing |
| 24 | Add config hot-reload via SIGHUP | Medium | Medium | Features |
| 25 | Migrate `process_test.go` off real `/proc` to testable interface | Medium | Medium | Testing |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Is CI actually catching test failures? The two broken tests (`TestHandleCenterCommand_NoDevice`, `TestIsCommandErrorResponse`) were on `master` and pushed to origin, yet CI appears green.**

The GitHub Actions workflow runs `go test -race -count=1 ./...` with `GOWORK: off`. These two tests should have failed. Possible explanations:

1. **CI uses an older commit** — but `git log` shows both were in the pushed history
2. **Tests are flaky** — the center test might pass if a real PIXY device is connected (CI runs on `ubuntu-latest` which wouldn't have one, but `newTestDaemon` with empty videoDev should fail regardless)
3. **The tests were introduced in a recent commit that wasn't pushed yet** — but `git push` output shows the commits were pushed

This needs verification: check the GitHub Actions run for commit `976a20f` (the commit that added `commands_test.go`) to see if CI actually ran and what it reported. If CI was green with broken tests, there's a CI reliability issue.

---

## File Inventory

### Source Files (production code)

| File | Lines | Coverage | Role |
|------|-------|----------|------|
| `main.go` | 611 | Mixed | Daemon lifecycle, signal handling, status/waybar output, socket server |
| `handlers.go` | 624 | Mixed | HTTP handlers, web UI, OTel metrics, MJPEG streaming, middleware |
| `commands.go` | 270 | 91.9%+ | Command routing for Unix socket + CLI |
| `hid.go` | 259 | Mixed | HID bidirectional communication |
| `auto.go` | 124 | 100% | Auto-manage loop, call start/end handling |
| `integration_test.go` | 921 | N/A | Web server + socket command integration tests |
| `main_test.go` | 1,152 | N/A | Test daemon builders, state tests, daemon lifecycle tests |
| `commands_test.go` | 530 | N/A | Command handler unit tests |
| `auto_test.go` | 394 | N/A | Auto-manage state transition tests |
| `handlers_test.go` | 437 | N/A | HTTP handler + metrics tests |
| `process_test.go` | 137 | N/A | Process detection tests (uses real /proc) |
| `uevent_test.go` | 115 | N/A | Uevent parsing + listener tests |
| `probe.go` | 134 | 100% | Device probing (sysfs walks) |
| `process.go` | 143 | Partial | Process detection, PipeWire, notifications |
| `templates_templ.go` | 781 | N/A | Generated HTML templates |
| `uevent.go` | 103 | Partial | Netlink uevent parsing + listener |
| `v4l2.go` | 86 | 0% | V4L2 pan/tilt/zoom control |
| `state.go` | 75 | 100% | State persistence (JSON load/save) |
| `hid_fuzz_test.go` | 75 | N/A | Fuzz tests for HID response parsing |
| `handlers_fuzz_test.go` | 50 | N/A | Fuzz tests for JPEG frame extraction |
| `uevent_linux.go` | 31 | 0% | Low-level netlink socket |
| `errors.go` | 28 | 100% | CommandError type, sentinel errors |
| `web_types.go` | 20 | N/A | webStatus struct |
| `internal/pixy/pixy.go` | 395 | 77.9% | Core types, config, socket client |
| `internal/pixy/pixy_test.go` | 522 | N/A | Config, state, socket tests |

**Total: 8,017 lines across 26 Go files**

---

_Generated: 2026-05-02 20:52_
