# 8-Skill Comprehensive Audit — Status Report

**Generated:** 2026-05-07
**Scope:** Full codebase review covering all 8 requested skills

---

## 1. Code Quality Scan

| Metric          | Result               |
| --------------- | -------------------- |
| `go build`      | ✅ Clean             |
| `go vet`        | ✅ Clean             |
| `golangci-lint` | 0 issues             |
| `go test -race` | ✅ Pass (2 packages) |
| `gofumpt`       | ✅ Clean             |

**Duplication findings:** Minimal. The only actionable duplication is the PTZ axis dispatch pattern repeated in 3 switch statements (handlers.go:267-298). The lock-copy-unlock pattern repeated ~15 times is idiomatic Go.

---

## 2. Full Code Review

Reviewed all 28 Go source files (8,185 LOC) and all test files.

### Findings Summary

| Severity | Count | Examples                                                                                                           |
| -------- | ----- | ------------------------------------------------------------------------------------------------------------------ |
| Critical | 0     | —                                                                                                                  |
| Medium   | 5     | Anonymous embedded structs, stale roadmap, decorative whitespace, init() global state, WATCHDOG tied to autoManage |
| Low      | 12    | Toast type bug, hardcoded zoom default, PTZ axis consolidation, stream constants location                          |

### Top 3 Actionable Items

1. **`applyResponseToStatus` toast type bug** (handlers.go:179) — Always uses `toastTypeInfo` for success, ignores the type returned by `actionToast`. 10-minute fix.
2. **Move stream constants** (handlers.go:33-34) — `streamBufSize` and `ffmpegShutdown` belong in `stream.go`. 5-minute fix.
3. **Remove decorative blank lines** (stream.go:108-117) — Excessive whitespace inside select/case blocks. 5-minute fix.

See `docs/planning/2026-05-07_20-47_full-review-8-skill-audit.md` for the complete file-by-file analysis.

---

## 3. Features Audit

**FEATURES.md verified current.** All 43 features match the code. Updated timestamp to 2026-05-07.

- Total features: 43
- Fully functional: 43
- Partially functional: 0
- Broken: 0

---

## 4. Architecture Review

### Modularity Score: 8/10

**Strengths:**

- Clean package boundary: `internal/pixy` holds all shared types, config, and IPC helpers
- Excellent dependency injection: 9 function fields on Daemon for test injectability
- Type-safe string types with `Valid()` methods prevent invalid states
- Generic `queryHIDState[T]` eliminates repetitive HID query code
- Proper lock discipline with consistent copy-then-release pattern

**Weaknesses:**

- `main.go` (623 lines) is the largest file — contains Daemon struct, lifecycle, socket server, signal handling, and status formatting
- `metrics.go` uses `init()` for global metric registration, forcing serial tests
- `probeDevices()` still mutates Daemon fields directly rather than returning values

### Scalability

This is a single-device hardware daemon — scalability is inherently limited to one PIXY camera. The architecture is appropriate for its scope. The command dispatch via `handleCommand` string matching is simple and sufficient.

### Service Orientation

The function-field injection pattern is already service-oriented at the function level. The codebase doesn't need interface-based DI at this scale — function fields are more idiomatic Go for this project size.

### Composability

Good. The `webServer` struct composes `*Daemon` cleanly. The `State` and `Config` types in `internal/pixy` are used consistently across all files. Templates use typed comparisons (`pixy.StateTracking`) instead of raw strings.

---

## 5. Architecture Visualization

Generated 2 D2 diagrams:

| Diagram       | File                                                                    |
| ------------- | ----------------------------------------------------------------------- |
| Current state | `docs/architecture-understanding/2026-05-07_20-47_current.d2` → `.svg`  |
| Ideal state   | `docs/architecture-understanding/2026-05-07_20-47_improved.d2` → `.svg` |

**Key difference:** The improved diagram shows `lastFrame` and `ptzCache` extracted to named types, `Run()` decomposed into focused lifecycle methods, and stream constants moved to the stream module.

---

## 6. Improve Codebase Architecture — Deepening Opportunities

### Candidate 1: PTZ Module

- **Files:** `handlers.go` (PTZ constants, ptzAxisLabel/Unit/Value/Limits), `v4l2.go` (PTZ operations), `middleware.go` (ptzAxisValid)
- **Problem:** PTZ logic is scattered across 3 files. Understanding PTZ requires bouncing between handlers, v4l2, and middleware.
- **Solution:** Extract `ptz.go` containing all PTZ types, constants, validation, formatting, and V4L2 interaction.
- **Benefits:** Single file for PTZ concerns. `handlers.go` shrinks. Easier to add presets and relative mode.

### Candidate 2: Command Router

- **Files:** `commands.go`
- **Problem:** The `handleCommand` switch is 80 lines. Adding new commands requires modifying the switch.
- **Solution:** Map-based dispatch with command handler functions. Each command is a self-contained function.
- **Benefits:** O(1) command lookup. Adding commands doesn't touch the router. Natural place for command help text.

### Candidate 3: Auto-manage State Machine

- **Files:** `auto.go`, `auto_test.go`
- **Problem:** Call detection state transitions are spread across debounce counters, in-call flag, and mode checks. No explicit state machine.
- **Solution:** Extract a `CallDetector` type with explicit states (Idle, DebounceIn, InCall, DebounceOut) and transitions.
- **Benefits:** State transitions become testable in isolation. Easier to reason about edge cases. Debounce logic encapsulated.

**Recommendation:** Start with Candidate 1 (PTZ module) — it's the simplest extraction with the most immediate benefit. Candidate 3 (state machine) has the deepest architectural impact but requires careful design.

---

## 7. BDD Testing Assessment

### Current State

The project already has excellent BDD-style tests in `behavior_test.go` (639 lines, 12 scenarios) using the standard `testing` package with Given/When/Then structure expressed through test naming and inline comments.

### Assessment: **Do NOT adopt Ginkgo**

**Rationale:**

1. The project's convention is `standard testing package only (no ginkgo/testify)` — documented in AGENTS.md
2. Current BDD structure is clean, readable, and consistent
3. Adding Ginkgo would introduce a heavy dependency (10+ transitive packages) for marginal readability gain
4. All existing tests would need rewriting — massive churn for no functional improvement
5. The `testing.T` approach keeps tests simple and accessible to all Go developers

### What IS needed instead:

| Test Area                                 | Priority | File                |
| ----------------------------------------- | -------- | ------------------- |
| PTZ axis helper functions (unit)          | P1       | handlers_test.go    |
| Tilt and zoom web slider (BDD)            | P1       | behavior_test.go    |
| Stream error handling (BDD)               | P2       | behavior_test.go    |
| Config validation edge cases              | P2       | pixy_test.go        |
| Concurrent command dispatch (integration) | P2       | integration_test.go |

---

## 8. TODO List Builder

Built `TODO_LIST.md` by reading all markdown docs and verifying against actual code.

- ✅ DONE: 12 items
- 🔶 PARTIAL: 3 items
- ⬜ TODO: 27 items
- **Total: 42 items**

The main finding: **SUPERB_ROADMAP.md is stale** — 12 of its 22 items are already completed but not marked as done.

---

## Overall Assessment

| Dimension      | Score | Notes                                                 |
| -------------- | ----- | ----------------------------------------------------- |
| Build quality  | 10/10 | Zero warnings, zero lint issues                       |
| Test quality   | 9/10  | BDD + unit + integration + fuzz, comprehensive        |
| Type safety    | 9/10  | Strong types, generics, Valid() methods               |
| Architecture   | 8/10  | Good DI, clean boundaries, monolithic main.go         |
| Documentation  | 7/10  | AGENTS.md excellent, roadmap stale                    |
| Error handling | 8/10  | CommandError type, sentinel errors, some toast bugs   |
| Observability  | 7/10  | OTel metrics, pprof, structured logging could improve |

**The codebase is production-ready and well-maintained.** The gap between current state and "superb" is primarily:

1. Stale roadmap/docs (easy fix)
2. Minor bugs (toast type, stream constants)
3. Architecture extraction (PTZ module, state machine)
4. Additional metrics and test coverage
