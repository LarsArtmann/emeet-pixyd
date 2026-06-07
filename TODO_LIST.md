# emeet-pixyd — TODO List

**Updated:** 2026-06-06 (Round 6)
**Source docs verified:** docs/SUPERB_ROADMAP.md, AGENTS.md, all planning/status docs

---

## Status Legend

- ✅ DONE — Verified in code
- 🔶 PARTIAL — Started but incomplete
- ⬜ TODO — Not started

---

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
| 14  | ⬜ TODO | Structured log levels audit (standardize Debug/Info/Warn/Error usage)             | Roadmap 4.2   |
| 15  | ✅ DONE | Graceful degradation for missing optional deps (`checkExternalDeps()` at startup) | Roadmap 3.1   |

## Phase 3: Observability (P1-P2)

| #   | Status  | Task                                                                                                                  | Source      |
| --- | ------- | --------------------------------------------------------------------------------------------------------------------- | ----------- |
| 16  | ✅ DONE | Additional Prometheus metrics (command_total, probe_total, uevent_total, hid_failures, stream_duration, frames_total) | Roadmap 4.1 |
| 17  | ✅ DONE | Circuit breaker for HID failures (`hidFailCount` + `hidCircuitBreakerThreshold`)                                      | Roadmap 3.2 |
| 18  | ✅ DONE | Stream health monitoring (`metricStreamDuration` histogram + `metricFramesTotal` counter)                             | Roadmap 3.3 |
| 19  | ✅ DONE | Benchmark suite (7 benchmarks: JPEG, HID, Waybar, HandleCommand, GetWebStatus, FormatLastSynced)                      | Roadmap 6.3 |
| 20  | ⬜ TODO | Continuous fuzz in CI (60s per test, store corpus, fail on crash)                                                     | Roadmap 6.2 |

## Phase 4: Architecture (P2-P3)

| #   | Status  | Task                                                                                       | Source            |
| --- | ------- | ------------------------------------------------------------------------------------------ | ----------------- |
| 21  | ⬜ TODO | Extract `Commander` interface for shell commands (wpctl, notify-send, ffmpeg, v4l2-ctl)    | Roadmap 1.1       |
| 22  | ✅ DONE | Extract `HIDDevice` interface for HID I/O (`Send`/`SendRecv` methods, `hidrawDevice` impl) | Roadmap 1.2       |
| 23  | ⬜ TODO | Extract `ProcessInspector` interface for /proc traversal                                   | Roadmap 1.3       |
| 24  | ⬜ TODO | Extract `UeventListener` interface for netlink                                             | Roadmap 1.4       |
| 25  | ✅ DONE | `probeDevices()` — pure function returning `probeResult`, applied via `applyProbeResult`   | Quality sweep 4.1 |

## Phase 5: Web UI (P2-P3)

| #   | Status  | Task                                                       | Source      |
| --- | ------- | ---------------------------------------------------------- | ----------- |
| 26  | ⬜ TODO | Mobile-responsive layout                                   | Roadmap 5.3 |
| 27  | ⬜ TODO | WebSocket for live state updates (replace 3s HTMX polling) | Roadmap 5.1 |
| 28  | ✅ DONE | Keyboard shortcuts for PTZ (arrow keys, +/- for zoom)      | Status E.12 |
| 29  | ✅ DONE | PTZ relative mode (`pan+10`, `tilt-5`) via `parsePTZValue` | Status E.8  |
| 30  | ⬜ TODO | Camera preset support (save/recall PTZ positions)          | Status F.9  |

## Phase 6: Testing (P3-P4)

| #   | Status  | Task                                                                      | Source      |
| --- | ------- | ------------------------------------------------------------------------- | ----------- |
| 31  | ⬜ TODO | Integration test harness with fake devices                                | Roadmap 6.1 |
| 32  | ⬜ TODO | Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths | Status E.2  |
| 33  | ✅ DONE | Surface auto-manage errors to web UI (`autoError` field + `errors.Join`)  | Status E.3  |
| 34  | ⬜ TODO | Improve MJPEG stream reconnection                                         | Status E.4  |
| 35  | ⬜ TODO | Integration test with real hardware (build tag guarded)                   | Status F.15 |

## Phase 7: Code Nits (from this review)

| #   | Status  | Task                                                                                  | Source     |
| --- | ------- | ------------------------------------------------------------------------------------- | ---------- |
| 36  | ✅ DONE | Extract toast type from `actionToast`, propagate through `applyResponseToStatus`      | Review L2  |
| 37  | ✅ DONE | Extract `lastFrame`/`ptzCache` into named types in `cache.go`                         | Review M1  |
| 38  | ✅ DONE | Moved `streamBufSize`/`ffmpegShutdownTimeout` constants from handlers.go to stream.go | Review L12 |
| 39  | ✅ DONE | Removed decorative blank lines in stream.go select/case blocks                        | Review M5  |
| 40  | ✅ DONE | Update `SUPERB_ROADMAP.md` — completion status table added, marked archived           | Review M4  |
| 41  | ✅ DONE | Consolidate PTZ axis dispatch into `ptzAxes` lookup table                             | Review     |
| 42  | ⬜ TODO | PTZ readback accuracy — delay before readback or maintain in-memory "last set" value  | Status E.1 |

## Phase 8: From 15-Skill Comprehensive Audit (2026-05-12)

|     | #       | Status                                                                                                   | Task            | Source |
| --- | ------- | -------------------------------------------------------------------------------------------------------- | --------------- | ------ |
| 43  | ✅ DONE | Fix `hidSendRecv` nil error wrapping bug (zero-write produces `%!w(<nil>)`)                              | Code Review C1  |
| 44  | ✅ DONE | Fix `hasPixyVendorProduct` — `return false` → `continue` on malformed HID_ID                             | Code Review C4  |
| 45  | ✅ DONE | Fix `flake.nix` — remove invalid `env` attribute from app definition                                     | Nix Review      |
| 46  | ✅ DONE | Fix `package.nix` — deduplicate version string via `let version` binding                                 | Nix Review      |
| 47  | ✅ DONE | Fix `autoManage` — only call `saveState` when state actually changed                                     | Self-Review 4   |
| 48  | ✅ DONE | Validate loaded state in `loadState()` — reject garbage CameraState/AudioMode/AutoMode values            | Code Review C2  |
| 49  | ✅ DONE | Fix `uevent.go` — transient read errors permanently disable hotplug, added retry with continue           | Code Review C5  |
| 50  | ✅ DONE | Moved PTZ limits to shared constants in `internal/pixy/` (eliminated split brain with templates)         | Self-Review S1  |
| 51  | ✅ DONE | Consolidate 10 DI function pointers into `Dependencies` struct in `deps.go`                              | Architecture 3  |
| 52  | ✅ DONE | Replace `handleCommand(string) string` with typed `CommandResult` struct                                 | Architecture 2  |
| 53  | ✅ DONE | Consolidate PTZ logic into `ptz.go` (extracted from handlers.go + v4l2.go, v4l2.go deleted)              | Architecture 3  |
| 54  | ✅ DONE | Added systemd hardening to NixOS module (ProtectSystem, PrivateTmp, NoNewPrivileges, MemoryMax)          | Nix Review H2   |
| 55  | ✅ DONE | Fixed false-positive tests — proper assertions for sync/toggle-privacy commands                          | BDD Review P0   |
| 56  | ✅ DONE | Removed `, change` from PTZ slider hx-trigger (was doubling requests)                                    | Frontend Review |
| 57  | ✅ DONE | Suppress toast spam during PTZ slider drag (empty toast on success)                                      | Frontend Review |
| 58  | ✅ DONE | Added `role="alert"` to error banners for screen reader announcement                                     | Frontend A11y   |
| 59  | ❌ SKIP | `encoding/json/v2` not available in Go 1.26.2 stdlib — revisit when landed                               | How-to-Go       |
| 60  | ✅ DONE | Added `extractJPEGFrame` max-iterations guard (10M) to prevent infinite loop on corrupt stream           | Self-Review 4.8 |
| 62  | ✅ DONE | Enrich `PTZValues` with `Get(axis)`/`Set(axis, val)` methods, eliminate all hardcoded V4L2 control names | Session 7       |

---

## Docs Verified

| File                   | Status                                           |
| ---------------------- | ------------------------------------------------ |
| AGENTS.md              | ✅ Current as of 2026-06-06                      |
| FEATURES.md            | ✅ Verified — 44 features, all match code        |
| docs/SUPERB_ROADMAP.md | ✅ Archived — completion status added 2026-06-05 |
| README.md              | ✅ Current                                       |
| CHANGELOG.md           | ✅ Current                                       |

## Summary

|            | Status | Count |
| ---------- | ------ | ----- |
| ✅ DONE    | 44     |
| 🔶 PARTIAL | 0      |
| ❌ SKIP    | 1      |
| ⬜ TODO    | 17     |
| **Total**  | 62     |
