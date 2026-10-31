# Status: Architecture Review Session — samber/do Premise, Self-Health, Resilience

**Date:** 2026-10-31 14:30
**Format note:** .md per requester's explicit instruction; the series default is HTML (html-report-kit) — the series directory deliberately mixes both.
**Session scope:** User asked "Are we using samber/do v2 with Health Checks superbly? Do we have a PROPER Service Oriented, composable, resilient, self-health architecture?" → deep review + detailed .md report. This status covers ONLY this session's run and what I noticed during it, plus the self-review the user now demands.

**Artifacts produced this session:**

1. `docs/architecture-understanding/2026-10-31_14-17_service-orientation-resilience-self-health.md` — the review deliverable
2. `TODO_LIST.md` — header updated, findings harvested as rows #190–#193

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| a1 | Loaded BOTH matching skills before acting (`samber-do-best-practices`, `architecture-review`) per skill mandate | session log, first tool call |
| a2 | Premise verified, not assumed: samber/do absent from `go.mod`; only mention repo-wide is the 2026-05-01 rejection note | `go.mod:5-18`, `docs/planning/archive/2026-05-01_04-32_quality-sweep-round2.md:83` |
| a3 | Rejection rationale traced to source (quality-sweep decision table), plus the consistent family of rejections (kardianos/service, Daemon decomposition #86, cmd/ layout #85) | ROADMAP rejected table |
| a4 | DI audit against DO-1→DO-6 with per-rule evidence (`deps.go:13-27`, `main.go:116-130`, 163 test-option usages counted via grep+awk) | report §1 |
| a5 | Five-layer self-health model mapped with file:line evidence (systemd notify/watchdog, `/api/health` 503 semantics, `/api/status` always-200 contract, 9 OTel metrics, degradation-by-omission) | report §2; `handlers.go:139-180`, `metrics.go`, `modules/nixos.nix:116-120` |
| a6 | Resilience audit (breaker, HID 500ms timeout, privacy trap, reconcile, debounce, caches incl. negative-result power cache, broadcaster non-blocking, uevent continue-on-error, lock contract) | report §3 |
| a7 | Series continuity honored: prior reports listed, 2026-06-06 modularity review skimmed, its stale recommendations (HTMX-era, TODO #21/23/24) identified as shipped | report header |
| a8 | 6 findings produced, severity-ranked, each with file:line evidence + concrete fix (F1 subprocess timeouts, F2 silent listener degradation, F3 trapReasserts drain, F4 watchdog invariant, F5 health dimension, F6 container/service duality) | report §5 |
| a9 | Scored all four asked dimensions + DI lens, with justification per score | report §6 |
| a10 | Honest samber/do tradeoff table incl. the one stealable idea (health-check registry as plain map, not a dependency) | report §8 |
| a11 | Findings harvested into `TODO_LIST.md` as #190–#193 per docs-health HARVEST; rows verified present (`rg -c` = 4) | TODO_LIST rows 190-193 |
| a12 | Claims spot-verified during research rather than asserted: `trapReasserts.Wait()` call sites (only tests), WatchdogSec=30/Restart=on-failure, HID timeout 500ms, CLI client deadlines (`internal/pixy/ipc.go:26-62`), server conn deadlines 5s, `autoError` surfaced via snapshot | session greps |

## b) PARTIALLY DONE

| # | Item | What's missing |
| --- | --- | --- |
| b1 | Prior-review (2026-06-06) cross-reference | Spot-checked its sections and 3 of its recommendations (CommandRunner/UeventListener shipped, DataStar shipped); its "17 open TODOs" list declared stale from sampling, not exhaustively re-audited row by row |
| b2 | Skill methodology compliance | `architecture-review` SKILL.md read and its process followed in substance (map → dependencies → domain alignment → pain points → scores → roadmap), but its `references/assessment-rubric.md` and `references/review-methodology.md` were NOT loaded — my 1-5/4.5 scoring scale was self-invented, not the skill's rubric |
| b3 | Output-format decision | User explicitly demanded .md; skill default is HTML via html-report-kit template. Override is legitimate but the deviation was never explicitly flagged, and the series directory now mixes .html (1 review) and .md (1 review) formats |
| b4 | F1 (subprocess timeouts) evidence chain | The daemon-side claim is source-verified (`commander.go` has no timeout wrapper; socket path passes daemon-lifetime ctx). The HTTP-side claim ("request contexts carry no handler deadline") ⚠ **inferred** — rests on net/http semantics knowledge, not an in-repo empirical check |
| b5 | TODO_LIST header update | New "Updated:" line written, but the old one was compressed into a parenthetical "Prior:" instead of being demoted to a separate `**Previous:**` line — mild format drift from the file's established convention |

## c) NOT STARTED (deliberately out of this session's scope)

| # | Item | Why not |
| --- | --- | --- |
| c1 | Implementing any of #190–#193 | User asked for review + report, not refactoring; roadmap belongs in TODO_LIST |
| c2 | D2/SVG architecture diagrams for the review | Series convention has them; report is prose+tables only |
| c3 | Verifying the June review's full 17-Item TODO table | Would have been re-auditing old work unrelated to the question asked |
| c4 | Empirical watchdog-stall test (wedge a fake v4l2-ctl, measure loop stall vs WatchdogSec) | Good future pinning test for F1/F4, not needed to justify the finding |
| c5 | CHANGELOG entry | Docs-only session; CHANGELOG carries code work |

## d) TOTALLY FUCKED UP

Nothing catastrophically broken — no code touched, no tests regressed (none applicable), all file edits verified applied. Honest process misses, in descending severity:

| # | Miss | Severity |
| --- | --- | --- |
| d1 | Did not read the two reference files the architecture-review skill points to for scoring + methodology (b2) — risk: my scores are not calibrated against the rubric the series' HTML review used, so cross-review score comparisons are approximate | Medium |
| d2 | TODO_LIST "Previous:" demotion convention bent (b5) — I authored format drift in a file whose conventions are explicit | Low |
| d3 | Report asserts "read/write deadlines" for the CLI client from the `SetDeadline` call name without opening the function body — if `SetDeadline` were narrowly scoped the claim would overreach (it isn't; stdlib sets both, but I didn't look). **DE-INFERED 2026-10-31:** read the body — `internal/pixy/ipc.go:14` calls `conn.SetDeadline(now+timeout)`, the stdlib call that sets BOTH read and write deadlines; the claim is source-verified now (nuance: the one absolute deadline set before the write also bounds the subsequent read, so `DefaultWriteTimeout` is effectively the whole round-trip bound) | Low |
| d4 | No explicit note in the report that the deliverable deviates from the series' HTML format by user instruction | Low |

## e) WHAT WE SHOULD IMPROVE (session-derived)

1. **Read skill references before scoring**, not just SKILL.md — the rubric files exist precisely so scores are comparable across the series. A 10-minute read would have made §6 rubric-aligned.
2. **Flag format overrides explicitly** in the artifact itself ("format: .md per requester, series default is HTML") — future readers won't have to wonder.
3. **Respect established doc-file micro-conventions** (Previous-line demotion) even when editing quickly; they exist for scan-ability.
4. **Distinguish source-verified from knowledge-inferred claims** in reports — I did this implicitly (b4) but a `⚠ inferred` marker on the 2-3 inferred claims would be more honest.
5. **Diagrams earn their keep in this series** — a small D2 of the five health layers would communicate §2 faster than the table.
6. **Counting hygiene**: "163 test usages" is a grep-sum of option-name occurrences, not a count of distinct call sites or tests — label such numbers with their counting method.

## f) NEXT THINGS (session-scoped, ranked; NOT a fresh backlog audit)

**P1 — from this review's findings (already TODO rows):**

1. Implement #190: `context.WithTimeout` wrappers for `v4l2Set`/`parsePTZValues`/stream/notify (F1)
2. Implement #191a: fail-fast on startup socket-listen failure (error out of `Run`) (F2)
3. Implement #191b: `degraded` dimension in `/api/health` + `/api/status` (socket/web listener liveness) (F2/F5)
4. Implement #192: drain `d.trapReasserts` in `handleShutdown` — or reclassify won't-do with rationale (F3)
5. Implement #193: WatchdogSec↔PollInterval invariant comment in `modules/nixos.nix` (F4)
6. Write the pinning test for F1: simulator/fake commander that hangs, assert the daemon-side call returns within the new bound
7. Write the watchdog-stall pinning test (c4): fake hung subprocess under `hidMu`+`v4l2Mu`, assert loop iteration bound stays ≪ WatchdogSec

**P2 — close this session's own gaps:**

8. Read `assessment-rubric.md` + `review-methodology.md`; re-score the report's §6 table against the actual rubric; annotate deltas
9. Fix the TODO_LIST header to proper `**Previous:**` demotion form
10. Add the format-deviation note to the report header (one line)
11. Add `⚠ inferred` markers to the 2 inferred claims (HTTP request-ctx deadline behavior; CLI SetDeadline scope — or just read `SetDeadline` and de-infer it)
12. Add a D2 diagram of the five health layers to the series directory, linked from the report
13. Re-audit the June review's 17 open TODOs exhaustively and mark each shipped/stale in a one-line addendum (it feeds the next series entry)
14. Rename or cross-tag the report so the series index makes the .md/.html format mix explicit

**P3 — natural follow-ons noticed during the session:**

15. Per-component health registry (the stealable samber/do pattern): `map[string]func() error` walked by the health handler — pairs with #191b
16. `/api/health` could expose breaker state (`hidFailCount`/threshold) as a gauge — currently only the cumulative `hid_failures_total` counter exists
17. `checkExternalDeps` results (missing ffmpeg/v4l2-ctl/wpctl/notify-send) could feed the same `degraded` dimension instead of being startup-log-only
18. Consider a `emeet_pixyd_auto_errors_total` counter — `autoError` is surfaced to humans (status line) but not to metrics
19. Socket accept-loop death (persistent accept errors) currently `continue`s forever; a consecutive-failure ceiling would convert F2's silent class into a logged/observable one
20. The `powerCache` negative-result TTL (remembered absence) pattern could be reused for the June review's "probe cost" concern if probes ever get expensive
21. Document the health-layer model (report §2) in `website/src/content/docs/guides/monitoring.mdx` — the doctrine file covers /api/health semantics but not the five-layer picture
22. If #191b lands, extend the Quickshell guide with the `degraded` field (stable-interface discipline: docs ship with the field)

**P4 — hygiene:**

23. Add "counting method" labels to any future numeric claims in reports
24. Session memory: AGENTS.md needs nothing new (findings are TODO work, not enduring context — correct call this time)

## g) QUESTIONS (cannot be figured out from the repo)

1. **Series format going forward:** architecture reviews as .md (this session, your instruction) or back to the html-report-kit HTML the June review used? The series directory now holds one of each.
2. **Implement the P1s now?** #190 (subprocess timeouts) and #191 (fail-fast + `degraded` health) are S/S-M effort with MED impact — green light this session, or queue behind your current priorities (e.g. the #155 v0.5.0 release cut, #166 hardware bundle)?
3. **#192 ruling:** drain `trapReasserts` at shutdown (2 lines, cosmetic-benefit) or record won't-do (the 2026-10-01 session already judged it cosmetic)? Either is defensible; it's your call whether the WaitGroup stays as test-only plumbing.

---

**State of the tree:** `TODO_LIST.md` modified (header + rows #190–#193), new report file untracked; auto-commit daemon will sweep. No code changed, no tests applicable, lint unaffected.
