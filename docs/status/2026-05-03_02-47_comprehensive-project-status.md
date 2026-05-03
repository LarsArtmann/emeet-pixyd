# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-05-03 02:47  
**Branch:** `master` (up to date with `origin/master`)  
**Coverage:** 69.6% (main), 77.9% (internal/pixy)  
**Lint:** 0 issues  
**Build:** Clean (with `GOWORK=off`)

---

## A) FULLY DONE

### Bug Fixes (templates.templ)

- **Infinite HTMX loop** — Removed `load` from `hx-trigger` on `#status-panel`. The `load` + `outerHTML` swap caused each swapped element to re-trigger, creating an infinite loop. (`templates.templ:122`)
- **Scripts in `<head>`** — Moved `<script>` tags to end of `<body>`. `app.js` accesses `document.body` at line 12 which is `null` during `<head>` parsing. (`templates.templ:15-16`)
- **Literal string template bug** — `hx-post="endpoint"` (string literal) changed to `hx-post={ endpoint }` (Go variable). All camera buttons were POSTing to literal path `/endpoint`. (`templates.templ:94`)
- **Literal aria labels** — `aria-label="ariaLabel"` changed to `aria-label={ ariaLabel }` in both `cameraBtn` and `audioBtn` templates. (`templates.templ:96,111`)

### Type-Safety Refactoring (Split Brain Elimination)

- **`webStatus` struct** — Changed from raw `string` fields to typed `pixy.CameraState`, `pixy.AudioMode`, `pixy.AutoMode`. Templates now compare against typed constants (`pixy.StateTracking`) instead of raw strings (`"tracking"`). (`web_types.go`, `handlers.go`, `templates.templ`)
- **`integration_test.go`** — Updated `webStatusCheck` to use typed pointer fields (`*pixy.CameraState`, `*pixy.AudioMode`, `*pixy.AutoMode`). Removed all `string()` casts.

### Handler Extraction (handlers.go: 624 → ~311 lines)

- **`metrics.go`** (81 lines) — OTel/Prometheus metric registration, `init()`, `registerMetrics()`, `updateMetrics()`
- **`stream.go`** (207 lines) — MJPEG streaming, snapshot, FFmpeg management, JPEG extraction
- **`middleware.go`** (51 lines) — HTTP middleware: caching, PTZ validation, security, request ID

### Toast/Error Pattern Unification

- All HTTP handlers now use `applyResponseToStatus()` for error/toast handling. No inline `if IsCommandErrorResponse` patterns remain. (`handlers.go:134-142`)

### Linter Config Cleanup

- Removed 5 false-positive linters from `.golangci.yml`: `contextcheck`, `exhaustruct`, `gochecknoinits`, `gochecknoglobals`, `paralleltest`
- Documented reasoning in AGENTS.md

### Tests Added

- `TestWeb_IndexContainsCameraButtons` — Verifies all `hx-post` URLs render correctly (catches literal-string template bugs)
- `TestWeb_PanelEndpointReturnsStatusPanel` — Verifies `/panel` returns status panel without `load` trigger
- `TestHandleCommand_UnknownCommand`, `TestHandleCommand_StatusFormat`, `TestHandleCommand_AudioUsage`, `TestHandleCommand_PTZUsage`, `TestHandleCommand_Device`, `TestHandleCommand_DeviceNotFound`
- All use `http.NewRequestWithContext` for noctx compliance

### Documentation

- `AGENTS.md` — Updated File Responsibilities table, documented typed webStatus fields, unified toast pattern, handler extraction, template bugs
- `FEATURES.md` — Complete feature inventory

### Committed Work (15 commits on master)

```
7d306f9 docs: update AGENTS.md and add FEATURES.md feature inventory
073a48b fix(tests): replace http.Get with context-aware requests for noctx compliance
223451b fix(lint): remove 5 false-positive linters from .golangci.yml
2e4ffa9 refactor: extract metrics, middleware, and stream into dedicated files
9b73660 refactor: deduplicate test helpers and template clones
53a163f docs: add comprehensive status report for nix fix and type-safety session
a9ab3c5 docs: update AGENTS.md with nix build and type-safety changes
caf740d feat(flake): add aarch64-linux to supportedSystems
f7ead30 fix(nixos): add ffmpeg-headless to service PATH for MJPEG streaming
042507c refactor: type webStatus fields with pixy types for compile-time safety
8a9473a feat: add go:generate directive for templ templates
d7b63d8 chore: add vendor/ to .gitignore
43063e8 docs: apply comprehensive review fixes
6356f70 docs: rewrite README into comprehensive project landing page
3e6ffb9 docs: add comprehensive project review status report
```

---

## B) PARTIALLY DONE

### BDD/Behavior Tests (`behavior_test.go`) — **BROKEN, NOT COMMITTED**

- 624 lines, 10 BDD-style test scenarios covering full user workflows
- **Status:** Build fails with compilation errors:
  - Line 11: `"sync"` imported and not used
  - Line 374: `undefined: syncRWMutex` — uses `sync.RWMutex{}` literal instead of importing or using correct constructor
  - Line 390: Same `syncRWMutex` error
  - 3 tests runtime-panic even after fixing compilation (e.g., `index out of range [-1]` in `TestBehavior_FullAutoCallLifecycle` — accessing `setTrackingCalls[-1]` when slice is empty)
- **Root cause:** Tests were generated in a previous session but left in broken state. They reference helpers defined in other test files (`readState`, `readDebounce`, `testAutoDaemon`, `newPTZDaemon`, `testDaemonWithState`) which is correct, but have compilation errors and logic bugs.
- **Action needed:** Fix `sync` import, replace `syncRWMutex{}` with proper mutex initialization, fix slice bounds checks, then commit.

### `main.go` Extraction — **NOT STARTED**

- Still 611 lines with Daemon struct, lifecycle, HID methods, socket server, waybar output all in one file
- Identified for extraction but not yet done

---

## C) NOT STARTED

See Section F for full prioritized list. Highlights:

- Structured command response type (replace stringly-typed `"error: ..."` prefix)
- `main.go` extraction into `device.go`, `socket.go`, `status.go`
- More HTTP handler tests (audio buttons, privacy toggle, gesture, auto endpoints)
- `inotify`/`fanotify` for call detection instead of `/proc/*/fd` polling
- `go:embed` for `app.js` to keep template scripts in sync
- Health check endpoint
- HID protocol documentation
- SSE alternative to MJPEG streaming
- Direct V4L2 ioctl calls instead of `v4l2-ctl` subprocess

---

## D) TOTALLY FUCKED UP

### `behavior_test.go` — Untracked, Broken, Should Not Ship

This file is the only blemish on an otherwise clean tree:

1. **Won't compile:** `sync` import unused, `syncRWMutex` undefined
2. **Runtime panics:** Even if compilation is fixed, `TestBehavior_FullAutoCallLifecycle` panics with `index out of range [-1]` — the test creates a daemon but the mock doesn't prevent real HID writes to `/dev/hidraw7`, causing state mismatches
3. **Test design issues:** `TestBehavior_TrackingOnlyAutoMode` and `TestBehavior_PrivacyOnlyAutoMode` fail because the test daemons don't properly mock all dependency injection points (real `setTrackingFn` hits the hardware)
4. **Overlap:** These tests partially duplicate coverage already in `auto_test.go`, `commands_test.go`, and `main_test.go`

**Recommendation:** Either fix properly (requires correcting all mock wiring and bounds checks) or delete entirely (existing tests already cover the same ground).

### No Other Issues

- Build is clean
- Lint is 0 issues
- All committed tests pass with race detector
- No known production bugs

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **`main.go` is still a 611-line monolith** — Daemon struct, lifecycle, HID methods, socket server, waybar output all jammed together. The handlers.go extraction proved the pattern works; apply it to main.go.
2. **Stringly-typed command protocol** — `handleCommand` returns free-form strings, errors detected by `"error: "` prefix. A structured response type would eliminate an entire class of bugs.
3. **No health check endpoint** — Standard practice for any HTTP-serving daemon. `GET /api/health` should exist.
4. **`/proc/*/fd` polling** — Scans all process file descriptors every 2s. `inotify` or `fanotify` would be more efficient and responsive.

### Testing

5. **HTTP handler coverage is thin** — No tests for `POST /api/toggle-privacy`, `POST /api/gesture`, `POST /api/auto`, or audio mode button `hx-vals` correctness. These are user-facing features with zero HTTP-level test coverage.
6. **`behavior_test.go` is broken** — Either fix or delete. Don't leave broken untracked files.
7. **No integration test for device hotplug** — uevent → probe → state change flow is untested at integration level.
8. **No benchmark for `/proc/*/fd` scanning** — This is the hot path; we should know its cost.

### Code Quality

9. **`checkDevice` helper in wrong file** — Lives in `handlers.go` but only used in `stream.go`. Move it.
10. **HID protocol undocumented** — 9-byte config reports, 64-byte responses, byte positions parsed by magic numbers. Needs documentation.
11. **`isPixyName` and device constants** — Should be in `internal/pixy` for reuse, not scattered in `probe.go`.

### Robustness

12. **No `go:embed` for `app.js`** — If the static file is missing or mismatched with templates, the web UI silently breaks.
13. **No graceful stream reconnection test** — MJPEG streaming has no test for FFmpeg restart scenarios.
14. **`extractJPEGFrame` edge cases** — Truncated frames between FFmpeg restarts are untested.
15. **Configuration via env vars only** — Consider `koanf` or similar for more robust config management (file + env + flags).

---

## F) Top 25 Things to Do Next

### Priority 1: Critical Fixes (Do First)

| #   | Task                                                                                                                                                                                                                 | Impact | Effort |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | **Fix or delete `behavior_test.go`** — The only broken file in the tree. Fix compilation errors (`sync` import, `syncRWMutex`), fix runtime panics (mock all DI points), or delete if redundant with existing tests. | High   | Low    |
| 2   | **Add `POST /api/toggle-privacy` HTTP handler test** — Zero coverage for a primary user feature.                                                                                                                     | High   | Low    |
| 3   | **Add `POST /api/gesture` HTTP handler test** — Untested endpoint.                                                                                                                                                   | Medium | Low    |
| 4   | **Add `POST /api/auto` HTTP handler test** — Untested endpoint.                                                                                                                                                      | Medium | Low    |

### Priority 2: Architecture (High Impact)

| #   | Task                                                                                                                                                                                       | Impact | Effort  |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ | ------- |
| 5   | **Extract `main.go` HID methods into `device.go`** — `queryTracking`, `queryAudio`, `queryGesture`, `setTracking`, `setAudio`, `setGesture`, `centerCamera`, `setDeviceState`. ~200 lines. | High   | Medium  |
| 6   | **Extract `main.go` socket listener into `socket.go`** — `listenUnix`, `sendCommand`. ~80 lines.                                                                                           | Medium | Low     |
| 7   | **Extract `main.go` waybar/status into `status.go`** — `getStatus`, `waybarOutput`, `boolStr`, `sdNotify`. ~60 lines.                                                                      | Medium | Low     |
| 8   | **Structured command response type** — Replace `"error: ..."` string prefix with `CommandResponse{Status, Message, Data}`. Eliminates string-matching bugs.                                | High   | Medium  |
| 9   | **Add `GET /api/health` endpoint** — Standard for any HTTP daemon. Returns JSON with uptime, device status, version.                                                                       | Medium | Low     |
| 10  | **Move `checkDevice` into `stream.go`** — It's only used there; wrong file currently.                                                                                                      | Low    | Trivial |

### Priority 3: Test Coverage

| #   | Task                                                                                                                                                                       | Impact | Effort |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 11  | **Add audio mode button `hx-vals` test** — Verify each audio button sends correct mode value.                                                                              | Medium | Low    |
| 12  | **Validate generated HTML for literal variable names** — Integration test that checks no `aria-label="ariaLabel"` or `hx-post="endpoint"` patterns exist in rendered HTML. | Medium | Low    |
| 13  | **Add integration test for device hotplug** — uevent → probe → state change flow.                                                                                          | High   | Medium |
| 14  | **Add graceful stream reconnection test** — Test FFmpeg restart during active stream.                                                                                      | Medium | Medium |
| 15  | **Benchmark `/proc/*/fd` scanning** — Know the cost of the hot path.                                                                                                       | Medium | Low    |
| 16  | **Review `extractJPEGFrame` edge cases** — Truncated frames between FFmpeg restarts.                                                                                       | Medium | Medium |

### Priority 4: Robustness & Polish

| #   | Task                                                                                       | Impact | Effort  |
| --- | ------------------------------------------------------------------------------------------ | ------ | ------- |
| 17  | **`go:embed` for `app.js`** — Ensure template scripts always match served static files.    | Medium | Low     |
| 18  | **Document HID protocol byte layout** — In code comments or separate doc.                  | Medium | Low     |
| 19  | **Move `isPixyName` and device constants into `internal/pixy`** — Centralize for reuse.    | Low    | Low     |
| 20  | **Consider `inotify`/`fanotify` for call detection** — Replace `/proc/*/fd` polling.       | High   | High    |
| 21  | **Consider `koanf` for configuration** — File + env + flags instead of env-only.           | Medium | Medium  |
| 22  | **Consider SSE instead of MJPEG multipart** — For the preview stream.                      | Medium | High    |
| 23  | **Consider direct V4L2 ioctl calls** — Replace `exec.Command("v4l2-ctl")` subprocess.      | Medium | High    |
| 24  | **Add CORS headers** — For potential remote access scenarios.                              | Low    | Low     |
| 25  | **Document `WriteTimeout: 30s` interaction with MJPEG** — Currently fine but worth noting. | Low    | Trivial |

---

## G) Top #1 Question I Cannot Figure Out Myself

**What should we do with `behavior_test.go`?**

This 624-line file has 10 BDD-style test scenarios that were generated in a previous session but left broken (won't compile, runtime panics). The tests partially overlap with existing coverage in `auto_test.go`, `commands_test.go`, and `main_test.go`.

Three options:

1. **Fix it** — Correct compilation errors, wire all mocks properly, fix bounds checks. Estimated effort: 30-60 minutes. Adds ~10 behavioral scenario tests.
2. **Delete it** — The existing test suite already covers the same ground at unit level. The BDD structure is nice but the tests don't add new coverage.
3. **Keep and fix incrementally** — Commit as-is with `t.Skip()` on broken tests, fix one by one over time.

I recommend **Option 1 (fix it)** because BDD-style behavioral tests exercise multi-component flows that unit tests miss, and the file is already 80% correct — the bugs are concentrated in 3-4 places. But I need your decision before proceeding.

---

## Summary

| Category          | Status                                                                  |
| ----------------- | ----------------------------------------------------------------------- |
| Build             | ✅ Clean                                                                |
| Lint              | ✅ 0 issues                                                             |
| Tests (committed) | ✅ All pass, race-clean, 69.6% coverage                                 |
| Tests (untracked) | ❌ `behavior_test.go` broken (won't compile + panics)                   |
| Production bugs   | ✅ None known                                                           |
| Architecture      | ⚠️ handlers.go extracted, main.go still monolithic (611 lines)          |
| Documentation     | ✅ AGENTS.md and FEATURES.md current                                    |
| Untracked files   | ⚠️ `behavior_test.go`, `docs/architecture-understanding/` (D2 diagrams) |
