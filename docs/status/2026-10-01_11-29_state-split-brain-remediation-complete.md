# Status Report — State / Split-Brain Remediation COMPLETE (2026-10-01 11:29 CEST)

**Scope:** Execute the SUPERB plan `docs/planning/2026-10-01_04-24_SUPERB-state-split-brain-remediation-plan.md` (M1–M24) against the 11 findings in `docs/status/2026-10-01_04-21_state-bug-split-brain-audit.md`, and verify end to end.
**Guiding rule:** *Do not verschlimmbessern* — reproduce first, fix only what is proven, keep every change no worse than found.
**End state:** All 24 medium tasks resolved (fixed-with-test or documented-intentional). Full gate green: `go test -race` (2 packages), `golangci-lint` 0 issues, `nix build`, `templ generate` no diff.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| a1 | Findings 1/2 fixed: camera intent decoupled from connectivity; `hadPersistedState` → derived `persistedIntent` | `device.go`, `probe.go`, `state.go`, `main.go`; ADR `2026-10-01_camera-intent-vs-connectivity`; `state_split_brain_test.go` |
| a2 | Finding 3 fixed: `syncState` serializes under `hidMu`; `syncStateLocked` for callers already holding it (also fixes a latent nil-deref when only video is present) | `device.go:230-247`; `TestHIDSerialization_ConcurrentSyncAndAutoManage` |
| a3 | Finding 4 fixed: `powerStatus` caches authoritative failures with a short TTL; invalidation on presence change | `power.go`; `TestPowerStatus_FailureIsCached` |
| a4 | Finding 5 fixed: `Online` (video node) vs `Controllable` (HID node) threaded through probe → status → web/JSON | `device.go` `devicePresence`, `handlers.go`, `web_types.go`, `templates.templ` |
| a5 | Finding 6/11 fixed: metrics no longer derive connectivity from the camera gauge — new `emeet_pixyd_online` gauge; `camera_state` = intent only | `metrics.go`; `TestUpdateMetrics` |
| a6 | Finding 7 fixed: `ptzCache`/`powerCache`/`lastFrame` invalidated in one place on a presence change | `probe.go`, `cache.go`; `TestProbeResultInvalidatesCachesOnPresenceChange` |
| a7 | Finding 10 fixed: ghost `ProcessInspector` interface + `procInspector` DI field retired | `process.go`, `deps.go`, `commander.go`, `fake_device_test.go` |
| a8 | M11: one canonical `statusSnapshot`/`snapshot()` feeds CLI, Waybar, web panel, `/api/status`, health | `device.go` `statusSnapshot`; `handlers.go`, `waybar.go`, `device.go` `getStatus` |
| a9 | M18: lock-contract table documented for every HID/v4l2 entry point | `AGENTS.md` §Concurrency model |
| a10 | M19: findings + open remainder filed (#182 in `TODO_LIST.md`); CHANGELOG [Unreleased] Fixed entries | `TODO_LIST.md`, `CHANGELOG.md` |
| a11 | M20: ADR written (connectivity vs intent) | `docs/adr/2026-10-01_camera-intent-vs-connectivity.md` |
| a12 | M21: `AGENTS.md` state/architecture/concurrency/testing bullets + `docs/DOMAIN_LANGUAGE.md` glossary updated | both files |
| a13 | M23: `lastFrameCache` invalidation audited — cleared on device removal, deliberately retained on stream stop (snapshot feature) | `probe.go:348`, `cache.go` `Clear()` |
| a14 | Regression suite added (11 tests) incl. the simulator concurrency high-water detector and an injectable-probe autoManage-appear test | `state_split_brain_test.go`, `pixy_simulator_test.go` (`MaxInFlight`) |
| a15 | M7: probing made injectable (`Dependencies.probeDevices`) so every device-appear path is testable; autoManage-appear reconcile pinned | `deps.go`, `auto.go`, `TestAutoManage_DeviceAppearsRunsReconcile` |

## b) PARTIALLY DONE / DELIBERATELY SCOPED

| # | Item | State | Rationale |
|---|------|-------|-----------|
| b1 | M7 — "unify the three device-appear paths behind one entry" | The three paths (startup, uevent, autoManage) already call the single `reconcileOnDeviceAppear`; no further extraction | A `probeAndReconcile` helper would only be used by 1 of 3 (uevent needs the probe + `v4l2Mu` + udev hint) — extracting it would obscure, not clarify. The autoManage call site is not directly testable because `probeDevices` is not injectable; reconcile behavior itself is covered by `reconcile_test.go` |
| b2 | M13 — `sync` reconciling Speeds/TrackMode | Documented as NOT done (blocked on #166 readback) | A readback with an unknown speed unit / unreliable mode query would risk overwriting good intent with garbage. Filed as TODO #182 |
| b3 | M14 — breaker reset policy | Documented + tested, not rewired | The probe-success reset is intentional (CHANGELOG); pinned by `TestCircuitBreaker_ProbeResetsStrikesOnlyWhenPresent` |

## c) FINDINGS → DISPOSITION (M22 retro intent-check)

| Finding | Disposition | Where |
|---------|-------------|-------|
| 1 Camera overloaded with offline | FIXED (projection removed; intent-only field + presence) | ADR, `probe.go`, tests |
| 2 `hadPersistedState` stale flag | FIXED (derived `persistedIntent`) | `state.go`, `main.go` |
| 3 `syncState` HID without `hidMu` | FIXED | `device.go`, concurrency test |
| 4 `powerStatus` failure caching | FIXED | `power.go`, test |
| 5 Online semantics | FIXED (`Online`/`Controllable`) | `device.go`, handlers |
| 6 projection alignment | DISSOLVED — one canonical `snapshot()` | `device.go` |
| 7 cache invalidation | FIXED | `probe.go`, test |
| 8 sync coverage | DOCUMENTED-intentional (#182) | `device.go` comment |
| 9 InCall persistence | DECIDED + documented + tested (kept, reset on start) | `pixy.go`, `NewDaemon` test |
| 10 ghost ProcessInspector + breaker reset | FIXED (ghost removed) / DOCUMENTED (reset intentional) | `process.go`, test |
| 11 metrics gauges | FIXED (`emeet_pixyd_online`) | `metrics.go`, test |

**No finding remains "unknown/filed-but-unanalyzed."** Every one is either fixed with a test or explicitly documented as an intentional tradeoff with a test pinning the behavior.

## d) WHAT I'D DO DIFFERENTLY (honest)

1. **Mid-flight, I invented a display projection (`displayCamera` collapsing camera→offline everywhere) that broke the pre-existing JSON/Waybar contract** and cost a detour to unwind. The right move was to read the *tests* as the contract first — `assertWebStatusOffline` expects `camera=privacy, online=false` — before choosing a shape. Reproduce-first should include "read the existing pins first" even when the plan says "change the field".
2. **The Waybar collapse was a subtle regression risk** the initial fix introduced silently (an unplugged camera would have shown the last intent rather than offline). Caught by reasoning about real-world behavior, not by a failing test — the golden tests bypass the probe path. I added `TestWaybarProjectsOfflineWithoutDevice` to convert that reasoning into a pin.
3. **`nix build` was only run twice**; per-change verification leaned on `go test`/`golangci-lint` (fast) which is right, but the final `nix build` is the true gate and should bookend the session (it does now).

## e) VERIFICATION (all green)

| Gate | Result |
|------|--------|
| `GOWORK=off go test -race -count=1 ./...` | `ok` (main + `internal/pixy`) |
| `GOWORK=off golangci-lint run --timeout 2m ./...` | `0 issues` (benign `erraudit` nolint warning only) |
| `nix build` | exit 0 |
| `templ generate` | no diff (generated file in sync) |
| `GOWORK=off go vet -tags=integration ./...` | clean |

## f) WHAT'S NEXT (open, all external or hardware-gated)

1. #166 hardware session — closes #182 (sync speeds/trackmode readback), #138/#139/#140/#141/#150.
2. #129 online screenshots; #155 release cut; #156 branch protection (Lars-only).

## g) QUESTIONS

None blocking. The three open questions from the prior report (schema version, InCall placement, breaker/sync disposition) are all resolved here: schema → v2 (documented), InCall → kept + reset on start, breaker/sync → documented-intentional.
