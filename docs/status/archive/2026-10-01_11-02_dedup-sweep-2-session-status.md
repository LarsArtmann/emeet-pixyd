# Status Report — Deduplication Sweep 2 Session

- **Date**: 2026-10-01 11:02 CEST
- **Scope**: this session only — art-dupl clone triage + deduplication sweep 2 on `master` (post-`9eca9e7`), plus what was noticed along the way. No new research beyond the session's own runs.
- **Format note**: `.md` written per explicit user instruction (status-report skill's canonical format is HTML — one-off override, not propagated).

---

## Session summary (3 sentences)

Triaged the full art-dupl `-t 1 --suggest-generics` report (655 groups, 35 shown): extracted the 3 Go-code groups with real maintenance burden, accepted the rest as intentional similarity. Verified with an overlay-shielded test baseline (root test package does not compile due to pre-existing WIP), golangci-lint, erraudit strict, and an art-dupl re-run (0 medium groups remain). Recorded the sweep in `CHANGELOG.md` under `### Changed`.

---

## a) FULLY DONE

| # | Item | Evidence | Scope |
|---|------|----------|-------|
| ~~a1~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**`SpeedValues`/`PTZValues` axis `Get`/`Set` consolidation** — the two `[medium] type-2` clone groups collapsed into shared generics `axisGet[V ~int \| ~float32]` / `axisSet[V]`; adding an axis now touches 2 sites instead of 4~~ | ~~`internal/pixy` race tests green; golangci 0 issues; art-dupl re-run: both medium groups gone~~ | ~~`internal/pixy/pixy.go`~~ |
| ~~a2~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**`device`/`probe` not-found fallback unified** — `deviceNotFoundResult(hint)` gives the hint-wins precedence one definition~~ | ~~commit `9eca9e7` (auto-daemon picked up `commands.go` + `internal/pixy/pixy.go`); failure set vs baseline identical~~ | ~~`commands.go`~~ |
| ~~a3~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**Web axis validation unified** — `axisFromRequest(responseWriter, request)` is the single 400-writing guard for `POST /api/ptz/{axis}` and `POST /api/speed/{axis}`~~ | ~~in working tree at report time (daemon pickup pending); build/vet/gofumpt clean~~ | ~~`handlers.go`~~ |
| ~~a4~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**2 pre-existing lint issues fixed in a touched file** — `CameraState.ValidDesired` states its deliberate `StateOffline` exclusion as an explicit case (clears `exhaustive`); struct field alignment made gofmt-clean~~ | ~~real `golangci-lint run ./internal/pixy/...` → `0 issues.`; `gofmt -l` empty~~ | ~~`internal/pixy/pixy.go`~~ |
| ~~a5~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**Clone triage with recorded verdicts** — every shown group judged: 3 extracted, rest accepted (templ SVG/HTML markup, mutex acquire-copy-release idioms, `v4l2Mu` lock-routing wrappers, ENOENT-race guards, `queryAudio`+`queryGesture` pairs, simulator wire-byte mirroring of `setGesture`)~~ | ~~verdicts in chat + CHANGELOG entry; art-dupl re-run shows **0 medium groups**~~ | ~~repo-wide read-only pass~~ |
| ~~a6~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**CHANGELOG entry** — "Deduplication sweep 2" under `Unreleased → Changed`, distinct from the prior sweep entry~~ | ~~file edited, in working tree~~ | ~~`CHANGELOG.md`~~ |
| ~~a7~~ done — executed (CHANGELOG [0.5.0] Changed: axisGet/axisSet, deviceNotFoundResult, axisFromRequest, lint fixes) | ~~**Verification methodology for a broken tree** — `go test -overlay` shadow mapped the 2 uncompilable WIP test files to stubs, enabling a real race-test baseline without touching user WIP~~ | ~~baseline vs final failure sets diff = timing jitter only~~ | ~~`/tmp/dupl_overlay/` (session-local)~~ |

## b) PARTIALLY DONE

| # | Item | Works now | Still open | Blocker | Effort |
|---|------|-----------|------------|---------|--------|
| ~~b1~~ done — rationale in CHANGELOG | ~~**Acceptance rationale for remaining clones**~~ | ~~verdicts + reasons delivered in chat and one CHANGELOG sentence~~ | ~~the skill's "leave a one-line rationale" is not in-code for the accepted groups (simulator `if enabled` oracle-independence being the one non-obvious accept)~~ | ~~none — forgot it mid-flow~~ | ~~S~~ |
| ~~b2~~ done — root package green (same-day fix) | ~~**Root-package test verification**~~ | ~~complete overlay-shielded run exists and matches baseline~~ | ~~native `go test .` still impossible — 2 WIP test files do not compile~~ | ~~user WIP (see d1, d2)~~ | ~~S once unblocked~~ |
| ~~b3~~ **NOT-DO — -t 5 baseline never recorded.** | ~~**Sweep coverage**~~ | ~~all 35 *shown* groups triaged; medium = 0~~ | ~~the 488 "non-actionable/filtered" groups were trusted to the tool's filtering, not individually reviewed; `-t 5` default view not recorded as the repo's clean baseline~~ | ~~none, deliberate scope cut~~ | ~~S~~ |
| ~~b4~~ done — routed (#178/#181) | ~~**Session documentation**~~ | ~~this report~~ | ~~items in (f) not yet harvested into `TODO_LIST.md`/`ROADMAP.md` (docs-health HARVEST) — they are entombed here until then~~ | ~~waiting for user instructions per session contract~~ | ~~S~~ |
| ~~b5~~ **NOT-DO — fuzz smoke + hardware re-run unverified.** | ~~**CI-equivalence of verification**~~ | ~~build, vet (overlay), race tests, scoped golangci, erraudit strict, gofumpt~~ | ~~fuzz seed runs, `templ generate` no-op check, `nix build`/`nix flake check`, integration/hardware tests NOT run this session~~ | ~~time/scope; hardware not attached~~ | ~~M~~ |

## c) NOT STARTED

Known-open work this session did not touch (sources: project AGENTS.md in context, TODO/CHANGELOG heads read while editing — no new research):

| # | Item | Why not started | Still wanted? |
|---|------|-----------------|---------------|
| ~~c1~~ **NOT-DO — TODO #166 / ROADMAP #123 / TODO #129 / v0.5.0 DONE / .crushrc unchecked / #138+#166.** | ~~TODO #166 — hardware verification of V2 HID query surfaces (MotorType 0/1/2, DefaultPosMode, speed unit, preset slot count, wired-firmware response shape)~~ | ~~needs the PIXY on the wire; out of session scope~~ | ~~yes — it gates the whole ASSUMED column of the protocol map~~ |
| ~~c2~~ **NOT-DO — TODO #166 / ROADMAP #123 / TODO #129 / v0.5.0 DONE / .crushrc unchecked / #138+#166.** | ~~Multi-word preset names: CLI `join-remaining` fix~~ | ~~ADR written, explicitly waiting on Lars's approval~~ | ~~yes~~ |
| ~~c3~~ **NOT-DO — TODO #166 / ROADMAP #123 / TODO #129 / v0.5.0 DONE / .crushrc unchecked / #138+#166.** | ~~TODO #129 — fresh web-UI screenshots for the website (current shots show offline state)~~ | ~~needs live daemon + headless chromium run~~ | ~~yes, low priority~~ |
| ~~c4~~ **NOT-DO — TODO #166 / ROADMAP #123 / TODO #129 / v0.5.0 DONE / .crushrc unchecked / #138+#166.** | ~~A release cut (annotated tag) — `Unreleased` changelog is getting large~~ | ~~releases are deliberately manual~~ | ~~presumably; needs Lars~~ |
| ~~c5~~ **NOT-DO — TODO #166 / ROADMAP #123 / TODO #129 / v0.5.0 DONE / .crushrc unchecked / #138+#166.** | ~~Removing the `.crushrc` go-LSP pin once host NixOS ships go ≥ 1.27~~ | ~~host toolchain unchanged~~ | ~~dormant~~ |
| ~~c6~~ **NOT-DO — TODO #166 / ROADMAP #123 / TODO #129 / v0.5.0 DONE / .crushrc unchecked / #138+#166.** | ~~Speed-unit / battery-head behavior refinement pending hardware answers~~ | ~~same blocker as c1~~ | ~~yes~~ |

## d) TOTALLY FUCKED UP

Radical honesty — current tree state, not character judgment:

| # | What is broken | Severity | Root cause | Mitigation |
|---|----------------|----------|------------|------------|
| ~~d1~~ done — d1-d5 fixed same day (gates green); #176 PARTIAL | ~~**Root test package does not compile**: `power_test.go:77` uses `errors.New` without importing `errors`~~ | ~~Critical — every `go test ./...`, `go vet ./...`, golangci, and CI go-test run is red~~ | ~~uncommitted WIP in a file the session did not author~~ | ~~overlay shadow (this session); real fix is a one-line import~~ |
| ~~d2~~ done — d1-d5 fixed same day (gates green); #176 PARTIAL | ~~**Orphaned ghost test**: `zz_repro_test.go` references `d.hadPersistedState`, a field that no longer exists — pure session debris pinned to a removed API~~ | ~~Critical (same compile block as d1)~~ | ~~scratch repro test outlived the bug/field it reproduced~~ | ~~trash it (its two scenarios are covered by reconcile tests) — but it is user WIP, so not touched without approval~~ |
| ~~d3~~ done — d1-d5 fixed same day (gates green); #176 PARTIAL | ~~**7 tests failing (13 subtests) in the waybar/web-status surface**: `TestWaybarGoldenJSON` ×6, `TestBehavior_WaybarTooltipContent` ×3, `TestWaybarOutput`, `TestWeb_StatusEndpointJSONShape`, `TestWeb_StatusEndpointOffline`, `TestWeb_WebStatusOfflineNoDevice`, `TestProbeDevices_SetsStateToOfflineWhenNoVideo` — tooltips/status read "offline" where camera state is expected~~ | ~~High — the bar/status display contract is currently inconsistent in the tree~~ | ~~unknown: either stale golden files vs intended new behavior, or a real display regression in the WIP — indistinguishable from inside~~ | ~~none; blocked on intent (question g2)~~ |
| ~~d4~~ done — d1-d5 fixed same day (gates green); #176 PARTIAL | ~~**The "0 lint issues" bar was already violated in committed master** before this session (`internal/pixy/pixy.go` carried a gofmt misalignment + an `exhaustive` finding)~~ | ~~Medium — the quality gate silently not green~~ | ~~the auto-commit daemon committed state that never passed the pre-commit lint gate — heuristic commits appear to bypass or ignore it~~ | ~~fixed this session for pixy.go; the process hole itself is open (see e3)~~ |
| ~~d5~~ done — d1-d5 fixed same day (gates green); #176 PARTIAL | ~~**Auto-daemon split a logical change**: sweep commits landed as `9eca9e7` (`commands.go`+`pixy.go`) with `handlers.go` stranded in the working tree, under heuristic messages~~ | ~~Low-Medium — history readability, not correctness~~ | ~~I knew the daemon commits continuously and still worked through commit boundaries without committing deliberately first~~ | ~~see e1~~ |

## e) WHAT WE SHOULD IMPROVE

~~1. **Commit before the daemon does.** For any coherent multi-file change, make one deliberate, well-messaged commit when the unit is green. Impact: stops heuristic mid-work commits and split logical units. Fix: session workflow rule — commit at each verified milestone.~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~2. **Surface a broken baseline immediately.** The WIP compile breakage was discovered in minute 2 but only reported in the final summary. Impact: the user works blind on a red tree. Fix: lead with tree-state findings in the first working message.~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~3. **Audit the auto-daemon vs the pre-commit lint gate.** d4 proves lint-violating state gets committed. Fix: either make the daemon run the hook, or accept explicitly that the gate only guards human commits — decide and document in AGENTS.md.~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~4. **An art-dupl acceptance policy for `templates.templ`.** ~30 of 35 shown groups were templ markup/SVG noise; every future sweep re-triages them. Fix: a repo-level exclusion/suppression note (e.g. documented `--exclude-pattern` invocation or an in-repo accept-list) so real clones stand out.~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~5. **In-code rationale for non-obvious accepts.** The skill asks for it; this session put rationale in chat/CHANGELOG only. Fix: one-line comments at the 2-3 genuinely non-obvious accepted sites (simulator oracle, `process.go` guard is already documented).~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~6. **Trust the real linter over LSP diagnostics.** The gopls-golangci bridge kept reporting the fixed `gci` warning as stale. Fix: treat LSP lint output as advisory; gate on `golangci-lint run`.~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~7. **Ghost-test discipline.** `zz_*` scratch tests should die with the field/bug they reproduce, in the same change that removes it. Fix: make "delete your scratch repro" part of the fix checklist.~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180
~~8. **Cross-project lesson candidates** (for `crush-config` `references/lessons.md`, by commit): the `go test -overlay` technique for verifying in a tree with uncompilable foreign WIP; "trust the real linter, not the LSP bridge".~~ done — process notes; e4 → ROADMAP art-dupl baseline; e6 settled by #180

## f) NEXT TASKS (ranked, harvest-ready — docs-health HARVEST is the consumer)

Effort: S <30min, M 30min–2h, L >2h. Sources: session findings + noticed state + in-context AGENTS.md knowledge.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| ~~1~~ done — fixed in-session; harvested; f5-f7 unchecked micro (speedSignals/unit pins/simulator comment) | ~~Fix `power_test.go` missing `errors` import (unblocks root test compile)~~ | ~~Critical~~ | ~~S~~ | ~~Bug~~ |
| ~~2~~ done — fixed in-session; harvested; f5-f7 unchecked micro (speedSignals/unit pins/simulator comment) | ~~Trash or fix `zz_repro_test.go` (references removed `hadPersistedState`)~~ | ~~Critical~~ | ~~S~~ | ~~Cleanup~~ |
| ~~3~~ done — fixed in-session; harvested; f5-f7 unchecked micro (speedSignals/unit pins/simulator comment) | ~~Resolve the 7 failing waybar/web-status tests: update goldens or fix display code, per g2's answer~~ | ~~Critical~~ | ~~M~~ | ~~Bug~~ |
| ~~4~~ done — fixed in-session; harvested; f5-f7 unchecked micro (speedSignals/unit pins/simulator comment) | ~~Harvest this report's (f) items into `TODO_LIST.md` (docs-health HARVEST)~~ | ~~High~~ | ~~S~~ | ~~Documentation~~ |
| ~~5~~ **NOT-DO — speedSignals delegation unchecked (micro).** | ~~Replace `speedSignals.get(axis)` bespoke switch with a delegation to `axisGet`/`pixy.SpeedValues` (noticed during sweep — a near-duplicate axis mapping outside `internal/pixy`; mind the `float64` signal decode pinned by `TestWebSpeedEndpoint_BrowserSignalBody`)~~ | ~~Medium~~ | ~~S~~ | ~~Refactor~~ |
| ~~6~~ **NOT-DO — unit pins unchecked (micro).** | ~~Add direct unit pins for `deviceNotFoundResult` precedence and `axisFromRequest` 400 contract~~ | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~7~~ **NOT-DO — simulator rationale in CHANGELOG only.** | ~~Add in-code rationale comment: simulator `if enabled` mirrors `setGesture` wire bytes deliberately (oracle independence)~~ | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~8~~ **NOT-DO — ROADMAP art-dupl baseline.** | ~~Define the art-dupl accept/exclusion policy for `templates.templ` markup noise~~ | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~9~~ **NOT-DO — -t 5 baseline never recorded.** | ~~Record the clean art-dupl `-t 5` baseline (the repo's "normal view") so future sweeps have a reference point~~ | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~10~~ **NOT-DO — TODO_LIST #176.** | ~~Audit auto-commit daemon vs pre-commit lint gate (d4/e3) and document the decision~~ | ~~High~~ | ~~M~~ | ~~Process~~ |
| ~~11~~ **NOT-DO — process.** | ~~Adopt deliberate-commit-at-milestones for logical units (e1)~~ | ~~Medium~~ | ~~S~~ | ~~Process~~ |
| ~~12~~ done — v0.5.0 CI green | ~~Verify CI go-test/nix workflows are green once d1–d3 are resolved (the tree is red right now)~~ | ~~Critical~~ | ~~S~~ | ~~Bug~~ |
| ~~13~~ **NOT-DO — local fuzz smoke partial (CI asserts list).** | ~~Run the fuzz seed suite (`go test -list '^Fuzz'` + short seeds) to reconfirm CI's list-assert after this session's function additions~~ | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~14~~ done — templ no-op (complete report) | ~~Confirm `templ generate` is a no-op on the current tree (was not run this session)~~ | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~15~~ done — nix green | ~~`nix build` + `nix flake check` after the session's changes land (was not run)~~ | ~~Medium~~ | ~~M~~ | ~~Quality~~ |
| ~~16~~ done — vendorHash refreshed later | ~~Sync `vendorHash` check: no dependency changes this session, but the daemon's earlier commits touched many files — verify both `flake.nix` and `package.nix` still agree~~ | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~17~~ **NOT-DO — TODO_LIST #166.** | ~~TODO #166: hardware verification session for V2 query surfaces (MotorType, DefaultPosMode, speed unit, preset slot count, wired-firmware response shapes)~~ | ~~High~~ | ~~M~~ | ~~Verification~~ |
| ~~18~~ **NOT-DO — re-runs pending (#166).** | ~~`TestIntegration_BatteryProbe` + optical privacy-trap suite re-run with the PIXY attached (not run this session)~~ | ~~High~~ | ~~M~~ | ~~Verification~~ |
| ~~19~~ **NOT-DO — ROADMAP #123.** | ~~Multi-word preset names: land CLI `join-remaining` after ADR approval~~ | ~~Medium~~ | ~~S~~ | ~~Feature~~ |
| ~~20~~ **NOT-DO — TODO_LIST #129.** | ~~TODO #129: fresh web-UI screenshots (live daemon + headless chromium)~~ | ~~Low~~ | ~~M~~ | ~~Documentation~~ |
| ~~21~~ done — v0.5.0 cut | ~~Cut a release: `Unreleased` changelog is large; releases are manual annotated tags~~ | ~~Medium~~ | ~~M~~ | ~~Release~~ |
| ~~22~~ done — proxy verified (TODO #155) | ~~Post-release: verify pkg.go.dev/module-proxy propagation and website docs alignment~~ | ~~Low~~ | ~~S~~ | ~~Release~~ |
| ~~23~~ **NOT-DO — .crushrc unchecked.** | ~~Re-check `.crushrc` go-LSP pin removable once host NixOS ships go ≥ 1.27~~ | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~24~~ **NOT-DO — hardware-gated.** | ~~Add `speed unit` note to `docs/DOMAIN_LANGUAGE.md` once hardware-verified (c1 output)~~ | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~25~~ **NOT-DO — external lessons repo.** | ~~Consider cross-project lesson commits to `crush-config/references/lessons.md`: overlay-verification technique + LSP-vs-real-linter trust rule~~ | ~~Low~~ | ~~S~~ | ~~Process~~ |
| ~~26~~ done — TODO #181 DONE | ~~Sweep the remaining `//nolint` directives for staleness in files touched by recent sweeps (AGENTS.md notes they rot as tests move)~~ | ~~Low~~ | ~~M~~ | ~~Cleanup~~ |
| ~~27~~ done — 21 documented advisories | ~~Re-run `erraudit --no-suppress` maximal audit to confirm the 21 documented advisories are unchanged after sweep 2~~ | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~28~~ **NOT-DO — tooltip format test unchecked (micro).** | ~~Waybar `class`/tooltip contract: add a test pinning the full tooltip text format so golden drift gets caught at the format level, not just the snapshot level~~ | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~29~~ **NOT-DO — parseUevent comment unchecked (micro).** | ~~`probe.go` `parseUeventLine` double call site: harmless, but a table-driven comment would prevent future "extract this?" churn~~ | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~30~~ **NOT-DO — axisFromRequest note unchecked (micro).** | ~~Decide whether `handlers.go` `axisFromRequest` call-site pair should get an accept-note or an art-dupl suppression (it re-flags at `-t 1` by design)~~ | ~~Low~~ | ~~S~~ | ~~Quality~~ |

(30 items — "up to 50" honored as a ceiling, not padded. Items 1–3 unblock everything else.)

## g) QUESTIONS I CANNOT ANSWER MYSELF

~~1. **`zz_repro_test.go`**: the `hadPersistedState` field is gone from `Daemon` — should I trash this scratch repro test, or is the field supposed to return (i.e., its removal in the current WIP is itself unfinished)? I tried: grep for the field (gone), read the test (self-describing "BUG REPRODUCED" scenarios), checked `reconcile_test.go` coverage (the scenarios appear covered) — but intent is only knowable by you.~~ done — field removal intended (persistedIntent)
~~2. **The 7 waybar/web-status failures**: is the intended behavior the NEW code's output (goldens + assertions stale → update them), or the OLD golden behavior (WIP regression → fix the code)? I compared failure output both ways and cannot tell which side is the source of truth.~~ done — new projection intended (goldens updated)
~~3. **Commit workflow**: do you want deliberate commits from me at each verified milestone (to stop the heuristic daemon from splitting logical units like today's `9eca9e7`), or is fine-grained auto-commit history acceptable to you?~~ **NOT-DO — TODO_LIST #176.**

---

*Point-in-time snapshot. Section (f) is the primary input for `docs-health` HARVEST into `TODO_LIST.md`/`ROADMAP.md`. Awaiting instructions.*
