# Full Code Review — 8-Skill Comprehensive Audit

**Generated:** 2026-05-07
**Reviewer:** Senior Staff+ Engineering Partner
**Scope:** All 28 Go source files (8,185 LOC excl. generated), all test files, all docs

---

## Code Quality Scan Results

| Check           | Result   |
| --------------- | -------- |
| `go build`      | ✅ Clean |
| `go vet`        | ✅ Clean |
| `golangci-lint` | 0 issues |
| `go test -race` | ✅ Pass  |
| `gofumpt`       | ✅ Clean |

### Duplication Analysis (manual)

Two remaining duplication patterns identified:

1. **PTZ axis dispatch** — `ptzAxisLabel`, `ptzAxisUnit`, `ptzAxisValue` all repeat the same `switch axis { pan/tilt/zoom }` pattern (handlers.go:267-298). Could use a lookup table.
2. **Lock-copy-unlock pattern** — Repeated ~15 times across main.go, handlers.go, auto.go. This is idiomatic Go and not a real duplication concern.

---

## Full Code Review — File-by-File Findings

### Critical Issues (0)

None. The codebase is production-ready.

### Medium Issues (5)

| #   | File           | Line    | Issue                                                                                                 | Recommendation                                         |
| --- | -------------- | ------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| M1  | main.go        | 40-49   | `lastFrame` and `ptzCache` are anonymous embedded structs                                             | Extract to named types for clarity                     |
| M2  | main.go        | 539     | `ticker` runs `autoManage` + `sdNotify` in same select case                                           | `sdNotify("WATCHDOG=1")` should run regardless of auto |
| M3  | metrics.go     | 29      | `init()` registers global metrics, tests must be serial                                               | Lazy registration or constructor injection             |
| M4  | SUPERB_ROADMAP | —       | Roadmap is stale — many items completed (pprof, .golangci.yml, structured errors, keyboard shortcuts) | Update to reflect current state                        |
| M5  | stream.go      | 108-117 | Excessive blank lines inside select/case blocks                                                       | Remove decorative whitespace                           |

### Low Issues / Nits (12)

| #   | File                    | Issue                                                                       | Recommendation                                                |
| --- | ----------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------- |
| L1  | commands.go:26          | `cmdAutoOn`/`cmdAutoOff` are legacy, `auto full`/`auto off` preferred       | Consider deprecating or documenting                           |
| L2  | handlers.go:179         | `applyResponseToStatus` always sets `ToastType = toastTypeInfo` for success | Should use the toast type from `actionToast`                  |
| L3  | handlers.go:20-21       | `zoomDefault = 100` hardcoded separately from zoom range constants          | Derive from `zoomMin`                                         |
| L4  | probe.go:20-24          | `isPixyName` uses 3 string.Contains checks                                  | Use `strings.ContainsAny` or a regex                          |
| L5  | process.go:115-126      | `findPixySource` iterates all fields for numbers — fragile                  | Parse the source ID more robustly                             |
| L6  | main.go:296-300         | `boolStr` helper could use `strconv.FormatBool`                             | Actually, the custom version returns custom strings — keep it |
| L7  | handlers.go:206         | `ptzLimits` returns (0,0) for unknown axis                                  | Should never happen; consider logging                         |
| L8  | web_types.go            | `Error` and `Toast`/`ToastType` are stringly typed                          | Could use typed toast system                                  |
| L9  | auto_test.go:313        | `TestAutoManage_UpdatesMetrics` is NOT parallel (correctly)                 | Document why clearly in the test                              |
| L10 | integration_test.go:171 | `ptr[T any](v T) *T` is a general helper in test file                       | Could move to a shared test helpers file                      |
| L11 | behavior_test.go        | BDD tests don't use ginkgo (project convention is standard testing)         | The current approach is fine — just noting                    |
| L12 | handlers.go:33-34       | `streamBufSize` and `ffmpegShutdown` are in handlers, not stream.go         | Move to stream.go with the stream code                        |

### Architectural Observations

**Strengths:**

- Excellent dependency injection via function fields — 9 injectable dependencies
- Clean separation: `internal/pixy` for shared types, root package for daemon
- Type-safe string types (`CameraState`, `AudioMode`, `AutoMode`) with `Valid()` methods
- Generic `queryHIDState[T]` — elegant use of Go generics
- Atomic state persistence with tmp+rename
- Proper lock discipline (RLock for reads, Lock for writes, always copy-then-release)
- Comprehensive test coverage: BDD + unit + integration + fuzz

**Architecture quality: 8/10.** The main structural weakness is the monolithic `main.go` (623 lines) which contains Daemon struct definition, lifecycle, socket server, signal handling, and status formatting. The road map correctly identifies decomposing `Run()` as P0.

---

## Pareto Breakdown

### 1% → 51% Impact

| Task                                               | Impact | Effort |
| -------------------------------------------------- | ------ | ------ |
| Decompose `Run()` into focused methods             | High   | 30 min |
| Move `streamBufSize`/`ffmpegShutdown` to stream.go | Low    | 5 min  |
| Fix `applyResponseToStatus` toast type bug (L2)    | Medium | 10 min |

### 4% → 64% Impact

| Task                                               | Impact | Effort |
| -------------------------------------------------- | ------ | ------ |
| Update SUPERB_ROADMAP.md to reflect completion     | Medium | 20 min |
| Extract `lastFrame`/`ptzCache` to named types (M1) | Medium | 30 min |
| Remove decorative blank lines in stream.go (M5)    | Low    | 5 min  |

### 20% → 80% Impact

| Task                                | Impact | Effort |
| ----------------------------------- | ------ | ------ |
| Lazy metrics registration (M3)      | Medium | 1 hr   |
| PTZ axis lookup table consolidation | Low    | 30 min |
| Update all docs to current state    | Medium | 1 hr   |

---

## Priority Matrix

| #   | Task                                        | Impact | Effort | Priority |
| --- | ------------------------------------------- | ------ | ------ | -------- |
| 1   | Fix `applyResponseToStatus` toast type (L2) | Medium | 10 min | **P0**   |
| 2   | Remove decorative blank lines (M5)          | Low    | 5 min  | **P0**   |
| 3   | Move stream constants to stream.go (L12)    | Low    | 5 min  | **P0**   |
| 4   | Decompose `Run()` into focused methods      | High   | 30 min | **P1**   |
| 5   | Extract `lastFrame`/`ptzCache` named types  | Medium | 30 min | **P1**   |
| 6   | Update SUPERB_ROADMAP.md                    | Medium | 20 min | **P1**   |
| 7   | Lazy metrics registration                   | Medium | 1 hr   | **P2**   |
| 8   | PTZ axis lookup table                       | Low    | 30 min | **P2**   |
| 9   | Update all project docs                     | Medium | 1 hr   | **P2**   |

---

## D2 Execution Graph

```
direction: down

"P0 Quick Wins" -> "P1 Decomposition" -> "P2 Polish"

P0_Quick_Wins: {
  shape: rectangle
  "Fix toast type bug"
  "Remove blank lines"
  "Move stream constants"
}

P1_Decomposition: {
  shape: rectangle
  "Decompose Run()"
  "Extract named types"
  "Update roadmap"
}

P2_Polish: {
  shape: rectangle
  "Lazy metrics"
  "PTZ lookup table"
  "Docs sweep"
}
```
