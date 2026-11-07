# Status Report — Pareto Type-Safety Plan Execution (Full Sweep)

**Written:** 2026-11-07 10:55 CET
**Scope:** execution of `docs/planning/2026-11-07_07-16_pareto-type-safety-and-layout-plan.md` (T1–T7) in this session, plus honest self-review of how it went. No research beyond this session's run.
**Verdict:** plan COMPLETE and pushed. Seven commits, one per task, on `master` (`362c32f..7c5a90f`), tree clean, origin in sync, all gates green on final HEAD. Three self-inflicted corruptions happened mid-flight and were caught; one of them (D1) is the most important lesson of the session.

---

## a) FULLY DONE

| Task | Commit | What landed |
|---|---|---|
| T3 DesiredCameraMode | `6a72b2a` | `pixy.DesiredCameraMode` type + `Desired{Idle,Tracking,Privacy}` + total `.CameraState()` projection; `State.Camera` swapped (tag `camera`, schema stays 2); write-side saveState normalization DELETED (unrepresentable); 56-file compiler-driven sweep of daemon + tests; newTestDaemon family takes the intent type; new `TestDesiredCameraMode` pin; offline waybar goldens re-expressed as the real collapse branch (`Online=false`). |
| T1 DisplayCameraState | `8971f90` | `pixy.DisplayCameraState(online, DesiredCameraMode) CameraState` — single source of the human-surface offline collapse; `statusSnapshot.Camera` now carries the intent type (snapshot stops pre-projecting); machine surfaces project with `.CameraState()`; table-driven pin test. |
| T2 USBProductID | `3f2d6b5` | `pixy.USBProductID` int64-backed type: 8 known PIDs, both registries, `ResolveProductID`/`ModelFromProductID`/`ExtraModelName`/`UnsupportedDeviceHint`, uevent `PRODUCT=`/`HID_ID=` parser output, `Config.ExtraProductIDs`, `probeDevices` dep. `ParseUSBProductID` + `ErrInvalidProductID` (Rejection family) with `ParseExtraProductIDs` refactored onto it, policy unchanged (skip-and-warn). |
| T4 TrackModePref + DegradedReason | `157a67b` | `State.TrackMode` → `pixy.TrackModePref` (`TrackModePrefFromTarget` single conversion, `.Effective()` face-default resolution, `.Valid()` load contract); `State.EffectiveTrackMode` deleted. `DegradedReason` types the reasons map + `markDegraded`/`clearDegraded`; `ExternalDepReason(binary)` the sole `extdep:` constructor; sorted read still returns `[]string` (wire byte-stable). |
| T5 PresetMap.Set | `77aaac2` | `PresetMap.Set(name, values) error` — the single mutator (ValidatePresetName + capacity via new `ErrPresetMapFull`, overwrites always allowed); both write sites routed (CLI/web save; pull sweep `admitReading`, Set failure books as slot failure); `TestPresetMapSet` pin (store, overwrite-at-cap, name rejection, cap rejection). |
| T6 battery fields | `368c62d` | `/api/status` gains `batteryPercent,omitzero` + `batteryCharging,omitzero` from the SAME TTL-cached `powerStatus` read (no second HID query); legacy `battery` string untouched; `TestStatusContract_BatteryFieldsAdditive` pins both halves (legacy byte-identical, new keys present+typed only with a reading); quickshell.mdx rows. Website proof battery green (frozen-lockfile, audit 0, tsc, 22-page build, html-validate). |
| T7 docs sync | `7c5a90f` | AGENTS.md connectivity-vs-intent bullet rewritten for the type-enforced model (+ pointers to review/plan/ADR); FEATURES State Persistence + Status API rows; TODO_LIST header records the sweep (I1–I7 landed, I8 rejected); ROADMAP gains three trigger-conditioned deferred rows (view unification: 5th surface; god-struct decomposition: named second consumer; cosmetic cmd/ move: Lars's call). |

**Also done en route:**
- The 0.3.1020 templ drift recurred at session start (a concurrent release session had regenerated `templates_templ.go`); fixed with the go.mod-pinned CLI, result byte-identical to HEAD before any T-task started.
- Every task ran the full gate battery before its commit: build, `go test -race -count=1 ./...` (both packages), erraudit strict (0 violations every run), golangci-lint (0 issues every final run), and `nix build .#emeet-pixyd` at the end (passed — committed `_templ.go` compiles under the pinned toolchain).
- Pre-commit hook (golangci-lint on staged files) fired on every final commit: 0 issues.
- Pushed once at the end: `362c32f..7c5a90f master -> master`.

## b) PARTIALLY DONE

1. **Clean-cache golangci-lint verdicts.** The documented gotcha says a suppression verdict is trustworthy only on a clean cache (`rm -rf ~/.cache/golangci-lint`). My `rm` was blocked by a concurrent session holding the cache (only the first attempt partially succeeded); every subsequent run reused the warm cache. The 0-issue verdicts are still very likely correct (pre-commit hook re-lints staged files independently), but strictly speaking most runs are warm-cache verdicts, not clean-cache ones.
2. **Planning file not stamped executed.** `docs/planning/2026-11-07_07-16_pareto-type-safety-and-layout-plan.md` still reads as "awaiting execution". TODO_LIST's header records the sweep, but the plan file itself never got an execution note/stamp. Two-line fix, not done.
3. **CHANGELOG entry for the sweep.** Deferred BY DOCTRINE (completed work prunes to CHANGELOG at the next release, per TODO_LIST header note), so the release cut will need to write it. Not "missed" — but the work is not yet reflected there, so it is partial until the next release.
4. **CHANGELOG-driven bench/battery on the perf suite.** I did not run benchmarks (`docs/benchmarks/`) for the type swaps. The changes are compile-time typing + one helper call per snapshot, so perf impact is expected nil, and the shared-machine noise gotcha makes a single run meaningless anyway — but formally, no bench evidence was gathered this session.
5. **Erraudit tool invocation discovery.** The first invocation guessed a wrong module path (`go-error-family/cmd/erraudit@latest` does not exist); the system-installed `erraudit` binary was on PATH all along. Gates ran green after, but the first attempt was wasted.

## c) NOT STARTED

Nothing from the plan itself — all 31 micro-tasks are covered by the 7 commits. Explicitly out of plan scope and not started:

1. Release cut (`v0.6.1`?) carrying the sweep's CHANGELOG entries — releases are deliberately manual.
2. All externally blocked items (#129 online screenshots, #138–#141/#150/#166 hardware session, #154/#156 GitHub settings, #200/#204 Actions suspension) — untouched, still blocked.
3. ROADMAP deferred rows' triggers: view-struct unification (needs a 5th display surface), god-struct decomposition (needs a named second consumer), cosmetic `cmd/` move (needs Lars's call).
4. The `package-split` endgame from the layout ADR — same trigger, nothing started (correctly).

## d) TOTALLY FUCKED UP (caught, but each is a real failure mode)

1. **Worst: the `DefaultState()` glued line (T4).** A sloppy multiedit removed a newline, gluing `Speeds: SpeedValues{}` INTO the `TrackMode` line comment — silently commenting out the field. Every gate stayed green because the zero value equals the intended default; the corruption was semantically invisible to build/vet/race/erraudit and only the `exhaustruct_v5` linter at the PRE-COMMIT hook caught it. A "gates green" tree can still be semantically corrupted — structural linters are a gate class of their own. Also: the failed edit left the file broken for ~2 minutes while other gates ran on it.
2. **GNU sed `$1` backreference corruption (T2).** I used a Perl-ism in `sed -E`, writing literal `$1` into six table rows of `internal/pixy/config_test.go`. Caught by vet; repaired with exact-match edits. Sloppy tool choice for a transformation I hadn't verified sed supports.
3. **Invented API in a test (T3).** The first `TestDesiredCameraMode` draft called a nonexistent `ValidDesiredShim()` method — I wrote an assertion about an API I imagined instead of checking what exists. Caught immediately by compile; rewritten properly.
4. **Stray `mutator ` prefix (T3).** The very first multiedit of the session put a garbage token into `writeTracking`, producing invalid syntax. Caught by my own sed check one step later.
5. **Wrong-file edit (T6).** The `powerStatus` population edit targeted `web_types.go` when the call site lives in `handlers.go` — wasted a round trip and briefly confused the state of two files.
6. **Waybar golden leftover (T3).** After retyping the table field to `DesiredCameraMode` I left the `camera: pixy.StateOffline` row in place — a guaranteed compile error that a second edit then had to fix. Should have been one edit.
7. **Daemon-sweep surprise, five times.** T2/T4/T5/T6/T7 all ended with "nothing to commit, working tree clean" because the auto-commit daemon swept the work mid-battery (~4 min gate runs vs ~2–4 min sweep cadence). I recognized the pattern after T1 but never changed strategy — every consolidation was a soft-reset + recommit. Worked, but it's five avoidable rounds of history surgery on unpushed commits.
8. **Repeated staleness failures.** "File has been modified since last read" (treefmt/daemon racing my edits) hit at least three times; each cost a re-read + re-apply cycle.

## e) WHAT WE SHOULD IMPROVE

1. **Commit BEFORE the full battery, amend after.** With a daemon sweeping every few minutes, the right pattern is: stage+commit the task immediately, run gates, then `--amend` (or a follow-up fix commit) if anything fails. The plan's "gates before every commit" ordering is what exposed every task to mid-battery sweeps.
2. **Structural linters belong IN the gate battery, not just the pre-commit hook.** D1 proves build+vet+race+erraudit can all be green on semantically-corrupted code. `exhaustruct` caught it by luck of ordering (commit-time hook). The documented battery should name `golangci-lint` BEFORE "commit" — it actually did, but the corruption slipped because lint ran AFTER the glue in one pass. Rule going forward: lint before declaring a task's edits finished, not just before the commit.
3. **No regex rewrites with capture semantics in sed.** For table-row retype sweeps, use exact-string edits (multiedit) or `python3` — sed's BRE/ERE has no backreferences and fails silently-then-loudly (as seen).
4. **Read-before-guessing tool paths.** `which erraudit` first would have saved the wrong `go run` attempt (the memory instruction "check for build scripts/tooling first" applies to binaries too).
5. **Consolidation is a job for ONE bash chain.** The reset+commit+verify should always be a single atomic command (it was, mostly) — and `git status` immediately before `git commit` should become a reflex.
6. **The edit-tool staleness loop:** when a formatter/daemon may have touched a file, `view` the exact region FIRST, then edit — never retry blind.
7. **Warm-cache lint caveat should be recorded in AGENTS.md's lint bullet** (the clean-cache rule exists; the "concurrent session may hold the cache" variant does not).
8. **The pin-test discipline worked** — TestDesiredCameraMode, TestDisplayCameraState, TestPresetMapSet, TestStatusContract_BatteryFieldsAdditive all pin exactly the new invariant. Keep writing one per migration; it's the cheapest regression insurance this repo has.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Ranked, grounded in this session's observations + the repo's documented state. (✦ = this session produced the need.)

**Release & records**
1. Cut `v0.6.1` (or fold into next minor): CHANGELOG entries for the six type-safety commits (the sweep currently lives only in git history + TODO_LIST header).
2. Stamp `docs/planning/2026-11-07_07-16_pareto-type-safety-and-layout-plan.md` as EXECUTED with commit hashes (✦).
3. Annotate the two older status reports (`2026-11-07_06-40`, `2026-11-07_08-31`) as superseded by the executed plan (✦ — docs-health ANNOTATE mode).
4. Push-tag protocol: annotated tag + `pkg.go.dev` propagation check per go-release skill when the release lands.
5. Add the sweep to the website release notes filter if it has a type-safety category.

**Type-safety tail (from the review, deliberately deferred — now with recorded triggers)**
6. ROADMAP row: 5th display surface → unify the 4 view structs (D2).
7. ROADMAP row: named second consumer → decomposition sketch (`hidTransport`/`streamServer`/`callDetector`) then package split.
8. ROADMAP row: Lars's call → cosmetic `cmd/`+`internal/daemon/` move.
9. When the hardware session (#166) lands, re-check `TrackModePref.Valid()` against the verified readback spellings (currently accepts ParseTargetTrackMode aliases by contract).
10. `ParseTargetTrackMode` accepts CLI aliases as VALID state values (pre-existing contract) — consider a strict `ValidCanonicalPref()` if a corrupted-but-parsing alias ever matters.

**Daemon/robustness hygiene (✦ observed)**
11. Add `exhaustruct`-class structural lint earlier in the documented gate battery (AGENTS.md Testing/Gates section) — D1 lesson.
12. Record the "concurrent session holds the golangci-lint cache" caveat in the AGENTS.md lint bullet (✦).
13. Consider a `justfile`-free `flake.nix` app `gate` runner that chains build+race+erraudit+lint+nix-build in one command (the battery is 5 memorized commands; a single target removes the memorization tax and makes the pre-battery commit pattern cheap).
14. Daemon sweep cadence vs gate battery: document in AGENTS.md that tasks should commit-then-gate-then-amend (✦ process lesson) — or tighten the auto-commit daemon to skip dirs with an active `.gate-lock` file if that's feasible.
15. `getBody` helper lives in `integration_test.go` while half its callers are contract tests — fine, but a `test_helpers_test.go` consolidation pass would reduce import thrash (✦ saw `io`/`http` import churn three times in one file).
16. `controllable_contract_test.go` mixes wire-contract tests and panel tests — splitting the /api/status contract pins into one file would make "the public JSON interface" greppable.
17. `readState`-generic test helper pattern (used with TrackModePref) is repeated across 4+ test files — one shared typed helper could cut boilerplate.
18. The three fake `setTracking` test injections duplicate the same 5-line body — a shared `withFakeTrackingCapture(&calls)` option would DRY them (✦ edited three copies today).

**Web/widget surface**
19. Waybar `battery` JSON still carries only the display string — consider the same additive percent/charging split for waybar JSON (FEATURES row already notes "charging/discharging classes not yet included (ROADMAP)").
20. `batteryCharging` omits on `false` — a widget that wants tri-state (unknown/not-charging/charging) can't distinguish; document or add `batteryKnown,omitzero` if a widget asks.
21. quickshell.mdx example JSON block (:28) shows only the legacy field — add the two new keys to the example.
22. `/api/health` also carries `camera` — decide whether it should gain the same additive battery fields or stay minimal (it's a liveness probe; probably stay minimal — write the decision down).
23. Web panel battery line still display-string only; the typed fields exist in webStatus now — a UI tick could render a charging icon from `BatteryCharging`.

**Verification depth**
24. Fuzz targets: CI runs them, but `FuzzParseHIDResponse` etc. never ran locally this session — a local smoke (`go test -fuzz=... -fuzztime=30s`) before releases would catch wire-shape regressions earlier.
25. Benchmarks: run the 9-bench suite + benchstat for the type swaps during a quiet window (shared-machine gotcha applies) to have formal perf-neutrality evidence for the release notes.
26. vmTest / integration hardware tests remain BLOCKED on #166 — unchanged.
27. The `-tags=integration` battery on real hardware (#166 bundle) — still the single highest-value blocked item.
28. Golden file for `state.json` bytes (the round-trip test pins the shape; a committed golden file would pin it harder across refactors).
29. Consider pinning `/api/status` FULL JSON with a golden (fields are additive; a golden makes accidental removals fail loudly) — or accept the contract tests as sufficient.
30. `go vet` in CI covers tests; the session's corruptions were all test-file corruptions — a `golangci-lint --tests` emphasis note (already on) is fine; nothing to do beyond awareness.

**Docs**
31. `docs/DOMAIN_LANGUAGE.md` — add DesiredCameraMode/TrackModePref/USBProductID/DegradedReason as domain terms (the glossary should own the new vocabulary).
32. AGENTS.md gotchas: the "auto-commit daemon may sweep mid-task; commit-then-amend" process note (✦).
33. AGENTS.md gotchas: erraudit binary is system-installed (`/run/current-system/sw/bin/erraudit`), not a `go run` module (✦).
34. The layout ADR cross-link: review HTML ↔ ADR ↔ plan ↔ TODO_LIST header all reference each other now; one "type-safety sweep" index line in the review doc pointing at the four landed commits would complete the loop.
35. FEATURES.md "two PARTIALLY_FUNCTIONAL items" prose (§:205) still says v0.5.0 shipped 2026-10-31 — the v0.6.0 release happened since; refresh the prose at the next release.

**Website**
36. Website battery ran green locally; the CI `website.yml` job will re-run it — watch the first CI run on master for the new `batteryPercent` docs rows (types in MDX tables don't affect it, but it's the first docs change post-release-sweep).
37. `#129` online screenshots remain blocked — bundling with #166 unchanged.
38. Demo video re-render is gitignored — untouched, fine.

**CI/workflows**
39. `lint-workflows` job pins actionlint checksum — untouched; verify on next CI run that the new commits pass (they will; no workflow changes landed).
40. Actions suspension (#204) still means NO GitHub runs — branch protection (#156) remains unenforceable until Actions returns; the local CI-parity battery is the only gate (✦ it ran; it held).
41. Consider re-checking #203's placeholder-hash guard caught nothing this session — `nix build` passed on first try, so no hash drift occurred (vendorHash untouched — no dependency changes).

**Nice-to-leverage**
42. `DesiredCameraMode` makes a `String()`-only API; a `ParseDesiredCameraMode(string)` (strict, no offline) would complete the surface if a future CLI verb takes a mode argument directly.
43. `PresetMap.Set` still allows nil-map panic by contract comment — a nil-safe variant or `ensure()` helper if a third write path ever appears.
44. `DisplayCameraState` could take the snapshot struct instead of two args if it grows a third input (YAGNI now).
45. `statusSnapshot.Camera` doc comment references `.CameraState()` projection — the machine-surface projection is currently a bare method call at three sites; a tiny `machineCameraState()` helper would DRY if a fourth appears.
46. `ExternalDepReason` prefix constant `degradedExternalDepPrefix` is unexported and only used by the constructor — could inline it into the constructor (one less symbol).
47. The `withCameraState` option name now duplicates `newTestDaemon`'s positional param — one of the two could go.
48. `TestDisplayCameraState` loops both online and offline cases — a 4th desired mode added later auto-covers; no action, just noting the table is future-proof.
49. Keep the templ drift watch: the concurrent-session recurrence happened TWICE in one day; if a third occurs, consider a git hook blocking `_templ.go` commits whose version header ≠ go.mod version (mechanical check, cheap).
50. After #166 hardware session: revisit `loadState`'s normalize-ANY-invalid-camera semantics (session widened "offline-only" → "any non-intent value"; garbage-string files now survive with privacy + preserved rest — deliberate, but worth a second opinion after the wire format is hardware-pinned).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Release cadence:** should I cut `v0.6.1` NOW with the six type-safety commits (CHANGELOG + annotated tag + push), or do you want the sweep batched into a later release? (Affects when TODO_LIST's "pruned to CHANGELOG" happens and whether website release notes update.)
2. **Daemon-commit policy going forward:** the auto-commit daemon swept every task mid-battery and I squashed each back into one commit per task. Do you WANT that consolidation (clean rollback units), or should future sessions leave the daemon's fragments as-is and stop spending rounds on history surgery?
3. **`batteryCharging` tri-state:** with `omitzero`, a device reporting 0% or not-charging is indistinguishable from "no reading" in `/api/status`. Is absent-on-false acceptable for your widgets, or do you want an explicit third field (`batteryKnown`) so a widget can render "unknown" vs "not charging"?

---

**Session stats:** ~10:55 CET end; 6 code commits + 1 docs commit pushed; ~40 files touched across Go source/tests + 4 docs + 1 MDX; 4 new pin tests; 2 corruptions caught by gates, 1 caught by structural linter at commit; zero gate failures on any pushed HEAD.
