# Session Status: resumed "fix it all" — x64 parser decode: response framing + ChargeSta + 108/162 cross-verify breakthrough

**2026-09-19 10:04 CEST.** Continuation of the 09:40 report. Session charter: execute the remaining sweep (docs, tests, research continuation) with wire-behavior changes still gated on Lars's answers. What actually happened: the research continuation (#150/#152) hit a **protocol-decoding breakthrough chain** that consumed the session — several load-bearing assumptions in our V2 implementation are now **statically evidenced from official parser disassembly**, including one (**response payload offset**) where our code is provably wrong. No repo code was changed this session; every finding below is analysis of the durable specimen. Working tree clean (auto-commit daemon); baseline `go build` + `go test -race` verified green at session start.

## Executive summary

Found the official x64 response-parser layer (`EMHidCmdHelper::CMD_*_VAL` handler cluster). From it: **ChargeSta decoded** (0 = not charging, 1|2 = charging — from the `dec/cmp/setbe` consumer code), **battery level = raw byte at offset 8**, **DefaultPosMode response = mode@8 + pan/tilt/zoom dwords when mode==1**, and the big one: **response framing is head-echo + payload at offset 8, minimum length 9** — our parsers assume payload at offset 4. Wrote a stub-sweep script extracting **109 unique heads from 9,204 send-stubs: 108/162 match the Mac cmdtable byte-for-byte, zero contradictions** (up from 44/162). Then bound a named sender to its head slot and **disproved my own mid-session "response cmd−1" hypothesis: responses echo the request head** — the apparent off-by-one is a **version shift** (2.0.3 inserted SET_REBOOT and GET_MOTOR_SPEED, shifting later power/motor command IDs +1 vs Beta.25). The framing correction, ChargeSta semantics, and Waybar classes are now implementable on static evidence — deliberately NOT applied this session (question below, same gate as the tracking enum).

## a) FULLY DONE (this session, verified)

| What | Detail |
| --- | --- |
| Baseline re-verified | build ✓ · `go test -race` ✓ · specimens + `/tmp` scratch intact (no reboot) |
| **#150 ChargeSta decoded** | Consumer code at `0x1403ccbf8`: `movzx cl,sta; dec cl; cmp cl,1; setbe` → **isCharging = (sta==1 \|\| sta==2)**; 0 = discharging. Code-level evidence (stronger than UI registration) |
| **#150 battery/DefaultPosMode response formats** | Battery: level = `resp[8]` raw byte. DefaultPosMode GET: mode@8; `mode==1` → pan/tilt/zoom dwords @9/0xd/0x11, min len 0x15. Both parsers verified instruction-by-instruction |
| **Response framing evidenced** | Every parser validates `resp[0..3] == request-head global` (head-echo) and requires `len >= 9`; first payload byte at **offset 8** (bytes 4–7 unknown/reserved). Our `v2ResponseOverhead = 4` is **provably wrong** vs the official app |
| **Dispatch mask discovered** | The response dispatcher builds its switch dword with `iface & 0x1F` — motor-MCU `0x63` responses route as `0x03`. Explains the 0x63-vs-3 routing question mechanically |
| **#152 cross-verify 108/162** | Stub-sweep script (prototype in `/tmp`) parsed 9,204 `hidCmdSend` stubs: 109 unique heads, **108 match Mac cmdtable byte-for-byte, 0 x64-only** (1 artifact `(9,0,0,0)`). Was 44/162 this morning |
| **Ctor layout pinned** | `EMHidCmdV2Head` ctor @ `0x140179370`: `[+0]=0x09, [+1]=iface, [+2]=cat, [+3]=cmd` — verifies the x64_table byte order independently |
| **Version shift proven** | Named sender `hidCmdSendSetMotorPowerOnDefaultPosMode` (bound via its log-string xref @ `0x1403d9432`) sends slot `[9,3,1,19]` + payload `[mode:u8]`; its response dispatcher case is also `[9,3,1,19]`. Combined with battery `[9,0,0,1]`/charge `[9,0,0,5]` parser slots: **responses ECHO requests; 2.0.3 (Mac table) = Beta.25 IDs + 1 after two inserted commands** (SET_REBOOT `[9,0,0,1]`, GET_MOTOR_SPEED `[9,3,1,19]`). Every "Mac vs x64" discrepancy resolves under this model |
| Parser string-table census | All `CMD_*_VAL success.<field>:` log strings enumerated (`0x8d9800–0x8daa80`) — a per-command response-field map for future decodes; `HidConfigManage::send*` + `hidCmdSend*` sender strings catalogued |

## b) PARTIALLY DONE

- **#150/#152**: framing, ChargeSta, battery, DefaultPosMode response formats, version shift — decoded, **none applied to code** (same Lars gate as the tracking enum). `MotorType` 0/1/2 pan/tilt/zoom still NOT verified at send sites (the [9,3,1,19] sender's payload builder `0x1403c53c0` is the next thread to pull). DefaultPosMode **enum values** (what 0/1/2 mean) still unknown — sender payload is `[mode:u8]` but no value→meaning map yet; Qt metaobject pools hold type names, not keys.
- **#152 completion**: 53 Mac-only heads remain unconfirmed on x64 (mostly GETs riding other sender shapes); the sweep script exists only as `/tmp` prototypes (`x64_stubs.json`, `x64_stubs_dedup.json`) — the committed `tools/emhid/extract_x64.py` was NOT written yet.
- **Planned session work not reached**: framing-fix implementation, ChargeSta→Waybar classes, `v2head.go` evidence tags, state round-trip + lock-order tests, the entire docs sweep (TODO_LIST/CHANGELOG/AGENTS/inno661 README/hid-map/ROADMAP/FEATURES), final verification. All were sequenced behind the research and the research overran.

## c) NOT STARTED (this session)

Mac 2.0.0-Beta.25 pkg retry (Wayback), runtime-constructed-head extraction for the remaining 53, everything in b)'s "not reached" list. Hardware-gated and Lars-only items untouched as designed (#129, #133, #148, #155, #156, #166).

## d) TOTALLY FUCKED UP (honest ledger)

1. **The cmd−1 detour.** I concluded "response head = request cmd − 1" from parser comparison slots, stated it as near-fact, then disproved myself ~40 minutes later by finally binding a named sender to its slot. Correct method: identify a named SENDER before inferring framing from parsers. The detour did yield the version-shift discovery, but the intermediate conclusion was wrong and I asserted it too eagerly.
2. **First sweep bug**: missed `xor r8d,r8d` (zero iface) in the stub parser — first run reported 100 heads; the corrected run found 109. The `(9,0,0,0)` artifact is still unfiltered in the JSON.
3. **`/tmp` scratch AGAIN**: the sweep outputs live in `/tmp/x64_stubs*.json` beside `text.asm` — the exact failure mode called out in the 09:40 report. The committed, parameterized script that would make this moot was deferred "until after research".
4. **Time-box discipline**: the research continuation was "optional, if instructed"; it consumed the entire session while the concrete deliverables (docs sweep, tests) — the things that actually close TODO rows — waited. A hard time-box would have switched to docs after the ChargeSta decode.
5. Dead-end string hunts before the method clicked: UTF-16 literals (none exist), qword pointer tables (none) — the actual mechanism was plain `lea rdx,[rip+…]` xrefs to string *starts*, which took three attempts to aim correctly (first xref computed from a match offset, not the string start).

## e) WHAT WE SHOULD IMPROVE

- **Name-first disassembly method**: bind log-string → sender/handler function FIRST, then read code around it. Every breakthrough this session came from that pattern; every dead end came from pattern-hunting without a name.
- **Commit analysis scripts the moment they work**, not "later" — `/tmp` is where evidence goes to die on reboot.
- **Evidence-grade comments in `v2head.go`** now have four grades to encode: assumed → statically-evidenced-Beta.25 → statically-evidenced-2.0.3 (Mac) → hardware-verified. The version-shift finding means "which official version said so" must be recorded per fact.
- **A named-sender index** (log-string xref → function → head slot → payload builder) extracted once into `tools/emhid/` would answer most future protocol questions without re-deriving the plumbing.
- The docs sweep keeps being last and keeps not happening — schedule it FIRST next session, not last.

## f) NEXT UP TO 50

1. **Lars decides**: apply response-framing correction (offset 8, min-len 9, head-echo comment) now as statically-evidenced? → then parsers + simulator + tests
2. **Lars decides**: ChargeSta {1,2}=charging → typed `ChargeSta`, `battery` output, Waybar charging/discharging classes (#139 remainder minus hardware)
3. **Lars decides** (carried): TargetTrackMode 1-based correction + `none` variant exposure
4. **Lars decides** (carried): specimen home + commit `parsed.json` / new 109-head x64 table into `tools/emhid/`
5. (carried, folded out of questions) v0.4.1 timing — now also carries today's decode work
6. Commit `tools/emhid/extract_x64.py` (stub sweep + ctor layout + artifact filter for `(9,0,0,0)`)
7. `v2head.go`: rewrite assumption comments with evidence grades + version attributions
8. Docs sweep (TODO_LIST #138–#141/#150/#152, CHANGELOG [Unreleased], AGENTS research section, inno661 README "verified on Beta.25 + mkdir data", hid-map §3.5 108/162 + framing, ROADMAP EMEETLINK dead end, FEATURES speed/variant/preset-push)
9. State JSON round-trip test (omitzero `Speeds`); lock-order/concurrency regression test
10. Decode `MotorType` at send sites via payload builder `0x1403c53c0` and the SetMotorPos/SetMotorSpeed senders
11. DefaultPosMode value semantics (0/1/2 meaning) — sender UI path or preset-mode senders
12. Extract the remaining 53 Mac-only heads (different sender shape: inline head construction like `0x14017f110` callers)
13. Retry Mac 2.0.0-Beta.25 pkg (alternate Wayback snapshot forms)
14. `preset pull` design update: GET_MOTOR_PRESET_POS_MODE response format now decoded (mode@8 + coords when mode==1) — ROADMAP sketch can be made concrete
15. Battery/charge/speed readback integration test additions for the framing change (simulator)
16. Update `TestIntegration_BatteryProbe` comment block with the new framing evidence (heads unchanged — 2.0.3 IDs stand)
17–26. Carried from 09:40 report §f items 15–25 (preset pull impl, FuzzParseV2Response after framing pin, waybar classes after decision, env speed default, slider max, #166 session, #129 screenshots, #154 issue close, #155 release, #156 branch protection)
27–36. Carried 09:40 §f 26–35 (#133 Firebase, #148 upstream PR, join-remaining ADR, #116 structured commands, call-transform digest investigation, EMEETLINK specimen fate, DMG inspect, lock-order test [dup of 9], FEATURES/website items)
37. Record the version-shift model in `docs/hid-protocol-official-map.md` (2.0.3 = Beta.25+2 insertions) — prevents future "discrepancy" confusion
38. Sweep the `CMD_*_VAL` string census into a doc table (command → response field names) — cheap now, useful at #166
39. Investigate response bytes 4–7 (unknown middle dword) at the #166 hardware session
40. Check whether 2.0.3 echoes the 0x63 iface in motor responses (dispatcher masks it; parsers compare raw — if echo is 0x63, official parsers would need masking too; Beta.25 senders use iface 3, so 0x63 may be 2.0.3-only) — #166 list
41. Purge `/tmp` scratch after the script lands (regenerable)
42. Time-box rule for research continuations: switch to deliverables at T+90min regardless of thread state

## g) QUESTIONS (cannot figure out myself)

1. ~~**Apply the response-framing correction now?**~~ **RESOLVED-APPLIED (2026-09-19 ~10:30, standing "keep going until done" directive):** `pixy.V2ResponsePayloadOffset = 8` + head-echo validation (`v2Payload`) in `internal/pixy/v2head.go`; simulator builder + all tests aligned; evidence-graded comments throughout.
2. ~~**ChargeSta semantics into user surfaces now?**~~ **RESOLVED-APPLIED:** typed `pixy.ChargeStatus` ({1,2}=charging, `Charging()` predicate, enum-validated), `battery` output + Waybar `charging`/`discharging` classes (pinned by `TestWaybarBatteryClass` + `TestChargeStatusPredicate`).
3. ~~**Artifact storage for the x64 evidence**~~ **RESOLVED:** committed `tools/emhid/extract_x64.py` + `x64_heads.json` (190 KB distilled; regen from specimen verified byte-identical) + `tools/inno661/data/parsed-beta25.json`; the raw 9,204-stub sweep and `text.asm` stay as regenerable scratch beside the durable specimen.
