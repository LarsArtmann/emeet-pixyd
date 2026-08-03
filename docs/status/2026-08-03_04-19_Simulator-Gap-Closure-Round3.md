# Status Report: PIXY HID Simulator — Round 3 Gap Closure (Brutal Self-Review)

**Date:** 2026-08-03 04:19
**Session Goal:** Continue gap closure on the PIXY HID protocol simulator — "Break this down into multiple actionable steps. Think about them again. Execute and Verify them one step at a time. Repeat until done."
**Prior State:** 48 test functions (74 with subtests), all high-priority gaps from Round 2 closed
**This Session:** 50 test functions + 1 fuzz target + 1 benchmark (9th). 2 clean commits.

---

## Executive Summary

This session added **2 new test functions** (`TestSimulator_MultiInterfacePending`, `TestSimulator_OverwritePendingConfig`), **1 new fuzz target** (`FuzzHandleConfigAndCommit`), **1 new benchmark** (`BenchmarkSimulatorRoundTrip`), annotated 3 prior status reports as SUPERSEDED, and updated AGENTS.md. All tests pass with `-race`. Lint 0 issues. Vet clean.

**Key correction:** The prior self-review (Round 2) claimed `parseHIDResponse` lacked direct table tests — this was **wrong**. `hid_test.go` already has 8 direct tests for it. The session caught this before duplicating coverage.

**Key discovery about the self-review process itself:** The Round 2 self-review's "50 next steps" list was largely low-value padding. The real gaps were: (1) the protocol state machine's multi-interface pending map was completely untested, (2) no fuzz coverage existed for `handleConfig`/`handleCommit`, and (3) no performance baseline existed for the simulator. Those 3 items are now closed. The remaining 47 items from the prior list are mostly YAGNI or "nice to have" documentation polish.

---

## a) FULLY DONE

| Item                                        | Details                                                                                                                                                                                                                                                                                                                                                                                                            |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Fuzz test for `handleConfig`/`handleCommit` | `FuzzHandleConfigAndCommit` in `pixy_simulator_fuzz_test.go` (80 lines). Feeds random byte slices through the protocol state machine via the same `isCommitReport` routing the simulator uses. Verifies no panic + pending map invariants (entry exists after successful config, deleted after successful commit). Seeds: 12 (nil, empty, valid configs, valid commits, garbage). Ran 256K+ execs with 0 findings. |
| Multi-interface pending test                | `TestSimulator_MultiInterfacePending` in `pixy_simulator_impl_test.go`. Queues configs for all 3 interfaces (tracking, audio, gesture) without committing. Verifies: state unchanged until commits, committing one interface doesn't leak into others, all pending entries cleared after all commits. Guards against per-interface isolation regressions.                                                          |
| Overwrite pending config test               | `TestSimulator_OverwritePendingConfig`. Queues tracking-mode config, then privacy-mode config on the same interface without an intervening commit. Verifies commit applies the latest pending value (privacy), not the first (tracking). Locks in last-write-wins semantics.                                                                                                                                       |
| Simulator round-trip benchmark              | `BenchmarkSimulatorRoundTrip` in `benchmark_test.go`. Full config→commit→query round-trip for all 3 interfaces. Baseline: **1.49µs/op** (AMD Ryzen AI MAX+ 395). Establishes performance baseline for comparing future Layer 2 (/dev/uhid) and Layer 3 (NixOS VM) test harnesses.                                                                                                                                  |
| Superseded status reports annotated         | 3 prior reports (`02-52`, `03-16`, `03-40`) now have `> **SUPERSEDED**` blockquote headers pointing to the final self-review. Non-destructive (content preserved, header prepended).                                                                                                                                                                                                                               |
| AGENTS.md updated                           | CI fuzz target list: 4→5 (`FuzzHandleConfigAndCommit` added). Test count: 48→50. Fuzz test file list: +`pixy_simulator_fuzz_test.go`. Benchmarks: 8→9 (`BenchmarkSimulatorRoundTrip` added). Still 377 lines (edits were within-line).                                                                                                                                                                             |
| Lint                                        | 0 issues (`golangci-lint run --timeout 2m ./...`)                                                                                                                                                                                                                                                                                                                                                                  |
| Vet                                         | Clean                                                                                                                                                                                                                                                                                                                                                                                                              |
| Race detector                               | All tests pass with `-race -count=1` (1.74s)                                                                                                                                                                                                                                                                                                                                                                       |
| Commits                                     | 2 clean commits: `3082afb` (tests + fuzz + benchmark), `83d7b1e` (docs + AGENTS.md)                                                                                                                                                                                                                                                                                                                                |

---

## b) PARTIALLY DONE

| Item                               | What Exists                                                                                   | What's Missing                                                                                                                                                                                                                 |
| ---------------------------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Fuzz coverage of protocol handlers | `FuzzHandleConfigAndCommit` fuzzes `handleConfig`/`handleCommit` via `isCommitReport` routing | Doesn't fuzz `buildResponse` (query path). Could add a second fuzz target or extend the existing one. Low value — `buildResponse` is simple switch logic already covered by 6 byte-layout subtests.                            |
| Multi-interface pending coverage   | Tests queue all 3 interfaces and commit them sequentially                                     | Doesn't test interleaving (config tracking → config audio → commit tracking → config gesture → commit audio → commit gesture). Low value — the pending map is a simple `map[byte]*pendingConfig`, interleaving can't break it. |
| Benchmark coverage                 | `BenchmarkSimulatorRoundTrip` covers all 3 interfaces end-to-end                              | Doesn't benchmark the failure injection paths (sendErr, commitErr) or the `Send`-only path without query. Low value — failure paths are dominated by the 200ms sleep, not simulator overhead.                                  |

---

## c) NOT STARTED

| Item                                          | Why It Matters / Why Not Started                                                                                                                                                                                                                                                                                                                       |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **`delayResponse` field for timeout testing** | Would enable testing `hidResponseTimeout` (500ms) expiration. Skipped because the timeout lives in `hidrawDevice.SendRecv` (production file I/O), not in the simulator. Adding a delay to the simulator tests the simulator's own delay, not the production timeout path. To test the real timeout, you'd need a `/dev/uhid` virtual device (Layer 2). |
| **Config-failure re-probe verification test** | Would prove config Send failures can't accumulate to circuit breaker threshold. Skipped because the claim is provable by reading 4 lines of code (`device.go:40-42`): the re-probe either resets the counter or nils `hidDev`. A test would be fragile (calls real `probeDevices()` which scans `/sys/class/video4linux`).                             |
| **Concurrent daemon-level test**              | Would exercise `d.mu` + `hidMu` lock interaction with 5 goroutines. Skipped because `hidMu` serializes all HID commands — the test would just verify a mutex works. The simulator-level concurrent test (`TestSimulator_ConcurrentAccess`, 10 goroutines × 20 iterations) already proves the simulator's own thread safety.                            |
| **`/dev/uhid` virtual device (Layer 2)**      | Creates real `/dev/hidraw*` from userspace. Would test actual file I/O paths (`hidrawDevice.Send`/`SendRecv`). Not started — requires root or `uhid` group, research into Go uhid bindings, and significant infrastructure.                                                                                                                            |
| **NixOS VM test (Layer 3)**                   | QEMU VM testing systemd service lifecycle. Not started — requires `makeTest` infrastructure, significant Nix knowledge, and the value is in integration testing not unit testing.                                                                                                                                                                      |
| **`isCommitReport` move to production**       | Currently test-only, encodes protocol knowledge (`report[3] == report[1]`). Skipped — YAGNI. It's only used by the simulator. Moving it to `hid.go` would add dead code to production.                                                                                                                                                                 |

---

## d) TOTALLY FUCKED UP

| Item                                                   | What Happened                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 | Severity                                                            |
| ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| **Believed the prior self-review without verifying**   | The Round 2 self-review listed "`parseHIDResponse` direct table test" as NOT STARTED. I initially added it to my todo list. Then I read `hid_test.go` and found **8 existing direct tests** (`TestParseHIDResponseTracking`, `TestParseHIDResponseAudio`, `TestParseHIDResponseGesture`, `TestParseHIDResponseTooShort`, `TestParseHIDResponseNil`, `TestParseHIDResponseUnknownInterface`, `TestParseHIDResponseUnknownAudioByte`, `TestParseHIDResponseUnknownTrackingByte`). I wasted a planning cycle on a non-existent gap. The lesson: **verify claims from prior self-reviews before acting on them.** | **Medium** — wasted time, but caught before writing duplicate tests |
| **Fuzz test has a concurrency issue in the test body** | The fuzz test body calls `t.Parallel()` inside `f.Fuzz()`. This is technically valid but unusual — most fuzz tests don't call `t.Parallel()` in the fuzz function. It doesn't cause failures but it's an odd pattern that a reviewer might question. The fuzz corpus runs sequentially per worker anyway.                                                                                                                                                                                                                                                                                                     | **Low** — cosmetic                                                  |
| **Didn't test `buildResponse` fuzz path**              | The fuzz test only covers `handleConfig`/`handleCommit`. `buildResponse` (the query response builder) is not fuzzed. It IS covered by 6 byte-layout subtests + the existing `FuzzParseHIDResponse` (which fuzzes the parser that consumes `buildResponse` output). But direct fuzzing of `buildResponse` with random query inputs would be marginally more thorough.                                                                                                                                                                                                                                          | **Low** — coverage gap, but indirectly covered                      |
| **Benchmark doesn't measure failure paths**            | `BenchmarkSimulatorRoundTrip` only measures the happy path. The failure injection paths (sendErr, commitErr, nilResponse, corruptResp) have no benchmark. This is fine — they're dominated by the 200ms sleep, not simulator overhead — but it means the benchmark doesn't characterize the full simulator surface.                                                                                                                                                                                                                                                                                           | **Low** — scope limitation, not a bug                               |

---

## e) WHAT WE SHOULD IMPROVE

### Process

1. **Verify before planning** — I trusted the Round 2 self-review's gap list and initially planned to write `parseHIDResponse` direct tests that already existed. I should have verified ALL claims from the prior self-review before creating my todo list. Instead, I verified during execution and caught it, but I wasted a planning cycle. **Rule: a prior self-review's claims are hypotheses, not facts. Verify each one.**

2. **The "50 next steps" pattern is low-value** — The Round 2 self-review listed 50 next steps. Of those, only 3 were genuine gaps (multi-interface pending, fuzz coverage, benchmark). The other 47 were YAGNI, nice-to-have polish, or restatements of items already in other sections. The "50 things" format encourages padding. A shorter list of genuinely valuable next steps would be more actionable.

3. **Should have extended the existing fuzz test instead of creating a new file** — `FuzzParseHIDResponse` lives in `hid_fuzz_test.go`. `FuzzHandleConfigAndCommit` could have gone there too. Instead, I created `pixy_simulator_fuzz_test.go`. Both files fuzz the HID protocol, just from different angles (parser vs. state machine). Having them in separate files is defensible (one is production parser, the other is test-only state machine), but it fragments the fuzz coverage landscape slightly.

### Code Quality

4. **`isCommitReport` is a heuristic, not a protocol rule** — The function distinguishes commit from config by checking `report[3] == report[1]`. This works because commit reports repeat the interface byte at position 3. But it's an inference, not an explicit protocol marker. If the config report format ever changed to set position 3 to the interface byte, `isCommitReport` would silently misclassify. This is a latent correctness risk in the simulator. A more explicit approach would be a report type field or a length check (commits are 4 bytes, configs are 9 bytes).

5. **Fuzz test assertions are minimal** — The fuzz test verifies no-panic + pending map invariants. It doesn't verify that `handleConfig` + `handleCommit` with valid inputs produces the correct state transition. Adding state assertions for valid inputs would make the fuzz test more valuable — but it would also make it slower (more work per iteration). The current trade-off (fast fuzzing for panics, dedicated tests for correctness) is reasonable.

### Testing

6. **No test for `Send` with `sendErr` AND `commitErr` set simultaneously** — The simulator has independent `sendErr` and `commitErr` fields. When both are set, `commitErr` takes priority for commit reports and `sendErr` for config reports. This interaction is implicit (the `if s.commitErr != nil && isCommitReport(report)` check comes first) but not explicitly tested. A test setting both and verifying the priority would document the behavior.

7. **No test for `SendRecv` with `sendRecvErr` AND `corruptResp` set simultaneously** — Similar to above. `sendRecvErr` takes priority (checked first). Not tested explicitly.

---

## f) Next Things to Get Done

### High Priority — Genuine Test Gaps (1-5)

| #   | Task                                                                                                                                | Impact | Effort |
| --- | ----------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Test `sendErr` + `commitErr` simultaneous priority (verify commitErr wins for commits, sendErr for configs)                         | Medium | 10 min |
| 2   | Test `sendRecvErr` + `corruptResp` simultaneous priority (verify sendRecvErr wins)                                                  | Low    | 10 min |
| 3   | Extend fuzz to cover `buildResponse` (random query bytes, verify no panic)                                                          | Medium | 15 min |
| 4   | Test `handleConfig` with nil report (should error, not panic) — already covered by fuzz but an explicit test documents the contract | Low    | 5 min  |
| 5   | Test `handleCommit` with nil report (same)                                                                                          | Low    | 5 min  |

### Medium Priority — Simulator Robustness (6-10)

| #   | Task                                                                              | Impact | Effort |
| --- | --------------------------------------------------------------------------------- | ------ | ------ |
| 6   | Add `Reset()` method to `pixySimulator` for clearing state between subtests       | Low    | 5 min  |
| 7   | Add `SentReportsByType()` helper (returns configs and commits as separate slices) | Low    | 10 min |
| 8   | Add `StateSnapshot()` method returning a value copy of `pixyProtocolState`        | Low    | 10 min |
| 9   | Track `commitCount` and `configCount` separately for finer assertions             | Low    | 10 min |
| 10  | Add `LastSentReport()` convenience accessor                                       | Low    | 5 min  |

### Medium Priority — Layer 2 Infrastructure (11-15)

| #   | Task                                                                          | Impact | Effort  |
| --- | ----------------------------------------------------------------------------- | ------ | ------- |
| 11  | Research Go `uhid` syscall feasibility (create `/dev/hidraw*` from userspace) | High   | 30 min  |
| 12  | Prototype `/dev/uhid` virtual device using `pixyProtocolState` as backend     | High   | 2 hours |
| 13  | Test `hidrawDevice.Send` against uhid-created device                          | High   | 1 hour  |
| 14  | Test `hidrawDevice.SendRecv` against uhid-created device (including timeout)  | High   | 1 hour  |
| 15  | Test circuit breaker against real file I/O (not simulator)                    | Medium | 1 hour  |

### Medium Priority — Layer 3 Infrastructure (16-18)

| #   | Task                                                           | Impact | Effort |
| --- | -------------------------------------------------------------- | ------ | ------ |
| 16  | Create `tests/nixos-vm.nix` skeleton with `makeTest`           | Medium | 30 min |
| 17  | NixOS VM test: verify systemd service starts + socket creation | Medium | 1 hour |
| 18  | NixOS VM test: verify `sd_notify READY=1`                      | Low    | 30 min |

### Lower Priority — Documentation & Polish (19-25)

| #   | Task                                                                                           | Impact | Effort |
| --- | ---------------------------------------------------------------------------------------------- | ------ | ------ |
| 19  | Consolidate 4 simulator status reports into 1 authoritative (keep self-review, archive others) | Low    | 20 min |
| 20  | Add `// Example` test function showing simulator usage                                         | Low    | 10 min |
| 21  | Create `docs/testing.md` with simulator usage guide + Layers 1/2/3 explanation                 | Low    | 20 min |
| 22  | Give AGENTS.md a 20-line buffer (trim to ~357 lines)                                           | Low    | 15 min |
| 23  | Research `vivid` kernel module for V4L2 testing                                                | Low    | 30 min |
| 24  | Auto-manage lifecycle test using `withPixySimulator()` instead of `withFakeDevices()`          | Medium | 20 min |
| 25  | Test `nilResponse`/`corruptResp` path through full daemon `syncState`                          | Low    | 15 min |

### Lower Priority — Edge Cases (26-30)

| #   | Task                                                                                                 | Impact | Effort |
| --- | ---------------------------------------------------------------------------------------------------- | ------ | ------ |
| 26  | Test config→commit→config→commit sequencing for different interfaces (interleaved)                   | Low    | 10 min |
| 27  | Test `Send` with empty `[]byte{}` (should error from handleConfig)                                   | Low    | 5 min  |
| 28  | Test `SendRecv` with context already cancelled before call                                           | Low    | 5 min  |
| 29  | Test stale pending config (config for tracking, commit for audio → no match, audio pending survives) | Low    | 10 min |
| 30  | Test circuit breaker recovery: open → probe finds device → reset → works                             | Low    | 25 min |

---

## g) Questions

**1. Should I squash the garbage auto-commits (`c34ac8a` blank, `83a5343` malformed) from prior sessions?**

These are from the auto-commit daemon and predate this session. They pollute `git log --oneline` with unsearchable entries. I did NOT touch them this session (rule: don't revert changes you didn't author). The new commits (`3082afb`, `83d7b1e`) are clean. But the old garbage remains. Should I `git rebase -i` to squash them, or leave history as-is? This rewrites history — your call.

**2. Should I pursue Layer 2 (`/dev/uhid` virtual device) or Layer 3 (NixOS VM test) next?**

Layer 2 tests real file I/O paths (`hidrawDevice.Send`/`SendRecv`) against a userspace-created HID device backed by `pixyProtocolState`. It would catch bugs the simulator can't (file descriptor handling, timeout expiration, partial writes). Layer 3 tests the full systemd service lifecycle in QEMU. Both are significant infrastructure investments (2-4 hours each). Which layer would deliver more value for this project?

**3. Should the 4 simulator status reports be consolidated into one, or left as-is with SUPERSEDED annotations?**

I annotated the 3 prior reports with SUPERSEDED headers (non-destructive). An alternative is to move them to `docs/status/archive/` and keep only the final self-review. The current approach preserves history but clutters `docs/status/` (which already has 38 files). Archiving would be cleaner but loses the narrative of how the simulator evolved. Which do you prefer?
