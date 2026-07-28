# Nix FOD Build Failure — Root-Cause Fix & Self-Review

**Date:** 2026-07-28 15:24 CEST
**Session scope:** Resolve `nix build` failure: `fixed-output derivations must not reference store paths: ... references '/nix/store/.../go-1.26.5'`.
**Outcome:** 🔴 → 🟢 Build green. Workaround landed. ~~Root cause fixed upstream.~~ **Upstream binary untracked (`c29a034`) but NO new version published — proxy still serves v0.5.0 with the binary, so the in-sandbox `replace` workaround is still active and TEMPORARY.** **Self-grade: B-** (see §d, §e).

---

## TL;DR

`go-branded-id@v0.5.0` (on the public Go proxy) ships a **committed compiled `namer` ELF binary** built without `-trimpath`, so it embeds `/nix/store/.../go-1.26.5`. When the `go-modules` fixed-output derivation (FOD) ran `go mod download` and copied that module's zip into `$out`, nix's FOD invariant forbids referencing store paths → build hard-failed.

I fixed it two ways: (1) stopped tracking the binary in the `go-branded-id` repo (commit `c29a034`), and (2) wired an **in-sandbox-only `replace`** into `emeet-pixyd`'s nix build so the poisoned module is never downloaded — leaving committed `go.mod`/`go.sum` canonical so GitHub Actions `go test` is unaffected. Build, lint, and test checks all pass.

---

## a) FULLY DONE ✅

| #   | Item                                         | Evidence                                                                                                                                                                                                                                                     |
| --- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Root-caused the FOD failure**              | Manually reproduced the FOD build steps; `grep -rl "go-1.26"` pinpointed the `namer` binary inside `go-branded-id@v0.5.0.zip`.                                                                                                                               |
| 2   | **Stopped tracking the binary upstream**     | `go-branded-id` commit `c29a034`: `git rm --cached namer` + `/namer` added to `.gitignore`.                                                                                                                                                                  |
| 3   | **Verified removing the binary is harmless** | `go vet ./...` + `go test ./...` pass in `go-branded-id` (the binary is regenerable from `cmd/namer/`).                                                                                                                                                      |
| 4   | **Designed a CI-safe workaround**            | In-sandbox `go mod edit -replace=...=fetchFromGitHub(v0.5.0, postFetch rm namer)`. Committed `go.mod`/`go.sum` stay canonical → `go test` against the real proxy is untouched. Verified `go mod download` skips the module entirely under a local `replace`. |
| 5   | **Landed the workaround in `emeet-pixyd`**   | `flake.nix` (`goBrandedSrc` + `replaceBrandedId` let-binding, passed to package + lint `preBuild`) and `package.nix` (new params + `preBuild` block).                                                                                                        |
| 6   | **Recomputed `vendorHash`**                  | `sha256-zawNYoJyvw9fGGBSLlIIltvij6gQ2si0MvJ1OgEEH70=` in both `package.nix` and the lint derivation.                                                                                                                                                         |
| 7   | **Verified all nix outputs**                 | `nix build .#emeet-pixyd`, `.#checks.x86_64-linux.lint`, `.#checks.x86_64-linux.test` all green; `nix flake check --no-build` passes; binary runs (`emeet-pixyd --version`).                                                                                 |
| 8   | **Documented the issue + follow-up**         | New "go-branded-id committed-binary workaround (TEMPORARY)" gotcha in `AGENTS.md`.                                                                                                                                                                           |

---

## b) PARTIALLY DONE 🟡

| #   | Item                               | What's missing                                                                                                                                                                                             |
| --- | ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Permanent upstream fix**         | Binary is untracked in `go-branded-id`, but **no new version tagged/published**. The proxy still serves v0.5.0 with the binary. The `emeet-pixyd` workaround is explicitly TEMPORARY until v0.5.1+ exists. |
| 2   | **Regression prevention**          | No CI guard exists that would catch a _future_ committed-binary-in-dependency regression (e.g., a new dep shipping a stray ELF). The fix is reactive, not preventive.                                      |
| 3   | **go-branded-id repo cleanliness** | `namer` is untracked but the stale binary still sits on disk (3.3 MB). Not harmful, just untidy.                                                                                                           |

---

## c) NOT STARTED ⬜

1. Tagging `go-branded-id` v0.5.1 and pushing (blocked — requires explicit push permission).
2. Removing the TEMPORARY `goBrandedSrc`/`replaceBrandedId` shims after the bump above.
3. Auditing **other** dependencies for committed binaries / nix-path leakage (only `go-branded-id` was checked).
4. Adding a `nix flake check` (full, with builds) step to whatever runs locally/CI so this class of failure is caught before it reaches a human.
5. Building a real NixOS system test that exercises `nixosModules.default` end-to-end (I only verified the overlay _delegates_ to the perSystem package, not a full system boot).

---

## d) TOTALLY FUCKED UP 💥 (honest self-critique)

1. **I wasted multiple cycles to a daemon I should have detected on the FIRST revert.** A `buildflow --fix --semantic` process was running concurrently and kept committing/reverting my edits. I saw "file modified since last read" errors 3+ times before I ran `pgrep` to find the culprit. **Should have:** the moment a file "reverted itself", immediately suspect a concurrent modifier and run `ps`/`pgrep`. Cost: ~4 wasted edit cycles.
2. **I left a TEMPORARY workaround and called the task "Done".** Per the project's own philosophy ("execute, don't delegate back"), I should have completed the permanent fix (tag + publish v0.5.1). I punted it to a "follow-up" note. The only reason this is defensible: pushing tags/remotes is blocked without explicit user approval. This belongs in §g, not in a TODO buried in AGENTS.md.
3. **`lib.fakeSha256` vs `lib.fakeHash` fumble.** I first used `lib.fakeSha256` (raw sha256, no SRI type prefix), which fails FOD evaluation with "hash does not include a type". `lib.fakeHash` is the correct SRI placeholder. Minor, but a sign I was iterating by trial instead of knowing the nixpkgs hash conventions cold.
4. **No test proves the workaround works _in isolation_.** I verified the nix outputs build, but there's no unit test asserting "the go-modules FOD does not reference the Go store path". If someone later removes the `replace` without bumping the module, the failure recurs silently until the next `nix build`.
5. **`postFetch = "rm -f \"$out/namer\""` is fragile.** It hardcodes the binary name. If `go-branded-id` ever adds another stray artifact, this silently won't catch it. A more robust strip would remove _all_ ELF executables at the repo root.

---

## e) WHAT WE SHOULD IMPROVE 🚀

1. **Detect concurrent file modifiers proactively.** Before a long editing session, glance at `ps` for buildflow/watch/daemon processes that touch the repo. (The auto-git-commit daemon is documented in AGENTS.md, but buildflow's `--fix --semantic` _content_ rewriting was a nastier surprise than plain auto-commits.)
2. **Prefer permanent fixes over workaround shims.** Every "TEMPORARY" in code is a debt. The correct end state here is a published `go-branded-id` version with no binary — the shim should not exist in 1 week.
3. **Add a "FOD reference scanner" CI step.** A trivial `nix build .#checks.x86_64-linux.test` in CI would have caught this before it reached a local developer. Confirm it's actually wired into GitHub Actions, not just declared in the flake.
4. **Stricter upstream hygiene in `go-branded-id`.** Add a pre-commit/CI check that rejects committed ELF binaries (e.g., `git ls-files | xargs file | grep ELF` → fail). Prevents recurrence at the source.
5. **Document the buildflow interference pattern in AGENTS.md.** Future sessions need to know: "if edits mysteriously revert, kill buildflow first." I documented the _fix_ but not the _operational lesson_ about the daemon.
6. **Use `lib.fakeHash` (SRI) everywhere, never `lib.fakeSha256`, for FOD placeholders.** Standardize.

---

## f) Top Next Tasks (impact-ordered)

| #   | Task                                                                                                                                                    | Impact                                | Effort |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- | ------ |
| 1   | **Tag & publish `go-branded-id` v0.5.1** (binary-free), bump `emeet-pixyd` go.mod, remove `goBrandedSrc`/`replaceBrandedId` shims, recompute vendorHash | 🔴 Critical — kills the debt          | S      |
| 2   | Add CI guard: fail if `git ls-files` contains ELF binaries (in `go-branded-id` and optionally here)                                                     | 🔴 Prevents recurrence                | S      |
| 3   | Confirm `nix flake check` (full build) actually runs in GitHub Actions `go-test.yml`                                                                    | 🔴 Catches this class of bug upstream | S      |
| 4   | Audit all transitive deps for committed binaries (generalize the `grep -rl nix/store` scan)                                                             | 🟠 Defensive                          | M      |
| 5   | Replace `postFetch rm namer` with "strip all root-level ELF files" (more robust)                                                                        | 🟠 Hardening                          | S      |
| 6   | Add a nix check that asserts the go-modules FOD references no store paths (regression test)                                                             | 🟠 Preventive                         | M      |
| 7   | Build a NixOS VM test for `nixosModules.default` (currently only unit-checked)                                                                          | 🟡 Confidence                         | L      |
| 8   | Document the "buildflow reverts edits" operational pattern in AGENTS.md Gotchas                                                                         | 🟡 Future-self aid                    | S      |
| 9   | Delete the stale `namer` binary from `go-branded-id` working tree (disk cleanup)                                                                        | 🟢 Tidy                               | S      |
| 10  | Standardize on `lib.fakeHash` for all FOD hash placeholders                                                                                             | 🟢 Consistency                        | S      |
| 11  | Add a `just`-free `nix run .#test` / `nix run .#lint` app (if not present) for ergonomic local checks                                                   | 🟢 DX                                 | S      |
| 12  | Sweep AGENTS.md for other "TEMPORARY" / "follow-up" markers and convert to tracked TODOs                                                                | 🟢 Hygiene                            | S      |
| 13  | Add a `vendorHash` mismatch CI annotation that prints the `got:` hash in the failure summary                                                            | 🟢 DX                                 | S      |
| 14  | Consider `nix-update`/`nvfetcher` automation for `goBrandedSrc` so it tracks upstream tags                                                              | 🟢 Maintenance                        | M      |
| 15  | Verify the website build (`website/flake.nix`) is unaffected by the go-modules change                                                                   | 🟡 Confirm no collateral              | S      |
| 16  | Run `GOEXPERIMENT=jsonv2 GOWORK=off go test -race -count=1 ./...` locally to confirm green post-fix                                                     | 🟡 Confirm parity with CI             | S      |
| 17  | Add a top-level `Makefile`-equivalent note in AGENTS.md: the single command to reproduce the FOD scan                                                   | 🟢 Onboarding                         | S      |
| 18  | Check whether `go-error-family` (same author) has the same committed-binary anti-pattern                                                                | 🟠 Defensive                          | S      |
| 19  | Pin the `fetchFromGitHub` rev via a comment linking to the upstream commit that removed the binary                                                      | 🟢 Traceability                       | S      |
| 20  | Review whether `proxyVendor = true` is still strictly needed now that the binary issue is gone                                                          | 🟡 Simplification                     | M      |
| 21  | Add `nix flake update` cadence / automation                                                                                                             | 🟢 Maintenance                        | M      |
| 22  | Surface the `buildVersion` ldflag injection in a `--version` smoke test in CI                                                                           | 🟢 Confidence                         | S      |
| 23  | Document the FOD invariant ("must not reference store paths") in AGENTS.md for future nix work                                                          | 🟢 Knowledge                          | S      |
| 24  | Consider switching `goBrandedSrc` to `builtins.fetchGit` with `shallow = true` for smaller fetches                                                      | 🟢 Perf                               | S      |
| 25  | Once v0.5.1 ships: remove this entire §d/§e burden by doing task #1                                                                                     | 🔴 Debt clearance                     | S      |

---

## g) Questions I CANNOT answer myself

1. **May I push to the `go-branded-id` remote?** Tagging `v0.5.1` and `git push --tags` would make the workaround permanent and let me delete the shim — but my rules forbid pushing without explicit approval. This is the single highest-impact unblock.
2. **Should I permanently disable the `buildflow --fix --semantic` daemon, or is it a deliberate part of your workflow?** It aggressively reverts in-flight edits during failed builds. If it's intentional, I'll adapt (work in a temp checkout, then copy in); if not, killing it would prevent the churn I hit this session.

---

_Report scoped strictly to this session's work. No unrelated research performed._

---

## Resolution (2026-07-28)

**Still TEMPORARY — the permanent fix never shipped.** Verified against current code:

- `go.mod` still requires `go-branded-id v0.5.0` (no v0.5.1 was ever tagged/published).
- `flake.nix` still defines `goBrandedSrc` + `replaceBrandedId` (lines ~77–94) and both derivations' `preBuild` still run the `go mod edit -replace`.
- `package.nix` still takes the `goBrandedSrc`/`replaceBrandedId` args.
- The `AGENTS.md` "go-branded-id committed-binary workaround (TEMPORARY)" gotcha is still present and accurate.

So §b.1 (permanent upstream fix), §c.1–c.5, and §f.1 (tag v0.5.1, bump go.mod, remove the shims) are **all still open**. This is the single highest-impact debt from this report and is tracked as the top item in `TODO_LIST.md` — it is **blocked on push permission** to the `go-branded-id` remote (§g.1).

The build/lint/test gates remain green with the workaround in place; the debt is the workaround's continued existence, not a correctness regression.
