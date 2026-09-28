# Status Report — Full-List Execution: Security, CI, Docs Sync, ADRs

**Date:** 2026-09-29 00:34 CEST
**Session scope:** Execute the entire outstanding plan list end-to-end: M28 Dependabot triage+fix, M05 CI/deploy verification (incl. fixing the red Nix gate), M06 stale-docs sync, M07 erraudit-debt persistence, M08 harvest of both 2026-09-28 status reports, M04 decision-bundle ADR drafting, website OBS-positioning mirror, final verification, push (explicitly authorized as part of M05/M28), live production verification.
**Branch:** master · pushed `2261e38..e7cfc36` · working tree clean · all three CI workflows green on HEAD · **Dependabot open alerts: 0** (was 3).
**Format note:** user explicitly requested `.md`; the status-report skill's HTML default is overridden for this report.

---

## a) FULLY DONE

| # | Item | Evidence / verification |
|---|------|-------------------------|
| a1 | **Dependabot alerts closed (M28)** — all 3 open alerts fixed at the root: `fast-uri` 3.1.6→3.1.7 (two HIGH advisories, CVSS 7.5: authority injection via unvalidated port; host confusion via unclosed bracket — fixed by bumping the pin in `website/pnpm-workspace.yaml` `overrides`), `devalue` 5.9.0→5.9.4 in-range (MEDIUM CVSS 5.3, DoS via malformed input). `pnpm audit --prod` reports **no known vulnerabilities**; GitHub now reports **0 open alerts** (auto-closed after push). | `gh api …/dependabot/alerts` → `[]`; local audit; lockfile grep |
| a2 | **Stale supply-chain excludes removed** — `minimumReleaseAgeExclude` entries for `astro@7.3.3` and `@astrojs/starlight@0.42.2` deleted per their own expiry rule once the lockfile moved to 7.3.5 / 0.42.4 (a stale exclude silently disables the gate — the file's own warning). Supply-chain policy check passes: "Lockfile passes supply-chain policies (523 entries)". | `pnpm-workspace.yaml`; install output |
| a3 | **CI-fidelity checks pass** — `pnpm install --frozen-lockfile` (the CI guard) green; full website build: **20 pages, CSP patched 20/20, CI sentinel grep intact**. | local CI simulation; build output |
| a4 | **Nix workflow fixed — vendorHash refresh (M05)** — master's Nix gate had been red since at least `d5e7afd` (green last on `57301f2`, 2026-09-20): `go-modules` FOD hash mismatch after go.mod changed in auto-commits. Applied CI's reported hash to **both** `flake.nix:127` and `package.nix:15` (the AGENTS.md gotcha: they must move together), verified locally by `nix build` (binary produced) and `nix flake check --no-build` ("all checks passed"), and in CI: **Nix: success on e7cfc36**. | CI log hash hint; local build; `gh run list` |
| a5 | **Push + deploy + live verification (M05)** — pushed `2261e38..e7cfc36`; all three workflows green on HEAD (Go tests ✓, Nix ✓, Website ✓ incl. deploy). Live-fetched production: `/getting-started/supported-devices/` renders the full compatibility matrix (PIXY 00c0/0118 full support; C960 family/C950/C970/S600L recognized-not-controlled; unknown-`328f` funnel; `EMEET_PIXYD_EXTRA_PRODUCT_IDS` onboarding) and `/related-tools/` renders the new "Not an OBS fork — and not trying to be" block. | `gh run list`; fetch of both live URLs |
| a6 | **Stale docs synced (M06)** — `CONTRIBUTING.md` device-reporting paragraph now states the recognition reality (other EMEET devices recognized, deliberately not controlled) + the `EMEET_PIXYD_EXTRA_PRODUCT_IDS` opt-in/report funnel. `docs/hid-protocol.md` Device Identification: family table (`00c0` PIXY, `0118` PIXY 2K), extras env, pointer to `hid-protocol-official-map.md`, re-verified stamp. `docs/emeet-studio-official-app-comparison.md`: new "emeet-pixyd's side of this matrix" note under §2.2, §4 gap list annotated (items 1/2/4/5/6 shipped since, #166 caveat), §3.2 battery/charge moved to our column. Repo-wide two-PID remnant grep: only intentional matches remain (AGENTS.md header, DOMAIN_LANGUAGE — both accurate; status/planning archives deliberately untouched). Website docs verified already-correct (`quick-start`, `troubleshooting` carry "or 328f:0118"). | grep sweeps; file diffs |
| a7 | **Erraudit baseline persisted (M07)** — AGENTS.md Gotchas gained the full rationale: 31 findings → 2 genuine (socket.go `%v`→`%w`, fixed) + 29 accepted-by-design (22 `ignored` best-effort closes, 2 `silent_swallow` /proc scan races, 7 `sentinel_concrete_type` go-error-family sentinels), "do NOT re-triage from scratch", buildflow exit 69 = expected. Future sessions stop paying the re-triage tax. | AGENTS.md diff |
| a8 | **Harvest of both 2026-09-28 reports (M08)** — `docs-health` HARVEST executed with routing rigor: CHANGELOG gained the dedup-sweep entry (closed 23-47 b5); TODO_LIST gained **#173–#181** (9 evidence-cited rows: hint surfacing, hint-branch tests, SEO pack, auto-commit green-gate, BuildFlow ruff scoping, ttlCache benchmark, presetAction validation decision, `.crushrc` lint alignment, nolint sweep) and an updated awaiting-ADR footer; ROADMAP gained art-dupl baseline, vmTest-recognition note, demo-video honesty line, hidraw-side recognition, udev-snippet doc. Already-done items dropped (verified: config_test.go `slices` import committed, `nix flake check` green, comparison/hid-protocol/CONTRIBUTING done). Both source reports **not** rewritten. | TODO_LIST/ROADMAP/CHANGELOG diffs; docs-health skill followed |
| a9 | **3 recommendation ADRs drafted (M04)** — `docs/adr/2026-09-28_unsupported-hint-surfacing.md` (recommends short label + tooltip via typed `webStatus` field; gates #173), `docs/adr/2026-09-28_nixos-extra-product-ids.md` (recommends declarative list option; env stays the wire format), `docs/adr/2026-09-28_erraudit-debt-policy.md` (recommends `//nolint:erraudit` at the 29 sites + `erraudit nolint-audit` staleness auditing — **tool capability verified against `erraudit --help` before writing**, not assumed). All three cross-linked in ROADMAP "Needs a design decision" + TODO_LIST footer. **Awaiting Lars's sign-off — by design, not forgotten.** | 3 new ADR files; ROADMAP/TODO_LIST links |
| a10 | **Website OBS-positioning mirror** — `related-tools.mdx` OBS Studio section expanded with the evidence-based fork positioning (condensed from README's proven copy); verified in `dist/` AND live on production. Landing comparison (manual/extension axes) intentionally untouched — the official-app landing comparison remains the Lars-gated idea (ROADMAP/M21). | dist grep; live fetch |
| a11 | **Final gate sweep green** — `GOWORK=off go test -race -count=1 ./...` ok (both packages); `GOWORK=off golangci-lint run --timeout 2m ./...` → **0 issues**; website build 20 pages + CSP 20/20 + sentinel. | tool output |
| a12 | **Plan file updated with M28** — the Dependabot row appended to the Pareto plan's M-table (totals recalculated 2,025→2,055 min) so the plan stays the complete inventory. | plan diff |

## b) PARTIALLY DONE

| # | Item | Works | Remaining | Effort |
|---|------|-------|-----------|--------|
| b1 | **M04 decision bundle** | 3 ADRs drafted, cross-linked, recommendations grounded (erraudit suppression mechanics verified in-tool) | Lars's sign-off on all three; then the gated work unblocks: #173 (hint UI), NixOS `extraProductIds` option, erraudit `//nolint` sweep | Lars's time |
| b2 | **Docs-page OG metadata** | Verified: `og:title`/`og:type`/`og:url`/`og:locale`/`og:description` all present on `/getting-started/supported-devices/` | `og:image` exists ONLY on the landing page (site-wide pattern, not a regression); per-page OG images for docs routes = TODO #175 | S |
| b3 | **Erraudit debt closure** | Rationale persisted (a7); suppression mechanics verified; ADR written | The 29-site `//nolint:erraudit` sweep itself — gated on ADR option B | 30–45m post-ADR |
| b4 | **Website changelog sync** | Repo CHANGELOG gained Security section + dedup entry | Site `changelog.mdx` does not yet mirror the [Unreleased] Security/Security-adjacent entries — by design the site syncs at release cut (M24 lineage); becomes visible at #155 | S at release |
| b5 | **astro 7.3.5 / starlight 0.42.4 in-range bump** | Build + CSP + pages + sentinel + live fetch all green with the bumped versions | The strict typecheck gate (`pnpm typecheck` / `astro check`) was **not** run this session — CI's website job builds but (as noticed, not re-verified) may not run `astro check`; flagged as f-item | S |
| b6 | **Pareto plan bookkeeping** | M28 appended; M04–M08 work now done | The plan file has no status column — executed rows (M04–M08, M28) are not marked in-plan; TODO_LIST/ROADMAP carry the truth, but the plan reads as all-open | S |

## c) NOT STARTED

Deliberately out of this session's scope; all are tracked (plan M-table, TODO_LIST, ROADMAP):

1. **Hardware sessions M01–M03** (#166 bundle: battery/identity, speed-unit/MotorType pin, preset sweep + online screenshots #129) — no PIXY attached; still the single highest-impact open work.
2. **M09/M10** hint surfacing + recognition-hardening tests (created as #173/#174, not executed; #173 gated on ADR).
3. **M11** NixOS option (gated on ADR).
4. **M12–M18** error-wrapping audit, go-error-family expansion, structured command types (#116 ADR), multi-word presets (#123 ADR), CI store-path guard, vmTest extension, a11y checklist execution.
5. **M19–M23** screenshot pipeline, website polish, public comparison page (Lars gate), SSE heartbeat/replay, Waybar enrichment.
6. **M24** release bundle (#148 upstream PR send, #154 close #6, #155 cut, #156 branch protection).
7. **M25–M27** quality pack, tooling cadence, long-tail umbrella.
8. New ROADMAP rows from the harvest (hidraw-side recognition, udev snippet doc, art-dupl baseline, demo honesty line) — recorded, none scheduled.

## d) TOTALLY FUCKED UP

Nothing data-destroying; all mistakes caught and corrected within the session. Honest ledger:

| # | What | Detail | Cost / root cause |
|---|------|--------|-------------------|
| d1 | **Applied the pnpm≤9 override pattern** — added `pnpm.overrides` to `website/package.json` before checking the installed pnpm major (11), which no longer reads that field ("The pnpm field is no longer read… ignored"). Had to revert the edit and put the override where pnpm 11 actually looks: `pnpm-workspace.yaml`. | One wasted edit + revert cycle. Root cause: pattern recall from older pnpm, not verified against the tool in use. The WARN line caught it — read the full tool output. |
| d2 | **First `pnpm update fast-uri` silently no-oped** — command exited clean, `devalue` bumped, `fast-uri` stayed 3.1.6. Near-miss: declaring victory on exit code. Caught by re-grepping the lockfile before moving on. Real blocker: a **pre-existing `fast-uri: 3.1.6` pin in `pnpm-workspace.yaml` overrides** (an earlier security pin) defeating scoped updates. | Lesson: security pins live in the workspace overrides and silently defeat version-scoped `pnpm update`; always re-verify the lockfile after dependency commands. |
| d3 | **Master's Nix gate sat red for ~a day before anyone looked** — `nix.yml` failed on `d5e7afd` and `2261e38` while previous verification only celebrated green Go tests. Inherited process gap, fixed this session (a4), but the "push master, watch workflows" step of the prior session clearly watched only the expected-green workflow. | ~1 day of red CI at HEAD. Fix: check ALL workflows after every push (3-line `gh run list`), not just the one you expect to matter. |
| d4 | **Bogus live-verify command** — drafted shell pseudo-code (with a `fetch_url` variable and a stray loop) instead of going straight to the fetch tool; the command backgrounded and had to be killed. `curl` is banned in this harness anyway. | One wasted round; zero damage. |
| d5 | **CSP verification false alarm** — `grep -c "Content-Security-Policy"` returned 0 on built HTML and was briefly treated as a failure; the tag is embedded with different casing/attribute order (`security-policy` matches case-insensitively). Resolved by reading `fix-csp.mjs` output ("patched 20/20") and inspecting the built HTML. | Verification-script bug, not a product bug: HTML-attribute greps must be case-insensitive or reuse the patcher's own count. |
| d6 | **Dependabot's own update-PR run was red** (`npm_and_yarn in /website … failure` on `d5e7afd`) — noticed only in passing while polling CI; superseded by our direct fix (alerts now 0, so the PRs are moot), but it is another signal channel nobody read. | Zero residual; noted so future Dependabot PR failures get triaged, not ignored. |

## e) WHAT WE SHOULD IMPROVE

1. **Verify all workflows after every push** — a 3-line `gh run list --branch master` covers go-test/nix/website; "green" claims must name all three. (Direct fix for d3.)
2. **Pin pnpm-11 knowledge where the next session will look** — overrides + settings live in `pnpm-workspace.yaml`, not package.json. Candidate AGENTS.md website-gotcha line (not yet written — f-item).
3. **Annotate security pins with their GHSA/alert refs** — `pnpm-workspace.yaml` overrides carry bare versions; a one-line comment per pin (`fast-uri: 3.1.7  # GHSA-qw65…/GHSA-58mr…`) makes the next Dependabot triage a lookup instead of an archaeology dig. Same sweep should check whether `brace-expansion`/`js-yaml`/`svgo` pins are still needed.
4. **Re-grep the lockfile after EVERY dependency command** — exit codes lie by omission; the diff is the truth. (Worked this time because it was done; make it a reflex, not a lucky habit.)
5. **Live verification goes through the fetch tool, full stop** — no shell drafts, no curl-class tools. (Fix for d4.)
6. **Case-insensitive greps for HTML attributes** in any verification snippet, or reuse the patcher's own counts. (Fix for d5.)
7. **Run the strict typecheck gate alongside build checks when dependencies move** — the astro/starlight in-range bump was verified by build+deploy only; `astro check` should have run once locally (b5/f-item).
8. **Push only after the working tree is confirmed clean** — the auto-commit daemon races edits; the final push correctly followed a clean `git status` + sleep. Keep that order sacred.
9. **Fold the erraudit `//nolint` sweep into the #181 golangci nolint sweep** (once the ADR lands) — one mechanical directive pass over the codebase, not two.

## f) UP TO 50 THINGS TO GET DONE NEXT

*(Brainstorm-ranked; ~10 TODO_LIST-grade, the rest ROADMAP fuel — HARVEST with routing rigor, do not entomb.)*

**Session leftovers — highest leverage, do first**

| # | Task | Impact | Effort | Note |
|---|------|--------|--------|------|
| 1 | Lars signs off / amends the 3 ADRs (hint surfacing, NixOS option, erraudit policy) | HIGH | S | Gates #10/#11/#173 + plan M09/M11/M07b |
| 2 | Add AGENTS.md gotcha: pnpm 11 reads overrides from `pnpm-workspace.yaml`, not package.json | MED | S | This session's d1/d2 lesson, 2 lines |
| 3 | Comment security pins in `pnpm-workspace.yaml` with GHSA refs; audit whether `brace-expansion`/`js-yaml`/`svgo` pins are still needed | MED | S | Speeds the next Dependabot triage |
| 4 | Run `pnpm typecheck` (astro check) once against the bumped astro 7.3.5 tree; if CI lacks the typecheck job, add it to website.yml | MED | S | b5 gap |
| 5 | #174: branch tests for `device`/`probe` hint output + `unsupportedWarnLimiter` log branch | MED | S | TODO #174, un-gated |
| 6 | #175: SEO pack — per-page description audit, sitemap ping post-deploy, per-page og:image for docs routes (astro-og-canvas) | LOW | S | TODO #175 |
| 7 | #176: gate the auto-commit daemon on a green pre-commit build (kills red-at-HEAD) | HIGH | M | TODO #176 |
| 8 | #177: scope BuildFlow's ruff away from `tools/inno661/`+`tools/emhid/` (or ruff-clean, SIM115 ×20) | MED | S | TODO #177 |
| 9 | #178: benchmark delta — `ttlCache[T]` + preset double-RLock perf-neutral | LOW | S | TODO #178 |
| 10 | #173: web UI offline panel + Waybar tooltip/JSON hint surfacing (post-ADR-1) | MED | M | TODO #173 |
| 11 | Erraudit `//nolint:erraudit` sweep at the 29 sites + `nolint-audit` (post-ADR-3); fold into #181 | MED | M | Un-gates buildflow green |
| 12 | NixOS `hardware.emeet-pixy.extraProductIds` option + vmTest case + docs (post-ADR-2) | MED | S | Plan M11 |
| 13 | #179: decide `ValidatePresetName`-always in web preset action (400 vs passthrough) | LOW | S | TODO #179 |
| 14 | #180: align `.crushrc` LSP lint config with `.golangci.yml` | LOW | S | TODO #180 |
| 15 | #181: mechanical nolint sweep (golangci directives; + erraudit if #11 lands) | MED | M | TODO #181 |
| 16 | Mark executed rows (M04–M08, M28) in the Pareto plan or add a status column | LOW | S | b6 |

**Hardware (the big rock — one wired session closes most)**

| # | Task | Impact | Effort | Note |
|---|------|--------|--------|------|
| 17 | M01: battery probe + identity heads live (`TestIntegration_BatteryProbe`) | HIGH | 90m | Feeds #139/M27 |
| 18 | M02: speed-query duality verdict, unit/limit pin, MotorType/DefaultPosMode | HIGH | 90m | Un-gates clamps/sliders |
| 19 | M03: preset slot sweep + live push→pull round-trip + online screenshots | HIGH | 90m | #129, #141 |
| 20 | #129 online web-UI screenshots + panel crop + video poster (bundle with M03) | MED | S | BLOCKED row |
| 21 | #150 optional Windows usbmon capture to settle MotorType | LOW | M | Lars's Q3 |

**Feature/code streams**

| # | Task | Impact | Effort | Note |
|---|------|--------|--------|------|
| 22 | M12: error-wrapping audit (`%v`→`%w` where chains matter) + pin tests | HIGH | 90m | ROADMAP error theme |
| 23 | M13: go-error-family adoption (`HTTPHandler`, `LogError`, `Assert*`) + scope ADR | MED | 90m | Sequenced after M12 |
| 24 | M14: structured command types slice 1 (post-#116-ADR) | HIGH | 100m | Lars gate |
| 25 | M15: multi-word preset names via CLI join-remaining (post-#123-ADR) | MED | S | ~6 lines + tests |
| 26 | #172: shared V2 query helper — re-check trigger after M02 (speed readback = 4th GET family) | LOW | M | TODO #172 |
| 27 | M16: CI guard — fail if go-modules FOD references store paths | MED | S | Supply-chain net |
| 28 | M17: vmTest fake sysfs + daemon boot inside the VM | MED | 100m | Exercises recognition |
| 29 | M25: quality pack — `ResolveProductID` property test, `probeVideo4linux` benchmark, simulator vendor-byte refusal | MED | S | Pins invariants |
| 30 | hidraw-side fixed-EMEET recognition or recorded video-only decision | LOW | S | ROADMAP row |
| 31 | #139 follow-through: background/hybrid battery refresh only if hardware verdict + demand | LOW | S | ADR revisit trigger |

**Website / marketing**

| # | Task | Impact | Effort | Note |
|---|------|--------|--------|------|
| 32 | M19: scripted headless-chromium screenshot pipeline (online/offline) | MED | 90m | #129 stops being manual |
| 33 | M20: website polish — per-page feedback links, reading time, who-is-this-for mirror | MED | 90m | ROADMAP web |
| 34 | M21: public EMEET-STUDIO comparison page | MED | 90m | **Lars gate** (exposure call) |
| 35 | M22: SSE heartbeat + `LastEventID` replay + OTel PTZ-latency span | MED | 100m | Observability |
| 36 | M23: Waybar enrichment (auto mode, pan/tilt; charge classes post-M02) | LOW | S | ROADMAP UX |
| 37 | Demo-video multi-device honesty line (optional reshoot) | LOW | S | ROADMAP row |
| 38 | udev snippet doc for extra-ID devices (default stays no-rules) | LOW | S | ROADMAP row |
| 39 | Sync site `changelog.mdx` with [Unreleased] (incl. Security) at next release cut | LOW | S | M24 lineage |
| 40 | Starlight callouts sweep for buried prose notes | LOW | S | ROADMAP web |

**Release / community / process**

| # | Task | Impact | Effort | Note |
|---|------|--------|--------|------|
| 41 | M24 bundle: #148 innoextract upstream PR send (verify-before-filing gate first) | MED | S | Lars's call |
| 42 | #154: close issue #6 after @zutto's real-2K confirmation | HIGH | S | BLOCKED |
| 43 | #155: cut v0.4.1 vs v0.5 (accumulated [Unreleased] incl. security + multi-device) | MED | S | Lars cadence call |
| 44 | #156: branch protection requiring go-test/nix/website on master | MED | S | Lars settings — would have caught d3 mechanically |
| 45 | M26: dprint/prettier for `.mdx`/`.mjs`; Renovate or scheduled `nix flake update` | LOW | S | Ends formatter/dep-rot classes |
| 46 | M27 long-tail umbrella: koanf ADR, device-disappear ADR, `FuzzParseV2Response` (post-M02), `GET_FUNC_STA` decode (post-M01), `EMEET_PIXYD_MOTOR_SPEED` env (post-M02), S600L/PIXY-Wireless watch rows, privacy-trigger-time + `hidCmdSend` retry research | LOW | 100m | Mostly gated on M01/M02 |
| 47 | art-dupl accepted-clone baseline (if the tool supports one) | LOW | S | ROADMAP row |
| 48 | Decide push cadence: daemon pushes per task vs manual (ROADMAP open question) | MED | S | Lars's call; interacts with #176/#44 |
| 49 | Triage future Dependabot PR runs as first-class signals (d6) — decide bump ownership for `website/` (ROADMAP open question) | MED | S | Pairs with #48 |
| 50 | `go-structure-linter` cmd/ findings stay documented-won't-do — revisit only if the tool changes | LOW | — | Recorded; keep closed |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **ADR bundle sign-off:** do you approve the three recommendations as written — (1) hint surfacing via typed `webStatus` field + short label/tooltip, (2) declarative `hardware.emeet-pixy.extraProductIds` with env as the wire format, (3) `//nolint:erraudit` at the 29 accepted sites + `erraudit nolint-audit` (buildflow gate back to meaningful green)? Any amendment changes which of #10/#11/#12 unblock next.
2. **Push cadence + bump ownership (they collide):** should the auto-commit daemon push per task (ROADMAP open question), and should website dependency bumps be Dependabot-only with local sessions never bumping? This session bumped website deps locally *and* pushed — both were authorized here, but the standing policy is still yours to set; it decides whether #176's green-gate idea also needs a push-gate.
3. **Release cut:** accumulate for v0.5 (multi-device + battery/presets/tracking + security) or cut v0.4.1 now? [Unreleased] carries user-visible features and a security section; the site changelog stays behind until the cut.

---

*Report covers only this session's run and what it noticed in passing. Per the harness contract, not committed manually — the auto-commit daemon picks it up.*
