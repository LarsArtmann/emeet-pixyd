# PR Review & Cherry-Pick Status — 2026-07-14 06:37

> Session focus: Reviewed two open PRs from external contributor, cherry-picked the still-valid fixes, closed PRs.

---

## a) FULLY DONE

### PR Reviews

- **PR #1** (Linux compat: sysfs USB walk, Flusher passthrough, MJPEG write deadline) — reviewed in detail against current master, posted comprehensive review comment
- **PR #2** (PTZ slider rendering) — reviewed in detail against current master, posted comprehensive review comment
- Both PRs assessed against the **base commit** (`25f5504`) they were filed against, not just current master, to fairly evaluate their original value
- Thank-you comments posted on both PRs acknowledging the genuine bugs found
- Both PRs **closed** with explanatory comments

### Code Fixes Applied (3 files, +27/-2 lines)

1. **`templates.templ:211-212`** — `"&#176;"` → `"°"` (literal U+00B0)
   - Templ HTML-escapes the `&` in `&#176;`, producing literal `120&#176;` on initial server render
   - Self-healed on first slider drag (app.js uses `\u00b0`), but initial render was broken
   - Verified: no `&#176;` remaining anywhere in codebase

2. **`http.go:78-80`** — Added `Unwrap()` method to `statusRecorder`
   - Required for `http.ResponseController` to reach the underlying `http.ResponseWriter`
   - Without this, `SetWriteDeadline` cannot traverse the wrapper
   - The existing `Flush()` was already present; `Unwrap()` was the missing piece

3. **`stream.go:155-159`** — Clear write deadline via `ResponseController`
   - `WriteTimeout: 30s` in `main.go:155` was killing MJPEG streams after 30 seconds
   - `setupStream()` now calls `rc.SetWriteDeadline(time.Time{})` to opt out of the server-level timeout
   - Logs a `slog.Warn` if clearing fails

4. **`stream.go:169-182`** — ffmpeg stderr → `slog.Debug`
   - Was silently discarded before; now piped to debug logging for diagnostics
   - Goroutine-based scanner; pipe creation errors logged at debug level

### Verification

- `templ generate` — regenerated successfully
- `go vet` — clean
- `go test -race -count=1` — all pass
- `golangci-lint run --timeout 2m` — 0 issues
- Generated `_templ.go` verified to contain `°` literal

### Cleanup

- Local `pr-1` and `pr-2` branches deleted

---

## b) PARTIALLY DONE

Nothing — all fixes were applied completely and verified.

---

## c) NOT STARTED

- **Changes are uncommitted** — all work is in the working tree, no commit made yet
- **Changes are not pushed** — user hasn't asked to commit or push

---

## d) TOTALLY FUCKED UP

Nothing this session. Minor notes:

- Initial `gh pr review --request` used wrong flag (`--request` instead of `--comment` or `--request-changes`); fixed immediately
- DNS outage briefly blocked PR closure; retried successfully after user confirmed network was back
- First templ edit attempt used the wrong old_string format (spaces vs tabs mismatch); fixed by using simpler unique substring
- WSL lint warning on the stderr scanner goroutine (`missing whitespace above if`); fixed by adding blank line

---

## e) WHAT WE SHOULD IMPROVE

1. **`Unwrap()` should have been added when `statusRecorder` was introduced** — it's a standard Go pattern for ResponseWriter wrappers. The `Flush()` and `Push()` methods were added but `Unwrap()` was forgotten. This is a latent bug that existed since the refactor.

2. **The 30-second stream timeout has been a live bug since the HTTP server was configured** — `WriteTimeout: 30s` is incompatible with MJPEG streaming by design. The fix should have been applied when streaming was first implemented. Anyone using the web UI would have seen the preview die after 30 seconds.

3. **`&#176;` in templ was a known footgun** — templ HTML-escapes string content, so HTML entities in Go string literals get double-escaped. The codebase should audit for other HTML entity patterns (`&#x...`, `&...;`) passed as template parameters.

4. **No test covers the MJPEG stream timeout scenario** — the fix works but there's no regression test proving the deadline is cleared. An integration test that verifies `ResponseController.SetWriteDeadline` succeeds through `statusRecorder` would prevent regressions.

5. **No test covers `statusRecorder.Unwrap()`** — a simple type assertion test would ensure the method exists and returns the inner writer.

6. **PR response time was ~2 months** — both PRs were filed May 7, reviewed July 14. External contributors deserve faster feedback.

7. **`WriteTimeout` may need rethinking** — clearing the deadline entirely means a hung stream connection never times out. A better long-term fix might be a per-stream idle timeout (e.g., no data written for 60s → close). The current fix is correct but opens a (minor) resource leak vector.

8. **The `responseWriter` wrapper in the old `middleware.go` (at base commit) lacked both `Flush()` and `Unwrap()`** — this was correctly identified by the contributor as the cause of "streaming not supported". The refactor to `statusRecorder` added `Flush()` but still missed `Unwrap()`. This suggests the wrapper's interface compliance should be tested.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (this session's followup)

1. **Commit the changes** — 4 fixes across 3 files, all verified
2. **Audit templates for other HTML entities** — search for `&#`, `&x`, named entities passed as templ parameters
3. **Add a test for `statusRecorder.Unwrap()`** — verify it returns the inner writer
4. **Add a test for `statusRecorder` interface compliance** — `http.Flusher`, `http.Pusher`, `http.ResponseWriter`, and `unwrapper` interfaces
5. **Consider a per-stream idle timeout** instead of clearing the deadline entirely — prevents resource leaks from abandoned connections

### Testing

6. **Add integration test for MJPEG stream longevity** — verify stream survives beyond `WriteTimeout`
7. **Add test that ffmpeg stderr pipe errors are handled gracefully** — goroutine shouldn't panic if pipe closes
8. **Add fuzz test for `extractJPEGFrame` corrupt stream recovery** — already has iteration guard, needs testing
9. **Test streaming with `streamSema` contention** — two concurrent stream requests, verify second gets 503

### PR / Community

10. **Add a CONTRIBUTING.md** — guide external contributors on build setup, test commands (`GOWORK=off`, `GOEXPERIMENT=jsonv2`)
11. **Add PR response SLA note to README** — set expectations for review time
12. **Enable Dependabot or Renovate** — keep dependencies current
13. **Consider enabling GitHub Actions for PR checks** — the CI workflow exists but may need `pull_request` trigger verification

### MJPEG Stream Robustness

14. **Add stream heartbeat/reconnection logic in app.js** — client-side reconnection if stream stalls
15. **Log stream duration metrics** — `recordStreamDuration` exists; verify it's captured correctly with the new deadline fix
16. **Consider HTTP/2 support for streaming** — avoids head-of-line blocking
17. **Add `Connection: close` header for stream response** — prevents keepalive reuse issues with long-lived connections

### UI / Templates

18. **Audit all templ string parameters for HTML entity issues** — systematic check
19. **Add a CSP-friendly favicon** — current inline SVG data URI may conflict with strict CSP
20. **Consider server-side rendering tests** — verify initial render produces correct HTML without JS

### Probe / Device Detection

21. **Test `matchesPixyID` against real kernel 6.x uevent data** — the current approach is different from PR #1's sysfs walk
22. **Add test for partial device detection** — video found but no hidraw (already logged but untested?)
23. **Document kernel version compatibility** — README should mention tested kernel versions

### Code Quality

24. **Add `staticcheck` to CI** — in addition to golangci-lint
25. **Run `govulncheck` locally** — CI has it but local dev should too
26. **Consider adding `errcheck` explicitly** — ensure all errors are handled or explicitly ignored
27. **Add pre-commit hook for `templ generate`** — prevent building stale templates
28. **Consider Go 1.27 upgrade prep** — `encoding/json/v2` becomes stable, `GOEXPERIMENT` flag can be removed

### Monitoring / Observability

29. **Add stream health metric** — active stream count, duration histogram, error rate
30. **Add ffmpeg crash counter** — track how often ffmpeg dies unexpectedly
31. **Add alerting on probe failures** — repeated device-not-found should alert

### Documentation

32. **Update AGENTS.md** with the new `Unwrap()` method on `statusRecorder`
33. **Document the stream timeout behavior** — explain why deadline is cleared and the tradeoff
34. **Add troubleshooting guide for "streaming not supported"** — link from web UI error
35. **Update FEATURES.md** with MJPEG streaming status

### NixOS / Packaging

36. **Verify NixOS module works with kernel 6.x** — the probe changes should be tested
37. **Add `v4l-utils` version pin** — PTZ behavior depends on v4l2-ctl version
38. **Consider systemd watchdog for stream health** — if stream dies, restart daemon

### Security

39. **Audit MJPEG stream for resource exhaustion** — single-stream semaphore helps but verify
40. **Consider rate limiting on `/api/snapshot`** — prevents frame extraction abuse
41. **Verify CSP headers don't interfere with MJPEG multipart** — `img-src` must allow `/api/stream`

### Performance

42. **Benchmark `extractJPEGFrame` with large frames** — ensure the 10M iteration guard is sufficient
43. **Consider `io.Copy` instead of byte-by-byte frame extraction** — potential performance win
44. **Profile stream goroutine** — identify if stderr scanner goroutine adds overhead
45. **Consider `sync.Pool` for `bytes.Buffer` in `extractJPEGFrame`** — reduce allocations

### Misc

46. **Add `nix flake check` to local dev workflow** — catch FOD issues before pushing
47. **Verify website docs reflect current streaming behavior** — docs may be stale
48. **Consider adding `golangci-lint` cache to CI** — speed up lint runs
49. **Review all `//nolint` directives** — ensure they're still needed after fixes
50. **Add `CHANGELOG.md` entry for these fixes** — if changelog discipline is wanted

---

## g) Top 2 Questions

### 1. Should we commit these changes, and if so, what's the commit structure?

The 4 fixes are logically related (all from PR review) but semantically distinct: a template rendering fix, a Go interface fix, a streaming timeout fix, and a logging improvement. **One commit or four?** The total diff is small (27 lines), so a single commit with a descriptive message seems right. But I didn't commit because the user hadn't asked me to.

### 2. Should the `WriteTimeout` clearing use an idle timeout instead of `time.Time{}`?

Clearing the write deadline entirely means an abandoned MJPEG connection (client gone, TCP still open) holds the stream semaphore forever, blocking all other stream attempts. A more robust fix would set a recurring idle timeout (e.g., reset a 30s deadline after each successful frame write). This is the approach nginx uses for streaming. The current fix is correct for the immediate bug but may need revisiting. **Should I implement the idle-timeout variant now, or ship the simple fix first?**
