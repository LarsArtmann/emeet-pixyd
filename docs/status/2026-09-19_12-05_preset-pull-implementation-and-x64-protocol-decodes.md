# Status Report — 2026-09-19 12:05 CEST — Preset Pull Implementation & x64 Protocol Decodes

**Session scope:** execute the open TODO list. Code work landed: `preset pull`
(TODO #141's open half) end-to-end. Research work landed: Beta.25 x64
send/parse-site decodes that corrected three documented protocol assumptions.
Everything verified: `go test -race ./...` exit 0, golangci-lint 0 issues,
`go vet` clean, `templ generate` idempotent, `nix build` green.

**Format note:** the status-report skill defaults to a styled HTML dashboard;
the user explicitly requested `.md` at this path, so this file is Markdown.

---

## a) FULLY DONE

1. **`preset pull` command, full stack.** Read-only sweep of hardware motor
   slots 1..8 over the official V2 `GetMotorPresetPosMode` (`09 63 01 17` +
   slot byte). Full-shape answers land as additive `hw-<slot>` presets
   (float→int rounding, V4L2 clamping); mode-only answers count as "set
   (position not exposed)"; never overwrites existing names; unreachable
   device aborts; per-slot failures degrade the summary; full map stops the
   sweep without eviction. Evidence: `commands.go:556` `handlePresetPull`,
   17 new tests in `preset_pull_test.go`, full suite green.
2. **Web surface for pull.** `POST /api/preset/pull` (`handlers.go`) + Pull
   button in the preset header (`templates.templ`, `static/style.css`); no
   confirm gate because the sweep is read-only. Pinned by
   `TestWebPresetPull_RendersHeaderButton`.
3. **V2 response parser for preset slots (`internal/pixy/v2head.go`).**
   `MotorPresetReading` + `ParseMotorPresetPosResponse` accepting both
   statically evidenced shapes: mode-only (1 payload byte, the Beta.25 GET
   parser's shape) and full SET-echo (`[slot][mode][pan f32][tilt f32][zoom
   f32]`, floats gated on mode==1, min 0x16). `Occupied()`/`HasPosition()`/
   `PTZValues()` predicates.
4. **Echo routing mask implemented (`v2EchoMatches`).** Every official
   parser masks `resp[1] & 0x1F` before validating the echo; ours now do too,
   so a device echoing the logical motor dev byte (0x03) instead of the
   routed 0x63 still validates. Pinned by
   `TestParseMotorPresetPosResponse_RoutingMaskEcho`.
5. **Transport generalization (`power.go`).** `v2Read` split into
   `v2ReadLocked(ctx, head, payload)` so payload-carrying GET queries work
   under a caller-held `hidMu` (battery path behavior unchanged).
   `motor.go:176` `queryMotorPresetPos` rides it.
6. **Simulator fidelity (`pixy_simulator_test.go`).** `pixyProtocolState` now
   models preset slots (`SetMotorPresetPos` snapshots the live position;
   `SetMotorPresetPosMode` flips the mode byte) and returns V2 GET responses
   at their **exact payload length** (real hidraw short-reads) instead of
   padded 64-byte buffers. `presetFullResponses` knob flips between the two
   evidenced shapes.
7. **Static protocol decodes from the Beta.25 x64 disassembly (map doc
   §3.5a updated):**
   - Preset GET is **mode-only** (parser thunk `0x14017e5d0` → shared
     single-byte parser `0x140179640`); positions ride the
     `SET_MOTOR_PRESET_POS_MODE` echo (parser `0x14017e330`: min 10/0x16,
     floats at 0xa/0xe/0x12 gated on byte@9==1).
   - Motor-speed query **rides the SET head** (`09 63 01 03` + motorType;
     send site `0x14017ecad`), response `[motorType][speed f32][limit f32]`
     min 0x11 (parser `0x14017e430`) — matches our existing
     `MotorSpeedReading` shape.
   - The `0x63` routing is **confirmed for Beta.25**: send site computes
     `mergeType(3,3)` via the literal `(dev<<5)|func` helper `0x140179c30`.
   - The **version-shift model is disproved/retracted**: all 108 x64-confirmed
     heads (incl. whole power + motor families) match the Mac table by exact
     tuple; docs and `extract_x64.py` docstring corrected.
8. **TODO #152 closed.** Exhaustive scans for inline head construction in the
   x64 build (dword-immediate stores, byte-wise adjacent stores,
   rip-relative stores) found **zero** additional sites → the 53 Mac-only
   heads are absent from the Windows build (consistent with the §5
   subset-linking finding). Cross-verification is complete: every head the
   x64 slice contains agrees with the Mac table. Row removed from
   TODO_LIST; resolution recorded in CHANGELOG.
9. **Documentation sweep.** CHANGELOG [Unreleased] (pull + decodes + #152
   closure), TODO_LIST (header, #141/#150/#166 refreshed, #152 removed),
   ROADMAP (pull design marked implemented + evidence-corrected; usbmon item
   largely resolved), FEATURES (Motor-Preset Pull row), AGENTS.md (V2
   evidence grades, presets bullet, simulator exact-length note),
   `docs/hid-protocol-official-map.md` (§3.5 routing + shift retraction,
   §3.5a new decodes, cross-verification completion, MotorType thread
   narrowed — the old `0x1403c53c0` pointer is a Qt copy helper), website
   `changelog.mdx`, `tools/emhid/extract_x64.py` docstring.
10. **`main.go` CLI help** — `preset pull` line added alongside push.

## b) PARTIALLY DONE

1. **TODO #141 (preset push/pull).** Works: both directions implemented,
   tested, wired to CLI + web. Remains: the #166 hardware session must pin
   the real slot count and which response shape the wired firmware answers;
   if it answers mode-only, position acquisition needs a product decision
   (see questions). Effort to finish: S once hardware is attached.
2. **TODO #150 (enum/framing decodes).** Now covers framing, routing mask,
   ChargeSta, TargetTrackMode, per-command min lengths for the
   preset/default-pos/speed families. Remains: `MotorType` values (send site
   passes the byte through from vtable-indirect callers — static decode
   exhausted, needs live tracing) and `DefaultPosMode` value semantics.
   Blocker: hardware. Effort: M (inside #166).
3. **TODO #138/#139/#140 (speed, battery, tracking).** Code complete and
   persisted; the _only_ remainder of each is hardware verification inside
   the #166 bundle. Effort: S each once attached.
4. **TODO #133 (website deploy).** Build half shipped; deploy job
   self-skips until Lars creates `FIREBASE_SERVICE_ACCOUNT`. Note: this
   session edited `website/src/content/docs/changelog.mdx` but did **not**
   run the Astro build to verify the MDX — should be built before the next
   deploy. Effort: S.
5. **TODO #129 (online screenshots).** The new Pull button changes the panel
   the screenshots would capture; retake folds into the #166 hardware
   session as planned. Not started (hardware).

## c) NOT STARTED

1. **v0.4.1 release** (#155) — blocked on Lars's cadence call; the release
   now additionally carries preset pull + the protocol corrections.
2. **Issue #6 closure** (#154) — waiting on @zutto's hardware confirmation.
3. **Branch protection** (#156) — GitHub settings, Lars-only.
4. **innoextract upstream PR** (#148) — staged in `tools/inno661/UPSTREAM.md`;
   sending is Lars's call.
5. **ROADMAP research offshoots** (elink protocol doc, EMVideoInput.dll,
   hidCmdSend retry semantics, GET_FUNC_STA bitfield UI, `FuzzParseV2Response`,
   structured command types #116 ADR, multi-word preset names #123 ADR,
   `EMEET_PIXYD_MOTOR_SPEED` env default) — all deliberately unscheduled;
   the two ADRs still await Lars's decision.
6. **Demo video / screenshots refresh** for the new Pull button — folds
   into #129's retake session.

## d) TOTALLY FUCKED UP

Nothing shipped broken — final gates were all green. But these were real
defects I wrote during this session; all caught and fixed **before** they
could land, listed with root causes because the pattern repeats:

1. **Deadlock in the first `handlePresetPull` draft** — a stray `d.mu.Lock()`
   at the loop tail (acquire without release before the next iteration's
   acquire) would have deadlocked every pull. Caught by self-review seconds
   after writing, pre-test. Root cause: composing lock placement while
   restructuring mid-edit.
2. **Wrong response-shape attribution — I coded the parser against the wrong
   decoded shape.** I implemented the ROADMAP's claimed GET shape (mode@8 +
   floats at 9/0xd/0x11, min 0x15) which actually belongs to the
   power-on-default SET-echo parser (`0x14017e210`, head `…14`); the real
   preset GET parser is single-byte. Root cause: I trusted the doc's
   attribution instead of verifying which echo head each parser compares
   against _before_ coding. Fixed by disassembling all three parsers and
   reworking to a both-shapes parser; docs corrected. Severity had it
   shipped: pull would have misread every slot on hardware.
3. **Simulator padded responses broke shape detection** — my parser
   distinguishes shapes by payload length, but the simulator returned
   64-byte padded buffers, so a mode-only answer parsed as
   "full shape, mode=0" (two test failures). Root cause: I changed the
   parser's detection basis without checking the test double's response
   realism. Fixed by making the simulator return exact-length responses
   (strictly better fidelity).
4. **Test-file placeholder garbage** — the first `preset_pull_test.go` write
   contained a nonsense `d.mu := 0` line and an unused-import hack. Caught
   by re-reading my own diff before running.
5. **Pipeline masking, twice** — `go test … | grep -v ok` / `| tail` made
   the pipeline's exit code grep's/tail's, not go test's; I initially read
   one green-looking result that was actually unverified and one spurious
   EXIT:1 that was just "no lines matched". My own AGENTS.md has a lesson
   on exactly this; I re-verified both with direct exit codes.
6. **Lint round-tripping waste** — 4 fix rounds (exhaustruct → golines →
   wsl → golines), including a `//nolint` comment that itself tripped
   golines for line length. Should have written the full struct literal
   immediately; each round is a full-module lint run (~1 min).
7. **Edit/external-modify race** — two multiedits failed with "file modified
   since read" (auto-daemon reformatting raced me); one retry batch then
   applied against content I had not re-read, and I only noticed via a
   later grep. Recovered, but the sequence was sloppy.

## e) WHAT WE SHOULD IMPROVE

1. **Verify attribution, not just content, of evidence notes.** Miss #2
   above cost a rework: the ROADMAP said _what_ the response shape was but
   attributed it to the wrong command. Standing rule for future protocol
   work: every decoded shape must cite its parser address **and** the echo
   head it validates against, in the doc, at decode time.
2. **Forgot the domain glossary.** I updated six docs but not
   `docs/DOMAIN_LANGUAGE.md` — `preset pull`/`hw-N` were missing and the
   tracking-variant row still claimed "in-memory only" (wrong since #140).
   Found only because the report forced a doc audit. Fixed this session,
   but the doc checklist should include DOMAIN_LANGUAGE.md explicitly.
3. **Format-then-verify ordering.** I edited `changelog.mdx` but did not run
   the Astro build; MDX is stricter than Markdown and a bad build fails CI
   later. Rule: website edits end with `pnpm build`.
4. **Test-double realism should be designed, not discovered.** The padded-
   response bug surfaced only because a test asserted the new parser's
   contract. Cheaper: when a parser starts using a byte stream property
   (like length), first audit what the simulator guarantees.
5. **Exit-code hygiene in my own verification loops.** Despite the existing
   AGENTS.md lesson I still built masked pipelines twice. Personal default
   should be `cmd > file 2>&1; echo EXIT=$?` for every gate, no in-line
   filters.
6. **Lock-placement review before test-running.** The deadlock draft would
   have been caught by the watchdog test anyway, but a 10-second
   re-read of freshly written lock code is cheaper than a race-test cycle.
7. **All-slots-failed pull error omits the failure count** (returns just the
   first wrapped cause). Small UX gap; should read "8/8 slots unreadable: …".
8. **Sweep latency on a half-dead device** — 8 queries × 500 ms timeout can
   cost ~4 s with no progress feedback. An early-abort heuristic
   (N consecutive failures → stop) is unimplemented.
9. **`presetFullResponses` is a raw struct field** poked by tests
   (`sim.state.presetFullResponses = true`) rather than a builder option;
   fine for now, will not scale as knobs accumulate.
10. **Auto-commit daemon wrote heuristic history for this session** (many
    "chore: auto-commit N changed file(s)"). Not a violation (commits were
    not authorized), but per-task commits would make the preset-pull change
    bisectable; sessions that want that should ask for explicit commits.
11. **dprint is configured (`dprint.json`) but absent from the devShell**, so
    markdown formatting has no local runner; TODO_LIST/ROADMAP tables are
    now hand-aligned. Either vendor dprint into the shell or drop the config.

## f) Top 50 things to get done next

Ranked by impact. "Routed" = already lives in TODO_LIST/ROADMAP; "HARVEST" =
new from this report, needs docs-health HARVEST to route.

| #  | Task                                                                                                       | Impact | Effort | Category      | Route                 |
| -- | ---------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- | --------------------- |
| 1  | Wire the PIXY and run the #166 hardware-verification bundle (one session closes items 2–12 below)          | High   | M      | Verification  | Routed #166           |
| 2  | Run `TestIntegration_BatteryProbe` on wired hardware (battery verdict)                                     | High   | S      | Verification  | Routed #139           |
| 3  | Pin real preset slot count via a live `preset pull` sweep                                                  | High   | S      | Verification  | Routed #166           |
| 4  | Pin which preset-response shape the wired firmware answers (mode-only vs full SET-echo)                    | High   | S      | Verification  | Routed #166           |
| 5  | If firmware answers mode-only: verify position acquisition via `SET_MOTOR_PRESET_POS_MODE` echo (needs Q1) | High   | M      | Feature       | HARVEST               |
| 6  | Pin `MotorType` values (pan/tilt/zoom = 0/1/2) with live speed reads                                       | High   | S      | Verification  | Routed #150           |
| 7  | Pin speed unit + hardware limit; clamp at command layer; set web slider max                                | Med    | S      | Feature       | Routed #138           |
| 8  | Hardware-verify TargetTrackMode variants on-wire                                                           | Med    | S      | Verification  | Routed #140           |
| 9  | Decide + implement whether reconcile re-asserts the tracking variant after power cycles                    | Med    | S      | Feature       | Routed #140           |
| 10 | Verify motor iface echo (0x63 vs 0x03) on-wire — validates `v2EchoMatches`                                 | Med    | S      | Verification  | Routed #166           |
| 11 | Decode the reserved dword (response bytes 4..7) from live captures                                         | Low    | S      | Research      | Routed #166           |
| 12 | Live round-trip test: `preset push` → `preset pull` → values match                                         | Med    | S      | Verification  | HARVEST               |
| 13 | Retake online web UI screenshots (live MJPEG, tracking active) + regenerate panel crop and video poster    | Med    | S      | Documentation | Routed #129           |
| 14 | Cut v0.4.1 (5 HID families + preset pull + protocol corrections) — needs Lars's cadence call               | High   | S      | Release       | Routed #155           |
| 15 | Close issue #6 once @zutto confirms PIXY 2K on hardware                                                    | High   | S      | Release       | Routed #154           |
| 16 | Create `FIREBASE_SERVICE_ACCOUNT` secret and enable the website deploy job                                 | Med    | S      | Release       | Routed #133           |
| 17 | Require go-test/nix/website workflows on master (branch protection)                                        | Med    | S      | Quality       | Routed #156           |
| 18 | Build the website (`pnpm build`) to verify this session's `changelog.mdx` edits compile                    | Med    | S      | Quality       | HARVEST               |
| 19 | Add a pre-release local gate: `govulncheck` + fuzz smoke (`go test -fuzz` seeds)                           | Med    | S      | Quality       | HARVEST               |
| 20 | Pull early-abort heuristic: stop sweep after N consecutive slot timeouts to cut worst-case ~4 s latency    | Med    | S      | Feature       | HARVEST               |
| 21 | Include the failure count in pull's all-slots-unreadable error ("8/8 slots unreadable: …")                 | Low    | S      | Feature       | HARVEST               |
| 22 | Web UI: render per-slot pull outcome (occupied/empty/pulled) beyond the toast line                         | Low    | M      | Feature       | HARVEST               |
| 23 | Land the `EMEET_PIXYD_MOTOR_SPEED` env default (speed follow-up)                                           | Med    | S      | Feature       | Routed ROADMAP        |
| 24 | Decide structured command types ADR (#116) — join-remaining for multi-word names rides on it               | Med    | M      | Feature       | Routed, needs Lars    |
| 25 | Decide multi-word preset names ADR (#123) and land join-remaining                                          | Med    | S      | Feature       | Routed, needs Lars    |
| 26 | Add `FuzzParseV2Response` now that framing has dual-shape parsers                                          | Med    | M      | Quality       | Routed ROADMAP        |
| 27 | Switch mode reads to the authoritative `GET_DEVICE_MODE` head or document why the SET-head query stays     | Low    | S      | Cleanup       | Routed ROADMAP        |
| 28 | Decode `GET_FUNC_STA` bitfield into capability-gated UI                                                    | Low    | M      | Feature       | Routed ROADMAP        |
| 29 | Implement `hidCmdSend`-style bounded retry (w4=50) in the HID layer once #166 pins retry semantics         | Low    | M      | Feature       | Routed ROADMAP        |
| 30 | Reconcile the map doc §3.5 payload table's "GETs: bare heads" row with the slot/motorType-byte reality     | Low    | S      | Documentation | HARVEST               |
| 31 | Measure coverage for the new preset-pull paths; close gaps if under suite average                          | Low    | S      | Quality       | HARVEST               |
| 32 | Wrap `presetFullResponses` in a proper simulator option instead of direct field pokes                      | Low    | S      | Cleanup       | HARVEST               |
| 33 | Update `handlePresetWithLock` doc comment: push/pull take hidMu internally, save/load take v4l2Mu          | Low    | S      | Documentation | HARVEST               |
| 34 | Add a property test: pull never evicts or mutates presets under arbitrary name collisions                  | Low    | S      | Quality       | HARVEST               |
| 35 | Model the speed-query duality (SET-head vs `09 03 01 13`) in the simulator for #166 comparisons            | Low    | S      | Quality       | HARVEST               |
| 36 | README: mention hardware preset mirroring (push/pull) on the sales page                                    | Low    | S      | Documentation | HARVEST               |
| 37 | Decide dprint ownership: vendor it into the devShell or delete `dprint.json`                               | Low    | S      | Cleanup       | HARVEST               |
| 38 | Extend auto-management docs/ADR: preset pull stays manual (no auto-sync on device appear)                  | Low    | S      | Documentation | HARVEST               |
| 39 | Optional usbmon capture on Windows to decode MotorType values from caller sites                            | Low    | L      | Research      | HARVEST               |
| 40 | Route Waybar slot-occupancy surface to ROADMAP (won't-do candidate)                                        | Low    | S      | Cleanup       | HARVEST               |
| 41 | Deploy the website after item 18 passes so the public changelog carries preset pull + corrections          | Med    | S      | Release       | HARVEST (needs #16)   |
| 42 | Run docs-health HARVEST on this report's items (per skill handoff rule)                                    | Med    | S      | Documentation | HARVEST               |
| 43 | Run the vmTest suite once before the next release (not exercised this session)                             | Med    | S      | Quality       | HARVEST               |
| 44 | Review stale `//nolint` directives around the touched files (per AGENTS.md lint note)                      | Low    | S      | Cleanup       | HARVEST               |
| 45 | Add `preset pull` to the NixOS module docs page if it enumerates commands                                  | Low    | S      | Documentation | HARVEST               |
| 46 | Consider a `preset pull --dry-run` (report-only sweep) as the safe discovery surface                       | Low    | S      | Feature       | HARVEST               |
| 47 | Tag + verify module proxy after the next release (go-release checklist)                                    | Low    | S      | Release       | HARVEST               |
| 48 | Extract a shared V2 query helper as more GET families land (identity/battery/preset each wrap it)          | Low    | M      | Cleanup       | HARVEST               |
| 49 | Record the session's 0x1403c53c0→Qt-copy-helper correction in ROADMAP research notes (done in map doc)     | Low    | S      | Documentation | HARVEST (verify only) |
| 50 | Plan the next docs-health sweep to prune this report per the annotate-not-rewrite rule                     | Low    | S      | Documentation | HARVEST               |

## g) Three questions I cannot answer myself

1. **Pull position acquisition vs read-only purity.** If the wired firmware
   answers the preset GET mode-only (what the Beta.25 parser reads), the only
   static path to slot positions is sending `SET_MOTOR_PRESET_POS_MODE
   [slot][mode]` and parsing its echo — which **mutates** the slot's mode
   state. May `preset pull` do that (documented, opt-in flag?), or must pull
   stay strictly read-only and positions remain push-sync only? I tried:
   disassembled all preset-family parsers and send sites — no read-only
   position read exists in the Beta.25 surface; this is a product call.
2. **Release cadence.** v0.4.1 now (carries the five HID command families,
   preset pull, and the framing/enum corrections) or accumulate toward v0.5?
   This is TODO #155, flagged as yours; every release-adjacent session
   stalls on it. What I tried: the CHANGELOG [Unreleased] keeps growing —
   the longer it grows, the bigger the release note.
3. **Scope of the #166 wired session.** Is a Windows usbmon capture of the
   official app in scope for that session (it would settle `MotorType`
   values from real sender traffic, the one remaining static-dead-end), or
   should the session stay probe-only (our own integration test + `preset
   pull` sweep), with MotorType staying an assumption until a later effort?

---

_Point-in-time snapshot — goes stale. Section (f) is HARVEST input for
TODO_LIST.md/ROADMAP.md; items 42 + 49–50 track that handoff._
