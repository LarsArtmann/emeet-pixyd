# Status Report: PIXY HID Simulator — Gap Closure Round 2 (Brutal Self-Review)

**Date:** 2026-08-03 03:48
**Session Goal:** Close remaining critical gaps from the prior gap-closure status report, then brutally self-review
**Prior State:** 39 test functions (57 with subtests), `commitErr` + `sentTimestamps` added, pointer fixed
**This Session:** 48 test functions (74 with subtests), all high-priority gaps closed, 1 code fix

---

## Executive Summary

This session added **9 new test functions** covering the realistic circuit breaker accumulation path (commit failures), context cancellation during the 200ms sleep, concurrent stress testing (10 goroutines), `buildResponse` byte-layout verification (6 subtests), gesture response last-byte direct assertion, `queryHIDState[T]` generic wrapper end-to-end (3 tests), and syncState write-then-read round-trip. Also fixed `sentTimestamps` lock consistency.

All tests pass with `-race`. Lint 0 issues. Vet clean. AGENTS.md updated (still 377 lines). Work was captured by the auto-commit daemon in commit `45693ee` (good message) and `c34ac8a` (blank message — see "TOTALLY FUCKED UP").

**Key architectural insight discovered:** Only the commit failure path (`device.go:55-65`) can naturally accumulate `hidFailCount` to the circuit breaker threshold. Config Send failures trigger `probeDevices()` which either resets the counter (device found) or nils `hidDev` (device not found). This makes `probeDevices` mocking unnecessary — the commit failure path is both simpler and more realistic to test.

---

## a) FULLY DONE

| Item                                                     | Details                                                                                                                                                                                                                                                                                                                        |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Circuit breaker: real accumulation via 3 commit failures | `TestSimulator_CircuitBreaker_RealAccumulationViaCommitFailures` — sets `commitErr`, drives 3 real `setTracking` calls, verifies count increments 1→2→3 and circuit opens on 4th call. hidDev stays intact because commit failures don't trigger re-probe. 6 reports recorded (3 config + 3 commit). Takes ~600ms (3 × 200ms). |
| Context cancellation during 200ms sleep                  | `TestSimulator_ContextCancellationDuringSleep` — cancels context 50ms after config Send succeeds. Verifies: error wraps `context.Canceled`, only 1 report sent (config only), `hidFailCount` stays 0 (cancellation is not a failure).                                                                                          |
| Concurrent stress test                                   | `TestSimulator_ConcurrentAccess` — 10 goroutines × 20 iterations doing Send (config+commit) + SendRecv. Verifies no deadlock/panic, all 400 reports + 200 queries recorded, final state consistent. Uses `sync.WaitGroup.Go` (Go 1.25+).                                                                                       |
| `buildResponse` byte-layout table test                   | `TestBuildResponse_ByteLayout` — 6 subtests: tracking (idle/tracking/privacy) + audio (nc/live/original). Asserts positions 0-8: prefix, interface, markers, mode byte.                                                                                                                                                        |
| Gesture response last byte                               | `TestBuildResponse_GestureLastByte` — 2 subtests (enabled/disabled). Directly asserts `resp[63]` is `gestureEnabledByte` when enabled, `0x00` when disabled.                                                                                                                                                                   |
| `queryHIDState[T]` generic wrapper                       | `TestSimulator_QueryHIDState_GenericWrapper` — full round-trip through the real generic function for all 3 interfaces (tracking→Privacy, audio→Original, gesture→true). Type inference exercised with different `T` types.                                                                                                     |
| `queryHIDState` error paths                              | `TestSimulator_QueryHIDState_NilResponse` (wraps `errNoHIDResponse`) + `TestSimulator_QueryHIDState_CorruptResponse` (wraps `errUnrecognizedHID`).                                                                                                                                                                             |
| syncState write-then-read round-trip                     | `TestSimulator_SyncState_WriteThenReadRoundTrip` — sets Privacy via daemon → changes simulator state directly (simulated button press to Tracking/Original/true) → sync detects drift → daemon state updated. Tests both write and read paths in sequence.                                                                     |
| `sentTimestamps` lock consistency                        | `time.Now()` moved inside `s.mu.Lock()` for consistency with `sentReports` recording pattern.                                                                                                                                                                                                                                  |
| AGENTS.md updated                                        | Test count 39→48, descriptions updated with new test names, circuit breaker insight added. Still 377 lines (edits were within-line).                                                                                                                                                                                           |
| Round-2 status report                                    | Written to `docs/status/2026-08-03_03-40_Simulator-Gap-Closure-Round2.md` (now superseded by this report).                                                                                                                                                                                                                     |
| Lint                                                     | 0 issues (`golangci-lint run --timeout 2m ./...`)                                                                                                                                                                                                                                                                              |
| Vet                                                      | Clean                                                                                                                                                                                                                                                                                                                          |
| Race detector                                            | All tests pass with `-race -count=1` (1.78s)                                                                                                                                                                                                                                                                                   |

---

## b) PARTIALLY DONE

| Item                          | What Exists                                                                                               | What's Missing                                                                                                                                                                                                                                                                                                                                                                                  |
| ----------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Circuit breaker test coverage | 4 tests: pre-loaded accumulation, reset-on-success, commit-failure, real accumulation via commit failures | No test for the **config-failure re-probe path** (line 41 of `device.go`: `applyProbeResultLocked(probeDevices())`). Analysis shows this path CAN'T naturally accumulate to threshold, but there's no test VERIFYING that claim. A test that sets `sendErr`, triggers one config failure, and verifies that `probeDevices()` was called + `hidDev` was nilled would make the analysis concrete. |
| Byte-layout tests             | `pixyConfig`, `pixyCommit`, and `buildResponse` all have direct table tests                               | `parseHIDResponse` only tested via round-trip through the simulator, not as a direct table test on raw byte inputs (though it IS fuzz-tested via `FuzzParseHIDResponse`).                                                                                                                                                                                                                       |
| Concurrency testing           | `TestSimulator_ConcurrentAccess` exercises the simulator directly (10 goroutines, Send + SendRecv)        | No test drives concurrent `setDeviceState` calls through the daemon (would exercise `d.mu` + `hidMu` lock interactions). Would take 200ms × N calls.                                                                                                                                                                                                                                            |
| Failure injection modes       | `sendErr`, `commitErr`, `sendRecvErr`, `nilResponse`, `corruptResp`                                       | No `delayResponse` mode (for testing `hidResponseTimeout` 500ms expiration). No `failOnNthCall` counter (for flaky device simulation).                                                                                                                                                                                                                                                          |

---

## c) NOT STARTED

| Item                                            | Why It Matters                                                                                                                                                                                                            |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Fuzz test for `handleConfig`/`handleCommit`** | Random byte inputs could find panics in the simulator's protocol validation. Project already has `FuzzParseHIDResponse`; fuzzing the simulator handlers would be the natural extension.                                   |
| **Benchmark for simulator round-trip**          | `BenchmarkSimulatorRoundTrip` (config → commit → query) would measure simulator overhead. Project has 8 benchmarks; this would be #9.                                                                                     |
| **`delayResponse` field for timeout testing**   | The simulator responds instantly. A `delayResponse time.Duration` field would enable testing `hidResponseTimeout` (500ms) expiration in `hidrawDevice.SendRecv`.                                                          |
| **Config-failure re-probe verification test**   | Verify that config Send failure + no device found → `hidDev` is nilled → subsequent calls return `ErrPIXYNotConnected` immediately. Currently only analyzed, not tested.                                                  |
| **Concurrent daemon-level test**                | Multiple goroutines calling `d.deps.setTracking`/`setAudio`/`setGesture` simultaneously through the daemon's `hidMu`. Would verify no deadlock between `d.mu` and `hidMu`.                                                |
| **`/dev/uhid` virtual device (Layer 2)**        | Creates real `/dev/hidraw*` from userspace via uhid syscall. Tests actual file I/O. `pixyProtocolState` is ready to serve as backend.                                                                                     |
| **NixOS VM test (Layer 3) skeleton**            | QEMU VM testing systemd service lifecycle, socket creation, `sd_notify`.                                                                                                                                                  |
| **`vivid` V4L2 test driver**                    | Kernel module for virtual video device. Would enable real `v4l2-ctl` PTZ round-trip testing.                                                                                                                              |
| **AGENTS.md 20-line buffer**                    | Currently at 377/377. Any future addition requires a trim. Should be reduced to ~357 for breathing room.                                                                                                                  |
| **Prior status reports not updated**            | `2026-08-03_03-16_Simulator-Gap-Closure.md` and `2026-08-03_02-52_PIXY-HID-Protocol-Simulator.md` still show items as "NOT STARTED" that are now done. Three simulator status reports now exist with overlapping content. |

---

## d) TOTALLY FUCKED UP

| Item                                                    | What Happened                                                                                                                                                                                                                                                                                                              | Severity                           |
| ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| **Auto-commit daemon produced garbage commit messages** | Commit `c34ac8a` has a **completely blank subject line**. Commit `83a5343` has subject starting with `)`. These are auto-commit daemon artifacts — the daemon parsed the commit message poorly. The work IS captured (no data loss), but the git history is polluted with unsearchable commits.                            | **HIGH** — git history quality     |
| **Three overlapping status reports**                    | `2026-08-03_02-52_PIXY-HID-Protocol-Simulator.md`, `2026-08-03_03-16_Simulator-Gap-Closure.md`, `2026-08-03_03-40_Simulator-Gap-Closure-Round2.md`, and now this file. Four status reports for essentially the same body of work, each superseding the last. Creates confusion about which is authoritative.               | **Medium** — documentation clutter |
| **No explicit commit**                                  | I relied entirely on the auto-commit daemon. I should have committed explicitly with a descriptive message after verifying all tests pass. The daemon split my work across two commits (`45693ee` for tests, `c34ac8a` for AGENTS.md + status report + lint fix), which is less clean than a single well-described commit. | **Medium** — process discipline    |
| **Concurrent test doesn't exercise 200ms sleep path**   | `TestSimulator_ConcurrentAccess` calls `sim.Send`/`sim.SendRecv` directly (bypassing the daemon). It verifies the simulator's own mutex safety, not the daemon's `hidMu`/`d.mu` lock interaction. A more valuable concurrent test would drive `d.deps.setTracking` from multiple goroutines.                               | **Low** — test depth               |
| **Didn't verify config-failure re-probe claim**         | I stated that "config Send failures can NEVER naturally accumulate to threshold" based on code analysis, but didn't write a test that PROVES it. The claim is correct (the re-probe either resets or nils), but "analyze + assert" is stronger than "analyze + assume."                                                    | **Low** — test completeness        |

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Move `isCommitReport` to production code** — It encodes protocol knowledge (commit reports have `report[3] == report[1]`) that currently lives only in `pixy_simulator_test.go`. If the commit format changes, the simulator's heuristic silently breaks. Moving it to `hid.go` alongside `pixyConfig`/`pixyCommit` makes the invariant explicit and testable. The function is pure (no side effects) and could be useful in production debugging.

2. **Add `parseHIDResponse` direct table test** — Currently only tested via round-trip through the simulator and via fuzz testing. A direct table test with crafted byte arrays (valid + invalid) would make the expected behavior explicit and catch regressions faster than fuzz testing.

3. **Consider `delayResponse` field** — Adding a `delayResponse time.Duration` to `pixySimulator` would enable testing `hidResponseTimeout` (500ms) without hardware. The simulator would sleep before responding, and the test would verify `SendRecv` returns `nil, nil` after timeout.

4. **`failOnNthCall` counter** — A counter that fails only the Nth Send/SendRecv call would enable testing flaky device recovery scenarios: fail once, succeed on retry, verify circuit breaker doesn't accumulate.

### Testing

5. **Concurrent daemon-level test** — Spawn 5 goroutines each calling `d.deps.setTracking`/`setAudio`/`setGesture` in a tight loop. This exercises `d.mu.RLock` → `hidMu.Lock` → `d.mu.Lock` lock ordering. The simulator-level concurrent test doesn't cover this.

6. **Config-failure re-probe test** — Set `sendErr`, call `setTracking`, verify: (a) `hidFailCount` incremented, (b) `probeDevices()` was called, (c) on no-device-found, `hidDev` is nilled, (d) subsequent call returns `ErrPIXYNotConnected` immediately. This makes the "can't accumulate via config failures" claim concrete.

7. **Fuzz `handleConfig` and `handleCommit`** — Feed random bytes to the protocol state handlers and verify no panic. Quick to write, catches edge cases in byte validation.

### Process

8. **Commit explicitly** — Never rely solely on the auto-commit daemon. After all tests pass, commit with a descriptive message. The daemon is a safety net, not the primary commit mechanism.

9. **Consolidate status reports** — Four reports for the same body of work is excessive. After this session, the two prior simulator reports (`03-16`, `03-40`) should be annotated as superseded or consolidated into this one.

10. **Update prior reports before writing new ones** — Before creating a new status report, check if updating an existing one would be more appropriate. The `03-16` report's "NOT STARTED" section is now stale.

---

## f) Next 50 Things to Get Done

### High Priority — Test Depth (1-10)

| #  | Task                                                                                | Impact | Effort |
| -- | ----------------------------------------------------------------------------------- | ------ | ------ |
| 1  | Config-failure re-probe verification test (prove config failures can't accumulate)  | High   | 20 min |
| 2  | Concurrent daemon-level test (5 goroutines × `setTracking`/`setAudio`/`setGesture`) | High   | 20 min |
| 3  | Fuzz test for `handleConfig` (random bytes, verify no panic)                        | Medium | 15 min |
| 4  | Fuzz test for `handleCommit` (random bytes, verify no panic)                        | Medium | 15 min |
| 5  | `parseHIDResponse` direct table test (crafted valid + invalid byte arrays)          | Medium | 20 min |
| 6  | Add `delayResponse` field to simulator                                              | Medium | 15 min |
| 7  | Test `hidResponseTimeout` (500ms) expiration with `delayResponse`                   | Medium | 15 min |
| 8  | Add `failOnNthCall` counter for flaky device simulation                             | Medium | 15 min |
| 9  | Test flaky device recovery: fail once → retry succeeds → count resets               | Medium | 15 min |
| 10 | `BenchmarkSimulatorRoundTrip` (config → commit → query latency)                     | Low    | 10 min |

### Medium Priority — Simulator Enhancements (11-20)

| #  | Task                                                                                  | Impact | Effort |
| -- | ------------------------------------------------------------------------------------- | ------ | ------ |
| 11 | Move `isCommitReport` to `hid.go` (production code)                                   | Medium | 10 min |
| 12 | Add `queryErr` failure injection (SendRecv-specific, distinct from corrupt/nil)       | Low    | 10 min |
| 13 | Track `commitCount` and `configCount` separately for finer assertions                 | Low    | 10 min |
| 14 | Add `Reset()` method to clear simulator state between subtests                        | Low    | 5 min  |
| 15 | Add `SentReportsByType()` helper (returns configs and commits separately)             | Low    | 10 min |
| 16 | Add `QueriesByInterface()` helper (returns queries grouped by interface byte)         | Low    | 10 min |
| 17 | Add `StateSnapshot()` method returning a value copy of current state                  | Low    | 10 min |
| 18 | Add `LastCommitIface()` accessor to verify commit sequencing                          | Low    | 5 min  |
| 19 | Auto-manage lifecycle test using `withPixySimulator()` instead of `withFakeDevices()` | Medium | 20 min |
| 20 | Test `nilResponse`/`corruptResp` path through full daemon `syncState`                 | Low    | 15 min |

### Medium Priority — Architecture & Infrastructure (21-30)

| #  | Task                                                                          | Impact | Effort  |
| -- | ----------------------------------------------------------------------------- | ------ | ------- |
| 21 | Research Go `uhid` syscall feasibility (create `/dev/hidraw*` from userspace) | Medium | 30 min  |
| 22 | Prototype `/dev/uhid` virtual device using `pixyProtocolState` as backend     | Medium | 2 hours |
| 23 | Test `hidrawDevice.Send` against uhid-created device                          | Medium | 1 hour  |
| 24 | Test `hidrawDevice.SendRecv` against uhid-created device                      | Medium | 1 hour  |
| 25 | Create `tests/nixos-vm.nix` skeleton with `makeTest`                          | Medium | 30 min  |
| 26 | NixOS VM test: verify systemd service starts                                  | Medium | 1 hour  |
| 27 | NixOS VM test: verify unix socket creation                                    | Medium | 30 min  |
| 28 | NixOS VM test: verify `sd_notify READY=1`                                     | Low    | 30 min  |
| 29 | Research `vivid` kernel module for V4L2 testing                               | Low    | 30 min  |
| 30 | Prototype `vivid`-based PTZ round-trip test                                   | Low    | 2 hours |

### Lower Priority — Polish & Documentation (31-40)

| #  | Task                                                                                  | Impact | Effort |
| -- | ------------------------------------------------------------------------------------- | ------ | ------ |
| 31 | Give AGENTS.md a 20-line buffer (trim to ~357 lines)                                  | Low    | 15 min |
| 32 | Annotate `2026-08-03_03-16` and `2026-08-03_03-40` reports as superseded              | Low    | 5 min  |
| 33 | Consolidate 4 simulator status reports into one authoritative report                  | Low    | 20 min |
| 34 | Add `// Example` test function showing simulator usage                                | Low    | 10 min |
| 35 | Create `docs/testing.md` with simulator usage guide                                   | Low    | 20 min |
| 36 | Add README section on testing strategy (Layers 1/2/3)                                 | Low    | 15 min |
| 37 | Verify simulator works with `go test -short` skip pattern                             | Low    | 10 min |
| 38 | Add CI step to run simulator tests separately with verbose output                     | Low    | 15 min |
| 39 | Add test coverage report for simulator files specifically                             | Low    | 10 min |
| 40 | Review whether `pixyProtocolState` should be a separate package (`internal/pixysim/`) | Low    | 20 min |

### Lower Priority — Edge Cases & Hardening (41-50)

| #  | Task                                                                              | Impact | Effort |
| -- | --------------------------------------------------------------------------------- | ------ | ------ |
| 41 | Test `handleConfig` with `nil` report (should return error, not panic)            | Low    | 5 min  |
| 42 | Test `handleCommit` with `nil` report (should return error, not panic)            | Low    | 5 min  |
| 43 | Test `buildResponse` with `nil` query (should return error, not panic)            | Low    | 5 min  |
| 44 | Test `Send` with empty `[]byte{}` (should return error from handleConfig)         | Low    | 5 min  |
| 45 | Test config→commit→config→commit sequencing for different interfaces              | Low    | 10 min |
| 46 | Test stale pending config (config for tracking, commit for audio → no match)      | Low    | 10 min |
| 47 | Test `SendRecv` with context already cancelled before call                        | Low    | 5 min  |
| 48 | Test circuit breaker recovery: open → `probeDevices` finds device → reset → works | Low    | 25 min |
| 49 | Test that `broadcastStateChanged` fires on circuit breaker state transitions      | Low    | 15 min |
| 50 | Archive planning doc after all Layer 1 items are closed                           | Low    | 5 min  |

---

## g) Questions

**1. Should I squash the two auto-committed commits (`45693ee` + `c34ac8a`) into one clean commit with a proper message?**

The auto-commit daemon split the work into two commits and gave one of them (`c34ac8a`) a completely blank subject line. The work is saved but the history is messy. I can `git reset --soft` to the parent and recommit as one, but that rewrites history. Alternatively I can leave it and just be more disciplined about explicit commits going forward. Your call — is git history cleanliness worth a rewrite?

**2. Should I consolidate the four simulator status reports (`02-52`, `03-16`, `03-40`, `03-48`) into one authoritative report?**

Four reports for the same body of work is excessive. Each superseded the last as new gaps were closed. I could annotate the prior three with a "SUPERSEDED" header pointing to the latest, or I could merge their unique content into this one and delete the rest. The `update-old-docs` skill exists for exactly this purpose. Which approach do you prefer?

**3. Should `isCommitReport` move to production code (`hid.go`)?**

It encodes protocol knowledge (commit reports have `report[3] == report[1]`) that currently lives only in `pixy_simulator_test.go`. If the commit format changes, the simulator silently breaks. But it's currently only used by the simulator — YAGNI says leave it. The function is pure and could be useful in production debugging. I can't decide this without knowing whether you envision the simulator ever being used outside of `_test.go` files (e.g., a `/dev/uhid` prototype in a `cmd/` binary).
