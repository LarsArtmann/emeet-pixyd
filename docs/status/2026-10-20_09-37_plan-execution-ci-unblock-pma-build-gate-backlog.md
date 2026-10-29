# Status Report — Pareto Plan Execution: CI Unblock, pma Build Gate, Backlog Slices

**Date:** 2026-10-20 09:37 CEST
**Session scope:** Execution of `docs/planning/2026-10-19_04-29_SUPERB-systemnix-closeout-and-backlog-pareto-plan.md` (P1 remainder, P2, P5, P6, P7, P8 slices) plus the unplanned CI failure it surfaced. This report covers THIS session only; prior state lives in `2026-10-19_04-19_systemnix-extraction-port-option-monitoring-guide.md`.
**Repos touched:** `emeet-pixyd` (pushed, all CI green through `90efa06`), `projects-management-automation` (LOCAL commits only, not pushed).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **CI unblocked: golangci-lint pin v2.13.2 → v2.14.0** (unplanned, found at P1) | Go tests were red on EVERY master push since 2026-10-10: go1.27.2 shipped stdlib export data v5; the pinned lint (built with older Go) failed typecheck on every import. Empirically verified the exact CI combo locally FIRST (downloaded go1.27.2 tarball, ran devShell's v2.14.0 against it → 0 issues) before bumping. Commit `55b8fd1`; workflows green from then on. |
| 2 | **P1 closed** — all three workflows green on every push this session (Go/Nix/Website verified at `db51226`, `0d75cc8`, `401ee57`, `83c420a`, `90efa06`); monitoring guide confirmed live with sidebar entry |
| 3 | **P2 website polish** — `hardware.emeet-pixy.port` row + Web Port section in configuration.mdx; metrics↔monitoring and quickshell↔monitoring cross-links (incl. `#the-stale-binary-trap` anchor); frozen-lockfile proof (`pnpm install --frozen-lockfile` green); tsc strict exit 0; build 22 pages, CSP 22/22; dist grepped for NEW strings; **deployed page fetched live and content confirmed** |
| 4 | **P6 lockstep cross-refs** — `modules/nixos.nix` port option ↔ `pixy.DefaultWebAddr` each name the other ("change both or neither"); monitoring-guide URL in the option description (renders in NixOS manual/search); fmt + eval green |
| 5 | **P5 default-port pin** — second defaults-only vmTest node asserting unset-port renders `EMEET_PIXYD_WEB_ADDR=127.0.0.1:8090`; full two-node vmTest ran green; the 8090 convention SystemNix couples to is now pinned, not implied |
| 6 | **P8 #177** — `ruff.toml` per-file-ignores (SIM115/EXE001/F841 for `tools/**` only; frozen RE artifacts, byte-verified outputs — restyling would invalidate verification). BuildFlow ruff-check/fix/format/config steps all ✔ with it |
| 7 | **P8 #180** — verdict OBSOLETE: `.crushrc` carries no lint list (toolchain pins only); the LSP reads `.golangci.yml` directly, which DOES gate `wsl_v5`+`golines`. Premise stale, nothing to align; documented in the TODO row |
| 8 | **P8 #181 chunk 1** — nolint inventory (107 directives / 39 files, top: handlers.go ×12) + removed uevent_test.go's phantom exhaustruct prose (linter is test-excluded; the directives it described never existed). All other test-file directives verified LIVE |
| 9 | **P8 #148 prep** — duplicate search done: innoextract master's newest version work is 6.4.0, no 6.6.x issues/commits → the staged draft (`tools/inno661/UPSTREAM.md`) is novel and ready; sending stays gated on Lars |
| 10 | **P8 M18** — root-caused the `--all-systems` failure (flake declared `aarch64-darwin` while the package is `lib.platforms.linux` → eval always refused there), dropped darwin from `systems`, verified `--all-systems` green locally, wired it into `nix.yml`. aarch64-linux eval now CI-covered |
| 11 | **P8 #178 (baseline half)** — `docs/benchmarks/baseline-2026-10-20.txt`: 9 benchmarks × 5 runs, benchstat-able — see (b) for why this is only half the task |
| 12 | **Session gotchas into AGENTS.md** — lint-pin-tracking rule (next Go patch release will recur), BuildFlow templ-generate version-drift trap, local sandbox format-check false failure |
| 13 | **Final local gates** — `go vet` + `go test -race ./...` green on emeet-pixyd at session end |

## b) PARTIALLY DONE

| # | Item | What's missing |
|---|------|----------------|
| 1 | **P7 / #176 — pma pre-commit build gate** | Code DONE and verified: `buildgate.go` (opt-in `require_clean_build` + `require_clean_vet`), gate in `processBatch` after every cheap check and before any `git add`, separate escalating cooldown that re-arms on events NEWER than the last failure (flaw my own e2e test caught — original design would have delayed a fixed tree's commit by the 5m floor), `GOWORK=off`, missing-go → WARN-once + ungated (never a fleet blackout), config Validate/Merge, NixOS `requireCleanBuild`/`requireCleanVet` options + go added to service PATH, vendorHash reconverged, 12/12 lint stages + race tests + `nix build .#default` green (re-verified after final fmt). **Missing:** (a) pma master NOT pushed — deliberate, Lars's repo, deploy rides his SystemNix rebuild; (b) gate never exercised against the LIVE daemon watching a real repo (unit + processBatch-level e2e only); (c) default-on-for-Go-repos ruling still open (J3); (d) host activation = set the option in SystemNix + rebuild |
| 2 | **#178 perf-neutrality** | The TODO asks to CONFIRM `ttlCache[T]` + preset double-RLock are perf-neutral post-refactor — a post-only baseline cannot confirm that retroactively (no pre-refactor numbers exist). What I shipped anchors FUTURE deltas. Honest redefinition; the original ask needs either a pre-refactor cherry-pick measurement or an explicit downgrade of the row |
| 3 | **#173** | ADR Option B is complete and I recorded the 5-bullet implementation design into the TODO row during self-review — but the IMPLEMENTATION (~90m) is untouched and gated on Lars's sign-off |
| 4 | **#155** | Verdict recorded (cut **v0.5.0**, not v0.4.1 — 57 entries incl. new commands + CLI format changes) into the TODO row during self-review; the actual cut (changelog section, tag, proxy verify) is Lars's cadence call |
| 5 | **#181** | 1 of ~4 chunks done; handlers.go (×12) and the production files remain |
| 6 | **pma repo hygiene discovered mid-task** | Stale untracked `vendor/` (64 MB, gitignored residue from a 03:27 nix build) was flooding the split-brain lint stage — trashed it, gotcha recorded in pma AGENTS.md. Root cause (WHICH build path materializes vendor/ into the source tree) not chased |

## c) NOT STARTED (this session's scope boundary)

- **P3/M3 SystemNix wiring** (`port = ports.emeet-pixyd`, lock bump, runbook) — gated on Lars's cross-repo go-ahead, unchanged
- **M8/M9** (#173 implementation + #174 tests) — gated on the ADR sign-off
- **#179 implementation** — recommendation recorded (validate in `presetAction`; flag the multi-word truncation through the command-string path), code untouched
- **#148 send / #156 branch protection / #154 @zutto ping** — owner actions
- **#166 hardware bundle** (#129, #138–#141 verdicts, #150, #182) — needs the wired PIXY session
- **Dependabot triage** — push output showed 5 npm vulns (3 high, 2 mod) and failing `npm_and_yarn` dependabot runs; noticed, never opened. Nothing recorded until now
- **`website/src/styles/global.out.css`** — the daemon swept +1723 lines of Tailwind build output into git (`6dfc98a`); it was previously tracked so I left it, but the track-vs-gitignore question was never investigated
- **BuildFlow upstream filing** — its templ-generate step used a templ version ≠ go.mod's pin, regenerating `templates_templ.go` into non-canonical form that the daemon committed (I regenerated canonically + documented the gotcha locally; the cross-repo finding was NOT filed against buildflow — close-the-loop rule says fixed, filed, or documented-non-fix; it is currently only documented here)

## d) TOTALLY FUCKED UP!

Nothing irreversible, nothing red at HEAD, no data lost. Honest failures:

1. **Lost the daemon commit race TWICE.** Composed detailed commit messages for `ruff.toml` and the backlog batch; both times the auto-commit daemon swept the staged files first and my `git commit` died on "nothing to commit". Wasted work both times; the daemon's heuristic messages shipped instead of mine. I KNEW it sweeps continuously — should commit immediately after staging or not compose long messages at all.
2. **Chased the wrong failure briefly on pma lint.** First lint run failed; tail showed stages 6–12 passing so I re-ran and scrolled — stage 5 (split-brain) was spewing 60+ vendor/ "duplicates". I briefly treated it as possibly-my-code before recognizing untracked-build-residue. Cost: one wasted lint cycle. Fix was correct (trash vendor/).
3. **Verification-ordering gap in pma.** Ran lint, THEN `nix fmt` (which reordered imports in 4 files), and initially moved on without re-running lint/tests after the format change. Caught in self-review; re-ran both — green. Sloppy sequencing regardless.
4. **tsc gate evidence chain was weaker than claimed.** First invocation accidentally ran `pnpm run typecheck` (= `astro check`, broken-by-design per AGENTS), output was ambiguous; second run was bare `tsc --noEmit` (exit 0, project tsconfig) — correct but I asserted "type gate green" before the clean run existed.
5. **Session truth evaporated into chat until forced out.** The #155 verdict and #173 five-bullets existed only in my closing message until this self-review — exactly the "entombed in a conversation" failure docs-health exists to prevent. Also left the TODO_LIST "Updated" header at 2026-10-19 and added no CHANGELOG entries for the session's shippable work (CI pin fix, all-systems eval, ruff.toml). All fixed during this self-review (header bumped, verdicts into rows, three CHANGELOG entries added).
6. **The `.crushrc` obsolescence claim (#180) was asserted from absence.** I verified no lint list exists in `.crushrc` and none in the wrapper — but I never actually OBSERVED live LSP wsl_v5/golines noise to confirm the reported problem is gone vs. never-existed. The verdict is very likely right; the evidence is "config says so", not "reproduced".

## e) WHAT WE SHOULD IMPROVE!

1. **Commit-fast-or-don't-compose** (vs. the auto-commit daemon): stage → commit immediately with a short message; save long-form rationale for CHANGELOG/AGENTS, not commit bodies the daemon will orphan.
2. **Verdicts into TODO_LIST the moment they're formed** — chat is not memory. This session violated it twice before self-review caught it.
3. **Close the loop on cross-repo findings**: the buildflow templ-version drift is documented in THIS repo's AGENTS but the owning repo (buildflow) never heard about it. Skill rule: fixed, filed, or documented-non-fix IN THE OWNING REPO.
4. **Format before lint, always** (pma gap): run `nix fmt` first, then the lint/test gates, so the gates measure the final form.
5. **Investigate tracked build artifacts on sight**: `global.out.css` (+1723 lines of daemon-swept Tailwind output) — "previously tracked" is not the same as "should be tracked".
6. **CI red ≠ investigate later**: Go tests had been red for NINE DAYS (10-10 → 10-19) across multiple sessions incl. pushes; everyone (me included, until forced by P1) treated "watch CI go green" as the task without asking "is it green NOW?". The P1 check caught it only because the plan explicitly said to look.
7. **Dependabot failures on push banners**: three pushes printed "5 vulnerabilities, 3 high" — screen-blindness to repeated warnings is how the 9-day CI red happened in miniature. Record or triage on first sight.
8. **Status-report skill default is HTML**; user explicitly demanded `.md` — honored, flagged (same as 2026-10-19 session; this is now a twice-recurring operator preference worth a skill-side standing exception, like the task-queue one).

## f) NEXT — up to 50 things, impact-sorted

**Owner-gated (unblocks the most):**
1. Lars: go-ahead for SystemNix port wiring (P3: lock bump + `port = ports.emeet-pixyd` + runbook, 30m)
2. Lars: push pma master + set `requireCleanBuild = true` for the host + rebuild — activates the red-at-HEAD killer fleet-wide
3. Lars: J3 default-on ruling (opt-in vs default-on-for-Go-repos)
4. Lars: v0.5.0 cut decision (verdict recorded in #155; changelog cut + annotated tag + proxy/pkg.go.dev verify ≈ 60m once approved)
5. Lars: #156 branch protection requiring the three workflows on master (20m, console)
6. Lars: #148 — send the staged innoextract 6.6.1 issue/PR (draft + duplicate-search done)
7. Lars: schedule the #166 wired-PIXY session (closes #129, #138–#141 verdicts, #150, #182)

**High-value, unblocked:**
8. pma: live-daemon gate validation — run the real daemon against a temp watched repo with a broken Go file, confirm hold + journal WARN + fix-commits (the one verification level not yet exercised)
9. #173 implementation + #174 tests (ADR Option B fully specified, ~90m + 60m)
10. #179 implementation (4 lines + test; plus fold the multi-word decision into the 2026-09-18 ADR)
11. Dependabot triage: read the 5 npm vulns, rebase/recreate the failing `npm_and_yarn` branches, decide the update path
12. `global.out.css`: determine if it's load-bearing (Astro build input?) or sweepable; gitignore + `git rm --cached` if not
13. File the buildflow templ-version-mismatch finding upstream (its codegen step must pin templ from go.mod or skip when the repo pins differently)
14. #181 remaining chunks: handlers.go ×12, then metrics/main/identity/hid (mechanical, nolintlint- or grep-driven)
15. pma J2 (heuristic fallback rate — 7,898/week; log the provider failure cause) — adjacent to this session's gate work
16. pma M2: wire `NoVerify: c.SkipHooks` (its own plan 2.1–2.5 rows) so repos CAN run their pre-commit hooks again
17. #178 downgrade-or-measure decision: either cherry-pick the pre-refactor commit and benchmark it, or reword the row to "baseline anchored, future deltas comparable"
18. vmTest runtime cost check: the second node adds ~30–60s to every CI nix run — confirm acceptable or convert the default-pin to a cheap `nix eval` assertion
19. Website screenshots refresh (#129) — blocked on hardware, but the OFFLINE-state shots could be re-taken against the improved panel now
20. Read the go1.27.2 export-data lesson forward: add a note to ROADMAP/AGENTS to expect the same break at go1.28.0 (already partially in AGENTS; a CI canary would be better)
21. pma: chase which build path materializes `vendor/` into the source tree (root cause of the split-brain flood; the trash is symptom relief)
22. Re-verify the #180 claim empirically (open an editor session, observe LSP noise for wsl_v5/golines) then close the row
23. CHANGELOG drift check: my three new entries reference workflow behavior — confirm the CHANGELOG conventions want CI entries (they have precedent, but worth a docs-health VERIFY pass)
24. The dependabot go_modules PRs that failed on the old lint pin: rebase one to prove they go green with v2.14.0 (unblocks the vuln-fix pipeline for Go deps too)
25. `website/flake.nix` unused-input drop (from b109788 audit) — verify no other stale inputs linger
26. Add a `benchstat` comparison workflow note to docs/benchmarks/ so the next delta run has a canonical command line
27. pma: the gate's journal-verification plan (httputil proposal step 3) — after a week of live gating, `journalctl -t pma | grep build-gate` vs heuristic counts
28. Sweep the session's docs/status harvesting: this report's (f) items 8–27 into TODO_LIST/ROADMAP if not already rows (most map to existing rows; 11, 12, 13, 17, 18, 21, 22, 24 are NEW)
29. Consider a tiny CI check that fails when `[Unreleased]` exceeds N entries or M days — release-cadence pressure made mechanical
30. Backfill: `docs/planning/2026-10-19_04-29` plan errata — P7 was implemented cross-repo (pma), the plan's verschlimmbessern guardrail said "no daemon changes beyond the gate" — the gate IS the sanctioned change, but the errata should record where it landed

**Lower / nice:**
31. `tools/inno661/UPSTREAM.md` checklist item 2 (re-download specimen + rerun toolchain end-to-end) before Lars files upstream
32. Quick interview of `/tmp/go1.27.2` download: delete it (70 MB in /tmp) or keep for the next pin bump
33. Test count assertion: this session added 9 pma tests + 1 config case — pma FEATURES.md/coverage unaffected; emeet-pixyd coverage stayed flat (docs-only + nix) — nothing to do, recorded for honesty
34. Investigate whether `TestGateBlocksCommit_*` runtime (2.8s race) is worth a `testing.Short()` guard for pma's faster CI loops
35. pma NixOS module: consider asserting `requireCleanVet → requireCleanBuild` at the nix level too (currently daemon-validated only)
36. emeet-pixyd: `website.yml` could cache pnpm store by lockfile hash if not already (check before doing — may exist)
37. The three heuristic daemon commits that carried this session's work (`09dc1b0`, `6dfc98a`, `83c420a`) — consider `git rebase` to reword with real messages while unpushed... they ARE pushed; note as accepted cost instead
38. Record in crush-config `references/lessons.md`: "verify the CI pin toolchain combo in isolation before bumping" (cross-project lesson from the export-data break)

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **pma gate activation:** May I push pma master, and do you want `requireCleanBuild = true` on your host (SystemNix) now — or do you want to review the implementation first? (Also: default-on for Go repos per J3, or stay opt-in?)
2. **SystemNix go-ahead (P3):** wire `hardware.emeet-pixy.port = ports.emeet-pixyd` + bump the lock there now?
3. **v0.5.0 timing:** cut it from [Unreleased] now, or hold until the #166 hardware session upgrades the "shipped but unverified" features (speed/tracking/battery/presets) to verified-in-changelog?

---

## Verification state at time of writing

| Surface | State |
|---|---|
| emeet-pixyd master `90efa06` | Go tests ✓, Nix ✓ (all CI pushes green this session); local vet + race tests ✓ |
| Live website | configuration page with port row + Web Port section confirmed fetched; monitoring guide + sidebar confirmed earlier |
| vmTest | green with both nodes (defaults-pin + non-default-pin) |
| pma master (local, ahead of origin) | 12/12 lint, race tests, `nix build .#default` green incl. after final fmt; commits: heuristic sweeps + `1170ec01` |
| Working trees | emeet-pixyd: clean except this report (daemon will sweep); pma: clean |
| Known red | dependabot `npm_and_yarn` branches (pre-existing, untriaged); no master workflow red |
