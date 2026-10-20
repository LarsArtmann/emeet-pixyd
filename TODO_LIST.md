# emeet-pixyd — TODO List

**Updated:** 2026-10-19 (SystemNix extraction follow-ups harvested from plan `docs/planning/2026-10-19_04-29_SUPERB-systemnix-closeout-and-backlog-pareto-plan.md`: new rows #183–#189; b109788 non-authored hunks audited = benign templ-regen churn, see plan errata)

**Previous:** 2026-10-01b (state/split-brain remediation: camera intent decoupled from connectivity (ADR `2026-10-01_camera-intent-vs-connectivity`, schema v2), `syncState` under `hidMu`, `powerStatus` negative cache, ghost `ProcessInspector` retired, metrics `online` gauge, lock-contract table — see `CHANGELOG.md` [Unreleased]; findings 1/2/3/4/5/6/7/10/11 fixed; new row #182)

**Previous:** 2026-10-01 (PIXY attached — first wired session: privacy-trap bug reproduced, root-caused optically, and FIXED (see CHANGELOG [Unreleased]); BatteryProbe run: battery/charge/target-track/func/ver/device-ver heads time out, motor-speed/pos + device-mode + SN answer 32-byte echo frames, motor echo rewrites iface→0x63, V1 `09 01 01 01` answers constant 0x02 (0x03 in the trap) — full #166 checklist still open)

**Previous:** 2026-09-19 (shipped #167 pull early-abort, #171 `preset pull --dry-run`, #169 collision property test, #168 simulator knobs + speed-query duality, #170 dprint vendored into the devShell — details in `CHANGELOG.md` [Unreleased]; #172 evaluated: trigger NOT met — still 3 production GET families, the 4th is speed readback and lands with #138/#166; no PIXY on the bus that session, hardware rows untouched)

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
| 173 | ◻ OPEN     | Surface the unsupported-device hint in the web UI (offline panel) + Waybar tooltip/JSON — gated on the ADR answer | MED | M | `commands.go` `device`/`probe` hint; report `2026-09-28_23-48` f5 |
| 174 | ◻ OPEN     | Branch tests for the hint paths: `probe`/`device` hint output + rate-limited `unsupportedWarnLimiter` log branch | MED | S | `probe.go`; report `2026-09-28_23-48` f7 |
| 175 | ◻ OPEN     | Website SEO pass: per-page description frontmatter audit, sitemap ping after deploy, per-page OG images for docs routes (astro-og-canvas) | LOW | S | `website/astro.config.mjs`; report f8/f32 — verified 2026-09-28: docs pages carry og:title/description but no og:image (landing-only) |
| 176 | ◻ OPEN     | Gate the auto-commit daemon on a green pre-commit build (kills the red-at-HEAD class: bad commits land mid-session and every later session pays the diagnosis tax) | HIGH | M | report `2026-09-28_23-47` f1/d1 |
| 177 | ✅ DONE (2026-10-20) | Scope BuildFlow's ruff step away from `tools/inno661/`+`tools/emhid/` (or ruff-clean them, SIM115 ×20) so the full dev gate goes green for Go work | MED | S | Done via `ruff.toml`: per-file-ignores SIM115+EXE001+F841 for `tools/**` only (frozen RE artifacts, byte-verified outputs — restyling would invalidate verification); ruff stays on everywhere else. BuildFlow ruff-check/fix/format/config steps all green with it in place |
| 178 | ◻ OPEN     | Benchmark delta: confirm `ttlCache[T]` + preset double-RLock are perf-neutral post-refactor | LOW | S | Baseline stashed 2026-10-20: `docs/benchmarks/baseline-2026-10-20.txt` (9 benchmarks × 5 runs, benchstat-able). Compare the next refactor against it (`go test -run XXX -bench . -benchtime 1s -count 5`) |
| 179 | ◻ OPEN     | Decide: always `ValidatePresetName` in the web preset action (400 for garbage names vs today's passthrough to the command layer) | LOW | S | `handlers.go` `presetAction`; report `2026-09-28_23-47` b4 |
| 180 | 🗑 OBSOLETE (2026-10-20) | Align `.crushrc` LSP lint config with `.golangci.yml` (drop golines/wsl_v5 noise the repo does not gate on) | LOW | S | Premise stale: `.crushrc` carries NO lint list (toolchain pins only), and the LSP wrapper is cache plumbing — golangci-lint-langserver runs the repo's `.golangci.yml` directly, which DOES gate `wsl_v5` (line 118) and `golines` (formatter). LSP signal already matches CI; nothing to align |
| 181 | ◻ OPEN     | Mechanical nolint sweep post-refactor (nolintlint- or grep-driven): remove directives whose suppressing rationale rotted | MED | M | Inventory 2026-10-20: 107 directives across 39 files, top = handlers.go 12. Chunk 1 DONE: uevent_test.go's phantom exhaustruct prose removed (linter is test-excluded; the referenced directives never existed). All other test-file directives verified LIVE (paralleltest/dupword/usetesting/bodyclose are not test-excluded) |
| 182 | 🚫 BLOCKED | Make `sync` reconcile `Speeds`/`TrackMode` (re-read hardware) — blocked on #166 pinning the readback: the motor-speed GET payload unit is unknown and the mode query is unreliable, so a readback would risk clobbering good intent | LOW | S | `device.go` `syncStateLocked` (M13); ADR `2026-10-01_camera-intent-vs-connectivity` |
| 183 | ◻ OPEN | SystemNix wrapper: wire `hardware.emeet-pixy.port = ports.emeet-pixyd` + bump this repo's lock there + runbook note — cross-repo, needs Lars's go-ahead | HIGH | S | plan `2026-10-19_04-29` P3; port option shipped in `modules/nixos.nix` |
| 184 | ◻ OPEN | Document `hardware.emeet-pixy.port` in the website configuration guide + monitoring-guide URL in the option description | MED | S | plan P2/P6; report `2026-10-19_04-19` §b |
| 185 | ◻ OPEN | Prove website CI gates post-lockfile-regen: `pnpm install --frozen-lockfile` + `tsc --strict` | MED | S | plan P2-9/10; lockfile repaired 2026-10-19, proof pending |
| 186 | ◻ OPEN | vmTest default-port pin: assert unset-port renders `EMEET_PIXYD_WEB_ADDR=127.0.0.1:8090` (second node or `nix eval` assertion) | LOW/MED | S | plan P5; non-default pin already green |
| 187 | ◻ OPEN | Cross-link guides: metrics→monitoring, quickshell→monitoring (shared stale-binary trap) | LOW | S | plan P2-7/8 |
| 188 | ◻ OPEN | aarch64 eval coverage: try `nix flake check --no-build --all-systems`, wire into nix.yml or record the accepted gap | LOW | S | plan P8-40 |
| 189 | ◻ OPEN | Lockstep cross-refs: comment pair module port default ↔ `pixy.DefaultWebAddr` so drift is loud | LOW | S | plan P6-22 |

---

All completed work lives in `CHANGELOG.md` ([Unreleased] carries the 2026-09-17 → 09-19 work). Design decisions and open questions live in `ROADMAP.md` — #116 (structured command types), #123 (multi-word preset names), hint surfacing (#173), the NixOS `extraProductIds` option, and the erraudit debt policy have recommendation ADRs (`docs/adr/2026-09-18_*.md`, `docs/adr/2026-09-28_*.md`) awaiting Lars's decision.

> Full ranked backlog with rationale: `docs/status/2026-09-19_14-00_harvest-and-fix-on-sight-sweep.md` §f — older ranked backlogs are harvested into this file; items not covered above are ROADMAP-grade research or micro-hygiene.
