# Status Report: Pareto Plan Execution — Resumed Session (Interrupt #2)

**Date:** 2026-10-31 20:58
**Scope:** This report covers ONLY the resumed session (start ≈ 20:30, after Lars's "keep going until everything works" directive answered the 3 standing questions with an explicit continue). Prior context: `docs/status/2026-10-31_18-15_full-plan-execution-status.md` (T01–T05 done, T06/T07 partial). Plan: `docs/planning/2026-10-31_14-33_SUPERB-v05-release-first-pareto-plan.md`.
**Interpretation ruling applied:** explicit "continue" = the 3 gates stay CLOSED (deploy, input float/pin, upstream PR were never answered specifically); every other task is unblocked and in scope.

---

## a) FULLY DONE (verified green, pushed to origin/master)

| # | Item | Evidence |
| --- | --- | --- |
| 1 | **Baseline verified at resume** — full race suite green, LSP diagnostics confirmed stale | `go test -race` ok; build ok (LSP typecheck warnings were false) |
| 2 | **T06/T07 doc-side finish (F033, F037)** — SystemNix runbooks: pma build-gate section (hold-not-discard semantics, go_1_27 mkBefore trap, INERT-until-switch, journal WARN signature) + emeet-pixyd port-wiring section (registry-wired, deploy-pending, float/pin = Lars's open call, `version` field proof). Commit-message recovery for `d0862d746` ruled IMPOSSIBLE without touching a foreign change — the pma heuristic commit includes `inboxclean.nix` (NOT mine; safety rule: never revert others' changes) — detail folded into the runbook commit instead. emeet-pixyd AGENTS.md gained the gate note | SystemNix commit `7ca1bd708`; emeet-pixyd `a4050f2` (pushed) |
| 3 | **T13 preset-name 400 (F063–F064, #179)** — `presetAction` now runs `pixy.ValidatePresetName` on every route (load/delete/push), same 400 path as save; 2 new tests (garbage→400 incl. escaped-NUL; valid→200 unchanged). CHANGELOG + TODO #179 row rewritten (multi-word truncation explicitly left to the pending ADR) | tests green; `4bfa8ac` pushed (code itself landed via pma sweep `1ebf37a` — content verified identical) |
| 4 | **T19 trapReasserts drain (F082–F083, #192)** — `handleShutdown` waits on `d.trapReasserts` BEFORE the state save (goroutines are deliberately `WithoutCancel`; the drain guarantees no exit mid-HID-write and a settled-belief save); `TestHandleShutdown_DrainsInFlightTrapReassert` pins bounce+rewrite completed when shutdown returns | race green; `d25cc69` pushed |
| 5 | **T18 watchdog invariant (F081, #193)** — comment pair: `WatchdogSec` in `modules/nixos.nix` states the ≥10× PollInterval invariant; the `sdNotify("WATCHDOG=1")` site in `main.go` cross-references it | same commit |
| 6 | **T23 lockstep pair (F087, #189)** — verified ALREADY COMPLETE (module `modules/nixos.nix:27-28` ↔ `internal/pixy/pixy.go:22-25`, each names the other); TODO row marked DONE | `d25cc69` |
| 7 | **T17 observability (F076–F080, #194)** — `emeet_pixyd_breaker_state` gauge (recorded immediately on the 3rd strike; re-derived from the strike count on every poll tick via `updateMetrics` — architecture decision: folding the derivation into `updateMetrics` cost 1 nolint instead of 7 ctx-less call-site nolints), `emeet_pixyd_auto_errors_total` counter (op label: track/audio/source/privacy), `emeet_pixyd_socket_accept_failures_total`, `checkExternalDeps` → `extdep:<binary>` degraded reasons (tests use `noopCommandRunner` LookPath-success → hermetic). 4 new tests (`observability_test.go`); metrics.mdx tables updated INCLUDING the pre-existing stale `camera_state` semantics + missing `online` gauge (fixed on sight). `updateMetrics` grew a 3rd param (2 test callers updated). BREAKER GAUGE RESET SEMANTICS: derived-per-tick, not instant — documented in code + test | race+lint 0 issues; `ddd53b0` pushed; `nix build` green at that state |
| 8 | **T11 hint surfacing — CODE + TESTS complete (F054–F058)** — Option B of ADR 2026-09-28 implemented: `statusSnapshot`/`webStatus`/`statusResponse` gain `UnsupportedHint` (additive, omitempty); panel note INSIDE `#status-panel` (short clause via new `unsupportedHintShort` = text before first colon; full hint in `title` tooltip; new `.unsupported-hint` CSS, muted non-error styling); Waybar tooltip line + additive `unsupportedHint` JSON key. `templ generate` run via devShell (canonical), build + all 3 new tests green | pma swept the 7 code files into `63169fc` (content intact); `unsupported_hint_test.go` still untracked — see b) |

Gates run at each step: `GOWORK=off go test -race -count=1 ./...` + `golangci-lint run` (0 issues, `--fix` where style moved) — green before every push. Commits pushed: `a4050f2`, `4bfa8ac`, `d25cc69`, `ddd53b0` (+ pma heuristics between). CI runs on those pushes: not re-checked this session (all previous sessions green; no reason to expect otherwise).

## b) PARTIALLY DONE

- **T11 last mile (code done, docs/commit not):** `unsupported_hint_test.go` UNTRACKED (pma took only tracked files); quickshell.mdx field-table row for `unsupportedHint` NOT yet added; CHANGELOG entry + TODO #173 row NOT yet written; `nix build` not re-run after the templ regen (go build + tests green). ~10 min to close.
- **T12 (F060–F062):** research done (hint branches at `commands.go:188-228` via `deviceNotFoundResult`; `warnLimiter` at `ratelimit.go`; existing `TestResolveProductID`/`TestUnsupportedDeviceHint` in `internal/pixy/model_test.go` already cover the classification table — F062's gap is thin). Tests NOT written (~30 min).
- **SystemNix:** local commits `d0862d746` + `7ca1bd708` NOT pushed (no explicit instruction; pushing is reversible but cross-repo). Deploy/lock-bump remain gated on Lars Q1/Q2.

## c) NOT STARTED

T14 (port-option docs), T15 (website CI proof), T16 (nolint sweep chunk 2), T20 (vmTest port pin), T21 (guide cross-links), T22 (aarch64 eval), T24 (SEO pass), T25 (hygiene pack F092–F097), #156 (branch protection), final verification sweep + final report. Blocked-not-started (unchanged): T08/T09/T10 (PIXY absent), T27 (depends #138/#166), #154 (external), #182 (blocked on #166).

## d) TOTALLY FUCKED UP / WENT WRONG (honest ledger)

1. **pma swept my staged code 4× mid-gauntlet** (`1ebf37a`, `cbcc280`, `380e96f`/`1130c22`, `63169fc`) — detailed commit messages lost for those file sets each time; content verified identical every time; the intended message still landed on the doc-only commits. The gate preventing this class is itself INERT until the NixOS switch (its own activation is Q1).
2. **Edit-tool freshness races** — `golangci-lint --fix` reformatted files between my read and my edit → 5 failed multiedits (device.go, circuitbreaker.go, auto.go, socket.go, observability_test.go). Recovered via re-read/sed/python each time; ~15 min lost. Lesson: never span a `--fix` run with a pending edit.
3. **T13 first test table was wrong by construction** — slash/empty preset names 404 at the ServeMux `{name}` pattern (never reach the handler), raw `\x00` breaks `url.Parse`. Caught by reasoning before running; replaced with overlong + `%00`-escaped cases.
4. **First `counterValue` Fatalf'd on absent counters** — OTel exports only after first write; the test failed once; semantics changed to return 0 (documented).
5. **ENVIRONMENT: `/tmp` tmpfs hit 100% (48G)** — "No space left on device" killed builds mid-session. Root cause: long-lived caches owned by OTHER processes (freshmc 1.3G, vra-gocache 1.2G, e2run 1.8G, four tmp.* ≈1.8G each, go-build-cache 2.7G). Not mine to delete. Workaround: `GOTMPDIR=$HOME/.cache/gotmp` (+ TMPDIR for nix develop). **Still open** — anything defaulting to /tmp will keep failing.
6. **`updateMetrics` signature change broke the build once** (TestUpdateMetrics callers) — expected cost, fixed in the same commit.
7. One commit subject exceeded SystemNix's 72-char house limit — rejected by its hook; shortened and retried (hook worked as designed).

## e) IMPROVE (do better next time)

- Stage+commit in ONE bash invocation immediately after a green gate (the two times I did this, pma lost the race; the four times I lagged, it won).
- Re-grep exact current text before every edit in this repo (auto-fixers + pma mutate files under me); prefer `lsp_replace_symbol`/python for multi-file touch-ups after any `--fix`.
- `nix fmt` was NOT run after the last templ regen — run it in the T11 close-out to keep treefmt canonical before committing the test file.
- The metrics.mdx staleness (camera_state semantics wrong for weeks?) went unnoticed by every review — schedule a docs VERIFY pass (docs-health skill) after the plan completes.
- Consider a `GOTMPDIR` line in AGENTS.md Gotchas if /tmp pressure recurs.

## f) NEXT UP TO 50 (ordered; unblocked first)

1. Close T11: `git add unsupported_hint_test.go` + quickshell.mdx row + CHANGELOG + TODO #173 + `nix fmt` + full gates + commit/push
2. T12 F060: probe/device hint-output branch tests (`deviceNotFoundResult` precedence over generic not-found)
3. T12 F061: warnLimiter branch test (allow → suppress within interval, re-allow after; injectable `now`)
4. T12 F062: thin hint-case table (fixed-UVC vs unknown vs PIXY precedence precondition) — most already covered in model_test.go
5. T14 F065: website config guide `hardware.emeet-pixy.port` section + loopback rationale
6. T14 F066: `modules/nixos.nix` option description + monitoring-guide URL
7. T15 F067: `pnpm install --frozen-lockfile` in website/ (nix shell nodejs), record
8. T15 F068: `tsc --strict` gate, record
9. T15 F069: confirm website.yml green on the next website/** push (quickshell.mdx+metrics.mdx rides along)
10. T16 F070: nolint inventory (grep 107 directives → scratch table)
11. T16 F071: sweep handlers.go (~12 directives)
12. T16 F072: sweep commands.go + device.go
13. T16 F073: sweep remaining production files
14. T16 F074: sweep test files (verify-live only)
15. T16 F075: full lint + race green + CHANGELOG entry
16. T20 F084: vmTest default-port pin (second vmTest node or nix eval assertion)
17. T21 F085: cross-link metrics→monitoring, quickshell→monitoring guides
18. T22 F086: `nix flake check --no-build --all-systems`; wire into nix.yml or record gap
19. T24 F088: SEO frontmatter description audit (all pages)
20. T24 F089: astro-og-canvas OG images for docs routes
21. T24 F090/F091: robots.txt-reference approach (sitemap ping deliberately NOT wired — Google deprecated 2023) + deploy + verify live
22. T25 F092: TODO_LIST header `**Previous:**` demotion convention fix
23. T25 F093: arch report format-deviation note + `⚠ inferred` markers on the 2 inferred claims
24. T25 F094: read `assessment-rubric.md`, re-score report §6, annotate deltas
25. T25 F095: read `pixy.SetDeadline` body, de-infer/correct the CLI deadline claim
26. T25 F096: health-layers diagram (D2/mermaid) into architecture-understanding series
27. T25 F097: June-review 17-TODO exhaustive audit + addendum
28. #156: branch protection via `gh api` (required status checks, enforce_admins OFF so direct pushes keep working) — pending Q3
29. Final sweep: full gates (test/lint/nix build/nix fmt), TODO rows #176/#183 final states with activation caveats, final report doc, blocked-items list
30. Verify CI green on all pushed commits (`gh run list`)
31. Push SystemNix local commits (pending Q2)
32. Free /tmp or relocate offenders (pending Q1)
33. NixOS switch to activate pma gate + port wiring (pending Q1; then F032 dry-run: break a file, verify held commit, revert)
34. Float/pin emeet-pixyd input in SystemNix via float-register.sh (pending Q2)
35. T26: upstream innoextract 6.6.1 PR (pending Q3; verify-before-filing + github-voice skills first)
36. Hardware session when PIXY attached: T08 (F038–F044), T09 (F045–F050), T10 (F051–F053), T27 (F100–F104), closes #129/#138/#139/#140/#141/#150/#166/#182
37. #154: close issue #6 on @zutto confirmation (external)
38. Decision queue triage with Lars: #116/#123, extraProductIds, erraudit-strictness
39. Post-deploy website check: quickshell.mdx `unsupportedHint` row + metrics.mdx tables live
40. AGENTS.md: add GOTMPDIR gotcha if /tmp pressure recurs; update Unsupported-hint surfacing bullet after T11 lands

## g) UP TO 3 QUESTIONS

1. **/tmp is 100% full (48G tmpfs)** from caches owned by other processes of yours (freshmc, vra-gocache, e2run, four tmp.* dirs, go-build-cache). I will not delete others' files. Options: (a) you clean them, (b) leave them and I keep redirecting GOTMPDIR/TMPDIR, (c) point me at a sanctioned cleanup command. What's the call?
2. **SystemNix downstream activation (combines the two standing sub-questions):** may I (i) push the two local commits (`d0862d746` gate+port, `7ca1bd708` runbooks) and (ii) run `nix run .#deploy` there (float-register runs a long toplevel build proof)? And for the input bump that follows — float master tip or pin the `v0.5.0` tag URL? (Recommendation: push yes, deploy no — ride your next switch; pin the tag — SystemNix tracks a host service, not a library.)
3. **External/irreversible actions:** (i) explicit go for T26 — the upstream innoextract PR opened in your name (byte-verified extractor evidence; I'd run verify-before-filing + github-voice first)? (ii) #156 branch protection via `gh api` (required checks for PRs, `enforce_admins:false` so your direct pushes keep working) — apply it, or do you prefer the 2-minute web-UI route yourself?

---

**State on pause:** working tree has ONLY `unsupported_hint_test.go` untracked (T11 code already committed by pma as `63169fc`, tests green pre-sweep). All pushed emeet-pixyd commits passed local full gates. Awaiting instructions.
