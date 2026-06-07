# Quality Sweep — Round 2

**Date:** 2026-05-01  
**Status:** In Progress

## Honest Retrospective

### What we got right

- ConfigFromEnv() with proper validation and tests
- Function field injection for testability (isCameraInUseFn, findSourceFn, setSourceFn, notifyFn)
- Metrics init() race fix — clean parallel tests with -race
- CommandError.Op rename, testConfig fix, t.Parallel() sweep

### What we missed

1. **NixOS ghost options**: `autoTracking`, `autoPrivacy`, `defaultAudio` are declared but never passed to daemon. Users configure them thinking they work.
2. **integration_test.go has its own Daemon constructor**: `newIntegrationDaemon()` duplicates `newTestDaemon()`, missing function fields, missing `WebAddr`
3. **Zero-assertion tests**: `TestAutoManage_UpdatesMetrics` and `TestAutoManage_SavesStateAfterRun` call functions without verifying anything
4. **Goroutine leak on shutdown**: `listenUevents` has no cancellation — blocks on `fd.Read()` forever when daemon stops
5. **I/O under mutex**: `probeDevices()` and `saveState()` do filesystem I/O while `d.mu.Lock()` is held — daemon blocks on disk during state operations
6. **Duplicated device matching**: `process.go:116` checks `["EMEET", "Pixy", "PIXY"]` while `probe.go:105` uses `hasPixyVendorProduct` — different strategies for the same device

### What could still improve

- `Run()` is 101 lines — signal setup + socket + HTTP + event loop in one function
- `handleStream` in handlers.go mixes ffmpeg lifecycle + frame extraction + HTTP writing
- `getStatus` and `getWebStatus` duplicate state reading + PTZ queries
- `Config` doesn't store `AutoMode` or `DefaultAudio` from NixOS — these are runtime state, not config

---

## Execution Plan

Sorted by **impact × ease** (high impact + easy = do first).

### Phase 1: Ghost System Fixes (HIGH impact, LOW effort)

| #   | Task                                                                                | Files                              | Est   |
| --- | ----------------------------------------------------------------------------------- | ---------------------------------- | ----- |
| 1.1 | Wire NixOS `autoTracking`/`autoPrivacy`/`defaultAudio` to daemon env vars           | `modules/nixos.nix`                | 5min  |
| 1.2 | Add `AutoMode` and `DefaultAudio` to `Config`, read from env in `ConfigFromEnv()`   | `internal/pixy/pixy.go`, `main.go` | 10min |
| 1.3 | Apply initial config state in `NewDaemon()` (auto mode + default audio from config) | `main.go`                          | 5min  |
| 1.4 | Tests for new config fields                                                         | `internal/pixy/pixy_test.go`       | 5min  |

### Phase 2: Test Quality (HIGH impact, LOW effort)

| #   | Task                                                                                        | Files                 | Est   |
| --- | ------------------------------------------------------------------------------------------- | --------------------- | ----- |
| 2.1 | Consolidate `newIntegrationDaemon()` → use `newTestDaemon()` with `testConfig(t.TempDir())` | `integration_test.go` | 10min |
| 2.2 | Add real assertions to `TestAutoManage_UpdatesMetrics`                                      | `auto_test.go`        | 5min  |
| 2.3 | Add real assertions to `TestAutoManage_SavesStateAfterRun`                                  | `auto_test.go`        | 5min  |

### Phase 3: Goroutine Leak Fix (MEDIUM impact, LOW effort)

| #   | Task                                                         | Files                  | Est   |
| --- | ------------------------------------------------------------ | ---------------------- | ----- |
| 3.1 | Add context parameter to `listenUevents`, close fd on cancel | `uevent.go`, `main.go` | 10min |

### Phase 4: I/O Under Lock (MEDIUM impact, MEDIUM effort)

| #   | Task                                                                     | Files                            | Est   |
| --- | ------------------------------------------------------------------------ | -------------------------------- | ----- |
| 4.1 | Extract `probeDevices()` to return values instead of mutating under lock | `probe.go`, `main.go`, `auto.go` | 15min |
| 4.2 | Extract state snapshot helper to avoid lock-copy boilerplate             | `main.go`                        | 10min |

### Phase 5: Duplicated Logic Consolidation (MEDIUM impact, LOW effort)

| #   | Task                                            | Files                    | Est  |
| --- | ----------------------------------------------- | ------------------------ | ---- |
| 5.1 | Extract device name matching to shared function | `probe.go`, `process.go` | 5min |
| 5.2 | Extract HTTP server constants                   | `main.go`                | 5min |

---

## Decisions

### Why NOT add koanf/gin/lo/samber-do?

- **koanf**: We have 5 env vars. `ConfigFromEnv()` is 30 lines. koanf adds a dependency for zero gain.
- **gin**: 15 endpoints on localhost. `net/http` + `http.ServeMux` is the right choice.
- **samber/lo**: Go 1.26 has `slices`/`maps` stdlib. Not needed.
- **samber/do**: 1 struct, ~6K LOC. DI framework is absurd overkill.

### Type model decisions

- `AutoMode` and `DefaultAudio` belong in `Config`, not `State`. `State` is runtime (camera/inCall), `Config` is startup policy.
- No new types needed — reuse existing `bool` and `AudioMode`.
