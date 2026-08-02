# Status Report: 2026-08-02 18:34 — Stream Header Flush Fix

## Session Summary

Fixed `TestHandleStream_NoFFmpeg` which was timing out (2s `context deadline exceeded`) in the race-enabled test run.

### Root Cause

`setupStream` in `stream.go` set `Content-Type` and `Cache-Control` headers but never called `WriteHeader(http.StatusOK)` or flushed the response. The HTTP 200 status was only implicitly sent when the first MJPEG frame was written in `writeFrames`. When ffmpeg is available but hangs waiting on a device (no `/dev/video0` data), no frame arrives within the test's 2s timeout, causing the client request to time out.

### Fix

`stream.go:208-209` — Added explicit `responseWriter.WriteHeader(http.StatusOK)` + `flusher.Flush()` immediately after setting headers, before returning from `setupStream`. This sends the 200 status to the client as soon as the stream is established, which is also the correct behavior for MJPEG streaming (browser shows "loading" immediately rather than appearing to hang).

Commit: `b6fc96c fix(stream): explicitly send response headers and flush before streaming`

---

## a) FULLY DONE

1. **Diagnosed root cause** of `TestHandleStream_NoFFmpeg` timeout — missing `WriteHeader`/`Flush` in `setupStream`.
2. **Fixed** by adding `responseWriter.WriteHeader(http.StatusOK)` + `flusher.Flush()` after header setup.
3. **Verified** — `TestHandleStream_NoFFmpeg` passes with `-race`.
4. **Verified** — all stream/snapshot tests pass with `-race`.
5. **Verified** — full test suite (`go test -race -count=1 ./...`) passes.
6. **Verified** — `golangci-lint run --timeout 2m ./...` clean (0 issues).

---

## b) PARTIALLY DONE

Nothing.

---

## c) NOT STARTED

Nothing relevant to this session's scope.

---

## d) TOTALLY FUCKED UP

Nothing.

---

## e) WHAT WE SHOULD IMPROVE (Self-Critique)

### Things I Could Have Done Better

1. **No regression test for the actual bug** — The existing `TestHandleStream_NoFFmpeg` already covers this case but only because it happened to use a 2s timeout. I should have added a test that explicitly asserts the response status code arrives _before_ any frame data — a test that would have caught the missing `WriteHeader` even without the timeout symptom. The current test still passes by coincidence (it accepts 503 _or_ 200); if the bug regressed, the test would still pass (503 path), and the timeout would return.

2. **Didn't check the `setupStream` flusher nil-safety** — I added `flusher.Flush()` but `setupStream` already validated `flusher` is non-nil earlier (line 162-167). So this is safe, but I didn't explicitly verify this in my reasoning chain — I just added the call. A more rigorous approach would have traced the `flusher` variable's provenance before adding the call.

3. **Didn't consider whether `WriteHeader` should be guarded** — If `setupStream` ever gets called after a prior `WriteHeader` (e.g., from a middleware that already wrote), calling `WriteHeader` again is a Go warning ("superfluous response.WriteHeader"). Currently `setupStream` is the first writer, so this is fine, but I didn't verify the full middleware chain to confirm no middleware pre-writes headers. A defensive `rc := http.NewResponseController(responseWriter)` pattern (already present at line 169) suggests the code is careful about response control — but I didn't audit it.

4. **Didn't run the integration test build tag** — The `//go:build integration` tests (`integration_hardware_test.go`) also exercise streaming paths with real hardware. I didn't run `go test -tags=integration` (which would fail without hardware anyway), but I should have at least confirmed they compile.

5. **Didn't check if the SSE `Broadcaster` has the same pattern** — SSE streaming (`sse.go`) also writes headers before streaming. If it has the same "headers set but not flushed" bug, it would manifest the same way. I noticed this but didn't investigate — it's out of scope but worth flagging.

6. **No memory update** — I didn't update `AGENTS.md` with the "always call WriteHeader+Flush before streaming frames" pattern. This is a code convention worth recording.

### Observations from the Codebase (Not My Work)

7. **10 gopls warnings** — All `stdversion` warnings about `encoding/json/v2` APIs (`json.Marshal`, `json.Unmarshal`, `json.MarshalWrite`, `json.RejectUnknownMembers`) requiring go1.27. These are expected — the project uses `GOEXPERIMENT=jsonv2` and targets go1.26. They'll resolve when go1.27 ships. Not actionable now.

8. **2 `bloop` warnings** in `sse_test.go` — `b.N` can be modernized to `b.Loop()`. Minor, not related to this session.

---

## f) Up to 50 Things We Should Get Done Next

### Stream/HTTP Robustness

1. Add a regression test that asserts `setupStream` sends HTTP 200 _before_ any frame data (not just "503 or 200 within 2s").
2. Audit `sse.go` `sseStream` for the same "headers set but not flushed" pattern — SSE streams may have the same bug.
3. Add a test for `setupStream` that verifies `WriteHeader` is called exactly once (detect double-write regressions).
4. Audit all HTTP handlers that stream (stream, snapshot, SSE) for consistent header-then-flush pattern.
5. Consider a shared `writeHeadersAndFlush(w, contentType, cacheControl)` helper to standardize the pattern.

### Test Quality

6. `TestHandleStream_NoFFmpeg` should assert a _specific_ status code, not "503 or 200" — the current assertion masks bugs.
7. Add a test for the case where ffmpeg starts but the device produces no frames for N seconds (the actual bug scenario).
8. Add a test that the stream response includes `Content-Type: multipart/x-mixed-replace; boundary=frame` header.
9. Run `go test -tags=integration ./...` in CI (or at least compile-check) to catch integration test build failures.
10. Add a benchmark for `setupStream` to ensure header setup is not a bottleneck.

### Error Handling

11. If `setupStream` fails after `WriteHeader(200)` was already called (e.g., ffmpeg `Start()` fails), the error path calls `http.Error` which tries to write a 503 — but 200 was already sent. Audit this race.
12. Move `cmd.Start()` before `WriteHeader(200)` so that start failures return 503 before any status is sent.
13. Consider using `http.ResponseController` for all write-deadline and flush operations consistently.

### Code Quality

14. Update `AGENTS.md` with the "always WriteHeader+Flush before streaming" convention.
15. Modernize `b.N` to `b.Loop()` in `sse_test.go` benchmarks (2 warnings).
16. Run `nix flake check` to verify the Nix build still works with the change.
17. Run `govulncheck` to verify no new vulnerabilities introduced (CI step, but good to verify locally).
18. Check if `writeFrames` should also flush after the `--frame` separator write, not just after the frame body.

### Streaming Improvements

19. Add a `Content-Length: 0` or chunked transfer encoding hint to the stream response for better HTTP/2 compatibility.
20. Consider sending an initial empty boundary frame (`--frame\r\n\r\n`) before the first real frame to signal stream readiness faster.
21. Add a stream timeout/keepalive — if no frame arrives in N seconds, send a keepalive comment to prevent proxy timeouts.
22. Log stream start/end with request ID for debugging.
23. Add metrics for "time to first frame" (TTFF) — currently only `recordStreamDuration` exists.

### General

24. Review all `http.Error` calls after potential `WriteHeader` to ensure they don't double-write.
25. Add a lint rule or custom analyzer that flags `responseWriter.Header().Set(...)` without a following `WriteHeader` in streaming handlers.
26. Consider moving `setupStream` logic into a `StreamHandler` type for better testability.
27. Document the MJPEG streaming protocol in `docs/` for contributors.
28. Add a test for concurrent stream requests (semaphore already tested, but the header race is not).
29. Verify the stream cleanup path (`cleanupFFmpeg`) works correctly when `WriteHeader` was sent but no frames were written.
30. Add a test for client disconnect during stream setup (context cancelled between `WriteHeader` and first frame).

---

## g) Questions I Cannot Answer Myself

1. **Should `cmd.Start()` happen before `WriteHeader(200)`?** Currently the order is: LookPath → flusher check → SetWriteDeadline → cmd.Start() → WriteHeader. If `cmd.Start()` fails, we return `errStreamStart` (503) — but `WriteHeader` hasn't been called yet, so this is fine. However, if `cmd.Start()` succeeds but ffmpeg immediately exits, `writeFrames` will get an EOF from the pipe and return silently — the client gets a 200 with no frames. Is that acceptable behavior, or should we wait for the first frame before committing to 200?

2. **Should the test environment have ffmpeg in PATH?** The test name says "NoFFmpeg" but the comment says "ffmpeg likely not in PATH during test." On the CI runner, ffmpeg _is_ available (Nix devShell includes it). Should we make the test deterministic by explicitly setting `PATH` to exclude ffmpeg, or by mocking `exec.LookPath`?

3. **Is the `multipart/x-mixed-replace` content type correct for all clients?** Some browsers/proxies handle this poorly. Should we consider Server-Sent Events or WebSocket as an alternative for the live preview, or is MJPEG the right choice for camera preview?
