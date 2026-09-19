# emeet-pixyd — TODO List

**Updated:** 2026-09-19 (docs-health sweep: the 18 ✅ DONE rows pruned — they live in `CHANGELOG.md` [Unreleased]; #140's table corruption repaired; the M27 hardware-verification bundle consolidated as #166)

**Previous:** 2026-09-19 (Pareto plan executed: M1–M26 + M28-build done; 27 rows carried dated `→ Resolution` notes, pruned by this sweep)

> Completed work lives in `CHANGELOG.md` — it does NOT live here. Long-term ideas, design-heavy items, "decided won't-do" decisions, and open questions live in `ROADMAP.md`. This file is **open work only**.

---

## Status Legend

- 🔶 PARTIAL — Started but incomplete
- 🚫 BLOCKED — Waiting on an external unblock (permission, decision, upstream, hardware)

---

## Blocked (highest impact — needs an external unblock)

| #   | Status     | Task                                                                                                                                                                                                                                                                                              | Impact | Effort | Evidence                                                                                                    |
| --- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------- |
| 129 | 🚫 BLOCKED | **Retake web UI screenshots with the PIXY connected** (online state: live MJPEG preview, tracking active) — current shipped screenshots show the offline UI. Regenerate `webui-panel.png` crop + video poster from the online shots. Folds into the #166 hardware session                            | MED    | S      | `2026-08-17_18-53` §a.4/§e.3 — camera was disconnected during capture; files in `website/public/screenshots/` |
| 133 | 🔶 PARTIAL | **Website CI/CD deploy half**: `FIREBASE_SERVICE_ACCOUNT` secret (only Lars can create the SA + set it). Build half landed 2026-09-17; the deploy job is wired and self-skipping until the secret exists — `FIREBASE_DEPLOY_SETUP.md` is Lars's 5-minute checklist                                  | MED    | S      | `.github/workflows/website.yml`; `FIREBASE_DEPLOY_SETUP.md`                                                  |
| 154 | 🚫 BLOCKED | **Close issue #6** with the closing formula once @zutto confirms the PIXY 2K works on real hardware (follow-up comment `issues/6#issuecomment-5716075020` already posted; release ref v0.4.0 available)                                                                                            | HIGH   | S      | `2026-09-17_17-34` c1/f2                                                                                     |
| 155 | 🚫 BLOCKED | **Cut v0.4.1** — Lars's cadence call vs accumulating toward v0.5. The release now carries the post-tag fixes (model-aware output, permission warning, vmTest fix) **plus the five new HID command families** (speed, tracking variants, battery, preset push, identity queries)                     | MED    | S      | `CHANGELOG.md` [Unreleased]                                                                                  |
| 156 | 🚫 BLOCKED | **Branch protection requiring the three workflows** (go-test, nix, website) on master — GitHub settings only Lars can change; makes the ungated-auto-commit failure class impossible to merge                                                                                                      | MED    | S      | `2026-09-17_17-34` f16                                                                                       |
| 166 | 🚫 BLOCKED | **M27 hardware-verification bundle** (PIXY attached): one session closes ~6 open threads — run `TestIntegration_BatteryProbe` (battery verdict → #139); pin V2 response framing + `MotorType`/`TargetTrackMode` enum values + motor iface byte (→ #150); pin speed unit + hardware limit (→ #138); preset slot-count sweep (→ #141); hardware-verify tracking variants; retake online screenshots (#129) | **HIGH** | M   | `2026-09-19_06-44` §b/§f 1–7; `TestIntegration_BatteryProbe` in `integration_hardware_test.go`               |

---

## TODO

| #   | Status     | Task                                                                                                                                                                                                                                                                                              | Impact | Effort | Evidence                                                        |
| --- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | --------------------------------------------------------------- |
| 138 | 🔶 PARTIAL | **Finish PTZ speed** — implemented (`speed <pan\|tilt\|zoom> <value>`, 0–10000 sanity bound, unit assumed °/s); remaining: hardware-verify the unit + real limit, then clamp at the command layer and set the slider max (#166); wire speed into PTZ moves + preset recall (the original #138 scope) | MED    | S/M    | `motor.go`, `internal/pixy/v2head.go`; shipped `81955d5`        |
| 139 | 🔶 PARTIAL | **Finish battery/charge surface** — implemented (`battery` command + status/Waybar/web lines, 60s TTL cache, graceful absence); remaining: hardware verdict — the wired PIXY answering battery queries is UNVERIFIED (#166); then polling design (background refresh vs on-demand TTL) and Waybar charging/discharging classes once `ChargeSta` is known | MED    | S/M    | shipped `1a6cb3e`; `tools/emhid/cmdtable.json`                  |
| 140 | 🔶 PARTIAL | **Finish tracking variants** — implemented (`tracking <face\|halfbody\|fullbody>` + web segmented picker, in-memory only); remaining: hardware-verify the enum values (UI-order assumption) (#166); show the variant state in `webStatus` after a daemon restart so the picker doesn't lie            | MED    | S      | shipped `e60d2ff`; simulator asserts modes 0/1/2                |
| 141 | 🔶 PARTIAL | **Finish preset push** — implemented (`preset push <name>`: SetMotorPos ×3 + SetMotorPresetPos, alphabetical slot mapping capped at assumed 8 slots); remaining: slot-count sweep via `GetMotorPresetPosMode` + per-slot semantics (#166); design `preset pull` (hardware → state); add a web confirmation prompt (the command moves the physical camera) | LOW/MED | M     | shipped `db4943e`                                               |
| 148 | 🔶 PARTIAL | **innoextract upstream PR** — `tools/inno661/UPSTREAM.md` staged (issue text with the characterized 6.6.1 deltas, PR outline against `setup/`, pre-send checklist); remaining: sending is Lars's call                                                                                              | MED    | S      | `tools/inno661/UPSTREAM.md`                                     |
| 150 | 🔶 PARTIAL | **Decode remaining enum values + response framing** — `MotorType`, `TargetTrackMode`, `DefaultPosMode`, `ChargeSta` are encoded as documented assumptions in `internal/pixy/v2head.go`; response framing documented as head-echo + payload; static decode from disasm is BLOCKED (raw binaries died with `/tmp`, no committed download URL — re-download unblocks, see #152) — the #166 session pins both from live hardware | HIGH   | M      | map doc §3.5; `internal/pixy/v2head.go`                         |
| 152 | 🚫 BLOCKED | **Cross-verify cmdtable against the Windows x86_64 slice** — second-source the 162 command IDs; blocked on re-obtaining an installer specimen (died with the `/tmp` reboot; re-download + a durable-storage decision is pending, see `ROADMAP.md` open questions)                                   | MED    | S/M    | `tools/emhid/extract_cmdtable.py` (needs x86_64 adaptation)     |

---

All completed work lives in `CHANGELOG.md` ([Unreleased] carries the 2026-09-17 → 09-19 work). Design decisions and open questions live in `ROADMAP.md` — #116 (structured command types) and #123 (multi-word preset names) now have recommendation ADRs (`docs/adr/2026-09-18_structured-command-types.md`, `docs/adr/2026-09-18_multi-word-preset-names.md`) awaiting Lars's decision.

> Full ranked backlog with rationale: `docs/status/2026-09-19_06-44_pareto-plan-execution-full-session.md` §f — items not covered above are ROADMAP-grade research or micro-hygiene.
