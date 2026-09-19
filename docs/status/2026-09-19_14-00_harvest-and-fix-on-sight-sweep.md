# Status Report — Harvest + Fix-on-Sight Sweep (post preset-pull session)

**When:** 2026-09-19 14:00 CEST
**Session scope:** resumed from the 12-05 report's wait state under "execute and verify until done". This session did docs-health HARVEST of that report's §f, executed every fix-on-sight item it contained, and ran the verification gates that report flagged as unexecuted (website build, coverage, vmTest). It wrote **no new features** beyond a one-line error-message improvement. Per instruction, this report covers only this session's run — no unrelated research.

**Context discovered mid-session (not my work, load-bearing):** Lars ran a concurrent session activating the website deploy CI (commit `70eec37`: `workflow_dispatch` trigger + a fresh-content sentinel grep in `website.yml`, CHANGELOG entry, new report `2026-09-19_13-45_website-deploy-ci-activation-todo-133.md`, TODO row #133 removed). The auto-commit daemon interleaved with my edits twice because of this. Everything of his was respected, nothing reverted.

---

## a) FULLY DONE

1. **Stale todo list corrected.** The prior session's carried items (#150/#152 static decode, gates, docs) marked completed before new work started.
2. **Unannounced `website.yml` change investigated before touching anything.** Diffed it first: `workflow_dispatch` trigger, deploy-job condition extended to it, and the stale-deploy sentinel grep changed from `0.4.0` to `"Corrections from official-app evidence"`. Judged intentional (Lars's), left untouched, and later **verified the sentinel passes against the built site**.
3. **HARVEST of `2026-09-19_12-05` §f — all 22 HARVEST-marked items dispositioned**, each verified against code before routing (docs-health skill loaded, harvest-guide anti-patterns applied):
   - 6 executed on sight (see 4–6 below);
   - 6 routed as new TODO rows **#167–#172** (early-abort heuristic, simulator fidelity knobs, pull-additivity property test, dprint ownership decision, `preset pull --dry-run`, shared V2 query helper);
   - 4 folded into existing rows with evidence (`12-05` §f/§g citations): Q1 decision → #141, round-trip + usbmon option → #166, pre-release gates + post-tag proxy verify → #155;
   - 2 routed to ROADMAP (per-slot pull outcome in web UI; Waybar slot-occupancy as won't-do candidate);
   - 4 closed by verification: #18 (website build), #31 (coverage), #43 (vmTest), #44 (nolint sweep — all directives intentional, lint green with the full linter set), #49 (Qt-copy-helper correction already present in map doc:251).
4. **Pull's all-slots-unreadable error now carries the failure count** (`12-05` §e.7/§f 21): `preset pull` on a fully dead device reports `"preset pull: 8/8 slots unreadable: <cause>"` instead of a bare cause. Implemented by enriching the `CommandError.Op` label so the `Unwrap` chain (and therefore `errors.Is` classification) stays intact (`commands.go:729`); doc comment updated to match; pinned by a strengthened `TestPresetPull_AllSlotsUnreadable` asserting both the count and the wrapped cause.
5. **Doc freshness fixed on sight (5 files):** `handlePresetWithLock` lock-contract comment now documents push/pull taking `hidMu` internally (`commands.go:109`); map doc §3.5 "GETs: bare heads" row reconciled with the §3.5a slot-byte/speed-rides-SET reality (`docs/hid-protocol-official-map.md:164`); README gained the `preset pull` command line and the web-UI bullet mention; AGENTS.md Presets bullet now states pull-stays-manual explicitly (no auto-sync on device appear — was implicit only); website `cli-reference.mdx` gained the pull row and `presets.mdx` gained the entire hardware-mirroring section (push was missing there too) plus both API endpoint rows.
6. **Verification gates executed and green:**
   - `GOWORK=off go test -race -count=1 ./...` → exit 0;
   - `GOWORK=off golangci-lint run --timeout 3m ./...` → 0 issues; `go vet` clean;
   - website `pnpm run build` → exit 0, 19 pages, CSP patched; `changelog.mdx` compiles with the pull content; **Lars's new deploy sentinel grep passes** on `dist/changelog/index.html`; my guide edits render in `dist/`;
   - **vmTest** `nix build .#checks.x86_64-linux.vmTest` → exit 0 (first run since the NixOS-module assertions; closes `12-05` §f 43);
   - **coverage** (`12-05` §f 31): every pull path measured 94–100% — `handlePresetPull` 100%, web handler 100%, `queryMotorPresetPos` 100%, `pullSummary` 94.1% — against a 78.6% suite average. No gap; item closed without a TODO.
7. **Living docs updated in place:** `TODO_LIST.md` (header rotated with session summary; #141/#155/#166 extended; #167–#172 created with Impact/Effort/Evidence), `ROADMAP.md` (two UX additions), `CHANGELOG.md` (preset-pull [Added] bullet amended with the failure-count behavior — amend-in-place because the feature is unreleased).
8. **Concurrent-session conflicts handled correctly.** Two "modified since read" rejections (TODO_LIST, CHANGELOG) were resolved by re-reading — the TODO_LIST re-read revealed Lars had **removed row #133** (deploy CI now active), which my stale mental model would have been blind to.

## b) PARTIALLY DONE

1. **TODO #141 (preset push/pull)** — both directions shipped; remainder unchanged: hardware slot-count sweep + response-shape pin (#166), **plus** the now-explicitly-routed Q1 product call (if the wired firmware answers mode-only, position acquisition via the SET echo mutates slot mode and needs Lars's approval).
2. **TODO #150 (enum/framing decodes)** — static decode exhausted; `MotorType` and `DefaultPosMode` value semantics need hardware (#166, optionally usbmon per Q3).
3. **TODO #138/#139/#140 (speed/battery/tracking)** — code complete and persisted; only hardware verdicts remain inside #166.
4. **TODO #155 (v0.4.1)** — release content grew again this session (failure-count polish); release-time gates (local govulncheck + fuzz smoke, post-tag proxy verify) are now written into the row but not run (CI runs the equivalents; local run folds into the release session). Still blocked on Lars's cadence call.
5. **TODO #129 (online screenshots)** — the new Pull button is one more surface the retake session must capture; hardware-gated.
6. **Public website content** — the repo's site source now documents pull everywhere (guides, CLI reference, changelog verified to build), but the **deployed** site doesn't carry it until the next deploy — which is now one `workflow_dispatch` away (Lars's action).

## c) NOT STARTED

1. **#167** pull early-abort heuristic (N consecutive slot timeouts → stop).
2. **#168** simulator fidelity knobs (builder option for `presetFullResponses`; speed-query duality modeling).
3. **#169** pull-additivity property test (never evicts/mutates under name collisions).
4. **#170** dprint ownership decision (vendor into devShell vs delete `dprint.json`).
5. **#171** `preset pull --dry-run` (report-only sweep).
6. **#172** shared V2 query helper (trigger: 4th GET family).
7. **#154** issue #6 closure (needs @zutto's hardware confirmation).
8. **#156** branch protection (GitHub settings, Lars-only).
9. **#148** innoextract upstream PR send (staged; Lars's call).
10. **Website redeploy** carrying preset pull (workflow_dispatch, Lars).
11. **ADR decisions** #116 (structured command types) and #123 (multi-word preset names) — both have recommendation ADRs awaiting Lars.
12. **ROADMAP research offshoots** (elink protocol doc, `EMVideoInput.dll`, `hidCmdSend` retry semantics, `GET_FUNC_STA` UI, `FuzzParseV2Response`, device-disappear reconcile, …) — deliberately unscheduled.

## d) TOTALLY FUCKED UP

Nothing shipped broken — every gate was green at session end and the final working tree contains only `CHANGELOG.md` (left for the auto-daemon). But these were real defects and near-misses this session, listed with root causes because the patterns repeat:

1. **Edited `README.md` without viewing it first** — I had only `rg` output and the edit tool rejected with "you must read the file before editing". One wasted round trip. Root cause: I treated a grep hit as "having read the file". The rule exists because grep hides surrounding structure (here: the exact bullet list layout).
2. **Two stale-read collisions with the daemon/Lars session** (TODO_LIST, CHANGELOG — "modified since read"). The first TODO_LIST multiedit was rejected wholesale. Recovery was correct and load-bearing: the re-read showed Lars had deleted row #133; applying my planned edits against the stale mental model wouldn't have touched that row, but I would have been editing blind in a file whose blocking section had just changed underneath me. Root cause: in this repo, hot docs (TODO_LIST/CHANGELOG/AGENTS.md) mutate mid-session _by design_; re-read immediately before every multiedit on them.
3. **Coverage grep case bug** — I grepped `cover -func` output for `presetPull` (lowercase p), which cannot match `handlePresetPull`; the function briefly looked missing/0%, which under this report's own §f 31 mandate could have spawned phantom "close the coverage gap" work. Caught within the same step because the expected row was conspicuously absent; re-ran case-insensitively.
4. **Sloppy first website-content check** — `grep -c "preset pull\|preset pull"` (the same alternative twice, and a pointless `\|` escape). It returned a number by accident, not by design; the three targeted sentinel/content greps after it are the real check. Root cause: assembling a pipeline faster than thinking about it — the exact `12-05` §d.5 failure class, one day later.
5. **QMD path guess failed** (`projects/emeet-pixyd/…` document path invented rather than looked up); fell back to `view` immediately. Noise, not damage.
6. **Near-miss, self-corrected before landing:** I first drafted the failure-count change as a separate `### Changed` CHANGELOG bullet; the project's established pattern for unreleased features is amend-the-[Added]-bullet (the extractor entry was amended the same way last session). Caught during writing, not after.

## e) WHAT WE SHOULD IMPROVE

1. **"Read before edit" must mean view, not grep.** A grep shows content; it doesn't show the file state the edit tool checks against. Standing rule: `view` the target region immediately before any edit, every time, no exceptions for "I just saw it in grep output".
2. **Treat TODO_LIST/CHANGELOG/AGENTS.md as live files.** In this repo a concurrent Lars session plus the auto-daemon means any of them can change between my read and my edit. Re-read right before multiedits; expect one rejection per session and don't grind when it happens — the re-read is where the real information is (this session: the #133 removal).
3. **Verification greps deserve the same care as code.** Case-sensitive patterns over generated output (`cover -func`) and duplicated-pattern greps both produced misleading intermediate signals this session. Cheap fix: `grep -i` for function-name lookups, and one pattern per check, always.
4. **DOMAIN_LANGUAGE.md wasn't even consulted** — last session's §e.2 lesson ("the doc checklist should include DOMAIN_LANGUAGE.md explicitly") and I still didn't open it this session while touching seven docs. No drift resulted (the failure-count wording and the no-auto-sync clause don't add domain terms), but the _check_ didn't happen. The doc-edit checklist needs it as an explicit line item, not a memory.
5. **Two cheap gates were skipped that would have strengthened claims:** an explicit `nix build` (covered transitively by vmTest building the package, but the production gate deserves its own line in a report that claims "all gates green"), and a local `govulncheck` spot-run (folded into #155's release-time gates instead; CI runs it, but a 30-second local run would have made "release-ready" concrete). Both are five-minute additions to any future sweep.
6. **Decision-shaped items got routed, not pre-decided.** #170 (dprint) is a genuine Lars call, but the row could carry a recommendation ("recommend: vendor dprint into the devShell — keeps the existing config honest") so his decision is one keystroke. Route with a position, not just options.
7. **The public changelog is stale but one click from fresh.** The deploy path is now verified end-to-end (build ✓, sentinel ✓, `workflow_dispatch` exists); the session stopped at Lars's trigger. Future sessions that verify deploy-readiness should surface "run workflow_dispatch" as the single concrete next action, this report does (§f 17).
8. **Six new TODO rows in one harvest** is on the aggressive side of the "don't dump into TODO_LIST" anti-pattern. Each is genuinely bounded and evidence-cited, but #168/#169 (both pull-test hardening) could have merged. Watch this on the next harvest; merge ruthlessly when rows share a code path.
9. **Gate parallelism left wall-time on the table.** Race tests + website build ran as parallel background shells (good), but vmTest started only after lint finished because I looked up the checks attr late. Next time: resolve the flake attr first and run all three concurrently (~2 minutes saved).
10. **Gate runs should leave a durable trace when they close a flagged item.** The vmTest-green and coverage numbers live in the TODO_LIST header note and this report, but a one-line CHANGELOG or ADR-style note ("vmTest exercised 2026-09-19, exit 0") would survive the header's rotation. Point-in-time reports go stale; the closure evidence shouldn't only live in them.
11. **Unreleased-feature changes amend the [Added] bullet** — now demonstrated twice (extractor, preset pull). Worth keeping as a standing convention note so the next session doesn't re-derive it (or write a duplicate [Changed] bullet).

## f) Up to 50 things to get done next

Ranked by impact, ties by effort. 🔌 = hardware-gated, 👤 = Lars-gated, 💬 = zutto-gated. "Routed" = already lives in TODO_LIST/ROADMAP; "NEW" = first surfaced by this report. Items 1–3 are the carried unanswered questions from `12-05` §g — they gate more downstream work than any code task.

| #  | Task                                                                                                                                                            | Impact | Effort | Category      | Route / Gate                      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- | --------------------------------- |
| 1  | Answer Q1: may `preset pull` send the SET command to read positions if the firmware answers GETs mode-only?                                                     | HIGH   | S      | Decision      | 👤 gates #141 remainder + item 18 |
| 2  | Answer Q2: cut v0.4.1 now or accumulate toward v0.5?                                                                                                            | HIGH   | S      | Decision      | 👤 gates #155                     |
| 3  | Answer Q3: is a Windows usbmon capture in scope for the #166 wired session?                                                                                     | MED    | S      | Decision      | 👤 shapes #166                    |
| 4  | Wire the PIXY and run the #166 hardware-verification bundle (closes items 5–12)                                                                                 | HIGH   | M      | Verification  | 🔌 Routed #166                    |
| 5  | Run `TestIntegration_BatteryProbe` on wired hardware (battery verdict → #139)                                                                                   | HIGH   | S      | Verification  | 🔌 Routed #139                    |
| 6  | Pin real preset slot count via a live `preset pull` sweep                                                                                                       | HIGH   | S      | Verification  | 🔌 Routed #166                    |
| 7  | Pin which preset-response shape the wired firmware answers (mode-only vs full SET-echo)                                                                         | HIGH   | S      | Verification  | 🔌 Routed #166                    |
| 8  | Pin `MotorType` values with live speed reads (usbmon capture if Q3=yes)                                                                                         | HIGH   | S/M    | Verification  | 🔌 Routed #150                    |
| 9  | Live round-trip: `preset push` → `preset pull` → values match                                                                                                   | MED    | S      | Verification  | 🔌 Routed #166                    |
| 10 | Pin speed unit + hardware limit; clamp at command layer; set web slider max (#138)                                                                              | MED    | S      | Feature       | 🔌 Routed #138                    |
| 11 | Hardware-verify TargetTrackMode variants; decide reconcile re-assert (#140)                                                                                     | MED    | S      | Feature       | 🔌 Routed #140                    |
| 12 | Verify motor iface echo (0x63 vs 0x03) on-wire — validates `v2EchoMatches`; decode reserved dword                                                               | MED    | S      | Verification  | 🔌 Routed #166                    |
| 13 | Cut v0.4.1 with the release-time gates now written into #155 (govulncheck + fuzz smoke, post-tag proxy check)                                                   | HIGH   | S      | Release       | 👤 Routed #155                    |
| 14 | Close issue #6 once @zutto confirms PIXY 2K on hardware                                                                                                         | HIGH   | S      | Release       | 💬 Routed #154                    |
| 15 | If Q1=yes: implement SET-echo position acquisition behind an opt-in flag + simulator mode                                                                       | HIGH   | M      | Feature       | Routed #141                       |
| 16 | Trigger the website deploy (`workflow_dispatch`) so the public changelog carries preset pull + corrections — content verified build+sentinel-green this session | MED    | S      | Release       | 👤 NEW                            |
| 17 | #167 pull early-abort heuristic (N consecutive slot timeouts → stop, surface in summary)                                                                        | MED    | S      | Feature       | Routed #167                       |
| 18 | Retake online web UI screenshots (now includes the Pull button) + panel crop + video poster                                                                     | MED    | S      | Documentation | 🔌 Routed #129                    |
| 19 | #171 `preset pull --dry-run` (report-only sweep; pairs with #167)                                                                                               | LOW    | S      | Feature       | Routed #171                       |
| 20 | #168 simulator fidelity knobs (builder option for `presetFullResponses`, speed-duality modeling)                                                                | LOW    | S      | Quality       | Routed #168                       |
| 21 | #169 pull-additivity property test (never evicts/mutates under collisions)                                                                                      | LOW    | S      | Quality       | Routed #169                       |
| 22 | Close `pullSummary`'s one uncovered branch (94.1% → 100%)                                                                                                       | LOW    | S      | Quality       | NEW                               |
| 23 | #170 dprint ownership decision — carry a recommendation in the row so it's one keystroke                                                                        | LOW    | S      | Decision      | 👤 Routed #170                    |
| 24 | Decide structured command types ADR (#116) — join-remaining for multi-word names rides on it                                                                    | MED    | S      | Decision      | 👤 Routed                         |
| 25 | Decide multi-word preset names ADR (#123) and land join-remaining (~6 lines)                                                                                    | MED    | S      | Feature       | 👤 Routed                         |
| 26 | Land `EMEET_PIXYD_MOTOR_SPEED` env default (#138 follow-through)                                                                                                | MED    | S      | Feature       | Routed ROADMAP                    |
| 27 | Add `FuzzParseV2Response` (framing now has dual-shape parsers)                                                                                                  | MED    | M      | Quality       | Routed ROADMAP                    |
| 28 | #172 shared V2 query helper (trigger: 4th GET family)                                                                                                           | LOW    | M      | Cleanup       | Routed #172                       |
| 29 | Switch mode reads to authoritative `GET_DEVICE_MODE` or document why SET-head query stays                                                                       | LOW    | S      | Cleanup       | Routed ROADMAP                    |
| 30 | Decode `GET_FUNC_STA` bitfield into capability-gated UI                                                                                                         | LOW    | M      | Feature       | Routed ROADMAP                    |
| 31 | Implement `hidCmdSend`-style bounded retry once #166 pins retry semantics                                                                                       | LOW    | M      | Feature       | Routed ROADMAP                    |
| 32 | Reconcile the map doc §3.5 SET-payload rows against §3.5a once hardware confirms shapes                                                                         | LOW    | S      | Documentation | 🔌 NEW                            |
| 33 | Branch protection requiring the three workflows on master                                                                                                       | MED    | S      | Quality       | 👤 Routed #156                    |
| 34 | Decide Waybar slot-occupancy won't-do candidate (currently a ROADMAP candidate note)                                                                            | LOW    | S      | Decision      | 👤 NEW                            |
| 35 | Per-slot pull outcome in the web UI (beyond the toast line)                                                                                                     | LOW    | M      | Feature       | Routed ROADMAP                    |
| 36 | Error-family adoption ADR (why DataStar handlers + breaker stay outside classification)                                                                         | LOW    | S      | Documentation | Routed ROADMAP                    |
| 37 | Demo video refresh mentioning hardware preset mirroring                                                                                                         | LOW    | M      | Documentation | Routed ROADMAP                    |
| 38 | elink wireless protocol documentation (~90 families, community value)                                                                                           | LOW    | L      | Research      | Routed ROADMAP                    |
| 39 | `EMVideoInput.dll`/`.plugin` inspection (OBS↔app pipe)                                                                                                          | LOW    | M      | Research      | Routed ROADMAP                    |
| 40 | Privacy-trigger-time semantics decode                                                                                                                           | LOW    | M      | Research      | Routed ROADMAP                    |
| 41 | Device-DISAPPEAR reconcile semantics (clean reset vs keep-last)                                                                                                 | LOW    | M      | Design        | Routed ROADMAP                    |
| 42 | SSE heartbeat + `LastEventID` replay after reconnect                                                                                                            | MED    | M      | Feature       | Routed ROADMAP                    |
| 43 | HTTP panic-recovery middleware                                                                                                                                  | LOW    | S      | Feature       | Routed ROADMAP                    |
| 44 | Camera diagnostics endpoint (full V4L2 control dump)                                                                                                            | LOW    | M      | Feature       | Routed ROADMAP                    |
| 45 | `koanf` layered config (file + env)                                                                                                                             | LOW    | M      | Feature       | Routed ROADMAP                    |
| 46 | PTZ patrol/sweep mode; configurable home position                                                                                                               | LOW    | M      | Feature       | Routed ROADMAP                    |
| 47 | OTel tracing (PTZ command latency)                                                                                                                              | LOW    | M      | Observability | Routed ROADMAP                    |
| 48 | Push-cadence / dependabot-ownership decision (ROADMAP open question)                                                                                            | MED    | S      | Decision      | 👤 Routed                         |
| 49 | Next docs-health sweep: ANNOTATE the 12-05, 13-45, and this report (annotate-not-rewrite)                                                                       | LOW    | S      | Documentation | NEW                               |
| 50 | Consider `nix build` + local `govulncheck` as standing sweep gates (both skipped-but-covered this session)                                                      | LOW    | S      | Process       | NEW                               |

## g) Three questions I cannot answer myself

Carried forward from `12-05` §g — unchanged, because they are product/ownership calls, and each one gates real work (items 13/15/18 above):

1. **Pull position acquisition vs read-only purity.** If the wired firmware answers the preset GET mode-only, the only static path to slot positions is sending `SET_MOTOR_PRESET_POS_MODE [slot][mode]` and parsing its echo — which **mutates** the slot's mode state. May `preset pull` do that (documented, opt-in flag?), or must pull stay strictly read-only? I re-verified last session: no read-only position read exists in the Beta.25 surface.
2. **Release cadence.** v0.4.1 now (five HID families + preset pull + the framing/enum corrections) or accumulate toward v0.5? The CHANGELOG [Unreleased] keeps growing — this session added the failure-count polish to it.
3. **Scope of the #166 wired session.** Is a Windows usbmon capture of the official app in scope (settles `MotorType` from real sender traffic — the one remaining static dead-end), or should the session stay probe-only?

---

_Point-in-time snapshot — goes stale. Section (f) is HARVEST input for a future docs-health run; items 1–3 are the unblock set for the largest chunk of the table._
