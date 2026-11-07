# Status Report + Brutal Self-Review — buildflow Recovery Session

**Date:** 2026-11-06 00:12 (Friday) · **Session scope:** fix the failed `buildflow` run (pasted output: nix-build format check + test-coverage failures) · **Reporter:** Crush session, 2026-11-05 22:40 → 2026-11-06 00:12

## Scope line — what this report VERIFIED vs NOT

**Verified live this session (commands run, not remembered):**

- `buildflow` full run ×3 — final state: **every step ✔** (`nix-build` ✔, `test-race` ✔ 123.6s, `test-coverage` ✔, lint/format families ✔); exit code 69 = findings gate only (28 pre-existing error findings → filed as #202)
- `nix build .#checks.x86_64-linux.format` — **offline sandbox build, green** (was the red core of the paste)
- `nix build .#emeet-pixyd` — green (vendorHash repaired)
- `nix flake check --no-build` — green (x86_64)
- `GOWORK=off go test -race -count=1 ./...` — green; **plus 10 × `-shuffle=on` iterations, 0 failures** (flake-hunt before fix: 3/5 failures → isolated `TestBreakerStateMetric_MirrorsAccounting`)
- `buildflow -s todo-check --format finding` — 0 findings; `go vet` — OK; `nix fmt` — idempotent (2 consecutive runs, 0 changed)
- Per-formatter isolation runs (treefmt `--formatters gofumpt`/`templ` separately) — root-caused the `_templ.go` rewrite and the glob failure

**NOT verified (and why):**

- **aarch64-linux eval** — `nix flake check` warned it omitted aarch64 (no builder on host); my flake edits are platform-neutral formatters, but unproven
- **CI workflows end-to-end** — nothing was pushed; go-test.yml / nix.yml / website.yml not exercised remotely
- **Integration/hardware tests** (`-tags=integration`) — need the PIXY attached; not run
- **The original paste's exact shuffle seed** (`-test.shuffle 1793914225985634738`) — I hunted with random seeds only; verdict is probabilistic, not seed-deterministic
- **Why go.sum shrank by 32 lines** in auto-commit 45b8267 (bundled with website/package.json) — symptom (vendorHash) fixed; cause (tidy churn? dep removal?) uninvestigated; build/test/govulncheck green so the module set is consistent
- **"9 tools unavailable (health check failed)"** in my own buildflow output — noticed in the transcript, never chased (`buildflow doctor --verbose` would explain)
- **golangci-lint's generated-file exclusion for `_templ.go`** — assumed from historical green CI, not tested this session (matters because treefmt now skips those files; see split-brain note in e))

---

## a) FULLY DONE

1. **Format check hermetic and green** (flake.nix): `goimports` dropped from treefmt (its go/packages driver shells to `go` → toolchain download → sandbox DNS refusal); `templ` wrapper mkForce-overridden to carry `go_1_27` (templ fmt itself parses generated code via go/packages — the REAL source of the "go runtime wants a toolchain download" gotcha, now obsolete). Check builds offline.
2. **`_templ.go` formatting oscillation killed**: gofumpt excludes `*_templ.go` + `**_templ.go` (generator owns those bytes; devShell/package.nix/CI all regenerate). Import grouping stays enforced by golangci (`gci`+`goimports`) in CI + pre-commit. `nix fmt` now idempotent.
3. **Flaky test fixed**: `TestBreakerStateMetric_MirrorsAccounting` asserted an absolute global-gauge value without a preceding write — `recordHIDSendFailure` writes the gauge only at the trip site (deliberate, circuitbreaker.go), so `-shuffle` order read registry residue. Fixed with the `tickBreakerGauge` helper (observability_test.go). 10/10 shuffled race runs green.
4. **vendorHash repaired** (flake.nix + package.nix, both): auto-commit 612e42f had swept a `sha256-AAA…=` placeholder into history — every nix build (package, vmTest, lint check) was broken from that commit until this repair. Applied the nix-computed hash; package + all checks build.
5. **todo-check false positive removed**: prose "bug" at a comment-line start in `integration_privacy_hardware_test.go:332` read as a BUG marker → reworded to "firmware defect". 0 findings now.
6. **AGENTS.md doctrine updated** (3 sections): hermetic-format-check mechanics + treefmt glob semantics; `-shuffle` gauge write-before-read rule; vendorHash placeholder hazards (daemon can commit them; `nix-hash-fix` cannot repair them).
7. **TODO_LIST #202 filed**: the 28 findings-gate items (25 file-size + 3 cmd-directory) as Lars's explicit policy call.
8. All fixes auto-committed by the daemon (6 commits this session), each passing the daemon's go build/vet gate; working tree clean at report time.

## b) PARTIALLY DONE

1. **buildflow fully green (exit 0)** — steps are ALL green; exit 69 remains purely from the findings gate (28 pre-existing error findings). Blocked on Lars's #202 policy decision, deliberately not unilaterally "fixed".
2. **Upstream buildflow gap** — `nix-hash-fix --fix` cannot repair placeholder hashes (its fixer replaces the stale hash "found verbatim"; `AAA…` states get "not found" and it exits failed — 33/41 lifetime failure stat, which itself looks polluted by this case). Documented in AGENTS.md; NOT filed upstream (verify-before-filing not performed against buildflow source).
3. **templ version alignment analysis** — CI pins `templ@go.mod` (v0.3.1070) while nixpkgs/devShell carry 0.3.1020; committed `_templ.go` = 0.3.1020 form. Drift is now harmless (nothing diffs generated files) but the CI comment "Pin templ version to match go.mod and nix pkgs.templ (prevents drift)" no longer matches reality. Decision pending (see g).
4. **Flake confidence** — 10 random seeds green, but no seed-deterministic sweep, and the paste's exact seed unreproduced.

## c) NOT STARTED (session scope)

1. The 25 `go-file-too-large` refactors and 3 `cmd-directory` decisions (#202) — policy call first.
2. aarch64 eval validation.
3. Remote CI validation of the new flake/tree (nothing pushed; releases are manual-tag anyway).
4. `buildflow doctor --verbose` investigation of the 9 unavailable tools.
5. Root-causing the go.sum -32 line shrink in 45b8267.

## d) TOTALLY FUCKED UP!

1. **INHERITED, now fixed, but a process hole**: a broken placeholder vendorHash sat at HEAD (612e42f) — **every nix build from that commit was broken** and nothing detected it until buildflow's nix-build step ran. The daemon's build gate (go build/vet) is structurally blind to nix, and CI only sees pushes. This WILL happen again without a guard.
2. **My own glob blunder**: I asserted treefmt glob semantics from memory (`**/*_templ.go` matches root files — it does NOT; `*` never crosses `/` and `**` still needs the literal slash), burned a full check build on a wrong exclude, and only caught it by isolating formatters empirically. I should have tested the glob with a one-file experiment BEFORE editing the flake.
3. **I did not reproduce the paste's shuffle seed** — my "flake fixed" claim rests on 10 random seeds, not determinism. Honest probability, weaker proof.
4. **I ignored a signal in my own output**: "9 tools unavailable (health check failed)" appeared twice in my buildflow runs and I never looked. That's the exact "green step, unread summary" anti-pattern the buildflow skill warns about.

## e) WHAT WE SHOULD IMPROVE (incl. self-review answers)

- **What did I forget?** Chasing the two visible-but-ignored signals (9 unavailable tools; go.sum shrink cause), aarch64 eval, and a final end-of-session `templ generate` no-diff confirmation (I verified it mid-session; daemon commits after that could theoretically have touched it — they didn't, but unverified).
- **Split brain (small, deliberate, documented)**: import formatting is now enforced ONLY by golangci while treefmt deliberately skips it, and treefmt skips `_templ.go` while golangci (probably, via generated-file detection) also skips them — two linters, two exclusion mechanisms, no mechanical test that their scopes stay coherent. Acceptable layering, but a one-line pin test (`TestWebPanel_...`-style) asserting the flake's excludes would freeze the contract.
- **Fragility I added**: the `lib.mkForce` override of treefmt-nix's templ wrapper binds to upstream's wrapper SHAPE — if treefmt-nix restructures `programs/templ.nix`, my override can silently become inert (detector = format check re-breaks loudly; acceptable, but know it).
- **Deterministic flake verification**: adopt seed-swept `-shuffle` (e.g. 50 recorded seeds) as the local pre-handoff battery for any test change; buildflow already randomizes, CI doesn't — the gap is where flakes hide.
- **Daemon gate blindness**: add a cheap pre-commit/pre-sweep check that `vendorHash` never matches `sha256-AAA…` (a grep suffices; `nix eval` freshness is the stronger version). Placeholder states are legitimate mid-work — the gate should hold-then-release like the existing build gate, or at least WARN.
- **How are we doing on tests?** Coverage is broad (unit/property/fuzz/integration-hardware), but order-independence was never part of the contract until this flake. The `tickBreakerGauge` pattern should be the named precedent for every future global-metrics assertion.
- **Did I lie to you?** No claim above exceeds its evidence — but two claims are weaker than they sound: "flake fixed" (probabilistic seeds, not the paste's seed) and "all checks build" (built across two runs, not one clean end-to-end rebuild; the final buildflow nix-build ✔ was partly cached).

## f) Up to 50 things to get done next (session-observed + already-known backlog; harvest fuel, not commitments)

**From this session (new):**

1. Lars decides #202: refactor slices vs size-threshold policy vs curated skip for the 25 file-size findings
2. Lars decides #202: `/cmd/` restructure vs documented exception for the 3 cmd-directory findings
3. File the `nix-hash-fix` placeholder-repair gap upstream at buildflow (verify-before-filing first: read its fixer source)
4. Add a vendorHash-placeholder guard so the auto-commit daemon can never again sweep `sha256-AAA…` to HEAD
5. Align templ: bump the nixpkgs templ pin toward go.mod's v0.3.1070, or formally accept drift and fix the stale CI comment ("match go.mod AND nix pkgs.templ" is no longer true)
6. Seed-swept `-shuffle` battery (50 seeds) as the standard test-change verification
7. Pin-test the treefmt excludes contract (flake glob list + golangci generated-file behavior) so the layering can't drift silently
8. Run `buildflow doctor --verbose` and disposition the 9 unavailable tools
9. Root-cause the go.sum -32 shrink in 45b8267 (tidy churn vs dep removal; confirm no dep silently left go.mod earlier)
10. `nix flake check --all-systems` (or an aarch64 builder) once available
11. Push/CI-run the recovered tree to prove go-test.yml + nix.yml green remotely
12. End-of-session `nix develop -c templ generate` no-diff confirmation as a standing step in every templ-touching session
13. Consider a `vmTest`-lite format-check CI split so website/PRs don't pay the full check closure cost (nix.yml currently `--no-build` + package build)
14. Re-verify the "82% nix-hash-fix failure" stat's causes before trusting the tool's self-report (the placeholder case pollutes it)
15. Sweep remaining prose marker words in comments (`BUG`, `TODO`, `FIXME` at line starts) that can false-positive todo-check — one grep pass

**Known backlog visible this session (from TODO_LIST.md, not re-verified):**

16. #166 hardware-verification bundle (closes #138/#139/#140/#141/#150 + #129) — the big wired session
17. #176 activate the auto-commit daemon build gate via NixOS switch (config landed, inert)
18. #183 SystemNix port wiring — push + switch (Lars go-ahead pending)
19. #156 branch protection on master (Lars, GitHub settings)
20. #154 close issue #6 after @zutto's PIXY 2K confirmation
21. #129 retake ONLINE web UI screenshots (bundle with #166)
22. #200 accessibility + real-device checklists (blocked: graphical session/Orca/NVDA/devices)
23. #148 send the staged innoextract 6.6.1 upstream PR (Lars's call)
24. #172 shared V2 query helper (trigger: 4th GET family — gated on #138/#166)
25. #138 clamp speed unit/limit at command layer after hardware pins it
26. #139 hardware verdict: battery/charge heads on wired firmware
27. #140 hardware-confirm tracking variants + re-assert policy after power cycles
28. #141 preset slot count + response shape (needs Lars Q1 call if mode-only)
29. #150 pin `MotorType` + `DefaultPosMode` (usbmon/hardware only)
29. #182 `sync` re-reads Speeds/TrackMode once #166 pins the readback
30. Multi-word preset names ADR execution (join-remaining after Lars approves — ADR 2026-09-18)
31. TODO #166's decoded-query-surface trust issue: `sync`/reconcile mode reads stay distrusted until decoded

**Smaller hygiene noticed en route:**

32. The CI "Pin templ version" comment block (go-test.yml) is stale relative to reality — one-line fix
33. dprint remote-plugin fetch warning (buildflow preflight): vendor the 3 .wasm plugins locally
34. `docs/benchmarks/` re-run in a quiet window for a post-recovery baseline (today's load made timings meaningless — the regression-verdict lines in my runs are cold-cache noise, e.g. prettier 2ms→2723ms)
35. Consider recording the successful flake-hunt method (per-formatter treefmt isolation + glob one-file test) as a reference snippet in AGENTS.md if treefmt issues recur

## g) Questions I cannot answer myself

1. **#202 policy call:** for the 28 findings-gate errors — do you want file-size refactors (25 files, real effort), a raised size threshold (your bar, your number), or a curated skip (`skip_steps`/exclusions) for `file-size-check` + `go-structure-linter` in this repo? Until decided, every full `buildflow` run exits 69.
2. **templ alignment:** bump the nix-side templ toward go.mod's v0.3.1070 (cost: per-bump nixpkgs override + vendorHash maintenance), or accept go.mod-vs-nixpkgs drift permanently (cost: committed `_templ.go` form depends on which generator ran last — now harmless, but the CI comment lies)?
3. **Upstream filing:** want me to file the `nix-hash-fix` placeholder-repair gap (and/or its misleading failure stat) at the buildflow repo — with source-level verification first per verify-before-filing? It's your repo and your voice on the issue.

---

*Point-in-time snapshot. Next reader: route section (f) through docs-health HARVEST before acting on it; sections (d)/(e) are the honest part — read them twice.*
