# Full Plan Execution Status — 2026-10-31 18:15

**Scope:** Executing the entire `docs/planning/2026-10-31_14-33_SUPERB-v05-release-first-pareto-plan.md` (T01–T27 + F001–F104). Gates were opened by Lars's blanket "GET SHIT DONE — the WHOLE TODO LIST". Hardware-gated items verified against the actual bus.

**Session window:** ~14:30–18:15. Everything below is first-hand verified unless marked otherwise.

---

## a) Done (verified green: tests + lint + CI where applicable, committed + pushed)

| # | What | Proof |
|---|------|-------|
| **T01** | **v0.5.0 released.** CHANGELOG frozen `[Unreleased]→[0.5.0] - 2026-10-31`; website changelog retitled + release summary; quickshell.mdx version example bumped; website.yml pre-deploy grep retargeted to `v0.5.0 (2026-10-31)` (stale-deploy sentinel rule). Local gates: `go test -race -count=1 ./...` green, golangci-lint 0 issues, `nix build` green. CI green on release commit `696e977` (Go tests 7m18s, Nix, Website incl. deploy). Annotated tag `v0.5.0` on `696e977`, pushed. GitHub release published with curated notes. `nix flake metadata github:…/v0.5.0` resolves rev `696e977`; `go list -m -versions` shows v0.5.0 (proxy indexed). TODO #155 DONE. | tag, release URL, flake metadata, proxy listing |
| **T02** | **Subprocess timeouts (#190).** `commander.go` const block (v4l2/wpctl/notify/ffmpeg-first-frame, all 5s); `v4l2Set` + `parsePTZValues` wrapped; `findPixySource`/`setDefaultSource`/`notifyCmd` wrapped; MJPEG stream bounded-first-frame (kill-on-expiry unblocks the pipe read; new `stream.no_first_frame` Transient sentinel). Deadline errors classify Transient via the existing stdlib registry entry. Commit `c7eb946`, pushed. | commit + tests |
| **T03** | **Hung-subprocess pinning tests.** `hangCommander` double (blocks until ctx done); `TestV4L2Set_HungCommanderIsBounded` pins the REAL 5s bound + Transient + `v4l2Mu` re-acquirable; `TestParsePTZValues_HungCommanderIsBounded` (parent-deadline propagation); `TestAudioAndNotifyPaths_HungCommanderIsBounded`; `TestStream_SilentFFmpegBoundedAndHealthyStarts` (fake ffmpeg via PATH: silent → 503 at 5.00s, healthy first frame → setup ok). All pass under `-race`. | test run output |
| **T04** | **Fail-fast socket listen (#191a).** `listenUnix` split: `newUnixListener` (synchronous bind, startup errors returned) + `acceptUnixLoop` (retry-forever). `Run()` binds BEFORE `READY=1`, returns error → `main` exits via `errorfamily.ExitCode`; `STOPPING=1` sent. `socket_test.go` helper simplified (no ready-poll loop). `failfast_test.go` pins unlistenable-path → error + non-zero exit. CHANGELOG + AGENTS contracts written. Commit `c7f98a9`, pushed. | commit + tests |
| **T05** | **Degraded health dimension (#191b).** `degraded.go` (mark/clear/sorted-snapshot under `d.mu`; reason keys `web-server`, `socket-accept`); Daemon field; `statusSnapshot`/`webStatus`/`healthResponse`/`statusResponse` plumbing (additive `degraded` + `reasons`, status codes unchanged); web listener death marks degraded; accept-loop ceiling (10 consecutive @ 100ms retry — no busy-spin, never fatal, healthy accept clears); web panel banner OUTSIDE `#status-panel` (morph-safe) + CSS; docs: monitoring.mdx reason table + alerting semantics, quickshell.mdx field rows. Tests: contract pin (absent healthy / present + sorted / status mirrors health), banner render, flaky-accept-storm recovery. `templ generate` with devShell templ; `nix fmt`; lint 0; full suite green. Commit `e0c2f99`, pushed. TODO #191 DONE. | commit + tests |
| Recon | **PIXY confirmed NOT attached** (no `328f` USB device, `/dev/v4l2/by-id/` empty, only OBS Cam on video1) → T08/T09/T10/T27 are hardware-blocked, correctly untouched. **pma located and read** (`/etc/systemd/system/projects-management-automation.service`, `pma-daemon/buildgate.go`, upstream NixOS module): the #176 gate ALREADY EXISTS upstream as `requireCleanBuild`/`requireCleanVet` (hold-not-discard, GOWORK=off, cooldowns, go-missing fallback) — enabling is config, not code. | lsusb, source reads |

Pushed emeet-pixyd master: `696e977` (release prep) → `c7eb946` (T02/03) → `c7f98a9` (T04) → `e0c2f99` (T05). All CI runs green.

## b) Partial (work exists, not fully landed/active)

| # | State | What remains |
|---|-------|--------------|
| **T06** pma green gate (#176) | SystemNix wiring **done and eval-verified** (`requireCleanBuild=true`, `requireCleanVet=true`, plus `go_1_27` via `mkBefore` on the service PATH — the module's gate PATH carries nixpkgs `pkgs.go` = 1.26.8 while emeet-pixyd's go.mod needs ≥1.27.1; without this the gate would hold emeet-pixyd commits on the toolchain, not the tree). **Commit landed only as a pma heuristic auto-commit** (`d0862d746`, see d) — content safe, message lost. | ① Recover a proper commit message (soft-reset recommit or fold into the runbook commit). ② **Activation requires a NixOS switch** — pma has no config hot-reload and I cannot run systemctl/nixos-rebuild from this sandbox. ③ F032 dry-run (broken file → held commit) only possible AFTER activation. ④ F033 AGENTS.md note in emeet-pixyd not yet written. |
| **T07** SystemNix port wiring (#183) | `hardware.emeet-pixy.port = ports.emeet-pixyd;` wired in `configuration.nix` (was daemon-default by convention), **eval-verified** (`nix eval …hardware.emeet-pixy.port` → 8090). Same pma-swept commit as T06. F034 (registry entry) turned out to already exist (`lib/ports.nix:75`). | Same commit-message recovery; **F036 lock bump** (float emeet-pixyd input + `scripts/float-register.sh` registry regen + toplevel proof) NOT started; **F037 runbook note** not written; activation on next switch. |

## c) Not started (all remaining plan items)

- **T11** #173 unsupported-hint surfacing (ADR Option B ready — blanket go received; not yet executed) + **T12** #174 hint-path tests
- **T13** #179 ValidatePresetName in web preset action · **T14** #184 port-option docs · **T15** #185 website CI proof
- **T16** #181 nolint sweep chunk 2 · **T17** #194 observability (breaker gauge, auto-errors counter, ext-deps→degraded, accept ceiling metric)
- **T18** #193 watchdog invariant comment · **T19** #192 trapReasserts drain · **T20** #186 vmTest port pin · **T21** #187 guide cross-links · **T22** #188 aarch64 eval · **T23** #189 lockstep comments
- **T24** #175 SEO pass · **T25** session-hygiene pack (F092–F097) · **T26** #148 upstream innoextract PR · **#156** branch protection (will attempt via `gh api` despite plan's "web-UI only" claim — it IS API-able)
- **T27** #172 shared V2 helper — gated on #138/#166 (hardware), stays closed
- **T08/T09/T10** — hardware-gated (PIXY absent), correctly deferred
- Blocked-by-design rows: #154 (external @zutto), #182 (on #166), decision queue #116/#123/extraProductIds/erraudit-strictness (Lars triage)

## d) Fucked up (honest ledger)

1. **SystemNix commit race (LIVE loose end):** my detailed commit and pma's auto-commit collided — `fatal: cannot lock ref 'HEAD'`. pma won; my two file edits + the `nix fmt` change landed as heuristic commit `d0862d746` (content intact, message gone). Root cause: I staged + ran the full pre-commit gauntlet (gitleaks, fmt) in a repo pma was actively sweeping, leaving a wide race window. Fix pattern going forward: in pma-swept repos, `git add` + `git commit` back-to-back with `--no-verify` for doc-only changes, and re-check HEAD freshness right before committing.
2. **emeet-pixyd 427ce02 partial sweep** (recovered): pma committed the test file alone mid-work; squashed via `git reset --soft` into the proper commit. Cost: one re-commit cycle.
3. **Test-first-draft bugs:** stray pseudo-code note written into `subprocess_timeout_test.go` (caught by diagnostics before any test run); wrong module path case (`larsartmann` vs `LarsArtmann` — one failed build); `net/http` unused import + nonexistent `readAll` helper in `degraded_test.go`. All fixed; none reached a commit.
4. **Conceptual test-design error (caught by test failure):** first accept-ceiling test used two parallel listeners (failing + healthy) and asserted the mark clears — wrong model: the broken listener CORRECTLY keeps the daemon degraded. Rewrote as `flakyListener` (fails past the ceiling, then recovers) which pins the real production shape.
5. **Chased stale LSP ghosts briefly:** the templ LSP kept reporting `s.Degraded undefined` after `webStatus` was edited; `templ generate` + build + tests were green throughout. Restarted the LSP late; should have trusted the compiler immediately.

## e) Improvements (what I'd do differently / next)

1. **Atomic commits in pma-swept repos** (see d1) — stage and commit in one breath; the irony of racing the daemon I'm trying to gate is noted.
2. **Push SystemNix commits before starting downstream work** (float-register, runbook) to create a stable base.
3. **Trust `go build`/`templ generate` over LSP diagnostics** for generated-code files; restart LSPs at first sign of staleness.
4. **Eval-verify → fmt → commit** order held up well in SystemNix (both options eval-verified BEFORE the commit attempt) — keep it.
5. **The T05 stream probe discards frame 1** of every stream (bounded-first-frame wait consumes one JPEG). Invisible on an MJPEG stream (next frame ~33ms later) — documented tradeoff, revisit only if a client ever complains.
6. T06 revealed the plan's F029 assumption ("locate script/config, add gate hook point") was moot — the gate existed upstream, disabled. **Reading the owning repo before building anything saved ~90 minutes of duplicate work.** Keep doing that.

## f) Next up to 50 (priority order; ⛔ = needs Lars/go/hardware first)

1. Recover SystemNix detailed commit (soft-reset `d0862d746`+`216cb3771` if cleanly mine, recommit) or fold detail into the runbook commit
2. ⛔ Lars call: activate the pma gate + port wiring (needs a NixOS switch — `nix run .#deploy` / `nh os switch`)
3. ⛔ Lars call: emeet-pixyd input — float master tip (uniform `ref=master`) vs pin `v0.5.0` tag URL, then `nix flake update emeet-pixyd` + `scripts/float-register.sh` (toplevel proof, long — background)
4. F037 + T06 docs: runbook notes both sides (SystemNix docs/services/*.md; emeet-pixyd AGENTS.md git-section note per F033)
5. F032 dry-run the gate (broken file → held commit → WARN in journal → fix → commit lands) once activated
6. T13/F063–F064: `pixy.ValidatePresetName` + 400 path in `presetAction` (30m, ungated)
7. T25 quick hits: F092 TODO header Previous-format, F093 report inferred-claim markers, F095 SetDeadline de-infer, F081/T18 watchdog comment (10–30m each)
8. T19/F082–F083: `d.trapReasserts.Wait()` in `handleShutdown` + test (or won't-do row)
9. T17/F076–F080: breaker gauge, auto-errors counter, ext-deps→degraded feed, accept-ceiling metric + tests (rides on T05's degraded.go — ready)
10. T11/F054–F059: webStatus.UnsupportedHint + panel line + Waybar tooltip + templ regen + pinning tests
11. T12/F060–F062: probe/device hint branch tests + rate-limited log branch + PID-case table
12. T14/F065–F066: website config guide port section + module option description
13. T15/F067–F069: `pnpm install --frozen-lockfile` + `tsc --strict` local proof; confirm website.yml green on next website push
14. T16/F070–F075: nolint inventory → handlers.go → commands.go/device.go → rest → tests → full gates
15. T20/F084: vmTest port pin (second node or `nix eval` assertion)
16. T21/F085: metrics→monitoring, quickshell→monitoring cross-links
17. T22/F086: `nix flake check --no-build --all-systems` attempt; wire or record gap
18. T23/F087: lockstep comment pair (module port default ↔ `pixy.DefaultWebAddr`)
19. T24/F088–F091: SEO frontmatter audit, OG images, sitemap handling (NOTE: Google's sitemap ping endpoint is deprecated since 2023 — do NOT wire a dead ping; robots.txt sitemap reference instead), deploy + verify
20. #156 branch protection via `gh api -X PUT repos/…/branches/master/protection` (required status checks for PRs, include-admins OFF so direct pushes keep working)
21. ⛔ T26/F098–F099: upstream innoextract PR (needs explicit go — external, in Lars's name)
22. F094 rubric re-score + F096 health-layers D2 + F097 June-review audit (rest of T25)
23. Final verification sweep: full gates, TODO/CHANGELOG status writes, push, report
24. ⛔ T08–T10 hardware bundle at the next wired session (F038–F053 checklist ready in the plan)
25. ⛔ T27 shared V2 helper (after #138/#166)

## g) Questions (max 3, blocking or taste)

1. **Activate the SystemNix generation now?** T06+T07 are committed but inert until a switch (`pma` has no config hot-reload; I can't run systemctl/nixos-rebuild from here). Say go and I run SystemNix's deploy path (`nix run .#deploy`, which also runs float-register); say wait and it rides your next switch.
2. **emeet-pixyd input semantics:** float to master tip (uniform `?ref=master` convention, lock-float machinery native) or pin the `v0.5.0` tag URL (explicit version anchoring, never floats until you change it)? The lock bump + proof run either way.
3. **T26 external PR:** the plan gates the innoextract 6.6.1 upstream PR on your call. Blanket "whole list" received, but an upstream PR in your name is the one action I won't take on a shout — confirm and I'll run verify-before-filing + github-voice and send it.

---

**Bottom line:** 5 of 27 medium tasks fully shipped and pushed (release + the entire resilience batch), 2 landed-but-inert pending one switch, hardware-gated items correctly parked. Zero regressions in the shipped gates: every commit rode green tests, 0-issue lint, green CI. The one live blemish is the SystemNix commit-message race (content safe, recoverable).
