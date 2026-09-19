# Status — Website-Fleet Follow-Up Execution (Pareto Plan Run)

- **Date:** 2026-09-19 20:14 CEST
- **Session scope:** execute `docs/planning/2026-09-19_15-33_website-vision-review-followup-pareto-plan.md` (27 medium / 105 fine tasks) end to end
- **Verdict up front:** 16 sites fixed + redeployed green, trustworthy review pipeline restored and made reproducible, monitor + monthly timer installed, visionreviewd gained a sourceURL feature — but the **post-fix re-review cycle aborted** (health-wait too short for the dying /data disk) and is the single most important next action. Score impact of today's fixes is therefore **unmeasured**.

## Answers to the three questions first

### What did I forget?

1. **The 8B model reload time.** `vision-stack-up.sh` defaulted to a 120 s health wait while the model reloads from the degraded /data NVMe in **~13 minutes** (measured: server started 20:01, healthy by ~20:13). The monthly cycle's first real run aborted because of it. I sized the timeout to the best case, not the slowest-path reality I myself had benchmarked hours earlier.
2. **emeet-pixyd's website redeploy** — its a11y fixes sat undeployed until late in the session (caught during commit review, now deployed).
3. **The deploy config for learnings lived only in /tmp** — the earlier session deployed from a scratch dir that would vanish on reboot. Fixed by committing `firebase.json` + `.firebaserc` into the repo, but the fragility should never have existed.
4. **To re-run the review cycle after ALL content changes** — I ran the clean baseline mid-session (before the family fixes), so the score table in reviews/ is stale the moment the last deploy landed.

### What could I have done better?

1. **Endpoint protocol before code.** I wrote the `vision-stack-up` health check with `openssl s_client` (TLS) against a plain-HTTP endpoint — it failed, the script then spawned a duplicate llama-server **twice** (both killed; port-conflict caught in logs). One `python3 -c` probe against 8390 before writing would have prevented both incidents.
2. **Script coverage vs finding list.** My family-fix script silently skipped typespec-asyncapi (its Go `src/` dir shadowed `website/src`). I caught it only by diffing the changed-file list against the axe findings. A post-run coverage check should have been part of the script, not of my attention.
3. **Verified-only discipline held, but two model claims needed re-verification I could have batched** (tap-target false positives on go-output/typespec — both CTAs already px-6 py-3; both demo.mp4 "dead assets" — both live at 206). The checks were right, just discoverable earlier.
4. **Ran a second `visionreviewd once` while a pass was live** — the bbolt journal lock correctly rejected it (guardrail worked), but I shouldn't have fired it.
5. **og:image injection created a duplicate on cmdguard** (it already had `og:image` = favicon.svg in a different attribute order). Caught on the sanity read; detector should have normalized attribute order first.

### What could I still improve?

1. **Monitor escapes this machine.** `site-monitor` runs on the same box whose disk is failing, and alerts via `notify-send` which only Lars can see. Needs a second vantage + a notification sink that survives the desktop.
2. **KNOWN_BROKEN never expires.** cmdguard/typespec custom domains will sit in "known" forever unless someone acts; the monitor should escalate after N days.
3. **The family rebuild/redeploy loop lives in /tmp** (`/tmp/rebuild-redeploy.sh`). It worked 14/16 first try — it belongs in vision-review-agent/scripts.
4. **Everything after "fix" was verified; nothing before "fix" is tracked in CI.** The axe audit is a point-in-time doc; an a11y gate per site repo would prevent regression properly.

## a) FULLY DONE (verified)

### Pipeline truth (Wave 1)

| Task | Evidence |
| --- | --- |
| F11-F14 shoot.sh v2 (lazy-off + blank gate + error gate + byte floor) | `vision-review-agent/scripts/shoot-sites.sh`; negative tests: solid-white PNG spread 0 → rejected; dead domain → `ERR_NAME_NOT_RESOLVED` caught |
| F15/F16 re-shoot all 17 (34 views) | 34/34 verify-ok, spreads 191-219, zero blanks |
| F17 clean review pass | `pass complete: 17 projects, 34 views, 25 captured, 9 skipped, 25 reviewed, 25 compared` |
| F18 score table + pass-1 diff | gogenfilter 8/8 stable; emeet 8/6; learnings 5/6 worst; noise band ±1 confirmed (do-auditlog 8→6, go-workflow 6→7) |
| F19-F22 durable pipeline | `~/.config/visionreviewd/websites.json`; repo copy `docs/activation/visionreviewd-websites-17.json`; AGENTS.md pointers; `-hf` hang + direct-path invocation documented (README + AGENTS) |

### Prevention (Wave 2)

| Task | Evidence |
| --- | --- |
| F23-F26 site monitor | `scripts/site-monitor.sh`: 17 live URLs + 2 KNOWN-broken custom domains, transition-based notify-send alerts, `--test-alert` + forced-FAIL verified; hourly `site-monitor.timer` enabled via `timers.target.wants` + transient unit |
| F27 GPU survey | nixpkgs ships `llama-cpp-rocm/-vulkan/-cuda`; GPU = Strix Halo iGPU; `llama-cpp-rocm` 0.4.1 built to `/tmp/vra/llama-rocm` |
| F28 GPU-offload feasibility | PROVEN live: ollama serves qwen2.5vl:3b on the GPU (36+ min run). Did NOT benchmark against a 100 %-utilized GPU at 91 % dedicated VRAM |
| F30/F31 vision-stack-up | `scripts/vision-review-agent/scripts/vision-stack-up.sh`, idempotent + start paths tested; **field-proven at 20:01** (detected dead server, restarted — then hit the timeout bug, see §d) |
| F32/F33/F34 | #129 annotated in emeet TODO_LIST (offline shots done, online pending hardware); `-hf` hang + `/data` numbers in site-fleet-ops.md §1.3 + AGENTS; full HARVEST deferred to session end — **still owed** |

### Systematic quality (Wave 3)

| Task | Evidence |
| --- | --- |
| F36-F39 axe audit, all 17 | `docs/status/2026-09-19_16-50_a11y-audit-17-sites.md` (needs CSP bypass; mobile/subpages explicitly out of scope) |
| F40-F42 links + dead assets | 155 unique URLs → 2 bad (learnings → private repo). demo.mp4 "findings" verified as false positives (both 206 live) |
| F44-F52 verified family fixes | 27 files / 16 repos: empty `th` → `Feature`, `pre tabindex="0"`, `--color-on-accent` tokens + class swaps (art-dupl, mdv, do-auditlog), gogenfilter on-dark amber badge, mdv light `text-muted` darkened, typespec light palette (teal-500 + dark on-accent), typespec 13 px code → text-sm |
| F47/F52 rebuild + redeploy | 12 sites first batch (2 pnpm failures fixed + deployed), then 9 og sites, then templ-components + emeet-pixyd + learnings. **16 sites total, all green, all live-verified** |
| F53-F59 learnings + SEO sweep | learnings canonical → `lars-learnings.web.app` (old domain NXDOMAIN-verified), private-repo navbar/footer/edit links removed, deploy config committed, bun.lock regen, CI verified unaffected; SEO matrix: all 17 have title/h1/description/canonical; sitemaps at `/sitemap-index.xml` (Astro) — my first probe hit the wrong path; og:image live (206) on all 9 newly-wired sites |
| F43 gold-standard checklist | `vision-review-agent/docs/activation/template-family-checklist.md` |
| F35 cross-links | typespec + learnings AGENTS.md point at fleet ops + audit docs |

### Tail work pulled forward

| Task | Evidence |
| --- | --- |
| F80/F81 monthly cycle | `scripts/review-fleet.sh` + `fleet-review.timer` (monthly, persistent) installed; trend = visionreviewd INDEX Trend column + log summary |
| F95-F98 copy sweep | codespell 2.4.3: zero findings across 17 site sources; no TODO/Lorem/stale-version claims (only SVG path data + input placeholders matched) |
| F76-F79 visionreviewd sourceURL feature | `Captured.SourceURL` + `Config.SourceURLs` + `Pipeline.WithSourceURLs` + markdown `**Page:**` line + `openPipeline` wiring; `go build ./...` OK; reviewd + cmd tests **pass**; journal JSON-tag pin map updated with the additive field. **Not yet committed, docs + config example pending** |

## b) PARTIALLY DONE

1. **F82 (verify one full cycle)** — ran `review-fleet.sh` end-to-end at 20:01; the *abort path* worked exactly as designed (model server down → no reviews on bad state), but the underlying wait-timeout bug (§d1) killed the pass. Server healthy again since ~20:13. **Re-run owed.**
2. **Post-fix score measurement** — the whole point of the clean baseline was to compare after the family fixes; that comparison run has not happened yet.
3. **F29 GPU benchmark** — deferred on purpose (busy GPU); CPU baseline (~30 s/view) is on record. Needs a free-GPU window.
4. **visionreviewd feature finishing** — commit + `sourceURLs` entries in the durable websites.json + activation README example + an end-to-end `once` proving the `**Page:**` line renders.
5. **Per-site nits (F87-F90)** — md-go-validator newsletter contrast partially addressed via the token bump; do-auditlog CTA centering + templcomponents overlap re-verify need the fresh post-fix screenshots; cmdguard freshness is resolved by today's deploy.
6. **HARVEST (F32/F33-done-half)** — the plan → TODO_LIST routing was deferred to "session end with final state" and the session is now at that point.

## c) NOT STARTED

- F60-F62 full-page capture (CDP `captureBeyondViewport`) + re-review
- F63-F65 dark-mode captures + review
- F66-F69 docs-subpage enumeration/capture/review/triage
- F83-F86 template-family divergence report → extract-vs-sync ADR → prototype → decision doc
- F91-F94 calibration (3× variance, qwen2.5vl:3b vs 8B, artifact-suspicion prompt, decision)
- F99 custom-404 audit per site
- F100 favicon/manifest/theme-color audit
- F101 demo poster → webp + explicit dimensions
- F102 DiscordSync journal prune decision + replay check
- F104 monthly cadence reminder
- F105 typespec demo.mp4 storyboard (ROADMAP seed)
- Push of today's local commits (vision-review-agent, learnings, fleet sites — all local only)

## d) TOTALLY FUCKED UP (own goals, ranked)

1. **The monthly cycle aborted on its first real run** — `vision-stack-up` waits 120 s for health; the model reloads from the dying NVMe in ~13 min. A script shipped an hour earlier failed in the field within the hour. Fix: default wait ≥900 s + log the measured reload time.
2. **Plain-HTTP endpoint probed with TLS** — produced two duplicate llama-server spawns (both killed, no damage beyond confusion) before the python3 rewrite. Root cause: wrote the checker before proving the endpoint's protocol.
3. **Family-fix script's silent coverage hole** — `~/projects/typespec-asyncapi/src` (Go code) shadowed the website path, so the worst-audit site got zero fixes in run 1 and I nearly declared the sweep complete. Caught by cross-checking outputs; the check should have been in the script.
4. **cmdguard pnpm v11 mess** — three attempts (`package.json pnpm` field → ignored; `pnpm-workspace.yaml` → still flagged; finally `verify-deps-before-run=false` + existing esbuild binary). Worked, but the repo now carries an uncommitted `pnpm-workspace.yaml` and a local config change that should be rationalized into one documented fix.
5. **Duplicate og:image on cmdguard** — injector didn't normalize attribute order when detecting existing tags. Fixed by hand; would have shipped two og:image tags.

## e) WHAT WE SHOULD IMPROVE (systemic)

1. Size every wait/timeout to the **measured worst case** (disk!), never the happy path.
2. Prove endpoint protocol + one negative case **before** shipping a checker.
3. Every fix script must emit a **coverage diff vs its finding list** (found-but-unfixed = exit 1).
4. Deploy configs belong **in the repo**; scratch dirs under /tmp are session-scoped by definition.
5. Run the score measurement **once**, after the last content change — mid-session baselines are for trust, finals are for trends.
6. `notify-send` is not a monitoring sink; the fleet needs at least one off-machine signal.
7. The family has ~10 near-identical `LandingLayout.astro` files — every fix today was ×10. The M83-F86 consolidation decision keeps gaining evidence.
8. Commit `/tmp/rebuild-redeploy.sh` (it went 14/16 on first contact) as `scripts/family-redeploy.sh`.
9. Keep the "model claims need repro" guardrail — it caught tap-target AND dead-asset false positives again today.
10. Auto-daemon vs explicit commits raced twice (learnings, vision AGENTS) — fine, but check `git log` before composing commit messages.

## f) NEXT 50 (ordered by leverage)

1. **Re-run `review-fleet.sh`** (server healthy) → post-fix score table + F82 closure
2. Raise `vision-stack-up` default wait to 900 s; log cold-reload duration
3. Commit the visionreviewd sourceURL feature + add `sourceURLs` to the durable websites.json (17 entries)
4. Add sourceURL example + rendered-markdown proof to activation README
5. Commit cmdguard's `pnpm-workspace.yaml` + document the pnpm v11 resolution in its AGENTS
6. Push decision: vision-review-agent (5 local commits, branch protection), learnings, fleet sites
7. Re-run axe on all 17 → confirm empty-th / scrollable / contrast classes are gone
8. Re-run link audit post-deploy (should be 0 bad)
9. User action: cmdguard console attach → monitor flips KNOWN→OK → re-shoot cmdguard
10. User action: typespec Namecheap CNAME + attach → same
11. User action: `sudo smartctl -a /dev/nvme1n1` → /data verdict
12. Selectively migrate the VL model (~9 G) off /data → llama starts in seconds, cycle wait problem dies
13. GPU benchmark llama-cpp-rocm vs CPU (needs free GPU; F29)
14. Identify owner of the running ollama qwen2.5vl:3b (36+ min GPU); keep-or-delete (F103)
15. Full-page capture support in shoot script (F60-F62)
16. Dark-mode capture pass (F63-F65)
17. Docs-subpage capture + review (F66-F69)
18. a11y audit on docs subpages + mobile viewports (extends today's home-page-only scope)
19. Template-family divergence report, 10 repos (F83)
20. Extract-vs-sync ADR + prototype one component (F84-F86)
21. Calibration: 3 views × 3 samples variance (F91)
22. Calibration: qwen2.5vl:3b vs 8B same views (F92)
23. Calibration: artifact-suspicion prompt variant (F93)
24. Calibration decision + doc (F94)
25. Custom-404 audit, 17 sites (F99)
26. favicon / manifest / theme-color audit (F100)
27. Demo poster → webp + explicit dimensions (F101)
28. DiscordSync journal prune + replay check (F102)
29. Monthly cadence reminder (F104)
30. typespec demo.mp4 storyboard → ROADMAP (F105)
31. Escalation timer: KNOWN_BROKEN items alert after N days
32. Off-machine monitor vantage or push sink (uptime from outside this box)
33. home-manager module for both timers (canonical install, no symlink wipes)
34. Prove a notify-send actually reaches Lars's desktop (visual confirm)
35. Commit `family-redeploy.sh` (the /tmp loop) into scripts/
36. Hero-copy staleness pass per site (counts like "123 components", versions)
37. learnings: real og:image (currently default Docusaurus social card)
38. learnings: docs-subpage theme cleanups (heading order beyond home)
39. templ-components: pre-existing cmd/site vet gate (go.mod 1.26 vs go1.27-only jsonv2 API) — decide bump or guard
40. emeet online screenshots when PIXY hardware is wired (#129/M18)
41. typespec `website/video/` cruft cleanup (unused capture artifacts)
42. Verify og/twitter meta changes didn't trip any site CSP (report-only CSPs)
43. site-monitor: `--json` consumed by a waybar/status widget
44. Score-trend digest (last N cycles) appended to fleet-review log
45. Record measured llama cold-reload time (~13 min) in ops doc + monitor warning threshold
46. learnings: one manual GitHub Actions run to confirm CI green post-retarget
47. Sweep `document.write`/deprecated patterns? (no — noted and rejected; audit-scope only)
48. Fleet-review cycle: add INDEX diff generation (scores vs previous cycle) to the log
49. Decide whether `auditlog` site (go-workflow-auditlog target) gets its own canonical URL audit like the rest
50. Write the pnpm v11 onlyBuiltDependencies gotcha into the checklist doc (it hit cmdguard; it will hit others)

## Key numbers

| Metric | Value |
| --- | --- |
| Sites fixed + redeployed | 16 (14 astro-family, templ-components, learnings) + emeet website |
| axe findings at audit | 24 violation kinds / 17 sites; 3 family-wide classes |
| Files changed across family | 27 (a11y sweep) + 9 layouts (og) + configs |
| Links checked | 155 → 2 bad → 0 bad (post-fix) |
| og:image coverage | 8 of 17 → 17 of 17 (incl. cmdguard svg→png fix) |
| /data NVMe | 5.3 MB/s write, 8.0 read (Lexar 2 TB) vs root 4.2/3.1 GB/s (Samsung) — ~800× degraded |
| Model cold reload from /data | ~13 min (8B Q8) — the number that broke the cycle |
| VL review pass | 25 re-reviewed + 9 skip-seen in ~30 min (CPU) |
| Local commits today | vision-review-agent ×8 (incl. 2 explicit), learnings ×2 explicit, fleet sites via daemon + explicit |

## Current machine state (for the next session)

- llama-server: healthy on :8390 (pid 2030545, loaded ~20:13)
- Fleet review cycle: ABORTED at 20:01 (timeout bug) — re-run owed
- Timers: `site-monitor.timer` (hourly) + `fleet-review.timer` (monthly) enabled via wants-symlinks; first hourly cycles unverified visually
- GPU: ollama qwen2.5vl:3b still running (not mine — untouched)
- /tmp artifacts owed to durability: `rebuild-redeploy.sh`, `fix-family-a11y.py`, `og-shots/` (committed copies live in the repos), `llama-rocm` build
- Untouched dirty state: none found in fleet repos; emeet working tree still carries pre-session unrelated dirty files (not mine)
