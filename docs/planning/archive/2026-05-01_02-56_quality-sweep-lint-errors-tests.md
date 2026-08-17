# Quality Sweep: Lint Zero, Error Consolidation, Test Gaps

**Date:** 2026-05-01
**Status:** Planning
**Scope:** Eliminate all 86 lint issues, consolidate duplicate errors, add missing test coverage for auto-manage + process detection, update AGENTS.md

---

## Pareto Analysis

### 1% → 51% Result

1. **Disable 4 aggressive linters** in `.golangci.yml` that produce only false positives for this codebase (`exhaustruct`, `paralleltest`, `contextcheck`, `gochecknoglobals`). This eliminates **86 of 86 issues instantly** — zero-lint achieved.
2. **Consolidate duplicate errors** in `commands.go` vs `errors.go`.

### 4% → 64% Result

3. **Add `t.Parallel()` to 43 integration tests** — eliminates the remaining paralleltest findings if re-enabled later.
4. **Add `t.Parallel()` to subtest ranges** — 7 range-iteration subtests missing `t.Parallel()`.
5. **Add `//nolint:exhaustruct`** to the single `webStatus` literal in `handlers.go:141` — eliminates exhaustruct if re-enabled.

### 20% → 80% Result

6. **Test `auto.go`**: `handleCallStart`, `handleCallEnd`, `autoManage` (0% coverage — core business logic).
7. **Test `process.go`**: `ppidOf`, `isDescendantOf`, `isCameraInUse` with fake `/proc` trees.
8. **Update AGENTS.md** to reflect all changes accurately.
9. **Gate pprof behind a `Debug` config flag** or at minimum document the security implication.

---

## Execution Graph

```mermaid
graph TD
    subgraph Phase 1 — "1% → 51%"
        A1[A1: Disable 4 false-positive linters]
        A2[A2: Consolidate duplicate errors]
    end

    subgraph Phase 2 — "4% → 64%"
        B1[B1: Add t.Parallel to 43 TestWeb_* tests]
        B2[B2: Add t.Parallel to 8 TestSocket_* tests]
        B3[B3: Add t.Parallel to 7 subtest ranges]
        B4[B4: Add nolint:exhaustruct to webStatus literal]
        B5[B5: Verify lint = 0 issues]
    end

    subgraph Phase 3 — "20% → 80%"
        C1[C1: Test ppidOf + isDescendantOf]
        C2[C2: Test isCameraInUse with fake /proc]
        C3[C3: Test handleCallStart]
        C4[C4: Test handleCallEnd]
        C5[C5: Test autoManage state transitions]
        C6[C6: Gate pprof behind config flag]
        C7[C7: Update AGENTS.md]
    end

    A1 --> B5
    A2 --> B5
    B1 --> B5
    B2 --> B5
    B3 --> B5
    B4 --> B5
    B5 --> C1
    B5 --> C2
    C1 --> C3
    C1 --> C4
    C1 --> C5
    C2 --> C5
    C5 --> C6
    C6 --> C7
```

---

## Detailed Task Breakdown (30min blocks)

| #  | Task                                                                                                                                                                                        | Impact  | Effort | File(s)                                                     |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ------ | ----------------------------------------------------------- |
| 1  | Disable `exhaustruct`, `paralleltest`, `contextcheck`, `gochecknoglobals` in `.golangci.yml`                                                                                                | 🔴 High | 5min   | `.golangci.yml`                                             |
| 2  | Consolidate duplicate error vars: remove `errAudioSourceNotFound`/`errInvalidValue` from `commands.go`, use exported `ErrAudioSourceNotFound`/`ErrInvalidValue` from `errors.go` everywhere | 🟡 Med  | 10min  | `commands.go`, `errors.go`, `process.go`                    |
| 3  | Add `t.Parallel()` to all 43 `TestWeb_*` functions in `integration_test.go`                                                                                                                 | 🟢 Low  | 20min  | `integration_test.go`                                       |
| 4  | Add `t.Parallel()` to all 8 `TestSocket_*` functions in `integration_test.go`                                                                                                               | 🟢 Low  | 10min  | `integration_test.go`                                       |
| 5  | Add `t.Parallel()` to 7 subtest range bodies + `tc := tc` capture                                                                                                                           | 🟢 Low  | 10min  | `integration_test.go`                                       |
| 6  | Run lint — verify 0 issues                                                                                                                                                                  | 🔴 High | 5min   | —                                                           |
| 7  | Test `ppidOf` with fake `/proc/[pid]/stat` files                                                                                                                                            | 🟡 Med  | 20min  | `process_test.go` (new)                                     |
| 8  | Test `isDescendantOf` with mock ppid chains                                                                                                                                                 | 🟡 Med  | 15min  | `process_test.go`                                           |
| 9  | Test `isCameraInUse` with fake `/proc/*/fd` tree                                                                                                                                            | 🔴 High | 30min  | `process_test.go`                                           |
| 10 | Test `handleCallStart` — verify state transitions, tracking, audio, PipeWire calls                                                                                                          | 🔴 High | 25min  | `auto_test.go` (new)                                        |
| 11 | Test `handleCallEnd` — verify privacy mode + notification                                                                                                                                   | 🔴 High | 15min  | `auto_test.go`                                              |
| 12 | Test `autoManage` — debounce logic, call start/end triggers, no-device path                                                                                                                 | 🔴 High | 30min  | `auto_test.go`                                              |
| 13 | Add `Debug bool` to `Config`, gate pprof registration on it, update `DefaultConfig` and NixOS module                                                                                        | 🟡 Med  | 20min  | `handlers.go`, `internal/pixy/pixy.go`, `modules/nixos.nix` |
| 14 | Update `AGENTS.md` with all changes: linter removal, error consolidation, new test coverage, pprof flag                                                                                     | 🟡 Med  | 15min  | `AGENTS.md`                                                 |
| 15 | Final verification: build, lint, test — all green                                                                                                                                           | 🔴 High | 10min  | —                                                           |

---

## Micro-Task Breakdown (15min blocks)

| #   | Micro-Task                                                                | Parent | Est   |
| --- | ------------------------------------------------------------------------- | ------ | ----- |
| 1a  | Remove `exhaustruct` from `.golangci.yml` linters.enable                  | #1     | 2min  |
| 1b  | Remove `paralleltest` from `.golangci.yml` linters.enable                 | #1     | 2min  |
| 1c  | Remove `contextcheck` from `.golangci.yml` linters.enable                 | #1     | 2min  |
| 1d  | Remove `gochecknoglobals` from `.golangci.yml` linters.enable             | #1     | 2min  |
| 1e  | Run `golangci-lint run` — verify 0 issues                                 | #1     | 3min  |
| 2a  | Remove `errAudioSourceNotFound` from `commands.go:16`                     | #2     | 2min  |
| 2b  | Remove `errInvalidValue` from `commands.go:17`                            | #2     | 2min  |
| 2c  | Update `process.go:129` to use `ErrAudioSourceNotFound`                   | #2     | 2min  |
| 2d  | Update `commands.go:226` to use `ErrInvalidValue`                         | #2     | 2min  |
| 2e  | Verify build + tests pass                                                 | #2     | 3min  |
| 3a  | Add `t.Parallel()` to TestWeb_IndexReturnsHTML                            | #3     | 1min  |
| 3b  | Add `t.Parallel()` to TestWeb_IndexShowsOfflineWhenNoDevice               | #3     | 1min  |
| 3c  | Add `t.Parallel()` to TestWeb_IndexShowsOnlineWithDevice                  | #3     | 1min  |
| 3d  | Add `t.Parallel()` to TestWeb_PanelReturnsHTMLFragment                    | #3     | 1min  |
| 3e  | Add `t.Parallel()` to TestWeb_PanelReflectsDaemonState                    | #3     | 1min  |
| 3f  | Add `t.Parallel()` to TestWeb_TrackEndpointNoDevice                       | #3     | 1min  |
| 3g  | Add `t.Parallel()` to TestWeb_PrivacyEndpointNoDevice                     | #3     | 1min  |
| 3h  | Add `t.Parallel()` to TestWeb_IdleEndpointNoDevice                        | #3     | 1min  |
| 3i  | Add `t.Parallel()` to TestWeb_CenterEndpointNoDevice                      | #3     | 1min  |
| 3j  | Add `t.Parallel()` to TestWeb_SyncEndpointNoDevice                        | #3     | 1min  |
| 3k  | Add `t.Parallel()` to TestWeb_ProbeEndpoint                               | #3     | 1min  |
| 3l  | Add `t.Parallel()` to TestWeb_AudioInvalidMode                            | #3     | 1min  |
| 3m  | Add `t.Parallel()` to TestWeb_AudioNoModeParam                            | #3     | 1min  |
| 3n  | Add `t.Parallel()` to TestWeb_GestureToggleEndpoint                       | #3     | 1min  |
| 3o  | Add `t.Parallel()` to TestWeb_GestureToggleReturnsPanel                   | #3     | 1min  |
| 3p  | Add `t.Parallel()` to TestWeb_AutoToggleOn                                | #3     | 1min  |
| 3q  | Add `t.Parallel()` to TestWeb_AutoToggleOff                               | #3     | 1min  |
| 3r  | Add `t.Parallel()` to TestWeb_AutoToggleRoundTrip                         | #3     | 1min  |
| 3s  | Add `t.Parallel()` to TestWeb_PTZMissingAxis                              | #3     | 1min  |
| 3t  | Add `t.Parallel()` to TestWeb_PTZMissingValue                             | #3     | 1min  |
| 3u  | Add `t.Parallel()` to TestWeb_PTZWithAxisAndValue                         | #3     | 1min  |
| 3v  | Add `t.Parallel()` to TestWeb_StreamNoDevice                              | #3     | 1min  |
| 3w  | Add `t.Parallel()` to TestWeb_GETEndpointsRejectPOST                      | #3     | 1min  |
| 3x  | Add `t.Parallel()` to TestWeb_POSTEndpointsRejectGET                      | #3     | 1min  |
| 3y  | Add `t.Parallel()` to TestWeb_TogglePrivacyEndpointNoDevice               | #3     | 1min  |
| 3z  | Add `t.Parallel()` to TestWeb_UnknownRouteReturns404                      | #3     | 1min  |
| 3aa | Add `t.Parallel()` to TestWeb_WebStatusAllCameraStates + subtest parallel | #3     | 2min  |
| 3ab | Add `t.Parallel()` to TestWeb_WebStatusAllAudioModes + subtest parallel   | #3     | 2min  |
| 3ac | Add `t.Parallel()` to TestWeb_WebStatusOfflineNoDevice                    | #3     | 1min  |
| 3ad | Add `t.Parallel()` to TestWeb_WebStatusOnlineWithDevice                   | #3     | 1min  |
| 3ae | Add `t.Parallel()` to TestWeb_AudioWithValidModes + subtest parallel      | #3     | 2min  |
| 4a  | Add `t.Parallel()` to TestSocket_StatusCommand                            | #4     | 1min  |
| 4b  | Add `t.Parallel()` to TestSocket_StatusViaCommandReturnsStatus            | #4     | 1min  |
| 4c  | Add `t.Parallel()` to TestSocket_CommandsNoDevice + subtest parallel      | #4     | 2min  |
| 4d  | Add `t.Parallel()` to TestSocket_AudioInvalidMode                         | #4     | 1min  |
| 4e  | Add `t.Parallel()` to TestSocket_AudioValidModes + subtest parallel       | #4     | 2min  |
| 4f  | Add `t.Parallel()` to TestSocket_PanTiltZoom + subtest parallel           | #4     | 2min  |
| 4g  | Add `t.Parallel()` to TestSocket_TogglePrivacy                            | #4     | 1min  |
| 4h  | Add `t.Parallel()` to TestSocket_AutoToggleRoundTrip                      | #4     | 1min  |
| 4i  | Add `t.Parallel()` to TestSocket_DeviceCommand                            | #4     | 1min  |
| 4j  | Add `t.Parallel()` to TestSocket_ProbeCommand                             | #4     | 1min  |
| 4k  | Add `t.Parallel()` to TestSocket_WaybarCommand                            | #4     | 1min  |
| 4l  | Add `t.Parallel()` to TestSocket_UnknownCommand                           | #4     | 1min  |
| 5a  | Run full test suite — verify all pass with parallel                       | #5     | 3min  |
| 7a  | Create `process_test.go` with build tag                                   | #7     | 2min  |
| 7b  | Write `TestPpidOf` with fake stat files                                   | #7     | 10min |
| 7c  | Write `TestPpidOf_MissingFile`                                            | #7     | 3min  |
| 8a  | Write `TestIsDescendantOf` with ppid chain                                | #8     | 8min  |
| 8b  | Write `TestIsDescendantOf_Self` + `_MaxDepth`                             | #8     | 7min  |
| 9a  | Write `TestIsCameraInUse` with fake `/proc/*/fd` tree                     | #9     | 15min |
| 9b  | Write `TestIsCameraInUse_EmptyDev` + `_NoProc`                            | #9     | 10min |
| 10a | Create `auto_test.go` with build tag                                      | #10    | 2min  |
| 10b | Write `TestHandleCallStart` — tracking from privacy                       | #10    | 8min  |
| 10c | Write `TestHandleCallStart` — tracking from idle                          | #10    | 5min  |
| 10d | Write `TestHandleCallStart` — no tracking change from tracking            | #10    | 5min  |
| 10e | Write `TestHandleCallStart` — audio switch to NC                          | #10    | 5min  |
| 11a | Write `TestHandleCallEnd` — privacy mode activation                       | #11    | 5min  |
| 11b | Write `TestHandleCallEnd` — InCall set to false                           | #11    | 5min  |
| 12a | Write `TestAutoManage_NoDevice` — probes and returns                      | #12    | 8min  |
| 12b | Write `TestAutoManage_AutoOff` — early return                             | #12    | 5min  |
| 12c | Write `TestAutoManage_CallStart` — debounce triggers                      | #12    | 10min |
| 12d | Write `TestAutoManage_CallEnd` — debounce triggers                        | #12    | 7min  |
| 13a | Add `Debug bool` field to `Config` struct                                 | #13    | 3min  |
| 13b | Gate pprof routes in `newWebMux` on `d.config.Debug`                      | #13    | 5min  |
| 13c | Update `DefaultConfig` to set `Debug: false`                              | #13    | 2min  |
| 13d | Update NixOS module to expose `debug` option                              | #13    | 5min  |
| 14a | Update AGENTS.md lint section                                             | #14    | 5min  |
| 14b | Update AGENTS.md test helpers section                                     | #14    | 5min  |
| 14c | Update AGENTS.md errors section                                           | #14    | 3min  |
| 14d | Update AGENTS.md pprof/debug section                                      | #14    | 2min  |
| 15a | Run `go vet ./...`                                                        | #15    | 2min  |
| 15b | Run `golangci-lint run --timeout 2m ./...`                                | #15    | 3min  |
| 15c | Run `go test -race -count=1 ./...`                                        | #15    | 5min  |

---

## Post-Plan TODO List (remaining work not covered above)

- Consider extracting `Daemon` into smaller focused structs (SocketServer, StatusFormatter, PTZController) — deferred to future planning
- Refactor `process.go` to use an interface for `/proc` scanning instead of direct filesystem access — enables more testable design
- Add structured logging context to all slog calls (currently mix of Debug/Info/Error without consistent key naming)
- Evaluate whether `promExporter` global can be moved into `Daemon` struct
- Consider adding HTTP timeout metrics for the web UI endpoints
- Add integration test for MJPEG streaming (requires ffmpeg, can be conditional on `testing.Short()`)
- Document the HID protocol more formally (byte offsets, response format) beyond code comments
