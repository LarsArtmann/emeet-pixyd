# emeet-pixyd — Comprehensive Status Report

**Generated:** 2026-05-01 02:42\
**Branch:** `master` (clean, up to date with `origin/master`)\
**Latest commit:** `b40c951` feat: add pprof endpoints and CommandError type

---

## Current Metrics

| Metric                                     | Value                                            |
| ------------------------------------------ | ------------------------------------------------ |
| Total source lines (excl. tests/generated) | ~2,706                                           |
| Total test lines                           | ~3,098                                           |
| Test-to-source ratio                       | 1.14:1                                           |
| Test functions                             | 112                                              |
| Coverage (main pkg)                        | 63.0%                                            |
| Coverage (pixy pkg)                        | 89.7%                                            |
| Coverage (total)                           | 63.9%                                            |
| 0%-coverage functions                      | 18                                               |
| `go vet`                                   | Clean                                            |
| Race detector                              | Clean                                            |
| `golangci-lint run`                        | 0 issues                                         |
| Fuzz crashes                               | 0 / 1.3M+ execs                                  |
| Commits since last report                  | 3 (`c600362`, `b40c951`, plus push of `2d1461e`) |

---

## a) FULLY DONE

| Item                                                                    | Commit    | Date       |
| ----------------------------------------------------------------------- | --------- | ---------- |
| Close response bodies + context-aware HTTP in tests                     | `c600362` | 2026-04-30 |
| Add `/debug/pprof/*` endpoints (index, cmdline, profile, symbol, trace) | `b40c951` | 2026-05-01 |
| Add `CommandError` type with `Unwrap()` for structured error handling   | `b40c951` | 2026-05-01 |
| Add sentinel errors (`ErrAudioSourceNotFound`, `ErrInvalidValue`)       | `b40c951` | 2026-05-01 |
| Add `t.Parallel()` to `TestUpdateMetrics`                               | `b40c951` | 2026-05-01 |
| Convert all 5 command error paths to use `CommandError`                 | `b40c951` | 2026-05-01 |
| OTel metrics migration (completed in prior session)                     | `9db2b3d` | 2026-04-30 |
| golangci-lint 0 issues (stable since `c600362`)                         | `c600362` | 2026-04-30 |
| Consolidated PTZ error-path tests + shared helpers                      | `0754bda` | 2026-04-30 |

---

## b) PARTIALLY DONE

| Item                                               | Status                                  | What Remains                                                                                              |
| -------------------------------------------------- | --------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `CommandError` adoption                            | 5 of ~8 error-returning paths converted | `syncState`, `setDeviceState`, socket handler, and HTTP error paths still use `fmt.Sprintf("error: ...")` |
| `t.Parallel()` in integration tests                | 1 test fixed (`TestUpdateMetrics`)      | ~50 integration tests still missing `t.Parallel()` — blocked by CRLF line endings and shared daemon state |
| Roadmap Phase 2.2 (eliminate `init()` for metrics) | `sync.Once` replaces `init()`           | Still uses global `promExporter` + `metricInCall` etc. — need to move into `webServer` struct             |

---

## c) NOT STARTED

| Item                                             | Priority | Roadmap Ref |
| ------------------------------------------------ | -------- | ----------- |
| Extract `Commander` interface for shell commands | P1       | Phase 1.1   |
| Extract `HIDDevice` interface for HID I/O        | P3       | Phase 1.2   |
| Extract `ProcessInspector` interface for /proc   | P3       | Phase 1.3   |
| Extract `UeventListener` interface               | P4       | Phase 1.4   |
| Decompose `Run()` (104 lines → <80)              | P1       | Phase 2.1   |
| Move global metrics into `webServer` struct      | P2       | Phase 2.2   |
| Graceful degradation for missing deps            | P1       | Phase 3.1   |
| Circuit breaker for HID failures                 | P2       | Phase 3.2   |
| Stream health monitoring                         | P3       | Phase 3.3   |
| WebSocket for live state updates                 | P3       | Phase 5.1   |
| Keyboard shortcuts in web UI                     | P2       | Phase 5.2   |
| Mobile-responsive layout                         | P3       | Phase 5.3   |
| Integration test harness with fake devices       | P4       | Phase 6.1   |
| Continuous fuzz in CI                            | P2       | Phase 6.2   |
| Benchmark suite                                  | P2       | Phase 6.3   |
| CI fuzz automation                               | P2       | Phase 6.2   |
| Additional Prometheus metrics                    | P1       | Phase 4.1   |
| Structured log levels audit                      | P1       | Phase 4.2   |
| `handleCommand` → dispatch table (cyclop=20)     | P2       | Phase 7     |

---

## d) TOTALLY FUCKED UP (Lessons Learned)

| What Happened                                          | Root Cause                                                                                                       | Impact                                 | Resolution                                                                                            |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------- | -------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Python script v1–v4 failures for adding `t.Parallel()` | Wrong regex (`\.` in raw strings), CRLF line endings breaking `rstrip('\n')`, sed mangling single-line functions | ~45 min wasted, 4 `git restore` cycles | Should have used Go tools or simple Go script. CRLF discovery was valuable but the approach was wrong |
| Commander interface refactor started but not finished  | Changed `process.go`/`v4l2.go` signatures without updating ALL callers → build broken for 4+ errors              | Build broken mid-refactor              | **Reverted to clean state.** Must change signatures AND all callers in ONE atomic step                |
| Stale task list from prior session                     | Previous summary listed pending steps that were already completed by commit `c600362`                            | Wasted time on already-done work       | **Always run `golangci-lint` first** before acting on stale context                                   |
| `CommandError.Ok` field name                           | Initially named `Op` in struct but used `Ok` in callers                                                          | 5 compiler errors                      | Fixed by renaming struct field to match usage                                                         |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`Commander` interface is the #1 blocker for testability** — `process.go` and `v4l2.go` call `exec.CommandContext` directly, making 8 functions (11% of codebase) untestable without hardware. This must be done in one atomic step: create interface + real implementation + add field to Daemon + update ALL callers + update tests.

2. **`handleCommand` cyclomatic complexity = 20** — At the `cyclop` threshold. A dispatch table (`map[string]handlerFunc`) would drop it to ~5 and make each command independently testable.

3. **Global metrics state** — `promExporter`, `metricInCall`, `metricAutoMode`, `metricCameraState` are package-level vars with `sync.Once`. Moving them into `webServer` would make tests hermetic.

4. **`Daemon` struct has too many concerns** — It holds HID logic, V4L2 logic, process detection, HTTP server state, socket server state, debounce counters, and PTZ cache. Extracting interfaces (`Commander`, `HIDDevice`, `ProcessInspector`) would let us split these into testable units.

### Code Quality

5. **CRLF line endings in `integration_test.go`** — Silent source of bugs. Should normalize to LF via `.gitattributes` or `goimports`.

6. **`t.Parallel()` missing in ~50 integration tests** — LSP reports `paralleltest` warnings. Not in golangci-lint config, but adding it would speed up test suite.

7. **Error string matching in handlers** — `strings.HasPrefix(resp, "error:")` in `handlers.go:211` and `handlers.go:245`. With `CommandError` now available, callers should use `errors.As()` instead.

### Testing

8. **63.9% coverage cap** — 18 functions at 0% all require hardware (`/dev/hidraw*`, `/dev/video*`, `/proc`, netlink). `Commander` interface alone unlocks +8-10%.

9. **No benchmarks** — Hot paths (`extractJPEGFrame`, `parseHIDResponse`, `updateMetrics`) have no performance regression detection.

10. **`String()` methods at 0% coverage** — `CameraState.String()` and `AudioMode.String()` are trivial but uncovered.

### DevEx

11. **`golangci-lint fmt` is a footgun** — It re-enables all default linters and reformats `.golangci.yml`. Document this in AGENTS.md as a "NEVER RUN" warning.

12. **LSP diagnostics vs CLI diagnostics divergence** — LSP shows ~95 warnings (exhaustruct, contextcheck, noctx, paralleltest) that `golangci-lint run` doesn't report because these linters are excluded from `.golangci.yml`. This causes confusion.

---

## f) Top #25 Things to Do Next

Sorted by **impact / effort ratio** (highest first):

| #  | Task                                                                                          | Impact                          | Effort       | Category      |
| -- | --------------------------------------------------------------------------------------------- | ------------------------------- | ------------ | ------------- |
| 1  | Extract `Commander` interface + wire into `process.go`, `v4l2.go`, `handlers.go`              | HIGH (+8-10% coverage)          | Medium       | Architecture  |
| 2  | Fix CRLF → LF in `integration_test.go` via `.gitattributes`                                   | MEDIUM (prevent silent bugs)    | Trivial      | Code Quality  |
| 3  | Normalize `CommandError` usage in remaining paths (`syncState`, socket, HTTP)                 | MEDIUM (type safety)            | Small        | Code Quality  |
| 4  | Add `errors.As(CommandError)` checks in `handlers.go` replacing `HasPrefix("error:")`         | MEDIUM (type safety)            | Small        | Code Quality  |
| 5  | Add `t.Parallel()` to all integration tests                                                   | LOW (test speed)                | Small        | Testing       |
| 6  | Add `CameraState.String()` + `AudioMode.String()` tests                                       | LOW (+2% pixy coverage)         | Trivial      | Testing       |
| 7  | Add `CommandError` unit tests                                                                 | MEDIUM (error type safety)      | Trivial      | Testing       |
| 8  | Move global metrics into `webServer` struct                                                   | MEDIUM (hermetic tests)         | Small        | Architecture  |
| 9  | Add benchmark suite (4 benchmarks for hot paths)                                              | LOW (perf regression detection) | Small        | Testing       |
| 10 | Graceful degradation: cache `ffmpeg`/`wpctl`/`notify-send`/`v4l2-ctl` availability at startup | MEDIUM (cleaner logs)           | Small        | Robustness    |
| 11 | Structured log levels audit (Debug/Info/Warn/Error)                                           | MEDIUM (log noise reduction)    | Small        | Observability |
| 12 | Add `emeet_pixyd_command_total` + `emeet_pixyd_command_errors_total` counters                 | MEDIUM (production health)      | Small        | Observability |
| 13 | Refactor `handleCommand` → dispatch table (cyclop 20→5)                                       | MEDIUM (readability)            | Medium       | Architecture  |
| 14 | Decompose `Run()` into `startWebServer()`, `runPollLoop()`, `handleShutdown()`                | MEDIUM (testability)            | Small        | Architecture  |
| 15 | Extract `ProcessInspector` interface for `/proc` traversal                                    | MEDIUM (+3-5% coverage)         | Small        | Architecture  |
| 16 | Extract `UeventListener` interface for netlink                                                | LOW (+2-3% coverage)            | Small        | Architecture  |
| 17 | Add `.gitattributes` with `*.go text eol=lf` to prevent CRLF issues                           | LOW (prevention)                | Trivial      | DevEx         |
| 18 | Add CI fuzz job (60s per fuzz test)                                                           | MEDIUM (safety net)             | Small        | CI            |
| 19 | Add `emeet_pixyd_stream_duration_seconds` histogram                                           | LOW (stream observability)      | Small        | Observability |
| 20 | Circuit breaker for HID failures                                                              | MEDIUM (log spam prevention)    | Medium       | Robustness    |
| 21 | Keyboard shortcuts in web UI                                                                  | LOW (UX)                        | Small        | Web UI        |
| 22 | WebSocket for live state updates                                                              | MEDIUM (instant UI)             | Medium       | Web UI        |
| 23 | Mobile-responsive layout                                                                      | LOW (phone control)             | Small        | Web UI        |
| 24 | Integration test harness with fake devices                                                    | HIGH (E2E confidence)           | Large        | Testing       |
| 25 | Extract `HIDDevice` interface                                                                 | HIGH (+15-20% coverage)         | Medium-Large | Architecture  |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `Commander` interface live in `internal/pixy/` (shared package) or stay in `main`?**

Arguments for `internal/pixy/`:

- `pixy.SendCommand` already does socket I/O — a `Commander` interface in pixy would be consistent
- Test helpers in `internal/pixy/pixy_test.go` could use mock commanders
- Future: other consumers of the pixy package might need the interface

Arguments for `main`:

- `Commander` is only used by `process.go`, `v4l2.go`, `handlers.go` (all in `main`)
- Moving it to `pixy` creates a dependency cycle concern (pixy shouldn't depend on exec)
- The interface is an implementation detail of the daemon, not a shared abstraction

**My recommendation:** Keep in `main` for now. If `pixy.SendCommand` needs to be testable later, extract a `SocketCommander` interface in `internal/pixy/` separately. But I'd like your input on this.

---

## Session Timeline

| Time   | Action                                                                | Result                                            |
| ------ | --------------------------------------------------------------------- | ------------------------------------------------- |
| ~00:00 | Restored clean state from previous broken Commander refactor          | Build + tests + lint all clean                    |
| ~00:15 | Added `t.Parallel()` to `TestUpdateMetrics`                           | Build + tests OK                                  |
| ~00:20 | Added `/debug/pprof/*` routes to `handlers.go`                        | Build + tests + lint OK                           |
| ~00:30 | Created `errors.go` with `CommandError` type + sentinels              | Initial field name mismatch (`Op` vs `Ok`), fixed |
| ~00:35 | Converted 5 error paths in `commands.go` to `CommandError`            | Build + tests + lint OK                           |
| ~00:40 | Removed duplicate `ErrDeviceNotConnected` (already in `pixy` package) | Final state clean                                 |
| ~00:45 | Committed as `b40c951`, pushed to `origin/master`                     | All clean, tree clean                             |
