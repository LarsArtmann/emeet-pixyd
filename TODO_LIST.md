# emeet-pixyd — TODO List

**Updated:** 2026-05-07
**Source docs verified:** SUPERB_ROADMAP.md, all planning docs, all status docs

---

## Status Legend

- ✅ DONE — Verified in code
- 🔶 PARTIAL — Started but incomplete
- ⬜ TODO — Not started

---

## Phase 1: Quick Wins (P0)

| #   | Status     | Task                                                                                                            | Source        |
| --- | ---------- | --------------------------------------------------------------------------------------------------------------- | ------------- |
| 1   | ✅ DONE    | `.golangci.yml` centralized configuration                                                                       | Roadmap 2.3   |
| 2   | ✅ DONE    | Fix linter suppressions (nlreturn, whitespace, goconst, perfsprint, modernize)                                  | Roadmap 7.1   |
| 3   | ✅ DONE    | `CommandError` structured error type                                                                            | Roadmap 3.4   |
| 4   | 🔶 PARTIAL | `String()` method test coverage — `CameraState.String()`, `AudioMode.String()` exist but no explicit test calls | Roadmap 7.2   |
| 5   | ✅ DONE    | `t.Parallel()` in all tests (only 2 justified serial tests)                                                     | Quality sweep |

## Phase 2: Decomposition (P1)

| #   | Status  | Task                                                                                   | Source        |
| --- | ------- | -------------------------------------------------------------------------------------- | ------------- |
| 6   | ✅ DONE | Decompose `Run()` into focused helpers                                                 | Roadmap 2.1   |
| 7   | ✅ DONE | pprof endpoint gated behind `Config.Debug`                                             | Roadmap 4.3   |
| 8   | ✅ DONE | Keyboard shortcuts in web UI (T/I/P/C)                                                 | Roadmap 5.2   |
| 9   | ✅ DONE | `AutoMode`/`DefaultAudio` from env vars                                                | Quality sweep |
| 10  | ✅ DONE | Uevent context cancellation (goroutine leak fix)                                       | Quality sweep |
| 11  | ✅ DONE | Device name matching shared `isPixyName()` helper                                      | Quality sweep |
| 12  | ✅ DONE | Error var consolidation (no duplicates)                                                | Quality sweep |
| 13  | ⬜ TODO | Eliminate `init()` for Prometheus metrics — lazy registration or constructor injection | Roadmap 2.2   |
| 14  | ⬜ TODO | Structured log levels audit (standardize Debug/Info/Warn/Error usage)                  | Roadmap 4.2   |
| 15  | ⬜ TODO | Graceful degradation for missing optional deps (cache availability at startup)         | Roadmap 3.1   |

## Phase 3: Observability (P1-P2)

| #   | Status  | Task                                                                                     | Source      |
| --- | ------- | ---------------------------------------------------------------------------------------- | ----------- |
| 16  | ⬜ TODO | Additional Prometheus metrics (stream duration, frames, command counters, probe, uevent) | Roadmap 4.1 |
| 17  | ⬜ TODO | Circuit breaker for HID failures (stop retrying after N consecutive failures)            | Roadmap 3.2 |
| 18  | ⬜ TODO | Stream health monitoring (frame counter, uptime metric)                                  | Roadmap 3.3 |
| 19  | ⬜ TODO | Benchmark suite (ExtractJPEGFrame, ParseHIDResponse, UpdateMetrics, HandleCommand)       | Roadmap 6.3 |
| 20  | ⬜ TODO | Continuous fuzz in CI (60s per test, store corpus, fail on crash)                        | Roadmap 6.2 |

## Phase 4: Architecture (P2-P3)

| #   | Status     | Task                                                                                    | Source            |
| --- | ---------- | --------------------------------------------------------------------------------------- | ----------------- |
| 21  | ⬜ TODO    | Extract `Commander` interface for shell commands (wpctl, notify-send, ffmpeg, v4l2-ctl) | Roadmap 1.1       |
| 22  | ⬜ TODO    | Extract `HIDDevice` interface for HID I/O                                               | Roadmap 1.2       |
| 23  | ⬜ TODO    | Extract `ProcessInspector` interface for /proc traversal                                | Roadmap 1.3       |
| 24  | ⬜ TODO    | Extract `UeventListener` interface for netlink                                          | Roadmap 1.4       |
| 25  | 🔶 PARTIAL | `probeDevices()` — pure functions extracted, but still mutates under caller's lock      | Quality sweep 4.1 |

## Phase 5: Web UI (P2-P3)

| #   | Status  | Task                                                       | Source      |
| --- | ------- | ---------------------------------------------------------- | ----------- |
| 26  | ⬜ TODO | Mobile-responsive layout                                   | Roadmap 5.3 |
| 27  | ⬜ TODO | WebSocket for live state updates (replace 3s HTMX polling) | Roadmap 5.1 |
| 28  | ⬜ TODO | Keyboard shortcuts for PTZ (arrow keys, +/- for zoom)      | Status E.12 |
| 29  | ⬜ TODO | PTZ relative mode (`pan+10`, `tilt-5`)                     | Status E.8  |
| 30  | ⬜ TODO | Camera preset support (save/recall PTZ positions)          | Status F.9  |

## Phase 6: Testing (P3-P4)

| #   | Status  | Task                                                                      | Source      |
| --- | ------- | ------------------------------------------------------------------------- | ----------- |
| 31  | ⬜ TODO | Integration test harness with fake devices                                | Roadmap 6.1 |
| 32  | ⬜ TODO | Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths | Status E.2  |
| 33  | ⬜ TODO | Surface auto-manage errors to web UI                                      | Status E.3  |
| 34  | ⬜ TODO | Improve MJPEG stream reconnection                                         | Status E.4  |
| 35  | ⬜ TODO | Integration test with real hardware (build tag guarded)                   | Status F.15 |

## Phase 7: Code Nits (from this review)

| #   | Status  | Task                                                                                               | Source     |
| --- | ------- | -------------------------------------------------------------------------------------------------- | ---------- |
| 36  | ⬜ TODO | Fix `applyResponseToStatus` — should use toast type from `actionToast`, not always `toastTypeInfo` | Review L2  |
| 37  | ⬜ TODO | Extract `lastFrame`/`ptzCache` from anonymous embedded structs to named types                      | Review M1  |
| 38  | ⬜ TODO | Move `streamBufSize`/`ffmpegShutdown` constants from handlers.go to stream.go                      | Review L12 |
| 39  | ⬜ TODO | Remove decorative blank lines in stream.go select/case blocks                                      | Review M5  |
| 40  | ⬜ TODO | Update `SUPERB_ROADMAP.md` — many items completed                                                  | Review M4  |
| 41  | ⬜ TODO | Consolidate PTZ axis dispatch into lookup table                                                    | Review     |
| 42  | ⬜ TODO | PTZ readback accuracy — delay before readback or maintain in-memory "last set" value               | Status E.1 |

---

## Docs Verified

| File              | Status                                         |
| ----------------- | ---------------------------------------------- |
| AGENTS.md         | ✅ Current as of 2026-05-03                    |
| FEATURES.md       | ✅ Verified — 43 features, all match code      |
| SUPERB_ROADMAP.md | 🔶 Stale — many items completed but not marked |
| README.md         | ✅ Current                                     |
| CHANGELOG.md      | ✅ Current                                     |

## Summary

| Status     | Count |
| ---------- | ----- |
| ✅ DONE    | 12    |
| 🔶 PARTIAL | 3     |
| ⬜ TODO    | 27    |
| **Total**  | 42    |
