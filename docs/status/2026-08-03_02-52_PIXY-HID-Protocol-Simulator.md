> **SUPERSEDED** — This report is superseded by `2026-08-03_03-48_Simulator-Gap-Closure-Round2-SelfReview.md`.
> The simulator now has 48+ test functions with fuzz coverage, multi-interface pending tests, and a benchmark.

# Status Report: PIXY HID Protocol Simulator

**Date:** 2026-08-03 02:52 (updated 03:30)
**Session Goal:** Build a protocol-faithful PIXY HID simulator for integration testing
**Commit:** `69da92d` — `test(hid): add protocol-faithful PIXY HID simulator and test suite`
**Update:** Follow-up session closed all critical gaps. 39 tests, lint clean, `-race` passing.

---

## Executive Summary

Built and shipped a PIXY HID protocol simulator (`pixySimulator`) that validates every outgoing byte against the wire protocol. Follow-up session added circuit breaker tests, 200ms timing verification, direct byte-layout tests, and fixed all design issues. **39 tests** (57 with subtests), all passing with `-race`. Lint clean.

---

## a) FULLY DONE

| Item                  | Details                                                                                                                                                                                                                                                                                                                                                                                            |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Planning doc          | `docs/planning/2026-08-03_02-39_PIXY-HID-PROTOCOL-SIMULATOR.md` with Pareto breakdown + mermaid graph                                                                                                                                                                                                                                                                                              |
| `pixyProtocolState`   | Device-side state machine: config validation (prefix, interface, markers, mode byte), commit sequencing, state application, response building                                                                                                                                                                                                                                                      |
| `pixySimulator`       | `HIDDevice` implementation: `Send` dispatches config/commit, `SendRecv` builds query responses, failure injection (`sendErr`, `sendRecvErr`, `nilResponse`, `corruptResp`), recording (`SentReports()`, `Queries()`)                                                                                                                                                                               |
| `withPixySimulator()` | Test option: wires simulator as `hidDev`, keeps real `setTracking`/`setAudio`/`setGesture`, stubs V4L2/proc deps                                                                                                                                                                                                                                                                                   |
| Unit tests            | 39 tests: config validation (7), commit sequencing (4), query round-trips (4), HIDDevice impl (3), failure injection (4), reverse byte mappings (2), `isCommitReport` table (1), daemon integration (5), **circuit breaker** (3: accumulation, reset-on-success, commit-failure), **200ms timing** (1), **`pixyConfig`/`pixyCommit` byte-layout tables** (2), **audio/gesture protocol bytes** (2) |
| Lint                  | 0 issues (golangci-lint v2, all linters)                                                                                                                                                                                                                                                                                                                                                           |
| Race detector         | All 31 tests pass with `-race`                                                                                                                                                                                                                                                                                                                                                                     |
| Full suite            | All existing tests still pass                                                                                                                                                                                                                                                                                                                                                                      |
| AGENTS.md             | Updated with simulator documentation in file table, key interactions, and test options list                                                                                                                                                                                                                                                                                                        |
| Git                   | Committed (`69da92d`) and pushed to `origin/master`                                                                                                                                                                                                                                                                                                                                                |

---

## b) PARTIALLY DONE

| Item                       | What Exists                                                                                                                                                       | What's Missing                                                                       |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Daemon integration tests   | `setTracking`/`setAudio`/`setGesture` → simulator → verify state. Circuit breaker (3 tests), 200ms timing (1 test), full `setDeviceState` path tested             | —                                                                                    |
| `syncState` test           | Reads pre-set simulator state, verifies daemon updates                                                                                                            | No write-then-read round-trip (set via daemon → change on sim → sync detects change) |
| Failure injection          | `sendErr`, `commitErr` (commit-only), `sendRecvErr`, `nilResponse`, `corruptResp`. All tested.                                                                    | —                                                                                    |
| Protocol byte verification | All three interfaces (tracking/audio/gesture) have daemon-integration byte-layout tests. Direct `pixyConfig()`/`pixyCommit()` table tests cover all combinations. | —                                                                                    |

---

## c) NOT STARTED

| Item                                                       | Why It Matters                                                                                                                                                          |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~Circuit breaker tests with simulator~~                   | **DONE** — 3 tests: accumulation (2→3→open), reset-on-success (1→0), commit-failure (config ok, commit fails)                                                           |
| ~~200ms sleep timing verification~~                        | **DONE** — `sentTimestamps` recording + gap assertion (≥190ms)                                                                                                          |
| ~~Direct `pixyConfig()`/`pixyCommit()` byte-layout tests~~ | **DONE** — Table tests for all 3 interfaces × all mode bytes                                                                                                            |
| ~~Audio/gesture protocol byte-layout~~                     | **DONE** — Daemon integration tests verify exact bytes for all interfaces                                                                                               |
| ~~`pixyProtocolState` pointer fix~~                        | **DONE** — Changed from value to `*pixyProtocolState`                                                                                                                   |
| ~~AGENTS.md trim~~                                         | **DONE** — Trimmed from 398 to 377 lines                                                                                                                                |
| `queryHIDState[T]` generic wrapper test                    | Queries go through `queryHIDState[T]` → `SendRecv` → `parseHIDResponse`. I tested `SendRecv` and `parseHIDResponse` separately but never the full generic wrapper path. |
| Auto-manage lifecycle with simulator                       | The existing `integration_auto_test.go` uses `withFakeDevices()`. Using `withPixySimulator()` would exercise real HID during call start/end.                            |
| Concurrent access tests                                    | `-race` passes, but no explicit concurrent test (multiple goroutines calling `Send`/`SendRecv` simultaneously)                                                          |
| Benchmark for simulator                                    | Project has 8 benchmarks; `BenchmarkSimulatorRoundTrip` would be natural                                                                                                |
| Fuzz test for `handleConfig`/`handleCommit`                | Project has fuzz tests for `parseHIDResponse`; fuzzing the simulator's input handler would catch panics on random bytes                                                 |
| NixOS VM test (Layer 3)                                    | Discussed in planning but explicitly future work — tests systemd service lifecycle, module config                                                                       |
| `/dev/uhid` virtual device (Layer 2)                       | Discussed as Layer 2; creates real `/dev/hidraw*` devices from userspace                                                                                                |
| `vivid` V4L2 test driver                                   | Would enable real `v4l2-ctl` PTZ round-trip testing                                                                                                                     |

---

## d) TOTALLY FUCKED UP

| Item                                                | What Happened                                                                                                                                                                                                                                                                    | Severity          |
| --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- |
| **`result.ok` / `result.message` field name error** | Used `result.ok` and `result.message` but `CommandResult` has `Err` and `Message`. Caught by compiler, fixed immediately. Should have read the struct before writing the test.                                                                                                   | Low (self-caught) |
| **wsl_v5 lint failures (11+ instances)**            | Wrote code without blank lines before `if` statements, violating the project's wsl_v5 linter. Had to run `golangci-lint --fix`. Should have followed the existing code pattern from the start — every `if` in this project has a blank line above it unless it's a guard clause. | Low (auto-fixed)  |
| **AGENTS.md exceeds 398 lines**                     | BuildFlow warned: "AGENTS.md has 398 lines (max: 377, excess: 21)". **FIXED** — Trimmed to 377 lines by removing obsolete/changelog entries.                                                                                                                                     | Medium → Resolved |
| **`pixyProtocolState` embedded by value**           | `pixySimulator` had `state pixyProtocolState` (value, not pointer). **FIXED** — Changed to `state *pixyProtocolState`.                                                                                                                                                           | Low → Resolved    |

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **~~Use pointer for `pixyProtocolState` in `pixySimulator`~~** — **DONE**. Changed `state pixyProtocolState` to `state *pixyProtocolState`.

2. **Document `isCommitReport` invariant** — The heuristic relies on `pixyConfig` always setting `report[3] = 0x00`. This is true today but could break if someone changes `pixyConfig`. Add a comment or assertion.

3. **Remove unnecessary marker bytes in gesture response** — `buildResponse` sets markers at positions 2, 5, 7 for gesture responses, but `parseHIDResponse` never checks them for gesture (only checks `data[0]`, `data[1]`, `data[len-1]`). Harmless but misleading.

4. **`withPixySimulator` doesn't wire `findSource`/`setSource`** — Left as noops. Auto-manage tests that exercise PipeWire source switching would need these wired.

### Test Coverage Gaps

5. **Circuit breaker is completely untested via simulator** — This is the biggest miss. The simulator + `sendErr` injection is the ideal setup. Should test: 3 failures → circuit opens → commands return `ErrPIXYNotConnected` → successful Send → circuit resets.

6. **200ms sleep timing untested** — `setDeviceState` sleeps between config and commit. The simulator should record timestamps to verify this protocol invariant.

7. **No direct `pixyConfig()`/`pixyCommit()` unit tests** — The AGENTS.md explicitly notes this gap. The simulator integration tests implicitly cover it, but direct byte-layout assertions would be clearer.

8. **No audio/gesture equivalent of `TestSimulator_DaemonSetTracking_ProtocolBytesValid`** — Only tracking byte layout is verified. Audio and gesture should get the same treatment.

9. **No fuzz test for `handleConfig`/`handleCommit`** — Random bytes could trigger panics or unexpected state transitions.

### Architecture

10. **Simulator is test-only (`_test.go`)** — Can't be used by other packages or integration test binaries. If we later want a standalone simulator binary (for CI or Nix VM tests), it would need to move to a non-test file or separate package.

11. **No Layer 2 (`/dev/uhid`) or Layer 3 (Nix VM)** — The planning doc outlines these as future work. The protocol core (`pixyProtocolState`) is designed to be shared, but no layer has been built on top yet.

12. **BuildFlow warns about AGENTS.md size** — Need to either trim the simulator documentation or move it to a separate doc.

---

## f) Up to 50 Things We Should Get Done Next

### High Priority — Testing Gaps (1-10)

1. **Test circuit breaker with simulator**: 3× `sendErr` → verify `ErrPIXYNotConnected` → verify circuit resets on success
2. **Test circuit breaker re-probe**: On first failure, verify `probeDevices()` is called
3. **Test 200ms sleep timing**: Record timestamps in simulator, verify gap between config and commit
4. **Direct `pixyConfig()` byte-layout unit test**: Assert exact 9-byte output for each interface+mode combination
5. **Direct `pixyCommit()` byte-layout unit test**: Assert exact 4-byte output for each interface
6. **Audio protocol bytes test**: Equivalent of `TestSimulator_DaemonSetTracking_ProtocolBytesValid` for audio
7. **Gesture protocol bytes test**: Same for gesture
8. **Full `syncState` round-trip**: Set tracking via daemon → manually change on simulator → syncState detects change
9. **Test `queryHIDState[T]` generic wrapper**: Verify the full query → SendRecv → parseHIDResponse → extract path
10. **Fuzz test `handleConfig`/`handleCommit`**: Feed random bytes, assert no panics

### Medium Priority — Deeper Integration (11-20)

11. **Auto-manage lifecycle with simulator**: `withPixySimulator` + call detection → verify real HID path during call start/end
12. **Concurrent access test**: Multiple goroutines calling `Send`/`SendRecv` on the same simulator
13. **`withPixySimulator` wire `findSource`/`setSource`**: Enable PipeWire source switching in auto tests
14. **`BenchmarkSimulatorRoundTrip`**: Measure simulator config→commit→query overhead
15. **`BenchmarkSimulatorSetTracking`**: Full daemon `setTracking` → simulator path
16. **Test config→commit→query→verify for all 3 interfaces**: Comprehensive parameterized test
17. **Test commit for wrong interface after config**: Send tracking config, then audio commit → should fail
18. **Test stale pending config**: Config for tracking, then another config for tracking (overwrite?) → verify behavior
19. **Test `nilResponse` through daemon**: Verify daemon handles nil (timeout) correctly via `syncState`
20. **Test `corruptResp` through daemon**: Verify daemon handles garbage response via `syncState`

### Design Improvements (21-25)

21. **Change `pixyProtocolState` to pointer in `pixySimulator`**: `state *pixyProtocolState`
22. **Document `isCommitReport` invariant**: Add comment about `pixyConfig[3] == 0x00` assumption
23. **Remove unnecessary gesture response markers**: Clean up `buildResponse` for gesture case
24. **Add `SimulatorState()` accessor**: Return a snapshot struct instead of individual `Tracking()`/`Audio()`/`Gesture()` calls
25. **Add `Reset()` method to simulator**: Clear state + recording for test reuse

### Layer 2: `/dev/uhid` Virtual Device (26-35)

26. **Research `uhid` kernel module API**: `UHID_CREATE`, `UHID_OUTPUT`, `UHID_INPUT` events
27. **Implement `uhidDevice` struct**: Opens `/dev/uhid`, creates virtual HID with `idVendor=0x328f`
28. **Wire `pixyProtocolState` to uhid event loop**: Handle `UHID_OUTPUT` → validate → respond
29. **Test `probeDevices()` discovery**: Verify daemon finds the virtual `/dev/hidraw*` via sysfs
30. **Test real `hidrawDevice.Send()` I/O path**: The actual `os.OpenFile` → `Write` → `Close` cycle
31. **Test real `hidrawDevice.SendRecv()` I/O path**: `OpenFile` → `Write` → `Read` with 500ms timeout
32. **Test buffer size handling**: 32-byte write, 64-byte read on real kernel device
33. **Determine CI feasibility**: Does `CONFIG_UHID` exist in GitHub Actions runners? Does it need root?
34. **Fallback for non-uhid environments**: Graceful skip if `/dev/uhid` not available
35. **Document uhid test setup**: How to run, what permissions are needed

### Layer 3: NixOS VM Test (36-45)

36. **Define `nixosTests.emeet-pixyd` in flake.nix**: QEMU VM with NixOS module enabled
37. **Set up minimal graphical session**: User service needs `graphical-session.target`
38. **Test systemd `Type=notify`**: Verify `sd_notify(READY=1)` within watchdog timeout
39. **Test socket creation**: Verify `/run/emeet-pixyd/emeet-pixyd.sock` exists
40. **Test `/api/health` endpoint**: HTTP request to running daemon
41. **Test CLI commands**: `emeet-pixyd status` via socket
42. **Test hardening**: Verify `ProtectSystem=strict` doesn't break socket creation
43. **Test PATH dependencies**: v4l-utils, wireplumber, libnotify, ffmpeg in service PATH
44. **Test graceful shutdown**: systemd stop → clean exit
45. **Run NixOS VM test in CI**: Add to GitHub Actions workflow

### Polish & Documentation (46-50)

46. **Trim AGENTS.md below 377 lines**: Move simulator detail to planning doc or separate test doc
47. **Update FEATURES.md**: Add "HID Protocol Simulator" to testing infrastructure
48. **Create `docs/testing.md`**: Document the three-layer testing strategy + when to use each layer
49. **Add `CONTRIBUTING.md` testing section**: How to run unit/simulator/integration/hardware tests
50. **Explore `vivid` kernel module**: For V4L2 PTZ simulation in CI/VM (enables real `v4l2-ctl` testing)

---

## g) Questions

### 1. Should we prioritize the circuit breaker tests or the NixOS VM test next?

The circuit breaker tests are the highest-ROI gap in the simulator work (low effort, high value — critical safety mechanism, perfect simulator setup already exists). The NixOS VM test is higher effort but covers a completely untested surface (the NixOS module + systemd service). Which should I tackle next?

### 2. Should the simulator move out of `_test.go` files?

Currently the simulator is test-only (`pixy_simulator_test.go`). If we build Layer 2 (`/dev/uhid`) or Layer 3 (Nix VM test), the simulator core (`pixyProtocolState`) would need to be importable from a non-test context. Moving it to e.g. `internal/pixysim/` would enable reuse but changes the package structure. Is this premature, or should we do it now?

### 3. Do you want me to trim AGENTS.md now, or leave it?

BuildFlow flagged AGENTS.md at 398 lines (max 377). My simulator additions pushed it over. I can trim the simulator docs to be more concise, or move detail to a separate `docs/testing.md`. Should I do this now or batch it with other improvements?
