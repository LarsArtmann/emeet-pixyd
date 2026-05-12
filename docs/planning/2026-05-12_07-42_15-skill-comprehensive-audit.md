# emeet-pixyd — 15-Skill Comprehensive Audit Plan

**Generated:** 2026-05-12
**Scope:** code-quality-scan, architecture-review, naming-review, how-to-golang, full-code-review, brutal-self-review, improve-codebase-architecture, go-modularize, frontend-design, nix-review, docs-freshness-check, features-audit, bdd-testing, todo-list-builder, pareto-planning

---

## Executive Summary

15 skill audit completed on emeet-pixyd. **4 critical bugs fixed** during audit. **19 new TODO items** identified across 7 categories. Project is in excellent shape — 0 lint issues, all tests pass, clean nix build — but has architectural debt in the `Daemon` god-struct, stringly-typed command routing, and test quality gaps.

### Current State

| Metric | Value |
|--------|-------|
| Source LOC | ~9,667 (incl. internal/pixy) |
| Test LOC | ~4,704 |
| Test-to-source ratio | 0.49:1 |
| Coverage (main) | 71.7% |
| Coverage (pixy) | 78.4% |
| Lint issues | 0 |
| Race detector | Clean |
| Nix build | ✅ Pass |
| Nix flake check | ✅ Pass |
| Features | 44 (all FULLY_FUNCTIONAL) |

---

## Pareto Breakdown

### 1% → 51% Impact (Foundational — Do First)

| # | Task | Category | Effort | Impact |
|---|------|----------|--------|--------|
| 1 | Move PTZ limits to `internal/pixy/` shared constants | Split Brain | 15min | Eliminates template/code desync risk |
| 2 | Fix `autoManage` — conditional `saveState` only when changed | Performance | 15min | Eliminates 30 unnecessary disk writes/min |
| 3 | Validate loaded state in `loadState()` | Correctness | 15min | Prevents garbage state from persisted JSON |
| 4 | Fix false-positive test `TestHandleCommandSyncWithDevice` | Test Quality | 10min | Makes test actually catch bugs |
| 5 | Fix `uevent.go` — retry on transient read errors | Reliability | 30min | Prevents permanent hotplug disable |

### 4% → 64% Impact (High Leverage)

| # | Task | Category | Effort | Impact |
|---|------|----------|--------|--------|
| 6 | Replace `handleCommand(string) string` with `CommandResult` struct | Architecture | 60min | Type-safe command routing, kills `IsCommandErrorResponse` |
| 7 | Consolidate PTZ into single `ptz.go` file | Architecture | 45min | One place to understand PTZ, eliminates duplicate validation |
| 8 | Consolidate 9 function pointers into `Dependencies` interface | Architecture | 45min | Compile-time safety, simpler test setup |
| 9 | Remove `, change` from PTZ slider trigger | Frontend | 5min | Eliminates doubled requests |
| 10 | Suppress toast spam on PTZ slider drag | Frontend | 15min | Usability fix |
| 11 | Migrate to `encoding/json/v2` | Go Policy | 30min | 10x JSON performance |
| 12 | Add `role="alert"` to error banners | Accessibility | 5min | Screen reader support |
| 13 | Add `extractJPEGFrame` max-iterations guard | Robustness | 10min | Prevents infinite loop on corrupt stream |

### 20% → 80% Impact (Broad Value)

| # | Task | Category | Effort | Impact |
|---|------|----------|--------|--------|
| 14 | Add systemd hardening to NixOS module | Security | 30min | MemoryMax, ProtectSystem, RestrictAddressFamilies |
| 15 | Add missing behavioral tests (hot-unplug, auto mode switch) | Testing | 90min | Covers critical untested user journeys |
| 16 | Archive/rewrite `docs/SUPERB_ROADMAP.md` | Docs | 30min | Eliminates dangerously stale reference |
| 17 | Fix false-positive tests (toggle privacy, gesture endpoint) | Testing | 30min | Makes 5+ tests actually verify behavior |
| 18 | Mobile responsiveness fixes (touch-action, touch targets, hide kbd) | Frontend | 30min | Usable on mobile |

### Remaining 80% (Polish / Lower Priority)

| # | Task | Category | Effort |
|---|------|----------|--------|
| 19 | Extract `AutoManager` sub-struct from Daemon | Architecture | 60min |
| 20 | Extract `Streamer` sub-struct from Daemon | Architecture | 45min |
| 21 | Centralize state mutation + persistence | Architecture | 60min |
| 22 | Eliminate `cmdMu` for single-lock serialization | Architecture | 60min |
| 23 | Add `cockroachdb/errors` + `uniflow` | Go Policy | 45min |
| 24 | Add visual depth to preview card (frontend polish) | Frontend | 15min |
| 25 | Add custom scrollbar styling | Frontend | 5min |
| 26 | Add keyboard shortcut discoverability (? button) | Frontend | 15min |
| 27 | Predictable request IDs → use crypto/rand | Security | 10min |

---

## Bugs Fixed During Audit

| # | File | Bug | Fix |
|---|------|-----|-----|
| 1 | `hid.go:132` | `%w` wrapping nil error produces garbled `%!w(<nil>)` | Split into two conditions |
| 2 | `probe.go:76` | `return false` on malformed HID_ID kills probe | Changed to `continue` |
| 3 | `flake.nix:61` | Invalid `env` attribute in app definition | Removed `env` |
| 4 | `package.nix:22` | Version string duplicated in ldflags | `let version` binding |

---

## Key Findings by Skill

### code-quality-scan
- Build: ✅ Clean
- Lint: ✅ 0 issues
- Duplication: 7 patterns found (inline lock access, save-state pattern, PTZ validation duplication, HID preamble duplication)

### architecture-review
- Daemon is a god-struct with 12+ responsibilities
- String-based command routing is the shallowest module
- PTZ logic scattered across 5 files
- 9 function pointers are ad-hoc, no compile-time guarantee
- webServer is pass-through, not earning its abstraction

### naming-review
- Overall: Above average. No Manager/Handler/Helper classes
- `handleCallStart`/`handleCallEnd` — imply telephony events that don't exist
- `checkDevice` — writes HTTP error as side effect (should be `requireDevice`)
- `stateSetter` — vague noun
- `debounceInUse`/`debounceIdle` — names imply booleans, values are counters

### how-to-golang
- 70% compliant
- Violations: `encoding/json` v1 (should be v2), `prometheus/client_golang` (transitional)
- Missing: `cockroachdb/errors`, `ginkgo/gomega`, `govalid`
- HTTP router: ✅ correct (`net/http.ServeMux`)
- Logging: ✅ correct (`slog`)
- ID types: ✅ correct (`go-branded-id`)

### full-code-review
- 10 critical/high/medium issues found
- Top: nil error wrapping, state validation gap, InCall stuck state, uevent permanent disable
- Test coverage gaps: streaming, process detection, socket server, middleware

### brutal-self-review
- PTZ limits split brain (handlers.go vs templates.templ)
- `autoManage` writes state every 2s even when unchanged
- `init()` for metrics is still stupid (acknowledged)
- AGENTS.md had fabricated `autoTracking+autoPrivacy` claim — **fixed**
- SUPERB_ROADMAP.md is a ghost (dangerously stale)

### improve-codebase-architecture
- 8 deepening opportunities identified
- Highest leverage: typed command methods + PTZ consolidation + Dependencies interface
- webServer should be absorbed or given a proper StatusProvider interface

### go-modularize
- **Recommendation: Do NOT modularize**
- Single-purpose hardware daemon, no reuse target
- The one natural boundary (`internal/pixy`) is already extracted

### frontend-design
- Visual: B+ (cohesive glassmorphism, needs depth layering)
- UX: B (PTZ slider doubles requests, toast spam on drag)
- Accessibility: C+ (missing ARIA live regions, keyboard shortcut discoverability)
- Mobile: C+ (touch-action blocks scrolling, tiny touch targets)

### nix-review
- Critical: NixOS module references `pkgs.emeet-pixyd` without overlay (needs package option or overlay injection)
- High: No systemd hardening
- Medium: Hardcoded/duplicated version (fixed), `env` in app (fixed)

### bdd-testing
- Rating: 7.5/10
- 14 behavioral tests, well-structured
- 7 false-positive tests that pass regardless of outcome
- Missing: hot-unplug, concurrent commands, stream lifecycle

---

## D2 Execution Graph

```d2
direction: down

title: emeet-pixyd 15-Skill Audit — Execution Graph

# 1% → 51%
ptz_constants: "PTZ limits to shared constants" {
  shape: task
}
save_state_conditional: "autoManage: conditional saveState" {
  shape: task
}
validate_loaded_state: "loadState: validate enums" {
  shape: task
}
fix_false_positive_test: "Fix TestHandleCommandSyncWithDevice" {
  shape: task
}
fix_uevent_retry: "uevent: retry on transient errors" {
  shape: task
}

# 4% → 64%
command_result: "handleCommand → CommandResult" {
  shape: task
}
consolidate_ptz: "PTZ → single ptz.go" {
  shape: task
}
deps_interface: "9 fn pointers → Dependencies interface" {
  shape: task
}
fix_ptz_trigger: "Remove ,change from PTZ trigger" {
  shape: task
}
suppress_ptz_toast: "Suppress toast spam on PTZ drag" {
  shape: task
}
json_v2: "encoding/json → json/v2" {
  shape: task
}
aria_alert: "role=alert on error banners" {
  shape: task
}
jpeg_max_iter: "extractJPEGFrame max-iterations" {
  shape: task
}

# 20% → 80%
systemd_harden: "Systemd hardening in NixOS module" {
  shape: task
}
missing_tests: "Missing behavioral tests" {
  shape: task
}
archive_roadmap: "Archive/rewrite SUPERB_ROADMAP.md" {
  shape: task
}
fix_more_tests: "Fix more false-positive tests" {
  shape: task
}
mobile_fixes: "Mobile responsiveness fixes" {
  shape: task
}

# Dependencies
command_result -> deps_interface
consolidate_ptz -> command_result
fix_ptz_trigger -> suppress_ptz_toast

# Grouping
tier1 -> ptz_constants -> save_state_conditional -> validate_loaded_state -> fix_false_positive_test -> fix_uevent_retry
tier2 -> command_result -> consolidate_ptz -> deps_interface -> fix_ptz_trigger -> json_v2
tier3 -> systemd_harden -> missing_tests -> fix_more_tests -> mobile_fixes

tier1: "1% → 51% Impact" { style.fill: "#1a6334" }
tier2: "4% → 64% Impact" { style.fill: "#2d6a9f" }
tier3: "20% → 80% Impact" { style.fill: "#6b4c9a" }
```

---

## What Was Done During This Audit

1. ✅ code-quality-scan — build, lint, duplication analysis
2. ✅ architecture-review — god-struct, coupling, module boundaries
3. ✅ naming-review — honesty, clarity, domain alignment, consistency
4. ✅ how-to-golang — banned deps, required libs, rule compliance
5. ✅ full-code-review — visited every file, found 10 issues
6. ✅ brutal-self-review — 11 honesty questions answered
7. ✅ improve-codebase-architecture — 8 deepening opportunities
8. ✅ go-modularize — recommendation: do NOT modularize
9. ✅ frontend-design — visual/UX/a11y/perf/mobile review
10. ✅ nix-review — 9 findings (1 critical)
11. ✅ docs-freshness-check — 6 docs audited, found fabricated claim in AGENTS.md
12. ✅ features-audit — 44 features verified (count was wrong: said 43)
13. ✅ bdd-testing — 7.5/10, found 7 false-positive tests
14. ✅ todo-list-builder — 61 items total (16 DONE, 3 PARTIAL, 42 TODO)
15. ✅ pareto-planning — this document

### Bugs Fixed (4)

- `hid.go` — nil error wrapping bug
- `probe.go` — malformed HID_ID kills probe
- `flake.nix` — invalid app attribute
- `package.nix` — version string duplication

### Docs Updated (5)

- `AGENTS.md` — fixed fabricated `autoTracking+autoPrivacy` claim
- `FEATURES.md` — fixed feature count (43 → 44)
- `TODO_LIST.md` — added 19 new items, fixed SUPERB_ROADMAP path, updated dates
- `CHANGELOG.md` — added branded types entry, fixed behavior_test count (11 → 14), added bug fixes
- `package.nix` — version deduplication
