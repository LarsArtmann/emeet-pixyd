# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-05-30 09:14
**Branch:** `master` at `db27a68` + uncommitted changes
**Session:** Web UI redesign — modernized CSS, removed glassmorphism, added design context files

---

## Executive Summary

This session completely redesigned the web UI control panel. The previous design relied heavily on glassmorphism (backdrop-filter blur), gradient text, and decorative glow effects — patterns that felt generic and AI-generated. The new design is tool-like, precise, and unobtrusive: solid surfaces, intentional color, tactile controls, and no decorative fluff. Two new design context files were added (`PRODUCT.md`, `DESIGN.md`) to anchor future UI work.

---

## A) FULLY DONE

### This Session

| # | Item | Impact | Files |
|---|------|--------|-------|
| 1 | Complete CSS rewrite — removed glassmorphism, gradient text, glow effects | Visual identity | `static/style.css` |
| 2 | Replaced inline badge styles with semantic CSS classes | Maintainability | `templates.templ` |
| 3 | Removed inline `margin-bottom` from cards (now CSS-driven) | Consistency | `templates.templ` |
| 4 | Removed inline color styles from "In call" indicator | Maintainability | `templates.templ` |
| 5 | Added `PRODUCT.md` — product register, users, brand personality, design principles | Design context | `PRODUCT.md` |
| 6 | Added `DESIGN.md` — color palette, typography, motion, component specs | Design context | `DESIGN.md` |

### Quality Metrics (Current)

| Metric | Value | Status |
|--------|-------|--------|
| Build | Clean | 0 errors |
| Lint (golangci-lint v2) | 0 issues | Clean |
| Tests (race detector) | 253 PASS / 4 FAIL | Same 4 pre-existing failures (go-branded-id v0.3.0) |
| Fuzz tests | 2 passing | `FuzzExtractJPEGFrame`, `FuzzParseHIDResponse` |
| Benchmarks | 7 passing | All green |
| Source lines (non-test) | ~4,300 | — |
| Test lines | ~6,100 | ~1.4:1 test:source ratio |
| Test functions | 257 | — |
| Source files | 19 | — |
| Test files | 14 | — |

### Feature Delivery (44/44 — 100%)

All 44 features in `FEATURES.md` remain FULLY_FUNCTIONAL. The "Dark Glassmorphism Theme" feature still works but the description is now outdated — it should read "Dark precision-tool theme, solid surfaces, semantic colors".

### TODO List Progress

| Status | Count | Percentage |
|--------|-------|------------|
| DONE | 34 | 55.7% |
| PARTIAL | 0 | 0% |
| SKIP | 1 | 1.6% |
| TODO | 26 | 42.6% |
| **Total** | **61** | **100%** |

---

## B) PARTIALLY DONE

**Nothing is partially done.**

---

## C) NOT STARTED

26 items remain in `TODO_LIST.md`. Key categories:

### Code Quality
- #14: Structured log levels audit
- #15: Graceful degradation for missing optional deps
- #40/#61: Update/archive `SUPERB_ROADMAP.md`

### Observability
- #16: Additional Prometheus metrics (stream duration, frames, command counters, probe, uevent)
- #17: Circuit breaker for HID failures
- #18: Stream health monitoring
- #20: Continuous fuzz in CI

### Architecture (Higher effort)
- #21-#24: Extract interfaces (`Commander`, `HIDDevice`, `ProcessInspector`, `UeventListener`)
- #51: Consolidate 9 function pointers into `Dependencies` interface
- #52: Replace `handleCommand(string) string` with typed `CommandResult`
- #53: Consolidate PTZ logic into single `ptz.go`

### Web UI
- #26: Mobile-responsive layout
- #27: WebSocket for live state updates (replace 3s HTMX polling)
- #28: Keyboard shortcuts for PTZ (arrow keys, +/- zoom)
- #29: PTZ relative mode (`pan+10`, `tilt-5`)
- #30: Camera preset support

### Testing
- #31: Integration test harness with fake devices
- #32: Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths
- #33: Surface auto-manage errors to web UI
- #34: Improve MJPEG stream reconnection
- #35: Integration test with real hardware (build tag guarded)

### Other
- #42: PTZ readback accuracy (delay or in-memory "last set")

---

## D) TOTALLY FUCKED UP

### Pre-existing Test Failures (4 tests)

Caused by `go-branded-id` v0.3.0 changing `String()` output to include a typed prefix:

| Test | Failure | Expected | Got |
|------|---------|----------|-----|
| `TestPpidOf_CurrentProcess` | `PID.String()` | `"42"` | `"PID:42"` |
| `TestNewPID` | `PID.String()` | `"42"` | `"PID:42"` |
| `TestHandleCallStart_SetsPipeWireSource` | `SourceID.String()` | `42` | `SourceID:42` |
| `TestBehavior_FullAutoCallLifecycle` | `SourceID.String()` | `[42]` | `[SourceID:42]` |

These are intentionally not fixed. `nix` builds skip tests via `doCheck = false`. CI runs `go test` via GitHub Actions. The library author owns both repos.

### No New Issues Introduced

- 0 build errors
- 0 lint issues
- UI changes are CSS-only + minor template markup cleanup
- No Go code changes
- No data corruption risks
- No security vulnerabilities known
- No new dead code paths

---

## E) WHAT WE SHOULD IMPROVE!

### Immediate Priority

1. **Fix go-branded-id test failures** — Either update test expectations to match v0.3.0 `String()` format, or pin the dependency. Currently 4 tests fail on every run.

2. **Middleware-aware integration tests** — The stream tests (and other handler tests) should test through the full middleware chain, not just the bare `mux`. This would have caught the `Flusher` bug (fixed in previous session). Consider adding a `newTestServerWithMiddleware()` helper.

3. **Update `FEATURES.md`** — The "Dark Glassmorphism Theme" feature description is now stale. Should reflect the new solid-surface, precision-tool aesthetic.

### Design Follow-ups (from this session)

4. **Template inline styles audit** — After this session, 3 inline `style=` attributes remain in `templates.templ`:
   - Line 38: `preview-fallback` display/position styles (functional, needed for fallback visibility toggle)
   - Line 130: `status-panel` `position:relative` (needed for HTMX indicator overlay)
   - Line 185: None remaining in "In call" section (was fixed)
   The preview fallback styles are functional (toggled via JS), but could be moved to a `.preview-fallback-hidden` class.

5. **Mobile responsiveness** — The grid collapses to 1 column at 720px, but the preview card and PTZ sliders could use better touch targets. The toggle switches at 40x22px are small for mobile.

### Architectural Improvements (Future Sessions)

Highest impact:

1. **`Dependencies` interface** (#51) — Replace 9 function pointers with compile-time-checked interface. Single biggest architectural win.
2. **Typed `CommandResult`** (#52) — Replace `handleCommand(string) string` with structured result. Enables richer responses.
3. **Consolidate PTZ** (#53) — Currently split across 4 files. After `v4l2SetMultiple` removal in Round 5, `v4l2.go` is smaller but still fragmented.

### Documentation Debt

- `docs/SUPERB_ROADMAP.md` is stale (#40, #61) — metrics, file tables, and dependency lists all outdated
- `TODO_LIST.md` has 26 remaining items; many overlap with the roadmap

---

## F) TOP #25 THINGS WE SHOULD GET DONE NEXT

Prioritized by impact × effort (Pareto order):

### Tier 1: Critical Fixes (30 min)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | Fix `go-branded-id` v0.3.0 test failures (4 tests) | CI reliability | 15 min |
| 2 | Add middleware-aware integration test harness | Prevents middleware regressions | 30 min |

### Tier 2: Code Quality (1-2 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 3 | Structured log levels audit (#14) | Observability | 30 min |
| 4 | Graceful degradation for missing optional deps (#15) | Robustness | 30 min |
| 5 | Update/archive `SUPERB_ROADMAP.md` (#40, #61) | Doc accuracy | 20 min |
| 6 | Update `FEATURES.md` theme description | Doc accuracy | 5 min |

### Tier 3: Observability (2-3 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 7 | Additional Prometheus metrics — stream duration, frames served (#16) | Observability | 1h |
| 8 | Stream health monitoring — frame counter, uptime metric (#18) | Reliability | 1h |
| 9 | Circuit breaker for HID failures (#17) | Stability | 1h |

### Tier 4: Architecture (4-8 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 10 | Extract `Dependencies` interface (#51) | Testability, compile-time safety | 2h |
| 11 | Typed `CommandResult` (#52) | Richer command responses | 2h |
| 12 | Consolidate PTZ into `ptz.go` (#53) | Maintainability | 1h |
| 13 | Extract `Commander` interface (#21) | Mockable shell commands | 2h |
| 14 | Extract `HIDDevice` interface (#22) | Mockable HID I/O | 2h |

### Tier 5: Web UI (4-8 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 15 | WebSocket for live state updates (#27) | Real-time UX | 3h |
| 16 | Mobile-responsive layout (#26) | Accessibility | 2h |
| 17 | Keyboard shortcuts for PTZ (#28) | UX | 1h |
| 18 | PTZ relative mode (#29) | UX | 1h |
| 19 | Camera preset support (#30) | UX | 2h |

### Tier 6: Testing (4-8 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 20 | Integration test harness with fake devices (#31) | Test coverage | 3h |
| 21 | Test coverage for real hardware paths (#32) | Confidence | 2h |
| 22 | Surface auto-manage errors to web UI (#33) | Debuggability | 1h |
| 23 | Improve MJPEG stream reconnection (#34) | Reliability | 2h |
| 24 | Fix flaky parallel tests (`TestHandleStream_NoFFmpeg`, `TestSocket_StatusCommand`) | CI reliability | 30 min |
| 25 | Integration test with real hardware (#35) | Validation | 3h |

---

## G) TOP #1 QUESTION WE CANNOT FIGURE OUT OURSELVES

**Why does `go-branded-id` v0.3.0 prefix `String()` output with the type name?** (`"PID:42"` instead of `"42"`). The project author owns both this repo and `go-branded-id`. Was this an intentional breaking change? Should we:
- (a) Update all test expectations to match the new format
- (b) Pin `go-branded-id` to v0.2.x in `go.mod`
- (c) Add a `Value()` or `Raw()` method to `go-branded-id` and update callers

The `nix` `doCheck = false` workaround masks this in builds but the tests fail on every `go test` run. We need a decision on the intended direction before fixing the 4 affected tests.

---

## Uncommitted Changes

| File | Change |
|------|--------|
| `static/style.css` | Complete rewrite — 157 insertions, 145 deletions. Removed glassmorphism, gradient text, glow effects. Solid surfaces, refined palette, better motion. |
| `templates.templ` | 16 changes — inline styles replaced with CSS classes, inline margins removed |
| `PRODUCT.md` | New file — product register, users, purpose, brand personality, design principles |
| `DESIGN.md` | New file — color palette, typography, elevation, components, motion, layout |
