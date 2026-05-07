# Strong ID Migration — Branded Types for PID & SourceID

**Date:** 2026-05-08 01:09
**Author:** Crush (GLM-5.1)
**Status:** DONE — all 3 violations fixed, tests green, lint clean (pre-existing `unparam` only)

---

## What Was Done

### Fully Done

| # | Change | Files | Impact |
|---|--------|-------|--------|
| 1 | Added `github.com/larsartmann/go-branded-id` v0.1.0 dependency | `go.mod`, `go.sum` | New dep |
| 2 | Created branded types `PID` and `SourceID` in `internal/pixy/ids.go` | `internal/pixy/ids.go` (new) | Compile-time type safety |
| 3 | `ppidOf(pid int) int` → `ppidOf(pid pixy.PID) pixy.PID` | `process.go` | Strong ID |
| 4 | `isDescendantOf(pid, ancestor int)` → branded `pixy.PID` | `process.go` | Strong ID |
| 5 | `findPixySource() (string, error)` → `(pixy.SourceID, error)` | `process.go` | Strong ID |
| 6 | `setDefaultSource(sourceID string)` → `(sourceID pixy.SourceID)` | `process.go` | Strong ID |
| 7 | DI fields `findSourceFn`/`setSourceFn` typed with branded types | `main.go` | Strong ID |
| 8 | All test files updated for branded types | `*_test.go` (7 files) | Consistency |
| 9 | Test helper consolidation (withNotifyMessages, withCaptureGestureArg, okHandler) | `main_test.go`, `auto_test.go`, `commands_test.go`, `behavior_test.go`, `handlers_test.go`, `integration_test.go` | Dedup |

### Partially Done

| # | Item | Why Partial |
|---|------|-------------|
| 1 | `behavior_test.go:TestBehavior_FullAutoCallLifecycle` still inlines `findSourceFn`/`setSourceFn`/`notifyFn` | Needs multi-capture helper (`withCaptureSourceCalls`); BDD test with specific assertions on call sequences makes extraction low-value |

### Not Started (out of scope — identified but deferred)

| # | Item | Priority | Effort |
|---|------|----------|--------|
| 1 | Brand `videoDev`/`hidrawDev` strings as `DevicePath` types | Medium | High — touches hid.go, v4l2.go, probe.go, main.go, all tests |
| 2 | Brand V4L2 control/value strings | Low | Medium |
| 3 | Unify test HTTP helpers (`get()`, `post()`, `getStream()`, `postPTZFormValue()`) | Low | Medium |
| 4 | Unify assertion helpers (`assertHTTPStatusOK` vs `assertStatusCode`, string contains/prefix) | Low | Low |
| 5 | Extract shared web-server setup from `behavior_test.go`/`stream_test.go`/`integration_test.go` | Low | Low |

### Totally Fucked Up

Nothing. Clean migration.

---

## What We Should Improve

### Architecture

1. **Device path branded types** — `videoDev`/`hidrawDev` are bare `string` throughout (hid.go, v4l2.go, probe.go, process.go, main.go). A `DevicePath` branded type would prevent accidentally passing a hidraw path where a video path is expected. HIGH impact but HIGH effort.
2. **Test helper unification** — Multiple test files duplicate HTTP helpers, assertion helpers, and daemon+server setup. A shared `testhelpers_test.go` file would reduce ~50 lines of duplication.
3. **Pre-existing `unparam` warning** — `withAutoMode` always receives `pixy.AutoOff`. The helper is used in exactly one test (`TestAutoManage_AutoOff_NoAction`). Could be inlined, but it's a test helper so low priority.

### Type Model Improvements

4. **`withFindSource` parameter type** — Currently takes `string` and wraps in `pixy.NewSourceID()`. Could take `pixy.SourceID` directly for consistency. Very low impact.
5. **V4L2 control types** — `v4l2SetFn` takes `(dev, ctrl, val string)` — `ctrl` could be an enum (`PanControl`, `TiltControl`, `ZoomControl`) and `val` could be typed. Medium impact, medium effort.

### Library Usage

6. **`go-branded-id` is correctly used** — Phantom type branding with `id.ID[B, V]` pattern. No changes needed. Library provides JSON, SQL, text, binary, gob serialization out of the box — useful if these IDs ever need to cross boundaries.

---

## Top #25 Things To Do Next (sorted by impact/effort)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Fix pre-existing `unparam` lint warning on `withAutoMode` | Low | 2min | Lint |
| 2 | Update AGENTS.md with `go-branded-id` dependency info | Med | 5min | Docs |
| 3 | Update AGENTS.md with branded type docs (PID, SourceID in ids.go) | Med | 5min | Docs |
| 4 | Create `withCaptureSourceCalls` helper, refactor `TestBehavior_FullAutoCallLifecycle` | Low | 10min | Test |
| 5 | Unify `assertHTTPStatusOK` and `assertStatusCode` | Low | 5min | Test |
| 6 | Unify string contains/prefix assertions across test files | Low | 15min | Test |
| 7 | Extract shared HTTP GET/POST helpers to reduce duplication | Low | 15min | Test |
| 8 | Extract shared daemon+web server setup from behavior/stream/integration tests | Low | 15min | Test |
| 9 | Brand `videoDev`/`hidrawDev` as `DevicePath` type | High | 60min | Types |
| 10 | Brand V4L2 control names as enum type | Med | 30min | Types |
| 11 | Add `ids_test.go` in internal/pixy for PID/SourceID type safety tests | Low | 10min | Test |
| 12 | Audit all remaining bare `string` params for branded type candidates | Med | 30min | Types |
| 13 | Consider `DevicePath` validation (must start with `/dev/`) | Med | 15min | Types |
| 14 | Add fuzz tests for `pixy.NewPID()`/`pixy.NewSourceID()` | Low | 10min | Test |
| 15 | Update nix flake hash for new go-branded-id dependency | Med | 5min | Nix |
| 16 | Run `nix build` to verify nix still builds | Med | 5min | Nix |
| 17 | Verify `nix flake check` passes | Med | 5min | Nix |
| 18 | Review `internal/pixy/pixy_test.go` for branded type test coverage | Low | 10min | Test |
| 19 | Consider extracting `ptzAxisLabel`/`ptzAxisUnit` to internal/pixy | Low | 15min | Arch |
| 20 | Consider branded type for debounce counter (prevent mixing inUse/idle) | Low | 10min | Types |
| 21 | Run full integration test suite with PIXY connected | Med | 5min | Test |
| 22 | Clean up stale gopls diagnostics (restart LSP) | Low | 1min | DX |
| 23 | Add benchmark for `pixy.PID.Equal()` vs raw `int == int` | Low | 5min | Perf |
| 24 | Consider `SourceID` validation (must be numeric string from wpctl) | Low | 10min | Types |
| 25 | Document the `go-branded-id` pattern in AGENTS.md Gotchas section | Low | 5min | Docs |

---

## Top #1 Question I Cannot Figure Out Myself

**None.** The migration was straightforward. The `go-branded-id` library's phantom type pattern maps cleanly to our use case. All 3 violations were in `process.go` and `main.go` DI fields with clear 1:1 mappings.

---

## Files Changed

```
 auto.go             |  2 +-
 auto_test.go        | 18 +++++++-----------
 behavior_test.go    | 16 +++++-----------
 commands_test.go    | 16 ++++++++--------
 go.mod              |  1 +
 go.sum              |  2 ++
 handlers_test.go    | 15 ++++++++-------
 integration_test.go | 10 ++++------
 main.go             |  4 ++--
 main_test.go        | 27 ++++++++++++++++++++++-----
 process.go          | 43 +++++++++++++++++++++++--------------------
 process_test.go     | 41 +++++++++++++++++++++++------------------
 internal/pixy/ids.go| 21 +++++++++++++++++++++ (NEW)
 12 files changed, 106 insertions(+), 89 deletions(-)
```
