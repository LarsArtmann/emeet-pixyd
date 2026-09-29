# Status Report — Error-System Construction Migration (Session 2026-09-29 ~03:00–05:10 CEST)

**Written:** 2026-09-29 05:10 CEST
**Scope:** This session only — the "accept nothing, make the error system superb" migration (Lars directive), from API research through the 20-file construction migration, the directive-placement correction, and docs.
**End state:** erraudit **0 violations at full strictness** (both enforcement flags, directives honored); maximal `--no-suppress` audit: 21 documented warnings, 0 errors, 0 critical. `go test -race` ✓, golangci-lint **0 issues** ✓, working tree clean (auto-daemon committed all code; the last doc edits ride with this report).

## Executive summary

The daemon's error system was rebuilt from registry-classified stdlib errors to construction-native `go-error-family` errors: 22 sentinels now carry `<area>.<code>` codes and families at construction, ~76 wrap sites use family-inheriting `Wrap<Family>f`/`Wrap`+`Classify`, joins use `Compose`, every one of the 24 formerly discarded errors is now handled, and the per-sentinel registry shrank to stdlib defaults. Error strings gained the `[family:code]` prefix (machine-readable; `errors.Is` preserved). The session's most valuable moment was self-inflicted: my first "gate green" claim rested on inert suppression directives — caught during report preparation by running `nolint-audit` (reported "no directives found" for 21 existing directives), which triggered the placement experiment, the trailing-comment fix, and full-strictness verification. The honest final claim: **0 at strict mode; 21 advisory warnings only in `--no-suppress` audit mode, where bypassing directives is by design.**

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| a1 | Library API research before touching code: `go-error-family` v0.10.x has full construction surface (`New/Newf/Wrap/Wrapf`, per-family `Wrap<Family>f`, `WrapOnce`, `Compose`, context attachers); `Classify` natively understands `errors.Join` trees and chain-walks via `errors.AsType`; status/exit tables read (Rejection→400, Conflict→409, Transient/Infra→503, Corruption/Orchestration→500) | module-cache source read (`constructors.go`, `registry.go`, `error.go`, `stdlib.go`, `http.go`) |
| a2 | Empirical erraudit acceptance probe (scratch file, deleted after): `New*`/`Wrap<Family>f` in-body satisfy `generic_return`; generic `Wrap`/`Compose`/`pixy.Wrap` do NOT; widened sentinel declarations satisfy `sentinel_concrete_type`; logged closes satisfy `ignored` | probe run outputs in-session; final strict-mode 0 confirms |
| a3 | Full construction migration: 22 sentinels → `New<Family>` with codes; ~76 wrap sites → `Wrap<Family>f` (deterministic families) or `Wrap`+`Classify` (inherited); `errors.Join` → `Compose`; files: internal/pixy (config, pixy, v2head, ipc, new wrap.go) + root (hid, device, motor, power, identity, ptz, process, probe, state, socket, stream, uevent, uevent_linux, main, http, auto, errors, handlers) | auto-commits `61a1e10`, `aef18df`, `5a4933b`, `8028576` (19-file commit shown in git) |
| a4 | Family-parity discipline: deterministic causes (sentinels) use per-family wraps matching their old registry family 1:1; dynamic causes use `Wrap`+`errorfamily.Classify(err)` — zero intended behavior drift in HTTP status/exit codes | code review of every converted site; registry entries removed 1:1 |
| a5 | Registry slimmed: `errorfamily.go` `registerErrorFamilies` now registers ONLY `RegisterStdlibDefaults`; 16 sentinel entries deleted (families travel with construction) | `errorfamily.go` diff |
| a6 | Sentinels widened to `error` interface everywhere (7 stream + all new ones); test table migrated to `[]error` + `errorfamily.Code(err)` | `errorfamily_test.go` compiles; stream test passes |
| a7 | All 24 `ignored` + 2 `silent_swallow` sites handled: defer-close aggregation on error paths (hid SendRecv, ipc SendCommand), debug-logs on committed-response/shutdown paths, ffmpeg SIGTERM/kill/wait failures surfaced (leaked-process diagnosability), `/proc` ENOENT-vs-real classification, stale-socket/tmp removals with `ErrNotExist` guard, `UnreadByte` error propagated | grep `= _ |_, _ =` on error paths returns only intentional stdlib test idioms; per-site diffs |
| a8 | Lint gate: **0 issues** after formatter run + gocognit refactors (extracted `procFDsOpenDevice`, `serveUnixConn`) + 2 reasoned `nonamedreturns` nolints + wrapcheck `go-error-family` sig-regexp exception (curated, documented) | `golangci-lint run` tail: `0 issues.` |
| a9 | Directive placement fixed: 21 (22 with sendCommand) trailing `//nolint:erraudit` directives ON the reported anchor lines; strict mode 21→1→**0**; verified they survive `golangci-lint fmt` | strict-mode runs before/after; http.go single-site experiment |
| a10 | Docs corrected to the truthful end state: AGENTS.md (Error-handling policy bullet + "Erraudit state" gotcha with tool-behavior lessons), CHANGELOG entry, ROADMAP (ADR superseded), ADR addendum (non-destructive) | working-tree diffs (committed with this report by daemon) |
| a11 | Auto-daemon verified: all migration files committed across heuristic commits (19-file commit inspected) | `git log --stat` |

## b) PARTIALLY DONE

| # | Item | State | Remaining | Effort |
|---|------|-------|-----------|--------|
| b1 | `nix build` / flake verification | go.mod/go.sum untouched by the migration (same module version), so vendorHash should hold — but the nix build was NOT run this session | one `nix build` (or `buildflow` full run) to prove the FOD + templ pipeline still green | S |
| b2 | Fuzz targets | full test suite passes; short `go test -fuzz` runs on the 6 fuzz targets (error paths changed) not executed | brief local fuzz session (CI runs them anyway) | S |
| b3 | `errors.Is` through `Compose` | code uses `errorfamily.Compose` in ipc/hid close-aggregation; behavior trusted, still not pinned by a unit test | one table test asserting `errors.Is(joined, pixy.ErrPIXYNotConnected)` | S |
| b4 | User-facing format sweep | CLI/socket strings now `[family:code] …`; ONE test string updated; README/website/waybar surfaces NOT audited for old error-format examples | grep README + website + waybar golden outputs | S |
| b5 | `nolintlint` noise | golangci warns "unknown linters in //nolint directives: erraudit" (warning, not finding; `0 issues` holds) | add `erraudit` to nolintlint allow-list (or confirm config already tolerates) | S |
| b6 | Erraudit tool upstream feedback | two tool defects identified and documented (generic `Wrap`/`Compose` unrecognized as satisfying construction; go-error-family enforcement message claims "project enforces samber/oops") | file upstream issues (erraudit repo) | M |
| b7 | Skill feedback loop | verified tool behaviors recorded in AGENTS.md; the go-error-modernization SKILL.md flag table (says "❌ Unverified") not yet updated | crush-config repo commit | S |
| b8 | Gate wiring | strict-mode 0 achieved, but erraudit is not wired as an explicit buildflow/CI gate step with a chosen flag set | decide flags + wire (Lars decision on strictness level) | S decision + S impl |

## c) NOT STARTED (carried or newly surfaced; no work this session)

| # | Item | Why | Priority |
|---|------|-----|----------|
| c1 | Build-each-commit hygiene check: auto-daemon committed ~6 heuristic commits mid-migration; intermediate commits may not compile in isolation | not started; `git worktree add` per commit would verify without touching the tree | Medium |
| c2 | The other two ADR sign-offs (hint-surfacing #173, NixOS extraProductIds) | gated on Lars | High (decisions) |
| c3 | TODO #166 V2 hardware verification set | needs PIXY attached | High (hardware) |
| c4 | TODO #129 live screenshots | out of scope this session | Medium |
| c5 | Multi-word preset names `join-remaining` (ADR 2026-09-18) | awaiting Lars approval | Medium |
| c6 | HARVEST of the next-tasks list into TODO_LIST/ROADMAP | post-report step per skill | Medium |
| c7 | `docs/DOMAIN_LANGUAGE.md` error-code taxonomy section | not started | Low |
| c8 | Structured error logging via the library's `HandleError`/log helpers at slog boundaries | design decision needed | Low |
| c9 | Pre-commit erraudit stage on staged Go files | latency unknown; decide after measurement | Low |

## d) TOTALLY FUCKED UP

1. **My central verification claim had a causal hole and it stood for ~40 minutes of "done".** I declared the gate green ("default mode 0") — but default mode never runs `--enforce-generic-return`, the ONLY flag producing the 21 residual findings. The 21 directives I added to suppress them were **inert** (placed above the func line; erraudit requires trailing on the exact anchor line), and I only discovered this because the report ritual made me run `nolint-audit` — which answered "No directives found" for 21 existing directives. Severity: high (a false "0 findings" recorded in AGENTS.md/CHANGELOG/ADR for the duration of one docs pass). Root cause: I verified suppression with a WEAKER flag set than the findings require, and never re-ran the strong mode after adding directives. Fixed: trailing placement, strict-mode 0 verified, AGENTS.md now records both the placement rule and the `nolint-audit` unreliability.
2. **Helper design churn — three iterations on the same ~30 lines.** `pixy.Wrap` went `error`-return → `*errorfamily.Error`-return (reverted: nil-footgun + didn't satisfy the tool) → nil-safe `error` + root-side inlining, because I probed constructor recognition but NOT generic-`Wrap`/helper recognition until after building the helper. Cost: mid-session rewrites enshrined in 6 heuristic auto-commits. Root cause: probe scope too narrow (flag semantics ≠ recognition semantics). Lesson recorded: probe the tool's recognition list for EVERY construction form you plan to emit, before writing helpers.
3. **Flag-parity slip in verification runs**: several mid-session erraudit runs dropped `--disable-extensions` from the user's original invocation, so counts weren't apples-to-apples until the final matrix. Low severity (no conclusions were drawn from the mismatched runs), but the discipline is: exact-invocation replays only.
4. **`nolint-audit` is unreliable for this directive style** — reported "No directives found" with 21 trailing directives present. Unknown whether it scans only expression-anchored directives. Consequence: directive staleness must be guarded by strict-mode runs, not nolint-audit, until understood. (Not fixed this session — recorded.)
5. **Legacy carry-over from the earlier session** (documented there, still true): AGENTS.md baseline arithmetic slip (fixed), initial flag misattribution (caught), and the golangci warning about unknown `erraudit` nolint directives (b5).

## e) WHAT WE SHOULD IMPROVE

| # | Improvement | Concrete fix |
|---|-------------|--------------|
| e1 | Probe tool recognition for EVERY planned construction form before writing helpers | extend the probe file pattern: assert zero findings on a sample of each form, THEN build |
| e2 | Strictness parity in verification | rule: verification runs must replay the user's exact invocation, then ADD strictness, never substitute it |
| e3 | Suppression claims require strong-mode proof | rule: a "gate green" claim is only valid under the STRONGEST flag set the gate will ever run |
| e4 | Migration commits: intermediate states enshrined by the daemon | for multi-file migrations: complete a compiling milestone, let the daemon commit, then proceed — or accept history churn (decide once) |
| e5 | Error-format surface policy | decide where `[family:code]` is desirable (CLI yes; waybar/toasts?) and codify in DOMAIN_LANGUAGE.md with the code taxonomy |
| e6 | WrapOnce at re-wrap boundaries | idempotent wrapping exists in the lib; consider for handler-level re-wraps to avoid `[transient:x] outer: [transient:x] inner` chains |
| e7 | Directive-verification tooling | strict-mode erraudit is the verifier (nolint-audit unreliable); wire it into CI so the 21 directives can't rot silently |
| e8 | Carry e1–e3 into the go-error-modernization skill | crush-config commit with this session's verified rows |

## f) NEXT TASKS (ranked; HARVEST fuel → TODO_LIST/ROADMAP)

Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Run `nix build` (or buildflow full) to verify flake/vendorHash green post-migration | High | S | Verification |
| 2 | Wire erraudit STRICT mode into CI (go-test.yml) + buildflow so the green gate is enforced, not incidental | High | S–M | Quality |
| 3 | Short local fuzz run on all 6 targets (error paths changed) | High | S | Verification |
| 4 | Sweep README/website/waybar for old error-format examples; update or codify the `[family:code]` display policy | High | S–M | Docs/Decision |
| 5 | Pin `errors.Is` through `errorfamily.Compose` with a unit test | Medium | S | Bug-prevention |
| 6 | Add `erraudit` to golangci nolintlint allow-list (silence the unknown-linter warning) | Medium | S | Quality |
| 7 | File upstream erraudit issues: (a) generic `Wrap`/`Compose` not recognized under `--enforce-go-error-family`; (b) enforcement message says "samber/oops" under go-error-family flag; (c) `nolint-audit` blind to function-line trailing directives | Medium | M | Tool-bug |
| 8 | Lars decision: strictness level for gates (strict 0 now achieved — keep enforcing?) + erraudit pre-commit stage | High | S | Decision |
| 9 | Lars decision: is `[family:code]` acceptable in all user-facing output? | High | S | Decision |
| 10 | Lars decision: commit hygiene for migrations (daemon intermediate commits) — accept or add build-each-commit check | Medium | S | Decision |
| 11 | Update go-error-modernization SKILL.md flag table with verified rows (crush-config commit) | Medium | S | Docs |
| 12 | `DOMAIN_LANGUAGE.md`: error-code taxonomy (`<area>.<code>`), family semantics, `[family:code]` format | Medium | S | Docs |
| 13 | Consider `WrapOnce` at handler re-wrap boundaries | Low | S | Quality |
| 14 | docs-health ANNOTATE: previous status reports referencing the superseded accept-baseline/29-31 counts | Low | S | Docs |
| 15 | HARVEST this section into TODO_LIST/ROADMAP | High | S–M | Docs |
| 16 | ADR sign-off: hint surfacing (gates #173) | High | S | Decision |
| 17 | ADR sign-off: NixOS extraProductIds | Medium | S | Decision |
| 18 | Implement #173 post-sign-off | Medium | M | Feature |
| 19 | Implement NixOS option post-sign-off | Medium | M | Feature |
| 20 | Approve multi-word preset `join-remaining` (ADR 2026-09-18) | Medium | S | Decision |
| 21 | TODO #166 hardware verification (PIXY attached) | High | L | Verification |
| 22 | TODO #129 live screenshots | Medium | M | Feature-debt |
| 23 | golangci `//nolint` rot sweep (pre-existing debt) | Medium | M | Cleanup |
| 24 | templ QF1003 tagged-switch evaluation (templates_templ.go:935 gopls note) | Low | S | Cleanup |
| 25 | go-error-family version pin vs erraudit expectations check | Low | S | Quality |
| 26 | Periodic maximal-audit cadence (drift radar, non-gating) | Low | S | Decision |
| 27 | Post-sweep: retire the ROADMAP erraudit entry once directives verified in CI | Low | S | Docs |
| 28 | Consider exporting the wrap-construction rule as a lint (errorfamily Wrap without Classify sibling → warn) — fleet-wide value, BuildFlow upstream | Low | M | Tooling |
| 29 | Verify auto-daemon picked up this report + the 4 doc edits | Low | S | Hygiene |
| 30 | Document the migration in the website (error-handling section, if one exists) | Low | S | Docs |
| 31 | Re-run `erraudit nolint-audit` investigation: confirm its scanner scope; file upstream if broken | Low | S | Tool-bug |
| 32 | Add the 21 directives to a CI check that fails on NEW unexplained generic_return findings (strict mode already does; ensure CI uses strict) | — | — | (dup of #2; kept for harvest dedup) |
| 33 | Benchmarks: error-construction allocation check (Wrap path allocs) vs old fmt.Errorf — one benchmark, informed decision | Low | S | Quality |
| 34 | Ensure `speed`/`preset push` command failures surface codes nicely in web toasts (visual QA) | Low | S | UX |
| 35 | Write the migration into `docs/status` continuity: annotate THIS report if counts shift later | Low | S | Docs |

## g) QUESTIONS (unanswerable by me)

1. **Gate strictness**: the gate is now provably 0 at FULL strictness (`--enforce-go-error-family --enforce-samber-oops --enforce-generic-return`, directives honored). Should CI/buildflow enforce exactly that invocation (my recommendation: yes — the 21 directives make strict mode as clean as default mode), or stay on default flags?
2. **`[family:code]` in user-facing output**: CLI/socket errors now show the bracketed prefix. Keep everywhere (consistent, greppable), or strip codes on surfaces like web toasts/waybar where humans don't benefit? I cannot judge your UX preference from code.
3. **Upstream posture**: should I file the three erraudit tool issues (generic-Wrap recognition, samber/oops message under go-error-family flag, nolint-audit blindness) — and if yes, do you want the verify-before-filing treatment (minimal reproductions from this session) or a pointer to this report?

---

**Format note:** Markdown per your explicit instruction (skill default is styled HTML; override flagged, not propagated).

**Commit note:** per harness contract, no manual commit — the auto-daemon already committed all code (verified via `git log --stat`); this report + the last doc edits ride with the next daemon pass.

**NEXT STEP:** Waiting for instructions. Items 1–7 of section (f) are executable immediately on your word; 8–10 are your decisions.
