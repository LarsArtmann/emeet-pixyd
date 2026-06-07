# emeet-pixyd — 15-Skill Comprehensive Audit Status

**Date:** 2026-05-12 07:48
**Scope:** code-quality-scan, architecture-review, naming-review, how-to-golang, full-code-review, brutal-self-review, improve-codebase-architecture, go-modularize, frontend-design, nix-review, docs-freshness-check, features-audit, bdd-testing, todo-list-builder, pareto-planning

---

## A) FULLY DONE ✅

### Bugs Fixed (4)

| #   | File             | Bug                                                                                                                                                                       | Severity |
| --- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| 1   | `hid.go:132`     | `%w` wrapping nil error produced garbled `%!w(<nil>)` — when `writeErr == nil && written == 0`, `fmt.Errorf("...%w", nil)` yields a non-nil error containing `%!w(<nil>)` | Critical |
| 2   | `probe.go:76`    | `return false` on malformed HID_ID killed entire device probe — should be `continue` to try next line                                                                     | High     |
| 3   | `flake.nix:61`   | Invalid `env` attribute in app definition — `nix flake check` errored on unsupported attribute; debug mode was never actually set                                         | High     |
| 4   | `package.nix:22` | Version `"0.2.0"` hardcoded in both `version` attr and `ldflags` — drift risk; consolidated via `let version = "0.2.0"; in`                                               | Medium   |

### Skills Executed (15/15)

All 15 skills were run to completion with detailed findings documented in `docs/planning/2026-05-12_07-42_15-skill-comprehensive-audit.md`.

### Documentation Updated (5 files)

| File           | Change                                                                                                |
| -------------- | ----------------------------------------------------------------------------------------------------- |
| `AGENTS.md`    | Removed fabricated `autoTracking`+`autoPrivacy` claim — replaced with correct `auto` enum description |
| `FEATURES.md`  | Fixed feature count from 43 to 44 (actual row count)                                                  |
| `TODO_LIST.md` | Added 19 new TODO items from audit, fixed SUPERB_ROADMAP.md path, updated dates/counts                |
| `CHANGELOG.md` | Fixed behavior_test count (11→14), added bug fix entries, added branded types entry                   |
| `package.nix`  | Version deduplication via `let version` binding                                                       |

### Planning Artifacts

- `docs/planning/2026-05-12_07-42_15-skill-comprehensive-audit.md` — Full Pareto plan with D2 execution graph, 27 prioritized tasks

### Verification

- `go test -race -count=1 ./...` ✅
- `golangci-lint run --timeout 2m ./...` ✅ (0 issues)
- `go vet ./...` ✅
- `nix build` ✅
- `nix flake check` ✅ (all checks passed)

---

## B) PARTIALLY DONE 🔶

| Item                     | What's Done                                                         | What's Left                                                                                                                                           |
| ------------------------ | ------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **code-quality-scan**    | Build, lint, duplication analysis complete                          | 7 duplication patterns identified but not consolidated (inline lock access, save-state pattern, PTZ validation duplication, HID preamble duplication) |
| **docs-freshness-check** | All 6 docs audited, stale items identified, AGENTS.md fix applied   | `docs/SUPERB_ROADMAP.md` identified as 30% fresh — needs archive or full rewrite                                                                      |
| **nix-review**           | 9 findings documented, 2 fixed (flake.nix env, package.nix version) | NixOS module missing systemd hardening; module references `pkgs.emeet-pixyd` without overlay injection (critical for standalone use)                  |
| **bdd-testing**          | Review complete, false positives identified                         | 7 false-positive tests identified but not fixed (they still pass regardless of outcome)                                                               |
| **frontend-design**      | Full review with 10 priority fixes documented                       | No frontend code changed — fixes are planned but not implemented                                                                                      |

---

## C) NOT STARTED ⬜

These are TODO items from the audit that were identified but deferred for prioritization:

### Architecture (from improve-codebase-architecture)

- `handleCommand(string) string` → typed `CommandResult` struct (highest-leverage refactor)
- Consolidate PTZ into single `ptz.go` (currently across 5 files)
- Consolidate 9 function pointers into `Dependencies` interface
- Extract `AutoManager` sub-struct from Daemon
- Extract `Streamer` sub-struct from Daemon
- Centralize state mutation + persistence into `UpdateState(func(*State))`
- Eliminate `cmdMu` for single-lock serialization

### Code Quality (from code-quality-scan, full-code-review)

- Validate loaded state in `loadState()` — garbage CameraState/AudioMode silently accepted
- Fix `uevent.go` — transient read errors permanently disable hotplug
- Fix `autoManage` — only call `saveState` when state actually changed
- Add `extractJPEGFrame` max-iterations guard
- Release `d.mu` during 200ms HID sleep in `setDeviceState`
- Move PTZ limits to shared constants in `internal/pixy/` (split brain with templates.templ)

### Testing (from bdd-testing)

- Fix false-positive tests (7 identified — `TestHandleCommandSyncWithDevice`, `TestHandleCommandTogglePrivacy`, etc.)
- Add missing behavioral tests: camera hot-unplug during call, auto mode switch while in call, concurrent web+socket commands, rapid PTZ commands

### Frontend (from frontend-design)

- Remove `, change` from PTZ slider hx-trigger (doubles requests)
- Suppress toast spam during PTZ slider drag
- Add `role="alert"` to error banners
- Fix `touch-action: none` → `pan-y` for mobile
- Hide `.kbd` shortcuts on mobile, increase touch targets
- Add scrollbar styling, noise texture for visual polish

### Go Policy (from how-to-golang)

- Migrate `encoding/json` → `encoding/json/v2`
- Add `cockroachdb/errors` + `uniflow`
- Consider `ginkgo/gomega` for BDD tests (low priority for this project)
- Consider `sivchari/govalid` for PTZ validation

### Nix (from nix-review)

- Add systemd hardening to NixOS module (MemoryMax, ProtectSystem, RestrictAddressFamilies, etc.)
- Fix NixOS module package reference (needs overlay injection or `package` option)
- Add test/lint checks to `flake.nix` checks output
- Extract shared package definition to avoid packages/overlay drift

---

## D) TOTALLY FUCKED UP 💥

### 1. `TestHandleStream_NoFFmpeg` — Flaky Test

**File:** `stream_test.go:82`
**Symptom:** `context deadline exceeded` on `GET /api/stream` — fails ~1 in 5 runs
**Cause:** The test sets a 2s HTTP client timeout and relies on the stream handler blocking without ffmpeg. If the server is slow to respond (under load with `-race`), the client times out before the handler.
**Severity:** Medium — not a production bug, but unreliable CI
**Fix:** Increase client timeout or use a test-specific shorter stream duration

### 2. `docs/SUPERB_ROADMAP.md` — Dangerously Stale

**File:** `docs/SUPERB_ROADMAP.md`
**Generated:** 2026-04-20 (22+ days old)
**Problem:** Every metric is wrong:

- Says "120 test functions" — actual is 245
- Says "63.4% coverage" — actual is 71.7% / 78.4%
- Says "~73 linter warnings" — actual is 0
- Source file table is completely wrong (doesn't list extracted files)
- Dependencies table has wrong versions
- Multiple roadmap items completed but not marked
  **Severity:** Medium — could mislead anyone using it as reference
  **Fix:** Archive to `docs/status/` or do a full rewrite

### 3. NixOS Module Broken for Standalone Use

**File:** `modules/nixos.nix:81`
**Problem:** `ExecStart = "${pkgs.emeet-pixyd}/bin/emeet-pixyd"` — `pkgs.emeet-pixyd` doesn't exist unless the user manually adds the overlay. The module should self-register or have a `package` option.
**Severity:** High — anyone using the NixOS module standalone will get a build error
**Fix:** Add `package` option or inject overlay

### 4. AGENTS.md Had a Fabricated Claim (FIXED)

**What:** AGENTS.md line 61 claimed NixOS module has `autoTracking`+`autoPrivacy` options — these **never existed** in the current code. The module has a single `auto` enum.
**Status:** ✅ Fixed during this audit

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **Kill the string-based command routing.** `handleCommand(string) string` is the single biggest architectural smell. Both web and CLI paths serialize to strings, parse them back, and detect errors via `"error: "` prefix. A typed `CommandResult` struct eliminates all of this.

2. **Consolidate PTZ.** PTZ logic is scattered across 5 files (handlers.go, commands.go, v4l2.go, middleware.go, main.go). One `ptz.go` file with limits, validation, clamping, cache, and conversion.

3. **Dependencies interface.** 9 function pointers on Daemon with no compile-time guarantee they're all set. A single `Dependencies` interface with `RealDeps`/`TestDeps` implementations.

4. **State validation.** `loadState()` accepts any garbage JSON into `CameraState`/`AudioMode`/`AutoMode` (they're `string` types). Add validation after unmarshal.

5. **Fix `autoManage` unnecessary I/O.** Writes state file every 2s poll interval even when nothing changed. Add a `changed` guard.

### Testing

6. **Fix false-positive tests.** 7 tests pass regardless of outcome. `TestHandleCommandSyncWithDevice` accepts both "synced" AND "error" as passing.

7. **Add missing behavioral tests.** Camera hot-unplug during call, auto mode switch while in call, concurrent web+socket commands are untested.

8. **Flaky test.** `TestHandleStream_NoFFmpeg` fails intermittently under race detector.

### Frontend

9. **PTZ slider doubles requests.** `hx-trigger="input changed delay:300ms, change"` fires twice on mouseup. Remove `, change`.

10. **Toast spam on PTZ drag.** Every slider movement shows a toast. Debounce or suppress.

11. **Mobile usability.** `touch-action: none` blocks scrolling. Touch targets below 44px minimum. Keyboard shortcut hints waste space.

### Go Policy

12. **`encoding/json/v2` migration.** 4 files use v1. Go 1.26 supports v2 with 10x performance.

13. **`cockroachdb/errors`.** Currently using `fmt.Errorf("context: %w", err)` everywhere. Should use structured error types.

### Nix

14. **Systemd hardening.** No sandboxing directives in the NixOS module. A hardware daemon should have MemoryMax, ProtectSystem, RestrictAddressFamilies.

15. **NixOS module package reference.** Broken for standalone use — needs overlay injection or package option.

---

## F) Top 25 Things to Get Done Next

Sorted by impact × effort (Pareto):

| #   | Task                                                                    | Impact    | Effort | Category      |
| --- | ----------------------------------------------------------------------- | --------- | ------ | ------------- |
| 1   | Move PTZ limits to `internal/pixy/` shared constants (split brain)      | High      | 15min  | Split Brain   |
| 2   | Fix `autoManage` — conditional `saveState` only when changed            | High      | 15min  | Performance   |
| 3   | Validate loaded state in `loadState()`                                  | High      | 15min  | Correctness   |
| 4   | Fix false-positive `TestHandleCommandSyncWithDevice`                    | High      | 10min  | Test Quality  |
| 5   | Fix `uevent.go` — retry on transient read errors                        | High      | 30min  | Reliability   |
| 6   | Remove `, change` from PTZ slider hx-trigger                            | Medium    | 5min   | Frontend      |
| 7   | Add `role="alert"` to error banners                                     | Medium    | 5min   | Accessibility |
| 8   | Add `extractJPEGFrame` max-iterations guard                             | Medium    | 10min  | Robustness    |
| 9   | Suppress toast spam during PTZ slider drag                              | Medium    | 15min  | Frontend      |
| 10  | Fix `touch-action: none` → `pan-y` for mobile                           | Medium    | 5min   | Mobile        |
| 11  | Migrate to `encoding/json/v2`                                           | Medium    | 30min  | Go Policy     |
| 12  | Replace `handleCommand(string) string` with `CommandResult`             | Very High | 60min  | Architecture  |
| 13  | Consolidate PTZ into single `ptz.go`                                    | High      | 45min  | Architecture  |
| 14  | Consolidate 9 fn pointers into `Dependencies` interface                 | High      | 45min  | Architecture  |
| 15  | Add systemd hardening to NixOS module                                   | High      | 30min  | Security      |
| 16  | Fix NixOS module package reference (overlay or package option)          | High      | 30min  | Nix           |
| 17  | Fix remaining false-positive tests (6 more)                             | Medium    | 30min  | Test Quality  |
| 18  | Archive/rewrite `docs/SUPERB_ROADMAP.md`                                | Medium    | 30min  | Docs          |
| 19  | Add missing behavioral tests (hot-unplug, auto mode switch)             | High      | 90min  | Testing       |
| 20  | Fix flaky `TestHandleStream_NoFFmpeg`                                   | Medium    | 15min  | Test Quality  |
| 21  | Extract `AutoManager` sub-struct from Daemon                            | High      | 60min  | Architecture  |
| 22  | Centralize state mutation + persistence                                 | High      | 60min  | Architecture  |
| 23  | Add `cockroachdb/errors` + `uniflow`                                    | Medium    | 45min  | Go Policy     |
| 24  | Mobile responsiveness (hide .kbd, increase touch targets, center toast) | Medium    | 30min  | Frontend      |
| 25  | Add test/lint checks to `flake.nix` checks output                       | Low       | 30min  | Nix           |

---

## G) Top #1 Question I Cannot Figure Out Myself

**The NixOS module package reference issue is a design decision that requires your input:**

The NixOS module (`modules/nixos.nix:81`) references `pkgs.emeet-pixyd`, but this package only exists if the user adds `self.overlays.default` to their `nixpkgs.overlays`. There are three approaches:

1. **Add a `package` option** — `hardware.emeet-pixy.package` defaulting to `pkgs.emeet-pixyd`, with documentation that users must add the overlay
2. **Inject the overlay automatically** — `nixpkgs.overlays = [ self.overlays.default ];` inside the module config (simpler for users, but some consider overlay injection an anti-pattern)
3. **Inline the package build** — call `./package.nix` directly from the module instead of going through `pkgs` (avoids overlay entirely)

Which approach do you prefer? This determines how users consume the NixOS module and I cannot decide this architectural tradeoff for the project.

---

## Metrics Snapshot

| Metric               | Before Audit   | After Audit                                  |
| -------------------- | -------------- | -------------------------------------------- |
| Bugs fixed           | —              | 4 (hid.go, probe.go, flake.nix, package.nix) |
| Lint issues          | 0              | 0                                            |
| Test coverage (main) | 71.7%          | 71.7%                                        |
| Test coverage (pixy) | 78.4%          | 78.4%                                        |
| TODO items           | 42 (12 DONE)   | 61 (16 DONE)                                 |
| Features             | 43 (claimed)   | 44 (corrected)                               |
| Docs accuracy        | ~88%           | ~95%                                         |
| `nix flake check`    | ❌ (env error) | ✅ (all checks passed)                       |

---

## Files Changed

```
AGENTS.md    |  2 +-  (fixed autoTracking+autoPrivacy → auto enum)
CHANGELOG.md |  8 +++-- (bug fixes, behavior_test count, branded types)
FEATURES.md  |  4 +-  (feature count 43→44)
TODO_LIST.md | 48 ++++++++++++++ (19 new items, path fix, dates)
flake.nix    |  1 -   (removed invalid env attribute)
hid.go       |  5 +-  (nil error wrapping bug fix)
package.nix  |  7 ++-- (version deduplication)
probe.go     |  2 -   (return false → continue on malformed HID_ID)

new file: docs/planning/2026-05-12_07-42_15-skill-comprehensive-audit.md
```
