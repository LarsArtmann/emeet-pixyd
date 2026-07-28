# Status Report: go-error-family Adoption Verification & Fixes

**Date:** 2026-07-23 21:00
**Session Focus:** Verifying and fixing the go-error-family adoption work from a previous session
**Branch:** master (4 commits ahead of origin)

---

## a) FULLY DONE

### go-error-family adoption — core implementation

- **`errorfamily.go`**: Sentinel classification registration via `sync.Once`. 18 sentinels mapped (4 Infrastructure, 10 Rejection, 4 Transient). Calls `RegisterStdlibDefaults()`.
- **`errorfamily_test.go`**: 7 test functions covering all three families, wrapped error chains, stdlib defaults, stream errors, and unknown error default.
- **`stream.go`**: 7 typed stream errors via `errorfamily.NewInfrastructure()`. 7 `http.Error()` calls replaced with `errorfamily.HTTPStatus(err)`. 3 genuine bugs fixed (500→503 for infrastructure failures).
- **`handlers.go`**: Preset validation uses `errorfamily.HTTPStatus(err)` instead of hardcoded `http.StatusBadRequest`.
- **`main.go`**: `registerErrorFamilies()` called in `NewDaemon()`. Two `os.Exit(1)` replaced with `os.Exit(errorfamily.ExitCode(err))`.
- **`main_test.go`**: `registerErrorFamilies()` called in `newTestDaemon()` so tests bypassing `NewDaemon()` get sentinel classifications.

### Verification & fixes done THIS session

- **`go.mod` fixed**: `go-error-family` was incorrectly marked `// indirect`. Ran `go mod tidy` — moved to direct require block, version bumped v0.7.0 → v0.8.0.
- **`vendorHash` synced**: Both `flake.nix` and `package.nix` updated with correct hash for v0.8.0.
- **godoclint false positive fixed**: `.golangci.yml` exclusion rule for `main.go` — `templates_templ.go`'s generated comment block triggers cross-file "more than one godoc" analysis.
- **`integration_hardware_test.go` fixed**: Comment block separated from `package main` with blank line.
- **AGENTS.md updated**: Version v0.7.0 → v0.8.0, godoclint exclusion documented.

### Verification gates — all green

| Gate            | Command                                | Result                |
| --------------- | -------------------------------------- | --------------------- |
| Build           | `go build .`                           | PASS                  |
| Tests           | `go test -race -count=1 ./...`         | PASS (2 packages)     |
| Lint            | `golangci-lint run --timeout 5m ./...` | **0 issues**          |
| Nix build       | `nix build`                            | PASS                  |
| Nix flake check | `nix flake check`                      | **All checks passed** |
| Working tree    | `git status`                           | Clean                 |

### Planning document

- `docs/planning/2026-07-23_18-12_ADOPT-GO-ERROR-FAMILY.md` — Full Pareto analysis, implementation plan, anti-verschlimmbessern guardrails.

---

## b) PARTIALLY DONE

### go-error-family adoption scope

The adoption is **intentionally partial** — only HTTP status derivation and CLI exit codes. The following were deliberately NOT touched (documented as design decisions):

- HTMX action handlers (return 200+HTML toast for `outerHTML` swap)
- Circuit breaker in `device.go` (more sophisticated than binary `IsRetryable`)
- `HTTPHandler` middleware (writes JSON, but UI needs HTML)
- Inline validation strings already returning correct 400s

**Status:** Correct as scoped, but could be expanded further (see section e).

---

## c) NOT STARTED

- ~~**Push to origin** — 4 commits are local only, not pushed. Previous session claimed push but origin shows 4 commits behind HEAD.~~ DONE: pushed shortly after this report (`a2dce73..53007db`); HEAD is now well past this on `master`.
- **Error family adoption in `internal/pixy/`** — The `pixy` package has its own sentinel errors that could benefit from classification.
- **Error wrapping consistency audit** — Not all `fmt.Errorf` sites use `%w` wrapping consistently; some use `%v` which breaks `errors.Is` chains and therefore breaks `errorfamily.Classify()`.

---

## d) TOTALLY FUCKED UP

### Previous session left 3 silent problems

1. **`go.mod` had `go-error-family` as `// indirect`** — Direct imports marked indirect is a lint violation (`gomoddirectives` linter) and semantically wrong. The previous session's `go get` command didn't trigger `go mod tidy`. BuildFlow flagged this as a warning.
2. **godoclint CI blocker** — The previous session claimed "0 lint issues" but `nix flake check` actually FAILED on the lint derivation because of the godoclint cross-file false positive. This was a pre-existing issue that the previous session should have caught and fixed during verification. It would have blocked CI.
3. **`integration_hardware_test.go` comment placement** — Comment block directly above `package main` (no blank line) acted as an implicit package doc, contributing to the godoclint false positive. Minor but sloppy.

### Previous session claimed push that didn't stick

The context summary says "code is pushed to `origin/master`" but `git status` shows "ahead of origin/master by 4 commits." Either the push was lost or never happened.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Always run `go mod tidy` after adding dependencies** — Non-negotiable. `go get` alone leaves deps as `// indirect`.
2. **Always run `nix flake check`, not just `nix build`** — `nix build` only builds the package derivation. `nix flake check` also runs the lint derivation, which caught the godoclint failure.
3. **Don't trust session summaries** — This session proved the summary was wrong in 3 specific ways. Always verify from scratch.
4. **Clear golangci-lint cache before verifying** — Stale cache produced false "0 issues" in the previous session. The godoclint issue only appeared with fresh cache.

### Code improvements (specific to this work)

5. **`errorfamily.go` triggers `unused` LSP warnings** — `errorFamiliesRegistered` and `registerErrorFamilies()` show as unused in the LSP because the LSP doesn't see cross-file usage in package main. Consider a `//nolint` or accept as false positive.
6. **7 stream errors are package-level vars** — They use `errorfamily.NewInfrastructure()` at init time, which means they're registered in the library's global registry during package init. This works but couples init order across packages.
7. **`RegisterClassifications` uses `DefaultRegistry`** — Implicit global state. The library supports custom registries; using one would be more testable.
8. **No integration test for the actual HTTP status codes** — Tests verify classification but don't make an actual HTTP request and assert the response code.

---

## f) Up to 50 Things We Should Get Done Next

### High priority — correctness & CI

1. **Push 4 commits to origin/master**
2. **Pin GitHub Actions to commit SHAs** (BuildFlow flagged 10 actions using tag pins — security risk)
3. **Move `main.go` to `cmd/emeet-pixyd/main.go`** (BuildFlow critical finding — Go convention)
4. **Audit all `fmt.Errorf` sites for `%v` vs `%w`** — sites using `%v` break `errorfamily.Classify()` chains
5. **Add `.editorconfig`** (BuildFlow info finding)
6. **Add `## Installation` section to README.md** (BuildFlow info finding)

### go-error-family expansion

7. **Classify HTTP handler errors in `handlers.go`** — `handleHealth`, `handleSnapshot`, etc. still use hardcoded status codes in some paths
8. **Add `errorfamily.HTTPStatus()` to `handlePTZ`** error paths (currently return raw 400/500)
9. **Add error classification to `commands.go`** socket command error responses
10. **Classify `auto.go` errors** — `handleCallStart`/`handleCallEnd` accumulate errors in `errs` slice
11. **Classify `hid.go` errors** — HID read/write failures are currently untyped
12. **Classify `process.go` errors** — PipeWire/wpctl failures
13. **Classify `uevent.go` errors** — Netlink read failures
14. **Add a custom registry instead of `DefaultRegistry`** — for test isolation
15. **Add HTTP integration tests that assert actual response status codes** for stream/snapshot errors
16. **Expand `errorfamily_test.go`** — test `ExitCode()` for Rejection family (currently only Infrastructure tested for exit codes)
17. **Test wrapped transient errors** — verify `fmt.Errorf("ctx: %w", errNoHIDResponse)` still classifies as Transient

### Error handling improvements

18. **Audit all `os.Exit(1)` calls** — ensure all use `errorfamily.ExitCode(err)`
19. **Add structured error logging** with family classification in slog output
20. **Standardize error message format** across socket commands and HTTP handlers
21. **Add error context** to HID failures (device path, command bytes sent)
22. **Improve ffmpeg startup error messages** — distinguish "not found" from "failed to start"
23. **Add retry logic for Transient errors** in the auto-manage loop
24. **Add circuit breaker state to health endpoint** — expose consecutive failure count

### Testing improvements

25. **Add fuzz test for `errorfamily.Classify()`** with random error wrapping chains
26. **Add benchmark for `registerErrorFamilies()`** — verify `sync.Once` overhead is negligible
27. **Test `exitWithDaemonError`** with various error types
28. **Add test for stream concurrent access** — two simultaneous `/api/stream` requests
29. **Test PTZ readback error classification**
30. **Add table-driven test matrix** for all sentinel → family → HTTP status → exit code mappings

### Nix & build

31. **Extract `vendorHash` to `vendorHash.nix`** — BuildFlow suggested this for cleaner diffs
32. **Add `golangci-lint` cache warmup** to CI for faster lint runs
33. **Add `nix flake check` to pre-commit hook** (currently skipped by BuildFlow build mode)
34. **Consider `nix-update` integration** for automated dependency bumps

### Code quality

35. **Remove `.golangci.yml` indentation churn** — formatter changed 4-space to 2-space across entire file (474 lines touched for a 4-line addition)
36. **Consolidate error variable declarations** — `stream.go` has `errJPEGMaxIterations` as a standalone `var`, then a `var()` block for the 7 stream errors
37. **Add `//go:generate` directive** for error family registration boilerplate
38. **Consider a code generator for sentinel → family mapping** to prevent drift

### Documentation

39. **Update `docs/DOMAIN_LANGUAGE.md`** with error family terminology
40. **Add error handling section to website docs** — explain the classification system to users
41. **Document the godoclint exclusion** in a comment in `.golangci.yml` itself
42. **Update FEATURES.md** with go-error-family adoption as a feature

### Architecture

43. **Consider an error reporter interface** — allows different error handling strategies (log, metric, notify)
44. **Add error family metrics** — count errors by family in Prometheus
45. **Consider structured error types** — `type DaemonError struct{ Family; Code; Err error }` instead of global registry
46. **Add request-scoped error context** — middleware that tags errors with request ID
47. **Consider adopting `errors.AsType[E]`** (Go 1.26+ generic error handling) alongside go-error-family
48. **Audit the `Dependencies` struct** — DI function pointers could return typed errors for better classification
49. **Add error family to SSE broadcast events** — clients could react differently to 503 vs 400
50. **Consider error budgets** per family — alert when Infrastructure error rate exceeds threshold

---

## g) Questions

### 1. Should I push the 4 local commits to origin/master now?

The previous session's summary claimed a push happened, but origin is 4 commits behind. I don't push without explicit instruction per my rules. The commits are: go-error-family adoption (3 commits from previous session) + lint/doc fixes (1 commit from this session).

### 2. Should I expand go-error-family adoption to the remaining `http.Error()` sites?

There are ~8 more `http.Error()` calls in `handlers.go` and `commands.go` that still use hardcoded status codes. Some are correct (e.g., 405 Method Not Allowed), but some could benefit from classification. Expanding would increase consistency but also the diff surface.

### 3. Should I fix the BuildFlow critical finding (move `main.go` to `cmd/emeet-pixyd/main.go`)?

This is a significant structural change that would affect imports, the Nix build (`package.nix`), CI, and the module path. It's the "correct Go layout" but this is a single-binary daemon where `main.go` at root is a defensible choice. BuildFlow flags it as critical but it's a convention, not a correctness issue.

---

## Session Summary

**Started with:** A session summary claiming "all work complete, pushed to origin" that was wrong in 3 specific ways.

**Ended with:** All issues found and fixed. Full verification chain green (build, test, lint, nix). 4 commits ahead of origin, working tree clean, 0 lint issues, all nix checks pass.

**Time invested:** ~20 minutes of systematic verification and root-cause analysis.

---

## Resolution (2026-07-28)

The adoption this report verified is **shipped and stable**. The dependency has since advanced from v0.8.0 → **`go-error-family` v0.10.0** (`go.mod`, bumped in `ca41926`); `errorfamily.go`/`errorfamily_test.go` are present and the 5 verification gates (build, race test, lint, nix build, nix flake check) remain green.

**What changed vs this report:**

- The "push to origin" blocker (§c.1) is resolved — see inline correction above.
- §c.2 (`internal/pixy` classification) and §c.3 (`%v` vs `%w` wrapping audit) are still open.

**Forward-looking items:** the 50 ideas in §f are a brainstorm, not a commitment. The genuinely actionable, bounded subset (e.g. GitHub Actions SHA pinning, broader `LogError()` coverage) has been routed to `TODO_LIST.md`; the vaguer/longer-term ones to `ROADMAP.md`. This report's §f is the source of record for those ideas — do not re-list them here.

**Key lesson:** Never trust session summaries. Verify everything from scratch using fresh caches and the full verification chain (`go build` → `go test -race` → `golangci-lint` → `nix flake check`).
