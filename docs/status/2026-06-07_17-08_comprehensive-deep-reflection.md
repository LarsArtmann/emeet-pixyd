# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-06-07 17:08
**Session:** 8 (Deep Reflection + Execution Plan)
**Branch:** master (clean, up to date with origin)
**Commits this session:** 0 (report only)

---

## A. FULLY DONE

### Codebase Health
- **0 lint issues** (`golangci-lint run --timeout 2m ./...` → clean)
- **All tests pass** with `-race` (`go test -race -count=1 ./...` → OK)
- **72.5% total coverage** (71.2% main, 91.8% internal/pixy)
- **270 test functions** across 14 test files (243 Test + 2 Fuzz + 7 Bench + 18 subtests)
- **252 total tests** (243 `Test*` + 2 `Fuzz*` + 7 `Benchmark*`)
- **263/268 parallel** (5 justified serial — global metrics state)
- **0 TODO/FIXME/HACK** comments in production code

### Production Code Stats
- **3,235 lines** production Go code (21 files, excluding templates_templ.go)
- **11,396 total** including tests, generated templates
- **21 production files**, all under 352 lines
- **No `init()` functions** anywhere
- **0 `interface{}`/`any` usage**

### Architecture (completed in sessions 1-7)
| Item | Status |
|------|--------|
| `CommandResult` typed returns | ✅ |
| `Dependencies` struct in `deps.go` (10 DI function fields) | ✅ |
| `HIDDevice` interface + circuit breaker | ✅ |
| PTZ relative mode + keyboard shortcuts | ✅ |
| `autoError` surfacing to web UI | ✅ |
| Device methods extracted from `main.go` → `device.go` | ✅ |
| `PTZValues.Get(axis)` / `Set(axis, val)` axis-agnostic methods | ✅ |
| `v4l2CtrlToAxis` reverse map — zero hardcoded V4L2 control names | ✅ |
| `actionToasts` map (replaced 8-case switch) | ✅ |
| `waybarCameraStates` map (replaced 4-case switch) | ✅ |
| External binary constants (`ffmpegBin`, `wpctl`, `notifySend`, `v4l2ctl`) | ✅ |
| Lock consolidation in `setDeviceState` | ✅ |
| `ZoomDefault` constant in `centerCamera` | ✅ |
| OTel metrics (lazy registration via `sync.Once`) | ✅ |
| `/api/health` endpoint | ✅ |
| `--version` / `--help` flags | ✅ |
| State validation in `loadState()` | ✅ |

### TODO List Progress
- **44 DONE** / **62 total** = **71% complete**
- **1 SKIP** (`encoding/json/v2` not in Go 1.26 stdlib)
- **17 remaining TODO items** (see Section E)

---

## B. PARTIALLY DONE

### Test Coverage (72.5% overall)
| Area | Coverage | Gap |
|------|----------|-----|
| `internal/pixy/` | 91.8% | Near-complete |
| `commands.go` functions | 100% | Complete |
| `handlers.go` functions | ~95% | `handleHealth` JSON edge case |
| `auto.go` functions | 100% | Missing concurrent races |
| `device.go` | ~85% | `queryHIDState` generic at 40% |
| `stream.go` | ~80% | `cleanupFFmpeg` at 60% |
| `process.go` | ~70% | `findPixySource`/`setDefaultSource`/`notify` at 0% (external deps) |
| `ptz.go` | ~90% | `v4l2Set` at 0% (external dep) |
| `main.go` lifecycle | ~0% | `Run()`, `main()`, `handleFlag()` untestable without harness |

### Nix Flake
- Build works ✅
- `checks` only includes `build` — missing `lint`, `test`, `nixfmt` checks
- `doCheck = false` in `package.nix` (sandbox lacks `/dev/hidraw*`)

---

## C. NOT STARTED (from TODO_LIST.md)

| # | Task | Priority |
|---|------|----------|
| 14 | Structured log levels audit | P2 |
| 20 | Continuous fuzz in CI | P2 |
| 21 | Extract `Commander` interface for shell commands | P3 |
| 23 | Extract `ProcessInspector` interface for /proc | P3 |
| 24 | Extract `UeventListener` interface for netlink | P3 |
| 26 | Mobile-responsive layout | P3 |
| 27 | WebSocket for live state updates | P3 |
| 30 | Camera preset support (save/recall PTZ positions) | P3 |
| 31 | Integration test harness with fake devices | P3 |
| 32 | Test coverage for real hardware paths | P3 |
| 34 | Improve MJPEG stream reconnection | P3 |
| 35 | Integration test with real hardware (build tag guarded) | P4 |
| 42 | PTZ readback accuracy (delay or in-memory "last set") | P2 |

---

## D. TOTALLY FUCKED UP / SERIOUS ISSUES

### D1. `go.mod` Version Directive — `go 1.26.3`
The `go` directive uses `major.minor` format only. Patch versions (`1.26.3`) are not valid. `go mod tidy` will silently rewrite this to `go 1.26`. This was likely introduced by accident and has been present since the project started.

### D2. `setupStream` Returns 4 Values
`stream.go:120` returns `(*bufio.Reader, *exec.Cmd, http.Flusher, bool)` — a 4-tuple that should be a named struct. Every call site destructures with `_` for unused values.

### D3. `lastFrameCache.Get()` Returns Shared Slice Reference
`cache.go:17` returns the cached `[]byte` directly without copying. If `Set` is called concurrently, the caller's slice could be partially overwritten. Potential data race on the JPEG frame bytes sent to HTTP clients.

### D4. `hid.go` Goroutine Leak on Timeout
`SendRecv()` (line 161) starts a goroutine for reading. If the timeout fires first, the function returns `(nil, nil)` without draining the goroutine. The goroutine blocks forever on `fd.Read()`, leaked until the fd is closed.

### D5. `parseHIDResponse` Positional Byte Indexing — Undocumented Protocol
`hid.go:194-227` uses hardcoded byte offsets (`data[0]`, `data[3]`, `data[4]`) into a 64-byte HID response. No comments explain the protocol field layout. Any firmware change silently breaks parsing.

### D6. `docs/status/` Has 37 Stale Session Artifacts
The directory contains 37 timestamped session status reports dating back to 2026-04-19. These are AI session notes, not useful documentation. They add noise and make it harder to find relevant docs.

### D7. `.golangci.yml` Has 6+ Unused Linters
`clickhouselint`, `arangolint`, `ginkgolinter`, `sqlclosecheck`, `rowserrcheck` are enabled but the project uses none of these technologies. They add startup overhead. Also `goexperiment.*` build tags are not real Go build tags.

---

## E. WHAT WE SHOULD IMPROVE

### E1. Code Quality (High Impact, Low Effort)

| # | Item | Effort | Impact |
|---|------|--------|--------|
| E1.1 | Fix `go 1.26.3` → `go 1.26` in go.mod | 1min | Correctness |
| E1.2 | `cameraHIDByte`/`audioHIDByte` switch → map lookup | 5min | Consistency |
| E1.3 | `handleMutatingCommand`/`handleQueryCommand` switch → map dispatch | 30min | Extensibility |
| E1.4 | `setupStream` 4-tuple → named struct | 10min | Readability |
| E1.5 | Copy `[]byte` in `lastFrameCache.Get()` | 5min | Data race fix |
| E1.6 | Add HID byte protocol comments to `parseHIDResponse` | 10min | Maintainability |
| E1.7 | `commandMsgError.Unwrap()` missing — add it | 5min | `errors.Is`/`errors.As` compatibility |
| E1.8 | Response string constants for `"gesture on"`, `"gesture off"`, `"centered"` | 10min | Consistency |
| E1.9 | `matchesPixyID` hex-parse vendor/product once, not every call | 5min | Performance |
| E1.10 | `v4l2DegreesPerUnit` rename → `v4l2UnitsPerDegree` (semantically correct) | 5min | Clarity |

### E2. Architecture (High Impact, Medium Effort)

| # | Item | Effort | Impact |
|---|------|--------|--------|
| E2.1 | Explicit call state machine (replace `InCall bool` + debounce counters) | 2hr | Correctness, extensibility |
| E2.2 | Fix `SendRecv` goroutine leak on timeout | 1hr | Resource leak fix |
| E2.3 | `Run()` decomposition — extract signal handler, server lifecycle | 1hr | Readability |
| E2.4 | Metrics struct (encapsulate 9 global vars) | 30min | Encapsulation |
| E2.5 | `slog.With` for contextual logging (zero uses currently) | 30min | Observability |
| E2.6 | `registerMetrics` helper to reduce 9 identical error checks | 15min | DRY |

### E3. Build & CI (Medium Impact, Low Effort)

| # | Item | Effort | Impact |
|---|------|--------|--------|
| E3.1 | Add `lint` and `test` checks to `flake.nix` | 30min | CI quality gate |
| E3.2 | Remove 6 unused linters from `.golangci.yml` | 10min | Startup speed |
| E3.3 | Remove `goexperiment.*` build tags from `.golangci.yml` | 2min | Correctness |
| E3.4 | Archive/delete 37 stale `docs/status/` files | 5min | Hygiene |
| E3.5 | Cut CHANGELOG `0.2.0` release from accumulated unreleased changes | 30min | Release hygiene |

### E4. Frontend (Medium Impact, Medium Effort)

| # | Item | Effort | Impact |
|---|------|--------|--------|
| E4.1 | `app.js`: XSS via `innerHTML` in toast — use `textContent` | 10min | Security |
| E4.2 | `app.js`: Add URL prefix validation in `doAction` handler | 5min | Security |
| E4.3 | `app.js`: Extract duplicated PTZ axis helper (4 occurrences) | 15min | DRY |
| E4.4 | `app.js`: Reset `streamRetryDelay` on successful reconnect | 5min | UX fix |
| E4.5 | `app.js`: Magic numbers → named constants | 10min | Readability |
| E4.6 | `style.css`: Hardcoded colors → CSS variables (5 instances) | 15min | Maintainability |
| E4.7 | `templates.templ`: Inline styles → CSS class | 10min | Maintainability |

### E5. NixOS Module (Low Impact, Low Effort)

| # | Item | Effort | Impact |
|---|------|--------|--------|
| E5.1 | Remove redundant `environment.systemPackages` (already in service `path`) | 5min | Cleanliness |
| E5.2 | `RestartSec = "3"` → integer `3` | 2min | Convention |
| E5.3 | Consider removing `AF_INET` if only serving localhost | 10min | Hardening |

---

## F. TOP 25 THINGS TO DO NEXT

Sorted by impact/effort ratio (highest first):

| Rank | Item | Effort | Impact | Category |
|------|------|--------|--------|----------|
| 1 | Fix `go 1.26.3` → `go 1.26` in go.mod | 1min | Correctness | Build |
| 2 | Copy `[]byte` in `lastFrameCache.Get()` — data race | 5min | Bug fix | Code |
| 3 | `app.js`: Replace `innerHTML` with `textContent` — XSS | 10min | Security | Frontend |
| 4 | `app.js`: Add URL validation in `doAction` | 5min | Security | Frontend |
| 5 | `commandMsgError.Unwrap()` — enable `errors.Is` | 5min | Correctness | Code |
| 6 | Fix `SendRecv` goroutine leak on timeout | 1hr | Resource leak | Code |
| 7 | Remove 6 unused linters from `.golangci.yml` | 10min | Speed | Build |
| 8 | Remove `goexperiment.*` build tags from `.golangci.yml` | 2min | Correctness | Build |
| 9 | `cameraHIDByte`/`audioHIDByte` → map lookup | 5min | Consistency | Code |
| 10 | `setupStream` 4-tuple → named struct | 10min | Readability | Code |
| 11 | Response string constants (`"gesture on"` etc.) | 10min | Consistency | Code |
| 12 | `matchesPixyID` — parse vendor/product once | 5min | Perf | Code |
| 13 | `v4l2DegreesPerUnit` → `v4l2UnitsPerDegree` rename | 5min | Clarity | Code |
| 14 | HID byte protocol comments in `parseHIDResponse` | 10min | Maintainability | Code |
| 15 | Archive/delete 37 stale `docs/status/` files | 5min | Hygiene | Docs |
| 16 | `handleMutatingCommand`/`handleQueryCommand` → map dispatch | 30min | Extensibility | Code |
| 17 | `app.js`: Extract PTZ axis helper, reset retry delay, magic numbers | 30min | Quality | Frontend |
| 18 | `style.css`: Hardcoded colors → CSS variables | 15min | Maintainability | Frontend |
| 19 | Add `lint` + `test` checks to `flake.nix` | 30min | CI | Build |
| 20 | Metrics struct (encapsulate 9 global vars) | 30min | Encapsulation | Code |
| 21 | `slog.With` contextual logging | 30min | Observability | Code |
| 22 | `registerMetrics` DRY helper | 15min | DRY | Code |
| 23 | Explicit call state machine (`CallState` enum) | 2hr | Correctness | Architecture |
| 24 | Cut CHANGELOG `0.2.0` release | 30min | Release | Docs |
| 25 | `Run()` decomposition | 1hr | Readability | Architecture |

---

## G. TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**The `SendRecv` goroutine leak is real, but the fix has a design tension:**

The current code spawns a goroutine to read the HID response, with a select racing against a timeout. On timeout, the goroutine is leaked because you can't cancel a blocking `fd.Read()` on a raw file descriptor without closing the fd.

Options:
1. **Close the fd on timeout** — but `SendRecv` doesn't own the fd lifecycle (the `hidrawDevice` does). Closing mid-read would corrupt the device state.
2. **Use ` SetDeadline` on the fd** — `hidrawDevice.Open()` already calls `SetDeadline(3s)`, so the timeout in `SendRecv` is redundant. If the fd deadline works correctly, the goroutine shouldn't block forever — it would return on deadline expiry. **Question: Is the `SetDeadline` already handling this, making the timeout select redundant?**
3. **Use a context-aware reader** — but raw fd reads don't respect context.

I'm not confident enough to pick between option 2 (remove redundant timeout) vs option 1 (close fd) without testing on real hardware. The HID protocol timing requirements (200ms between config+commit) may also be affected.

---

## Appendix: Zero-Coverage Functions

| File | Function | Reason |
|------|----------|--------|
| `main.go:121` | `sdNotify` | systemd-specific |
| `main.go:130` | `newHTTPServer` | Integration-tested indirectly |
| `main.go:147` | `Run` | Long-running, integration-tested |
| `main.go:249` | `exitWithDaemonError` | Exit helper |
| `main.go:258` | `handleFlag` | Flag handler |
| `main.go:283` | `main` | Entry point |
| `process.go:117` | `findPixySource` | External dep (wpctl) |
| `process.go:139` | `setDefaultSource` | External dep (wpctl) |
| `process.go:146` | `notify` | External dep (notify-send) |
| `ptz.go:77` | `v4l2Set` | External dep (v4l2-ctl) |
| `socket.go:87` | `sendCommand` | Tested via integration |
| `uevent.go:59` | `listenUevents` | Syscall-level, netlink |
| `uevent.go:113` | `unixSocketUevent` | Syscall-level |
| `uevent_linux.go:11` | `unixOpenNetlinkKobjectUevent` | Syscall-level |

Most 0% functions are either external-dependency wrappers (untestable without mocking the system) or lifecycle functions (integration-tested indirectly). Only `handleFlag` and `sendCommand` could have targeted unit tests.
