# emeet-pixyd — TODO List

**Updated:** 2026-09-19 (shipped #167 pull early-abort, #171 `preset pull --dry-run`, #169 collision property test, #168 simulator knobs + speed-query duality, #170 dprint vendored into the devShell — details in `CHANGELOG.md` [Unreleased]; #172 evaluated: trigger NOT met — still 3 production GET families, the 4th is speed readback and lands with #138/#166; no PIXY on the bus this session, hardware rows untouched)

**Previous:** 2026-09-19 (rows shrunk to one-line open work for terminal readability — every shipped half already lives in `CHANGELOG.md` [Unreleased]; #166's thread list moved to a checklist under the table; new rows #167–#172 relabeled 🔶 PARTIAL → ◻ OPEN, they have no shipped code)

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
| 155 | 🚫 BLOCKED | Cut v0.4.1 (Lars's cadence call vs v0.5)                                                          | MED    | S      | `CHANGELOG.md` [Unreleased]                                    |
| 156 | 🚫 BLOCKED | Branch protection: require go-test/nix/website workflows on master (GitHub settings, Lars only)   | MED    | S      | report `2026-09-17_17-34` f16                                  |
| 166 | 🚫 BLOCKED | Hardware-verification bundle — one wired session closes #138/#139/#140/#141/#150 + #129           | HIGH   | M      | checklist below; `integration_hardware_test.go`                |

### #166 bundle checklist (run in one wired session)

- Run `TestIntegration_BatteryProbe` — battery/charge verdict → #139
- Pin `MotorType` + `DefaultPosMode` enum values, motor iface echo (0x63 vs 0x03), response bytes 4..7 → #150
- Pin speed unit + real hardware limit → #138 clamp at command layer + web slider max
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

---

All completed work lives in `CHANGELOG.md` ([Unreleased] carries the 2026-09-17 → 09-19 work). Design decisions and open questions live in `ROADMAP.md` — #116 (structured command types) and #123 (multi-word preset names) have recommendation ADRs (`docs/adr/2026-09-18_*.md`) awaiting Lars's decision.

> Full ranked backlog with rationale: `docs/status/2026-09-19_14-00_harvest-and-fix-on-sight-sweep.md` §f — older ranked backlogs are harvested into this file; items not covered above are ROADMAP-grade research or micro-hygiene.
