> **SUPERSEDED** — This report is superseded by `2026-08-03_03-48_Simulator-Gap-Closure-Round2-SelfReview.md`.
> All items listed as NOT STARTED here were closed in the subsequent gap-closure rounds.

# Status Report: PIXY HID Simulator — Gap Closure Session

**Date:** 2026-08-03 03:16
**Session Goal:** Close all critical gaps identified in the prior simulator status report (circuit breaker, timing, byte-layout, pointer fix, AGENTS.md trim)
**Prior Commit:** `69da92d` — initial simulator (31 tests)
**This Session:** Uncommitted (auto-commit daemon will capture)

---

## Executive Summary

Follow-up session that closed **6 of 8** high-value gaps from the prior status report. Added 8 new test functions (39 total, 57 with subtests), added `commitErr` failure injection mode and `sentTimestamps` recording to the simulator, fixed the `pixyProtocolState` value-vs-pointer design issue, and trimmed AGENTS.md from 398 to exactly 377 lines. All tests pass with `-race`. Lint clean (0 issues). Vet clean.

**However**, the session has notable gaps: the work is uncommitted, the circuit breaker tests don't exercise the re-probe path, there are no concurrent stress tests, and several testing opportunities from the prior report were not addressed.

---

## a) FULLY DONE

| Item                                   | Details                                                                                                                                                                                                                                                                                                                                                                                  |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Pointer fix                            | Changed `state pixyProtocolState` → `state *pixyProtocolState` in `pixySimulator`. Eliminates `sync.Mutex` copy concern. `newPixySimulator()` now calls `newPixyProtocolState()` directly (no dereference).                                                                                                                                                                              |
| `commitErr` failure injection          | New field on `pixySimulator` that fails ONLY commit reports while allowing config to succeed. Enables testing the "config ok, commit fails" path which increments `hidFailCount` after the 200ms sleep.                                                                                                                                                                                  |
| `sentTimestamps` recording             | New `[]time.Time` slice paired with `sentReports`. `Send()` records `time.Now()` before any processing. `SentTimestamps()` accessor enables timing assertions.                                                                                                                                                                                                                           |
| Circuit breaker: accumulation test     | Pre-loads `hidFailCount = threshold-1`, sets `sendErr`, calls `setTracking`. Verifies: count reaches threshold, next call returns `ErrPIXYNotConnected`, only 1 report recorded (circuit blocks before Send).                                                                                                                                                                            |
| Circuit breaker: reset-on-success test | Pre-loads `hidFailCount = 1`, successful `setTracking` call. Verifies count resets to 0 and simulator state matches.                                                                                                                                                                                                                                                                     |
| Circuit breaker: commit-failure test   | Sets `commitErr`, calls `setTracking`. Verifies: error wraps `commitErr`, `hidFailCount = 1`, 2 reports recorded (config+commit), daemon state unchanged (commit failed).                                                                                                                                                                                                                |
| 200ms sleep timing test                | Calls `setTracking`, reads `SentTimestamps()`, asserts gap between config and commit ≥ 190ms (200ms with 10ms scheduling slack).                                                                                                                                                                                                                                                         |
| `pixyConfig()` byte-layout table test  | 8 subtests covering all 3 interfaces × all valid mode bytes. Asserts positions 0-8: prefix, interface, markers, zero-pads, mode byte.                                                                                                                                                                                                                                                    |
| `pixyCommit()` byte-layout table test  | 3 subtests (tracking/audio/gesture). Asserts 4-byte layout: prefix, interface, marker, interface-repeated.                                                                                                                                                                                                                                                                               |
| Audio protocol bytes test              | Daemon integration: `setAudio(Live)` → verify config[1]=audio, config[8]=live, commit[1]=audio, commit[3]=audio.                                                                                                                                                                                                                                                                         |
| Gesture protocol bytes test            | Daemon integration: `setGesture(true)` → verify config[1]=gesture, config[8]=enabled, commit bytes.                                                                                                                                                                                                                                                                                      |
| AGENTS.md trim                         | Removed 12 obsolete/changelog entries (cmdMu, handler extraction, waybarJSON perf, v4l2SetMultiple removed, ParseAudioMode, AutoMode.Toggle, /api/health, device paths, audio toast, temp file cleanup, justfile removed, TestAutoManage skip, No init(), handleHealth, checkDevice, findSource, Removed templates). Updated test count 31→39. Updated simulator gotcha with new fields. |
| Lint                                   | 0 issues (`golangci-lint run --timeout 2m ./...`)                                                                                                                                                                                                                                                                                                                                        |
| Vet                                    | Clean                                                                                                                                                                                                                                                                                                                                                                                    |
| Race detector                          | All tests pass with `-race -count=1`                                                                                                                                                                                                                                                                                                                                                     |
| Status report updated                  | `docs/status/2026-08-03_02-52_PIXY-HID-Protocol-Simulator.md` updated with resolved items                                                                                                                                                                                                                                                                                                |

---

## b) PARTIALLY DONE

| Item                          | What Exists                                                         | What's Missing                                                                                                                                                                                                                                                     |
| ----------------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Circuit breaker test coverage | 3 tests covering accumulation, reset, commit-failure                | No test for the **re-probe path** (`hidFailCount < threshold` triggers `probeDevices()` on config failure — line 41 of `device.go`). This path calls real `probeDevices()` which scans sysfs. The test uses `sendErr` but doesn't verify the re-probe side effect. |
| Failure injection modes       | `sendErr`, `commitErr`, `sendRecvErr`, `nilResponse`, `corruptResp` | No "config-ok-but-commit-returns-garbage" mode. No way to fail only the Nth call (flaky device simulation). No `queryErr` (SendRecv-specific error that isn't nil or corrupt).                                                                                     |
| Timing verification           | 200ms gap between config and commit verified                        | No test for `hidResponseTimeout` (500ms) on `SendRecv`. No test verifying that `context.Cancel` during the 200ms sleep propagates correctly.                                                                                                                       |
| Byte-layout tests             | `pixyConfig()` and `pixyCommit()` outputs tested directly           | `buildResponse()` output only tested via `SendRecv` round-trip, not as a direct byte-layout table test. Gesture response byte at `hidRespBufSize-1` not independently verified.                                                                                    |
| AGENTS.md                     | Trimmed to 377 lines exactly                                        | At the absolute limit — any future addition requires another trim. No buffer.                                                                                                                                                                                      |

---

## c) NOT STARTED

| Item                                            | Why It Matters                                                                                                                                                                                           |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`queryHIDState[T]` generic wrapper test**     | Queries go through `queryHIDState[T]` → `SendRecv` → `parseHIDResponse`. Tested parts separately but never the full generic wrapper with type inference.                                                 |
| **Concurrent stress test**                      | `-race` passes but no explicit test with multiple goroutines calling `Send`/`SendRecv` simultaneously. The simulator has two mutexes (`state.mu` + `s.mu`) — a concurrent test would verify no deadlock. |
| **Auto-manage lifecycle with simulator**        | `auto_test.go` uses `withFakeDevices()`. Using `withPixySimulator()` would exercise real HID during `handleCallStart`/`handleCallEnd`.                                                                   |
| **Fuzz test for `handleConfig`/`handleCommit`** | Project has fuzz tests for `parseHIDResponse`. Fuzzing the simulator's handlers would catch panics on random byte inputs.                                                                                |
| **Benchmark for simulator round-trip**          | Project has 8 benchmarks. `BenchmarkSimulatorRoundTrip` would measure config→commit→query latency.                                                                                                       |
| **`/dev/uhid` virtual device (Layer 2)**        | Creates real `/dev/hidraw*` from userspace. Tests actual file I/O, `os.OpenFile`, `hidFile.Write`/`Read` paths. Reuses `pixyProtocolState`.                                                              |
| **NixOS VM test (Layer 3)**                     | QEMU VM testing systemd service lifecycle, NixOS module config, socket creation, `sd_notify`.                                                                                                            |
| **`vivid` V4L2 test driver**                    | Kernel module providing a virtual video device. Would enable real `v4l2-ctl` PTZ round-trip testing without hardware.                                                                                    |
| **Re-probe side effect test**                   | `setDeviceState` line 41: `d.applyProbeResultLocked(probeDevices())` on config failure when `hidFailCount < threshold`. Completely untested via simulator.                                               |
| **Context cancellation during 200ms sleep**     | `setDeviceState` has `select { case <-ctx.Done() / case <-time.After(200ms) }`. No test cancels context during the sleep.                                                                                |
| **`hidResponseTimeout` (500ms) test**           | `SendRecv` has a `time.NewTimer(hidResponseTimeout)` that returns `nil, nil` on timeout. The simulator's `nilResponse` tests the nil-return path but not the timeout mechanism itself.                   |
| **Write-then-read syncState round-trip**        | Current `syncState` test pre-sets simulator state directly. No test does: set via daemon → change on simulator → sync detects change.                                                                    |
| **Circuit breaker recovery test**               | No test for: circuit opens → `probeDevices()` finds device → `hidFailCount` resets to 0 → commands work again. The full recovery loop.                                                                   |

---

## d) TOTALLY FUCKED UP

| Item                                                  | What Happened                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | Severity |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| **Work is UNCOMMITTED**                               | The auto-commit daemon may or may not have captured these changes. I did not explicitly commit. All work (pointer fix, new fields, 8 new tests, AGENTS.md trim, status report update) exists only in the working tree. If the daemon hasn't run, this is unsaved work.                                                                                                                                                                                                                             | **HIGH** |
| **Circuit breaker accumulation test pre-loads count** | `TestSimulator_CircuitBreaker_Accumulation` sets `d.hidFailCount = hidCircuitBreakerThreshold - 1` directly instead of driving 3 real failures. This bypasses the `applyProbeResultLocked(probeDevices())` side effect that fires on the 1st and 2nd failures. The test proves the circuit opens but doesn't prove the accumulation path through real `setDeviceState` calls works end-to-end. A real test would call `setTracking` 3 times with `sendErr` set, handling the re-probe side effect. | Medium   |
| **No re-probe handling in test setup**                | When `sendErr` fires and `hidFailCount < threshold`, `setDeviceState` calls `probeDevices()` which scans real sysfs. On a machine with a real PIXY connected, this could find the device and reset `hidDev` to a real `hidrawDevice`, breaking the test. The test implicitly assumes no PIXY is connected. This should be documented or mocked.                                                                                                                                                    | Medium   |
| **AGENTS.md at hard limit (377/377)**                 | Trimmed to exactly the maximum. Zero buffer for future additions. The trim removed some entries that were low-value but not zero-value (e.g., `/api/health` endpoint, `waybarJSON` optimization numbers). A future session adding documentation will immediately hit the limit.                                                                                                                                                                                                                    | Low      |

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Commit the work explicitly** — Relying on the auto-commit daemon is risky. The changes should be explicitly committed with a descriptive message.

2. **Mock `probeDevices` in circuit breaker tests** — The circuit breaker tests work because CI has no PIXY connected. On a developer machine with hardware, `probeDevices()` returns a real device and overwrites `hidDev`. The test should inject a mock prober or document the hardware assumption with `t.Skip()`.

3. **Add `queryErr` failure injection** — Currently `sendRecvErr` covers all SendRecv failures, but there's no way to simulate "query returns valid bytes but wrong interface" or "query returns truncated response". A `queryErr` field would enable testing `queryHIDState` error paths.

4. **Test the `buildResponse` byte layout directly** — Like `TestPixyConfig_ByteLayout` and `TestPixyCommit_ByteLayout`, there should be a `TestBuildResponse_ByteLayout` table test verifying response bytes for all interfaces × all states. Currently only tested via round-trip.

5. **Gesture response last-byte assertion** — `buildResponse` for gesture sets `resp[hidRespBufSize-1] = gestureEnabledByte`. This is tested indirectly via `parseHIDResponse` round-trip but never directly asserted in a byte-layout test.

6. **Remove `sentTimestamps` race risk** — `Send()` records `time.Now()` before locking `s.mu`, then appends inside the lock. The `now := time.Now()` call is safe (value type), but the pattern is inconsistent with how `sentReports` is recorded. Should record inside the lock for consistency.

7. **Consider moving `isCommitReport` to production code** — The heuristic is used only in the simulator's `Send()` method to dispatch config vs commit. But it encodes protocol knowledge that belongs alongside `pixyConfig`/`pixyCommit` in `hid.go`, not in a test file. If someone changes the commit format, the simulator's heuristic silently breaks.

### Testing

8. **Add concurrent stress test** — Spawn 10 goroutines calling `Send` and `SendRecv` in a tight loop. Verify no deadlock, no race, all reports recorded. The simulator has two separate mutexes that could theoretically deadlock if the lock order were wrong.

9. **Drive circuit breaker through real failures** — Instead of pre-loading `hidFailCount`, call `setTracking` 3 times with `sendErr`. This requires handling the `probeDevices()` side effect (inject a mock or skip on hardware).

10. **Test context cancellation during 200ms sleep** — Cancel context after sending config but before commit arrives. Verify `setDeviceState` returns `ctx.Err()` and the commit is never sent.

11. **Test `hidResponseTimeout` path** — The simulator should have a `delayResponse` mode that sleeps before responding, enabling a test that verifies `SendRecv` returns `nil, nil` after 500ms.

### Architecture

12. **Layer 2 (`/dev/uhid`) feasibility study** — Research whether Go can create uhid devices via syscall. If feasible, `pixyProtocolState` is ready to serve as the backend.

13. **Layer 3 (NixOS VM test) skeleton** — Create `tests/nixos-vm.nix` with a minimal `makeTest` that starts the daemon service and verifies the socket appears. Even a skeleton that always passes establishes the infrastructure.

---

## f) Next 50 Things to Get Done

### High Priority — Protocol & Safety (1-10)

| #   | Task                                                                                  | Impact   | Effort |
| --- | ------------------------------------------------------------------------------------- | -------- | ------ |
| 1   | Commit all session work explicitly with descriptive message                           | Critical | 2 min  |
| 2   | Mock `probeDevices` in circuit breaker tests (inject noop prober)                     | High     | 15 min |
| 3   | Drive circuit breaker test through 3 real `setTracking` failures                      | High     | 20 min |
| 4   | Test circuit breaker full recovery: open → probe finds device → reset → commands work | High     | 25 min |
| 5   | Test context cancellation during 200ms sleep                                          | High     | 10 min |
| 6   | Add concurrent stress test (10 goroutines, Send + SendRecv)                           | High     | 15 min |
| 7   | Add `TestBuildResponse_ByteLayout` table test (all interfaces × states)               | Medium   | 15 min |
| 8   | Direct assertion of gesture response last byte (`resp[63]`)                           | Medium   | 5 min  |
| 9   | Add `queryErr` failure injection to simulator                                         | Medium   | 10 min |
| 10  | Test `queryHIDState[T]` generic wrapper end-to-end                                    | Medium   | 15 min |

### Medium Priority — Test Depth (11-20)

| #   | Task                                                                                  | Impact | Effort |
| --- | ------------------------------------------------------------------------------------- | ------ | ------ |
| 11  | Write-then-read syncState round-trip test                                             | Medium | 15 min |
| 12  | Auto-manage lifecycle test using `withPixySimulator()` instead of `withFakeDevices()` | Medium | 20 min |
| 13  | Fuzz test for `handleConfig` (random bytes, verify no panic)                          | Medium | 15 min |
| 14  | Fuzz test for `handleCommit` (random bytes, verify no panic)                          | Medium | 15 min |
| 15  | `BenchmarkSimulatorRoundTrip` (config → commit → query)                               | Low    | 10 min |
| 16  | `BenchmarkSimulatorSendOnly` (config + commit, no query)                              | Low    | 10 min |
| 17  | Test `nilResponse` path through full `syncState` (not just `SendRecv`)                | Low    | 10 min |
| 18  | Test `corruptResp` path through full `syncState`                                      | Low    | 10 min |
| 19  | Add `delayResponse` field to simulator for timeout testing                            | Low    | 15 min |
| 20  | Test `SendRecv` with `hidResponseTimeout` (500ms) expiration                          | Low    | 15 min |

### Medium Priority — Simulator Enhancements (21-30)

| #   | Task                                                                          | Impact | Effort |
| --- | ----------------------------------------------------------------------------- | ------ | ------ |
| 21  | Add `failOnNthCall` counter for flaky device simulation                       | Low    | 15 min |
| 22  | Add `configErr` (config-only failure, distinct from `sendErr`)                | Low    | 10 min |
| 23  | Track `commitCount` and `configCount` separately for finer assertions         | Low    | 10 min |
| 24  | Add `LastCommitIface()` accessor to verify commit sequencing                  | Low    | 5 min  |
| 25  | Add `Reset()` method to clear simulator state between subtests                | Low    | 5 min  |
| 26  | Document `isCommitReport` invariant with assertion comment                    | Low    | 5 min  |
| 27  | Move `isCommitReport` to `hid.go` (production code, not test-only)            | Low    | 10 min |
| 28  | Add `SentReportsByType()` helper (returns configs and commits separately)     | Low    | 10 min |
| 29  | Add `QueriesByInterface()` helper (returns queries grouped by interface byte) | Low    | 10 min |
| 30  | Add `StateSnapshot()` method returning a value copy of current state          | Low    | 10 min |

### Lower Priority — Layer 2/3 Infrastructure (31-40)

| #   | Task                                                                          | Impact | Effort  |
| --- | ----------------------------------------------------------------------------- | ------ | ------- |
| 31  | Research Go `uhid` syscall feasibility (create `/dev/hidraw*` from userspace) | Medium | 30 min  |
| 32  | Prototype `/dev/uhid` virtual device using `pixyProtocolState` as backend     | Medium | 2 hours |
| 33  | Test `hidrawDevice.Send` against uhid-created device                          | Medium | 1 hour  |
| 34  | Test `hidrawDevice.SendRecv` against uhid-created device                      | Medium | 1 hour  |
| 35  | Create `tests/nixos-vm.nix` skeleton with `makeTest`                          | Medium | 30 min  |
| 36  | NixOS VM test: verify systemd service starts                                  | Medium | 1 hour  |
| 37  | NixOS VM test: verify unix socket creation                                    | Medium | 30 min  |
| 38  | NixOS VM test: verify `sd_notify READY=1`                                     | Low    | 30 min  |
| 39  | Research `vivid` kernel module for V4L2 testing                               | Low    | 30 min  |
| 40  | Prototype `vivid`-based PTZ round-trip test                                   | Low    | 2 hours |

### Lower Priority — Polish & Documentation (41-50)

| #   | Task                                                                                   | Impact | Effort |
| --- | -------------------------------------------------------------------------------------- | ------ | ------ |
| 41  | Give AGENTS.md a 20-line buffer (trim to ~357 lines)                                   | Low    | 15 min |
| 42  | Consolidate simulator documentation (planning doc + status report + AGENTS.md overlap) | Low    | 20 min |
| 43  | Add `README.md` section on testing strategy (Layers 1/2/3)                             | Low    | 15 min |
| 44  | Create `docs/testing.md` with simulator usage guide                                    | Low    | 20 min |
| 45  | Add `// Example` test function showing simulator usage                                 | Low    | 10 min |
| 46  | Verify simulator works with `go test -short` skip pattern                              | Low    | 10 min |
| 47  | Add CI step to run simulator tests separately with verbose output                      | Low    | 15 min |
| 48  | Add test coverage report for `pixy_simulator_test.go` specifically                     | Low    | 10 min |
| 49  | Review whether `pixyProtocolState` should be a separate package (`internal/pixysim/`)  | Low    | 20 min |
| 50  | Archive planning doc after all Layer 1 items are closed                                | Low    | 5 min  |

---

## g) Questions

1. **Should I commit this work now with an explicit message, or do you want to review the diff first?** The auto-commit daemon may have already captured it, but I'm not certain. The changes span `pixy_simulator_test.go` (pointer fix + new fields), `pixy_simulator_impl_test.go` (8 new tests), `AGENTS.md` (trim + doc updates), and the status report.

2. **Should `isCommitReport` move to production code (`hid.go`)?** It encodes protocol knowledge (commit reports have `report[3] == report[1]`) that currently lives only in a `_test.go` file. If the commit format changes, the simulator silently breaks. Moving it to `hid.go` alongside `pixyConfig`/`pixyCommit` would make the invariant explicit and testable. But it's currently only used by the simulator — YAGNI says leave it.

3. **Should I mock `probeDevices()` globally for all simulator tests, or only in circuit breaker tests?** Currently, circuit breaker tests with `sendErr` trigger `probeDevices()` on real sysfs. On CI (no hardware), this returns empty and the test passes. On a developer machine with a PIXY connected, it could find the device and overwrite `hidDev`, breaking the test. Options: (a) inject a noop prober in `withPixySimulator()`, (b) add `t.Skip()` when hardware detected, (c) document the assumption only.
