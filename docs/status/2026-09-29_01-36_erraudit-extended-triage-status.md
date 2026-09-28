# Status Report — Extended Erraudit Triage (Session 2026-09-29 ~00:50–01:30 CEST)

**Written:** 2026-09-29 01:36 CEST
**Scope:** This session only — triage of the extended erraudit run (`--type-aware --enforce-go-error-family --no-suppress --enforce-samber-oops --enforce-generic-return --explain --disable-extensions`, exit 2, 412 raw errors → 159 violations). No other project areas researched, per instruction.
**Session diff:** `AGENTS.md` (1 bullet rewritten), `ROADMAP.md` (1 line corrected). Zero Go code changes. Zero tests affected.

## Executive summary

All 159 violations were classified and every class verified against source. **Zero genuine code defects; zero code changes warranted.** The session's real outputs were two: (1) a corrected, evidence-backed erraudit baseline in AGENTS.md (the old note's arithmetic was wrong: 22+2+7 = 31 accepted, not 29), and (2) the discovery that the 99 `stdlib_constructor` findings come from the project's own `--enforce-go-error-family` flag — not from the `--enforce-samber-oops` flag I initially blamed. One edit away from institutionalizing a false causal story in AGENTS.md; a confirmation re-run in default mode caught it before the doc write. The pending `//nolint:erraudit` sweep (ADR `2026-09-28_erraudit-debt-policy.md`) was identified but deliberately NOT executed — it is explicitly gated on Lars's sign-off.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| a1 | Full audit log parsed; all 159 violations enumerated and mapped to exactly 5 finding classes (99 `stdlib_constructor` + 27 `generic_return` + 24 `ignored` + 7 `sentinel_concrete_type` + 2 `silent_swallow`) | `uniq -c` over the 1907-line log sums to 159/159; zero unclassified findings |
| a2 | All 24 `ignored` sites individually inspected — every one is a best-effort cleanup close, shutdown path, or already-committed response write (matches the accepted-by-design contract, including the newest site `internal/pixy/ipc.go:31`) | per-site `sed` dump of all 24 locations reviewed in-session |
| a3 | `sentinel_concrete_type` ×7 (stream.go:32–38) investigated to ground truth: sentinels are consumed directly at error sites (`http.Error` + `errorfamily.HTTPStatus`, stream.go:50–201), never via `errors.Is`; the test table builds `[]*errorfamily.Error` and calls `.ErrorCode()` (errorfamily_test.go:170–178) — the concrete declared type is load-bearing | grep: zero `errors.Is(err, errStream*)` call sites in repo |
| a4 | `generic_return` ×27 evaluated: zero `errors.As` / `errors.AsType` / error-type-assertion consumers exist codebase-wide, so concrete error return types would have no caller | grep across `*.go`: 0 hits |
| a5 | Classification architecture read and confirmed: `errorfamily.go:21` `registerErrorFamilies()` maps all 18 domain sentinels → families in one registry; `internal/pixy` imports no error library | file read |
| a6 | Default-mode erraudit re-run measured: 130 violations (99 + 31 baseline classes) — every delta between default and extended mode reconciled (extended = default + 27 `generic_return` + 2 `--no-suppress` extras) | two live erraudit runs in-session |
| a7 | AGENTS.md erraudit baseline corrected (arithmetic slip fixed: 22+2+7 = 31 accepted; "29" was wrong) and extended with the full 2026-09-29 triage (extended-mode table, project-correct invocation, re-triage conditions) | `git diff` — docs-only, 2 files |
| a8 | ROADMAP.md count drift fixed: ADR sweep scope corrected to "31 sites (2026-09-29 recount)" from stale "29" | `git diff` |
| a9 | ADR boundary respected: `docs/adr/2026-09-28_erraudit-debt-policy.md` (recommends `//nolint:erraudit` at accepted sites + `nolint-audit`) found and read — sweep NOT executed because ROADMAP explicitly gates it on Lars's decision | ROADMAP.md:91, status report `2026-09-29_00-34` |

## b) PARTIALLY DONE

| # | Item | Works now | Remains open | Blocker | Effort |
|---|------|-----------|--------------|---------|--------|
| b1 | Flag-behavior verification of erraudit | Verified empirically: `--enforce-go-error-family` emits all 99 `stdlib_constructor` (with a misleading "project enforces samber/oops" message); `--enforce-generic-return` emits the 27; `--no-suppress` bypasses a defer-close heuristic surfacing 2 extra `ignored` (hid.go:152, ipc.go:31) | Flagless baseline (`erraudit ./...`, no enforce flags) never measured — the "31 = 2026-09-28 baseline" equality is inferred from the class breakdown (22+7+2), not re-measured | none | S |
| b2 | Erraudit debt closure (ADR option B) | Rationale persisted in AGENTS.md (this session); ADR written + tool capability verified (prior session); sweep scope corrected to 31 sites (this session) | The `//nolint:erraudit` sweep itself: 0% executed; `nolint-audit` CI wiring: 0% | Lars's sign-off (by design, not forgotten) | M (30–45m) |
| b3 | `errors.Join` acceptance (auto.go:55) | Classified accepted: idiomatic join of per-step `%w`-wrapped errors; visually consistent with the registry design | No test pins that `errors.Is` finds `pixy.ErrPIXYNotConnected` / `ErrAudioSourceNotFound` **through** the Join tree — behavior currently trusted, not pinned | none | S |
| b4 | Cross-project skill feedback | This session verified rows of the go-error-modernization flag table that say "❌ Unverified" | SKILL.md not updated (lives in crush-config repo; needs a commit there, not here) | cross-repo commit | S |
| b5 | Erraudit tool diagnosis | Symptom captured: go-error-family enforcement diagnostic claims "project enforces samber/oops" and suggests oops-only fixes (which can never reach zero over idiomatic `%w` wraps) | Root cause unknown — tool-side message template or flag semantics bug; upstream investigation not started | erraudit repo access | M |

## c) NOT STARTED

| # | Item | Why not started | Priority |
|---|------|-----------------|----------|
| c1 | `//nolint:erraudit` directive sweep at the 31 accepted sites | ADR-gated on Lars's decision | High (un-gates buildflow green) |
| c2 | `erraudit nolint-audit` staleness guard wired into CI (go-test.yml or buildflow) | Depends on c1 | High |
| c3 | Gate-policy decision: which erraudit mode gates (default 31 vs extended 159) and where (buildflow step / CI / pre-commit) | Needs Lars's tooling-policy call | High (decision only) |
| c4 | samber/oops construction migration ruling (my rec: won't-do; record in ROADMAP) | Needs Lars's library-intent call | Medium (decision only) |
| c5 | Flagless erraudit baseline measurement (b1 remainder) | Deprioritized — inference is solid, measurement is cheap | Medium |
| c6 | Unit test pinning `errors.Is` through `errors.Join` (b3 remainder) | Not begun; small | Medium |
| c7 | TODO_LIST #166 (noticed): hardware-verify V2 assumption set (MotorType, DefaultPosMode, speed unit, preset slots, preset-response shape) | Needs PIXY attached | High (when hardware available) |
| c8 | TODO #129 (noticed): refresh web UI screenshots against live daemon (current shots show offline) | Out of session scope | Medium |
| c9 | Three recommendation ADRs awaiting sign-off (hint surfacing #173, NixOS extraProductIds, erraudit debt) — noticed via ROADMAP:91 + TODO_LIST:64; none actioned this session | All gated on Lars | High (decisions) |
| c10 | HARVEST of section (f) below into TODO_LIST.md / ROADMAP.md | Per skill, harvest is the follow-up step after this report | Medium |

## d) TOTALLY FUCKED UP

Nothing code-level is broken (zero Go changes; tree state before/after identical except 2 doc files). What IS fucked up is process-level — named precisely:

1. **The AGENTS.md baseline note carried a wrong count for ≥1 day.** "29 accepted-by-design" contradicted its own breakdown (22+2+7 = 31) in the same sentence. Severity: low (doc-only) but self-inflicted class error — a future session ordering 29 nolint directives would have under-swept by 2 sites and then misread the residue as sweep failure. Root cause: the 2026-09-28 triage counted classes but never summed them against the headline. Fixed this session (a7). Mitigation to keep: sum-check rule (e1).
2. **My first causal attribution was wrong, and it nearly became canonical.** I initially blamed the user's `--enforce-samber-oops` flag for the 99 `stdlib_constructor` findings; the default-mode re-run proved `--enforce-go-error-family` — the project's own policy flag — emits all 99 by itself. Had I written AGENTS.md after the first analysis pass, the baseline note would document a false story ("extended flags add the noise") and future triage would misjudge the project-correct invocation as clean. Caught in-session by the correction run (now codified as e2). Severity: medium — one edit away from a permanent wrong fact.
3. **The erraudit tool's go-error-family enforcement lies in its message.** It says "project enforces samber/oops" and suggests only `oops.*` fixes for a project whose adopted library is go-error-family. Severity: low here, but it actively misleads triage (it misled me for one analysis pass, see #2). Root cause: unknown — needs upstream investigation (b5).
4. **The buildflow gate is red-by-design on erraudit** (31 findings) until the ADR sweep lands. Standing non-green gate = live debt with a documented escape hatch (ADR option B) that has been awaiting a decision since 2026-09-28. Not fucked code — fucked decision latency. Every "just fix to zero" temptation is the documented trap; do not auto-fix.

## e) WHAT WE SHOULD IMPROVE

| # | Improvement | Impact | Concrete fix |
|---|-------------|--------|--------------|
| e1 | Triage notes never sum-checked | Wrong counts propagate into memory and misplan future sweeps | Rule: when writing accepted-finding counts into AGENTS.md, assert class-sum = headline before saving |
| e2 | Causal claims about tool output weren't re-verified in the project-correct invocation before doc-writing | Near-miss this session; would have been a permanent false doc | Rule: before recording "flag X causes finding Y", re-run without flag X and diff |
| e3 | The AGENTS.md erraudit bullet is now a ~300-word wall | Scannability for fresh sessions | Split into 1-line lead + 3 per-class sub-bullets |
| e4 | Verified flag behavior dies in this session/report instead of the skill | Next session re-doubts the same flags | Update go-error-modernization SKILL.md verification table (crush-config repo commit) |
| e5 | `errors.Join` Is-chain is trusted, not pinned | A future refactor could silently break auto-manage error aggregation | One table test: `errors.Is(joined, pixy.ErrPIXYNotConnected)` etc. |
| e6 | Post-sweep residue ambiguity | After the sweep, `--no-suppress` will still show 2 findings (heuristic-exempt defer-closes hid.go:152, ipc.go:31) — could be misread as sweep failure | One line in the ADR: expect 33 under `--no-suppress`, 31 default |
| e7 | Sentinel export policy is implicit (`errStream*` unexported vs `Err*` exported, both registry-classified) | Minor onboarding friction | One gotcha line in AGENTS.md or DOMAIN_LANGUAGE.md |

## f) NEXT TASKS (ranked; HARVEST fuel → TODO_LIST.md / ROADMAP.md)

Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Lars: approve/amend ADR option B (`//nolint:erraudit` at 31 sites) | Critical | S | Decision |
| 2 | Execute the nolint sweep at the 31 sites with per-site reason comments | High | M | Quality |
| 3 | Run `erraudit nolint-audit` post-sweep; wire staleness guard into CI | High | S–M | Quality |
| 4 | Decide + document the gating mode (default vs extended) and gate location (buildflow/CI/pre-commit) | High | S | Decision+Doc |
| 5 | Ruling: samber/oops construction — record won't-do in ROADMAP (closes the 99-finding question) | High | S | Decision |
| 6 | Sign off ADR: unsupported-hint surfacing (gates TODO #173) | High | S | Decision |
| 7 | Sign off ADR: NixOS `extraProductIds` declarative option | Medium | S | Decision |
| 8 | Implement hint surfacing (#173) post-sign-off | Medium | M | Feature |
| 9 | Implement NixOS option (module + docs + vmTest) post-sign-off | Medium | M | Feature |
| 10 | Approve multi-word preset names `join-remaining` CLI fix (ADR 2026-09-18 written) | Medium | S+M | Decision+Bug |
| 11 | Measure flagless `erraudit ./...` baseline; record counts beside the 2026-09-28 triage | Medium | S | Quality |
| 12 | Pin `errors.Is` through `errors.Join` with a unit test (auto.go aggregation) | Medium | S | Bug-prevention |
| 13 | TODO #166: hardware-verify V2 assumption set (needs PIXY attached) | High | L | Verification |
| 14 | TODO #129: refresh web UI screenshots against live daemon | Medium | M | Feature-debt |
| 15 | Update go-error-modernization SKILL.md flag table with this session's verified rows (crush-config commit) | Medium | S | Docs |
| 16 | Investigate erraudit "samber/oops" message under go-error-family enforcement; report upstream | Medium | M | Tool-bug |
| 17 | Diagnose WHY hid.go:152/ipc.go:31 are exempt in default mode (heuristic vs config); record the mechanism | Low | S | Docs |
| 18 | Split the AGENTS.md erraudit mega-bullet into lead + 3 sub-bullets | Low | S | Docs |
| 19 | Add one ADR line: expect 33 findings under `--no-suppress` post-sweep (2 heuristic-exempt) | Low | S | Docs |
| 20 | Record project-correct erraudit invocation in AGENTS.md Commands section | Low | S | Docs |
| 21 | Record sentinel export-policy rule (`Err*` vs `err*` + registry) in DOMAIN_LANGUAGE.md or AGENTS gotchas | Low | S | Docs |
| 22 | Run one full `buildflow` pass to confirm erraudit is the only red step (isolates gate debt) | Medium | M | Verification |
| 23 | Run `GOWORK=off go test -race -count=1 ./...` once as post-doc-edit green signal | Low | S | Verification |
| 24 | `erraudit fix` dry-run per skill Step 1 to confirm fix-path refuses `errors.Is` locally | Low | S | Verification |
| 25 | Decide periodic (non-gating) extended audit cadence as drift radar | Medium | S | Decision |
| 26 | Post-sweep: remove the "buildflow gate exits non-zero" caveat from AGENTS.md | Low | S | Docs |
| 27 | Sweep older status reports for stale "29" counts; ANNOTATE per docs-health (2026-09-29_00-34 mentions it) | Low | S | Docs |
| 28 | Verify the auto-commit daemon picked up AGENTS/ROADMAP + this report (don't trust pma — check `git log`) | Low | S | Hygiene |
| 29 | golangci-lint `//nolint` directive rot sweep (AGENTS.md notes test-file exclusions make directives rot) | Medium | M | Cleanup |
| 30 | templ source: evaluate gopls QF1003 (tagged switch on `mode`, templates_templ.go:935) at the `templates.templ` source level | Low | S | Cleanup |
| 31 | Check go-error-family version pin vs erraudit's enforced pattern; keep library and auditor aligned | Low | S | Quality |
| 32 | Consider an erraudit pre-commit stage on staged Go files (after measuring tool latency) | Medium | M | Quality |
| 33 | Add DOMAIN_LANGUAGE.md entries for the error-family vocabulary (Infrastructure/Rejection/Transient, registry) | Low | S | Docs |
| 34 | Decide whether doc-only corrections warrant CHANGELOG entries under project policy | Low | S | Decision |
| 35 | Optionally script the per-class recount (`erraudit … \| rg -c`) into scripts/ for repeatable audits | Low | S | Tooling |
| 36 | After c1–c3: consider `--fail-on`-style gate tuning in buildflow so green means green | Medium | S | Quality |
| 37 | Periodic `erraudit upgrade`/doctor check (tool freshness per buildflow triage) | Low | S | Hygiene |
| 38 | If samber/oops is declined AND generic_return declined: fold both into one ROADMAP won't-do entry with this report's evidence links | Medium | S | Docs |
| 39 | Re-run extended audit after the sweep to produce the post-sweep canonical residue count | Medium | S | Verification |
| 40 | Harvest this section into TODO_LIST.md (items 1–12) and ROADMAP.md (13–40 as applicable) | High | S–M | Docs |

## g) QUESTIONS (unanswerable by me)

1. **ADR option B — approve, amend, or decline?** `//nolint:erraudit` at the 31 accepted sites (22 `ignored` + 7 `sentinel_concrete_type` + 2 `silent_swallow`) + `nolint-audit` staleness guard in CI. I read the ADR and ROADMAP — the decision is explicitly yours; it gates the buildflow green state and tasks 2–4 above.
2. **Is "stdlib construction + registry classification" final architecture?** All 99 `stdlib_constructor` findings trace to `errors.New`/`fmt.Errorf`/`errors.Join` under your own `--enforce-go-error-family` flag; the registry (`errorfamily.go`) already classifies everything, and the tool's only suggestion is samber/oops. If the architecture is final, I record a ROADMAP won't-do (with this report's evidence); if you ever want oops construction, that needs its own ADR. I cannot know your library intent.
3. **Which erraudit mode should gate, and where?** Default mode (31 accepted findings, red until the sweep) vs extended mode (159, audit-only). Gate placement options I see: buildflow step, CI job in go-test.yml, or pre-commit. This is tooling policy across your fleet (buildflow skill says fleet-wide gates extend upstream) — your call.

---

**Format note:** the status-report skill's canonical output is a styled HTML dashboard; this report is Markdown per your explicit instruction (`.md` path given). Not propagated back into the skill as a default.

**Commit note:** per harness contract (never commit without explicit request) this file is left for the auto-commit daemon; `git status` at write time showed only `AGENTS.md` + `ROADMAP.md` modified — verify the daemon picked up all three files.

**NEXT STEP:** Waiting for instructions. Section (f) items 1–5 are decisions only you can make; everything else is executable on command.
