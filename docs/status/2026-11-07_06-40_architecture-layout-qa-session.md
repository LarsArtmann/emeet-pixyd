# Status Report — Architecture & Layout Q&A Session

**Date:** 2026-11-07 06:40 CET
**Session type:** Q&A only — zero code authored, zero files modified by this session.
**Topics:** project-root layout, go-modularize assessment, single-main.go feasibility, sub-folder (package) options.

## Scope Line

**Verified here (fresh commands, this session):**

- Package inventory: `GOWORK=off go list ./...` → exactly 2 packages (`emeet-pixyd` root, `internal/pixy`).
- File/line counts (fresh at 06:41 CET, `ls` + `wc -l`): root hand-written source = **33 files / 7,172 lines**; root tests = **72 files / 17,942 lines**; `internal/pixy` = **7 files / 1,562 lines**; generated `templates_templ.go` = **1,871 lines** (86 KB).
- `internal/pixy` dependency direction (`go list -deps/-f '{{.Imports}}'`): imports only domain-adjacent libs (`go-branded-id`, `go-error-family`) + stdlib; **zero** daemon infrastructure (no datastar, OTel, hidraw, v4l2).
- go-modularize skill loaded and its Unix gate applied; go.mod/go.work state: single go.mod, no committed `go.work` for this project (parent go.work excludes it — `GOWORK=off` mandatory per AGENTS.md).
- Git state at 06:41: `git status --short` shows 5 modified files **not authored by this session** (`AGENTS.md`, `static/style.css`, `templates_templ.go`, `website/package.json`, `website/src/styles/global.out.css`); HEAD = `2b4b123` (auto-commit daemon).

**NOT verified here (and why):**

- `.github/workflows/*.yml` contents — never opened this session; the "cosmetic move needs no CI changes" claim in chat rests on reasoning (`./...` patterns), not on reading the files. Flagged in (d).
- `.crushrc` contents (LSP/templ path pins) — not read.
- Authorship/intent of the 5 modified files — out of scope per the user's "don't research unrelated stuff" instruction; per the never-revert-others'-work rule they were left untouched.
- TODO_LIST / FEATURES / ROADMAP cross-references — deliberately not researched (user constraint); this report is session-scoped only.
- Nothing was built, tested, or deployed — there was nothing to run.

---

## Context: what the session actually was

Four architecture questions, answered with tool-verified evidence. No code changes. The value produced is **three verdicts** (flat root is deliberate; single main.go rejected; real sub-packages blocked) — and the session's biggest weakness is that **those verdicts exist only in this conversation**, not in any durable artifact.

---

## a) FULLY DONE

1. **Root-layout question answered with a verified inventory.** Root = one `package main` (33 source files / 7,172 lines), one domain package (`internal/pixy`, 7 files / 1,562 lines), ~72 colocated test files (17,942 lines), generated templ output (1,871 lines), plus standard configs/docs. Evidence: `ls`, `go list`, `wc -l` outputs in-session.
2. **go-modularize skill applied correctly.** SKILL.md loaded *before* analysis; Unix gate run on both packages. Verdict: **stay one module** — `internal/pixy` fails gate 2 (pipe test: no named second consumer exists anywhere — NixOS module consumes the binary, `tools/` is Python); root `main` passes purpose/direction/surface; FM#11 (god-module) not triggered (2 packages, not 14+). Evidence: gate table delivered; `go list` import check.
3. **Single-main.go question answered and rejected on numbers.** A merged file would be 7,172 hand-written lines plus an 86 KB generated file that cannot be inlined, and the ~17.9k lines of tests stay as ~72 files regardless — the clutter complaint would not even be solved. Evidence: fresh `wc -l` (originally answered with byte-based estimates — see (d)1).
4. **Sub-folder question answered with a seam table.** `internal/pixy` = real seam (exists); `cmd/emeet-pixyd/` + `internal/daemon/` = cosmetic move (viable, small enumerable cost); `internal/hid|stream|web/…` = blocked by the god-struct (`*Daemon` private state, `hidMu`/`v4l2Mu` lock-order contract pinned by same-package tests). Verdict: real sub-packages make sense only after a `Daemon` decomposition, and only with a named second consumer (gate 2).
5. **Skill compliance.** All three matching skills (go-modularize, status-report, brutal-self-review) were loaded via `view` before task action, per the mandatory activation flow.

## b) PARTIALLY DONE

1. **go-modularize formal workflow.** Phases 1–2 (detect state, dependency analysis) done; gates 1–5 applied informally. Phases 3–7 (HTML proposal in `docs/modularization/`, execution plan, reflection) intentionally skipped because the verdict is "no split" — but the skill's output contract was not met, and the verdict is chat-only. Remaining: S effort for a decision-record writeup, M for full formal artifacts. Blocker: none — awaiting user preference.
2. **Cosmetic `cmd/` + `internal/daemon/` move.** Designed (cost enumerated: `subPackages` in `package.nix`/`flake.nix`, AGENTS.md table rewrite) but **not executed and not fully verified** — workflows and `.crushrc` were never read, so the "no CI change" claim is unproven. Effort to finish: M. Blocker: user decision (offered twice, not picked up).
3. **Line-count accuracy.** First answer gave byte-derived estimates ("~4,700 source lines", "~11k test lines"); verified reality is 7,172 / 17,942 (35–60% low). Now corrected with fresh `wc -l`; the verdicts were unaffected (single-main.go rejection got *stronger*). Marked partial because the chat record still contains the wrong numbers.

## c) NOT STARTED

1. **The cosmetic move itself** (`git mv` to `cmd/emeet-pixyd/` + `internal/daemon/`) — awaiting user decision; reversible but touches flake paths and docs.
2. **Durable decision record** (ADR + AGENTS.md note) for the flat-layout/modularization verdict — the session's one real loose end.
3. **God-struct decomposition** (extract `hidTransport` / `streamServer` / `callDetector` types) — precondition for any real sub-package split; ROADMAP-grade, no consumer demand today.
4. **DI evolution** (`deps.go` function fields → interfaces) as decomposition prep — not started, not scheduled.
5. **HARVEST of this report's (f) into TODO_LIST.md/ROADMAP.md** (docs-health) — deferred because the user said WAIT FOR INSTRUCTIONS.
6. **Verification of CI/crush config path pins** before any move (workflows, `.crushrc`) — listed in (f), not started.
7. **Investigation of the 5 foreign modified files** — not started, deliberately out of scope; flagged as a coordination question in (g).

## d) TOTALLY FUCKED UP

Radical honesty. Nothing in the *repo* was broken by this session (no code touched). What was fucked up is **answer quality discipline**:

1. **Estimates asserted as facts.** I presented byte-derived line counts as confident numbers ("~4,700 lines", "~11k test lines") without running `wc -l`. Verified reality: 7,172 / 17,942 — errors of 35–60%. Severity: low (chat-only, nothing built on them, verdict direction unchanged) — but this is exactly the verify-before-encoding failure the project's doctrine bans. Mitigation applied: every number in this report is from a fresh command.
2. **Internal count inconsistency.** First answer: "32 non-test files, ~60 test files"; later answers drifted to "~70"; verified: 33 source / 72 tests. Cause: `_templ.go` counting ambiguity plus **the tree changed mid-session** (a concurrent session/daemon modified 5+ files). Severity: cosmetic. Mitigation: this report pins exact counts + commands; future sessions must timestamp inventories.
3. **Unverified cost claim.** "The cosmetic move needs no CI changes (everything runs `./...`)" was asserted without opening a single workflow file. If the move gets executed on that claim and go-test.yml's fuzz-list assert or the nix jobs carry root-path pins, the move breaks CI. Severity: medium *conditional on execution*. Mitigation: (f) items 3–4 make reading workflows/.crushrc a hard precondition of the move.
4. **Verdict nowhere durable (split brain).** The layout rationale + gate verdict live only in this conversation. AGENTS.md documents *what* each file does but not *why* the flat root is deliberate — a future session (or reviewer) will relitigate the same question, exactly as happened four times today. Severity: medium (process debt, recurring wasted discussion). Mitigation: (f) items 1, 5–6.
5. **Untouched foreign working-tree changes.** 5 modified files (incl. `AGENTS.md`, `templates_templ.go`) appeared during the session from outside this session. Not mine, not investigated, not reverted — correct per the never-revert-others'-work rule, but it means this report's inventory is already a snapshot of a moving tree. Severity: low here, but see (g) Q2.

## e) WHAT WE SHOULD IMPROVE

1. **Verify numbers before asserting them.** A `wc -l` costs 5 seconds and would have made answer #3 factually correct on first delivery. Rule: any concrete figure quoted in an architectural answer comes from a fresh command, not from bytes÷guess.
2. **Record architectural verdicts durably at decision time.** Chat verdicts evaporate; AGENTS.md/ADRs are the memory. I *offered* a decision record twice — offering is not recording. Default should be: gate-style verdict reached → short ADR written in the same session, unless the user says no.
3. **Read the affected configs before quoting refactor costs.** Cost lists built from memory silently miss pins (`.crushrc`, workflow path filters). Rule: enumerate cost from the files themselves.
4. **Q&A sessions still produce state.** This session's only durable product is this report. Answers that conclude "X is deliberate, Y is blocked by Z" are decisions — they belong in the project, not in scrollback.
5. **Concurrent-session hygiene.** File counts shifted mid-session; the auto-commit daemon had swept 7 files into HEAD (`2b4b123`) while 5 more sat modified. Any inventory-based answer must state its timestamp (done here) and `git status` (done here).

## f) Next Tasks (session-scoped, ranked)

*The user asked for "up to 50".* This session produced no code, so padding to 50 would mean fabricating work items — the honest list from this session's findings is **24 items**. Per the status-report skill: extra items are brainstorm/ROADMAP fuel, and (f) feeds `docs-health` HARVEST (deferred — user said WAIT).

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Decide: cosmetic `cmd/`+`internal/daemon/` move vs. status quo (blocking decision for #2–#12) | High | S | Decision |
| 2 | If approved: `git mv` 33 root source files + 72 test files → `internal/daemon/` (package name unchanged, same-package pins preserved) | High | M | Refactor |
| 3 | If approved: `git mv main.go` → `cmd/emeet-pixyd/main.go` | High | S | Refactor |
| 4 | Read `.github/workflows/*.yml` for root-path references (fuzz-list assert, nix jobs, website greps) and fix any hits **before** the move | High | S | Verification |
| 5 | Read `.crushrc` for path-pinned LSP/templ config; update if the move happens | Medium | S | Verification |
| 6 | `flake.nix`/`package.nix`: add `subPackages = ["cmd/emeet-pixyd"]`, confirm `vendorHash` unchanged | High | S | Build |
| 7 | Verify `nix build .#emeet-pixyd` green after the move | High | S | Verification |
| 8 | Full `GOWORK=off go test -race -count=1 ./...` after the move (lock-order + serialization pins must stay same-package green) | High | S | Verification |
| 9 | `templ generate` sanity check after the move (go.mod-pinned CLI, per AGENTS.md gotcha) | High | S | Verification |
| 10 | Update AGENTS.md architecture table + Commands section to new paths | High | M | Documentation |
| 11 | Grep `website/` + `docs/` for root-path file references; update | Medium | S | Documentation |
| 12 | Confirm pre-commit golangci hook (`staged *.go` filter) still covers moved paths | Medium | S | Verification |
| 13 | Write ADR: "flat root until a named second consumer exists" — records the gate verdict + single-main.go + sub-folder seam analysis | High | S | Documentation |
| 14 | Add AGENTS.md Gotcha: the extraction trigger (named second consumer fires go-modularize gate 2) so future sessions don't relitigate | Medium | S | Documentation |
| 15 | Reconcile the ADR/AGENTS.md updates with the foreign in-flight `AGENTS.md` modification (see g-Q2) before editing | High | S | Coordination |
| 16 | Run the full go-modularize 7-phase workflow → formal `docs/modularization/` HTML proposal (only if formal artifacts are wanted) | Low | M | Process |
| 17 | Sketch the god-struct decomposition (`hidTransport`/`streamServer`/`callDetector` types) as ROADMAP fuel — the real precondition for sub-packages | Medium | L | Design |
| 18 | Evaluate `deps.go` function-field DI → small interfaces, as decomposition prep | Medium | M | Design |
| 19 | Verify the current modified tree still builds in nix — the auto-commit gate only runs go build/vet; nix breakage surfaces late (AGENTS.md gotcha, 2026-11-05 placeholder hazard) | High | S | Verification |
| 20 | Check whether `templates_templ.go`'s concurrent modification is the buildflow-templ-version-drift gotcha (regenerate with go.mod-pinned CLI + `git diff`) | High | S | Verification |
| 21 | Confirm `git status` foreign changes are committed/landed before starting the move (never mix others' in-flight work into a refactor commit) | High | S | Coordination |
| 22 | HARVEST this (f) table into `TODO_LIST.md`/`ROADMAP.md` via docs-health — after the user releases the WAIT | Medium | S | Documentation |
| 23 | Baseline `golangci-lint run` green on the current tree before any move (clean-cache rule: `rm -rf ~/.cache/golangci-lint`) | Medium | S | Verification |
| 24 | Standup rule for future sessions: architectural Q&A verdicts get an ADR in the same session by default (see e-2) | Medium | S | Process |

## g) Three Questions I Cannot Figure Out Myself

1. **Aesthetic or structural?** Is the "too much code in root" complaint about *listing noise* (→ cosmetic `cmd/` + `internal/daemon/` move, ~2 hours, zero semantic change) or about *wanting enforced compile boundaries* (→ god-struct decomposition program, days, blocked on gate 2 anyway)? I cannot infer intent from the repo, and the two answers lead to completely different work.
2. **The 5 foreign modified files** (`AGENTS.md`, `static/style.css`, `templates_templ.go`, `website/package.json`, `website/src/styles/global.out.css`) — are they yours or another agent's in-flight work, and must this session hold off on AGENTS.md/website-adjacent edits until they land? Any (f) items touching those files (#10, #13–#15, #20) are blocked on this answer.
3. **ADR autonomy:** should future sessions write architectural verdicts (like today's layout gate) straight into `docs/adr/` without asking — default-yes — or always wait for explicit approval? AGENTS.md's memory rules say update proactively, but ADRs are heavier artifacts and I chose "offer" today instead of "write".

---

*Point-in-time snapshot — goes stale. Section (f) is HARVEST fuel for `docs-health` once the WAIT is lifted. Not committed manually (harness rule: no commit without explicit request); the auto-commit daemon picks this file up.*
