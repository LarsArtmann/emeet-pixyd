# emeet-pixyd — TODO List

**Updated:** 2026-06-30 (Round 8 — comprehensive June audit, code-verified)
**Source docs verified:** all June 2026 `.md` + `.html` (status, planning, ADR, roadmap, design), cross-referenced against actual code

---

## Status Legend

- ✅ DONE — Verified in code
- 🔶 PARTIAL — Started but incomplete
- ⬜ TODO — Not started

---

## Phase 1: Quick Wins (P0)

| #   | Status  | Task                                                                                                            | Source        |
| --- | ------- | --------------------------------------------------------------------------------------------------------------- | ------------- |
| 1   | ✅ DONE | `.golangci.yml` centralized configuration                                                                       | Roadmap 2.3   |
| 2   | ✅ DONE | Fix linter suppressions (nlreturn, whitespace, goconst, perfsprint, modernize)                                  | Roadmap 7.1   |
| 3   | ✅ DONE | `CommandError` structured error type                                                                            | Roadmap 3.4   |
| 4   | ✅ DONE | `String()` method test coverage — `CameraState.String()`, `AudioMode.String()` exist but no explicit test calls | Roadmap 7.2   |
| 5   | ✅ DONE | `t.Parallel()` in all tests (only 2 justified serial tests)                                                     | Quality sweep |

## Phase 2: Decomposition (P1)

| #   | Status  | Task                                                                              | Source        |
| --- | ------- | --------------------------------------------------------------------------------- | ------------- |
| 6   | ✅ DONE | Decompose `Run()` into focused helpers                                            | Roadmap 2.1   |
| 7   | ✅ DONE | pprof endpoint gated behind `Config.Debug`                                        | Roadmap 4.3   |
| 8   | ✅ DONE | Keyboard shortcuts in web UI (T/I/P/C)                                            | Roadmap 5.2   |
| 9   | ✅ DONE | `AutoMode`/`DefaultAudio` from env vars                                           | Quality sweep |
| 10  | ✅ DONE | Uevent context cancellation (goroutine leak fix)                                  | Quality sweep |
| 11  | ✅ DONE | Device name matching shared `isPixyName()` helper                                 | Quality sweep |
| 12  | ✅ DONE | Error var consolidation (no duplicates)                                           | Quality sweep |
| 13  | ✅ DONE | Eliminate `init()` for Prometheus metrics — lazy registration via `sync.Once`     | Roadmap 2.2   |
| 14  | ✅ DONE | Structured log levels audit (standardize Debug/Info/Warn/Error usage)             | Roadmap 4.2   |
| 15  | ✅ DONE | Graceful degradation for missing optional deps (`checkExternalDeps()` at startup) | Roadmap 3.1   |

## Phase 3: Observability (P1-P2)

| #   | Status  | Task                                                                                                                  | Source      |
| --- | ------- | --------------------------------------------------------------------------------------------------------------------- | ----------- |
| 16  | ✅ DONE | Additional Prometheus metrics (command_total, probe_total, uevent_total, hid_failures, stream_duration, frames_total) | Roadmap 4.1 |
| 17  | ✅ DONE | Circuit breaker for HID failures (`hidFailCount` + `hidCircuitBreakerThreshold`)                                      | Roadmap 3.2 |
| 18  | ✅ DONE | Stream health monitoring (`metricStreamDuration` histogram + `metricFramesTotal` counter)                             | Roadmap 3.3 |
| 19  | ✅ DONE | Benchmark suite (7 benchmarks: JPEG, HID, Waybar, HandleCommand, GetWebStatus, FormatLastSynced)                      | Roadmap 6.3 |
| 20  | ✅ DONE | Continuous fuzz in CI (60s per test, store corpus, fail on crash)                                                     | Roadmap 6.2 |

## Phase 4: Architecture (P2-P3)

| #   | Status  | Task                                                                                                                                         | Source            |
| --- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- |
| 21  | ✅ DONE | Extract `CommandRunner` interface for shell commands (`commander.go`: Run/Output/LookPath; ffmpeg intentionally excluded — needs StdoutPipe) | Roadmap 1.1       |
| 22  | ✅ DONE | Extract `HIDDevice` interface for HID I/O (`Send`/`SendRecv` methods, `hidrawDevice` impl)                                                   | Roadmap 1.2       |
| 23  | ⬜ TODO | Extract `ProcessInspector` interface for /proc traversal                                                                                     | Roadmap 1.3       |
| 24  | ⬜ TODO | Extract `UeventListener` interface for netlink                                                                                               | Roadmap 1.4       |
| 25  | ✅ DONE | `probeDevices()` — pure function returning `probeResult`, applied via `applyProbeResult`                                                     | Quality sweep 4.1 |

## Phase 5: Web UI (P2-P3)

| #   | Status  | Task                                                       | Source      |
| --- | ------- | ---------------------------------------------------------- | ----------- |
| 26  | ⬜ TODO | Mobile-responsive layout                                   | Roadmap 5.3 |
| 27  | ✅ DONE | SSE for live state updates (replace 3s HTMX polling)       | Roadmap 5.1 |
| 28  | ✅ DONE | Keyboard shortcuts for PTZ (arrow keys, +/- for zoom)      | Status E.12 |
| 29  | ✅ DONE | PTZ relative mode (`pan+10`, `tilt-5`) via `parsePTZValue` | Status E.8  |
| 30  | ⬜ TODO | Camera preset support (save/recall PTZ positions)          | Status F.9  |

## Phase 6: Testing (P3-P4)

| #   | Status  | Task                                                                      | Source      |
| --- | ------- | ------------------------------------------------------------------------- | ----------- |
| 31  | ⬜ TODO | Integration test harness with fake devices                                | Roadmap 6.1 |
| 32  | ⬜ TODO | Test coverage for `stream.go`, `process.go`, `hid.go` real hardware paths | Status E.2  |
| 33  | ✅ DONE | Surface auto-manage errors to web UI (`autoError` field + `errors.Join`)  | Status E.3  |
| 34  | ⬜ TODO | Improve MJPEG stream reconnection                                         | Status E.4  |
| 35  | ⬜ TODO | Integration test with real hardware (build tag guarded)                   | Status F.15 |

## Phase 7: Code Nits (from this review)

| #   | Status  | Task                                                                                  | Source     |
| --- | ------- | ------------------------------------------------------------------------------------- | ---------- |
| 36  | ✅ DONE | Extract toast type from `actionToast`, propagate through `applyResponseToStatus`      | Review L2  |
| 37  | ✅ DONE | Extract `lastFrame`/`ptzCache` into named types in `cache.go`                         | Review M1  |
| 38  | ✅ DONE | Moved `streamBufSize`/`ffmpegShutdownTimeout` constants from handlers.go to stream.go | Review L12 |
| 39  | ✅ DONE | Removed decorative blank lines in stream.go select/case blocks                        | Review M5  |
| 40  | ✅ DONE | Update `SUPERB_ROADMAP.md` — completion status table added, marked archived           | Review M4  |
| 41  | ✅ DONE | Consolidate PTZ axis dispatch into `ptzAxes` lookup table                             | Review     |
| 42  | ⬜ TODO | PTZ readback accuracy — delay before readback or maintain in-memory "last set" value  | Status E.1 |

## Phase 8: From 15-Skill Comprehensive Audit (2026-05-12)

|     | #       | Status                                                                                                   | Task            | Source |
| --- | ------- | -------------------------------------------------------------------------------------------------------- | --------------- | ------ |
| 43  | ✅ DONE | Fix `hidSendRecv` nil error wrapping bug (zero-write produces `%!w(<nil>)`)                              | Code Review C1  |
| 44  | ✅ DONE | Fix `hasPixyVendorProduct` — `return false` → `continue` on malformed HID_ID                             | Code Review C4  |
| 45  | ✅ DONE | Fix `flake.nix` — remove invalid `env` attribute from app definition                                     | Nix Review      |
| 46  | ✅ DONE | Fix `package.nix` — deduplicate version string via `let version` binding                                 | Nix Review      |
| 47  | ✅ DONE | Fix `autoManage` — only call `saveState` when state actually changed                                     | Self-Review 4   |
| 48  | ✅ DONE | Validate loaded state in `loadState()` — reject garbage CameraState/AudioMode/AutoMode values            | Code Review C2  |
| 49  | ✅ DONE | Fix `uevent.go` — transient read errors permanently disable hotplug, added retry with continue           | Code Review C5  |
| 50  | ✅ DONE | Moved PTZ limits to shared constants in `internal/pixy/` (eliminated split brain with templates)         | Self-Review S1  |
| 51  | ✅ DONE | Consolidate 10 DI function pointers into `Dependencies` struct in `deps.go`                              | Architecture 3  |
| 52  | ✅ DONE | Replace `handleCommand(string) string` with typed `CommandResult` struct                                 | Architecture 2  |
| 53  | ✅ DONE | Consolidate PTZ logic into `ptz.go` (extracted from handlers.go + v4l2.go, v4l2.go deleted)              | Architecture 3  |
| 54  | ✅ DONE | Added systemd hardening to NixOS module (ProtectSystem, PrivateTmp, NoNewPrivileges, MemoryMax)          | Nix Review H2   |
| 55  | ✅ DONE | Fixed false-positive tests — proper assertions for sync/toggle-privacy commands                          | BDD Review P0   |
| 56  | ✅ DONE | Removed `, change` from PTZ slider hx-trigger (was doubling requests)                                    | Frontend Review |
| 57  | ✅ DONE | Suppress toast spam during PTZ slider drag (empty toast on success)                                      | Frontend Review |
| 58  | ✅ DONE | Added `role="alert"` to error banners for screen reader announcement                                     | Frontend A11y   |
| 59  | ❌ SKIP | `encoding/json/v2` not available in Go 1.26.2 stdlib — revisit when landed                               | How-to-Go       |
| 60  | ✅ DONE | Added `extractJPEGFrame` max-iterations guard (10M) to prevent infinite loop on corrupt stream           | Self-Review 4.8 |
| 62  | ✅ DONE | Enrich `PTZValues` with `Get(axis)`/`Set(axis, val)` methods, eliminate all hardcoded V4L2 control names | Session 7       |

## Phase 9: Post-cqrs-htmx Removal Hardening (2026-06-21)

|     | #          | Status                                                                                                                                                                                                                                                                                                               | Task        | Source |
| --- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ------ |
| 63  | ✅ DONE    | Removed `cqrs-htmx/v2` dependency (~30 transitive deps eliminated)                                                                                                                                                                                                                                                   | Session 9   |
| 64  | ✅ DONE    | Reimplemented SSE `Broadcaster` + `sseStream` + middleware locally in `sse.go`/`http.go`                                                                                                                                                                                                                             | Session 9   |
| 65  | ✅ DONE    | Fixed nix build: updated `vendorHash` in `package.nix` + `flake.nix`                                                                                                                                                                                                                                                 | Session 9   |
| 66  | ✅ DONE    | Embedded HTMX v2.0.9 JS in `static/htmx.js` (was via `cqrshtmx.HTMXScriptHandler`)                                                                                                                                                                                                                                   | Session 9   |
| 67  | ✅ DONE    | Split `sse.go` → `sse.go` (SSE-only) + `http.go` (HTTP helpers + middleware)                                                                                                                                                                                                                                         | Session 9   |
| 68  | ✅ DONE    | Added 9 SSE unit tests (`writeSSEEvent`, `Broadcaster`, `splitSSELines`)                                                                                                                                                                                                                                             | Session 9   |
| 69  | ✅ DONE    | CI cleanup: removed `GOPRIVATE`, added `FuzzParsePTZValue`, added `nix flake check`                                                                                                                                                                                                                                  | Session 9   |
| 70  | ✅ DONE    | Updated CHANGELOG + AGENTS.md with cqrs-htmx removal docs                                                                                                                                                                                                                                                            | Session 9   |
| 71  | ✅ DONE    | `CommandRunner` interface wired (`commander.go`) — centralized subprocess logging via `realCommandRunner`, `noopCommandRunner` for tests. ffmpeg excluded by design (streaming API). NOTE: a broader "unified Commander facade" was mooted but the testability+logging goal is fully achieved — see `deps.go` wiring | Session 9   |
| 72  | ⬜ TODO    | Define + wire `ProcessInspector` interface for /proc traversal                                                                                                                                                                                                                                                       | Roadmap 1.3 |
| 73  | ⬜ TODO    | Define + wire `UeventListener` interface for netlink                                                                                                                                                                                                                                                                 | Roadmap 1.4 |
| 74  | ⬜ TODO    | Camera preset support (save/recall PTZ positions)                                                                                                                                                                                                                                                                    | Status F.9  |
| 75  | ⬜ TODO    | Mobile-responsive layout                                                                                                                                                                                                                                                                                             | Roadmap 5.3 |
| 76  | ⬜ TODO    | Fake device test infrastructure (fake HID + fake video)                                                                                                                                                                                                                                                              | Roadmap 6.1 |
| 77  | ⬜ TODO    | PTZ readback accuracy — in-memory last-set value                                                                                                                                                                                                                                                                     | Status E.1  |
| 78  | ⬜ TODO    | MJPEG stream reconnection with backoff                                                                                                                                                                                                                                                                               | Status E.4  |
| 79  | ⬜ TODO    | Evaluate removing `prometheus/client_golang` (only `promhttp.Handler` used)                                                                                                                                                                                                                                          | Session 9   |
| 80  | 🟡 PARTIAL | Move `SSEEvent` + `toastType` to `internal/pixy` (domain types) — `toastType` already moved to `web_types.go`; `SSEEvent` kept in `sse.go` by design decision (transport-layer DTO, not domain)                                                                                                                      | Session 9   |
| 81  | ⬜ TODO    | Integration test with real hardware (build tag guarded)                                                                                                                                                                                                                                                              | Status F.15 |

---

## Phase 10: June 2026 Comprehensive Audit (verified against code)

All items below were extracted by reading **every `.md`/`.html` doc from June 2026**, then **verified against the actual code** on 2026-06-30. Stale items found during verification are noted. Build confirmed green: `go test -race -count=1 ./...` passes, `golangci-lint` reports 0 issues.

### Stale items corrected in this audit (were TODO, actually DONE)

| Was                                                                                        | Now        | Evidence                                                                                                        |
| ------------------------------------------------------------------------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------- |
| #21 / #71 Commander interface                                                              | ✅ DONE    | `commander.go`: `CommandRunner` interface (Run/Output/LookPath), wired in `deps.go`. ffmpeg excluded by design. |
| #80 toastType move                                                                         | 🟡 PARTIAL | `toastType` moved to `web_types.go`; `SSEEvent` kept in `sse.go` by design decision (transport DTO).            |
| `setSource`/`findSource` error surfacing (old session-10 net-new)                          | ✅ DONE    | `auto.go:44-55` appends source errors to `errs`, joins into `autoError`, surfaced in web UI (`handlers.go:95`). |
| `TestHandleStream_NoFFmpeg` hang on hardware (old session-11 bug)                          | ✅ FIXED   | `stream_test.go:66-86` accepts 503 or 200, bounded by 2s context timeout — no longer hangs.                     |
| Continuous fuzz in CI / `nix flake check` in CI / govulncheck in CI (old net-new CI items) | ✅ DONE    | `.github/workflows/go-test.yml`: nix flake check (L45), govulncheck (L35), 4 fuzz targets 60s each (L59).       |
| State-race in parallel tests (`newTestDaemon` shared `/tmp` state file)                    | ✅ DONE    | commit `241348f` — `newTestDaemon` takes `testing.TB`, uses `tb.TempDir()`. Stress-tested `-race -count=10`.    |

### Genuinely open — Architecture & testability

|     | #       | Status                                                                                                               | Task                                      | Source / Evidence |
| --- | ------- | -------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- | ----------------- |
| 82  | ⬜ TODO | Extract `ProcessInspector` interface for `/proc` traversal (enables hardware-free testing)                           | Roadmap 1.3; absent in code               |
| 83  | ⬜ TODO | Extract `UeventListener` interface for netlink (enables hardware-free testing)                                       | Roadmap 1.4; absent in code               |
| 84  | ⬜ TODO | Split `cmdMu` from HID I/O serialization — 200ms HID sleep & v4l2-ctl subprocess block ALL mutating commands         | Sessions 10-11; deferred (4h design pass) |
| 85  | ⬜ TODO | Move `main.go` → `cmd/emeet-pixyd/main.go` (go-structure-linter CRITICAL) — author leans against; unresolved tension | Sessions 9-11                             |
| 86  | ⬜ TODO | `Daemon` struct (17 fields) decomposition — extract `broadcaster`+`cache` sub-structs. Low value; accepted as-is     | Session 9; verified 17 fields             |
| 87  | ⬜ TODO | `PTZValues.Get(axis) int` → `(int, bool)` — silent zero on unknown axis is a latent bug                              | Session 10; `pixy.go:275`                 |

### Genuinely open — Features & UX

|     | #          | Status                                                                                                                                | Task                      | Source / Evidence |
| --- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- | ----------------- |
| 88  | ⬜ TODO    | Camera preset support (save/recall PTZ positions: type + state.json + CLI + web UI + tests)                                           | Roadmap; not started      |
| 89  | 🟡 PARTIAL | Mobile-responsive layout — `@media (max-width:720px)` EXISTS (`style.css:110`) but incidental, never designed/tested on small screens | Roadmap 5.3; verified CSS |
| 90  | ⬜ TODO    | MJPEG stream reconnection — exponential backoff + max-retry cap (SSE has backoff; MJPEG preview lacks it)                             | `app.js` retry has no cap |

### Genuinely open — Quality & correctness

|     | #       | Status                                                                                                                                                           | Task                                     | Source / Evidence |
| --- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- | ----------------- |
| 91  | ⬜ TODO | PTZ readback accuracy — cache holds _requested_ value; if hardware rejects/rounds, UI shows a lie. Maintain authoritative in-memory last-set OR poll-after-delay | Status E.1; `cache.go` Set updates cache |
| 92  | ⬜ TODO | Structured log-levels audit — conventions exist in AGENTS.md but misclassification audit (e.g. stream pipe err as Debug) not done                                | Roadmap 4.2; modularity.html #14         |

### Genuinely open — Testing

|     | #       | Status                                                                                                                         | Task                                                   | Source / Evidence |
| --- | ------- | ------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------ | ----------------- |
| 93  | ⬜ TODO | Fake device test harness (fake HIDDevice + fake video4linux) → enables real-path coverage in `stream.go`/`process.go`/`hid.go` | Roadmap 6.1; many tests still default to real `/dev/*` |
| 94  | ⬜ TODO | Integration test with real hardware (`//go:build integration` tag guard)                                                       | Status F.15                                            |
| 95  | ⬜ TODO | Coverage threshold enforcement in CI — currently prints coverage but enforces no minimum                                       | Session 9; `go-test.yml:43` only prints                |

### Genuinely open — Dependencies / modernization

|     | #                     | Status                                                                                           | Task                       | Source / Evidence |
| --- | --------------------- | ------------------------------------------------------------------------------------------------ | -------------------------- | ----------------- |
| 96  | ⬜ TODO               | Remove `prometheus/client_golang` — only `promhttp.Handler` used; replace with OTel HTTP handler | `go.mod:9` still present   |
| 97  | ❌ SKIP               | `encoding/json/v2` — still needs `goexperiment.jsonv2` build tag; not usable until Go stabilizes | How-to-Go; verified absent |
| 98  | 🟢 DECIDED (won't-do) | Move `SSEEvent` to `internal/pixy` — author decision: keep as transport DTO in `sse.go`          | Session 9 §g               |

### Genuinely open — Build / CI hardening

|     | #       | Status                                                                                                                    | Task                                               | Source / Evidence |
| --- | ------- | ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- | ----------------- |
| 99  | ⬜ TODO | Post-`templ generate` validation — detect empty/EOF `*_templ.go` (latent flake hit on 2026-06-28; root cause never found) | 2026-06-28 status; `templates_templ.go` gitignored |
| 100 | ⬜ TODO | `templ` CLI version alignment — CI uses `@latest`, nix pins a version, go.mod has another. Drift warning in build logs    | 2026-06-20 status                                  |
| 101 | ⬜ TODO | Resolve `go-error-family` direct-vs-indirect dependency warning (transitive via go-branded-id)                            | Session 9; pre-commit warns                        |

### Genuinely open — Docs / cleanup drift (found in this audit)

|     | #       | Status                                                                                                                                                                          | Task                                      | Source / Evidence |
| --- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- | ----------------- |
| 102 | ⬜ TODO | `DESIGN.md` inaccuracies — claims `httputil` middleware library + "OTel-only metrics", but `prometheus/client_golang` is still a dep and middleware is local                    | `DESIGN.md:107-108` vs `go.mod:9`         |
| 103 | ⬜ TODO | `README.md` source layout lists deleted `v4l2.go` (PTZ logic moved to `ptz.go`)                                                                                                 | `README.md:199`                           |
| 104 | ⬜ TODO | `CHANGELOG.md` `[Unreleased]` missing 2026-06-28/29 work: makezero `always:false` fix, Go 1.26.4 dep upgrade vendorHash, state-race fix (`testing.TB`), templ regeneration note | `CHANGELOG.md` stops at cqrs-htmx removal |
| 105 | ⬜ TODO | NixOS module nits: `RestartSec="3"` should be integer; `v4l-utils` redundant in both `systemPackages` and `path`                                                                | `modules/nixos.nix:60,99,118`             |

### Low-value enhancements (brainstormed, consider only)

These are genuinely optional ideas surfaced across June status docs. Listed for completeness, not as committed work.

- Add `pan`/`tilt` to Waybar output (`waybar.go` currently only camera/audio/auto in tooltip)
- PTZ patrol/sweep mode (automatic periodic pan)
- `koanf` layered config (file + env, replacing env-only)
- OpenTelemetry tracing (not just metrics) — trace PTZ command latency
- SSE heartbeat (prevent proxy idle kills) + `LastEventID` replay after reconnect
- HTTP panic-recovery middleware
- Camera diagnostics endpoint (full V4L2 control dump)
- PTZ movement speed control; configurable home position
- Property-based test for `Range.Clamp` (gopter)
- Benchmark PTZ and auto-manage paths in CI

---

## Docs Verified

| File                                          | Status                                             |
| --------------------------------------------- | -------------------------------------------------- |
| AGENTS.md                                     | ✅ Current as of 2026-06-29                        |
| FEATURES.md                                   | ✅ Re-audited against code 2026-06-30              |
| docs/SUPERB_ROADMAP.md                        | ✅ Archived — completion status added 2026-06-05   |
| README.md                                     | 🟡 Minor drift: lists deleted `v4l2.go` (see #103) |
| CHANGELOG.md                                  | 🟡 Stale: missing 2026-06-28/29 work (see #104)    |
| DESIGN.md                                     | 🟡 Drift: claims httputil + OTel-only (see #102)   |
| All June 2026 status/planning `.md` + `.html` | ✅ Read & cross-referenced in this audit           |

## Summary

> Phase 10 supersedes and de-duplicates the open items from earlier phases (e.g. #23/#72→#82, #30/#74→#88). Counts below reflect unique actionable items.

|                       | Status                                                                                   | Count |
| --------------------- | ---------------------------------------------------------------------------------------- | ----- |
| ✅ DONE               | 57                                                                                       |
| 🟡 PARTIAL            | 2 (#80, #89)                                                                             |
| 🟢 DECIDED (won't-do) | 1 (#98)                                                                                  |
| ❌ SKIP               | 1 (#97)                                                                                  |
| ⬜ TODO               | 21                                                                                       |
| **Total**             | 82 unique actionable (items #1–#105, with #61 omitted) + ~10 unnumbered brainstorm ideas |

**Open high-impact items:** ProcessInspector/UeventListener interfaces (#82/#83), cmdMu split (#84), camera presets (#88), fake-device test harness (#93), templ-gen hardening (#99). Nothing blocks release.
