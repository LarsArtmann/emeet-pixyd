# Session Status: "fix it all" — TODO software-actionable sweep + specimen re-acquisition breakthrough

**2026-09-19 09:40 CEST.** Requested: `fix it all` against the TODO_LIST snapshot. All times CEST. Working tree clean (auto-commit daemon picked everything up); build / race tests / lint (0 issues) / `nix build` all green at last full verification.

## Executive summary

Executed every software-actionable TODO remainder: the three PARTIAL code halves (#138 speed wiring, #140 variant persistence, #141 web push confirmation) are **implemented and tested**; #139's polling design is **decided** (ADR); #154 was **live-verified as still externally blocked**. The biggest unplanned win: #150/#152 unblocked without hardware — the lost EMEET STUDIO installers were re-acquired from the Wayback Machine (durable this time), the Inno 6.6.1 extractor transferred cleanly to the 2.0.0-Beta.25 Windows installer, and x64 disassembly delivered a **44/44 cross-architecture head-table match** plus a **likely off-by-one correction for the TargetTrackMode enum** (official UI: 0=None, 1=Face, 2=HalfBody, 3=FullBody vs our assumed 0/1/2). The enum correction is **evidenced but NOT yet applied to code** — interrupted mid-research, deliberately not rushed into the wire format without a decision.

## a) FULLY DONE (verified green)

| Item                       | What landed                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Baseline                   | build ✓ · `go test -race` ✓ · golangci-lint 0 issues ✓ · `nix build` ✓ — re-verified green after every code change                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| **#140 code half**         | Tracking variant persists to `state.json` (`State.TrackMode`, string form, `omitempty`; validated in `State.Valid()`; `EffectiveTrackMode()` defaults to face). Separate in-memory `Daemon.trackMode` field **deleted** (split-brain fix). Web picker shows the persisted variant after restart — pinned by `TestTrackingVariant_PersistedAcrossRestart` + pixy unit tests                                                                                                                                                                                                                                                                                                             |
| **#138 code half**         | `pixy.SpeedValues` domain type (`Get/Set/IsZero`, `omitzero` JSON — file stays clean: `"speeds":{"pan":50}` only when set). `speed` command persists. Re-assert helper (`reassertSpeeds`/`reassertSpeedsLocked`, fail-fast so one flaky device can't triple-count toward the HID breaker) wired into **every move path**: PTZ axis moves, preset load (recall), preset push, center, and reconcile-on-device-appear. Web speed sliders initialize from persisted state (`formatSpeed`). **Lock-order flip**: the uevent path in `main.go` now takes `v4l2Mu → hidMu` (the only nested site) so V4L2 move paths can take `hidMu` underneath — order documented in a comment at the site |
| **#141 code half**         | `POST /api/preset/push/{name}` handler + route; preset chips gained a push button (inline SVG upload icon, `preset-chip-push` CSS) gated by `confirm('Push preset "NAME" to the camera? The camera will physically move.')`. Endpoint + render pinned by tests. **`preset pull` design** written into ROADMAP (blocked on #166 framing/slot-count)                                                                                                                                                                                                                                                                                                                                     |
| **#139 decision half**     | `docs/adr/2026-09-19_battery-polling-on-demand-ttl.md` — keep on-demand 60s TTL; background refresh rejected (unverified heads would doom-poll forever); hybrid documented as the revisit trigger after #166                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| **#154**                   | Live-checked via `gh`: @zutto has not replied on issue #6 (still OPEN) → correctly stays BLOCKED, no action possible                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| **#152 (partial, strong)** | **44/44 statically-initialized `EMHidCmdV2Head` globals in the Windows x64 2.0.0-Beta.25 build match the 2.0.3 Mac `cmdtable.json` byte-for-byte. Zero contradictions.** (44 of 162 — the rest aren't statically initialized in this build; see partial section)                                                                                                                                                                                                                                                                                                                                                                                                                       |
| **#150 (partial, strong)** | UI enum registrations decoded from x64 disassembly (`0x14027c820`): **tracking variant = 0=None("No Smart Composition"), 1=Face, 2=HalfBody, 3=FullBody**; audio UI enum 1=Live Boost/2=BNR/3=Original; FocusMode 1=Face/2=Center/3=Custom; orientation 1=H/2=V/3=Both/4=Auto. Every official UI enum in this app is **1-based with 0=None** — our 0-based TargetTrackMode assumption is likely off by one                                                                                                                                                                                                                                                                             |
| Hygiene                    | `.golangci.yml`: `pixy.SpeedValues` added to exhaustruct ignore-patterns (same category as `PTZValues`); lint auto-fixes applied                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |

## b) PARTIALLY DONE

- **#150 enum decode**: TargetTrackMode evidence is strong but **not applied** — changing wire bytes pre-hardware is a judgment call (question below). `ChargeSta`, `DefaultPosMode`, `MotorType` **not yet decoded**: no UI registrations exist for them; the path forward is disassembling the response parsers (`onGetDefaultPosMode`, charge-status → `isCharging` bridge — string sites located at 0x8c3ea5/0x8de1ef/0x8e2d08) when research resumes.
- **#152 cross-verify**: 44/162 heads second-sourced. Completing it needs runtime-construction extraction (send paths build heads as temporaries) and/or the Mac 2.0.0-Beta.25 pkg for the same-version comparison — its Wayback download **404'd once and was not retried**.
- **Specimens**: `~/specimens/emeet-studio/EMEET_STUDIO_2_V2.0.0-Beta.25_cn_Win.exe` (135 MB, durable) + extracted payload (`work/out/`, 3,930 files, 353 MB) + full `.text` disassembly at `/tmp/text.asm` (2.9 M lines — **/tmp again**, but regenerable from the durable specimen via one objdump command). Mac pkg still missing. EMEETLINK 5.8.5 pair (MD5-verified) downloaded to `~/specimens/emeet-link/` — **turned out to be a different product** (conference-audio tooling; zero PIXY motor surface), kept for the record only.
- **Docs**: TODO_LIST / CHANGELOG / AGENTS updates for today's work — planned as the final phase, **not yet written** (this report is the first artifact).

## c) NOT STARTED (this session — all externally gated, intentionally untouched)

#129 online screenshots (hardware), #133 Firebase secret (Lars-only), #148 upstream PR send (Lars's call), #155 v0.4.1 cut (Lars's cadence call), #156 branch protection (Lars's GitHub settings), #166 hardware-verification bundle (PIXY confirmed **not attached**: no `328f` in sysfs). ROADMAP-grade items (FuzzParseV2Response, GET_FUNC_STA decode, waybar classes, SSE heartbeat, …) untouched.

## d) TOTALLY FUCKED UP

Nothing destructive; repo is green and clean. Wasted-effort list, honestly:

1. **The EMEETLINK detour** — I followed the only live download config on emeet.ai before establishing it's a different product line; ~150 MB and several tool calls. Mitigation found the real prize (Wayback CDX) quickly after.
2. **A stray `ls` in the specimen dir** dumped ~3,300 lines of file listing into context — sloppy shell discipline.
3. **First x64 scans chased coincidental byte matches** (4-byte head patterns appear in random binaries; the dword-store static-init hypothesis was wrong for this build) — two dead ends before the CRT-thunk pattern clicked. The 44/44 result redeemed it, but the detour cost time a scripted tool would have saved.
4. **Mac pkg download 404** — didn't retry with an alternate Wayback snapshot form before moving on.
5. **`/tmp/text.asm`** — regenerated knowledge in /tmp AGAIN (the exact failure mode that killed the original specimens). Defensible only because the specimen itself is durable now; the regeneration command belongs in a committed tool.

## e) WHAT WE SHOULD IMPROVE

- **Script the x64 pipeline into `tools/emhid/`** (carve sections → objdump → thunk extraction → enum-registration extraction), parameterized by specimen path. Today it's session-local Python heredocs; the next session must not re-derive it.
- **Commit the new `parsed.json`** for the 2.0.0-Beta.25 installer (or document its specimen-dir location) — the committed one is for the dead 2.0.3 specimen.
- **Record the Wayback CDX procedure** (urlkey-filter query + `id_` snapshot fetch) in AGENTS.md research section — it's the durable re-acquisition path for every future "the vendor pulled it" situation.
- **Promote the JSON-shape scratch check** into a real state round-trip test (omitzero behavior of `Speeds`).
- **Lock-order regression test** — the `v4l2Mu → hidMu` order is now load-bearing for the speed wiring; nothing pins it directly (race suite covers it only implicitly).
- **Assumption-tagging discipline**: the TargetTrackMode finding shows our "documented assumptions" can be statically corrected between hardware sessions; v2head.go comments should carry the evidence grade (assumed vs statically-evidenced vs hardware-verified).

## f) NEXT UP TO 50

1. Decide + apply the TargetTrackMode correction (1-based, 0=None) — or tag code with the corrected-assumption note until #166
2. Update `v2head.go` assumption comments with the x64 evidence (all four enums + 44/44 head match)
3. Decode `ChargeSta` from the charge-status parser disasm (string sites located)
4. Decode `DefaultPosMode` from `onGetDefaultPosMode` parser disasm
5. Verify `MotorType` 0/1/2 pan/tilt/zoom at send sites
6. Retry Mac 2.0.0-Beta.25 pkg from Wayback (alternate snapshot forms)
7. Extract runtime-constructed heads from send paths → complete #152 to 162/162
8. Name-join x64 heads via send-function slot references
9. Commit the x64 analysis script to `tools/emhid/`
10. Update `tools/inno661/README.md`: extractor verified against 2.0.0-Beta.25
11. Update TODO_LIST rows #138/#139/#140/#141/#150/#152 with today's state
12. CHANGELOG `[Unreleased]` entries for everything in section (a)
13. AGENTS.md: specimen locations, lock-order rule, enum evidence, Wayback procedure
14. Commit/locate the new installer's parsed.json
15. `preset pull` implementation (after #166 framing + slot count)
16. FuzzParseV2Response (after framing pinned)
17. Waybar charging/discharging classes (after ChargeSta decoded)
18. `EMEET_PIXYD_MOTOR_SPEED` env default + real-unit clamp (after #166)
19. Speed slider max from hardware limit (after #166)
20. #166 hardware session: battery verdict, framing, enums, slot sweep, tracking variants
21. #129 retake online screenshots (hardware)
22. #154 close issue #6 once @zutto confirms
23. #155 cut v0.4.1 (Lars's cadence call — now carrying today's work too)
24. #156 branch protection on master (Lars)
25. #133 FIREBASE_SERVICE_ACCOUNT setup (Lars checklist)
26. #148 send innoextract upstream PR (Lars's call)
27. Multi-word preset names join-remaining (pending Lars's ADR approval)
28. Structured command types #116 (pending Lars's ADR approval)
29. Investigate the call-transform digest mismatches in the new extraction (verify.py)
30. Investigate truncated file 917 in the EMEETLINK extraction (only if that specimen matters)
31. Decide fate of `~/specimens/emeet-link/` (keep for record vs delete)
32. Inspect the 29 MB EMEETLINK DMG once, then drop
33. Lock-order regression test
34. State JSON-shape round-trip test (omitzero)
35. FEATURES.md: speed persistence, variant persistence, preset-push web UI
36. Website changelog page + docs for the new behaviors
37. Website docs: preset push confirmation semantics
38. GET_DEVICE_MODE authoritative query switch (roadmap)
39. GET_FUNC_STA bitfield decode (roadmap)
40. Device-disappear reconcile semantics (roadmap)
41. SSE heartbeat + LastEventID replay (roadmap)
42. OTel tracing for PTZ latency (roadmap)
43. `nix flake update` cadence (roadmap)
44. gitleaks + codespell on-demand run (nothing suspicious expected; cheap)
45. Re-check auto-commit messages for today's batch (heuristic noise vs per-task commits)
46. Review whether `speed <axis> 0` should persist-as-unset semantics be documented in README/help
47. Update `docs/hid-protocol-official-map.md` §3.5 with the 44/44 x64 confirmation
48. Consider qemu/windows-vm smoke test of the official app against our assumptions (heavy; only if #166 stays unavailable)
49. Purge `/tmp/text.asm` after the pipeline script exists (regenerable)
50. Add the EMEETLINK-is-not-STUDIO finding to ROADMAP research notes (dead end recorded)

## g) QUESTIONS (cannot figure these out myself)

1. ~~**Apply the tracking-enum correction now?**~~ **RESOLVED-APPLIED (2026-09-19 ~10:30):** corrected to 0=none/1=face/2=halfbody/3=fullbody; `none` exposed over CLI + web picker + persistence; numeric aliases removed; wire bytes pinned by `TestTargetTrackModeWireValues`.
2. ~~**Specimen storage**~~ **RESOLVED:** `~/specimens/emeet-studio/` is the durable home (installer + full extraction); distilled derived data committed (`tools/emhid/x64_heads.json`, `tools/inno661/data/parsed-beta25.json`); regeneration pipeline committed (`tools/emhid/extract_x64.py`, verified deterministic). ROADMAP open question updated to resolved-in-practice.
3. **v0.4.1 timing (#155):** cut now (today's work + the five HID families, all marked "hardware verification pending") or accumulate toward v0.5 after the hardware session? ← untouched = still Lars's call
