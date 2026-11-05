# Status Report — Preset-Pull Hardening + TODO Sweep (#167–#172)

**Date:** 2026-09-19 16:36 CEST
**Session scope:** Execute the six actionable OPEN rows from TODO_LIST (#167, #168, #169, #170, #171, #172) end-to-end: research → implement → test → lint → build → docs. Hardware-gated rows deliberately untouched (no PIXY on the USB bus this session — verified via sysfs sweep, no `328f:*` device present).

**Verification at close:** `go build` ✅ · `go vet` ✅ · `go test -race -count=1 ./...` both modules ✅ · golangci-lint **0 issues** ✅ · `nix build` ✅ · `nix flake check --no-build` ✅ · `dprint check` clean ✅ · templ v0.3.1020 (devShell) == go.mod pin ✅

---

## a) FULLY DONE

| Item | What shipped | Evidence |
| ---- | ------------ | -------- |
| ~~**#167 Pull early-abort**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~Sweep stops after `presetPullMaxConsecutiveFailures = 3` consecutive slot failures (any error kind; one success resets the counter). Summary carries `"aborted after 3 consecutive failures"`; all-attempted-failed error reports attempted vs total (`"3/8 slots unreadable, aborted after 3 consecutive failures"`). Worst case drops from ~4 s to ~0.75 s on a dead device. Bookkeeping extracted into `presetPullSweep` (`recordFailure`, `admitReading`, `allSlotsFailed`) to satisfy gocognit.~~ | ~~`commands.go`; `TestPresetPull_AllSlotsUnreadable` (rewritten), `TestPresetPull_EarlyAbortStopsSweep`, `TestPresetPull_SuccessResetsAbortCounter`~~ |
| ~~**#171 `preset pull --dry-run`**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~`flagDryRun` named constant; parsing accepts exactly `preset pull` or `preset pull --dry-run` (anything else → "takes no name"). Dry run = identical queries + accounting, zero store/persist/broadcast; summary switches to `"preset pull (dry run): would pull: …"` / `"no slot positions"`. Help text, README, and website CLI reference updated. Web endpoint deliberately unchanged.~~ | ~~`commands.go`, `main.go`, `README.md`, `website/src/content/docs/guides/cli-reference.mdx`; `TestPresetPull_DryRunDoesNotMutate`, `TestPresetPull_DryRunModeOnly`, `TestPresetPull_DryRunRejectsExtraName`~~ |
| ~~**#169 Collision property test**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~`TestProperty_PresetPull_NeverEvictsOrMutates`: 200 fixed-seed randomized scenarios (random user presets incl. hw-N collisions + out-of-range `hw-9`, random slot occupancy, random limit pressure) asserting: no eviction/mutation, count ≤ `pixy.MaxPresets`, new names only `hw-<slot>` with seeded values. Helpers: `randomPullScenario`, `randomPTZValues`, `assertPullAdditivity`. Matches the repo's stdlib-only property-test idiom (`math/rand` precedent in `internal/pixy/property_test.go`).~~ | ~~`preset_pull_test.go`~~ |
| ~~**#168 Simulator knobs + speed-query duality**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~`simulatorOption` type + `withPresetFullResponses()` builder option (all 5 construction-time field pokes converted); `SetPresetFullResponses(on)` for the one legitimate mid-test flip; `buildV2GetHeads` now also registers the Beta.25 speed-query form (`09 63 01 03` + motor byte) and `buildV2Response` answers it with the same `[motorType][speed][limit]` payload as the 2.0.3 `09 03 01 13` head — so the #166 session can compare either against wired firmware.~~ | ~~`pixy_simulator_test.go`; `TestPixySimulatorV2_SpeedQueryDuality` (both forms parse identically)~~ |
| ~~**#170 dprint ownership decided + shipped**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~`pkgs.dprint 0.57.4` vendored into the devShell (flake builds, `nix flake check` green). `dprint.json` trimmed to make the tool actually safe to run: markdown plugin DROPPED (its table re-padding would churn the deliberately compact TODO_LIST/report tables — discovered by running `dprint check` before deciding), excludes added for `website/pnpm-lock.yaml` (pnpm + frozen-lockfile guard own it) and `.config/**` (machine metadata). `dprint check` passes with zero findings — `fmt` is provably a no-op today.~~ | ~~`flake.nix`, `dprint.json`, CHANGELOG [Unreleased] Changed entry~~ |
| ~~**#172 trigger evaluation**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~Counted production V2 GET families: identity (sn/ver/devver/func), power (battery/charge), preset slots = **3**. The 4th (speed/position readback) is gated on #138/#166 → trigger NOT met; row kept OPEN with the evaluation recorded in TODO_LIST. No speculative refactor (respecting the row's own trigger rule).~~ | ~~`rg v2Read\|v2ReadLocked` sweep; TODO_LIST row #172~~ |
| ~~**Docs propagation**~~ done — executed (CHANGELOG [0.5.0]; commands.go:670; tests; TODO #172 evaluation recorded) | ~~Counted production V2 GET families: identity (sn/ver/devver/func), power (battery/charge), preset slots = **3**. The 4th (speed/position readback) is gated on #138/#166 → trigger NOT met; row kept OPEN with the evaluation recorded in TODO_LIST. No speculative refactor (respecting the row's own trigger rule).~~ | ~~`rg v2Read\|v2ReadLocked` sweep; TODO_LIST row #172~~ | ~~ ~~ | ~~CHANGELOG [Unreleased]: 4 new Added bullets + 1 Changed bullet + the Motor-preset-pull bullet rewritten to the new failure semantics (its "8/8 slots unreadable" claim had become false). TODO_LIST header + rows (5 shipped rows removed, #172 annotated). AGENTS.md: commands block (nix fmt treefmt note, dprint line), Presets behavior bullet (early-abort + dry-run), simulator lines (options, duality, "never poke the field").~~ | ~~`CHANGELOG.md`, `TODO_LIST.md`, `AGENTS.md`~~ |

Side effect kept intentionally: `nix fmt` (treefmt, templ enabled) regenerated `templates_templ.go` to the pinned templ v0.3.1020 output — aligns the committed generated file with what CI generates; auto-daemon committed it.

## b) PARTIALLY DONE

~~1. **#172 as a whole** — the row stays OPEN by design; only the evaluation step ran. The actual helper extraction awaits the 4th GET family.~~ **NOT-DO — TODO_LIST #172 (open, trigger NOT met).**
~~2. **Website doc freshness for pull** — `cli-reference.mdx` gained the `--dry-run` row, but `website/src/content/docs/guides/presets.mdx:54-57` still shows bare `preset pull` with no dry-run/early-abort mention. Caught during THIS report's self-review, not during the session.~~ done — FIXED 2026-11-05: presets.mdx now carries --dry-run + early-abort
~~3. **`FEATURES.md` inventory** — row 148 "Motor-Preset Pull" still describes pre-abort semantics and neither it nor any row knows about dry-run. Not touched at all this session (see e1).~~ done — FIXED 2026-11-05: FEATURES Motor-Preset Pull row carries early-abort + dry-run
~~4. **dprint adoption depth** — vendored + safe config, but: not wired into CI, not mentioned in CONTRIBUTING, and `dprint fmt` was verified only via `check` (clean check ⇒ fmt no-op, but never actually exercised the write path end-to-end).~~ **NOT-DO — dprint stays devShell-only; CI gate never added (TODO #170 decision).**
~~5. **Decision durability for the two judgment calls** — abort threshold N=3 and "no web dry-run surface" are implemented and documented in CHANGELOG prose, but neither is recorded as an explicit decision (ADR/ROADMAP won't-do style) that a future session will recognize as *decided* rather than *incidental*.~~ **NOT-DO — no ADR/won’t-do entry beyond CHANGELOG prose (micro).**

## c) NOT STARTED (in-scope-adjacent, consciously or not)

~~1. **`FEATURES.md` / `DOMAIN_LANGUAGE.md` updates** — not started (should have been: FEATURES row 148 + possible "Early Abort"/"Dry Run" domain terms).~~ done — FIXED 2026-11-05: DOMAIN_LANGUAGE Early Abort + Dry Run terms added
~~2. **`presets.mdx` guide update** — not started.~~ done — FIXED 2026-11-05: presets.mdx updated
~~3. **vmTest run** — harvest item 43 ("run once before next release") not exercised; my flake change was devShell-only so `nix build` + `flake check` coverage was judged sufficient. Defensible, but it remains on the release checklist.~~ done — vmTest green (CHANGELOG [0.5.0])
~~4. **Hardware bundle (#166 → #138/#139/#140/#141/#150) and #129/#148/#154/#155/#156** — all BLOCKED on external inputs (hardware presence, Lars's calls, zutto's confirmation, GitHub settings). Untouched by design; integration test not even attempted since no PIXY is on the bus.~~ **NOT-DO — TODO #166/#129/#148/#154/#156 blocked; #155 superseded (v0.5.0).**
~~5. **Website redeploy** — the site changelog/CLI reference edits are committed to the repo but not deployed (website.yml will do it on push to master; the auto-daemon commits are already on master, so this self-resolves — but it was not consciously verified).~~ done — redeployed + live-verified (CHANGELOG [0.5.0])

## d) TOTALLY FUCKED UP

**Nothing.** No broken state was left behind; every regression introduced during the session was caught by the same session's tests/lint before close. Near-misses worth logging (all self-caught, none escaped):

~~1. **Three consecutive test bugs in the new tests** (caught by the test run, fixed in one iteration): `flakyQuerySim` didn't record failed queries (asserted on the wrong counter), `TestPresetPull_SuccessResetsAbortCounter` forgot `withPresetFullResponses()` (mode-only answers can never "pull"), and the dry-run report assertion counted seeding traffic as HID reports.~~ done — near-misses fixed in-session (tests green)
~~2. **Stray duplicated line** in the speed-duality test edit (accidental `newPixySimulator().Send(...)` block) — removed immediately, never compiled in.~~ done — near-misses fixed in-session (tests green)
~~3. **Three lint rounds** instead of one (gocognit ×2 → refactor; golines ×2; nlreturn; exhaustruct on the new struct literal — foreseeable, since AGENTS.md documents exhaustruct_v5 with full-string ignore patterns; varnamelen on `o`; one missed `o.` rename broke the build once).~~ done — near-misses fixed in-session (tests green)

## e) WHAT WE SHOULD IMPROVE

**Process (what I forgot / would do differently):**

~~1. **I optimized for the Go surface and under-served the docs surface.** The session's own theme was "docs health" (these rows came from a harvest), yet FEATURES.md, presets.mdx, and DOMAIN_LANGUAGE.md were left stale — the exact drift class this project keeps repairing. A "docs touch-list" (FEATURES / relevant guides / DOMAIN_LANGUAGE / README / website) should be consulted *before* closing any feature task, not discovered by the next status report.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~2. **Website changes were untested.** I edited MDX but never ran the astro build. The CI guard would catch breakage only after push. Same "test after changes" rigor must extend to non-Go surfaces I touch.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~3. **Deviations from the TODO text were made silently.** #167 says "N consecutive slot **timeouts**"; I implemented N consecutive **failures** (any kind). The reasoning (parse errors also indicate a broken device; timeouts are the only slow failure mode the simulator can't isolate) is sound but should have been written into the CHANGELOG/TODO row as an explicit, vetoable decision.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~4. **Judgment calls weren't surfaced as calls.** N=3 and "CLI-only dry-run" were my picks; they're documented as facts, not as decisions open to override.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~5. **Lint could have been a first-pass pass.** exhaustruct (documented in AGENTS.md as config-level behavior) and long signatures were predictable; anticipating them saves round trips.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~6. **The stale gopls "writestring" warning** was assumed stale after the refactor and never explicitly re-verified against a fresh read (the final golangci-lint pass at 0 issues is the practical cover, but the AGENTS.md "independently verify tool output" rule argues for the explicit check).~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)

**Design (what could still improve in the shipped code):**

~~7. **`presetPullOutcome.dryRun` mixes input with result.** The outcome struct carries a config flag; cleaner is `dryRun` living on `presetPullSweep` (config) with the outcome purely a result. Small, worth doing when #172's helper lands.~~ **NOT-DO — dryRun still on outcome struct (micro refactor).**
~~8. **Early-abort counts failures, not timeouts** — semantic bluntness; if hardware shows parse errors are common on live slots (mode-only garbage), the abort could truncate legitimate sweeps. The #166 session should re-check the threshold against real failure distributions.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~9. **The summary omits "attempted"** when the sweep aborts mid-way with partial results — the abort note implies it, but an explicit `"3/8 attempted"` would be unambiguous.~~ done — reflections; e7 dryRun placement + e10 dprint CI remain (micro, below)
~~10. **dprint has no CI gate** — vendored tooling nobody runs automatically can rot again; either add a `dprint check` CI step (cheap) or accept devShell-only status explicitly.~~ **NOT-DO — TODO_LIST #199 optional (dprint CI).**

## f) UP TO 50 THINGS TO GET DONE NEXT

**Immediate follow-ups from THIS session (S effort, high value-per-effort):**
~~1. Update `FEATURES.md` row 148 (Motor-Preset Pull) — early-abort + dry-run semantics.~~ done — FIXED 2026-11-05 (FEATURES row)
~~2. Update `website/src/content/docs/guides/presets.mdx` — add `--dry-run` snippet + early-abort note.~~ done — FIXED 2026-11-05 (presets.mdx)
~~3. Add "Early Abort" / "Dry Run" (and consider "Speed-Query Duality") to `docs/DOMAIN_LANGUAGE.md`.~~ done — FIXED 2026-11-05 (DOMAIN_LANGUAGE)
~~4. Run the website build locally (`pnpm run build`) once to verify the MDX table row edits.~~ done — superseded: CI builds site (TODO #185)
~~5. Record the two judgment calls durably: N=3 threshold + CLI-only dry-run (ROADMAP won't-do/decision note or ADR-lite).~~ **NOT-DO — no decision record beyond CHANGELOG (micro).**
~~6. Verify the stale gopls writestring warning is really gone (fresh LSP pass on commands.go).~~ **NOT-DO — unverifiable (LSP state not persisted).**
~~7. Confirm website.yml deploy went green after the auto-daemon's master commits (site changelog now carries dry-run).~~ done — later green runs (TODO #185)
~~8. Optionally: `TestPresetPull_DryRunNoDevice` (explicit dry-run unreachable-device error test).~~ **NOT-DO — TestPresetPull_DryRunNoDevice never added (micro).**
~~9. Optionally: dry-run/real-pull parity property (same seed ⇒ same would-pull set) — cheap extension of #169.~~ **NOT-DO — parity property never added (micro).**
~~10. Move `dryRun` from `presetPullOutcome` into `presetPullSweep` when next touching the file (fold into #172's extraction).~~ **NOT-DO — dryRun placement unchanged (micro).**

**Hardware session (the #166 bundle — one wired session closes five rows):**
~~11. Run `TestIntegration_BatteryProbe` → #139 verdict.~~ **NOT-DO — TODO #166 checklist hardware items.**
~~12. Pin `MotorType` + `DefaultPosMode` enum values → #150.~~ **NOT-DO — TODO #166 checklist hardware items.**
~~13. Pin speed unit + real hardware limit → #138 (then clamp at command layer + set web slider max).~~ **NOT-DO — TODO #166 checklist hardware items.**
~~14. Preset slot-count sweep + response-shape pin (mode-only GET vs full SET-echo) → #141 (decide Lars's Q1: SET-echo acquisition if mode-only).~~ **NOT-DO — TODO #166 checklist hardware items.**
~~15. Live round-trip: `preset push` → `preset pull` → values match.~~ **NOT-DO — TODO #166 checklist hardware items.**
~~16. Use the new speed-query duality: probe BOTH `09 63 01 03`+motor and `09 03 01 13` forms on wired firmware.~~ **NOT-DO — TODO #166 checklist hardware items.**
~~17. Re-check the early-abort threshold against real per-slot failure/timeout distributions (post-#166 tuning of N).~~ **NOT-DO — TODO #166 checklist hardware items.**
~~18. Decide reconcile behavior for tracking variants after power cycles → #140.~~ **NOT-DO — TODO #166 checklist hardware items.**
~~19. Retake online web-UI screenshots (live MJPEG, tracking active) + panel crop + video poster → #129.~~ **NOT-DO — TODO #166 checklist hardware items.**
~~20. Optional (Lars's Q3): Windows usbmon capture of the official app to settle `MotorType` (harvest item 39).~~ **NOT-DO — TODO #166 checklist hardware items.**

**Release track:**
~~21. Run the full vmTest suite once (`.#checks…vmTest`) before the release (harvest item 43).~~ done — vmTest green (CHANGELOG [0.5.0])
~~22. Cut v0.4.1 vs v0.5 — Lars's cadence call → #155.~~ done — v0.5.0 cut (TODO #155 DONE)
~~23. Post-release: verify pkg.go.dev / GitHub release artifacts (if library surface applies) — per release checklist.~~ **NOT-DO — pkg.go.dev check never recorded (GitHub release verified in TODO #155).**
~~24. Retake demo-video screenshots if #129 lands before release (poster frame freshness).~~ **Superseded — demo source reconstructed + poster webp ([0.5.0]); #129 online pending.**

**Repo/CI hardening:**
~~25. Branch protection: require go-test/nix/website workflows on master → #156 (GitHub settings, Lars only).~~ **NOT-DO — TODO_LIST #156.**
~~26. Decide: add `dprint check` to CI (or explicitly record devShell-only as the decision).~~ **NOT-DO — dprint CI decision (TODO_LIST #199 optional).**
~~27. Close issue #6 once @zutto confirms PIXY 2K on real hardware → #154.~~ **NOT-DO — TODO_LIST #154.**
~~28. Send the staged innoextract 6.6.1 upstream PR → #148 (Lars's call; `tools/inno661/UPSTREAM.md` ready).~~ **NOT-DO — TODO_LIST #148.**
~~29. Sweep for stale `//nolint` directives after this session's refactors (pullSummary removal may have orphaned none, but the class rots).~~ done — TODO #181 DONE (both chunks)
~~30. Consider extracting `randomPullScenario`-style helpers into `test_helpers_test.go` if more property tests appear (avoid per-file helper sprawl).~~ **NOT-DO — helpers still inline (micro).**

**Feature surface (from TODO/ROADMAP context, post-hardware):**
~~31. #172: extract the shared V2 query helper when the speed/position readback (4th GET family) lands.~~ **NOT-DO — TODO_LIST #172.**
~~32. Speed readback UI once the unit/limit is pinned (#138 aftermath): slider max from hardware, not sanity bound.~~ **NOT-DO — TODO #138/#166.**
~~33. Decide web surface for dry-run (preview toggle before a real pull) — currently CLI-only by my call.~~ **NOT-DO — web dry-run surface undecided (CLI-only shipped).**
~~34. Multi-word preset names: `join-remaining` CLI dispatch after Lars's approval (ADR `2026-09-18_multi-word-preset-names.md`).~~ **NOT-DO — ADR awaiting Lars; TODO #179 REMAINS note.**
~~35. Battery hybrid polling revisit-trigger: implement only after #139's hardware verdict (ADR `2026-09-19_battery-polling-on-demand-ttl`).~~ **NOT-DO — ADR exists; gated on TODO #139 verdict.**
~~36. Route Waybar slot-occupancy surface to ROADMAP won't-do (harvest item 40) — needs the decision written down.~~ **NOT-DO — ROADMAP won’t-do candidate (undecided).**
~~37. Auto-management docs/ADR: preset pull stays manual, no auto-sync on device appear (harvest item 38 — partially recorded in AGENTS.md, ADR optional).~~ done — AGENTS documents pull-stays-manual
~~38. README: mention early-abort/dry-run in the feature bullet list if the preset section grows (currently only CLI block covers it).~~ done — README:199 dry-run; early-abort prose still absent (micro)

**Docs hygiene (continuous):**
~~39. Annotate the 2026-09-19 harvest report items 32/34/35/37/40/41/43 as DONE where this session closed them (docs-health ANNOTATE mode).~~ done — this sweep annotates it (2026-11-05)
~~40. Verify TODO_LIST internal references still resolve after the row removals (harvest self-review lesson: unverified anchors).~~ done — TODO footer reference resolves
~~41. Update the website changelog page if it mirrors CHANGELOG [Unreleased] sections (deploy pipeline greps for newest section).~~ done — site changelog carries v0.5.0 (sentinel guards)
~~42. Consider `dprint fmt` pre-commit opt-in for staged yaml/json (mirrors the golangci-lint gate pattern) — only if 26 lands.~~ **NOT-DO — dprint pre-commit opt-in never added (micro).**

**Bigger bets (ROADMAP-class, not this quarter):**
~~43. Motor-speed readback + persistence verification loop (needs #138 hardware pins).~~ **NOT-DO — TODO #138/#166.**
~~44. Position readback (`GetMotorPos`) for "where is the camera" surfaces (4th GET family — pairs with #172).~~ **NOT-DO — TODO #166 (4th GET family).**
~~45. Slot-occupancy in the web UI once slot count is hardware-pinned (#141 aftermath).~~ **NOT-DO — TODO #141 + ROADMAP:57.**
~~46. usbmon-based decoder tooling for the remaining 53 Mac-only heads (harvest item 39, large).~~ **NOT-DO — ROADMAP usbmon.**
~~47. Website: presets guide screencast/GIF showing pull + dry-run (marketing polish).~~ **NOT-DO — presets screencast never made.**
~~48. Fuzz the new `pullSummary`/`allSlotsFailed` message assembly if it grows parsing-adjacent complexity (currently trivial — note-only).~~ **NOT-DO — no fuzz target for summary assembly (micro).**
~~49. Revisit `PresetMap` typing: hw-<slot> names are machine-mirrored — a typed variant (software vs mirrored) would make the additive guarantee compiler-checked (data-model upgrade, breaking state schema → needs migration design).~~ **NOT-DO — PresetMap typing revisit never routed (micro).**
~~50. After #166: collapse the ASSUMED comments (`MotorType`, slot cap, speed unit) into pinned constants and delete the evidence-grade hedges they retire.~~ **NOT-DO — TODO #166 aftermath.**

## g) QUESTIONS FOR LARS (cannot figure out myself)

~~1. **Early-abort threshold:** I picked N=3 (circuit-breaker parity). Keep 3, or do you want a higher N (e.g. 5) to tolerate flaky USB hubs before truncating a sweep?~~ done — 3 shipped (commands.go:670; AGENTS documents it)
~~2. **Dry-run web surface:** I scoped `--dry-run` to CLI/socket only. Should the web panel ever get a preview step before the real Pull button, or is CLI-only the permanent answer?~~ **NOT-DO — CLI-only shipped; web undecided.**
~~3. **dprint markdown:** I dropped the markdown plugin (its table re-padding fights the compact TODO tables). Accept markdown as permanently ungated, or do you want it back later (accepting the churn, or with per-file excludes)?~~ done — answered: markdown deliberately ungated (CHANGELOG #170)
