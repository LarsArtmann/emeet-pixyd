# emeet-pixyd — TODO List

**Updated:** 2026-11-05 (docs-health AUDIT sweep: all DONE rows pruned — completed work lives in `CHANGELOG.md`; every open item from the annotated status reports verified against code and harvested; new rows #196–#201; #186 deleted — its vmTest default-port pin was verified green 2026-10-31 but the row never flipped)

**Previous:** 2026-10-31 (Pareto plan `docs/planning/2026-10-31_14-33_SUPERB-v05-release-first-pareto-plan.md` fully executed: #184–#188, #173, #174, #179, #181-chunk-2, #192–#195 DONE, v0.5.0 cut — those rows now live in `CHANGELOG.md`)

**Previous:** 2026-10-31 (arch review `2026-10-31_14-17` harvested #190–#193)

**Previous:** 2026-10-20 (Pareto plan P1/P2/P5/P6/P8 executed + CI unblocked; #177/#180/#181-chunk-1/#178-baseline resolved in-session)

**Previous:** 2026-10-01b (state/split-brain remediation: camera intent decoupled from connectivity (ADR `2026-10-01_camera-intent-vs-connectivity`, schema v2), `syncState` under `hidMu`, `powerStatus` negative cache, ghost `ProcessInspector` retired, metrics `online` gauge, lock-contract table — see `CHANGELOG.md` [0.5.0])

**Previous:** 2026-10-01 (PIXY attached — first wired session: privacy-trap bug reproduced, root-caused optically, and FIXED; BatteryProbe run: battery/charge/target-track/func/ver/device-ver heads time out, motor-speed/pos + device-mode + SN answer 32-byte echo frames, motor echo rewrites iface→0x63, V1 `09 01 01 01` answers constant 0x02 (0x03 in the trap) — full #166 checklist still open)

**Previous:** 2026-09-19 (shipped #167 pull early-abort, #171 `preset pull --dry-run`, #169 collision property test, #168 simulator knobs + speed-query duality, #170 dprint vendored into the devShell — details in `CHANGELOG.md` [0.5.0]; #172 evaluated: trigger NOT met — still 3 production GET families, the 4th is speed readback and lands with #138/#166)

> Completed work lives in `CHANGELOG.md` — it does NOT live here. Long-term ideas, design-heavy items, "decided won't-do" decisions, and open questions live in `ROADMAP.md`. This file is **open work only**: each row is the remainder, not the history.

---

## Status Legend

- ◻ OPEN — bounded, ready to implement; nothing shipped yet
- 🔶 PARTIAL — core shipped (see CHANGELOG); remainder named in the row
- 🚫 BLOCKED — waiting on an external unblock (hardware, decision, secret, person)

---

## Blocked (highest impact — needs an external unblock)

| #   | Status     | Task                                                                                              | Impact | Effort | Evidence                                                       |
| --- | ---------- | ------------------------------------------------------------------------------------------------- | ------ | ------ | -------------------------------------------------------------- |
| 129 | 🚫 BLOCKED | Retake online web UI screenshots (live MJPEG, tracking active) + panel crop + video poster        | MED    | S      | `website/public/screenshots/`; report `2026-08-17_18-53` §a.4. UPDATE 2026-09-19: offline-state shots re-captured (`webui-panel.png` 1440×1600, `webui-viewport.png` 1440×900) and deployed; only the ONLINE (PIXY-attached) retake remains, bundle with #166 hardware session |
| 154 | 🚫 BLOCKED | Close issue #6 once @zutto confirms the PIXY 2K on real hardware                                  | HIGH   | S      | report `2026-09-17_17-34` c1/f2                                |
| 156 | 🚫 BLOCKED | Branch protection: require go-test/nix/website workflows on master (GitHub settings, Lars only)   | MED    | S      | report `2026-09-17_17-34` f16                                  |
| 166 | 🚫 BLOCKED | Hardware-verification bundle — one wired session closes #138/#139/#140/#141/#150 + #129           | HIGH   | M      | checklist below; `integration_hardware_test.go`                |

### #166 bundle checklist (run in one wired session)

- Run `TestIntegration_BatteryProbe` — battery/charge verdict → #139 — **PARTIAL 2026-10-01: dev-0x00 heads (battery/charge) and target-track/func/ver/devver all time out; motor-speed/pos, device-mode, SN answer echo frames with mostly-zero payloads**
- Pin `MotorType` + `DefaultPosMode` enum values, motor iface echo (0x63 vs 0x03), response bytes 4..7 → #150 — **PARTIAL 2026-10-01: motor-pos echo rewrites the queried 0x03 iface to 0x63 on the wire; response [4..7] = `00 0X 00 0X` (NOT the assumed zero-reserved dword)**
- Pin speed unit + real hardware limit → #138 clamp at command layer + web slider max — **PARTIAL 2026-10-01: GET_MOTOR_SPEED answers payload `0x20` (=32) — unit still unknown**
- Decode the V1 mode-query semantics: `09 01 01 01` (= CMD_SET_DEVICE_MODE head, we SendRecv it as a GET) answers 0x02 constantly, 0x03 in the privacy-trap state — do NOT trust `sync`/reconcile mode reads until decoded
- Decide panel honesty for the transient trap cover (lens covered while believed=tracking at tilt ≤ −85): re-assert fix hides the window, but a slow far-travel + fast click could still show Track with a black feed
- Preset slot-count sweep + response-shape pin (mode-only GET vs full SET-echo) → #141
- Live round-trip: `preset push` → `preset pull` → values match
- Optional (Lars's Q3): Windows usbmon capture of the official app to settle `MotorType`
- Retake online screenshots → #129

---

## TODO

| #   | Status     | Task                                                                                         | Impact | Effort | Evidence                                                  |
| --- | ---------- | -------------------------------------------------------------------------------------------- | ------ | ------ | --------------------------------------------------------- |
| 138 | 🔶 PARTIAL | Hardware-verify speed unit + limit; then clamp at command layer + set web slider max         | MED    | S      | `motor.go`; shipped `81955d5`                             |
| 139 | 🔶 PARTIAL | Hardware verdict: does the wired PIXY answer the battery/charge heads at all?                | MED    | S      | `power.go`; ADR `2026-09-19_battery-polling-on-demand-ttl` |
| 140 | 🔶 PARTIAL | Hardware-confirm tracking variants; decide whether reconcile re-asserts variant after power cycles | MED | S   | `internal/pixy/v2head.go`; `TestTargetTrackModeWireValues` |
| 141 | 🔶 PARTIAL | Hardware: slot count + response shape; if mode-only, SET-echo acquisition needs Lars's Q1 call | LOW/MED | S   | `commands.go`, `motor.go`; shipped `db4943e`              |
| 148 | 🔶 PARTIAL | Send the staged innoextract 6.6.1 upstream PR (Lars's call)                                  | MED    | S      | `tools/inno661/UPSTREAM.md`                               |
| 150 | 🔶 PARTIAL | Pin `MotorType` + `DefaultPosMode` values (static decode exhausted; hardware/usbmon only)    | HIGH   | M      | `internal/pixy/v2head.go`; map doc §3.5a                  |
| 172 | ◻ OPEN     | Extract a shared V2 query helper (trigger: 4th GET family lands — evaluated 2026-09-19: NOT met, identity/power/preset are 3; 4th = speed readback, gated on #138/#166) | LOW | M | `identity.go`, `power.go`, `motor.go`                     |
| 176 | 🔶 PARTIAL (config wired, INERT until NixOS switch) | Gate the auto-commit daemon on a green pre-commit build (kills the red-at-HEAD class: bad commits land mid-session and every later session pays the diagnosis tax) | HIGH | M | Config landed in SystemNix (`7ca1bd708` doc + `d0862d746` gate config: hold-not-discard journal WARN, GOWORK=off, go_1_27 mkBefore toolchain trap documented). NOT ACTIVE on the host — needs the SystemNix switch (Lars Q1). Note: the gate cannot prevent heuristic file sweeps anyway (observed 6+ pma sweeps 2026-10-31 — stage+commit-in-one-call remains the working countermeasure) |
| 178 | ◻ OPEN     | Benchmark delta: confirm `ttlCache[T]` + preset double-RLock are perf-neutral post-refactor | LOW | S | Baseline stashed 2026-10-20: `docs/benchmarks/baseline-2026-10-20.txt` (9 benchmarks × 5 runs, benchstat-able). Compare the next refactor against it (`go test -run XXX -bench . -benchtime 1s -count 5`) |
| 182 | 🚫 BLOCKED | Make `sync` reconcile `Speeds`/`TrackMode` (re-read hardware) — blocked on #166 pinning the readback: the motor-speed GET payload unit is unknown and the mode query is unreliable, so a readback would risk clobbering good intent | LOW | S | `device.go` `syncStateLocked` (M13); ADR `2026-10-01_camera-intent-vs-connectivity` |
| 183 | 🔶 PARTIAL (wired, deploy-pending) | SystemNix wrapper: wire `hardware.emeet-pixy.port = ports.emeet-pixyd` + bump this repo's lock there + runbook note — cross-repo, needs Lars's go-ahead | HIGH | S | Wiring landed locally in SystemNix (`d0862d746` port wiring, `7ca1bd708` runbook note; upstream option exists since v0.5.0 — `version` field proof). NOT pushed, NOT switched: input still locked pre-tag; float-vs-pin is Lars Q2. No action available in this repo |
| 195 | ◻ OPEN | Website dependabot sweep: 12 open npm alerts (5 high: vite, undici, source-map-js, sharp, http-cache-semantics; 7 moderate: vite ×3, undici, smol-toml, fast-uri) — build-chain only (static output ships none of it; Go side clean via CI govulncheck). `pnpm audit fix` + lockfile regen + `--frozen-lockfile`/`tsc --strict` proof (pairs with the #185 proof, CHANGELOG). First rebase the failing `npm_and_yarn` dependabot branches — they predate the v2.14.0 lint-pin fix | MED | S | GitHub dependabot, noticed 2026-10-31 push; report `2026-09-19_13-45` f24 |
| 196 | ◻ OPEN | saveState write-side intent guard: `loadState` normalizes a legacy `offline` on read, but `saveState` has NO `ValidDesired()` guard on write — decide refuse-vs-normalize, then close the write path | MED | S | `state.go` `saveState`; audit `2026-10-01_04-21` f12; retro `2026-10-01_11-37` §b1/g1 |
| 197 | ◻ OPEN | Contract pins for `controllable`: JSON-shape test assertion, Waybar video-present/HID-absent golden, `/api/status controllable:false` contract test, CLI-collapse pin, panel `!Controllable&&Online` notice test — the guide field-table row already landed 2026-11-05 | MED | S | `web_status_test.go`; retro `2026-10-01_11-37` c3/f6–f10 |
| 198 | ◻ OPEN | Pin `errors.Is` propagation through `errorfamily.Compose` trees with a unit test (Compose is live at real call sites; only single-wrap chains are pinned today) | LOW | S | `errorfamily.go`, `auto.go`; erraudit triage `2026-09-29_01-36` b3/f5 |
| 199 | ◻ OPEN | Website CI/secret hardening bundle: pin `firebase-tools` (currently `@latest`), `concurrency` group, npx cache, post-deploy smoke test + release URL in the job summary, derive the page-count assert from source, `actionlint`/`yamllint`, `html-validate` + `tsc --noEmit` steps in CI, SA key-ID + rotation cadence in `FIREBASE_DEPLOY_SETUP.md`, rollback runbook section | MED | M | `website.yml`; report `2026-09-19_13-45` §f items 74–130 |
| 200 | ◻ OPEN | Execute the accessibility + real-device checklists (NVDA/VoiceOver/Orca screen readers; phone/iPad/landscape); flip the two 🟡 FEATURES rows | LOW | M | `docs/accessibility-audit.md`; FEATURES 🟡 rows; plan `2026-09-28_23-53` M18 |
| 201 | ◻ OPEN | vmTest watchdog-stall pinning test: wedge a fake hung `v4l2-ctl` under `hidMu`+`v4l2Mu`, assert the loop-iteration bound stays ≪ `WatchdogSec` (pairs with the shipped subprocess timeouts) | LOW | M | `flake.nix` vmTest; arch review `2026-10-31_14-30` c4; report `2026-11-01` f21 |

---

All completed work lives in `CHANGELOG.md` (`[Unreleased]` carries the post-v0.5.0 work; `[0.5.0]` carries the release). Design decisions and open questions live in `ROADMAP.md` — #116 (structured command types), #123 (multi-word preset names), the NixOS `extraProductIds` option, and hint surfacing (#173, shipped) have recommendation ADRs (`docs/adr/2026-09-18_*.md`, `docs/adr/2026-09-28_*.md`) awaiting Lars's decision.

> Historical ranked backlogs (e.g. `docs/status/archive/2026-09-19_14-00_harvest-and-fix-on-sight-sweep.md` §f) are fully harvested into this file + ROADMAP; the reports themselves are annotated and archived.
