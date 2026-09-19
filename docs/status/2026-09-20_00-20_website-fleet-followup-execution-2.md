# Status — Website-Fleet Follow-Up Execution II (resumed session)

- **Date:** 2026-09-20 00:20 CEST
- **Session scope:** resume `docs/planning/2026-09-19_15-33_website-vision-review-followup-pareto-plan.md`
  from the 20:14 status report; execute the "Next Steps" list end to end.
- **Verdict up front:** all 17 sites are now **axe-CLEAN** (24 violation
  kinds → 0, live-verified), the review pipeline gained true full-page +
  dark-mode capture, visionreviewd got its first real bug fix from field
  data (skip-seen swallowed failed reviews), and the family got an
  ADR-backed consolidation decision. The **post-fix fleet review cycle is
  still running** (llama is slow under an external load of 30–80); its
  score table is partially extracted below.

## Answers to the three questions first

### What did I forget?

1. **Binary freshness.** The 23:32 fleet cycle ran on a `visionreviewd`
   binary built **2026-09-07** — it had neither `sourceURL` support nor the
   skip-seen fix, and I noticed only when the `**Page:**` line failed to
   appear in fresh reviews. 40 minutes of reviews landed without
   provenance. Check `stat` on the binary before any long run.
2. **The kill side-effect.** Killing the v1 pass mid-request left
   llama-server chewing an orphan prompt; combined with the machine's
   external load (45–80) this produced a string of 5-minute timeouts.
3. **`pnpm approve-builds` garbage was already committed.** cmdguard's
   `pnpm-workspace.yaml` carried the literal placeholder
   `esbuild: set this to true or false` — pnpm silently ignored the whole
   key. The prior session's "fix" was never actually a fix.
4. **Timeout must scale with capture methodology.** Full-page screenshots
   blow up the vision prompt token count; the 5-minute per-view timeout
   that was fine for 4320 px shots times out on 6252–10162 px ones. Raised
   the fleet config to 12 minutes.
5. **Dark-mode shots are new ground.** Two of 17 sites don't respond to
   `prefers-color-scheme` at all — cmdguard is dark-by-design (manual
   toggle only), learnings shipped `respectColorScheme` (a key that does
   not exist in Docusaurus 3.10) and simply never followed the OS setting.

### What could I have done better?

1. **Coverage-diff, third strike.** My `--subpages` run reported
   `DONE: 0 failures` while capturing only the *last* subpage per site
   (28 files instead of 56) — a multiedit had dedented the wait/shot block
   out of the pages loop. I caught it by counting files against
   expectations, which is the exact lesson from the typespec `src/`
   shadowing incident. The check must be in the script: next iteration
   prints the expected-per-site shot count and fails short.
2. **Serialize heavy work.** I ran four site builds in parallel *with* the
   review pass on a box already at load 45+. The builds finished, but they
   contributed to llama timeouts that killed the cycle's first attempt.
3. **Batch the deploys.** learnings got deployed three times (manifest,
   og:image, dark-mode) and cmdguard-family sites one fix at a time. One
   rebuild+deploy per site after the fix round would have halved it.
4. **Verify config keys against the installed package, not memory.**
   `respectColorScheme` failed the Docusaurus schema validation;
   `respectPrefersColorScheme` is the real key in 3.10.2.
5. **Unbuffered logs for long runners.** Piping the restarted review pass
   through `tail -2` made a healthy process look hung (output invisible
   until exit) and triggered a needless restart. `nohup … > file` from the
   start.

### What could I still improve?

1. **Score comparability needs pinned capture methodology.** This cycle
   switched captures to full-page in the same pass that measured the a11y
   fixes, so fix-effects and methodology-effects are confounded (mobile
   scores dropped 2–3 points on long pages; the reviewer literally says
   "text is too small" — an artifact of the model seeing an 8666 px page).
   The monthly cycle should either stay on one capture mode or keep two
   explicit baselines. Needs a Lars decision (see questions for Lars).
2. **The review stack needs an SLA, not just retries.** Under external
   load the 17-site cycle stretched from ~35 min to hours. A pre-cycle
   probe (one canary review; if it exceeds N minutes, defer the cycle)
   would stop the timer from burning 5-minute timeouts all night.
3. **Provenance coverage is one monthly cycle behind.** Reviews completed
   before the sourceURL-enabled binary lack the `**Page:**` line; the next
   full re-capture cycle (new hashes) fills them all in.

## a) FULLY DONE (verified)

### Pipeline & tooling

| Task | Evidence |
| --- | --- |
| `vision-stack-up.sh` wait 120→900 s + reload-time logging | `scripts/vision-stack-up.sh`; rationale + measured ~13 min in `docs/activation/site-fleet-ops.md` §1.4 |
| CDP capture tool (new) | `scripts/cdp-shoot.py`: stdlib websocket client (nix `websockets` package envs proved broken), `captureBeyondViewport` full-page, `prefers-color-scheme=dark` emulation, `--subpages`/`--skip-home`/`--viewport`/`--mode`; 68 home shots (light+dark × desktop+mobile, docH 6252 px class) + 60 subpage shots, all verified |
| `review-fleet.sh` upgraded | CDP captures first, tall-viewport `shoot-sites.sh` as fallback path |
| site-monitor verified in production | 23:00 hourly cycle: 17×OK, 2×KNOWN correctly muted, no false alerts |
| sourceURLs config | 17 entries in `~/.config/visionreviewd/websites.json` + repo copy synced; timeout raised 5m→12m for full-page shots |
| activation README | new `sourceURLs` section with rendered example; fixed broken relative link to `visionreviewd-systemnix.md` |

### visionreviewd (the repo)

| Task | Evidence |
| --- | --- |
| sourceURL feature finished (F76–F79) | code was already committed by the daemon; config example + README docs + repo-copy sync landed this session |
| **skip-seen bug fix** | `internal/reviewd/pipeline.go`: skip condition now requires `!state.NeedsReview()` — a view whose review failed (transient model outage) is retried on the next pass instead of being skipped forever. Field-triggered: the aborted 20:01 cycle had permanently unreviewable views. BDD regression spec `when the view's review failed in a previous pass`; full `internal/reviewd` suite green; binary rebuilt |
| HARVEST (F32) | `TODO_LIST.md`: three new fleet sections + Blocked-on-Lars section; completed items removed; `ROADMAP` untouched (demo epic went to typespec repo) |

### Fleet quality (Wave 3/4 residue)

| Task | Evidence |
| --- | --- |
| **axe re-run: 17/17 CLEAN** | round 1: 24 kinds → 4; round 2 fixes below → 0 violations of any severity on all home pages (desktop, light) |
| atomicwrite `tabindex="0"` | `.min-w-0` install `<code>`; rebuilt+redeployed |
| gogenfilter theme-aware amber | `text-[#fbbf24]` → `text-amber` token; light `--color-amber` `#b45309`→`#92400e` (axe-measured 4.19:1 → ~5.9:1 on its own tint); redeployed, CLEAN |
| typespec `--color-accent-text` token | new token family (`#0f766e` light) for accent text; + same amber bump; redeployed, CLEAN |
| learnings `.button--secondary` | white/blue override (axe-measured dark-on-darkgreen 2.5:1 before); redeployed, CLEAN |
| cmdguard pnpm v11 fix | `pnpm-workspace.yaml` placeholder replaced with `allowBuilds: esbuild: true`; postinstall + `astro build` verified green; gotcha documented in cmdguard `AGENTS.md` + template-family checklist |
| learnings: algolia placeholder leak | link audit caught `YOUR_APP_ID-dsn.algolia.net` loading from prod pages; placeholder config removed, leak verified gone from build and live site |
| learnings: manifest + theme-color | `static/manifest.webmanifest` + `headTags` link + two media-scoped `theme-color` metas; live-verified (200s, meta present) |
| learnings: real og:image | 1200×630 chromium card shot → `static/og/home.png` + `themeConfig.image`; live `og:image` verified |
| learnings: dark mode | `respectPrefersColorScheme: true` (first attempt used a nonexistent key — schema error — then corrected) |
| demo posters → webp + dimensions (F101) | do-auditlog: `demo-poster.webp` (69 KB→26 KB) + `width/height` on `<video>` + JSON-LD `thumbnailUrl`; emeet-pixyd: same (205 KB→27 KB); both live-verified |
| custom-404 audit (F99) | all 17 sites return real 404s on a random path — no soft-404s, no Firebase default pages; audit script `scripts/cdp-shoot.py`-independent, results in `/tmp/vra/audit404_meta.json`, transcribed: CLEAN |
| favicon/manifest/theme-color (F100) | 16/17 were complete; learnings gap closed above |
| subpage enumeration (F66) + captures (F67) | sitemap-derived: 15 sites have getting-started docs; cmdguard + typespec are single-page (their sitemaps point at the KNOWN-broken custom domains — noted); 60 subpage captures (light+dark × desktop) |

### Family consolidation (F83–F86)

| Task | Evidence |
| --- | --- |
| divergence report (F83) | `docs/activation/template-family-divergence.md`: layouts mean pairwise similarity **0.875** (max 0.955), heroes 0.546 (per-product by design), core-token overlap only 5/14 repos |
| ADR (F84) | `docs/adr/0001-template-family-sync-over-extract.md`: **sync-script over extract-package** — version skew across 14 repos would be a worse failure mode than drift; the divergent part is the product surface |
| prototype (F85) | `scripts/family-a11y-guard.sh`: check-mode guard for the three violation classes; current family state **0 FAIL / 68 INFO** (INFO = `bg-accent` CTAs in repos without the `--color-on-accent` scheme — convergence backlog, explicitly not violations) |
| decision doc (F86) | covered by the ADR's Consequences section; guard wired as the pilot CI candidate |

### Score impact (partial — cycle still running)

Per-view diff, baseline snapshot (22:47, clean tall-viewport captures) vs
reviews landed this cycle (23:32+, full-page captures):

- **20 views unchanged** (±0, inside the established ±1 noise band)
- **8 up** (go-output desktop 7→8, errorfamily mobile 6→8, cleanwizard,
  learnings 5→6, …)
- **12 down** — of which the three big drops (gogenfilter 8→5,
  md-go-validator 7→4, art-dupl 6→4) are all **mobile full-page** views;
  the reviewer's stated reasons ("text is too small", "missing spacing")
  match the methodology artifact, not the fixes
- **7 new dark views** (no baseline; first data point recorded)

Honest read: desktop scores are stable-to-slightly-up with the a11y fixes
in; mobile full-page numbers are **not comparable** to the viewport-era
baseline (see question 2 for Lars).

## b) PARTIALLY DONE

1. **F82 (one full green cycle)** — the cycle is running right now on the
   12-minute timeout (started 00:11); under external load 30–80 each view
   takes 1–3+ min (vs ~30 s on an idle box). 35 reviews had already landed
   from the earlier attempts; skip-seen (with the retry fix) resumes
   cleanly. Not yet closed: the pass must finish with exit 0 once.
2. **Page-line E2E proof** — the scratch-dir capture recorded `sourceURL`
   in the event (new binary), but its review timed out twice under load.
   Test-proven at unit level; one green review renders the `**Page:**`
   line and closes it. The running cycle will produce these for all
   retried views anyway.
3. **Dark triage (F65)** — captures done; static triage done (15/17
   respond; cmdguard dark-by-design, learnings fixed); model-based triage
   of dark views lands with the running cycle.
4. **Subpage review (F68–F69)** — 60 captures in place; reviews happen on
   the next `once` pass (new hashes → automatic).

## c) NOT STARTED (deliberately)

- **F91–F94 calibration** — needs a non-saturated machine (GPU is 100 %
  busy with an external ollama qwen2.5vl:3b at 91 % VRAM; system load
  30–80 all session). CPU baseline exists; deferring was the right call.
- **F29 GPU benchmark** — same blocker.
- **emeet online shots (F70–F71)** — PIXY hardware still unwired.
- **Push of local commits** — vision-review-agent (sourceURL feature +
  skip-seen fix + tooling), learnings, family sites: all committed locally,
  none pushed (policy is Lars's).
- **learnings manual CI run; CSP report re-check; auditlog canonical
  audit; hero-copy staleness; templ-components cmd/site vet gate decision;
  home-manager timers module; off-machine monitor vantage; KNOWN_BROKEN
  escalation** — all harvested into `vision-review-agent/TODO_LIST.md`
  with context.

## d) TOTALLY FUCKED UP (own goals, ranked)

1. **Ran the flagship cycle on a stale binary** (2026-09-07 build): no
   sourceURL, no skip-retry. Every later symptom (missing `**Page:**`
   lines, permanently skipped timeout views) traces to this. The fix that
   came out of it (skip-seen retry) is genuinely valuable — but it was
   discoverable 40 minutes earlier by `stat`ing the binary.
2. **Subpage loop indentation bug** shipped with a `0 failures` banner —
   the same found-but-unfixed class as typespec's `src/` shadowing. Caught
   by file-count diffing. The guard script now exists for a11y; the shoot
   script needs the same expected-vs-actual assertion built in.
3. **Shipped a nonexistent config key** (`respectColorScheme`) instead of
   checking the installed Docusaurus schema (`respectPrefersColorScheme`).
4. **Three restarts of the review pass** (EOF → stale binary → timeout
   size). Each was individually reasonable; together they cost ~1 h and
   made the score table's provenance messy (three binary generations'
   worth of reviews in one INDEX).
5. **Parallel builds during model work** on a loaded box — caused the very
   EOF the first restart was reacting to.

## e) WHAT WE SHOULD IMPROVE (systemic)

1. **Pre-flight probe before cycles:** binary mtime, one canary review
   under N seconds, then commit to the full pass.
2. **Every capture script asserts its own shot inventory** (expected
   per-site count; exit non-zero on shortfall). Third incident of this
   class; it's now a rule, not a habit.
3. **Serialize builds vs model inference** — one heavy pipeline at a time
   on this box; the monthly timer should refuse to start if load > 10.
4. **Timeouts scale with capture surface** — full-page shots need ≥2× the
   viewport-era review timeout (config now 12m).
5. **Score baselines pin capture methodology**; switching modes means a
   new baseline, and trend claims only compare like with like.
6. **Site deploys batch per fix-round**; the ADC path is proven and
   scripted (`family-redeploy.sh`), use it wholesale.
7. **Config keys get validated against the installed package** (build once
   before deploy would have caught it — which it did, after a wasted
   deploy cycle).

## f) NEXT 50 (ordered by leverage)

1. Let the running cycle finish → final score table + F82 closure (it
   resumes cleanly thanks to the retry fix; log: `/tmp/vra/v3-once.log`)
2. Re-verify score table per capture mode; write the trend into the fleet
   log (INDEX diff generation is TODO_LIST #5)
3. Decide score methodology (Lars): full-page monthly vs viewport-pinned
   monthly + full-page quarterly (see question 2)
4. Wire `family-a11y-guard.sh` as CI check in one pilot family repo
5. Adopt `--color-on-accent` token scheme in the 9 INFO repos (68 INFO
   findings; not violations, convergence debt)
6. Re-review the 60 subpage captures (automatic next pass) + triage (F68–69)
7. Dark-view triage once the cycle lands them (F65 tail)
8. Add expected-shot-count assertion to `cdp-shoot.py` (close the d2 class)
9. Pre-flight probe (canary review) into `review-fleet.sh` + load guard
10. learnings: one manual CI run to prove Actions green post-retarget
11. CSP report re-check for og/meta additions (report-only CSPs)
12. auditlog canonical-URL audit (parity with the other 16)
13. Push decision (Lars) → then push vision-review-agent, learnings, sites
14. Release/tag visionreviewd with sourceURL + skip-retry fix (needs push)
15. KNOWN_BROKEN escalation timer (7-day alert) in `site-monitor.sh`
16. Off-machine monitor vantage (uptime push)
17. home-manager module for both timers (canonical install)
18. `site-monitor.sh --json` waybar widget
19. Score-trend digest appended to fleet-review log
20. GPU benchmark rocm vs CPU (needs free GPU window)
21. Calibration F91–F94 in a quiet window (variance, 3b-vs-8B, prompt)
22. Identify owner of the ollama qwen2.5vl:3b GPU workload (Lars)
23. `/data` SMART + replace-vs-migrate decision (Lars, sudo) — VL model
    selective migration kills the 13-min reload problem
24. cmdguard console attach (Lars) → custom domain → re-shoot/re-review
25. typespec Namecheap CNAME + attach (Lars) → same; also fixes their
    sitemap/canonical URLs which point at the custom domain
26. Hero-copy staleness pass (counts/versions vs repo state)
27. learnings docs-subpage heading-order cleanups
28. templ-components cmd/site vet gate (bump vs guard decision)
29. emeet online screenshots (PIXY hardware)
30. typespec `website/video/` cruft cleanup
31. Mobile subpage captures (desktop done) — extend `--subpages` to mobile
32. Interactive-states a11y pass (focus traps, menus) — audit extension
33. Per-repo axe CI gate (pilot: gogenfilter as 8/8 gold standard)
34. learnings search: real DocSearch/alternative or drop the UI stub
35. Poster files: audit remaining video posters family-wide (only 2 sites
    have video today)
36. Score-trend alerting (alert when a site drops >2 between cycles)
37. Capture sitemap health (both KNOWN-broken domains' sitemaps break
    consumers — validate sitemap targets in the family guard)
38. `family-redeploy.sh`: add learnings/templcomponents/emeet special
    flows so one script covers 17 sites
39. Alternates/`hreflang` audit (single-locale sites — likely no-op, close)
40. Favicon consistency: SVG-only favicons vs ICO fallback audit
41. og:image per-subpage (currently home-only) — low priority
42. Journal replay drill (prove the replay path after the fix)
43. Emit fleet cycle duration + canary time into the log for capacity
    planning
44. Checklist doc: add "dark-mode captures" item (done for cmdguard class,
    keep for new repos)
45. `cdp-shoot.py`: retry-once on per-view timeout instead of failing the
    site
46. Consider `--omit-background=false` guard for dark captures (verify
    page actually paints, not just media emulation)
47. Move `/tmp/vra/` audit scripts into `scripts/` with tests (linkcheck,
    404/meta audit)
48. Document the pnpm allowBuilds gotcha in remaining family AGENTS.md
    files (cmdguard done; it will hit the next pnpm v11 repo)
49. MONTHLY: confirm the 00:xx cycle output wrote scores to the log
    (first data point for trend alerting)
50. typespec demo.mp4 production (ROADMAP epic; storyboard seeded)

## Decisions needed from Lars

1. **The external workload:** something is pinning this box at load
   30–80 with 100 % GPU (ollama qwen2.5vl:3b, port 45113, custom
   template, `--image-min-tokens 1024` — not started by me). Expected?
   If it can pause for ~1 h, the fleet cycle finishes ~10× faster and the
   calibration work (F91–F94) unblocks.
2. **Score methodology:** keep monthly reviews on full-page captures
   (richer, but slower, noisier on mobile, and a new baseline vs today's
   table), or pin monthly to tall-viewport captures for trend continuity
   and run full-page as a separate quarterly deep pass?
3. **Push policy:** vision-review-agent carries the sourceURL feature +
   the skip-seen retry fix; learnings and the family sites carry today's
   a11y/SEO fixes. Push to origin now? (Releases/tags follow from that.)

## Key numbers

| Metric | Value |
| --- | --- |
| axe violations across fleet | 24 kinds → 4 → **0** (all 17 home pages CLEAN) |
| Link audit round 2 | 324 URLs, 2 KNOWN-broken, 1 npm-bot 403, 1 real (algolia leak, fixed) |
| Full-page captures | 68 home (light+dark × desktop+mobile) + 60 subpage shots |
| Doc heights captured | up to 10162 px (learnings docs page) vs 4320 px viewport-era |
| Skip-seen bug | views with failed reviews were skipped forever; now retried (BDD-pinned) |
| Family similarity | layouts 0.875 mean pairwise, heroes 0.546, tokens 5/14 complete |
| Guard result | 0 FAIL / 68 INFO across 14 family repos |
| Poster savings | do-auditlog 69→26 KB, emeet 205→27 KB (webp) |
| Fleet review timeout | 5m → 12m per view (full-page prompt growth) |
| System load during cycle | 30–80 (external), vs ~2 idle → review ~4× slower |

## Current machine state (for the next session)

- Fleet review pass: **RUNNING** (`nohup` log `/tmp/vra/v3-once.log`, pid
  alive at writing; resumes/skips correctly across restarts thanks to the
  NeedsReview fix). When it exits 0: extract final score table → F82 done.
- llama-server: healthy on :8390 (pid 2030545), saturated by load — expect
  ~1–3 min/view until the external workload stops.
- Timers: site-monitor (hourly, verified firing 23:00) + fleet-review
  (monthly) enabled.
- Subpage captures: 60 shots in place; reviewed automatically next pass.
- All repos: working trees clean (auto-daemon); commits local-only.
- Blocked on Lars: console/DNS domains, sudo SMART, ollama ownership,
  push policy, score-methodology decision (this report).
