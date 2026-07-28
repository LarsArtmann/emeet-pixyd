# go-error-family Adoption: Comprehensive Status Report

**Date:** 2026-07-23 21:27
**Scope:** Full go-error-family adoption audit, test gap closure, and verification
**Branch:** `master` (all 13 commits pushed to `origin/master`)

---

## a) FULLY DONE

### Adoption (Production Code)

| What                                 | Where                                         | Detail                                                                                                                    |
| ------------------------------------ | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Sentinel classification registration | `errorfamily.go`                              | 18 sentinels registered: 4 Infrastructure, 10 Rejection, 4 Transient via `sync.Once` idempotent `registerErrorFamilies()` |
| Stdlib defaults                      | `errorfamily.go`                              | `RegisterStdlibDefaults()` adds context.Canceled→Rejection, context.DeadlineExceeded→Transient, sql/os errors             |
| Constructor-born stream errors       | `stream.go`                                   | 7 typed errors via `errorfamily.NewInfrastructure()` (`errStreamNoFrame` through `errStreamStart`)                        |
| HTTP status derivation               | `stream.go` (7 sites), `handlers.go` (1 site) | `errorfamily.HTTPStatus(err)` replaces hardcoded status codes; fixed 3 genuine 500→503 bugs                               |
| CLI exit codes                       | `main.go` lines 303, 393                      | `os.Exit(errorfamily.ExitCode(err))` replaces `os.Exit(1)`                                                                |
| Structured error logging             | `main.go` line 392                            | `errorfamily.LogError(err, slog.Default())` adds family/code/retryable fields + correct severity                          |
| Wiring                               | `main.go:111`, `main_test.go:251`             | `registerErrorFamilies()` called from both `NewDaemon()` and `newTestDaemon()`                                            |

### Verification Gates (all green)

| Gate               | Command                          | Result                 |
| ------------------ | -------------------------------- | ---------------------- |
| Build              | `go build .`                     | PASS                   |
| Tests (race)       | `go test -race -count=1 ./...`   | PASS                   |
| Lint (90+ linters) | `golangci-lint run --timeout 5m` | 0 issues               |
| Nix build          | `nix build`                      | PASS                   |
| Nix flake check    | `nix flake check`                | PASS (all checks pass) |

### Test Coverage (8 test functions in `errorfamily_test.go`)

| Test                                                | What it verifies                                                                         |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `TestErrorFamilies_InfrastructureSentinels`         | 4 sentinels → Infrastructure, HTTP 503, exit 69                                          |
| `TestErrorFamilies_RejectionSentinels`              | 10 sentinels → Rejection, HTTP 400                                                       |
| `TestErrorFamilies_TransientSentinels`              | 4 sentinels → Transient, `IsRetryable` = true                                            |
| `TestErrorFamilies_WrappedErrorsStillClassify`      | `fmt.Errorf("%w")` chains preserve classification across all 3 families + exit codes     |
| `TestErrorFamilies_CLIExitCodes`                    | CLI boundary: Rejection→1, Infrastructure→69, Transient→75, unknown→75, wrapped config→1 |
| `TestErrorFamilies_StdlibDefaults`                  | context.Canceled→Rejection, context.DeadlineExceeded→Transient                           |
| `TestErrorFamilies_StreamErrorsAreInfrastructure`   | 7 stream errors → HTTP 503, exit 69, Infrastructure family                               |
| `TestErrorFamilies_UnknownErrorDefaultsToTransient` | Unknown errors → Transient (fail-open default), HTTP 503                                 |

### Dependency Hygiene

| What                                 | Status                                                                                                             |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------ |
| `go-error-family` v0.8.0 in `go.mod` | Direct require (not indirect) — fixed from previous session. **Now v0.10.0** (bumped `ca41926` after this report). |
| `go.sum`                             | Checksums match v0.8.0                                                                                             |
| `vendorHash` sync                    | `flake.nix:123` + `package.nix:14` both updated to `sha256-SiHVB/ev...`                                            |

### Documentation

| What                                         | Where                      |
| -------------------------------------------- | -------------------------- |
| `errorfamily.go` file responsibility         | `AGENTS.md` line ~100      |
| Error Handling section with adoption details | `AGENTS.md` lines ~175-179 |
| External Libraries entry (comprehensive)     | `AGENTS.md` line ~314      |
| godoclint exclusion rationale                | `AGENTS.md` line ~237      |
| Version reference v0.8.0                     | `AGENTS.md` line ~314      |

### Git

- 13 commits total across both sessions (`21b72d1` through `53007db`)
- 4 commits this session (`cd80e46`, `464de0f`, `6746ce2`, `53007db`)
- All pushed to `origin/master` (push confirmed: `a2dce73..53007db`)
- Working tree clean — no uncommitted or untracked files

---

## b) PARTIALLY DONE

### go-error-family adoption is scoped, not exhaustive

**What's adopted:** Sentinel registration, `HTTPStatus()`, `ExitCode()`, `LogError()` for the daemon init path.

**What's intentionally NOT adopted:**

1. **6 hardcoded `http.StatusBadRequest` guard sites** in `handlers.go` (lines 260, 266, 273, 320, 335, 369) — these are input validation guards with no error object (always 400). Forcing them through errorfamily adds ceremony for zero value.
2. **HTMX action handlers** — intentionally return `200 + HTML toast` for `outerHTML` swap (correct HTMX pattern, not classified JSON errors).
3. **13 `slog.Error()` sites** across the codebase — only the `NewDaemon` error path uses `LogError()`. The remaining 12 are operational errors (socket errors, HTTP server errors, OTel failures, etc.) that could benefit from classification metadata but are lower priority.

### `LogError()` adoption is minimal

Only 1 of ~14 error logging sites uses `errorfamily.LogError()`. The remaining sites use raw `slog.Error("msg", "error", err)` without family/code/retryable fields. High-value candidates: `state.go:129` (state save failure), `process.go:169` (audio source switch failure), `uevent.go:76,85` (hotplug failures).

---

## c) NOT STARTED

### Items identified but not actioned

1. **`cmd/emeet-pixyd/main.go` directory restructure** — BuildFlow flags this as CRITICAL every commit. `main.go` is at project root instead of `cmd/<appname>/main.go`. This is a structural convention, not a bug, but the linter will never stop complaining.
2. **GitHub Actions SHA pinning** — 9 workflow steps use `@tag` instead of `@SHA`. BuildFlow flags all as ERROR. Affects: `actions/checkout@v4`, `actions/setup-go@v5`, `golangci/golangci-lint-action@v7`, `actions/cache@v4`, `nick-fields/retry@v3`, `DeterminateSystems/nix-installer-action@v16`, `DeterminateSystems/magic-nix-cache-action@v9`.
3. **Dependabot vulnerabilities** (2 open):
   - **HIGH:** `fast-uri` vulnerable to host confusion via literal backslash authority delimiter (Go dependency)
   - **MEDIUM:** Astro Reflected XSS via unescaped View Transition animation properties (website dependency)
4. **`go.mod` direct/indirect mixing warning** — BuildFlow's `gomod-check` reports direct and indirect requires are in the same block (line 19). `go mod tidy` was run but this warning persists.
5. **vendorHash extraction** — BuildFlow suggests extracting inline `vendorHash` from `flake.nix` and `package.nix` into a dedicated `vendorHash.nix` file for cleaner diffs and tool interop.
6. **`.editorconfig`** — BuildFlow notes it's missing.
7. **README installation section** — BuildFlow notes it's missing.
8. **`errorfamily.HTTPHandler()` middleware** — The library provides a ready-made `HTTPHandler(HandlerFunc)` that wraps error-returning handlers with classification + JSON response. Not adopted because our handlers return HTML (templ), not JSON. Could be useful for the `/api/health` and `/api/snapshot` endpoints.
9. **`errorfamily.HandleError()` CLI boundary handler** — The library provides a full CLI error handler with structured What/Why/Fix/WayOut messages. Our `exitWithDaemonError` has a custom message ("Is emeet-pixyd running?") that's better UX for this specific daemon.
10. **Message templates** — The library supports `MessageTemplate` registration for user-facing error codes. None registered. Would enable `HandleErrorDetailed()` and `TemplateForCode()` at API boundaries.
11. **`errorfamilytest` assertion helpers** — The library has `AssertFamily`, `AssertCode`, `AssertRetryable`, `AssertExitCode`. Our tests use manual `t.Errorf` instead. Could reduce test boilerplate.

---

## d) TOTALLY FUCKED UP

### Nothing in this session

The previous session had 3 silent issues (indirect dependency, false "pushed" claim, stale lint cache), but this session's verification methodology caught and fixed all of them. No regressions introduced.

### Close calls (caught and avoided)

1. **Almost trusted stale LSP diagnostics** — The LSP reported `errorfamily.go:12: unused: var errorFamiliesRegistered` and `func registerErrorFamilies is unused`. These are stale — the functions are called from `main.go:111` and `main_test.go:251`. Verified by CLI `golangci-lint run` which reported 0 issues. Lesson: LSP diagnostics can lag behind reality; trust the CLI linter for verification.

2. **Almost didn't clear lint cache** — Previous session's "0 issues" was stale-cache-dependent. The godoclint false positive only appeared after clearing cache. This session confirmed clean by running `golangci-lint run` fresh.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture improvements

1. **Expand `LogError()` adoption** — 13 `slog.Error` sites could benefit from `errorfamily.LogError()` for structured family/code/retryable fields. Start with operational errors that flow through classified sentinels.

2. **Register `MessageTemplate`s for key error codes** — Would enable structured user-facing messages at the CLI boundary and API boundaries. Currently errors are raw strings.

3. **Consider `errorfamily.HTTPHandler()` for JSON API endpoints** — `/api/health` and `/api/snapshot` could benefit from classified JSON error responses instead of plain `http.Error`.

4. **Type-safe error construction** — The 6 hardcoded `http.StatusBadRequest` guard sites could use `errorfamily.NewRejection()` to get error objects, enabling consistent classification. Trade-off: ceremony vs. consistency.

### Process improvements

5. **Always clear `golangci-lint` cache before verifying** — `golangci-lint cache clean` or `trash -r ~/.cache/golangci-lint`. Stale cache produces false "0 issues".

6. **`go mod tidy` is mandatory after `go get`** — `go get` alone leaves deps as `// indirect`. Always follow with `go mod tidy`.

7. **Run `nix flake check`, not just `nix build`** — `nix flake check` runs the lint derivation that `nix build` skips.

8. **Test both halves of every contract** — Previous tests verified `HTTPStatus` (503) but not `ExitCode` (69) for stream errors. Each error-family contract has: family classification, HTTP status, exit code, retryability. Test all four.

### Technical debt

9. **GitHub Actions SHA pinning** — Security best practice. Tags can be moved to malicious code. Pin to commit SHAs.

10. **Dependabot alerts** — 2 open vulnerabilities. Should be triaged and updated.

11. **`cmd/` directory structure** — BuildFlow CRITICAL. Moving `main.go` to `cmd/emeet-pixyd/main.go` would silence the linter and follow Go project layout conventions. Requires updating `flake.nix`/`package.nix` build paths.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

### High impact, low effort

1. Pin all 9 GitHub Actions to commit SHAs (silences BuildFlow ERRORs, improves security)
2. Update `fast-uri` to fix HIGH dependabot vulnerability
3. Update Astro to fix MEDIUM dependabot XSS vulnerability
4. Expand `LogError()` to `state.go:129` (state save failure — already a classified sentinel path)
5. Expand `LogError()` to `process.go:169` (audio source switch failure)
6. Expand `LogError()` to `uevent.go:76,85` (hotplug failures — Infrastructure family)
7. Expand `LogError()` to `socket.go:45,57` (socket errors)
8. Add `.editorconfig` file (silences BuildFlow INFO)
9. Add `## Installation` section to README.md (silences BuildFlow INFO)
10. Run `go mod tidy` to investigate the direct/indirect mixing warning

### Medium impact, medium effort

11. Move `main.go` to `cmd/emeet-pixyd/main.go` (silences BuildFlow CRITICAL)
12. Update `flake.nix` and `package.nix` build paths for `cmd/` restructure
13. Extract `vendorHash` to `vendorHash.nix` (cleaner diffs, better tool interop)
14. Register `MessageTemplate`s for key error codes (`device.not_connected`, `config.invalid`, etc.)
15. Adopt `errorfamily.HTTPHandler()` for `/api/health` endpoint (JSON error response)
16. Adopt `errorfamily.HTTPHandler()` for `/api/snapshot` endpoint
17. Adopt `errorfamilytest.AssertFamily` helpers to reduce test boilerplate
18. Convert the 6 hardcoded `http.StatusBadRequest` guards to `errorfamily.NewRejection()` for consistency
19. Expand `LogError()` to remaining `slog.Error` sites in `main.go` (lines 166, 176, 221)
20. Expand `LogError()` to `metrics.go` (lines 44, 100, 110, 120)
21. Expand `LogError()` to `commander.go:55` (subprocess failure logging)
22. Expand `LogError()` to `probe.go` (lines 77, 136, 138 — partial device match warnings)
23. Add `errorfamily.RegisterClassifier()` for `*os.PathError` (file I/O errors → Infrastructure)
24. Add `errorfamily.RegisterClassifier()` for `syscall.Errno` (system call errors → Transient/Infrastructure)

### Architecture and design

25. Consider `errorfamily.Registry` with `Clone()` for test isolation (separate from `DefaultRegistry`)
26. Consider `errorfamily.HandleError()` for CLI command errors (full What/Why/Fix/WayOut)
27. Audit `commands.go` error returns — many return `"error: ..."` strings; could benefit from typed errors
28. Audit `hid.go` error returns — HID send failures could use `WrapTransient()` for retry hints
29. Audit `auto.go` error returns — `autoError` aggregation could use `errors.Join` + worst-family classification
30. Consider `errorfamily.WithContextAny()` for adding device path / command context to errors
31. Consider `errorfamily.WrapOnce()` at API boundaries to prevent double-wrapping
32. Add retry policy adoption (`Family.RetryPolicy()`) for transient HID errors
33. Consider `errorfamily.DiagnosticFunc` for device-not-connected errors (check `/dev/hidraw*` permissions)

### Testing

34. Add fuzz test for `errorfamily.HTTPStatus()` with arbitrary error chains
35. Add test verifying `errors.Join` picks worst family across our sentinel mix
36. Add test for `errorfamily.Code()` extraction through wrapped chains
37. Add benchmark for `errorfamily.Classify()` with our registered sentinels
38. Add test that `registerErrorFamilies()` is truly idempotent (concurrent calls)
39. Add integration test verifying CLI exit codes end-to-end (build binary, run with bad config, check exit code)
40. Add test for `errorfamily.LogError()` output format (structured fields present)

### Documentation and DX

41. Document error family adoption in website docs (`website/src/content/docs/`)
42. Add error codes reference table to AGENTS.md or docs
43. Create ADR for go-error-family adoption decision and scope
44. Document the scoped adoption rationale (why HTMX handlers are excluded)
45. Add `CONTRIBUTING.md` section on error handling conventions

### Infrastructure

46. Add Dependabot config for automated dependency updates (if not present)
47. Consider Renovate alternative for more configurable update PRs
48. Add `gosec` baseline file for tracking security findings over time
49. Add CodeQL analysis workflow for security scanning
50. Consider `golangci-lint` cache warming in CI for faster runs

---

## g) QUESTIONS (3)

### 1. Should we move `main.go` to `cmd/emeet-pixyd/main.go`?

BuildFlow flags this as CRITICAL on every commit. It's a Go project layout convention, not a bug. The move requires updating `flake.nix` and `package.nix` build paths (`cd .` → `cd cmd/emeet-pixyd`), and possibly the NixOS module binary path. Is the structural correctness worth the churn, or should we suppress the BuildFlow finding?

### 2. Should we expand `errorfamily.LogError()` to all 13 `slog.Error` sites?

Currently only the `NewDaemon` error path uses `LogError()`. The remaining 12 sites use raw `slog.Error`. Adopting `LogError()` everywhere would add family/code/retryable fields to all error logs automatically, but some of these errors are not classified sentinels (they'd classify as Transient/unknown by default). Is blanket adoption desired, or should we cherry-pick only sites where classified sentinels flow through?

### 3. Should we address the 2 Dependabot vulnerabilities now or batch them?

The HIGH severity `fast-uri` vulnerability is in a Go dependency (likely transitive through OTel or Prometheus). The MEDIUM Astro XSS is in the website. Both may require minor version bumps that could have breaking changes. Should we fix them immediately, or batch with the next dependency update cycle?

---

## Session Metrics

| Metric                                     | Value                                                               |
| ------------------------------------------ | ------------------------------------------------------------------- |
| Commits this session                       | 4                                                                   |
| Commits total (both sessions)              | 13                                                                  |
| Files changed this session                 | 3 (`AGENTS.md`, `errorfamily_test.go`, `main.go`)                   |
| Test functions added                       | 2 (`TestErrorFamilies_CLIExitCodes`, expanded wrapped/stream tests) |
| Test assertions added                      | ~30 (ExitCode + Classify across all families)                       |
| Bug fixes                                  | 0 (all bugs were fixed in the previous session)                     |
| Verification gates                         | 5/5 green (build, test, lint, nix build, nix flake check)           |
| Remaining `http.Error` without errorfamily | 6 (all hardcoded 400 guards — intentional)                          |
| Remaining `slog.Error` without `LogError`  | 12 (out of 13 total)                                                |
| Lint issues                                | 0                                                                   |
| Pushed to origin                           | Yes (`a2dce73..53007db`)                                            |

---

## Resolution (2026-07-28)

This report is the authoritative record of the **completed** go-error-family adoption. Everything in §a is shipped and still green (build, race test, lint 0 issues, nix build, nix flake check). The dependency has since moved v0.8.0 → **v0.10.0** (`ca41926`).

**§b (scoped adoption) is unchanged by design** — HTMX handlers returning 200+HTML toast, the 6 hardcoded 400 guards, and the circuit breaker remain intentionally outside classification. These are documented decisions, not gaps.

**§c / §f forward-looking items:** the 50 enhancement ideas (broader `LogError()`, `MessageTemplate`s, `HTTPHandler()` for JSON endpoints, `errorfamilytest` helpers, fuzz/benchmark additions, ADR, CodeQL, etc.) are a brainstorm. The actionable, bounded subset has been routed to `TODO_LIST.md`; the rest to `ROADMAP.md`. This report remains the source of record — they are not re-listed here.
