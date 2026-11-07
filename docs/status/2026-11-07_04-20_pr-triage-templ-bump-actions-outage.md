# Status Report — PR Triage Session: templ bump, cache-busting PR, hash-revert PR, Actions outage

**2026-11-07 04:20 CET** · branch `master` @ `b65c143` (+ uncommitted: CHANGELOG entries for #19/#12, a stale treefmt-comment fix in flake.nix, this report — the auto-commit daemon sweeps them) · scope: THIS SESSION ONLY (PR triage mandate). No project-wide inventory was attempted; TODO_LIST remains the authority for pre-existing open work.

## Scope — what this report verified vs. did NOT verify

**Verified live this session (command or artifact):**

- PR states via `gh pr view` / `gh pr list`: #19 MERGED (squash, master `e1ddff7`), #12 MERGED (rebase, master `4142f66`), #18 CLOSED, open-PR queue empty.
- Full local CI-parity battery on the final master tree (`b65c143`): `go vet` + `go build` OK, `go test -race -count=1 ./...` green (both packages), `golangci-lint` **0 issues**, `govulncheck` **no vulnerabilities**, `nix build .#emeet-pixyd` green (real rebuild of the `b65c143` derivation), `nix flake check --no-build --all-systems` all checks passed, hermetic format check `nix build .#checks.x86_64-linux.format` green (on the merged tree, pre-merge tree needed the master-side treefmt rework to pass).
- Outage facts via API: `actions/runs?created=>2026-11-06` → `total_count: 0`; `commits/master/check-runs` → 2 (GitGuardian + CodeRabbit only); last run ever = 42129137801 (2026-11-05 23:25 UTC); workflows all `state: active`; repo `private: false`; githubstatus.com all-operational; sibling repos (cqrs-htmx, templ-components, go-error-family) had runs on Nov 6-7.
- All session commits present on `origin/master` (daemon swept `.crushrc` + `TODO_LIST.md` as `db7752c`, mine carried `AGENTS.md` as `b65c143`).
- Pinned templ LSP binary: `~/.cache/emeet-pixyd/templ-lsp/v0.3.1100/templ` installed, `templ version` → `v0.3.1100`, rc skip-if-cached logic verified, `bash -n .crushrc` clean.

**NOT verified (and why):**

- **GitHub Actions itself never ran any of this.** Every merge (#19, #12) was gated by the local battery only — no CI stamp exists on these commits. Root cause of the outage is UNDETERMINED (billing API needs the `user` scope my token lacks; the Actions-tab suspension banner is only visible in the web UI).
- **The templ LSP fix is unproven in a real session.** `.crushrc` loads at crush startup only; THIS session still shows the 42 stale `templ.ResolveAttributeValue` diagnostics and will keep showing them until the next crush start.
- **No hardware tests** — no PIXY-attached claims were made or tested (no `-tags=integration` run; none needed for the changed files).
- **Website** — untouched this session (no `website/**` changes in any commit); assumed unaffected, not re-verified.
- **CHANGELOG completeness pre-Oct-31** — I added entries only for MY two PRs; no audit of older entries was attempted.

---

## a) FULLY DONE

| # | Work | Evidence |
| - | ---- | -------- |
| a1 | **PR #19 (dependabot: templ 0.3.1070→0.3.1100) fixed AND merged.** The bump alone broke the nix build twice: (1) stale `vendorHash` — new hash `sha256-Lbk/4A571Ix/HB8tuRlhlXYXfSi5ck9zcxjQxJ/BUks=` set in BOTH flake.nix (lint check) and package.nix, proven by a successful module-fetch + full build; (2) FATAL version drift — the nix `templ generate` preBuild (nixpkgs 0.3.1020) emitted `templ.ResolveAttributeValue`, which the v0.3.1100 runtime removed → compile errors only inside the nix sandbox. Root-cause fix: preBuild regen removed from BOTH derivations (committed `_templ.go` is canonical; CI go-test.yml validates it; stale files now fail-fast at `nix build`); `_templ.go` regenerated with the pinned CLI; AGENTS.md gotchas rewritten (canonical-source doctrine, pinned-CLI regen command, drift lesson). | squash commit `e1ddff7`; nix build green before merge; full battery green |
| a2 | **PR #12 (@Alex-Craftsman: datastar.js cache busting) resolved AND merged.** Rebased onto the post-#19 master, conflict resolved (kept master's regenerated file + his one-line templ change), regenerated with 0.3.1100, full battery green, rebased branch pushed to his fork, merged with HIS commit authorship preserved (`Alex CRFMN <alx@crf.mn>` on master `4142f66`) + thank-you comment posted. | `gh pr view 12` MERGED; `git log -1 --format=%an` on master |
| a3 | **PR #18 (@zshzebra: revert of the placeholder-hash auto-commit) closed obsolete — after verifying, not assuming.** His diagnosis was correct (commit 612e42f zeroed both vendorHashes), but master already carries the repaired hash (c4be084, 2026-11-06) and `nix build` on master HEAD succeeded (the `result` symlink proved the `emeet-pixyd-ac1acac` derivation builds). Merging the revert would have restored a stale hash + reverted CSS. Closed with an evidence-carrying comment; his prevention idea captured as TODO #203. | close comment on #18; `result` → `emeet-pixyd-ac1acac…` store path |
| a4 | **GitHub Actions outage discovered, bounded, and documented as TODO #204.** Zero runs repo-wide since 2026-11-05 23:25 UTC despite: all three workflows `active`, public repo (free minutes), GitHub status fully operational, sibling repos running Actions fine, master AND PR pushes yielding nothing but external-app check-runs. Every merge this session was gated by the local battery instead. | API queries listed in Scope; TODO_LIST row #204 |
| a5 | **templ LSP drift root-caused and fixed.** Same 0.3.1020-generator vs 0.3.1100-runtime drift that broke nix also blinded the project LSP (42 false `undefined` diagnostics, deterministic — `lsp_restart` cannot fix it). `.crushrc` now installs a go.mod-pinned templ binary (version-grep → `~/.cache/emeet-pixyd/templ-lsp/<v>/templ`) and re-registers the LSP, mirroring the existing project-overrides-global pattern (golangci_lint_ls). Binary installed + verified; AGENTS.md gotcha extended. | `templ version` → `v0.3.1100`; `bash -n` clean |
| a6 | **TODO_LIST updated on origin:** #203 (placeholder-hash gate guard — the PR #18 contributor's suggestion, cross-repo SystemNix), #204 (Actions outage investigation), header Updated/Previous lines per file convention. Evidence path corrected to the real report file after a first draft cited a wrong path. | `git show origin/master:TODO_LIST.md` contains both rows |
| a7 | **Final master state green end-to-end:** vet/build/race/lint/govulncheck/nix-build/flake-check/format all pass on `b65c143`; PR queue empty; local branches cleaned (`pr-12-cachebust`, dependabot branch, `fork12` remote all removed). | commands in Scope |
| a8 | **CHANGELOG gap closed (during this self-review):** `[Unreleased]` gained a Changed entry (#19) and a Fixed entry (#12) matching the file's per-PR convention — both merged PRs had initially landed WITHOUT changelog entries (see d1). | CHANGELOG.md (uncommitted, daemon will sweep) |

## b) PARTIALLY DONE

| # | Work | Done | Missing |
| - | ---- | ---- | ------- |
| b1 | templ LSP fix (a5) | Config + binary landed and verified in isolation | Real-session proof (the 42 diagnostics must vanish at next crush start) — pending, not provable mid-session |
| b2 | Actions outage (a4) | Symptom fully bounded; account-level and GitHub-level causes ruled out with evidence | Root cause undetermined — needs the Actions-tab banner (Lars) or support; TODO #204 |
| b3 | Merge-without-CI policy | #19 and #12 merged safely on the local battery | No standing doctrine: nothing prevents (or prescribes) the same behavior for the next agent/session while CI is down; also a judgment call I made without asking |
| b4 | PR #12 attribution context | Authorship + my PR comment carry the rebase/0.3.1100 note | The custom merge body I drafted (rebase context, co-author trailer) was silently IGNORED — rebase merges keep the commit's own message; the drafted body never landed anywhere |

## c) NOT STARTED

| # | Item | Why it exists |
| - | ---- | ------------- |
| c1 | TODO #203 implementation (nix placeholder-hash check in the projects-management-automation build gate) | Cross-repo (SystemNix) — needs Lars's go-ahead; only documented |
| c2 | CI enforcement gap: `nix.yml` runs `flake check --no-build` (eval only) + package build — the `checks.format` and `checks.lint` derivations are NEVER built by CI. Noticed mid-session (it's why the lint-check preBuild removal was zero-risk), never fixed | Hermetic format/lint gates are local-only today |
| c3 | Named CI-parity script (`scripts/ci-parity.sh`) encoding the exact battery I ran manually | My battery was a remembered sequence, not a repeatable artifact |
| c4 | `gh auth refresh -s user` to unlock billing/actions APIs for the outage diagnosis | Token scope gap hit live this session |
| c5 | dependabot strategy config (`.github/dependabot.yml` squash/rebase settings or auto-merge rules) — the 1070→1100 conflict dance and merge-method friction repeat on every bump | Lars's policy call |
| c6 | Branch protection (#156, pre-existing) — directly relevant again: nothing gates master while CI is down AND no protection exists | Lars-only setting |

## d) TOTALLY FUCKED UP (brutal section — what I got wrong)

1. **FORGOT the CHANGELOG for both merged PRs.** The repo's convention (`[Unreleased]`, every PR logged — #195…#201 all have entries) was one `head -30 CHANGELOG.md` away the whole session. Caught only by this self-review; fixed immediately after (a8). This is exactly the "two-line fix queued into a backlog" anti-pattern the owner mandate warns about — except I didn't even queue it, I missed it.
2. **Wasted merge attempts on GitHub merge-method mechanics I could have predicted.** `gh pr merge 19 --rebase` failed with "This branch can't be rebased" because MY conflict resolution had created a merge commit on the PR branch — GitHub's rebase merge refuses PRs containing merge commits. Then on #12 I passed `--subject/--body` to the rebase merge without knowing they are silently ignored (rebase keeps the commit's own message). Two avoidable round-trips, one silently-dropped artifact (b4).
3. **Pushed to the WRONG FORK — twice.** I had both PRs' JSON in context (#12 head = Alex-Craftsman, #18 head = zshzebra) and still addressed the cache-busting branch push to `zshzebra/emeet-pixyd`. Two permission-denied round-trips, then two more on `--force-with-lease` ("stale info") before using the explicit-lease form with `headRefOid`. Four failed pushes for one operation.
4. **Let the concurrent auto-commit daemon stomp an in-progress merge.** I knew the daemon commits/resets continuously on this host (it had already swept a placeholder-hash state onto my branch earlier in the session — the exact hazard AGENTS.md documents) and still left a multi-minute window between `git merge` and commit. Its `git reset` killed my unresolved merge (reflog: `reset: moving to HEAD`). Nothing unrecoverable, but I then rewrote the resolution as ONE atomic scripted sequence — which is what I should have done first.
5. **Tool-fight churn on daemon-shared files:** three failed `edit` calls (mtime-invalidated by concurrent touches) before switching to scripted python edits; a `sed` that removed conflict markers but kept BOTH hash lines; a second `sed` whose pattern missed a `+` in the base64 hash; a python merge script with a duplicated-bullet concatenation bug (self-caught, but each bug cost a cycle).
6. **Read a passing bot check as "CI green" before noticing CI wasn't running at all.** My first `gh pr checks 19 --watch` showed GitGuardian/DontMergeMeYet passing and I nearly proceeded on that signal — the outage signature (zero workflow runs, stale timestamps) was visible in the first check-run listing had I looked at dates instead of names. ~15 minutes lost before the `gh run list` timestamp check exposed it.
7. **Minor noise:** a `sleep 60` probe got auto-backgrounded and I then queried a guessed shell ID that didn't exist; the hermetic-format-check failure initially sent me into a log crawl before re-reading AGENTS.md line 144, which already documents that exact sandbox artifact — the file I myself had edited hours earlier. RTFM-own-docs failure.

**Did I lie?** No. Every green claim in the session summary maps to a command executed in-session (listed in Scope). The two claims that need softening rather than retraction: "master end-to-end green" means the local battery — CI never validated any commit this session, and no hardware tests ran (none were claimed).

## e) WHAT WE SHOULD IMPROVE

1. **Make the merge gate mechanical, not remembered.** Encode the exact CI battery as `scripts/ci-parity.sh` (generate → vet → lint → govulncheck → race → nix build → flake check → format check) and cite its output in merge comments. A remembered sequence is a drift-prone ghost process.
2. **Atomic git sequences around the daemon — always.** Any merge/rebase + resolve + commit on this host must be one scripted bash call from the first attempt, not an interactive dance. The daemon WILL interleave; the session proved it twice.
3. **Read `headRepositoryOwner` before any fork push; use explicit `--force-with-lease=<branch>:<headRefOid>` in the first attempt.** No guessing remotes, no FETCH_HEAD baselines.
4. **Choose the GitHub merge method before pushing resolution:** a PR branch carrying a merge commit can only merge squash/merge; if linear history matters, resolve conflicts as a rebase or plan squash from the start — don't discover it at merge time.
5. **Trust timestamps, not check names.** Before treating any check as a gate signal, verify a workflow run EXISTS for this SHA and is fresh (outage signature: only external-app check-runs).
6. **CHANGELOG entry ships with the merge follow-ups** — same change-set, not "later".
7. **`gh pr view --json` the fork/lease/mergeability fields once, up front, into a shell variable** — the whole fork-push dance was avoidable with 5 seconds of JSON reading.
8. **Re-read the project's own gotchas before debugging a failure the docs may already document** (the format-check sandbox artifact cost a log crawl that AGENTS.md line 144 answers verbatim).

## f) NEXT (up to 50 — a brainstorm from THIS session's observations, not a commitment list; HARVEST with routing rigor: ★ = TODO_LIST-worthy, the rest ROADMAP/decision fuel)

1. ★ **#204 (exists): resolve the Actions outage** — Lars: Actions-tab banner check, then support ticket if silent. Everything else CI-related queues behind this.
2. ★ **Prove the templ LSP pin** at next crush start: the 42 `ResolveAttributeValue` diagnostics must be gone (if not: check whether project `lsp add` truly overrides global same-name registrations — assumption untested).
3. ★ **Add a checks-build step to `nix.yml`** (`nix build .#checks.x86_64-linux.{format,lint}`) — the hermetic format + lint gates exist but CI never builds them (c2).
4. ★ **#203 (exists): placeholder-hash check in the daemon gate** (blocked on Lars's SystemNix go-ahead).
5. ★ **`scripts/ci-parity.sh`** (c3) + cite it in merge comments during any future CI outage.
6. ★ **AGENTS.md one-liner:** rebase-merge refuses PRs containing merge commits; rebase-merge ignores `--subject/--body`; prefer squash for PR branches that carried a master merge — the session's three merge-method lessons belong in Gotchas.
7. **`gh auth refresh -s user`** for billing/actions API access during outage diagnosis (Lars's token).
8. **dependabot config decision** (`.github/dependabot.yml`): squash-commit style + rebase strategy to stop the per-bump conflict dance; auto-merge for green minor+patch once CI returns.
9. **Branch protection #156 (pre-existing):** require the three workflows on master once Actions is back — this session had zero enforcement and it would have been invisible.
10. **Pre-seed the pinned templ binary** via the devShell/flake so the first crush start after a templ bump doesn't pay a ~30s `go install` compile.
11. **Drop-condition note for `.crushrc`:** when nixpkgs templ ≥ go.mod's templ, remove the pinned-LSP block (the file already says this — add a nixpkgs version check or a TODO anchor so it actually happens).
12. **CHANGELOG drift audit:** sweep every post-v0.5.0 master commit (Oct 31 → now) for missing `[Unreleased]` entries — my two entries fixed the known gap, not necessarily the only one (docs-health HARVEST/VERIFY).
13. **Merge-without-CI doctrine decision (Lars):** freeze merges during Actions outages vs. accept the local battery as the gate. TODO #204 carries the evidence; the policy needs a ruling and a line in AGENTS.md if the battery becomes legitimate.
14. **MERGE_HEAD-aware daemon:** propose upstream (SystemNix) that the auto-commit daemon skip `git reset` when a merge is in progress — this session's lost-resolution incident (d4) generalizes to every multi-session host.
15. **`flake.nix` treefmt comment:** FOUND STALE during report writing — master's comment claimed "package.nix … regenerate them" but the preBuild removal made that false; fixed on sight (comment now says the nix derivations consume committed files and stale files fail-fast). Resolved, listed for the audit trail.
16. **`gh pr merge` wrapper discipline:** for external PRs, standardize on: fetch → rebase locally with atomic script → explicit-lease push → `--rebase` merge; write it as a one-page runbook or script so the next session doesn't rediscover the failure modes (d2/d3).
17. **Website CI unaffected-claim:** `website.yml` path filters mean this session's commits didn't deploy — when Actions returns, confirm the website job's skip logic behaved as assumed (I asserted it in the report without a run to prove it).
18. **govulncheck toolchain pin:** CI installs `govulncheck@latest` while I ran it via `go run` with a rebuilt toolchain after a version-mismatch warning — pin the version for local/CI parity.
19. **`.crushrc` gopls note:** gopls not_started in this session's `crush_info` — the templ diagnostics pane is templ-LSP-only; confirm next session that gopls picks up the regenerated file cleanly (expected yes: go build passes).
20. **Add the session's outage evidence (API queries) to TODO #204** — already done (row carries them); re-verify freshness when harvested.
21. **Consider `nix flake check` build-mode variant for CI** (`--no-build` today): even one CI builder for `checks.build` (= packages.default) would have caught the #19 sandbox failure BEFORE merge; today's nix.yml DOES `nix build` the package — verify it exercises the same derivation the local build did (it does: `checks.build = config.packages.default` — no action, recorded for completeness).
22. **Fuzz-target list assertion:** untouched this session (no new fuzz targets) — nothing to do; recorded so the CI assert's absence during the outage is known-acceptable.
23. **CHANGELOG entry style:** my two entries used "External contribution by @…" / co-author phrasing — confirm Lars wants external credit phrasing in CHANGELOG (I followed the PR-comment voice, not a documented CHANGELOG convention; the file has no external-credit precedent).
24. **AGENTS.md "Commands" section:** still says `nix fmt # treefmt: nixfmt/templ/gofumpt (goimports dropped, *_templ.go excluded)` — accurate post-merge; no action (verified during writing).
25. **Housekeeping:** `~/.cache/emeet-pixyd/templ-lsp/` accumulates one binary per templ version — trivial size, add a keep-latest prune only if versions churn often (ROADMAP fuel, low priority).

(Items 21-22 are recorded-as-fine; kept because they were actively checked this session and the answer "no action" is otherwise lost.)

## g) Questions I cannot figure out myself

1. **Merge policy during the Actions outage:** should agents keep merging on the local CI-parity battery (what I did for #19/#12), or freeze all non-urgent merges until you've looked at the Actions tab / raised support? I cannot see the web UI banner or your support-ticket state, and "the battery is equivalent to CI" is my judgment, not a mandate.
2. **Once Actions is back:** do you want dependabot minor+patch bumps to auto-merge on green CI (with branch protection from #156), or stay manual? And should the local battery replace/augment CI permanently for nix-file changes (CI still won't build `checks.format`/`checks.lint` — c2)?
3. **#203 scope:** am I allowed to open/change the projects-management-automation gate config in SystemNix myself (add the placeholder-hash rejection), or do you drive that repo and I only keep the TODO? Touching the gate that governs every commit on this host without your explicit go-ahead felt out of bounds.

---

*Awaiting instructions.*
