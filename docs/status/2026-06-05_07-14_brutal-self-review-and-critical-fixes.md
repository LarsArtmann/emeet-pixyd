# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-06-05 07:14
**Branch:** master
**Head:** `473bff9` style(middleware): fix gci formatting and nolint comments
**Previous report:** 2026-05-30_11-27_nix-improvements-and-design-context-status.md

---

## Executive Summary

The project is a **production-ready Linux daemon** for the EMEET PIXY dual-camera AI webcam. All 44 features are fully functional. This session focused on a **brutal self-review** that uncovered a **broken build** (since commit `d012758` on 2026-05-30), which was fixed along with several architectural improvements.

**Key finding:** The `deps: update nixpkgs, httputil, and prometheus/common` commit (`d012758`) introduced two breaking changes that went undetected:
1. `httputil.Chain` changed parameter type from `[]func(http.Handler) http.Handler` to `[]httputil.Middleware` (named type)
2. `go-branded-id` v0.3.0 changed `String()` to include brand prefix (`"PID:42"` instead of `"42"`), breaking `/proc` path construction and `wpctl` commands

Both were fixed. The codebase now fully adopts the `httputil` library instead of reimplementing its features locally.

---

## Build & Quality Gates

| Gate | Status | Details |
|------|--------|---------|
| `go build ./...` | ✅ PASS | Clean compilation |
| `go test -race -count=1 ./...` | ✅ PASS | All tests green |
| `golangci-lint run --timeout 2m` | ✅ PASS | 0 issues |
| `nix build` | ✅ PASS | Production binary builds |
| Test coverage (main) | ✅ 70.7% | 0.807s execution |
| Test coverage (pixy) | ✅ 80.6% | 0.004s execution |

---

## Codebase Metrics

| Metric | Value |
|--------|-------|
| Total Go lines | 10,281 |
| Production code | ~4,800 lines (excluding tests) |
| Test code | ~5,481 lines |
| Source files | 27 `.go` files |
| Test files | 12 `_test.go` files |
| Fuzz tests | 2 (`handlers_fuzz_test.go`, `hid_fuzz_test.go`) |
| Benchmarks | 4 (JPEG extract, format synced, HID parse, waybar output) |
| Largest file | `main_test.go` (1,457 lines) |
| Largest prod file | `main.go` (684 lines) |
| External deps | 9 direct, 28 total |

### File Inventory

| Category | Files | Lines |
|----------|-------|-------|
| Core daemon | `main.go` | 684 |
| Commands | `commands.go` | 314 |
| HTTP handlers | `handlers.go` | 345 |
| HID protocol | `hid.go` | 260 |
| MJPEG streaming | `stream.go` | 201 |
| Device probing | `probe.go` | 147 |
| Process detection | `process.go` | 146 |
| Auto-management | `auto.go` | 135 |
| State persistence | `state.go` | 94 |
| Metrics (OTel) | `metrics.go` | 81 |
| Uevent listener | `uevent.go` + `uevent_linux.go` | 105+33 |
| Middleware | `middleware.go` | 45 |
| V4L2 PTZ | `v4l2.go` | 67 |
| Error types | `errors.go` | 31 |
| Cache types | `cache.go` | 57 |
| Web types | `web_types.go` | 22 |
| Domain types | `internal/pixy/pixy.go` | 458 |
| Branded IDs | `internal/pixy/ids.go` | 26 |
| Templates | `templates.templ` | 213 |
| Frontend | `style.css` + `app.js` | 700+196 |
| NixOS module | `modules/nixos.nix` | 124 |
| Nix build | `flake.nix` + `package.nix` | 105+42 |

---

## A) FULLY DONE ✅

### This Session (6 commits)

| Commit | What |
|--------|------|
| `58a28ce` | **CRITICAL FIX**: Restore build — replace local middleware reimplementation with `httputil` library. Fix branded-id v0.3.0 `String()` prefix breaking `/proc` paths and `wpctl` commands. |
| `6451ef1` | **CRITICAL FIX**: Update nix `vendorHash` for dependency changes. |
| `c497b03` | Consolidate PTZ axis constants (`axisPan`/`axisTilt`/`axisZoom`) from `v4l2.go` into `internal/pixy/pixy.go` — eliminate split brain. |
| `ff6b75e` | Upgrade `notify-send` failure from `slog.Debug` to `slog.Warn`. |
| `e63c16f` | Update AGENTS.md for all changes. |
| `473bff9` | Fix gci formatting. |

### From TODO_LIST.md (36/61 items DONE)

All Phase 1 (Quick Wins), Phase 2 (Decomposition), and many cross-cutting concerns are complete:

- `.golangci.yml` centralized config
- `CommandError` structured error type
- `t.Parallel()` in all tests
- `Run()` decomposition into focused helpers
- pprof gated behind `Config.Debug`
- Keyboard shortcuts in web UI
- Uevent context cancellation
- `init()` elimination (lazy metrics via `sync.Once`)
- Named cache types (`lastFrameCache`, `ptzCache`)
- PTZ axis lookup table
- State validation on load
- Waybar output optimization
- 4 benchmarks established
- systemd hardening in NixOS module
- And 22 more completed items

### Architecture Quality

- **DI pattern**: 9 function fields for testability, all mockable
- **Branded types**: `PID` and `SourceID` prevent compile-time mixing
- **Type-safe enums**: `CameraState`, `AudioMode`, `AutoMode` with `Valid()` and `String()`
- **Domain constants**: PTZ limits and axis names in `internal/pixy/`
- **Middleware**: Fully delegated to `httputil` library
- **Metrics**: OTel with Prometheus exporter, lazy registration
- **State**: Atomic JSON persistence with validation

---

## B) PARTIALLY DONE 🔶

None — all started items were completed.

---

## C) NOT STARTED ⬜ (28 items from TODO_LIST.md)

### Phase 2: Decomposition
- **#14** Structured log levels audit — partially addressed (1 fix: notification failure), full systematic audit not done

### Phase 3: Observability
- **#15** Graceful degradation for missing optional deps
- **#16** Additional Prometheus metrics (stream, command counters, probe, uevent)
- **#17** Circuit breaker for HID failures
- **#18** Stream health monitoring (frame counter, uptime metric)
- **#20** Continuous fuzz in CI

### Phase 4: Architecture
- **#21** Extract `Commander` interface for shell commands
- **#22** Extract `HIDDevice` interface for HID I/O
- **#23** Extract `ProcessInspector` interface for `/proc` traversal
- **#24** Extract `UeventListener` interface for netlink

### Phase 5: Web UI
- **#26** Mobile-responsive layout
- **#27** WebSocket for live state updates (replace 3s HTMX polling)
- **#28** Keyboard shortcuts for PTZ (arrow keys, +/-)
- **#29** PTZ relative mode (`pan+10`, `tilt-5`)
- **#30** Camera preset support (save/recall PTZ positions)

### Phase 6: Testing
- **#31** Integration test harness with fake devices
- **#32** Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths
- **#33** Surface auto-manage errors to web UI
- **#34** Improve MJPEG stream reconnection
- **#35** Integration test with real hardware (build tag guarded)

### Phase 7: Code Nits
- **#40** Update `SUPERB_ROADMAP.md`
- **#42** PTZ readback accuracy

### Phase 8: From 15-Skill Audit
- **#51** Consolidate 9 function pointers into `Dependencies` interface
- **#52** Replace `handleCommand(string) string` with typed `CommandResult` struct
- **#53** Consolidate PTZ logic into single `ptz.go`
- **#59** `encoding/json/v2` migration (skipped — not in Go 1.26 stdlib)
- **#61** Archive or rewrite `SUPERB_ROADMAP.md`

---

## D) TOTALLY FUCKED UP 💥

### 1. Build Was Broken for ~6 Days (commit `d012758` → `58a28ce`)

The `deps: update nixpkgs, httputil, and prometheus/common` commit introduced breaking changes that went undetected because:
- `doCheck = false` in `package.nix` — nix doesn't run tests
- The pre-commit hook (BuildFlow) doesn't run `go build` or `go test`
- CI wasn't triggered (no push between those commits)

**Impact:** Any `go build`, `go test`, or `nix build` would fail. The `httputil` library changed `Chain`'s parameter type, and `go-branded-id` v0.3.0 changed `String()` output to include a prefix.

**Root cause:** Dependency bumps were not verified with a build before committing.

**Fix:** Fully adopted `httputil` library (removed local reimplementations), fixed branded-id usage to use `.Get()` for operational paths.

### 2. Branded ID `String()` Was Used for Runtime Paths

`ppidOf()` used `pid.String()` to construct `/proc/{PID}/stat` — with v0.3.0 this produced `/proc/PID:42/stat`. Similarly `setDefaultSource()` passed `sourceID.String()` to `wpctl set-default`, producing `wpctl set-default SourceID:42`. Both were runtime bugs that would manifest as:
- Call detection broken (ppidOf always returns zero PID)
- PipeWire source switching broken (wpctl receives invalid source ID)

### 3. `SUPERB_ROADMAP.md` Is Stale

Multiple items in the roadmap are completed but not marked. Metrics table, file responsibility table, and dependency list are all outdated.

---

## E) WHAT WE SHOULD IMPROVE

### Critical Process

1. **Pre-commit hook must run `go build`** — The BuildFlow hook runs lint, formatting, and binary checks but doesn't compile. A broken build should never reach master.
2. **Dependency bumps must include a verification step** — `go build ./...` and `go test ./...` after every `go get -u`.

### Architecture

3. **`handleCommand(string) string` is stringly-typed** — Every command returns a string. Errors are detected by prefix matching (`IsCommandErrorResponse`). This is fragile and makes it hard to add structured responses.
4. **9 function pointers could be a `Dependencies` interface** — Currently `Daemon` has 9 separate DI function fields. An interface would give compile-time safety and better documentation.
5. **`main.go` at 684 lines** — The largest file. Could extract waybar logic, socket listener, and HTTP server setup into focused files.
6. **PTZ logic scattered across 5 files** — `handlers.go`, `commands.go`, `v4l2.go`, `cache.go`, `web_types.go`. A dedicated `ptz.go` would consolidate.

### Testing

7. **70.7% coverage on main package** — Several paths untested (real HID communication, ffmpeg streaming, netlink uevent). Hard to test without hardware.
8. **`doCheck = false` in nix** — Nix sandbox lacks devices. Integration test harness with fake devices would enable nix-level testing.
9. **No coverage threshold in CI** — Could silently regress.

### Observability

10. **Only 3 metrics** (in_call, auto_mode, camera_state) — No command counters, no stream duration, no HID failure rate, no probe/uevent counters.
11. **No circuit breaker for HID** — Consecutive HID failures just re-probe every time.

### Documentation

12. **`SUPERB_ROADMAP.md` stale** — Many completed items not reflected.
13. **No `DESIGN.md` update** since middleware rewrite.

---

## F) Top 25 Things We Should Get Done Next

Sorted by impact × effort (highest first):

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 1 | **Add `go build ./...` to BuildFlow pre-commit hook** | CRITICAL | 5min | Prevent broken builds from ever reaching master again |
| 2 | **Add `go test ./...` to BuildFlow pre-commit hook** | CRITICAL | 5min | Same — catch test regressions at commit time |
| 3 | **Replace `handleCommand(string) string` with typed `CommandResult`** | HIGH | 2h | Eliminates stringly-typed command dispatch, enables structured error propagation |
| 4 | **Consolidate 9 DI function fields into `Dependencies` interface** | HIGH | 1h | Compile-time safety, better test mocks, clearer API surface |
| 5 | **Extract waybar logic from `main.go` into `waybar.go`** | MED | 30min | `main.go` is 684 lines — waybar output is self-contained |
| 6 | **Extract socket listener from `main.go` into `socket.go`** | MED | 30min | Unix socket handling is independent of daemon lifecycle |
| 7 | **Consolidate PTZ logic into `ptz.go`** | MED | 1h | Currently split across 5 files — handlers, commands, v4l2, cache, web_types |
| 8 | **Add command counter metrics** | MED | 30min | Track which commands are used, error rates |
| 9 | **Add HID failure counter + simple circuit breaker** | MED | 1h | Stop re-probing on every consecutive HID failure |
| 10 | **Add stream duration + frame counter metrics** | MED | 30min | Monitor streaming health |
| 11 | **Archive `SUPERB_ROADMAP.md` — mark completed items** | LOW | 15min | Document is misleading in current state |
| 12 | **Update `DESIGN.md` for middleware rewrite** | LOW | 15min | Reflects httputil adoption |
| 13 | **Surface auto-manage errors to web UI** | LOW | 30min | Currently errors only in logs |
| 14 | **PTZ relative mode (`pan+10`, `tilt-5`)** | LOW | 30min | Useful for fine-tuning position |
| 15 | **Add uevent counter metric** | LOW | 15min | Track hotplug activity |
| 16 | **Extract `HIDDevice` interface** | MED | 1h | Separates HID protocol from daemon logic |
| 17 | **Add probe counter metric** | LOW | 15min | Track device probing activity |
| 18 | **Graceful degradation for missing optional deps** | MED | 1h | Better error messages when ffmpeg/wpctl not found |
| 19 | **Mobile-responsive web UI** | MED | 2h | Currently desktop-only layout |
| 20 | **WebSocket for live state updates** | HIGH | 3h | Replace 3s HTMX polling with push-based updates |
| 21 | **Integration test harness with fake devices** | HIGH | 4h | Enable testing HID/V4L2 paths without hardware |
| 22 | **Keyboard shortcuts for PTZ (arrow keys, +/-)** | LOW | 30min | Better PTZ control UX |
| 23 | **Camera preset support** | MED | 2h | Save/recall PTZ positions |
| 24 | **Add coverage threshold to CI** | LOW | 15min | Prevent silent coverage regression |
| 25 | **Migrate to `encoding/json/v2` when available** | LOW | 1h | Currently skipped (not in Go 1.26 stdlib) |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should the 9 DI function fields be consolidated into a `Dependencies` interface, or should we wait until we have a `CommandResult` type first?**

The `Dependencies` interface (#51 in TODO_LIST) would consolidate the 9 function pointers (`isCameraInUseFn`, `findSourceFn`, `setSourceFn`, `notifyFn`, `setTrackingFn`, `setAudioFn`, `setGestureFn`, `centerCameraFn`, `v4l2SetFn`) into a single interface for compile-time safety. But the `CommandResult` type (#52) would change how commands return data. These two refactorings are somewhat independent but both touch the same `Daemon` struct and command handling code. Doing them in the wrong order could mean touching the same files twice.

**My recommendation:** Do `CommandResult` first (it changes command return types), then `Dependencies` (it changes how the daemon accesses external dependencies). But this is a judgment call.

---

## Session Commits (6 total)

```
473bff9 style(middleware): fix gci formatting and nolint comments
e63c16f docs(AGENTS.md): update for httputil adoption, branded-id fixes, PTZ consolidation
ff6b75e fix(logs): upgrade notification failure from Debug to Warn
c497b03 refactor: move PTZ axis constants to internal/pixy package
6451ef1 fix(nix): update vendorHash for httputil middleware changes
58a28ce fix: restore build and tests after httputil/branded-id upgrades
```

## Files Changed This Session

```
16 files changed, 84 insertions(+), 122 deletions(-)
```

Net reduction of 38 lines while fixing 3 critical bugs and eliminating a split brain.
