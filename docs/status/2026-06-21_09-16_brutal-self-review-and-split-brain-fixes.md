# Status Report — 2026-06-21 09:16

**Session focus:** Brutal self-review of post-cqrs-htmx-removal work + split brain fixes + Commander interface extension

---

## a) FULLY DONE ✅

| #   | Task                                                                     | Commit    | Impact   |
| --- | ------------------------------------------------------------------------ | --------- | -------- |
| 1   | Fixed nix build (vendorHash in package.nix + flake.nix)                  | `e34be9c` | Critical |
| 2   | Verified nix build + nix flake check pass                                | verified  | Critical |
| 3   | CHANGELOG updated with cqrs-htmx removal (breaking change)               | `37486b8` | High     |
| 4   | AGENTS.md comprehensive update (11 edits, all cqrs-htmx refs)            | `37486b8` | High     |
| 5   | CI cleanup: removed GOPRIVATE, added FuzzParsePTZValue + nix flake check | `37486b8` | High     |
| 6   | Split sse.go → sse.go (SSE-only) + http.go (HTTP helpers)                | `dd01955` | Medium   |
| 7   | 9 SSE unit tests (writeSSEEvent, Broadcaster, splitSSELines)             | `dd01955` | Medium   |
| 8   | TODO_LIST.md updated with Phase 9                                        | `739f0d4` | Medium   |
| 9   | Commander interface (centralized subprocess logging)                     | `aa07cbe` | High     |
| 10  | PTZ readback accuracy (cache update vs invalidation)                     | `847e418` | High     |
| 11  | SSE fuzz test + benchmarks                                               | `159af01` | Medium   |
| 12  | Replaced contains() with strings.IndexByte (removed allocation)          | `7642147` | Low      |
| 13  | testdata/ gitignored                                                     | `7642147` | Low      |
| 14  | ptzCacheTTL moved from handlers.go to cache.go                           | `7642147` | Low      |
| 15  | toastType moved from handlers.go to web_types.go                         | `7642147` | Low      |
| 16  | parsePTZValues now logs v4l2-ctl read failures                           | `7642147` | Medium   |
| 17  | Commander extended with LookPath; checkExternalDeps routed through it    | `7642147` | Medium   |

---

## b) PARTIALLY DONE 🟡

| Task                   | Status                               | What remains                                                                                   |
| ---------------------- | ------------------------------------ | ---------------------------------------------------------------------------------------------- |
| Commander completeness | 5 of 6 exec call sites use Commander | ffmpeg streaming intentionally excluded (needs StdoutPipe+Start, documented)                   |
| AGENTS.md accuracy     | Mostly accurate                      | Needs final update after split brain fixes (contains removed, toastType moved, LookPath added) |

---

## c) NOT STARTED ❌

| Task                                                      | Priority             |
| --------------------------------------------------------- | -------------------- |
| Camera preset support (save/recall PTZ positions)         | Medium — new feature |
| Mobile-responsive layout                                  | Low — UX improvement |
| Fake device test infrastructure (fake HID + fake video)   | Medium — test infra  |
| Real hardware integration tests (build tag guarded)       | Low                  |
| Move SSEEvent + toastType to internal/pixy (domain types) | Low — layering       |

---

## d) TOTALLY FUCKED UP 💥

1. **`contains()` reimplemented `strings.IndexByte`** — I wrote a custom function that allocated `[]byte(s)` instead of using the stdlib that operates directly on string backing bytes. This is a textbook "reinventing the wheel" mistake. The function existed since the SSE reimplementation and I never caught it until this review.

2. **`parsePTZValues` silently swallowed errors** — When v4l2-ctl failed, the function returned `pixy.PTZValues{}` with zero logging. This made hardware read failures completely invisible. The function silently degraded instead of informing the user. This was pre-existing but I propagated it through the Commander refactoring without fixing it.

3. **`ptzCacheTTL` was a split brain** — The constant lived in `handlers.go` but was used in `commands.go`, `ptz.go`, and tests. I introduced this cross-file dependency when I changed PTZ cache to use update instead of invalidation. Created the split brain and didn't notice.

4. **`toastType` was a split brain** — Type defined in `handlers.go` (implementation file), consumed in `web_types.go` (domain/types file). I created this branded type and put it in the wrong file.

---

## e) WHAT WE SHOULD IMPROVE 🔧

| Area                   | Issue                                                                      | Fix                                                                        |
| ---------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| **Stdlib awareness**   | `contains()` reinvented `strings.IndexByte`                                | Always check stdlib before writing utility functions                       |
| **Error visibility**   | `parsePTZValues` swallowed errors                                          | Always log at Warn when falling back to defaults                           |
| **Constant placement** | `ptzCacheTTL` in handlers.go instead of cache.go                           | Constants belong with the types they configure                             |
| **Type placement**     | `toastType` in handlers.go instead of web_types.go                         | Types belong with their consumers                                          |
| **Commander scope**    | ffmpeg intentionally excluded (needs streaming API)                        | Documented — acceptable design constraint                                  |
| **Daemon struct**      | 17 fields (2 mutexes, state, device, caches, deps)                         | Acceptable for a daemon — decomposition would add complexity without value |
| **Test coverage**      | 72.5% — gaps in HID/ffmpeg/netlink (hardware-specific)                     | Acceptable for hardware daemon — integration test infra would help         |
| **json/v2**            | Available in Go 1.26.3 source but requires `goexperiment.jsonv2` build tag | Not usable until Go team stabilizes it                                     |

---

## f) Top #25 Things We Should Get Done Next

| #   | Task                                                                  | Impact | Effort |
| --- | --------------------------------------------------------------------- | ------ | ------ |
| 1   | Camera preset: define `Preset` type + add to `State`                  | Medium | 10min  |
| 2   | Camera preset: add CLI commands (`preset save/load/list`)             | Medium | 15min  |
| 3   | Camera preset: add web UI buttons + HTMX wiring                       | Medium | 15min  |
| 4   | Camera preset: add tests                                              | Medium | 10min  |
| 5   | Fake HID device for tests (implements `HIDDevice` interface)          | Medium | 20min  |
| 6   | Wire fake devices into `newTestDaemon`                                | Medium | 10min  |
| 7   | Integration tests using fakes (HID round-trip, stream lifecycle)      | Medium | 15min  |
| 8   | Mobile-responsive CSS (media queries, touch-friendly sliders)         | Low    | 20min  |
| 9   | Move `SSEEvent` to `internal/pixy` (domain type extraction)           | Low    | 10min  |
| 10  | Evaluate `otter/v2` for ptzCache (currently single-entry hand-rolled) | Low    | 15min  |
| 11  | Real hardware integration tests (`//go:build integration` tag)        | Low    | 20min  |
| 12  | Add `nix run .#test` to CI for reproducible test runs                 | Low    | 10min  |
| 13  | Consider structured error types (`cockroachdb/errors`)                | Low    | 20min  |
| 14  | Evaluate `samber/mo` `Result[T]` for `CommandResult`                  | Low    | 15min  |
| 15  | Review Daemon struct decomposition (17 fields)                        | Low    | 15min  |
| 16  | PTZ slider keyboard accessibility (arrow keys, ARIA labels)           | Low    | 10min  |
| 17  | Add retry logic for HID operations on transient failures              | Low    | 15min  |
| 18  | Document the V4L2 multiplier system in AGENTS.md                      | Low    | 5min   |
| 19  | Consider WebSocket as SSE alternative (bidirectional)                 | Low    | 30min  |
| 20  | Add graceful degradation when v4l2-ctl is missing                     | Low    | 10min  |
| 21  | Add health check for ffmpeg availability                              | Low    | 5min   |
| 22  | Consider rate limiting for PTZ commands                               | Low    | 10min  |
| 23  | Add telemetry for PTZ usage patterns                                  | Low    | 10min  |
| 24  | Consider camera profile support (different camera configs)            | Low    | 30min  |
| 25  | Evaluate Waybar module templating for richer display                  | Low    | 15min  |

---

## g) Top #1 Question I CANNOT Figure Out Myself 🤔

**Should we move `SSEEvent` to `internal/pixy` as a domain type?**

`SSEEvent` is currently defined in `sse.go` (implementation file) alongside `Broadcaster` and `sseStream`. However, the `Daemon` struct broadcasts events from domain logic (`broadcastStateChanged` in `device.go`, `auto.go`). The `SSEEvent` struct crosses the domain/implementation boundary.

Arguments FOR moving to `internal/pixy`:

- Domain code (`broadcastStateChanged`) shouldn't reference implementation types
- Consistent with how `pixy.PTZValues`, `pixy.CameraState`, etc. live in the shared package

Arguments AGAINST:

- SSE is an HTTP implementation detail, not a domain concept
- `internal/pixy` currently has no HTTP imports and adding SSE types would blur the boundary
- The `Broadcaster` and `sseStream` should stay in `sse.go` regardless

**My recommendation:** Keep `SSEEvent` in `sse.go`. It's a transport-layer DTO, not a domain type. The domain code calling `Broadcast(SSEEvent{...})` is fine — it's the same pattern as calling `writeJSON(w, status, healthResponse)` (domain code producing HTTP responses). Moving it would create a false domain boundary.

---

## Build & Test Status

| Check                            | Status      |
| -------------------------------- | ----------- |
| `go build`                       | ✅ Pass     |
| `go test -race -count=1 ./...`   | ✅ Pass     |
| `golangci-lint run --timeout 2m` | ✅ 0 issues |
| `nix build`                      | ✅ Pass     |
| `nix flake check`                | ✅ Pass     |
| `templ generate`                 | ✅ Pass     |
| `gofumpt -l .`                   | ✅ Clean    |
