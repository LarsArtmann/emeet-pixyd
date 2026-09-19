# Status — Website-Fleet Follow-Up Execution III (overnight session)

- **Date:** 2026-09-20 01:23 CEST
- **Session scope:** resume the 00:20 report's "NEXT 50" list while the
  fleet review pass was blocked by external load.
- **Verdict up front:** seven NEXT items closed with verified evidence
  (guard-as-CI-pilot, shot-inventory assertion, pre-flight gates, escalation
  timer, hero-copy audit, capture retry, pnpm-gotcha sweep 14/14), and the
  **token-convergence backlog (68 INFO) is cleared at source level** — the
  family guard now reports 0 FAIL / 0 INFO. The review pass itself never
  ran: load oscillated 14→88 all night, and llama-server **died** at ~01:07
  (restarted, healthy). A load+vision-probe-gated waiter will run the pass
  autonomously the moment the box is viable.

## a) FULLY DONE (verified this session)

| Item | Evidence |
| --- | --- |
| **#8 shot-inventory assertion** (`cdp-shoot.py`) | script now computes expected = pages×modes×viewports per site and fails the site on shortfall (`INVENTORY-FAIL` + exit 1) — the d2 class (silent last-subpage-only capture with `0 failures`) is closed. Verified: clean run exit 0; mutated-copy run prints `INVENTORY-FAIL: captured 1, expected 2` and exits **1** (checked real exit code, not the pipe's). DONE line now emits total shot count |
| **#9+#43 pre-flight gates** (`review-fleet.sh`) | 0a load guard (default 10, `FLEET_MAX_LOAD`), 0b binary-freshness gate (any non-test `.go` newer than the binary → ABORT — kills the stale-binary own-goal class), 0c canary latency check (1-token completion; FAIL→ABORT, >45 s→DEFER), plus cycle-duration + canary-time logged at the end. **All branches verified live:** DEFERRED at load 14.10; `canary ok: 0.2s` proceeded; forced FAIL (`FLEET_CANARY_SECS=0.000001`) → `ABORT: canary request failed: FAIL TimeoutError` |
| **#15 KNOWN-broken escalation** (`site-monitor.sh`) | a host continuously KNOWN re-alerts (critical) every 7 days; per-host `known-since`/`last-escalation` state files; recovery removes them. **E2E verified:** backdated `cmdguard.lars.software` known-since by 8 days → one run produced `ESCALATION cmdguard.lars.software known-broken 8d`; immediate re-run did **not** repeat (once-per-day); typespec (fresh clock) correctly silent |
| **#19 score-trend digest** (`review-fleet.sh`) | per-site averages compared against the previous cycle's logged scores (state file `fleet-scores.prev`); `TREND <site> avg X.Y -> Z.W` lines when the average moves ≥ 0.2. `avg10` helper unit-tested (21/3 → 7.0). First TREND lines appear on the second cycle |
| **#12/#11/#39/#40 one-pass fleet audits** (`scripts/fleet-audit.py`, new) | canonical: 15/17 ok incl. go-workflow-auditlog (parity done, no action); **cmdguard + typespec-asyncapi canonicals point at their KNOWN-broken `.lars.software` domains** — same root cause as the console-attach blockers, not fixable by repointing (would flip the mismatch when the domains go live). CSP: 16 none + templcomponents enforced and fully coherent (`manifest-src 'self'`, hashed script-src) — nothing to re-check after the og/meta additions (crawler-facing, CSP-invisible); no report-only CSPs exist. hreflang: 16 single-locale none = correct; learnings `en`+`x-default` = correct. Favicon: 17/17 SVG+manifest; 15/17 no `/favicon.ico` (404) and no apple-touch-icon — Safari-only cosmetic, **accepted, not worth 15 rebuilds**. Results: `/tmp/vra/fleet_audit2.json`, table in ops doc §2.4 |
| **#26 hero-copy staleness** (`scripts/hero-copy-audit.py`, new) | all hardcoded star counts across 13 sites match live GitHub `stargazers_count` (art-dupl 2/2, gogenfilter 3/3, do-auditlog 5/5, typespec 15/15, …) — **zero staleness, closed as clean** |
| **#4 guard-as-CI-pilot** (gogenfilter) | `scripts/a11y-guard.sh` (in-repo adapter of the family guard, ADR-0001 sync pattern) + an `A11y guard` step at the top of the `website.yml` build job + `scripts/a11y-guard.sh` added to both `paths:` filters. YAML validated; guard verified three ways: clean on gogenfilter (0/0 — gold standard), sandbox with all three violation classes fires 3 FAIL + 1 INFO + exit 1 |
| **#5 token convergence — SOURCE COMPLETE** | all 9 INFO repos define `--color-on-accent` per theme with **per-repo contrast-measured values** (see below) and their solid `bg-accent` CTAs now use `text-on-accent` (26 component/layout files, exact-match swaps with count assertions). `family-a11y-guard.sh`: **0 FAIL / 0 INFO family-wide** (was 68 INFO). Builds+deploys deliberately deferred (load rule) — tracked in TODO_LIST |
| **#45 capture retry-once** (`cdp-shoot.py`) | a hung navigation/capture retries once before failing the site (TimeoutError only). Verified: timeout-mutant shows exactly one `retry …` line then `FAIL` + exit 1; clean run unaffected |
| **#46 dark-capture paint check** | luma-verified (ffmpeg signalstats YAVG): atomicwrite dark 25.3 vs light 229.8; learnings dark 72.9 vs light 154.6 (the `respectPrefersColorScheme` fix works end-to-end); cmdguard dark-by-design 26.9. **No code change needed** — `captureBeyondViewport` shots paint real backgrounds |
| **#44 checklist dark-mode item** | `template-family-checklist.md` Capture hygiene: dark-capture gate (media response or dark-by-design, Docusaurus key is `respectPrefersColorScheme` — `respectColorScheme` does not exist) |
| **#48 pnpm allowBuilds gotcha** | **14/14 family AGENTS.md** now document: approvals live in `pnpm-workspace.yaml` under `allowBuilds:`, `pnpm.*` in package.json is ignored, placeholder values silently disable the key (cmdguard incident). All 14 workspace files also inspected — no other placeholder rot |

**Per-repo on-accent values (contrast math, not copy-paste):** the gold
standard's dark-ink/white split is only correct when the dark accent is
bright. Measured per repo: 6/9 dark themes → `#0c0a09` ink; go-error-family
dark accent is violet-600 `#7c3aed` → **white** (dark ink would be 3.46:1);
go-atomic-write light accent emerald-600 `#059669` → **dark ink** (white
would be 3.77:1); dynamic-markdown dark accent `#6366f1` → `#000000` (the
`#0c0a09` class computes to 4.41:1 there). Light themes otherwise white.

## b) PARTIALLY DONE

1. **F82 (green cycle)** — the 00:11 pass was a pure timeout-grinder under
   load 40–52 (3× 12-min timeouts, 0 landings in 31 min; small-image probe
   18.3 s proved the stack healthy — tall pages just cannot win). Killed
   it. Replaced by **waiter v2** (`/tmp/vra/wait-and-review2.sh`, pid
   4104365): polls every 5 min, starts `visionreviewd once` when load < 20
   **and** a small-image vision probe completes < 60 s (gates on the thing
   that actually fails). v1 (load-only, 15) was replaced after a brief
   14-dip proved how quickly load bounces (14→29 in minutes).
2. **llama-server death (new incident, ops doc §2.6)** — found DEAD at
   01:07 (connection refused; was healthy at 00:48; /data mounted, memory
   fine; original log overwritten by the restart, cause unknown). Restarted
   via the README direct-path invocation: **~5.5 min to healthy** (health
   200 at 01:18), vision probe re-verified (20.5 s TTFB at load ~38). Any
   pass started in the dead window would have burned 12-min timeouts —
   both the review-fleet canary and the waiter probe now catch this class.
3. **Score table + Page-line E2E + subpage/dark triage** — all blocked on
   the cycle; nothing to measure until it runs. Baseline, diff method, and
   expected-shot inventory are in place from session II.
4. **#5 deploys** — source done (above); `pnpm build` ×9 + firebase deploy
   MUST wait for the pass (never parallel to inference); axe re-run after.

## c) TOTALLY FUCKED UP (own goals)

1. **Fired a full capture cycle by accident** while "verifying" the
   pre-flight (forgot that a passing canary *proceeds to capture*). Killed
   it mid-run (KeyboardInterrupt during a settle sleep): 24 home shots
   re-captured across 6 sites → hash changes → those views re-review next
   pass. Harmless (retry fix converges), sloppy — pre-flight verification
   needed a canary stub, not the real script.
2. **Pipeline-masked exit codes twice** in my own verification (`cmd |
   tail` reports tail's status) — caught both times because the memory rule
   says to re-check; the second `grep` failure pattern (`Unmatched [`) then
   wasted three ffmpeg invocations on a bad regex before the correct
   `metadata=print` form surfaced the data.
3. **Tried to edit a running script** (waiter v1). Bash reads scripts
   incrementally; editing under execution is a corruption hazard. Wrote v2
   to a new file, killed v1, started v2.

## d) WHAT WE SHOULD IMPROVE (systemic)

1. **Pre-flight scripts must have a dry-run mode** — `review-fleet.sh`
   needs `--preflight-only` so verifying the gates can't trigger the
   pipeline (own goal c1).
2. **The external load makes nightly automation unreliable** — load swung
   14→88 in an hour. Until Lars decides on the ollama workload (decision ①),
   the waiter's vision-probe gate is the only sane trigger; the monthly
   timer (load guard 10) may defer for days. Consider a catch-up rule:
   if no green cycle in N days, run with a reduced view budget anyway.
3. **llama-server has no supervision** — it died unattended once already.
   A user-level Restart=always unit (the repo ships `nixos/visionreviewd.nix`
   with an optional llama unit — adopt it locally) beats rediscovery.

## e) NEXT (ordered)

1. When the waiter fires and the pass exits 0 → final score table vs
   `/tmp/vra/baseline/` (same python diff as session II), close F82, check
   `**Page:**` lines in retried views (sourceURL E2E), then subpage + dark
   triage.
2. Build + deploy the 9 token-converged sites (`family-redeploy.sh`),
   axe re-run to confirm contrast live.
3. Push (Lars) — now also carries: gogenfilter CI pilot, 9 token repos,
   visionreviewd tooling (cdp-shoot guards, review-fleet pre-flight,
   site-monitor escalation, three new audit scripts), learnings-class doc
   updates.
4. learnings CI (Lars): `ci.yml` is **`disabled_manually`** in Actions
   (that's why `gh run list` shows only Dependabot) — re-enable + prove
   green, delete the stale Go/docker workflow, or leave disabled; it tests
   the Go extractor, not the website either way. TODO_LIST updated.
5. From the 50-list, still open and unblocked-by-load: #37 sitemap targets
   in the family guard (build-time check), #17 home-manager timers module,
   #16 off-machine vantage, #36 score-drop alerting (digest infrastructure
   from today is its base).

## Decisions needed from Lars (3 carried + 1 new)

1. **External workload** (unchanged, now with data: load 14–88 overnight,
   GPU pinned by ollama qwen2.5vl:3b) — a 1 h pause window finishes the
   fleet cycle ~10× faster and unblocks F91–F94 calibration.
2. **Score methodology** (unchanged): full-page monthly vs viewport-pinned
   monthly + full-page quarterly.
3. **Push policy** (scope grown — see e.3).
4. **learnings CI fate** (new — see e.4).

## Current machine state (01:23)

- **Waiter v2 armed** (pid 4104365, log `/tmp/vra/v4-wait.log`): will start
  the review pass autonomously when load < 20 AND vision probe < 60 s; the
  pass resumes cleanly (NeedsReview fix) and closes F82 on exit 0.
- **llama-server**: restarted 01:12, healthy (health 200), pid 4122419,
  log `/tmp/vra/llama-server.log`; ~5.5 min cold load from /data.
- `visionreviewd` binary fresh (23:51 build; no sources newer — the exact
  check review-fleet.sh performs).
- v3 grind evidence preserved at `/tmp/vra/v3-once.log.grind-evidence`
  (three 12-min timeouts, zero landings).
- Captures in place: 68 home (some re-captured 00:43 by own goal c1) +
  60 subpage; review timeout 12 m; sourceURLs configured.
- All repos clean (auto-daemon commits as we go); nothing pushed.
- Site-monitor: hourly, verified; escalation clock started for both
  KNOWN-broken domains (will first fire at 7 days continuous KNOWN).
