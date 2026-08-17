# Status Report: cqrs-htmx/v2 Migration

**Date:** 2026-06-20 16:15
**Session:** Migration of HTTP/SSE/HTMX layer to `cqrs-htmx/v2`
**Branch:** master
**Previous commit:** `63ebd8b` — refactor: adopt cqrs-htmx/v2 (committed during session)

---

## Executive Summary

Migrated the web infrastructure layer of `emeet-pixyd` from hand-rolled implementations and `httputil` to `cqrs-htmx/v2` (same author's library). The migration covers SSE broadcasting, security headers, request logging, request ID enrichment, middleware chaining, HTMX JS serving, and JSON response writing. All Go tests pass (710 test functions), lint is at 0 issues. The nix build is **broken** due to a private transitive dependency (`go-cqrs-lite`) — this is documented but not fixable without making that repo public.

---

## a) FULLY DONE

| #  | Item                                     | Evidence                                                                                                                                                                                                                           |
| -- | ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **cqrs-htmx/v2 dependency added**        | `go.mod`: `github.com/larsartmann/cqrs-htmx/v2 v2.5.0` as direct dep                                                                                                                                                               |
| 2  | **SSE migrated to cqrshtmx.Broadcaster** | `main.go:56` — `broadcaster *cqrshtmx.Broadcaster` field; `broadcastStateChanged()` delegates to `Broadcaster.Broadcast()`. Old `eventClients map[chan struct{}]struct{}` + `subscribeEvents`/`unsubscribeEvents` methods deleted. |
| 3  | **SSE handler uses cqrshtmx.SSEStream**  | `handlers.go:173-195` — `NewSSEStream(w, r)` + `stream.Send()` + `stream.Context().Done()`. Old manual flusher/header code deleted.                                                                                                |
| 4  | **Security headers migrated**            | `middleware.go:27` — `cqrshtmx.SecurityHeadersMiddlewareWithConfig()` replaces `httputil.SecurityHeaders()`                                                                                                                        |
| 5  | **Request logging migrated**             | `middleware.go:24` — `cqrshtmx.RequestLoggingSlog()` replaces `httputil.Logging()`                                                                                                                                                 |
| 6  | **Request ID migrated**                  | `middleware.go:36` — `cqrshtmx.ContextEnrichmentMiddleware(nil)` replaces `httputil.RequestID()`. IDs are now 26-char ULIDs (was 8-char hex).                                                                                      |
| 7  | **Middleware chain migrated**            | `main.go:148` — `cqrshtmx.Chain(mws...)(mux)` replaces `httputil.Chain(mux, mws...)`                                                                                                                                               |
| 8  | **HTMX JS served via embedded handler**  | `handlers.go:312` — `cqrshtmx.HTMXScriptHandler()` at `/static/htmx.js` (embedded v2.0.9, ETag, immutable cache). Old `static/htmx-2.0.8.min.js` (82KB) deleted.                                                                   |
| 9  | **Template updated**                     | `templates.templ:55` — `<script src="/static/htmx.js">` (no version query param needed). `templates_templ.go` regenerated.                                                                                                         |
| 10 | **Health endpoint uses WriteJSON**       | `handlers.go:154` — `cqrshtmx.WriteJSON()` replaces manual `json.Marshal` + write. `encoding/json` import removed from handlers.go.                                                                                                |
| 11 | **httputil direct dependency removed**   | `go.mod` — `httputil` is now only `// indirect` (cqrs-htmx uses it internally for `ClientIP`). No `.go` file imports it directly.                                                                                                  |
| 12 | **Tests updated**                        | `middleware_test.go` — Request ID tests expect 26-char ULIDs, passthrough test generates valid ULID via `cqrshtmx.NewRequestID()`. `main_test.go` — `newTestDaemon` initializes `broadcaster`.                                     |
| 13 | **SSE tests pass**                       | `sse_test.go` — `TestSSEEndpoint_SendsConnectedEvent` and `TestSSEEndpoint_BroadcastsRefresh` pass with new Broadcaster.                                                                                                           |
| 14 | **Lint clean (0 issues)**                | `golangci-lint run --timeout 2m ./...` — 0 issues. `//nolint:exhaustruct` added for intentional partial struct initialization.                                                                                                     |
| 15 | **CI workflow updated**                  | `.github/workflows/go-test.yml` — `GOPRIVATE` env var + git config for private module access. `GOWORK: off` moved to job-level env.                                                                                                |
| 16 | **AGENTS.md updated**                    | Middleware, SSE, HTMX JS, and external libraries sections rewritten to reflect cqrs-htmx adoption.                                                                                                                                 |
| 17 | **Build passes**                         | `go vet ./...` ✓, `go build ./...` ✓                                                                                                                                                                                               |

---

## b) PARTIALLY DONE

| # | Item            | Status                      | What remains                                                                                        |
| - | --------------- | --------------------------- | --------------------------------------------------------------------------------------------------- |
| 1 | **CI workflow** | Updated but **uncommitted** | `.github/workflows/go-test.yml` is modified in working tree. Needs commit.                          |
| 2 | **AGENTS.md**   | Updated but **uncommitted** | Nix limitation documented. Needs commit.                                                            |
| 3 | **package.nix** | Updated but **uncommitted** | Contains placeholder vendorHash + documentation comment about private dep limitation. Needs commit. |
| 4 | **flake.nix**   | Updated but **uncommitted** | Lint check vendorHash restored to original (broken). Needs commit.                                  |

---

## c) NOT STARTED

| # | Item                         | Why                                                                                      |
| - | ---------------------------- | ---------------------------------------------------------------------------------------- |
| 1 | **Nix build fix**            | Requires making `go-cqrs-lite` public (external action outside this repo)                |
| 2 | **vendorHash update**        | Cannot compute until nix FOD can access private deps                                     |
| 3 | **Benchmarks for new SSE**   | No before/after benchmark comparing hand-rolled SSE vs cqrshtmx.Broadcaster              |
| 4 | **SSE reconnection support** | cqrs-htmx supports `LastEventID` + `SSEEventStore` for replay — not wired in emeet-pixyd |
| 5 | **SSE heartbeat**            | `stream.Heartbeat()` available but not used (proxy keepalive)                            |

---

## d) TOTALLY FUCKED UP

| # | Issue                   | Impact                              | Root Cause                                                                                                                                                                                                                                                                                                                                                                 |
| - | ----------------------- | ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **Nix build is broken** | `nix build` fails at go-modules FOD | `cqrs-htmx` depends on `go-cqrs-lite` (private repo, `github.com/larsartmann/go-cqrs-lite`). Not on Go module proxy. Nix sandbox has no SSH keys or GitHub credentials. Tried: `GOPROXY` env, `overrideModAttrs` with `GOPRIVATE`, `GIT_CONFIG_*`, `HOME`/`GOMODCACHE` override, `--option sandbox false`. All failed — nix FOD fundamentally cannot access private repos. |
| 2 | **vendorHash is stale** | Build mismatch                      | The `vendorHash` in `package.nix` and `flake.nix` still reflects the OLD dependency set (pre-cqrs-htmx). Cannot compute the new hash.                                                                                                                                                                                                                                      |

---

## e) WHAT WE SHOULD IMPROVE

| #  | Improvement                                   | Rationale                                                                                                                                      |
| -- | --------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Make `go-cqrs-lite` public**                | Unblocks nix build, simplifies CI (no GOPRIVATE/token needed), enables other consumers                                                         |
| 2  | **Add SSE heartbeat**                         | `stream.Heartbeat(ctx, 15s)` prevents proxy idle kills. Available in cqrs-htmx, just 1 line.                                                   |
| 3  | **Wire SSE reconnection**                     | Use `stream.LastEventID()` + `SSEEventStore` so missed events replay after reconnect                                                           |
| 4  | **Benchmark SSE before/after**                | cqrs-htmx Broadcaster uses `reflect.ValueOf(ch).Pointer()` for O(1) unsubscribe — may have perf implications vs the old direct map key         |
| 5  | **Consider cqrshtmx.RecoveryMiddleware**      | Currently no panic recovery on HTTP handlers. One middleware addition.                                                                         |
| 6  | **Evaluate cqrshtmx.HTMXMiddleware**          | Would provide `HTMXFromContext` for partial vs full-page rendering decisions — currently all handlers render the same regardless of HTMX boost |
| 7  | **Remove stale static dir reference**         | `static/` now only has `app.js` + `style.css`. The embed still works but could be cleaner.                                                     |
| 8  | **Update FEATURES.md**                        | No mention of cqrs-htmx adoption in the feature inventory                                                                                      |
| 9  | **Pin cqrs-htmx version in nix**              | Currently relies on `go get @latest`. Should pin to v2.5.0 explicitly in docs.                                                                 |
| 10 | **Add integration test for HTMX JS endpoint** | Verify `/static/htmx.js` returns correct Content-Type, ETag, and 304 on If-None-Match                                                          |

---

## f) Top 25 Things to Do Next

### Critical (blocks production)

1. **Commit the uncommitted CI/nix/AGENTS changes** — 4 files modified in working tree
2. **Make `go-cqrs-lite` public** (external) — unblocks nix build entirely
3. **Update vendorHash** after go-cqrs-lite is public — run `nix build` to get correct hash
4. **Verify `nix build` works end-to-end** after vendorHash update
5. **Run `nix flake check`** — verify all checks pass

### High value

6. **Add SSE heartbeat** — `go stream.Heartbeat(stream.Context(), 15*time.Second)` in `handleEvents`
7. **Add panic recovery middleware** — `cqrshtmx.RecoveryMiddleware` in the chain
8. **Benchmark SSE performance** — compare old hand-rolled vs Broadcaster throughput
9. **Write integration test for `/static/htmx.js`** — ETag, Content-Type, caching headers
10. **Update FEATURES.md** — document cqrs-htmx adoption as a feature

### Quality

11. **Add SSE reconnection support** — wire `LastEventID` + event store
12. **Evaluate `cqrshtmx.HTMXMiddleware`** — for partial rendering optimizations
13. **Add `cqrshtmx.StatusRecorder`** — already used internally by RequestLoggingSlog, but could expose status codes for custom metrics
14. **Pin templ CLI version** — CI uses `@latest`, nix uses v0.3.1001, go.mod has v0.3.1020. Version mismatch warning in build logs.
15. **Clean up `.golangci.yml`** — verify no new nolint directives are needed for cqrs-htmx patterns
16. **Add test for broadcaster unsubscribe race** — cqrs-htmx has this test, but emeet-pixyd should verify it works in context
17. **Update CHANGELOG.md** — document the cqrs-htmx migration as a breaking change (HTMX JS path changed)
18. **Review go.sum for unused indirect deps** — `go mod tidy` may have left transitives

### Refinement

19. **Consider extracting SSE handler to its own file** — `sse.go` for cohesion (currently in handlers.go)
20. **Add structured logging to SSE connections** — log subscribe/unsubscribe events
21. **Document the SSE protocol** — event names (`connected`, `refresh`), data format
22. **Add metrics for SSE** — subscriber count, events broadcasted, drops
23. **Review if `httputil` indirect dep can be eliminated** — cqrs-htmx uses it only for `ClientIP`; if that function is trivial, could be vendored
24. **Evaluate `cqrshtmx.StructuredError`** for health endpoint errors — RFC 7807 format
25. **Consider `cqrshtmx.WriteJSON` for waybar output** — currently uses manual formatting

---

## g) Top #1 Question I Cannot Figure Out

**When will `go-cqrs-lite` be made public?**

The entire nix build pipeline is blocked on this. `cqrs-htmx/v2` (which is public and on the Go module proxy) depends on `go-cqrs-lite` subdirectory modules (`codec/v2`, `command/v2`, `event/v2`, `id/v2`, `query/v2`, `dispatcher/v2`), all of which are private. Without these being public:

- `nix build` cannot work (FOD sandbox can't fetch private repos)
- `vendorHash` cannot be computed
- CI requires `GOPRIVATE` + `GITHUB_TOKEN` git config hacks
- Any external contributor cloning the repo cannot build without SSH access to the private repo

**Is there a timeline for making `go-cqrs-lite` public, or should we find an alternative approach (e.g., vendoring the specific cqrs-htmx functions we need locally to avoid the transitive dependency)?**

---

## Metrics Snapshot

| Metric                                  | Value                                                  |
| --------------------------------------- | ------------------------------------------------------ |
| Go source LOC (non-test, non-generated) | 3,981                                                  |
| Test functions                          | 710                                                    |
| `go test -race -count=1 ./...`          | ✅ PASS                                                |
| `golangci-lint run --timeout 2m ./...`  | ✅ 0 issues                                            |
| `go vet ./...`                          | ✅ PASS                                                |
| `nix build`                             | ❌ BROKEN (private dep)                                |
| Direct deps in go.mod                   | 8 (added cqrs-htmx/v2, removed httputil direct)        |
| Indirect deps added                     | ~15 (casbin, nosurf, go-cqrs-lite/\*, samber/lo, etc.) |
| Files deleted                           | 1 (`static/htmx-2.0.8.min.js`, 82KB / 3449 lines)      |
| Files changed in migration commit       | 12 (+127, -3571 lines net)                             |

---

## Migration Map (Before → After)

| Component        | Before                                    | After                                                      | File:Line          |
| ---------------- | ----------------------------------------- | ---------------------------------------------------------- | ------------------ |
| SSE fan-out      | `eventClients map[chan struct{}]struct{}` | `*cqrshtmx.Broadcaster`                                    | `main.go:56`       |
| SSE handler      | Manual headers + flusher + `fmt.Fprintf`  | `cqrshtmx.NewSSEStream` + `stream.Send`                    | `handlers.go:173`  |
| Security headers | `httputil.SecurityHeaders()`              | `cqrshtmx.SecurityHeadersMiddlewareWithConfig()`           | `middleware.go:27` |
| Request logging  | `httputil.Logging(slog.Default())`        | `cqrshtmx.RequestLoggingSlog(slog.Default())`              | `middleware.go:24` |
| Request ID       | `httputil.RequestID()` (8-char hex)       | `cqrshtmx.ContextEnrichmentMiddleware(nil)` (26-char ULID) | `middleware.go:36` |
| Middleware chain | `httputil.Chain(mux, mws...)`             | `cqrshtmx.Chain(mws...)(mux)`                              | `main.go:148`      |
| HTMX JS          | `static/htmx-2.0.8.min.js` (82KB file)    | `cqrshtmx.HTMXScriptHandler()` (embedded)                  | `handlers.go:312`  |
| Health JSON      | `json.Marshal` + `Write(data)`            | `cqrshtmx.WriteJSON()`                                     | `handlers.go:154`  |
| Broadcast        | `broadcastStateChanged()` manual loop     | `Broadcaster.Broadcast(SSEEvent{...})`                     | `main.go:199`      |
