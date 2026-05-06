# EMEET PIXY Daemon — Status Report

**Date:** 2026-05-06 05:24 CEST
**Branch:** master
**Last commit:** `36564d0 feat(core): add architecture diagram and handlers`
**Coverage:** 69.6% total (69.1% main, 77.9% internal/pixy)
**Lint:** 0 issues
**Tests:** 90+ tests, all PASS (race detector enabled)
**Total source:** 12,630 lines (Go + templ + JS + CSS)

---

## A. FULLY DONE

### PTZ Web Slider Fix (this session)

**Root cause:** `handlePTZ` returned PTZ values from the cache/hardware read instead of the user's input value. After setting pan=50 via the slider, the handler re-read the position (getting stale cached value 0) and returned a slider HTML reset to 0. The slider snapped back immediately after every drag.

**What was fixed:**

- `handlers.go` — Rewrote `handlePTZ` to:
  - Return the user's clamped value on success (not cached/hardware value)
  - Invalidate the PTZ cache after successful set
  - Show success toast with "Pan set to 50" message
  - Show error toast on failure (was previously silent)
  - Extracted `invalidatePTZCache()`, `ptzAxisLabel()`, `ptzAxisUnit()`, `ptzAxisValue()` helpers
  - Added `toastTypeError` constant
- `templates.templ` — Added `ptzSliderWithToast` template combining slider + OOB toast for HTMX responses
- `behavior_test.go` — Two new BDD tests:
  - `TestBehavior_PTZWebSliderReflectsUserInput` — verifies slider shows user's value not stale cache
  - `TestBehavior_PTZWebSliderShowsErrorOnFailure` — verifies error toast on failure

### Core Features (all ✅ FULLY_FUNCTIONAL)

- Camera control: tracking, idle, privacy, toggle-privacy, center, PTZ
- Audio modes: NC, Live, Original with cycling
- Auto-management: full, tracking-only, privacy-only, off — with debounce
- Call detection via `/proc/*/fd` scanning
- PipeWire source switching
- Gesture control toggle
- MJPEG streaming + snapshot
- Waybar integration
- OTel metrics + Prometheus `/metrics` endpoint
- Web UI with HTMX (buttons, sliders, toggles, toasts)
- NixOS module (systemd user service, udev rules, tmpfiles.d)
- Device probing via sysfs walks
- Netlink uevent hotplug detection
- State persistence (JSON, atomic write)
- Socket commands (status, waybar, all control commands)
- Keyboard shortcuts (T, I, P, C)
- BDD behavioral tests (13 scenarios)
- Fuzz tests (handlers, HID parsing)

### Infrastructure

- Nix flake build with `proxyVendor = true`
- GitHub Actions CI (go vet, golangci-lint, go test -race)
- golangci-lint config tuned for hardware daemon patterns
- `templ generate` for HTML templates
- AGENTS.md comprehensive project documentation
- FEATURES.md feature inventory with status
- 19 status reports in `docs/status/`
- Architecture diagrams in `docs/architecture-understanding/`

---

## B. PARTIALLY DONE

Nothing partially done at this time.

---

## C. NOT STARTED

1. **Mobile-responsive web UI testing** — CSS has `@media (max-width: 720px)` but no visual testing
2. **WebSocket support** — Polling every 3s via HTMX; WebSocket would reduce latency
3. **Multiple camera support** — Currently hardcoded to first PIXY found
4. **Configuration web UI** — All config via env vars; no runtime web config
5. **Access control / auth** — Web UI listens on localhost only, no auth
6. **Structured logging levels** — All logging via slog but no configurable log levels beyond debug flag
7. **i18n / localization** — All UI strings hardcoded in English
8. **Backup/restore state** — State persistence exists but no import/export
9. **Camera firmware update support** — No firmware version query or update
10. **Integration test with real hardware** — All tests use mocks

---

## D. TOTALLY FUCKED UP

Nothing is totally fucked up. The codebase is in good shape:

- Build: ✅ clean
- Tests: ✅ 90+ passing
- Lint: ✅ 0 issues
- Coverage: 69.6% (reasonable for hardware daemon)

---

## E. WHAT WE SHOULD IMPROVE

### High Impact

1. **PTZ readback accuracy** — Even with cache invalidation, `v4l2-ctl` readback may lag behind the actual hardware position. Could add a short delay (100ms) before readback, or maintain an in-memory "last set" value per axis
2. **Test coverage** — 69.6% is decent but `stream.go` (MJPEG streaming), `process.go` (real /proc scanning), `hid.go` (real HID writes) have limited coverage due to hardware dependency
3. **Error feedback in web UI** — Error banner in status panel works for non-PTZ, PTZ now shows toasts. But errors during auto-manage (tracking/audio/gesture failures) don't surface to the web UI at all
4. **Stream reliability** — MJPEG stream reconnection logic in `app.js` works but uses exponential backoff from 3s to 30s. Could be smarter about detecting camera state changes

### Medium Impact

5. **Handler test coverage** — `handlers.go` now has extract helpers but no direct unit tests for `ptzAxisLabel`, `ptzAxisUnit`, `ptzAxisValue`, `invalidatePTZCache`
6. **Docs freshness** — `FEATURES.md` and `AGENTS.md` last updated 2026-05-03; the PTZ fix changes behavior that should be documented
7. **Auto-manage notification** — Desktop notifications work but the user has no way to configure which notifications they want
8. **PTZ relative mode** — Camera supports relative pan/tilt (step by N degrees from current position) but only absolute mode is exposed

### Low Impact

9. **CSS cleanup** — `style.css` is 688 lines, could benefit from CSS custom property grouping
10. **Go module tidiness** — `prometheus/client_golang` is kept only for `promhttp`; could be replaced with pure OTel
11. **Template organization** — `templates.templ` is 206 lines; could split into multiple files as project grows

---

## F. Top 25 Things to Do Next

| # | Priority | Item | Effort | Impact |
|---|----------|------|--------|--------|
| 1 | P0 | Update AGENTS.md with PTZ fix details (cache invalidation, toast behavior) | S | M |
| 2 | P0 | Update FEATURES.md — PTZ Sliders description should mention toast feedback | S | S |
| 3 | P1 | Add unit tests for `ptzAxisLabel`, `ptzAxisUnit`, `ptzAxisValue`, `invalidatePTZCache` | S | M |
| 4 | P1 | Add BDD test for tilt and zoom web slider (currently only pan tested) | S | M |
| 5 | P1 | Test the actual web UI with a real PIXY camera end-to-end | M | H |
| 6 | P1 | Surface auto-manage errors to web UI (tracking/audio/gesture failures) | M | H |
| 7 | P2 | Add WebSocket support for real-time status updates (replace 3s polling) | L | H |
| 8 | P2 | Add PTZ relative mode (`pan+10`, `tilt-5`) for step-by-step control | M | M |
| 9 | P2 | Add camera preset support (save/recall PTZ positions) | M | H |
| 10 | P2 | Improve MJPEG stream reconnection (detect camera state changes) | M | M |
| 11 | P2 | Add configurable notification preferences | M | M |
| 12 | P3 | Add keyboard shortcuts for PTZ (arrow keys for pan/tilt, +/- for zoom) | S | M |
| 13 | P3 | Add PTZ value display with live hardware readback in web UI | M | M |
| 14 | P3 | Mobile-responsive testing and polish | M | M |
| 15 | P3 | Add integration test with real hardware (guarded by build tag) | L | H |
| 16 | P3 | Replace `prometheus/client_golang` with pure OTel prometheus exporter | M | S |
| 17 | P3 | Add structured logging with configurable levels | M | M |
| 18 | P3 | Add health check endpoint for monitoring | S | M |
| 19 | P4 | Add dark/light theme toggle | S | S |
| 20 | P4 | Add camera preview snapshot download button | S | S |
| 21 | P4 | Add firmware version query | M | M |
| 22 | P4 | Add multiple camera support | L | M |
| 23 | P4 | Add configuration web UI for runtime settings | L | M |
| 24 | P4 | Add i18n/localization support | L | M |
| 25 | P4 | Add access control / authentication for web UI | L | H |

---

## G. Top #1 Question I Cannot Figure Out Myself

**Has the PTZ slider fix been tested with a real EMEET PIXY camera?**

The fix addresses the code-level bug (stale cache, no toast), but the actual user experience depends on hardware behavior:
- Does `v4l2-ctl` accept the new position fast enough that the next readback (within 2s cache TTL) returns the correct value?
- Does the camera move smoothly with the 300ms debounce, or does it stutter on rapid slider drags?
- Does the success toast ("Pan set to 50") appear at the right time relative to the physical camera movement?

These can only be verified with the physical device connected.

---

## Session Changes (uncommitted)

| File | Lines Changed | Description |
|------|--------------|-------------|
| `handlers.go` | +60 / -12 | PTZ handler rewrite: user-value response, cache invalidation, toast feedback |
| `templates.templ` | +5 | `ptzSliderWithToast` template |
| `behavior_test.go` | +109 | 2 BDD tests: PTZ web slider user-input + error-toast |

**Total:** +174 / -12 across 3 files
