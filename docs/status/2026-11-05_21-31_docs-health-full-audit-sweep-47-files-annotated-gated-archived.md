# Status: Docs-Health FULL AUDIT Sweep — All 2026-* Reports Annotated, Gated, Archived

**Date:** 2026-11-05 21:31 CET
**Session scope:** The standing docs-health AUDIT ("view ALL 2026-0* files, execute docs-health SUPERBLY, make the six living docs superb, archive fully-done files"). This report covers ONLY this session's run. Brutal-self-review format per requester.

---

## Session summary (one line)

All 47 non-archived `2026-*` status reports + plans were read, annotated inline (per-item evidence-cited verdicts, ~1,900 lines struck), gated (`check-rows --strict` + `bare-rows` 47/47 clean), and archived with manifests; the six living docs were rebuilt/refreshed; six new TODO rows were harvested; the `docs-gates` battery passes **5/5**; Go build/vet/full race suite green.

---

## a) FULLY DONE

| # | What | Evidence |
| --- | --- | --- |
| a1 | **Ground truth first**: git log (80 commits), TODO_LIST/ROADMAP/FEATURES/README/CHANGELOG read in full before any judgment | session log |
| a2 | **All 48 docs inventoried** — 6 parallel extraction agents (one rate-limit retry) produced per-item verdict inventories with line numbers + evidence; newest reports (10-19 → 11-01) read personally | agent outputs + this session's reads |
| a3 | **Living docs made superb**: FEATURES.md (+Resilience section ×5 rows, +Website & Docs ×4 rows, counts recomputed 75→83/81/2, freshness re-stamp 2026-11-05); AGENTS.md (+nolint coverage semantics + clean-cache doctrine); ROADMAP.md (+14 bullets across error/web/observability themes); DOMAIN_LANGUAGE.md (+Early Abort, +Dry Run) | file diffs, committed by pma (`c3d201d` batch) |
| a4 | **TODO_LIST rebuilt**: DONE rows pruned (work → CHANGELOG per docs-health convention); #186 drift FIXED (done 2026-10-31, row sat OPEN); harvested **#196** (saveState write-side intent guard), **#197** (`controllable` contract pins), **#198** (Compose `errors.Is` pin), **#199** (website CI/secret hardening bundle), **#200** (a11y/device checklists), **#201** (vmTest watchdog-stall test); #195 extended (rebase failing dependabot branches) | `TODO_LIST.md` (23 open rows) |
| a5 | **Website doc gaps closed on sight**: quickshell.mdx gained the `controllable` field-table row (contract gap verified by grep: 0 mentions); presets.mdx gained `--dry-run` + early-abort; both verified against `handlers.go` contract | quickshell.mdx, presets.mdx |
| a6 | **Code-comment truth restored**: v2head.go's TWO version-shift remnants (evidence-grades block + V2GetMotorSpeed note) reworded to the retracted-hypothesis reality; build + pixy tests green after | `internal/pixy/v2head.go`, `go build` |
| a7 | **ANNOTATE sweep executed**: ~1,900 items struck across 47 files via the `annotate` toolkit (repair/repair-partial/repair-mangle class fixes), per-cell house grammar, per-item evidence verdicts (`done at hash` / `done — CHANGELOG [0.5.0]` / `NOT-DO — TODO_LIST #N` / `Superseded — pointer`); three genuinely-open classes honestly left bare | `git show c3d201d --stat` |
| a8 | **ARCHIVE**: 7 plans + 40 status reports + 1 HTML → `archive/` via `git mv`; **gates run BEFORE each move** (`check-rows --strict` + `bare-rows`, 47/47 complete + clean); bulk-archive manifests written (`docs/status/archive/README.md`, `docs/planning/archive/README.md` — filename → classification → deciding reason) | archive dirs + manifests |
| a9 | **Live report routing**: 2026-11-01 stays unarchived as the current report; all 35 §f items routed (`→ TODO_LIST #196–#201`, `→ ROADMAP`, `→ TODO #166 checklist`, 3 DONE-marked: nolint lesson, FEATURES sweep, plan archival) | 2026-11-01 §f |
| a10 | **docs-gates battery green**: `nix run github:Artmann-Technologies/docs-gates -- run .` → **PASS freshness, PASS audit-all, PASS annotate, PASS routing (0 unrouted open rows), PASS cadence (PURGE-HEALTHY + ARCHIVE-HEALTHY)** | battery verdict matrix |
| a11 | **Quality gates**: `go build ./...`, `go vet ./...`, targeted tests, FULL `go test -race -count=1 ./...` green (16.9s + 1.1s); `gofmt -l` clean on the touched Go file | test output |
| a12 | **8 stale claims corrected inline**: version-shift remnants ×2 (code), `.node-version` false negative (report claimed it missing; it EXISTS — verified by bash `ls`, glob skips dotfiles), TS-pin direction, "nothing pushed" ×2, CHANGELOG section pointers, nolint "all intentional" verdict (wrong — 16 were stale, #181) | struck lines in archived files |
| a13 | **One real content mangle repaired**: 13-46:16's udev code span (`00c0\|0118`) had been split across struck table cells — rebuilt without span-crossing pipes | archived file L16 |

## b) PARTIALLY DONE

| # | What | Why partial |
| --- | --- | --- |
| b1 | **Fleet-trio annotation granularity** — 20-14/00-20/01-23 (and parts of 15-09) got SECTION-grain verdicts (`executed — vision-review-agent ops trackers`) on their 100+ cross-repo §f rows instead of per-item verdicts. Honest at archive grain, coarser than the art-dupl corpus standard. The emeet-relevant rows ARE per-item | token budget; those items live in another repo's trackers |
| b2 | **#197 quick win left on the table** — the guide half (quickshell.mdx `controllable` row) is DONE, but the JSON-shape test assertion (5–10 lines in `web_status_test.go`) was harvested as TODO #197 instead of implemented. A code change in a docs sweep was judged out-of-scope; defensible either way | scope discipline vs fix-on-sight |
| b3 | **TODO_LIST header drift risk (self-inflicted, known)** — I compressed the 2026-10-20/10-31b history lines while rebuilding; the exact failure mode F092 fixed (header compressing row truth) applies to my header too. The rows are the truth; header says so | inherent tension |
| b4 | **Prose-question sections in archived reports** — `g)` questions carry verdicts where events answered them, but a few Lars-ops questions (series format, /tmp ownership) are annotated as standing questions rather than routed into ROADMAP open-questions (clutter judgment call) | judgment call, not omission |
| b5 | **Living-doc rule-F quality** — 3 of the 17 rule-F violations were MY OWN new text (`simulatorOption`s, `MessageTemplate`s plurals). Fixed mechanically (space separation) in archives, reworded in living docs; a re-read of my new ROADMAP/AGENTS bullets for the same idiom was NOT done | mechanical fix verified by gate; prose re-read skipped |

## c) NOT STARTED

- **golangci-lint run** — the session touched one Go file (comments only); build/vet/full race suite cover it, but the 0-issue lint bar was not re-confirmed on a clean cache.
- **`nix flake check` / vmTest** — untouched surface (docs + one comment); not run.
- **Push** — nothing pushed (standing ROADMAP push-cadence question; pma holds local commits).
- **Website deploy** — the quickshell/presets guide fixes ride the next `website/**` push (CI deploys; not triggered manually).
- **docs-gates wiring** — the battery ran manually; seeding `.docs-gates.json` and wiring one command into CI/BuildFlow (fleet adoption pattern) not started.
- **Cross-project lessons commit** — the off-by-one spec-shift lesson + "emit-keys/verify before spec" belong in crush-config `references/lessons.md` (by commit, never in-session writes) — not done.
- **TODO #197 test half, #196, #198, #199, #200, #201** — harvested this session; implementation not started (they are rows, correctly).
- **21 reasoned erraudit `//nolint` directives re-triage** — ROADMAP-grade, untouched.

## d) TOTALLY FUCKED UP

Nothing reached HEAD broken; all gates green at rest. Honest process failures, descending severity:

| # | Miss | Severity |
| --- | --- | --- |
| d1 | **Off-by-one strike shift** — my fine-row spec generator emitted 0-based indices as 1-based line numbers: ~400 strikes landed ONE LINE HIGH (table separators struck, each range's true last row left clean, boundary rows carrying the neighbor's verdict). Caught only because `check-rows --strict` flagged the "one CLEAN row per struck table" pattern; fixed by a truth-based rebuild (recompute verdicts, clean_cell + canonical restrike). The bug was visible in the FIRST dry-run (118/119, 9/10…) and I initially misread it as continuation rows — two misdiagnoses deep before reading the generator's own indexing. Root cause: mixing enumerate() 0-based with repair's 1-based linespecs without a unit check | **Medium** (caught by gate, no archive damage; cost ~5 rounds) |
| d2 | **Silent spec-generation bug** — the `rows()` helper RETURNED spec lists that the caller never appended: ~60 M-row verdicts for the 09-18 + 09-28 plans silently vanished; only the dry-run diff ("would annotate 0 action lines") exposed it. Writing generators without asserting expected-spec-counts is the same "trust a count without an expectation" class as the nolint-sweep d1 incident from 11-01 — I repeated a documented failure mode | **Medium** |
| d3 | **Verification-pipeline bug blamed on the world** — after the archive `git mv` loop, my status check ran `git status --short docs/status/` FROM INSIDE `docs/status` (double-prefixed path → warning) and `grep -c` (exit 1 on zero matches) — I read the clean tree + pma's commits as "moves failed, daemon did something" and spent a round investigating before `ls archive/` proved the moves had landed. The work was fine; my verification command was wrong | **Low-Medium** |
| d4 | **fix_shift.py cwd bug** — ran `check-rows` from repo root with planning-dir basenames → zero findings for six "fixed 0" rounds before printing the raw subprocess output. Should have printed findings BEFORE the fix loop | **Low** |
| d5 | **Own scripting churn** — `spec_status_a3.py` NameError (helpers not defined), one heredoc write that silently produced no file, an f-string quote mismatch: three rounds on my own tooling. Pattern: composing too much logic per bash call without incremental verification | **Low** |
| d6 | **Agent claim nearly propagated unverified** — an agent reported "no `.node-version` file exists" (glob skips dotfiles); I caught it with a bash `ls` before it reached any doc, but I had ALREADY drafted the claim into a stale-claims entry. The verify-external-claims reflex fired one step late | **Low** |
| d7 | **Ledger marker silently inert** — wrote `<!-- check-rows:ledger --> explanation` (suffix text); the scanner requires the marker ALONE on its line, so the exemption silently didn't apply until I read the tool source. A gate's exact-match contract should have been assumed, not discovered | **Low** |
| d8 | **`edit` without prior View** — two failed edits (v2head.go, quickshell.mdx) before reading them in-conversation; cost two round trips | **Low** |
| d9 | **HTML archived without inline banner** — the 05-05 HTML snapshot was archived on the strength of the manifest + superseding annotated reports; a purist ANNOTATE pass would add an HTML-comment supersession banner in the file itself | **Low** |

## e) WHAT WE SHOULD IMPROVE

1. **Assert spec counts before applying generators** — every spec-writing script should print `expected N, generated M` and abort on mismatch. Both d1 and d2 would have died instantly.
2. **0-based/1-based discipline** — any script that maps line indices into a 1-based linespec API needs a one-line self-check (round-trip one known line through the API) before bulk use.
3. **Gate-first loops, not gate-last** — the winning pattern this session was `check-rows → targeted fix → re-gate` (iterate-until-clean). I reached it late; start every annotation batch with the loop scaffold pre-built.
4. **Read the tool source BEFORE using exotic features** — the ledger-marker exact-match contract and repair's linespec semantics were both discovered by reading `annotate.go` after failures. Ten minutes up front beats five debugging rounds.
5. **Verify agent claims with the file system, not the agent's own method** — glob skips dotfiles; `ls` does not. Cross-tool verification for existence claims.
6. **Keep per-item grain even for cross-repo sections** — the fleet-trio uniform verdicts are honest but a future reader cannot tell WHICH ops item a row maps to. Next sweep: one clause per row.
7. **The battery belongs in the repo's quality loop** — `docs-gates run .` catching the rule-F drift + routing violations automatically is exactly the standing gate the fleet pattern prescribes; wire it.
8. **Re-read your own new prose for the idioms you just gated on** — I introduced three `` `X`s `` plurals while fixing others.

## f) NEXT THINGS (up to 50, ranked; ⛔ = external gate)

**P1 — Lars decisions that unblock chains:**

1. ⛔ Q1: SystemNix push + `nix run .#deploy` + gate dry-run (activates #176 pma build gate + #183 port wiring)
2. ⛔ Q2: emeet-pixyd input float-vs-pin in SystemNix (`scripts/float-register.sh`)
3. ⛔ Q3a: #148 — send the staged innoextract 6.6.1 upstream PR (verify-before-filing + github-voice queued)
4. ⛔ Q3b: #156 branch protection via `gh api` (2 min) or Lars's web UI
5. ⛔ #154: close issue #6 once @zutto confirms the PIXY 2K
6. ⛔ Schedule the #166 wired-PIXY session (closes #129/#138/#139/#140/#141/#150 + un-gates #182/#172)
7. ⛔ ADR queue triage: #116 structured command types, #123 multi-word presets (join-remaining ~6 lines after), NixOS `extraProductIds`, OG/comparison-page publishing calls
8. ⛔ /tmp cleanup ownership ruling (machine ops; GOTMPDIR workaround holding)
9. ⛔ v0.5.1 timing: [Unreleased] currently carries 9 entries (#173–#175, #179, #181, #190–#194 work); releases are deliberately manual

**P2 — new rows from this sweep (unblocked, highest value first):**

10. #197 contract pins: `controllable` in `TestWeb_StatusEndpointJSONShape` + Waybar two-axis golden + collapse pins (guide half already done)
11. #196 saveState write-side intent guard (decide refuse-vs-normalize first — retro g1)
12. #199 website CI/secret hardening bundle (pin firebase-tools, concurrency, smoke, derived page count, html-validate/tsc steps, SA key-ID + rotation cadence, rollback runbook)
13. #201 vmTest watchdog-stall pinning test
14. #198 errors.Is-through-Compose pin test
15. #200 execute the a11y + real-device checklists (flips the two 🟡 FEATURES rows)
16. #195 dependabot sweep: rebase failing `npm_and_yarn` branches first, then `pnpm audit fix` + proofs

**P3 — sweep follow-through (no blockers):**

17. Run `golangci-lint` on a clean cache to re-confirm 0 issues post-sweep (one comment touched)
18. Re-read the new ROADMAP/AGENTS prose for `` `X`s `` idioms (b5)
19. Seed `.docs-gates.json` + wire `docs-gates run .` into the repo's quality loop (fleet pattern: dprint → battery → repo gates)
20. Commit the two cross-project lessons to crush-config `references/lessons.md`: (a) 0-based/1-based linespec discipline, (b) spec-count assertion before apply
21. Website push so the quickshell/presets guide fixes deploy (rides website.yml)
22. Per-item verdict upgrade for the fleet-trio §f rows (b1) — next time that corpus is touched
23. Add an HTML supersession banner to the archived 05-05 HTML (d9)
24. ROADMAP: consider a "docs/status hygiene" line — the archive grew 12→52 status files this sweep; manifests exist, an archive index could help navigation
25. Verify the pma-swept commits preserved the manifest + routed-verdict content byte-identically (spot diff `c3d201d` vs working intent)

**P4 — standing hardware bundle (TODO #166 checklist, unchanged):**

26. ⛔ Run `TestIntegration_BatteryProbe` → #139 verdict
27. ⛔ Pin MotorType/DefaultPosMode → #150
28. ⛔ Speed unit + limit → #138 clamp + slider max
29. ⛔ V1 `09 01 01 01` mode-query decode (unblocks #182 trust)
30. ⛔ Preset slot-count + response-shape pin → #141 (+ Lars's Q1 on SET-echo acquisition)
31. ⛔ Live `preset push` → `preset pull` round-trip
32. ⛔ Online web-UI screenshots → #129 (+ panel crop + poster refresh)
33. ⛔ Panel-honesty decision for the transient trap cover
34. ⛔ Optional usbmon capture (Lars's Q3)

**P5 — ROADMAP-grade offshoots (captured 2026-11-05, unscheduled):**

35. Wire the erraudit STRICT gate into CI
36. DOMAIN_LANGUAGE error-code taxonomy section
37. Per-component health registry (`map[string]func() error`)
38. Five-layer health model documented in monitoring.mdx (D2 exists)
39. Breaker trip reason + last-trip timestamp in `/api/health`
40. Presence-dimension split on `emeet_pixyd_camera_state`
41. Identity-query TTL cache (power-cache pattern)
42. "Why this exists" docs page
43. OG-card custom font (Space Grotesk) + long-title truncation check
44. PNG→webp screenshots; demo-video reshoot triggers (audio, 9:16, scenes, honesty line, GIF fallback)
45. Shared-SA ADR + Workload Identity evaluation (web presence)

**P6 — hygiene (micro, no rows by convention):**

46. `dryRun` field placement in `presetPullOutcome` (micro refactor)
47. Preset-push tests still in `power_test.go` (cosmetic move)
48. Four-segment variant-picker golden assertion (micro)
49. `global.out.css` track-vs-gitignore decision (micro)
50. benchstat command note in `docs/benchmarks/` (micro)

## g) QUESTIONS (cannot figure out myself)

1. **TODO_LIST DONE-row pruning convention**: I pruned all ✅ DONE rows this sweep (docs-health canon: completed work lives in CHANGELOG only). Previous sessions kept DONE rows as inline verdict evidence (e.g. #155's "cut v0.5.0 not v0.4.1" verdict). The verdicts ARE recoverable from CHANGELOG + the archived reports — but do you prefer the pruned (current) shape, or should I restore a compact "recently resolved" digest line per sweep?
2. **Archive granularity for cross-repo fleet reports**: the vision-review-agent trio (20-14/00-20/01-23) is now archived HERE with section-grain verdicts routing to the ops trackers. Should this repo keep archiving those session reports at all, or should cross-repo fleet sessions live ONLY in the owning repo (this archive would stop accepting them)?
3. **docs-gates adoption**: the battery (5 gates) just caught real drift (rule-F spacing, routing, cadence) that manual review missed for weeks. Want it wired into this repo's standing quality loop (`.docs-gates.json` + one CI/BuildFlow step), fleet-wide like pbx-artmann's `scripts/docs-gates.sh` composition?

---

**State of the tree:** working tree = the 17 rule-F spacing fixes + this report (pma sweeps). HEAD `c3d201d` holds the annotation bulk + archive moves + living-docs rebuild. Nothing pushed. Race suite green at rest. `docs-gates run .` = PASS ×5.

*Point-in-time snapshot. When superseded, annotate — do not rewrite.*
