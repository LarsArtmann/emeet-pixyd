# Status Report — art-dupl Deduplication Sweep

**Date:** 2026-09-28 23:47 CEST
**Session scope:** art-dupl clone triage (8 actionable groups, `-t 3 --type-aware --suggest-generics`) → judgment (extract/accept) → extraction → verification
**Branch:** master · Working tree: session changes present (auto-commit daemon has been picking files up mid-session; final commit state to verify)

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| a1 | **All 8 flagged clone groups triaged; 6 eliminated, 2 accepted with in-code rationale.** art-dupl re-run: `8 actionable → 2 (62 non-actionable, 42 suppressed)` — the 2 remaining are the commit-failure strike (deliberately no re-probe; comment at `device.go:44-47`) and the preset lock routing (conditional vs unconditional V4L2 scoping; documented in `handlePresetWithLock`). | final `art-dupl` run, 2026-09-28 23:46 |
| a2 | **New `circuitbreaker.go`** — `hidSendGuard() (HIDDevice, bool)` + `recordHIDSendFailure(ctx)`: the threshold policy (`hidFailCount >= hidCircuitBreakerThreshold`), re-probe rule, and lock dance now have ONE definition. Wired into `device.go` (config path), `motor.go` (`sendV2Set`), `power.go` (`v2ReadLocked`). | grep: zero remaining inline `hidFailCount >=` comparisons outside the helper; probe.go's under-lock reset intentionally untouched |
| a3 | **`setTargetTrack` collapsed onto `sendV2Set`** — removed 36 lines of duplicated transport (guard → Send → accounting → reset). Error strings verified byte-identical (`"setTargetTrack (no device): …"`, `"setTargetTrack send: …"`); `motor_cmd_test.go:163` hidFailCount pin passes. | `motor.go:43-50`; race suite green |
| a4 | **commands.go helpers** — `applyAutoMode(mode)` (single write path for `state.AutoMode` from command handling), `presetLookup(name)` (read-lock snapshot), `handleMutatingCommandWithV4L2Lock(ctx, parts)` (encodes the v4l2Mu→hidMu order). | `TestLockOrder_V4L2MoveWithHIDCommands`, `TestAuto*`, `TestPreset*` green |
| a5 | **handlers.go preset factory** — `presetAction(verb)` (mirrors the existing `action(cmd)` pattern) + `patchPresetResult` shared tail. Deleted `handlePresetLoad/Delete/Push` methods (were byte-identical 14-line blocks); `handlePresetSave`/`handlePresetPull` refactored onto the tail; the MOVES-THE-CAMERA confirm() note preserved at the mux registration. | `TestWeb*`, `TestPreset*` green; mux at `handlers.go:429-433` |
| a6 | **Generic `ttlCache[T]`** replaces the parallel `ptzCache`/`powerCache` implementations (Get/Set/Invalidate). Aliases keep all call sites: `type ptzCache = ttlCache[pixy.PTZValues]`, `type powerCache = ttlCache[powerReading]`. Killed the duplicate `powerCache.Invalidate` + unused `sync` import in power.go. | build clean; `power_test.go:118` `powerCache{}` literal still compiles |
| a7 | **AGENTS.md updated** — architecture table row for `cache.go` now documents `ttlCache[T]` + aliases. (The `circuitbreaker.go` row was added by the concurrent session — accepted, accurate.) | `AGENTS.md:57`, `AGENTS.md` table |
| a8 | **Fixed pre-existing broken build at HEAD:** `internal/pixy/config_test.go` used `slices.Equal` without importing `slices` — the committed state did not compile. One-line import fix. | suite went from `FAIL [build failed]` to ok |
| a9 | **Verification stack green:** `GOWORK=off go test -race -count=1 ./...` ok (both packages, final run on merged tree); `buildflow -s golangci-lint` → **0 issues** (was 3: my circuitbreaker nonamedreturns + 2 pre-existing, all resolved); full race suite re-run after merging concurrent-session edits. | tool output in session log |
| a10 | **Concurrent-session merge handled safely:** another session edited `main.go`/`probe.go`/`circuitbreaker.go`/`AGENTS.md` while I worked. Read every diff before judging (per AGENTS.md rules), accepted all (they were the exact lint fixes + doc row I wanted), re-ran tests + lint on the merged tree. No clobbering. | git diff review; final green runs |

## b) PARTIALLY DONE

| # | Item | Works | Remaining | Effort |
|---|------|-------|-----------|--------|
| b1 | **Full BuildFlow dev gate green run** | Targeted `-s golangci-lint` passes with 0 issues | The full `buildflow --build-mode dev` run is **blocked by ruff failing on `tools/inno661/verify.py`** (20 errors, e.g. SIM115) — pre-existing, Python RE artifacts, unrelated to Go. 9 pipeline steps were skipped behind it; the Go steps beyond golangci-lint never got a full-gate verdict this session. | M |
| b2 | **Accepted-clone paper trail** | Both remaining groups have in-code rationale comments | art-dupl re-flags them on every run (no in-repo baseline/suppression mechanism configured). Decide: add an art-dupl baseline/config if supported, or accept the recurring noise. | S |
| b3 | **HARVEST of section (f) into TODO_LIST.md** | This report lists the ranked backlog | Per the status-report skill, (f) is the primary input for `docs-health` HARVEST; not yet pulled into `TODO_LIST.md`/`ROADMAP.md` — waiting for your instruction (you said WAIT). | S |
| b4 | **Preset web validation unification** | `handlePresetSave` keeps `ValidatePresetName`; load/delete/push keep the empty-name 400 check (behavior preserved exactly) | A deeper unification (always `ValidatePresetName` in `presetAction`) would tighten garbage-input handling (400 instead of passthrough to the command layer) but changes status codes for invalid names — needs your call. | S |
| b5 | **CHANGELOG entry for the dedup refactor** | Working tree contains the refactor | `CHANGELOG.md` [Unreleased] not yet updated (completed work lives there per AGENTS.md). | S |
| b6 | **Post-refactor benchmark delta** | 9 benchmarks exist and still pass in the suite | I did not run them before/after: `ttlCache[T]` and the double-RLock in `handlePresetLoad` (videoDev snapshot + presetLookup now take the lock twice instead of sharing one) are behavior-identical but perf-unmeasured. | S |

## c) NOT STARTED (relevant to this session's scope; existing backlog rows restated)

| # | Item | Why not started |
|---|------|-----------------|
| c1 | #166 hardware-verification bundle (closes #138/#139/#140/#141/#150 + #129 in one wired session) | 🚫 blocked: no PIXY on the bus this session (per TODO_LIST header) |
| c2 | #172 shared V2 query helper (4th GET family trigger) | Trigger re-checked during this session: still 3 production GET families — identity/power/preset; NOT met. I touched those paths but added no new query family |
| c3 | Multi-word preset names: CLI join-remaining fix | ADR `2026-09-18_multi-word-preset-names.md` awaits your approval |
| c4 | #116 structured command types | Recommendation ADR awaits your decision |
| c5 | Session commit | Harness rule: never commit without explicit request; the auto-commit daemon has been absorbing files (motor.go/device.go/circuitbreaker.go already committed mid-session; cache.go/commands.go/handlers.go/power.go/AGENTS.md/main.go/probe.go/config_test.go were still uncommitted at report time — VERIFY the daemon picks up the rest, incl. the config_test.go build fix) |
| c6 | DESIGN.md §circuit-breaker wording check ("resets on success or successful probe") | Still true post-refactor; wording not re-verified against the centralized helper names |

## d) TOTALLY FUCKED UP

| # | Item | Severity | Root cause | Mitigation |
|---|------|----------|-----------|------------|
| d1 | **The auto-commit daemon committed a red suite to master.** At session start, `internal/pixy/config_test.go` at HEAD did not compile (`undefined: slices`) — the last auto-commits landed without a green build. CI would catch it, but local HEAD was broken for every tool run until my fix. | HIGH (development friction; CI-blocker shape) | The daemon commits on a heuristic without running build/tests (known blind spot, cf. buildflow skill: "don't assume pma committed your work — verify") | Working-tree fix applied. Real fix: gate the daemon on a pre-commit `buildflow --build-mode pre-commit` (needs your call, see g3) |
| d2 | **Near-miss regression I introduced mid-refactor:** my first commands.go multiedit deleted the `default:` branch's current-mode snapshot in `handleAutoCommand` — bare `auto` would have returned the zero mode instead of the current one. Caught on immediate read-back of the edit result; restored in the next edit. Zero test or user exposure. | was-CRITICAL, caught pre-verify | I batched a control-flow change into a multiedit and only noticed by re-reading, not by design | Process fix adopted: after any multiedit touching control flow, re-read the whole function before building (see e2) |
| d3 | **The repo's BuildFlow dev gate is effectively red for everyone:** 20 ruff errors in `tools/inno661/verify.py` fail `ruff-check-fix` in a loop ("4 consecutive runs, same error"), gating 9 downstream steps. | MED (blocks full-gate confidence for ALL Go work) | Python RE artifacts are in the tree; no scoped ruff exclusion | Scope ruff away from `tools/inno661/`/`tools/emhid/` or clean them (needs policy call, g1) |
| d4 | **LSP signal noise / config split brain:** `.crushrc`-pinned LSP reported `golines`/`wsl_v5`/`unused` warnings all session (incl. stale `unused` on my new helpers for hours) that the repo's golangci config does not report. Two linter configs = two truths. | LOW (noise, misdirects attention) | `.crushrc` LSP toolchain/lint config ≠ repo `.golangci.yml` (documented gotcha, still biting) | Align the LSP lint config with the repo config or disable per-rule LSP linting |
| d5 | **My own lint miss:** initial `presetLookup` used named returns (`nonamedreturns` violation) — caught by golangci, fixed immediately. Also one stale-read edit failure on circuitbreaker.go (concurrent session had already applied my intended fix — the failure was the guardrail working). | LOW | Wrote code before running the lint loop | None needed beyond the existing fix-then-verify loop |

## e) WHAT WE SHOULD IMPROVE

1. **Gate the auto-commit daemon on a green pre-commit build** (d1). Impact: prevents red-at-HEAD sessions like this one; every future session currently pays a diagnosis tax. Concrete: hook `buildflow --build-mode pre-commit` (or at minimum `go build ./...`) into the daemon's commit path.
2. **Read-back rule after multiedit on control flow** (d2). Impact: cheap insurance against the exact near-miss that occurred; the edit tool's diff output is easy to skim past. Concrete: after batch edits, `view` the changed function before `go build`.
3. **Scope BuildFlow's ruff step away from RE artifact directories** (d3). Impact: turns the full dev gate green for Go work; today `-s golangci-lint` drill-downs are the only reliable Go verdict.
4. **Run the FULL gate before declaring done**, not targeted steps. This session ended green on tests + golangci-lint, but the full-gate verdict was never obtained (b1). Targeted steps can miss cross-cutting steps (gomod-check, nix checks).
5. **nolint directive audit after every refactor** — AGENTS.md already notes directives rot; this session moved `probeDevices` calls into `circuitbreaker.go` (directive preserved, verified load-bearing) and deleted three nolint'd call sites in handlers.go. A periodic `nolintlint`-driven sweep would catch the next rot batch mechanically.
6. **Benchmark before/after for cache-layer refactors** (b6) — `ttlCache[T]` changed hot-path shapes; a 1-minute benchmark run would convert "behavior-identical" into "measured identical".
7. **art-dupl accepted-clone tracking** (b2) — without a baseline, the 2 accepted groups will re-flag in every future sweep and someone will eventually "fix" them against the documented rationale.
8. **Session-start git snapshot discipline** — I started from the environment's stale git-status snapshot and only noticed the concurrent session when an edit failed on stale read. A `git status` + `git diff --stat` first thing would have surfaced the parallel actor 20 minutes earlier.

## f) Top things to get done next (ranked; harvest into TODO_LIST/ROADMAP)

Impact: Critical/High/Med/Low · Effort: S <30min, M 30min-2h, L >2h

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Gate the auto-commit daemon on a green pre-commit build (fixes d1 red-at-HEAD class) | Critical | M | Tooling |
| 2 | Verify the daemon committed the remaining session files — especially the `internal/pixy/config_test.go` `slices` import (build fix) | Critical | S | Cleanup |
| 3 | Scope BuildFlow ruff away from `tools/inno661/` + `tools/emhid/` (or ruff-clean them: SIM115 ×20) so the full dev gate goes green (d3/b1) | High | S | Tooling |
| 4 | #166 hardware bundle: run `TestIntegration_BatteryProbe` → battery/charge verdict (#139) | High | S | Feature (hardware) |
| 5 | #166: pin `MotorType` + `DefaultPosMode` + 0x63 iface echo → #150 | High | M | Feature (hardware) |
| 6 | #166: pin speed unit + hardware limit → clamp at command layer + web slider max (#138) | Med | S | Feature (hardware) |
| 7 | #166: preset slot-count sweep + response-shape pin (#141; your Q1 call if mode-only) | Med | S | Feature (hardware) |
| 8 | #166: live `preset push` → `preset pull` round-trip check | Med | S | Feature (hardware) |
| 9 | #166: retake ONLINE web UI screenshots + panel crop + video poster (#129) | Med | S | Documentation |
| 10 | #166 optional: Windows usbmon capture of official app to settle `MotorType` (your Q3) | Low | M | Research |
| 11 | Write CHANGELOG.md [Unreleased] entry for the deduplication refactor | Med | S | Documentation |
| 12 | Benchmark run: confirm `ttlCache[T]` + preset-load double-RLock are perf-neutral (b6) | Low | S | Quality |
| 13 | Decide: tighten `presetAction` to always `ValidatePresetName` (400 vs today's passthrough for garbage names) (b4) | Low | S | Feature |
| 14 | Add art-dupl accepted-clone baseline (if the tool supports one) so the 2 documented-accepted groups stop re-flagging (b2) | Low | S | Quality |
| 15 | Align `.crushrc` LSP lint config with repo `.golangci.yml` (kill golines/wsl_v5 noise, d4) | Low | S | Tooling |
| 16 | nolint sweep: mechanically find stale `//nolint` directives post-refactor (nolintlint or grep-driven) | Med | M | Quality |
| 17 | #172: after #138 lands (4th GET family = speed readback), extract the shared V2 query helper | Low | M | Refactor |
| 18 | Multi-word preset names: implement CLI join-remaining after you approve the 2026-09-18 ADR | Med | S | Feature |
| 19 | #116 structured command types: decide on the recommendation ADR | Med | S | Decision |
| 20 | Cut v0.4.1 vs v0.5 (your cadence call, #155) | Med | S | Release |
| 21 | Close issue #6 once PIXY 2K confirmed on real hardware (#154) | High | S | Feature (hardware) |
| 22 | Branch protection: require go-test/nix/website workflows on master (#156) | Med | S | Process |
| 23 | Send the staged innoextract 6.6.1 upstream PR (#148; run verify-before-filing gate first) | Med | S | Community |
| 24 | DESIGN.md: re-verify circuit-breaker paragraph wording against the centralized `circuitbreaker.go` helpers | Low | S | Documentation |
| 25 | docs/DOMAIN_LANGUAGE.md: check whether "circuit breaker" / "TTL cache" deserve entries after this refactor | Low | S | Documentation |
| 26 | Consider a lock-policy table (command → which lock) replacing the three ad-hoc lock sites in `handleCommand`/`handlePresetWithLock`/`handleMutatingCommandWithV4L2Lock` | Low | M | Refactor |
| 27 | Document the web↔command preset validation boundary (where names are validated and why it's two-layer) | Low | S | Documentation |
| 28 | Speed-query duality: once #166 pins the real GET form, delete the other evidenced simulator form (`TestPixySimulatorV2_SpeedQueryDuality`) | Med | S | Cleanup (hardware-gated) |
| 29 | `git status` + `git diff --stat` as session step zero (e8) — make it a habit/skill note | Low | S | Process |
| 30 | Re-check `applyAutoMode` doc claim ("single write path from command handling") if any new AutoMode writer lands outside commands.go | Low | S | Documentation |
| 31 | Consolidate TODO_LIST items #138-#141 statuses after the #166 session (rows are all 🔶 PARTIAL with the same blocker) | Low | S | Documentation |
| 32 | HARVEST this report's §f into TODO_LIST.md/ROADMAP.md (docs-health HARVEST mode) — scheduled, see b3 | Med | S | Documentation |
| 33 | Confirm no coverage regression: diff `go test -list '.*'` output or run coverage once over the refactored files | Low | S | Quality |
| 34 | `templates_templ.go:934` QF1003 (tagged switch) — fix in `templates.templ` source or confirm generated-file policy leaves it | Low | S | Quality |
| 35 | probe.go LSP-only warnings (wsl_v5/golines at lines 44/56/69/120) — clean or explicitly exclude so LSP and CLI agree | Low | S | Quality |

*(34 items — stopped where specificity stops; items 4-11, 17-24 are the existing TODO_LIST backlog restated with post-session context, not new scope.)*

## g) Top questions I can NOT figure out myself

1. **Ruff policy for RE artifacts:** Should `tools/inno661/` and `tools/emhid/` Python be (a) excluded from BuildFlow's ruff step, or (b) brought to ruff-clean? I tried reading BuildFlow config for an existing scoped exclusion and found none; this gates whether ANY full dev-gate run can go green (d3/b1). It's your quality-bar call.
2. **Concurrent session:** Another agent session edited `main.go`, `probe.go`, `circuitbreaker.go`, and `AGENTS.md` during this one (lint fixes + doc row — all accepted, all verified). Is that session still active, and should parallel sessions on this repo be coordinated (e.g. one at a time, or per-file ownership)? I cannot see other sessions; today it merged cleanly, but the AGENTS.md table is now edited by two actors.
3. **Auto-commit daemon gating:** The daemon committed a suite that didn't compile (d1). Do you want it gated on `buildflow --build-mode pre-commit` (or `go build ./...`), or is red-at-HEAD acceptable because CI catches it? It's your infrastructure; the buildflow skill documents pma's blind spot but not your tolerance for it.

---

*Point-in-time snapshot. Section (f) is the HARVEST input for TODO_LIST.md/ROADMAP.md (item 32). Format override note: skill default is a styled HTML dashboard; user explicitly requested `.md` for this report — honored, not propagated as a new default.*
