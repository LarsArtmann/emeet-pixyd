# Status Report — 2026-10-01 05:23 — Privacy-Trap Bug: Reproduced, Root-Caused, Fixed, Hardware-Verified

**Session scope:** Lars reported "tilt to −85 or more auto-enters Privacy and Track cannot re-enable it (at least from the web UI)" with real hardware attached, and asked for a bunch of tests. This report covers that session only.

**TL;DR:** The bug is real, firmware-level, fully characterized with optical ground truth, FIXED in the daemon, and the fix is verified on the physical camera across every measured race delay. 15 new tests (9 unit/simulator + 6 hardware). Full suite green: unit, `-race`, lint 0 issues, `nix flake check`, complete `-tags=integration` hardware run (227s). The running daemon service still executes the OLD build — restart needed to pick up the fix.

---

## a) FULLY DONE

1. **Bug reproduced on hardware.** Arriving at tilt ≤ −85° while tracking/idle makes the firmware cover the lens (mean frame luma ~130 → ~7-48). Verified at −85 and −90, repeatedly.
2. **Root cause established empirically** (optical verdicts — mean luma of MJPEG frames via `v4l2-ctl --stream-mmap` + JPEG Y-plane average; the mode-query surfaces lie, see b.3):
   - A camera-mode write landing within ~1.5-3s of trap arrival is **ACKed but silently dropped** (the user's "cannot re-enable from Privacy by putting it directly in Tracking").
   - The eaten write **poisons** the mode interface: every later tracking write stays dropped while parked. A privacy→track bounce does NOT recover (matrix round B).
   - Only an **idle write clears the poison** (round C), after which tracking sticks again even at −85 (round D).
   - Leaving the tilt zone re-opens the lens automatically, no write needed.
   - `privacy` covers at any tilt and holds through V4L2 moves; `center`/tilt moves work in every state (V4L2 motor control is independent of the cover).
3. **Fix implemented** — `privacy_trap.go` (new): tilt commands into the zone and the delayed PTZ readback arm a 6s race window (`d.trapArmedUntil`, under `d.mu`); `setTracking` splits into `writeTracking` (raw write) + scheduling — a track/idle write issued while armed schedules ONE deferred re-assert 3s later that re-enters tracking via an **idle bounce**. Skipped when the believed mode moved on (user changed intent), never scheduled for privacy writes, goroutines tracked in `d.trapReasserts` (WaitGroup, via `WaitGroup.Go`). No recursion (the re-assert calls `writeTracking`).
4. **Fix verified on hardware**: the timing-race test (0/300/800/1500ms delay between tilt and track) failed all four rounds before the fix (luma 12-38); passes all four after (luma 137-147).
5. **Test bunch built:**
   - `privacy_trap_test.go` — 9 unit/simulator pins: bounce sequence + write-order, no-reassert-outside-window, intent-change skip, idle-reassert-skips-bounce, privacy-never-reasserts, full-command-path scheduling, PTZ arming table (−85/−90 arm; −84/−20/pan don't), readback re-arm on arrival, window expiry.
   - `integration_privacy_hardware_test.go` — 6 hardware tests: query-surface snapshot walk (TiltPrivacyTrap), settled-write recovery (PrivacyTrapRecovery), the fix-verification timing race, the recovery matrix (rounds A-D), zone-exit/idle-parity boundaries. Verdicts are **baseline-relative** ratios (covered < 50% of open-lens baseline) after absolute thresholds broke against rising morning light.
6. **BatteryProbe executed against real hardware** (first time ever): dev-0x00 heads (battery `09 00 00 02`, charge `09 00 00 06`) and target-track/func-status/ver/device-version **time out**; motor-speed/pos, device-mode, SN answer 32-byte echo frames with mostly-zero payloads; the motor-pos echo **rewrites the queried 0x03 iface to 0x63 on the wire**; response bytes 4..7 are `00 0X 00 0X` (not the statically-assumed zero-reserved dword); V1 `09 01 01 01` answers constant `0x02` (and `0x03` in the trap state).
7. **Pre-existing hardware-machine test failure fixed**: `TestHandleSpeedCommand_SetMotorSpeedFailureCountsTowardBreaker` failed on any machine with a real PIXY attached (send-failure re-probe replaces the simulator and legitimately resets the breaker counter, `probe.go:323`). Now tolerates the re-probe-replaced case and keeps the CI assertion.
8. **devShell gained `v4l-utils`** — hardware tests shell out to `v4l2-ctl`, which was only on the daemon service PATH; every PTZ move in a devShell silently failed before.
9. **Docs updated**: CHANGELOG [Unreleased] Fixed entry (trap + fix + evidence); TODO_LIST #166 checklist annotated with this session's partial answers + two new checklist items (V1 query decode, panel honesty); AGENTS.md (privacy-trap key behavior, `privacy_trap.go` file-table row, hardware-testing notes, query-surface warning).
10. **All gates green at session end**: `GOWORK=off go test -race -count=1 ./...` ✓, `golangci-lint run` 0 issues ✓, `nix flake check --no-build` ✓, full `-tags=integration` suite ✓ (227s), `nix fmt` clean. Auto-commit daemon picked everything up (working tree clean).

## b) PARTIALLY DONE

1. **#166 hardware-verification bundle** — hardware was attached for ~1h and only the bug-related threads were pulled. Answered: query surfaces (above), 0x63 echo on the wire, bytes 4..7 shape, motor-speed payload `0x20` (=32, unit unknown). NOT answered while the camera was RIGHT THERE: MotorType enum pin, speed unit/hardware limit, preset slot count + response shape, live preset push→pull round-trip, online screenshots (#129).
2. **#139 battery/charge** — the de-risk question is answered (dev-0x00 heads time out on the wired unit → per map doc §#139, this is the "battery is PIXY-Wireless-only, close as wontfix" signal), but the wontfix decision + TODO closure is Lars's call, not made.
3. **Evidence-reference doc NOT updated** — `docs/hid-protocol-official-map.md` §3.5a is the designated home for evidence grades; this session's hardware findings live only in TODO/AGENTS/CHANGELOG. The map doc still describes bytes 4..7 as a reserved dword and does not carry the 2026-10-01 hardware observations.
4. **erraudit gate not re-run** — AGENTS documents STRICT-mode erraudit as a maintained gate (0 violations baseline). My new code follows the errorfamily construction rules and lint is clean, but the erraudit tool itself was not executed after the changes.
5. **CI parity partially verified** — `nix flake check --no-build` ✓ but a full `nix build` (production build, FOD path) was not run; vendorHash is untouched so risk is low, but unverified. `govulncheck` not run locally either.
6. **End-to-end web verification** — the fix was verified through `handleCommand`, which is byte-for-byte the same code path as `POST /api/track` (web action is a plain pass-through), but nobody literally clicked the button in a browser after the fix.

## c) NOT STARTED

1. **Daemon service restart/rebuild** — the running service (PID from the nix store, built at 1a7b698d-era) still has the bug. Lars's deployment flow (nixos-rebuild), deliberately not touched.
2. **Map doc §3.5a hardware-evidence update** (see b.3).
3. **Audio/gesture write-eating during the trap window** — only the tracking interface was evidenced; iface 0x04/0x05 writes during the ~1.5-3s window were never tested.
4. **Trap threshold stability across power cycles** — is the trigger still −85 after a power cycle? Unknown.
5. **Far-travel trap race** — all evidence used 0→−85 (85° travel); a 175° travel (from +90) may extend the eat-window beyond the 6s arm window.
6. **Preset-load arming** — a preset that loads tilt into the zone does not arm the re-assert window (only `handlePTZCommand` and the readback do).
7. **`TestSocket_PanTiltZoom` skip** — skipped in the full run (pre-existing, not investigated this session).
8. **`zz_dump_test.go` stale-vet-cache phantom** — a `go clean -testcache` fixed it; origin never identified (likely another session's scratch file, no git history).
9. **Daemon-shutdown wait for `trapReasserts`** — the WaitGroup exists but only tests wait on it; `handleShutdown` doesn't (cosmetic: goroutines die with the process anyway).

## d) TOTALLY FUCKED UP!

Nothing catastrophic, nothing shipped broken. Honest process-stumble ledger (all caught and fixed within the session):

1. **Clobbered an existing CHANGELOG entry** with a careless old_string replace (deleted the website-CI-lockfile entry; caught immediately and restored). Exactly the "replace that silently deletes memory" anti-pattern.
2. **Mangled a test-file comment header** by anchoring an edit on a doc-comment line (orphaned half a comment; fixed).
3. **Shipped a nonsense placeholder first** — a fake `luma()` stub with a bogus interface and an unused `slices` import-guard, caught by the compiler.
4. **Edit hygiene vs the formatter**: repeatedly edited files `nix fmt` had just reformatted without re-reading → several edit failures and one mangled comment; wasted cycles.
5. **Filtered out the evidence I needed**: `grep -vE "re-assert"` hid the exact log line that explained the first failed fix-verification round.
6. **Absolute luma thresholds** (dark<30 / bright>60) were broken by design — ambient light rose during the session and covered-lens readings drifted 7→48; mid-session fix to baseline-relative ratios.
7. **First hardware diagnostic ran with a broken environment** — `v4l2-ctl` missing from the devShell meant the first 40s experiment measured nothing (all moves failed silently to the test's eyes).
8. **`waitForCondition` raced config-vs-commit** — asserted on the config byte before the commit landed, producing a spurious pass and a wrong final mode; fixed with sequence-aware conditions (mode settled + last-config-ordering).
9. **First fix version had a data race** (`-race` caught the delay-var restore vs the leaked goroutine) — the goroutine outlived its test. Fixed properly (capture-at-schedule + WaitGroup.Go), but goroutine lifetime should have been designed in from the start.
10. **One multiedit contained an accidental no-op edit** (identical old/new strings) — sloppy tool input.
11. **Delayed realization that the re-assert of the SAME mode cannot work** — the first fix shipped a plain 3s same-mode retry and only the hardware failure of that version forced the matrix experiment that discovered the poisoning. Should have run the recovery matrix BEFORE designing the fix.

## e) WHAT WE SHOULD IMPROVE!

1. **Hardware-session discipline**: when the PIXY is attached, work the #166 checklist opportunistically — the camera sat there while only the current bug was chased. One wired session closes #138/#139/#140/#141/#150 + #129; we left most of that on the table.
2. **Evidence doc in the same session**: discoveries like the `00 0X 00 0X` framing and the lying V1 query belong in the map doc (the designated evidence reference) immediately, not just in TODO annotations.
3. **Gate completeness**: the project's own bar (erraudit STRICT, govulncheck, full `nix build`) should be re-run after nontrivial changes — I claimed "done" on lint+race+flake-check alone.
4. **Environment preflight before hardware runs**: assert `v4l2-ctl`/device presence at test start (the luma tests now degrade gracefully, but the early PTZ failures were pure environment).
5. **Never grep -v logs you don't fully understand yet.**
6. **Model-first for firmware state machines**: the recovery matrix (A-D) cost ~2min of hardware time and would have prevented a whole wrong fix iteration. Exhaustive small matrices before designing countermeasures.
7. **Edit discipline after external formatters**: re-read before every edit post-`nix fmt`; more than half my failed edits were self-inflicted this way.
8. **Consider surferving "lens covered" honesty in the panel** during the ≤3s race window (belief says tracking, feed is black) — currently invisible; the re-assert hides it in practice but a slow far-travel + fast click could still show a black feed under an active Track card (TODO item added).

## f) Up to 50 things to get done next

**Immediate (user decisions + quick wins):**
1. Restart/rebuild the daemon service so the running daemon has the fix (Lars's nixos-rebuild).
2. Close #139 as wontfix? (dev-0x00 battery/charge heads time out on the wired unit — the documented de-risk trigger fired).
3. Continue the #166 checklist THIS session while hardware is attached? (MotorType, speed unit, preset shape, round-trips, screenshots.)
4. Run `erraudit ./... --type-aware --enforce-go-error-family --enforce-samber-oops --enforce-generic-return` — confirm 0 violations still.
5. Run full `nix build` (CI parity) and `govulncheck`.
6. End-to-end verify via the actual web UI (click Track after tilting to −85).

**#166 bundle remainder (hardware attached):**
7. Pin `MotorType` enum values {0,1,2} against real moves.
8. Pin motor-speed unit + real hardware limit; clamp at the command layer + web slider max (#138).
9. Preset slot-count sweep + response-shape pin (mode-only GET vs full SET-echo) (#141).
10. Live round-trip: `preset push` → `preset pull` → values match.
11. Retake ONLINE website screenshots (live MJPEG, tracking active) → #129.
12. Decode response bytes 4..7 (`00 0X 00 0X`) — counts? framing?
13. Decode the V1 `09 01 01 01` answer semantics (constant 0x02, 0x03 in trap) — until then `sync`/reconcile mode reads are untrustworthy (already a TODO row).
14. Decode v2 device-mode payload `0x20` (=32) — what enum/bitfield is this?

**Protocol/robustness hardening of what shipped:**
15. Test audio/gesture writes during the trap window (iface 0x04/0x05 eating?).
16. Far-travel trap race test (+90 → −85); extend arm window if the eat-window grows.
17. Trap threshold after a power cycle (does −85 stay the trigger?).
18. Arm the trap window on preset loads that land tilt in the zone.
19. Wait on `d.trapReasserts` in `handleShutdown` (clean in-flight re-assert drain).
20. Surface a "recovering lens…" toast/info when a re-assert is scheduled (UX honesty).
21. Expose trap/lens-cover state in `/api/status` (widget contract addition, additive field).
22. Investigate the `TestSocket_PanTiltZoom` skip (pre-existing).

**Docs/evidence:**
23. Update `docs/hid-protocol-official-map.md` §3.5a with all 2026-10-01 hardware evidence (this session's biggest doc gap).
24. Consider a short website docs note on the tilt-bottom privacy behavior (it's firmware design, users will hit it).
25. Record the "optical ground truth when protocol reads lie" method in the map doc's method section.

**Housekeeping / CI:**
26. CI idea: an optional self-hosted hardware workflow running the integration suite when a label is set.
27. Pre-commit or CI check that devShell has needed runtime binaries for integration tags (v4l2-ctl presence assert).
28. Verify gopls LS diagnostics reconcile (paralleltest warnings shown stale all session while golangci-lint reported 0; possibly needs LSP restart).
29. The in-flight `/api/status` work from a parallel session rides in the same auto-commits — confirm it's complete/intended before release (not mine, untouched).
30. Release hygiene: #155 (v0.4.1 cut) now has a user-facing bugfix worth shipping.

## g) Questions I can NOT figure out myself (max 3)

1. **Restart the daemon now or later?** The running service still has the trap bug; picking up the fix needs your nixos-rebuild/restart (I did not touch the running service). Want me to leave that entirely to you?
2. **Close #139 (battery/charge) as wontfix?** The wired unit times out on both dev-0x00 heads — exactly the documented de-risk trigger for "battery is PIXY-Wireless-only". Your call per the ADR's revisit rule.
3. **Keep going on #166 while the hardware is attached?** The checklist remainder (items 7-11 above) is ~30-60min of wired session; say the word and I'll run it, or stop here.

---

*Point-in-time snapshot. Completed work detail lives in CHANGELOG.md [Unreleased]; open work in TODO_LIST.md. Auto-commit daemon has this report.*
