# emeet-pixyd — Refactoring & Quality Status

**Date:** 2026-04-30 20:28
**Status:** IN PROGRESS — Phase 1 complete, Phase 2 actionable

---

## Executive Summary

Session 1 (20:04) extracted emeet-pixyd from SystemNix into standalone project. Session 2 (current) performed major refactoring: broke `main.go` from 882 → 599 lines, fixed test bugs, cleaned dead code. Tests pass, all pushed. Linter shows 68 issues in production code (mostly gosec false-positives for a hardware daemon). Real actionable issues identified below.

---

## A) FULLY DONE

| #  | Item                                     | Commit    | Details                                                                                                                            |
| -- | ---------------------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Extract `webStatus` from templates.templ | `f7c4464` | Moved Go struct to `web_types.go` — templates should only contain HTML                                                             |
| 2  | Extract device probing to `probe.go`     | `a2d9222` | `probeVideo4linux`, `probeHidraw`, `hasPixyVendorProduct`, `Daemon.probeDevices`                                                   |
| 3  | Extract state persistence to `state.go`  | `7892caa` | `loadState`, `saveState`, `ensureStateDir`, `stateSetter` type                                                                     |
| 4  | Extract auto-manage to `auto.go`         | `4cb1168` | `handleCallStart`, `handleCallEnd`, `autoManage` with debounce                                                                     |
| 5  | Fix CI with GOWORK=off                   | `7ca3a43` | Tests were broken locally due to parent `go.work` not including this project                                                       |
| 6  | Normalize `new()` → `ptr()` in tests     | `a6d8c94` | Consistent pointer literal creation in `integration_test.go`                                                                       |
| 7  | Consolidate test daemon builders         | `17c23ab` | Fixed **missing `streamSema` bug** in `testDaemonBase`/`testDaemonWithState`; unified into `newTestDaemon` with functional options |
| 8  | Update AGENTS.md                         | `b0a428d` | Documented refactored file structure, GOWORK gotcha, test patterns                                                                 |
| 9  | All tests pass                           | —         | `GOWORK=off go test -race -count=1 ./...` — all green                                                                              |
| 10 | All pushed to remote                     | —         | 8 commits pushed to `master`                                                                                                       |

---

## B) PARTIALLY DONE

| # | Item              | Status              | What's Left                                                                                                                                                |
| - | ----------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Linter compliance | 68 issues remain    | Most are gosec false-positives (G304/G204 for a hardware daemon), revive doc comments, and test-only errcheck. Real production issues: ~15. See section E. |
| 2 | AGENTS.md         | Created and updated | Could still add more gotchas as we discover them during lint fixes                                                                                         |

---

## C) NOT STARTED

| #  | Item                                            | Effort | Impact | Notes                                                                                                                                                                                                                            |
| -- | ----------------------------------------------- | ------ | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Fix linter issues (production code)             | 1hr    | High   | ~15 real issues: goconst, staticcheck, perfsprint, unused param, revive docs                                                                                                                                                     |
| 2  | Fix linter issues (test code)                   | 30min  | Medium | ~53 issues: errcheck on Close/Write, paralleltest, exhaustruct                                                                                                                                                                   |
| 3  | Reduce cyclomatic complexity                    | 2hr    | Medium | 9 functions exceed cyclop threshold (10): `handleCommand`(20), `Run`(16), `syncState`(15), `handleStream`(17), `parseHIDResponse`(14), `autoManage`(12), `extractJPEGFrame`(12), `isCameraInUse`(12), `assertWebStatusField`(21) |
| 4  | Move HID constants to `internal/pixy`           | 30min  | Medium | Protocol bytes (`hidByteTracking`, etc.) live in `hid.go` main package — should be shared                                                                                                                                        |
| 5  | Replace `v4l2-ctl` subprocess with native ioctl | 2hr    | High   | Eliminates PATH dependency, faster, already have `golang.org/x/sys/unix` imported                                                                                                                                                |
| 6  | Replace `wpctl`/`notify-send` subprocess calls  | 1hr    | Medium | PipeWire API via D-Bus, libnotify via D-Bus                                                                                                                                                                                      |
| 7  | Add `golangci-lint` to CI                       | 5min   | High   | `.golangci.yml` exists but CI only runs `go vet`                                                                                                                                                                                 |
| 8  | Add `goreleaser` for releases                   | 30min  | Medium | Automate versioned GitHub releases                                                                                                                                                                                               |
| 9  | Add `/healthz` endpoint                         | 5min   | Medium | Trivial addition, useful for monitoring                                                                                                                                                                                          |
| 10 | JSON structured logging                         | 15min  | Medium | Already uses `slog`, just needs JSON handler config option                                                                                                                                                                       |
| 11 | Config from env/flags                           | 30min  | Low    | `koanf` or simple env parsing                                                                                                                                                                                                    |
| 12 | Metrics registration refactor                   | 15min  | Low    | Move `init()` Prometheus registration to explicit setup                                                                                                                                                                          |

---

## D) TOTALLY FUCKED UP

**Nothing.** All 8 commits built, tested, and pushed cleanly.

**Close call during refactoring:** The `multiedit` tool mangled `main.go` when extracting probe functions — it merged `loadState()` body into `setDeviceState()` signature. Fixed by reading the corrupted output and applying a targeted correction. Lesson: for large extractions, use `write` for new file + single `edit` for removal, not `multiedit` for the removal.

---

## E) WHAT WE SHOULD IMPROVE

### Production Code Linter Issues (Actionable)

**High-priority (real bugs/quality):**

| File                  | Line                                | Issue                                                   | Fix                                                           |
| --------------------- | ----------------------------------- | ------------------------------------------------------- | ------------------------------------------------------------- |
| `handlers.go:39`      | `ffmpegShutdownSecs`                | staticcheck ST1011: duration var named with unit suffix | Rename to `ffmpegShutdown`                                    |
| `handlers.go:315`     | `handleSnapshot`                    | revive: unused `request` parameter                      | Rename to `_`                                                 |
| `handlers.go:474-505` | `extractJPEGFrame`                  | nestif(6), cyclop(12)                                   | Extract SOI scan and payload scan into helpers                |
| `commands.go:69,75`   | `"toggle-gesture"`, `"toggle-auto"` | goconst: 3 occurrences each                             | Extract to constants like `cmdToggleGesture`, `cmdToggleAuto` |
| `uevent.go:66`        | `fd.Close()`                        | errcheck: unchecked error                               | Use `defer func() { _ = fd.Close() }()`                       |
| `uevent.go:93`        | `uintptr(fd)`                       | gosec G115: potential int overflow                      | Add bounds check or nolint with justification                 |
| `probe.go:29,94`      | `fmt.Sprintf("/dev/%s", name)`      | perfsprint: use concatenation                           | `"/dev/" + name`                                              |
| `v4l2.go:38`          | `args := []string{...}`             | prealloc: known size                                    | `make([]string, 0, 2+len(controls))`                          |
| `handlers.go:52-70`   | `metricInCall`, `init()`            | gochecknoglobals, gochecknoinits                        | Move to explicit `registerMetrics()` called from `Run()`      |

**Medium-priority (style/doc):**

| File                               | Issue                                        | Count | Fix                      |
| ---------------------------------- | -------------------------------------------- | ----- | ------------------------ |
| `internal/pixy/pixy.go`            | Missing doc comments on exported types/funcs | 14    | Add `// ...` comments    |
| `auto.go`, `internal/pixy/pixy.go` | Missing package comments                     | 2     | Add `// Package ...` doc |

### Architecture Debt

1. **`handleCommand` has cyclomatic complexity 20** — giant switch statement. Should be a command dispatch table (map of command name → handler func).

2. **`Run()` has complexity 16** — mixes lifecycle, signal handling, ticker, uevent, HTTP server. Signal handling and HTTP server startup could be extracted.

3. **`syncState` has complexity 15** — three parallel HID queries with per-result error handling. Could use errgroup.

4. **`Daemon` struct still owns too much** — 5 mutex locks, 4 caches/counters, device paths, state, config. Consider splitting into `DeviceManager`, `StateManager`, `CallDetector`.

5. **No structured error types** — commands return `"error: ..."` string prefixes. Should use a `CommandError` type.

### Type Model Improvements

1. **`CameraState` and `AudioMode` are string enums with hand-rolled methods** — `Valid()`, `String()`, `Parse*()`, `Next()`. A code generator like `github.com/abice/go-enum` would eliminate boilerplate while keeping the same interface.

2. **`webStatus` duplicates `pixy.State`** — `webStatus.Camera` is `string` while `pixy.State.Camera` is `pixy.CameraState`. The web layer loses type safety by converting to strings immediately. Consider embedding or using the typed values.

3. **HID protocol types are untyped byte constants** — `0x09`, `0x01`, `0x05` etc. should be typed constants or an enum. The `pixyConfig` and `pixyCommit` builders work with raw bytes.

### Library Opportunities

1. **`errgroup` for parallel HID queries** — `syncState` does 3 sequential queries that could be concurrent.
2. **`github.com/abice/go-enum`** — generate enum boilerplate for `CameraState`, `AudioMode`.
3. **`github.com/nicklaw5/helix` or similar** — not needed, this is a hardware daemon.
4. **Native V4L2 ioctl via `golang.org/x/sys/unix`** — already imported for netlink, could extend for V4L2.

---

## F) Top 25 Things to Get Done Next

Sorted by **impact / effort** (highest ROI first):

| #  | Task                                                                   | Effort | Impact | Why                                            |
| -- | ---------------------------------------------------------------------- | ------ | ------ | ---------------------------------------------- |
| 1  | Add `golangci-lint` to CI workflow                                     | 5min   | High   | `.golangci.yml` exists but isn't used in CI    |
| 2  | Fix staticcheck ST1011: rename `ffmpegShutdownSecs` → `ffmpegShutdown` | 1min   | Low    | Trivial, linter clean                          |
| 3  | Fix unused `request` param in `handleSnapshot`                         | 1min   | Low    | Trivial                                        |
| 4  | Extract `"toggle-gesture"` / `"toggle-auto"` to constants              | 2min   | Medium | goconst, prevent typos                         |
| 5  | Fix unchecked `fd.Close()` in uevent.go                                | 1min   | Low    | Defensive                                      |
| 6  | Replace `fmt.Sprintf("/dev/%s", name)` with concatenation              | 1min   | Low    | perfsprint                                     |
| 7  | Preallocate `v4l2.go` args slice                                       | 1min   | Low    | prealloc                                       |
| 8  | Add doc comments to `internal/pixy` exported types                     | 10min  | Medium | Linter clean, godoc                            |
| 9  | Add package comments to `auto.go`, `internal/pixy`                     | 2min   | Low    | Linter clean                                   |
| 10 | Refactor `handleCommand` to dispatch table (reduce complexity 20→5)    | 30min  | High   | Most complex function in codebase              |
| 11 | Extract `syncState` into parallel errgroup queries                     | 15min  | Medium | Faster sync, cleaner code                      |
| 12 | Move Prometheus metrics registration out of `init()`                   | 15min  | Medium | Testable, no global state                      |
| 13 | Add `handleStream` context propagation for `templ.Handler`             | 15min  | Medium | contextcheck: all 10 warnings are this pattern |
| 14 | Refactor `extractJPEGFrame` nested ifs into helper funcs               | 15min  | Medium | Reduces nestif and cyclop                      |
| 15 | Extract signal handling from `Run()` into `handleSignals()`            | 15min  | Medium | Reduces Run() complexity                       |
| 16 | Move HID protocol constants to `internal/pixy/hid.go`                  | 30min  | Medium | Shared types for tests and main                |
| 17 | Add `CommandError` type instead of `"error: "` string prefix           | 30min  | Medium | Type safety in command layer                   |
| 18 | Replace `v4l2-ctl` subprocess with native ioctl                        | 2hr    | High   | Eliminates runtime PATH dependency             |
| 19 | Add `/healthz` endpoint                                                | 5min   | Medium | Useful for systemd watchdog or monitoring      |
| 20 | Add JSON structured logging option                                     | 15min  | Medium | Better production observability                |
| 21 | Use `errgroup` for concurrent HID queries in `syncState`               | 15min  | Medium | Performance + clarity                          |
| 22 | Consolidate `webStatus` with `pixy.State` (typed, not string)          | 30min  | Medium | Eliminate stringly-typed web layer             |
| 23 | Add `goreleaser` for versioned GitHub releases                         | 30min  | Medium | Distribution                                   |
| 24 | Replace `wpctl` subprocess with PipeWire D-Bus API                     | 1hr    | Medium | Eliminate another PATH dependency              |
| 25 | Add snapshot endpoint tests with mock frame data                       | 15min  | Low    | Coverage gap                                   |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we suppress gosec G304/G204 warnings or refactor them?**

The daemon legitimately opens `/dev/hidraw*`, `/dev/video*`, and launches `ffmpeg`/`v4l2-ctl`/`wpctl` subprocesses — that's its entire purpose. 21 gosec warnings are all about "file inclusion via variable" and "subprocess launched with variable" which are inherent to a hardware control daemon. Suppressing via `//nolint:gosec` is reasonable, but I'd like your call on whether to:

1. **Suppress globally** in `.golangci.yml` for G304 and G204 (pragmatic — these will never be fixed)
2. **Suppress per-site** with `//nolint:gosec` comments (explicit but noisy)
3. **Refactor** to validated path construction (overkill for a single-purpose daemon)

---

## Metrics

### Code Size

| File                  | Lines     | Role                                            |
| --------------------- | --------- | ----------------------------------------------- |
| `main.go`             | 599       | Daemon struct, lifecycle, status, socket, Run() |
| `main_test.go`        | 1,128     | Unit tests                                      |
| `integration_test.go` | 935       | Web + socket integration tests                  |
| `handlers.go`         | 601       | HTTP, metrics, MJPEG streaming                  |
| `templates_templ.go`  | 779       | Generated HTML                                  |
| `commands.go`         | 255       | Command routing                                 |
| `hid.go`              | 266       | HID protocol                                    |
| `auto.go`             | 121       | Call detection + auto-manage                    |
| `probe.go`            | 129       | Device discovery                                |
| `state.go`            | 67        | State persistence                               |
| `process.go`          | 144       | /proc scanning, PipeWire, notifications         |
| `v4l2.go`             | 84        | V4L2 PTZ control                                |
| `uevent.go`           | 94        | Uevent listener                                 |
| `web_types.go`        | 20        | webStatus struct                                |
| `internal/pixy/`      | 606       | Shared types + tests                            |
| **Total**             | **6,535** |                                                 |

### Linter Summary

- **Total issues:** 573 (68 in production code, 505 in test code)
- **Production issues by real severity:**
  - Bugs/quality: ~8 (staticcheck, unused param, errcheck)
  - Style/docs: ~16 (revive comments, perfsprint, prealloc)
  - False positives: ~21 (gosec G304/G204 for hardware daemon)
  - Design: ~10 (cyclop complexity, goconst, gochecknoglobals)
  - Other: ~13 (contextcheck for templ pattern)

### Refactoring Impact

| Metric                    | Before         | After       | Change           |
| ------------------------- | -------------- | ----------- | ---------------- |
| `main.go` lines           | 882            | 599         | **-32%**         |
| Production files          | 11             | 15          | +4 focused files |
| Test builder duplication  | 5 constructors | 1 + options | **-80%**         |
| Missing `streamSema` bugs | 2 constructors | 0           | **Fixed**        |
| Linter issues addressed   | 0              | ~12         | Partial          |
