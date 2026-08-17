# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-05-30 07:46\
**Branch:** `master` at `f8b5be4`\
**Session:** Stream `http.Flusher` fix + comprehensive status update

---

## Executive Summary

This session fixed a **production 500 error** on `/api/stream` caused by `loggingMiddleware` wrapping `http.ResponseWriter` in a type that did not implement `http.Flusher`. When the middleware chain was active (real production), `handleStream`'s `responseWriter.(http.Flusher)` assertion failed, returning "streaming not supported" (500). Tests never caught this because they bypass middleware and test `mux` directly. A regression test was added.

---

## A) FULLY DONE ✅

### This Session

| # | Item                                                     | Impact                                | Files              |
| - | -------------------------------------------------------- | ------------------------------------- | ------------------ |
| 1 | Add `Flush()` to `responseWriter` in `loggingMiddleware` | 🔴 Fixes production `/api/stream` 500 | `middleware.go`    |
| 2 | Regression test `TestLoggingMiddleware_Flusher`          | 🔴 Prevents regression                | `handlers_test.go` |

### Quality Metrics (Current)

| Metric                  | Value               | Status                                             |
| ----------------------- | ------------------- | -------------------------------------------------- |
| Build                   | ✅ Clean            | 0 errors                                           |
| Lint (golangci-lint v2) | ✅ 0 issues         | Clean                                              |
| Tests (race detector)   | ⚠️ 253 PASS / 4 FAIL | All 4 failures pre-existing (go-branded-id v0.3.0) |
| Fuzz tests              | ✅ 2 passing        | `FuzzExtractJPEGFrame`, `FuzzParseHIDResponse`     |
| Benchmarks              | ✅ 7 passing        | All green                                          |
| Source lines (non-test) | 4,263               | —                                                  |
| Test lines              | 6,057               | 1.42:1 test:source ratio                           |
| Test functions          | 257                 | —                                                  |
| Source files            | 19                  | —                                                  |
| Test files              | 14                  | —                                                  |
| Total Go files          | 33                  | —                                                  |

### Feature Delivery (44/44 — 100%)

All 44 features in `FEATURES.md` remain ✅ FULLY_FUNCTIONAL. Unchanged.

### TODO List Progress

| Status     | Count  | Percentage |
| ---------- | ------ | ---------- |
| ✅ DONE    | 34     | 55.7%      |
| 🔶 PARTIAL | 0      | 0%         |
| ❌ SKIP    | 1      | 1.6%       |
| ⬜ TODO    | 26     | 42.6%      |
| **Total**  | **61** | **100%**   |

_(+2 DONE from this session's fix and test; TODO count reduced from 28 to 26)_

---

## B) PARTIALLY DONE 🔶

**Nothing is partially done.**

---

## C) NOT STARTED ⬜

26 items remain in `TODO_LIST.md`. Key categories:

### Code Quality

- #14: Structured log levels audit
- #15: Graceful degradation for missing optional deps
- #40/#61: Update/archive `SUPERB_ROADMAP.md`

### Observability

- #16: Additional Prometheus metrics (stream, command counters, probe, uevent)
- #17: Circuit breaker for HID failures
- #18: Stream health monitoring
- #20: Continuous fuzz in CI

### Architecture (Higher effort)

- #21-#24: Extract interfaces (`Commander`, `HIDDevice`, `ProcessInspector`, `UeventListener`)
- #51: Consolidate 9 function pointers into `Dependencies` interface
- #52: Replace `handleCommand(string) string` with typed `CommandResult`
- #53: Consolidate PTZ logic into single `ptz.go`

### Web UI

- #26: Mobile-responsive layout
- #27: WebSocket for live state updates (replace 3s HTMX polling)
- #28: Keyboard shortcuts for PTZ (arrow keys, +/- zoom)
- #29: PTZ relative mode (`pan+10`, `tilt-5`)
- #30: Camera preset support

### Testing

- #31: Integration test harness with fake devices
- #32: Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths
- #33: Surface auto-manage errors to web UI
- #34: Improve MJPEG stream reconnection
- #35: Integration test with real hardware (build tag guarded)

### Other

- #42: PTZ readback accuracy (delay or in-memory "last set")

---

## D) TOTALLY FUCKED UP 💥

### Production Bug — NOW FIXED

**`GET /api/stream` returned 500** with "streaming not supported" in production. The root cause was that `loggingMiddleware` wraps the real `http.ResponseWriter` in a `responseWriter` struct that only implemented `WriteHeader`, not `http.Flusher`. `handleStream` at `stream.go:89` asserts `responseWriter.(http.Flusher)` — when the middleware chain is active (production), this assertion fails.

**Why tests never caught it:** All stream tests use `httptest.NewServer(mux)` directly, bypassing the middleware `Chain()`. The middleware is only applied in `newHTTPServer()` which wraps `mux` with `securityMiddleware`, `loggingMiddleware`, and `requestIDMiddleware`.

**Fix:** Added `Flush()` method to `responseWriter` in `middleware.go` that delegates to the underlying `ResponseWriter` if it implements `http.Flusher`.

### Pre-existing Test Failures (4 tests)

Caused by `go-branded-id` v0.3.0 changing `String()` output to include a typed prefix:

| Test                                     | Failure             | Expected | Got             |
| ---------------------------------------- | ------------------- | -------- | --------------- |
| `TestPpidOf_CurrentProcess`              | `PID.String()`      | `"42"`   | `"PID:42"`      |
| `TestNewPID`                             | `PID.String()`      | `"42"`   | `"PID:42"`      |
| `TestHandleCallStart_SetsPipeWireSource` | `SourceID.String()` | `42`     | `SourceID:42`   |
| `TestBehavior_FullAutoCallLifecycle`     | `SourceID.String()` | `[42]`   | `[SourceID:42]` |

These are **intentionally not fixed** — `nix` builds skip tests via `doCheck = false` for this exact reason. CI runs `go test` via GitHub Actions. The library author (same as project author) may revert or the tests need updating. Documented in `AGENTS.md`.

### No New Issues Introduced

- 0 build errors
- 0 lint issues
- Fix + regression test verified
- No data corruption risks
- No security vulnerabilities known
- No new dead code paths

---

## E) WHAT WE SHOULD IMPROVE!

### Immediate Priority

1. **Fix go-branded-id test failures** — Either update test expectations to match v0.3.0 `String()` format, or pin the dependency to pre-v0.3.0. Currently 4 tests fail on every run.

2. **Middleware-aware integration tests** — The stream tests (and other handler tests) should test through the full middleware chain, not just the bare `mux`. This would have caught the `Flusher` bug. Consider adding a `newTestServerWithMiddleware()` helper.

3. **Flaky test investigation** — Two tests exhibit intermittent failures under parallel execution (though they passed this run):
   - `TestHandleStream_NoFFmpeg` — port/timing collision with parallel stream tests
   - `TestSocket_StatusCommand` — ephemeral socket path race

### Architectural Improvements (Future Sessions)

Highest impact:

1. **`Dependencies` interface** (#51) — Replace 9 function pointers with compile-time-checked interface. Single biggest architectural win.
2. **Typed `CommandResult`** (#52) — Replace `handleCommand(string) string` with structured result. Enables richer responses.
3. **Consolidate PTZ** (#53) — Currently split across 4 files. After `v4l2SetMultiple` removal in Round 5, `v4l2.go` is smaller but still fragmented.

### Documentation Debt

- `docs/SUPERB_ROADMAP.md` is stale (#40, #61) — metrics, file tables, and dependency lists all outdated
- `TODO_LIST.md` has 26 remaining items; many overlap with the roadmap

---

## F) TOP #25 THINGS WE SHOULD GET DONE NEXT

Prioritized by impact × effort (Pareto order):

### Tier 1: Critical Fixes (30 min)

| # | Item                                               | Impact                             | Effort |
| - | -------------------------------------------------- | ---------------------------------- | ------ |
| 1 | Fix `go-branded-id` v0.3.0 test failures (4 tests) | 🔴 CI reliability                  | 15 min |
| 2 | Add middleware-aware integration test harness      | 🔴 Prevents middleware regressions | 30 min |

### Tier 2: Code Quality (1-2 hours)

| # | Item                                                 | Impact           | Effort |
| - | ---------------------------------------------------- | ---------------- | ------ |
| 3 | Structured log levels audit (#14)                    | 🟢 Observability | 30 min |
| 4 | Graceful degradation for missing optional deps (#15) | 🟢 Robustness    | 30 min |
| 5 | Update/archive `SUPERB_ROADMAP.md` (#40, #61)        | 🟢 Doc accuracy  | 20 min |

### Tier 3: Observability (2-3 hours)

| # | Item                                                                 | Impact           | Effort |
| - | -------------------------------------------------------------------- | ---------------- | ------ |
| 6 | Additional Prometheus metrics — stream duration, frames served (#16) | 🟢 Observability | 1h     |
| 7 | Stream health monitoring — frame counter, uptime metric (#18)        | 🟢 Reliability   | 1h     |
| 8 | Circuit breaker for HID failures (#17)                               | 🟢 Stability     | 1h     |

### Tier 4: Architecture (4-8 hours)

| #  | Item                                   | Impact                              | Effort |
| -- | -------------------------------------- | ----------------------------------- | ------ |
| 9  | Extract `Dependencies` interface (#51) | 🔵 Testability, compile-time safety | 2h     |
| 10 | Typed `CommandResult` (#52)            | 🔵 Richer command responses         | 2h     |
| 11 | Consolidate PTZ into `ptz.go` (#53)    | 🟢 Maintainability                  | 1h     |
| 12 | Extract `Commander` interface (#21)    | 🔵 Mockable shell commands          | 2h     |
| 13 | Extract `HIDDevice` interface (#22)    | 🔵 Mockable HID I/O                 | 2h     |

### Tier 5: Web UI (4-8 hours)

| #  | Item                                   | Impact           | Effort |
| -- | -------------------------------------- | ---------------- | ------ |
| 14 | WebSocket for live state updates (#27) | 🔵 Real-time UX  | 3h     |
| 15 | Mobile-responsive layout (#26)         | 🟢 Accessibility | 2h     |
| 16 | Keyboard shortcuts for PTZ (#28)       | 🟢 UX            | 1h     |
| 17 | PTZ relative mode (#29)                | 🟢 UX            | 1h     |
| 18 | Camera preset support (#30)            | 🟢 UX            | 2h     |

### Tier 6: Testing (4-8 hours)

| #  | Item                                                                               | Impact            | Effort |
| -- | ---------------------------------------------------------------------------------- | ----------------- | ------ |
| 19 | Integration test harness with fake devices (#31)                                   | 🔵 Test coverage  | 3h     |
| 20 | Test coverage for real hardware paths (#32)                                        | 🟢 Confidence     | 2h     |
| 21 | Surface auto-manage errors to web UI (#33)                                         | 🟢 Debuggability  | 1h     |
| 22 | Improve MJPEG stream reconnection (#34)                                            | 🟢 Reliability    | 2h     |
| 23 | Fix flaky parallel tests (`TestHandleStream_NoFFmpeg`, `TestSocket_StatusCommand`) | 🔴 CI reliability | 30 min |
| 24 | Integration test with real hardware (#35)                                          | 🟢 Validation     | 3h     |

### Tier 7: Polish (1-2 hours)

| #  | Item                                               | Impact | Effort |
| -- | -------------------------------------------------- | ------ | ------ |
| 25 | PTZ readback accuracy — in-memory "last set" (#42) | 🟢 UX  | 1h     |

---

## G) TOP #1 QUESTION WE CANNOT FIGURE OUT OURSELVES

**Why does `go-branded-id` v0.3.0 prefix `String()` output with the type name?** (`"PID:42"` instead of `"42"`). The project author owns both this repo and `go-branded-id`. Was this an intentional breaking change? Should we:

- (a) Update all test expectations to match the new format
- (b) Pin `go-branded-id` to v0.2.x in `go.mod`
- (c) Add a `Value()` or `Raw()` method to `go-branded-id` and update callers

The `nix` `doCheck = false` workaround masks this in builds but the tests fail on every `go test` run. We need a decision on the intended direction before fixing the 4 affected tests.

---

## Uncommitted Changes

| File               | Change                                                |
| ------------------ | ----------------------------------------------------- |
| `middleware.go`    | Added `Flush()` method to `responseWriter`            |
| `handlers_test.go` | Added `TestLoggingMiddleware_Flusher` regression test |
