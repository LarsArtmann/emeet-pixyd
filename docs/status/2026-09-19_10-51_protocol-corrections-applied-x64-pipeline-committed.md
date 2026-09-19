# Session Status: applied the statically-evidenced protocol corrections — framing, ChargeSta, tracking enum, x64 pipeline committed

**2026-09-19 10:51 CEST.** Charter: the standing "READ, UNDERSTAND, RESEARCH, REFLECT — execute and verify, keep going until done" directive. The two prior reports ended "WAITING FOR INSTRUCTIONS" with 3+3 questions for Lars; the same keep-going directive arrived twice, so this session treated it as the decision and executed everything the evidence supports — while leaving genuinely Lars-only items (release tag, upstream PR, Firebase secret, branch protection, the join-remaining ADR) untouched. Build ✓, race tests ✓, lint 0 issues ✓, `nix build` ✓, website build 19 pages ✓ — verified after every phase.

## Executive summary

The V2 protocol implementation now matches the official app's statically decoded behavior instead of documented guesses: response framing corrected (payload@8 + head-echo validation + min-len 9 — our offset-4 parsers were provably wrong), `TargetTrackMode` corrected to the official 1-based enum with the `none` variant exposed end-to-end, `ChargeStatus` typed with the {1,2}=charging predicate wired into battery output and new Waybar charging classes. The x64 extraction pipeline is committed (`tools/emhid/extract_x64.py`) and **verified deterministic**: a fresh regeneration from the durable specimen reproduced the committed 108-head artifact byte-identically. Both prior reports' questions are annotated resolved inline; the full docs sweep (the thing that kept not happening) is done.

## a) FULLY DONE (this session, verified green)

| What                            | Detail                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Response framing correction** | `pixy.V2ResponsePayloadOffset = 8`; every V2 parser now takes the request head AS SENT and validates the echo + reserved dword + min length via the shared `v2Payload` helper (`ErrV2ResponseHeadMismatch` added); simulator response builder uses the same constant; evidence-graded comments throughout `v2head.go`                                                                                                                                                                                                                                       |
| **TargetTrackMode correction**  | 0=none, 1=face, 2=halfbody, 3=fullbody; `none`/`off` CLI aliases + web picker segment + persisted state validation; numeric aliases (encoded the disproved mapping) removed; `TestTargetTrackModeWireValues` pins the bytes                                                                                                                                                                                                                                                                                                                                 |
| **ChargeStatus typed + wired**  | `{1,2}`=charging predicate (`TestChargeStatusPredicate`), enum-validated parses, `battery` output and Waybar `charging`/`discharging` classes (`TestWaybarBatteryClass`, including charge value 2)                                                                                                                                                                                                                                                                                                                                                          |
| **x64 pipeline committed**      | `tools/emhid/extract_x64.py` (CRT-thunk sweep, xor-zero handling, artifact filter) + `x64_heads.json` (108 heads, 190 KB); developed against the surviving `/tmp/text.asm`, then a full objdump regeneration from the specimen produced a **byte-identical** artifact                                                                                                                                                                                                                                                                                       |
| **Beta.25 inno artifacts**      | `tools/inno661/data/parsed-beta25.json` + `setup0_offsets-beta25.json` committed; README notes the second-installer verification + `mkdir -p data`                                                                                                                                                                                                                                                                                                                                                                                                          |
| **Regression tests**            | `TestStateRoundTrip_SpeedsOmitZero_TrackModeNone` (on-disk JSON shape: omitzero speeds key, `trackMode:"none"`) and `TestLockOrder_V4L2MoveWithHIDCommands` (mixed pan/speed/tracking/status workload with a 10s deadlock watchdog pinning `v4l2Mu → hidMu`)                                                                                                                                                                                                                                                                                                |
| **Docs sweep**                  | TODO_LIST #138–#141/#150/#152 rewritten to current truth (all PARTIAL-with-wiring, #152 unblocked 108/162); CHANGELOG [Unreleased] entries for both sessions' work; AGENTS.md (evidence grades, lock-order rule, research-artifacts section incl. Wayback CDX recipe + method lesson); map doc §3.5a (framing + enums + version-shift model) + §6 reopened-as-broken-open; ROADMAP (specimen question resolved, EMEETLINK dead end, preset-pull design now concrete); FEATURES rows + summary; README + website CLI-reference/waybar/changelog pages synced |
| **Report annotations**          | 09:40 §g Q1/Q2 and 10:04 §g Q1–Q3 struck through with RESOLVED-APPLIED markers + evidence; v0.4.1 timing left open for Lars                                                                                                                                                                                                                                                                                                                                                                                                                                 |

## b) PARTIALLY DONE

- **#150/#152 research remainders** (deliberately analysis-only, no code impact): `MotorType` 0/1/2 still unverified at send sites (payload builder `0x1403c53c0` is the thread); DefaultPosMode value semantics undecoded; 53 Mac-only heads need the inline-sender extraction path; Mac pkg Wayback retry untried. All carry evidence-graded comments now, so a future session knows exactly what is assumed vs evidenced.
- **Hardware confirmation** for everything applied today stays on the #166 list (framing on-wire, reserved bytes 4..7, enum confirmation, speed unit, slot count). Nothing shipped today was hardware-verified — but nothing it replaced was either, and the replacements are the official app's own logic.

## c) NOT STARTED (intentionally)

Lars-gated: v0.4.1 tag (#155), innoextract PR send (#148), Firebase secret (#133), branch protection (#156), multi-word preset join-remaining (ADR approval). Hardware-gated: #166 bundle, #129 screenshots. #154 still waiting on @zutto.

## d) TOTALLY FUCKED UP (honest ledger)

1. **Blind edit clobbered a test body**: an insert-before edit on `pixy_test.go` replaced the first lines of `TestSpeedValuesSetGetZero` instead of inserting above it — caught immediately and restored; the final file is correct, but the lesson is the same one the edit-tool docs teach: anchor inserts on unique context ABOVE the target, never on the target's signature.
2. **Wrote the wrong file in a multiedit**: targeted a `v2head.go` const block with `file_path` pointing at `pixy_simulator_v2_test.go`; the edit failed atomically (old_string absent) and was re-applied to the right file. No damage, but sloppy.
3. **First extractor run found 0 heads** (call-target matching compared line starts against the target address) and the second found 0 complete thunks (regexes searched operands without the mnemonic). Both were single-fix bugs found by running against the known-good ground truth — the discipline of verifying against `/tmp/x64_stubs_dedup.json` before trusting the script is what made the committed artifact trustworthy.
4. **One transient socket-test flake** (`TestSocket_CommandsNoDevice/audio`: connection refused) failed one full-suite run and passed 5×-repeated + two full reruns. Not investigated further — recorded here so the next occurrence has a witness.
5. Reconstructed the extractor from outputs + disassembly instead of the lost prototype script — cost ~20 minutes, but the reconstruction is now the committed, documented, deterministic version (silver lining, not an excuse).

## e) WHAT WE SHOULD IMPROVE

- **The "waiting for instructions" deadlock broke**: two sessions stalled wire-behavior decisions on questions Lars never answered while repeatedly saying "keep going until done". The resolution rule used today — evidence-backed corrections to never-hardware-verified assumptions are reversible and land flagged; release-ops and ADR-gated designs stay gated — should be the default reading of that directive.
- **Commit analysis scripts the moment they work** — done today, and the byte-identical regeneration proof is the template: a script isn't "committed" until a from-source rerun reproduces the artifact.
- The docs sweep finally ran FIRST-class after the code, not "later": TODO/CHANGELOG/AGENTS/map/ROADMAP/FEATURES/README/website all tell the same story now.

## f) NEXT UP TO 20

1. #166 hardware session (now the single gate for: framing on-wire, reserved bytes 4..7, enum confirmation, MotorType, speed unit, slot count, battery verdict, screenshots #129)
2. v0.4.1 cut decision (Lars — today's corrections want a release)
3. Decode `MotorType` at send sites via payload builder `0x1403c53c0`
4. Extract the 53 Mac-only heads via inline-head senders (`0x14017f110` callers)
5. DefaultPosMode value semantics (sender UI path)
6. Mac 2.0.0-Beta.25 pkg Wayback retry (alternate snapshot forms)
7. `preset pull` implementation (response format decoded; blocked on #166 slot count)
8. `FuzzParseV2Response` (framing now pinned statically — worth fuzzing the parser against the head-echo validation)
9. #154 close issue #6 once @zutto confirms; #148 PR send (Lars)
10. Multi-word preset join-remaining after ADR approval (Lars)
11. `EMEET_PIXYD_MOTOR_SPEED` env default + slider max (after #166)
12. `TestSocket_CommandsNoDevice` flake — watch for recurrence
13. Sweep the `CMD_*_VAL` string census into a doc table (cheap, useful at #166)
14. EMEETLINK specimen fate (`~/specimens/emeet-link/`, ~600 MB) — keep-for-record vs delete (Lars)

## g) QUESTIONS

1. **v0.4.1 timing** (carried, the only surviving question): cut now with today's corrections (all marked "statically evidenced, hardware-unconfirmed") or after #166?
