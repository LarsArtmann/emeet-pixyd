# Status Report: PIXY HID Simulator — Gap Closure Round 2

**Date:** 2026-08-03 03:40
**Session Goal:** Close remaining critical gaps from the prior gap-closure status report
**Prior State:** 39 test functions (57 with subtests), `commitErr` + `sentTimestamps` added, pointer fixed
**This Session:** 48 test functions (65 with subtests), all high-priority gaps closed

---

## Executive Summary

Closed **all 7 high-priority gaps** from the prior report's "Not Started" and "Partially Done" lists. Added 9 new test functions covering the realistic circuit breaker accumulation path (commit failures), context cancellation during the 200ms sleep, concurrent stress testing, `buildResponse` byte-layout verification, `queryHIDState[T]` generic wrapper end-to-end, and syncState write-then-read round-trip. Fixed `sentTimestamps` lock consistency.

**Key architectural insight discovered:** Only the commit failure path (`device.go:55-65`) can naturally accumulate `hidFailCount` to the circuit breaker threshold. Config Send failures trigger `probeDevices()` which either resets the counter (device found) or nils `hidDev` (device not found) — neither can reach threshold naturally. This means the `probeDevices` mock is unnecessary; testing the commit failure path is both simpler and more realistic.

---

## a) FULLY DONE

| Item | Details |
|------|---------|
| Circuit breaker: real accumulation via 3 commit failures | `TestSimulator_CircuitBreaker_RealAccumulationViaCommitFailures` — sets `commitErr`, drives 3 real `setTracking` calls, verifies count increments 1→2→3 and circuit opens on 4th call. hidDev stays intact because commit failures don't trigger re-probe. 6 reports recorded (3 config + 3 commit). |
| Context cancellation during 200ms sleep | `TestSimulator_ContextCancellationDuringSleep` — cancels context 50ms after config Send succeeds (during 200ms sleep). Verifies: error wraps `context.Canceled`, only 1 report (config only, commit never sent), `hidFailCount` stays 0 (cancellation is not a failure). |
| Concurrent stress test | `TestSimulator_ConcurrentAccess` — 10 goroutines × 20 iterations doing Send (config+commit) + SendRecv. Verifies no deadlock/panic, all 400 reports + 200 queries recorded, final state consistent. Uses `sync.WaitGroup.Go` (Go 1.25+). |
| `buildResponse` byte-layout table test | `TestBuildResponse_ByteLayout` — 6 subtests: tracking (idle/tracking/privacy) + audio (nc/live/original). Asserts positions 0-8: prefix, interface, markers, mode byte. |
| Gesture response last byte | `TestBuildResponse_GestureLastByte` — 2 subtests (enabled/disabled). Directly asserts `resp[hidRespBufSize-1]` is `gestureEnabledByte` when enabled, `0x00` when disabled. |
| `queryHIDState[T]` generic wrapper | `TestSimulator_QueryHIDState_GenericWrapper` — full round-trip through the real generic function for all 3 interfaces (tracking, audio, gesture). Type inference exercised. |
| `queryHIDState` error paths | `TestSimulator_QueryHIDState_NilResponse` (wraps `errNoHIDResponse`) + `TestSimulator_QueryHIDState_CorruptResponse` (wraps `errUnrecognizedHID`). |
| syncState write-then-read round-trip | `TestSimulator_SyncState_WriteThenReadRoundTrip` — sets Privacy via daemon → changes simulator state directly (simulated button press) → sync detects drift → daemon state updated. Tests both write and read paths in sequence. |
| `sentTimestamps` lock consistency | `time.Now()` moved inside `s.mu.Lock()` for consistency with `sentReports` recording. |
| AGENTS.md updated | Test count 39→48, new test descriptions, circuit breaker insight added. Still 377 lines (edits were within-line). |
| Lint | 0 issues (`golangci-lint run --timeout 2m ./...`) |
| Vet | Clean |
| Race detector | All tests pass with `-race -count=1` |

---

## b) What Was Discovered

### Circuit Breaker Accumulation Path Analysis

After deep analysis of `setDeviceState` in `device.go`, I discovered:

1. **Config Send failure path** (lines 33-47): Increments `hidFailCount`, then calls `applyProbeResultLocked(probeDevices())` if count < threshold. If re-probe finds the device → `hidFailCount` resets to 0. If re-probe doesn't find the device → `d.hidDev = nil`, and subsequent calls return `ErrPIXYNotConnected` immediately without incrementing. **This path can NEVER naturally reach threshold 3.**

2. **Commit Send failure path** (lines 55-65): Increments `hidFailCount`, does NOT call `probeDevices()`. `hidDev` stays intact. **This path CAN naturally reach threshold 3** — it's the realistic circuit breaker trigger for flaky commit writes (device accepts config but fails on commit).

**Implication:** The prior report's concern about "mocking `probeDevices` in circuit breaker tests" is moot. The pre-loaded approach (for config Send failures) is correct because that path can't accumulate naturally. The commit failure path IS the realistic accumulation path, and it doesn't need `probeDevices` mocking.

---

## c) What Remains (Lower Priority)

| Item | Impact | Effort |
|------|--------|--------|
| Fuzz test for `handleConfig`/`handleCommit` | Medium | 30 min |
| Benchmark for simulator round-trip | Low | 10 min |
| `delayResponse` field for timeout testing | Low | 15 min |
| `/dev/uhid` virtual device (Layer 2) | Medium | 2+ hours |
| NixOS VM test (Layer 3) skeleton | Medium | 1+ hours |
| `vivid` V4L2 test driver | Low | 2+ hours |
| AGENTS.md 20-line buffer (trim to ~357) | Low | 15 min |

---

## d) Test Inventory (48 functions, 65 with subtests)

### Protocol State Tests (11)
- `TestSimulator_HandleConfig_Valid{Tracking,Audio,Gesture}`
- `TestSimulator_HandleConfig_{InvalidPrefix,UnknownInterface,InvalidModeByte,TooShort}`
- `TestSimulator_Commit_{AppliesPendingConfig,NoPendingConfig,InterfaceMismatch}`
- `TestSimulator_QueryBeforeCommit_ReturnsOldState`

### Query Round-Trip Tests (4)
- `TestSimulator_Query_{TrackingRoundTrip,AudioRoundTrip,GestureRoundTrip}`
- `TestSimulator_Query_ResponseParseableByDaemonParser`

### Send/SendRecv Tests (5)
- `TestSimulator_Send_{ConfigThenCommit,RecordsAllReports}`
- `TestSimulator_SendRecv_RoundTrip`
- `TestSimulator_SendErr`
- `TestSimulator_{SendRecvErr,NilResponse,CorruptResponse}`

### Heuristic & Mapping Tests (3)
- `TestIsCommitReport` (7 subtests)
- `TestCameraStateFromHIDByte` (4 cases)
- `TestAudioModeFromHIDByte` (4 cases)

### Daemon Integration Tests (5)
- `TestSimulator_DaemonSet{Tracking,Audio,Gesture}_RoundTrip`
- `TestSimulator_SyncState_ReadsFromSimulator`
- `TestSimulator_DaemonSetTracking_ProtocolBytesValid`

### Byte Layout Tests (3)
- `TestPixyConfig_ByteLayout` (8 subtests)
- `TestPixyCommit_ByteLayout` (3 subtests)
- `TestBuildResponse_ByteLayout` (6 subtests) **NEW**
- `TestBuildResponse_GestureLastByte` (2 subtests) **NEW**

### Circuit Breaker Tests (4)
- `TestSimulator_CircuitBreaker_Accumulation` (pre-loaded)
- `TestSimulator_CircuitBreaker_ResetOnSuccess`
- `TestSimulator_CircuitBreaker_CommitFailure`
- `TestSimulator_CircuitBreaker_RealAccumulationViaCommitFailures` **NEW**

### Timing & Cancellation Tests (2)
- `TestSimulator_200msSleepBetweenConfigAndCommit`
- `TestSimulator_ContextCancellationDuringSleep` **NEW**

### Generic Wrapper Tests (3) **ALL NEW**
- `TestSimulator_QueryHIDState_GenericWrapper`
- `TestSimulator_QueryHIDState_NilResponse`
- `TestSimulator_QueryHIDState_CorruptResponse`

### Concurrency & Round-Trip Tests (3) **ALL NEW**
- `TestSimulator_ConcurrentAccess`
- `TestSimulator_SyncState_WriteThenReadRoundTrip`
- `TestSimulator_SendRecvResponse_ProducesCorrectBytes`

### Audio/Gesture Protocol Bytes (2)
- `TestSimulator_DaemonSetAudio_ProtocolBytesValid`
- `TestSimulator_DaemonSetGesture_ProtocolBytesValid`
