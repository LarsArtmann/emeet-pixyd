# emeet-pixyd — Comprehensive Status Report

**Date:** 2026-05-30 11:27  
**Branch:** `master` at `765db53`  
**Session:** Nix flake improvements + UI redesign follow-up status

---

## Executive Summary

This session completed two independent workstreams: (1) a full web UI redesign removing glassmorphism and establishing design context files, and (2) Nix flake maintainability improvements. Both are committed and verified. Working tree is clean, 4 commits ahead of origin.

---

## A) FULLY DONE ✅

### This Session (Two Commits)

| # | Commit | Description | Files |
|---|--------|-------------|-------|
| 1 | `a8bc41e` | UI redesign — modernized CSS, removed glassmorphism/gradient text/glows, added PRODUCT.md + DESIGN.md | `static/style.css`, `templates.templ`, `PRODUCT.md`, `DESIGN.md` |
| 2 | `765db53` | Nix improvements — pattern-based sourceFiles, app meta, required user option | `flake.nix`, `modules/nixos.nix` |

### UI Redesign Details

- **Removed:** All `backdrop-filter` blur, gradient text on header, decorative glow effects on buttons/state dots, side-stripe toast borders
- **Replaced with:** Solid surfaces (`#13161d`), semantic color borders, tactile controls, refined palette
- **New palette:** Warmer bg (`#0a0c10`), richer semantic colors (green `#4cc88a`, yellow `#e5a13d`, red `#e85d5d`)
- **Motion:** Reduced to 150–200ms, `cubic-bezier(0.4, 0, 0.2, 1)`
- **Template cleanup:** Inline badge styles → CSS classes, inline margins → CSS-driven, inline colors → `.meta-call`
- **Design context:** `PRODUCT.md` (product register, users, brand personality, principles) + `DESIGN.md` (palette, typography, motion, components)

### Nix Improvements Details

- **Pattern-based sourceFiles:** Replaced 22-line manual file list with `lib.fileset.fileFilter` matching `*.go` (non-test), `go.mod`, `go.sum`, `*.templ`, plus `static/` directory. Adding a `.go` file no longer requires editing `flake.nix`.
- **App meta:** Added `meta.mainProgram` + `meta.description` to `apps.default` — fixes `nix flake check` warning.
- **Required user option:** Removed hardcoded `default = "lars"` from `hardware.emeet-pixy.user`. Now required when module is enabled.

### Quality Metrics (Current)

| Metric | Value | Status |
|--------|-------|--------|
| Build | Clean | 0 errors |
| Lint (golangci-lint v2) | 0 issues | Clean |
| Tests (race detector) | 253 PASS / 4 FAIL | Same 4 pre-existing failures (go-branded-id v0.3.0) |
| Nix flake check | All checks passed | No warnings |
| Nix build | Successful | 51.7 MiB closure |
| Source filtering | Verified | Correctly excludes `*_test.go`, includes `static/` |
| Fuzz tests | 2 passing | `FuzzExtractJPEGFrame`, `FuzzParseHIDResponse` |
| Benchmarks | 7 passing | All green |
| Source lines (non-test) | ~4,300 | — |
| Test lines | ~6,100 | ~1.4:1 test:source ratio |
| Test functions | 257 | — |

### Feature Delivery (44/44 — 100%)

All 44 features in `FEATURES.md` remain FULLY_FUNCTIONAL. Note: "Dark Glassmorphism Theme" description is now stale — should be updated to "Dark precision-tool theme, solid surfaces, semantic colors".

### TODO List Progress

| Status | Count | Percentage |
|--------|-------|------------|
| ✅ DONE | 34 | 55.7% |
| 🔶 PARTIAL | 0 | 0% |
| ❌ SKIP | 1 | 1.6% |
| ⬜ TODO | 26 | 42.6% |
| **Total** | **61** | **100%** |

---

## B) PARTIALLY DONE 🔶

**Nothing is partially done.**

---

## C) NOT STARTED ⬜

26 items remain in `TODO_LIST.md`. Key categories unchanged from previous session:

### Code Quality
- #14: Structured log levels audit
- #15: Graceful degradation for missing optional deps
- #40/#61: Update/archive `SUPERB_ROADMAP.md`

### Observability
- #16: Additional Prometheus metrics (stream duration, frames, command counters, probe, uevent)
- #17: Circuit breaker for HID failures
- #18: Stream health monitoring
- #20: Continuous fuzz in CI

### Architecture
- #21–#24: Extract interfaces (`Commander`, `HIDDevice`, `ProcessInspector`, `UeventListener`)
- #51: Consolidate 9 function pointers into `Dependencies` interface
- #52: Replace `handleCommand(string) string` with typed `CommandResult`
- #53: Consolidate PTZ logic into single `ptz.go`

### Web UI
- #26: Mobile-responsive layout (grid collapses at 720px but touch targets could improve)
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

## D) TOTALLY FUCKED UP 💥

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
- No Go code changes in this session
- Nix changes verified with `nix flake check` and `nix build`
- No data corruption risks
- No security vulnerabilities known
- No new dead code paths

---

## E) WHAT WE SHOULD IMPROVE!

### Immediate Priority

1. **Fix go-branded-id test failures** — Either update test expectations to match v0.3.0 `String()` format, or pin the dependency. Currently 4 tests fail on every run.

2. **Update `FEATURES.md`** — The "Dark Glassmorphism Theme" feature description is now stale. Should reflect the new solid-surface, precision-tool aesthetic.

3. **Middleware-aware integration tests** — The stream tests (and other handler tests) should test through the full middleware chain, not just the bare `mux`. This would have caught the `Flusher` bug (fixed in `db27a68`). Consider adding a `newTestServerWithMiddleware()` helper.

### Design Follow-ups (from UI session)

4. **Template inline styles audit** — After this session, 2 inline `style=` attributes remain in `templates.templ`:
   - Line 38: `preview-fallback` display/position styles (functional, needed for fallback visibility toggle via JS)
   - Line 130: `status-panel` `position:relative` (needed for HTMX indicator overlay)
   The preview fallback styles are functional (toggled via JS), but could be moved to a `.preview-fallback-hidden` class for consistency.

5. **Mobile responsiveness** — The grid collapses to 1 column at 720px, but the preview card and PTZ sliders could use better touch targets. Toggle switches at 40×22px are small for mobile.

### Nix Follow-ups (from this session)

6. **`sourceFiles` edge case** — The `fileFilter` uses `lib.hasSuffix ".go"` which will include `templates_templ.go` (the generated Go file from `templ generate`). This is harmless since `templ generate` runs in `preBuild` and will overwrite it, but including a generated file in the source is slightly impure. Consider adding `&& file.name != "templates_templ.go"` to the filter for stricter purity.

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

### Tier 2: Code Quality (1–2 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 3 | Structured log levels audit (#14) | Observability | 30 min |
| 4 | Graceful degradation for missing optional deps (#15) | Robustness | 30 min |
| 5 | Update/archive `SUPERB_ROADMAP.md` (#40, #61) | Doc accuracy | 20 min |
| 6 | Update `FEATURES.md` theme description | Doc accuracy | 5 min |
| 7 | Exclude `templates_templ.go` from Nix source filter | Purity | 5 min |

### Tier 3: Observability (2–3 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 8 | Additional Prometheus metrics — stream duration, frames served (#16) | Observability | 1h |
| 9 | Stream health monitoring — frame counter, uptime metric (#18) | Reliability | 1h |
| 10 | Circuit breaker for HID failures (#17) | Stability | 1h |

### Tier 4: Architecture (4–8 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 11 | Extract `Dependencies` interface (#51) | Testability, compile-time safety | 2h |
| 12 | Typed `CommandResult` (#52) | Richer command responses | 2h |
| 13 | Consolidate PTZ into `ptz.go` (#53) | Maintainability | 1h |
| 14 | Extract `Commander` interface (#21) | Mockable shell commands | 2h |
| 15 | Extract `HIDDevice` interface (#22) | Mockable HID I/O | 2h |

### Tier 5: Web UI (4–8 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 16 | WebSocket for live state updates (#27) | Real-time UX | 3h |
| 17 | Mobile-responsive layout (#26) | Accessibility | 2h |
| 18 | Keyboard shortcuts for PTZ (#28) | UX | 1h |
| 19 | PTZ relative mode (#29) | UX | 1h |
| 20 | Camera preset support (#30) | UX | 2h |

### Tier 6: Testing (4–8 hours)

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 21 | Integration test harness with fake devices (#31) | Test coverage | 3h |
| 22 | Test coverage for real hardware paths (#32) | Confidence | 2h |
| 23 | Surface auto-manage errors to web UI (#33) | Debuggability | 1h |
| 24 | Improve MJPEG stream reconnection (#34) | Reliability | 2h |
| 25 | Fix flaky parallel tests (`TestHandleStream_NoFFmpeg`, `TestSocket_StatusCommand`) | CI reliability | 30 min |

---

## G) TOP #1 QUESTION WE CANNOT FIGURE OUT OURSELVES

**Why does `go-branded-id` v0.3.0 prefix `String()` output with the type name?** (`"PID:42"` instead of `"42"`). The project author owns both this repo and `go-branded-id`. Was this an intentional breaking change? Should we:
- (a) Update all test expectations to match the new format
- (b) Pin `go-branded-id` to v0.2.x in `go.mod`
- (c) Add a `Value()` or `Raw()` method to `go-branded-id` and update callers

The `nix` `doCheck = false` workaround masks this in builds but the tests fail on every `go test` run. We need a decision on the intended direction before fixing the 4 affected tests.
