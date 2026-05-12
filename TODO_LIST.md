# emeet-pixyd — TODO List

**Updated:** 2026-05-12
**Source docs verified:** docs/SUPERB_ROADMAP.md, all planning docs, all status docs

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

## Phase 8: From 15-Skill Comprehensive Audit (2026-05-12)

|| #   | Status  | Task                                                                                               | Source          |
| --- | ------- | -------------------------------------------------------------------------------------------------- | --------------- |
| 43  | ✅ DONE | Fix `hidSendRecv` nil error wrapping bug (zero-write produces `%!w(<nil>)`)                       | Code Review C1  |
| 44  | ✅ DONE | Fix `hasPixyVendorProduct` — `return false` → `continue` on malformed HID_ID                       | Code Review C4  |
| 45  | ✅ DONE | Fix `flake.nix` — remove invalid `env` attribute from app definition                               | Nix Review      |
| 46  | ✅ DONE | Fix `package.nix` — deduplicate version string via `let version` binding                           | Nix Review      |
| 47  | ⬜ TODO | Fix `autoManage` — only call `saveState` when state actually changed                               | Self-Review 4   |
| 48  | ⬜ TODO | Validate loaded state in `loadState()` — reject garbage CameraState/AudioMode/AutoMode values      | Code Review C2  |
| 49  | ⬜ TODO | Fix `uevent.go` — transient read errors permanently disable hotplug, add retry                     | Code Review C5  |
| 50  | ⬜ TODO | Move PTZ limits to shared constants in `internal/pixy/` (split brain with templates.templ)         | Self-Review S1  |
| 51  | ⬜ TODO | Consolidate 9 function pointers into a `Dependencies` interface for compile-time safety            | Architecture 3  |
| 52  | ⬜ TODO | Replace `handleCommand(string) string` with typed `CommandResult` struct                           | Architecture 2  |
| 53  | ⬜ TODO | Consolidate PTZ logic into single `ptz.go` (currently split across 5 files)                       | Architecture 3  |
| 54  | ⬜ TODO | Add systemd hardening to NixOS module (MemoryMax, ProtectSystem, RestrictAddressFamilies)          | Nix Review H2   |
| 55  | ⬜ TODO | Fix false-positive tests — `TestHandleCommandSyncWithDevice` accepts both success AND error       | BDD Review P0   |
| 56  | ⬜ TODO | Remove `, change` from PTZ slider hx-trigger (doubles requests)                                    | Frontend Review |
| 57  | ⬜ TODO | Suppress toast spam during PTZ slider drag                                                         | Frontend Review |
| 58  | ⬜ TODO | Add `role="alert"` to error banners for screen reader announcement                                | Frontend A11y   |
| 59  | ⬜ TODO | Migrate to `encoding/json/v2` for 10x JSON performance                                             | How-to-Go       |
| 60  | ⬜ TODO | Add `extractJPEGFrame` max-iterations guard to prevent infinite loop on corrupt stream              | Self-Review 4.8 |
| 61  | ⬜ TODO | Archive or rewrite `docs/SUPERB_ROADMAP.md` — metrics/file table/deps all stale                    | Docs Freshness  |

---

## Docs Verified

| File                | Status                                         |
| ------------------- | ---------------------------------------------- |
| AGENTS.md           | ✅ Current as of 2026-05-12                    |
| FEATURES.md         | ✅ Verified — 44 features, all match code      |
| docs/SUPERB_ROADMAP.md | 🔶 Stale — metrics/file table/deps all wrong |
| README.md           | ✅ Current                                     |
| CHANGELOG.md        | ✅ Current                                     |

## Summary

| Status     | Count |
| ---------- | ----- |
| ✅ DONE    | 16    |
| 🔶 PARTIAL | 3     |
| ⬜ TODO    | 42    |
| **Total**  | 61    |
