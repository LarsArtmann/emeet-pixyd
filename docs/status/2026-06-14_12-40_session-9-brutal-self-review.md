# Status Report — 2026-06-14 (Session 9)

**Generated:** 2026-06-14 12:40
**Branch:** master @ `3905583`
**Commits this session:** 7 (95bc833 → 3905583)

---

## Executive Summary

The codebase is **production-ready** for its purpose: a Linux hardware daemon for the EMEET PIXY webcam. Build passes, lint is clean (0 issues), all tests pass with `-race`. This session focused on a brutal self-review that surfaced and fixed **11 real issues** across 3 rounds of work — from a missing `device` command in `--help` output to a nil-panic path in metrics, a DI split brain, and a swallowed error in the auto-manage pipeline.

---

## Build & Quality Metrics

| Metric                    | Value        | Target  | Status |
| ------------------------- | ------------ | ------- | ------ |
| Build                     | Pass         | Pass    | ✅      |
| Lint (golangci-lint v2)   | 0 issues     | 0       | ✅      |
| Tests (`-race`)           | Pass         | Pass    | ✅      |
| Coverage (main pkg)       | 71.3%        | 80%     | ⚠️     |
| Coverage (internal/pixy)  | 91.3%        | 80%     | ✅      |
| Fuzz tests                | 2 files      | —       | ✅      |
| Benchmarks                | 7            | —       | ✅      |
| Production files < 350 LOC| Yes (all)    | All     | ✅      |
| `init()` functions        | 0            | 0       | ✅      |
| Duplicate code blocks     | 0            | 0       | ✅      |
| Git working tree          | Clean        | Clean   | ✅      |

---

## a) FULLY DONE (This Session)

### Bugs Fixed

| # | Issue | File(s) | Commit |
|---|-------|---------|--------|
| 1 | **`--help` text missing `device` command name** — users saw `"  Show device paths"` with no command | `main.go:294` | `95bc833` |
| 2 | **`metricsInstance` nil panic** — if OTel exporter init failed, every `record*` call would crash the daemon | `metrics.go` (8 functions) | `95bc833` |
| 3 | **`findSource` error silently swallowed** — PipeWire source switch failures invisible to user, not added to `autoError` | `auto.go:43-48` | `cb871fa` |
| 4 | **`ConfigFromEnv` silently ignored invalid env values** — `EMEET_PIXYD_POLL_INTERVAL=abc` gave default with zero feedback | `internal/pixy/config.go` (4 vars) | `cb871fa` |
| 5 | **`recordFrame` pointless allocation** — `metric.WithAttributes()` with zero args | `metrics.go:154` | `cb871fa` |
| 6 | **Config doc comment lied** — claimed "Fields marked with env tags" but no struct tags exist | `internal/pixy/config.go` | `ace0e47` |

### Architecture / Split Brains Fixed

| # | Issue | Resolution | Commit |
|---|-------|------------|--------|
| 7 | **`parsePTZValues` DI split brain** — `deps.parsePTZ` existed but 2 of 3 callers called the function directly | Routed all callers through `deps.parsePTZ` | `95bc833` |
| 8 | **`collectMetrics` in production file** — test-only function in `metrics.go` | Moved to `metrics_test.go` | `95bc833` |
| 9 | **`checkDevice` in wrong file** — lived in `handlers.go`, only caller in `stream.go` | Moved to `stream.go` | `95bc833` |
| 10 | **`internal/pixy/pixy.go` was 501 lines** — mixed domain types, config, IPC | Split into `pixy.go` (319) + `config.go` (146) + `ipc.go` (51) | `26f77f9` |
| 11 | **go.mod had two direct require blocks** — formatting artifact | Merged into one sorted block | `ace0e47` |

### CI Improvements

| # | Change | Commit |
|---|--------|--------|
| 12 | Added `govulncheck` step to CI | `95bc833` |
| 13 | Added `templ generate` step to CI (generated files are gitignored) | `1ba990a` |
| 14 | Changed test step to generate coverage report | `1ba990a` |

### Tests Added

| # | Test | File | Commit |
|---|------|------|--------|
| 15 | `TestHandleCallStart_FindSourceErrorSurfacesInAutoError` | `auto_test.go` | `cb871fa` |
| 16 | `TestHandleCallStart_FindSourceSuccessClearsAutoError` | `auto_test.go` | `cb871fa` |

### Documentation

| # | Change | Commit |
|---|--------|--------|
| 17 | Fixed stale AGENTS.md linter docs (claimed linters were "removed" that are actually present) | `95bc833` |
| 18 | Added all new architectural notes to AGENTS.md (parsePTZValues DI, metricsInstance guards, findSource error propagation, env logging, pixy package split, CI changes) | `47535be`, `3905583` |

---

## b) PARTIALLY DONE

| Item | Status | Details |
|------|--------|---------|
| **Test coverage** | 71.3% main / 91.3% pixy | Gaps in hardware-dependent paths (`stream.go`, `process.go`, `hid.go`). Hard to close without device or fake harness. |
| **CI coverage enforcement** | Reporting only | CI generates `coverage.out` and prints summary but does not enforce a minimum threshold yet. |

---

## c) NOT STARTED (Open TODOs — 15 remaining)

### From TODO_LIST.md

| # | Priority | Task | Effort |
|---|----------|------|--------|
| 14 | P1 | Structured log levels audit | Medium |
| 20 | P2 | Continuous fuzz in CI (60s per test, store corpus) | Medium |
| 21 | P2 | Extract `Commander` interface for shell commands | Large |
| 23 | P2 | Extract `ProcessInspector` interface for /proc | Medium |
| 24 | P2 | Extract `UeventListener` interface for netlink | Medium |
| 26 | P3 | Mobile-responsive layout | Medium |
| 27 | P3 | WebSocket for live state updates | Large |
| 30 | P3 | Camera preset support (save/recall PTZ) | Medium |
| 31 | P3 | Integration test harness with fake devices | Large |
| 32 | P3 | Test coverage for stream.go, process.go, hid.go hardware paths | Large |
| 34 | P3 | Improve MJPEG stream reconnection | Medium |
| 35 | P4 | Integration test with real hardware (build tag guarded) | Medium |
| 42 | P3 | PTZ readback accuracy (delay or in-memory last-set value) | Medium |

---

## d) TOTALLY FUCKED UP

**Nothing.** No regressions, no broken builds, no data loss. All changes were verified with build + lint + tests before each commit. The LSP shows stale `DuplicateDecl` errors from the pixy.go split, but `go build` and `go test` pass — confirmed via CI-equivalent commands. This is a known gopls caching issue.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`main.go` at project root** — BuildFlow flags `main.go` should be in `/cmd/`. This is a deliberate choice for a single-binary daemon, but would require NixOS module path changes if moved. Low priority, high churn.
2. **`commands.go` at 355 lines** — Slightly over the 350-line guideline. Could extract PTZ command handling into `ptz.go`, but barely worth the churn.
3. **No `Commander` interface** — Shell commands (`v4l2-ctl`, `wpctl`, `notify-send`, `ffmpeg`) are direct `exec.CommandContext` calls. Extracting a `Commander` interface would make them testable without real binaries, but the existing DI function fields (`v4l2SetFn`, `findSourceFn`, etc.) already cover the testable paths.

### Testing

4. **Coverage stuck at 71.3%** — The gap is almost entirely in hardware-dependent code: HID writes to `/dev/hidraw*`, V4L2 reads via `v4l2-ctl`, `/proc/*/fd` scanning, MJPEG frame extraction. These require either a fake device harness or real hardware to test meaningfully.
5. **No continuous fuzz in CI** — Fuzz tests exist (`handlers_fuzz_test.go`, `hid_fuzz_test.go`) but CI doesn't run them with corpus storage. Go's native fuzzer needs `go test -fuzz` with time-based execution.
6. **Large test files** — `main_test.go` (1552 lines), `integration_test.go` (1156 lines), `pixy_test.go` (791 lines), `commands_test.go` (785 lines). These could be split by concern, but test file splitting adds churn without functional value.

### Observability

7. **Coverage threshold not enforced** — CI reports coverage but doesn't fail below a threshold. Adding `go tool cover -func=coverage.out | awk` parsing would enforce it.
8. **No tracing** — Only metrics (Prometheus) and structured logging (slog). No distributed tracing (OTel traces). For a single-process daemon, this is acceptable.

### Security

9. **CSP allows `'unsafe-inline'` for scripts** — Required by HTMX's `hx-*` attributes. Eliminating this would require nonce-based CSP, which is significant work for a localhost-only web UI.

---

## f) Top 25 Things to Do Next

Sorted by impact/effort ratio (highest impact, lowest effort first).

### P0 — Quick Wins (hours)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | **Enforce coverage minimum in CI** — fail build below 65% | Medium | Low |
| 2 | **Add `go test -bench` to CI** — run benchmarks on each PR, store results | Medium | Low |
| 3 | **PTZ readback accuracy (#42)** — maintain in-memory "last set" value to avoid V4L2 readback delay | Medium | Low |
| 4 | **Add `errors.Is`/`errors.As` test coverage** — verify sentinel errors are unwrappable from all call sites | Low | Low |

### P1 — High Value (days)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 5 | **Structured log levels audit (#14)** — standardize Debug/Info/Warn/Error across all files | Medium | Medium |
| 6 | **Extract `ProcessInspector` interface (#23)** — mockable `/proc` traversal for test coverage | High | Medium |
| 7 | **Extract `Commander` interface (#21)** — mockable shell commands, eliminates `exec.CommandContext` in production code | High | Medium |
| 8 | **Add stream.go test coverage** — test `extractJPEGFrame` with real MJPEG byte sequences, test `setupStream` error paths | High | Medium |
| 9 | **Add hid.go test coverage** — test `setDeviceState` circuit breaker logic, HID response parsing edge cases | High | Medium |
| 10 | **Continuous fuzz in CI (#20)** — add `go test -fuzz` step with 60s budget, store corpus in repo | Medium | Medium |

### P2 — Medium Value (weeks)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 11 | **Extract `UeventListener` interface (#24)** — mockable netlink for hotplug tests | Medium | Medium |
| 12 | **Integration test harness with fake devices (#31)** — create fake `/dev/hidraw` + `/dev/video` character devices for end-to-end testing | Very High | Large |
| 13 | **Camera preset support (#30)** — save/recall PTZ positions via state persistence | Medium | Medium |
| 14 | **Improve MJPEG stream reconnection (#34)** — exponential backoff, client-side reconnection improvements | Medium | Medium |
| 15 | **Mobile-responsive layout (#26)** — CSS media queries for small screens | Low | Medium |
| 16 | **WebSocket for live updates (#27)** — replace 3s HTMX polling with WebSocket push | Medium | Large |
| 17 | **Add `govulncheck` output to PR comments** — surface vulnerabilities in PR reviews | Low | Low |
| 18 | **Nonce-based CSP** — eliminate `'unsafe-inline'` for scripts | Low | Large |
| 19 | **Add `wpctl status` output parser tests** — test `findPixySource` with real output samples | Medium | Low |
| 20 | **Add `parsePTZValues` integration test** — test full V4L2 get/parse round-trip with mock `v4l2-ctl` output | Medium | Low |

### P3 — Strategic (months)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 21 | **Real hardware integration tests (#35)** — build-tagged tests that run on actual PIXY hardware | High | Large |
| 22 | **Split `main_test.go` (1552 lines)** — break into `daemon_test.go`, `config_test.go`, `probe_test.go` | Low | Medium |
| 23 | **Move `main.go` to `cmd/emeet-pixyd/`** — standard Go project layout | Low | Medium |
| 24 | **Add restart-on-crash integration test** — verify systemd watchdog + state recovery | Medium | Large |
| 25 | **Performance benchmark suite expansion** — add benchmarks for `autoManage`, `handleCommand`, `parsePTZValues` with large inputs | Low | Medium |

---

## g) Top Question I Cannot Answer Myself

**Should the `main.go` → `cmd/emeet-pixyd/main.go` move happen?**

The `go-structure-linter` flags this as CRITICAL. The standard Go project layout convention says executables go in `/cmd/`. But this project is:
- A **single binary** with no subcommands (no `cmd/server` + `cmd/cli` split needed)
- Uses `nix build` which references the root package directly
- The NixOS module, `package.nix`, and CI all assume root-level `main.go`
- The `flake.nix` build derivation calls `go build` without a target path

Moving `main.go` would require updating: `package.nix` (build target), `flake.nix` (if it references paths), CI workflows, the `templ generate` step, and potentially the NixOS module. That's **high churn for zero functional benefit** on a single-binary project. I lean strongly toward NOT moving it, but the linter disagrees and I can't resolve this tension without your input. Should I suppress the linter or move the file?
