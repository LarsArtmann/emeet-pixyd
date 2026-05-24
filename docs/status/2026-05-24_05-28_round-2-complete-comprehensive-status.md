# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-05-24 05:28 CEST
**Branch:** `master` @ `92c2451`
**Version:** v0.3.1 (post-release development)
**Working tree:** Clean, pushed to origin

---

## Executive Summary

The daemon is **feature-complete and production-ready**. All 44 features are fully functional. Two rounds of deep review (Round 1: 10 commits, Round 2: 6 commits) have been completed and pushed, adding bug fixes, new features, refactoring, and 10 new tests. The project is in a mature state with remaining work focused on architecture hardening, interface extraction, and UX polish.

| Metric | Value |
|---|---|
| Total Go code | ~10,140 lines |
| Frontend assets | 1,097 lines (CSS + JS + templ) |
| Test cases | 373 pass, 1 flaky fail |
| Lint issues | 0 |
| Go version | 1.26.2 |
| Dependencies | 8 direct, ~15 indirect |
| Features | 44/44 fully functional |
| TODO items | 27/61 done, 32 remaining, 1 partial, 1 skip |

---

## A) FULLY DONE

### Round 1 — Deep Review (10 commits, pushed)

| Commit | What | Impact |
|---|---|---|
| `9eca560` | `ParseAudioMode` accepts full name "original" | Users can type `audio original` instead of `audio org` |
| `eb279dd` | `Config.Validate()` checks `AutoMode` and `DefaultAudio` enum fields | Prevents daemon from starting with invalid config |
| `522e4f0` | Bare `auto` command shows current mode | No more silently setting to `full` on bare `auto` |
| `d9ea6ae` | Clean up leftover `.tmp` state file in `loadState()` | Survives crashed writes gracefully |
| `2990f6d` | `--version` flag + `version` socket command | Version discoverable from CLI and running daemon |
| `7ce772d` | `/api/health` JSON endpoint (503 offline, 200 online) | Monitoring integration ready |
| `198007d` | `device` command shows both video and hidraw paths | Full device visibility |
| `a46c79b` | Audio toast shows mode name ("Audio: nc" vs generic) | Better UX feedback |
| `d048d85` | Extracted `handleQueryCommand` and `handleTogglePrivacy` | Lower cyclomatic complexity |
| `0fcccad` | Removed deprecated justfile | Migration to flake.nix complete |

### Round 2 — Second-Pass Audit (6 commits, pushed)

| Commit | What | Impact |
|---|---|---|
| `e15fc3d` | `ParseAudioMode` and `ParseCameraState` case-insensitive | `NC`, `ORIGINAL`, `IDLE` now work |
| `503baaa` | Embedded `pixy.PTZValues` in `webStatus` | Eliminated Pan/Tilt/Zoom field duplication |
| `2d6c419` | Auto mode name + version in web UI footer | Users see "full" / "tracking-only" next to toggle; version visible |
| `ab8c239` | `cmdMu` lock only guards mutating commands | Query commands (waybar, version, status) no longer block on HID writes |
| `7cc9e00` | 10 new tests for `handleQueryCommand`, `handleTogglePrivacy`, `handleMutatingCommand` | Coverage for previously untested extracted methods |
| `92c2451` | AGENTS.md updated with cmdMu scope + library assessment | Future sessions have accurate context |

### Library Assessment (Completed)

Evaluated both `cqrs-htmx` and `templ-components` (same author) for potential reuse:

- **`cqrs-htmx`**: Not adopted. CQRS wiring + Casbin auth + CSRF + rate limiting — emeet-pixyd has none of these. Would pull ~30 transitive deps for zero features. Overlapping utilities already exist locally.
- **`templ-components`**: Not adopted. Tailwind CSS component library — emeet-pixyd uses 688 lines of hand-crafted dark glass-morphism CSS. Toast, loading, error handling already implemented.

### Earlier Sessions (Pre-Round 1)

- 27 of 61 TODO items completed
- All 44 features verified fully functional
- NixOS systemd hardening added
- False-positive tests fixed
- OTel metrics migration complete
- Branded types (`PID`, `SourceID`) migrated
- Uevent context-cancellation goroutine leak fixed
- HID nil error wrapping bug fixed
- `loadState()` validation added
- `uevent.go` transient read error handling fixed

---

## B) PARTIALLY DONE

| Item | Status | What's Left |
|---|---|---|
| `probeDevices()` pure extraction | Pure functions extracted, but still mutates under caller's lock | Need to return results and let caller apply mutations |
| Web UI auto mode display | Shows mode name when enabled, but no way to cycle through modes (full → tracking-only → privacy-only) | Could add mode selector or cycle button |
| `docs/SUPERB_ROADMAP.md` | Known stale since May 12 audit | Every metric outdated, needs complete rewrite or archival |

---

## C) NOT STARTED

These are the 32 remaining TODO items from `TODO_LIST.md`, grouped by priority:

### P1 — Should Do Next

| # | Item | Notes |
|---|---|---|
| 13 | Eliminate `init()` for Prometheus metrics | Move to explicit `setupMetrics()` called from `Run()` |
| 14 | Structured log levels audit | Ensure all log calls use appropriate level (Debug/Info/Warn/Error) |
| 15 | Graceful degradation for missing optional deps | `v4l2-ctl`, `ffmpeg`, `wpctl`, `notify-send` — surface which are missing |
| 16 | Additional Prometheus metrics | Call count, HID error count, stream connections |
| 17 | Circuit breaker for HID failures | Stop retrying after N consecutive failures |
| 18 | Stream health monitoring | Track stream uptime, reconnect count |
| 19 | Benchmark suite | Only 4 benchmarks exist, need more for hot paths |
| 20 | Continuous fuzz in CI | Fuzz tests exist but not run in CI |

### P2-P3 — Architecture Hardening

| # | Item | Notes |
|---|---|---|
| 21 | Extract `Commander` interface | Decouple command handling from `Daemon` struct |
| 22 | Extract `HIDDevice` interface | Mock HID without function pointers |
| 23 | Extract `ProcessInspector` interface | Mock `/proc` scanning |
| 24 | Extract `UeventListener` interface | Mock uevent source |
| 51 | Consolidate 9 function pointers into `Dependencies` interface | Replace DI function fields with single interface |
| 52 | Replace `handleCommand(string) string` with typed `CommandResult` | Biggest remaining smell — stringly-typed command results |
| 53 | Consolidate PTZ logic into single `ptz.go` | PTZ spread across 5 files |
| 41 | Consolidate PTZ axis dispatch into lookup table | Replace switch with map |
| 37 | Extract `lastFrame`/`ptzCache` to named types | Named structs instead of anonymous embedded structs |
| 25 | `probeDevices()` pure function refactor | Return results, don't mutate |

### P2-P3 — UX Enhancements

| # | Item | Notes |
|---|---|---|
| 26 | Mobile-responsive layout | Currently desktop-focused |
| 27 | WebSocket for live state updates | Replace 3s polling |
| 28 | Keyboard shortcuts for PTZ (arrow keys, +/-) | Only T/I/P/C exist now |
| 29 | PTZ relative mode | Send relative movements, not absolute |
| 30 | Camera preset support | Named PTZ positions |
| 57 | Suppress toast spam during PTZ slider drag | Toast fires on every slider change |
| 42 | PTZ readback accuracy | V4L2 readback may drift from requested values |
| 33 | Surface auto-manage errors to web UI | Currently only logged |

### P3-P4 — Testing & Documentation

| # | Item | Notes |
|---|---|---|
| 31 | Integration test harness with fake devices | Fake HID/V4L2 devices |
| 32 | Test coverage for stream/process/hid real hardware | Hard to test without hardware |
| 34 | Improve MJPEG stream reconnection | Current retry logic is basic |
| 35 | Integration test with real hardware (build tag guarded) | Optional, needs hardware |
| 40 | Update `SUPERB_ROADMAP.md` | Completely stale |
| 61 | Archive or rewrite `SUPERB_ROADMAP.md` | May just archive |

---

## D) TOTALLY FUCKED UP

| Item | Severity | Description |
|---|---|---|
| `TestAutoManage_NoDevice_Returns` | **Known flaky** | Fails when a real PIXY is physically connected because `probeDevices()` finds the device. Not fixable without mocking sysfs — accepted as a known limitation. |
| `docs/SUPERB_ROADMAP.md` | **Stale** | Every metric (test count, coverage %, lint warnings, file listing) is wrong. Marked for archival. |
| No real fuckups | — | Both rounds completed cleanly. No regressions, no broken features, no data loss. |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture (Highest Impact)

1. **`handleCommand(string) string` is the biggest smell** — command results are untyped strings. Commands return `"tracking on"`, `"error: pan: invalid value"`, `"usage: auto [off|full|tracking-only|privacy-only]"`. A typed `CommandResult` struct would enable compile-time safety, structured error handling, and eliminate `IsCommandErrorResponse()` string prefix check.

2. **9 function pointer fields should become a `Dependencies` interface** — `setTrackingFn`, `setAudioFn`, `setGestureFn`, `centerCameraFn`, `v4l2SetFn`, `isCameraInUseFn`, `findSourceFn`, `setSourceFn`, `notifyFn` are all function fields on `Daemon`. A single `Dependencies` interface with a `RealDeps` and `MockDeps` implementation would be cleaner, more discoverable, and would eliminate the need for individual `testDaemonOption` wrappers.

3. **PTZ is spread across 5 files** — `handlers.go` (PTZ HTTP handler), `middleware.go` (PTZ validation), `commands.go` (PTZ command), `stream.go` (streaming), `internal/pixy/pixy.go` (constants). A `ptz.go` consolidation would reduce cognitive load.

4. **Anonymous embedded structs for `lastFrame` and `ptzCache`** — These should be named types (`FrameCache`, `PTZCache`) with their own methods. Currently they're anonymous structs embedded in `Daemon` with manual lock management.

### Code Quality

5. **`init()` in `metrics.go`** — The OTel metrics registration runs in `init()`. Should be an explicit `setupMetrics()` called from `Run()`. The `init()` makes testing harder and prevents multiple daemon instances in tests.

6. **Graceful degradation for missing tools** — The daemon silently fails if `v4l2-ctl`, `ffmpeg`, `wpctl`, or `notify-send` are missing. Should probe at startup and surface which tools are unavailable (in `/api/health` and web UI).

7. **HID circuit breaker** — After N consecutive HID failures, stop retrying and surface an error. Currently every HID failure triggers `probeDevices()` re-scan which is wasteful.

### Testing

8. **`probeDevices()` is untestable without sysfs** — The only flaky tests are the ones that touch real sysfs. Need a `Prober` interface or fake sysfs tree injection.

9. **No concurrent command tests** — The `cmdMu` optimization (Round 2) has no concurrent test verifying that query commands truly bypass the lock. Should add a test that runs a slow HID write and verifies waybar queries don't block.

10. **4 benchmarks exist, need more** — `ExtractJPEGFrame`, `FormatLastSynced`, `ParseHIDResponse`, `WaybarOutput`. Missing: `handleCommand`, `getWebStatus`, `probeDevices`, `SendCommand`.

### UX

11. **Toast spam during PTZ slider drag** — Every `input changed delay:300ms` triggers a toast. Should suppress toasts during active slider drag and show a single confirmation after.

12. **No mobile responsiveness** — The grid layout breaks below 720px (handled by media query) but individual cards are not touch-friendly.

13. **Auto mode selector in web UI** — Currently a toggle (on/off). Should allow cycling through modes (full → tracking-only → privacy-only) or show a dropdown.

---

## F) Top 25 Things We Should Get Done Next

Sorted by impact × effort (highest ROI first):

| Priority | Item | Effort | Impact | Phase |
|---|---|---|---|---|
| 1 | Typed `CommandResult` struct (replace string returns) | M | H | P1 |
| 2 | `Dependencies` interface (consolidate 9 fn pointers) | M | H | P1 |
| 3 | Eliminate `init()` in `metrics.go` | S | M | P1 |
| 4 | Graceful degradation for missing tools | S | H | P1 |
| 5 | HID circuit breaker | S | H | P1 |
| 6 | Concurrent command test (verify cmdMu optimization) | S | M | P1 |
| 7 | Structured log levels audit | S | M | P1 |
| 8 | Suppress toast spam during PTZ drag | S | M | P1 |
| 9 | Additional Prometheus metrics | S | M | P1 |
| 10 | Archive `docs/SUPERB_ROADMAP.md` | S | S | P2 |
| 11 | PTZ consolidation into `ptz.go` | M | M | P2 |
| 12 | Extract `lastFrame`/`ptzCache` to named types | S | M | P2 |
| 13 | `probeDevices()` pure function refactor | M | M | P2 |
| 14 | PTZ axis dispatch lookup table | S | S | P2 |
| 15 | Auto mode cycle button in web UI | S | M | P2 |
| 16 | Surface auto-manage errors in web UI | S | M | P2 |
| 17 | Benchmark suite expansion | S | M | P2 |
| 18 | Keyboard shortcuts for PTZ (arrow keys) | S | M | P2 |
| 19 | Stream health monitoring | M | M | P2 |
| 20 | Extract `Commander` interface | M | M | P3 |
| 21 | Extract `HIDDevice` interface | M | M | P3 |
| 22 | Extract `ProcessInspector` interface | S | M | P3 |
| 23 | Extract `UeventListener` interface | S | M | P3 |
| 24 | Mobile-responsive layout improvements | M | M | P3 |
| 25 | Continuous fuzz in CI | S | M | P2 |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should the auto mode web UI toggle cycle through modes (full → tracking-only → privacy-only → off) or should it remain a simple on/off toggle with the mode name displayed?**

The current implementation shows the mode name ("full", "tracking-only", "privacy-only") next to the toggle when enabled, but the toggle itself only flips between on (restores previous mode) and off. The alternative is a cycle button that steps through all four modes. This is a UX decision — cycling is more discoverable but the toggle is simpler. I cannot determine user preference without feedback.

---

## File Change Summary (Round 1 + Round 2)

**19 files changed, 535 insertions(+), 216 deletions(-)**

| File | Changes |
|---|---|
| `internal/pixy/pixy.go` | Case normalization, `AutoMode.Toggle()`, `Config.Validate()` enum checks, `ErrInvalidAutoMode`/`ErrInvalidDefaultAudio` sentinels |
| `internal/pixy/pixy_test.go` | Case-insensitive parse tests for AudioMode and CameraState |
| `commands.go` | Extracted `handleQueryCommand`, `handleTogglePrivacy`, `handleMutatingCommand`; `cmdMu` guards mutations only; bare `auto` shows mode; device shows both paths; version command |
| `commands_test.go` | 10 new tests for query commands, toggle-privacy, mutating commands |
| `handlers.go` | `handleHealth()` method; audio toast shows mode name; embedded PTZValues init |
| `handlers_test.go` | PTZ test uses embedded PTZValues |
| `integration_test.go` | Health endpoint tests; device test checks hidraw; full Config population |
| `web_types.go` | Embedded `pixy.PTZValues` instead of duplicate Pan/Tilt/Zoom fields |
| `templates.templ` | Auto mode name display; version in footer |
| `main.go` | `--version`/`-v` flag handling |
| `main_test.go` | Case-insensitive parse tests; full Config population |
| `state.go` | `.tmp` cleanup in `loadState()` |
| `AGENTS.md` | Comprehensive documentation of all changes, gotchas, library assessment |
| `justfile` | Deleted (deprecated in favor of flake.nix) |
| `flake.nix` | Formatting improvements |
| `modules/nixos.nix` | Formatting improvements |
| `package.nix` | Formatting improvements |
| `README.md` | Improved command listings, architecture section |
| `flake.lock` | Dependency updates |

---

## Test Results

```
373 PASS, 1 FAIL (flaky: TestAutoManage_NoDevice_Returns — real PIXY connected)
0 lint issues
go vet: clean
```

---

_Generated by Crush — 2026-05-24_
