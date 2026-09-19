# emeet-pixyd — Roadmap

**Updated:** 2026-09-19
**Purpose:** Long-term direction and raw ideas **not yet refined into actionable tasks**. When an idea here becomes bounded and estimable, it graduates to `TODO_LIST.md`. When it is rejected, it moves to [Decisions (won't-do)](#decisions-wont-do) below.

> This is the project's living roadmap. The older `docs/SUPERB_ROADMAP.md` (archived 2026-06-05, metrics now stale) is retained only as a historical snapshot — read this file for current direction.

---

## Product vision

emeet-pixyd aims to be the **zero-touch Linux companion** for the EMEET PIXY: plug it in and the camera does the right thing (tracking when you're in a call, privacy when you're not), with a polished local web UI, full CLI/socket control, first-class NixOS integration, and no cloud dependency. Everything below either extends that vision or hardens what already ships.

The daemon is mature: 71/73 features `FULLY_FUNCTIONAL` (see `FEATURES.md`), build/test/lint/nix gates green. Roadmap work is therefore **enrichment and hardening**, not gap-filling.

---

## Themes & raw ideas

### Error handling (build on go-error-family)

go-error-family is adopted at the boundaries that matter (HTTP status derivation, CLI exit codes, daemon-init logging). These are incremental enhancements, not commitments:

- Expand `errorfamily.LogError()` beyond the single daemon-init site to the remaining `slog.Error` call sites that flow through classified sentinels (`state.go`, `process.go`, `uevent.go`, `socket.go`).
- Register `MessageTemplate`s for key error codes to enable structured user-facing messages at CLI/API boundaries.
- Adopt `errorfamily.HTTPHandler()` for the JSON-shaped endpoints (`/api/health`, `/api/snapshot`) — they currently use plain `http.Error`.
- Adopt `errorfamilytest.Assert*` helpers to cut classification-test boilerplate.
- Surface error-family counts in Prometheus (errors by family) and consider per-family error budgets.
- Write an ADR capturing the scoped-adoption decision (why DataStar handlers and the circuit breaker stay outside classification).
- Error-wrapping consistency audit: not every `fmt.Errorf` site uses `%w`; `%v` breaks `errors.Is` chains and therefore classification (flagged in `2026-07-23_21-00` §c).

### Build & release hardening

- Add a CI guard that fails if the go-modules FOD references store paths (regression test for the committed-binary-poisoning class of bug — the go-branded-id incident is fixed upstream, but the class is not).
- Extend the NixOS vmTest: fake sysfs + actually start the daemon inside the VM (the vmTest itself is green since 2026-09-19; this is the deeper extension).
- `nix flake update` cadence / automation (Renovate or a scheduled update job).

### Web presence

- Add more Starlight callouts (`:::tip`/`:::note`/`:::caution`) for notes currently buried in prose.
- Per-page feedback links (issue tracker with pre-filled title); enable reading time.
- Consider mirroring "Who is this for?" / "When NOT to use this" / the comparison matrix onto the Astro landing page (currently README-only).
- Add a `prettier`/`prettierd` config for `.mdx`/`.mjs` to prevent future formatter wars.
- Distill a public "how emeet-pixyd relates to EMEET STUDIO" page from the internal comparison doc — **needs Lars's call** (documents their internals publicly; positioning win vs legal/positioning risk).

### Observability & UX (lower-priority enhancements)

- OpenTelemetry **tracing** (not just metrics) — e.g. trace PTZ command latency.
- SSE heartbeat (prevent proxy idle kills) + `LastEventID` replay after reconnect.
- HTTP panic-recovery middleware.
- Camera diagnostics endpoint (full V4L2 control dump).
- PTZ patrol/sweep mode; configurable home position.
- `koanf` layered config (file + env, replacing env-only).
- Extend Waybar output: auto mode, pan/tilt values, and charging/discharging classes (model + battery level ship; the `ChargeSta` enum needs hardware verification, `TODO_LIST.md` #166). Waybar slot-occupancy (per-slot pull results in the bar JSON) is a **won't-do candidate** — toast/panel coverage suffices (`12-05` §f 40).
- Per-slot `preset pull` outcome in the web UI — render occupied/empty/pulled per slot beyond the one-line toast (`12-05` §f 22).

### EMEET STUDIO research offshoots (intel worth keeping alive)

These came out of the 2026-09 official-app reverse-engineering (`docs/emeet-studio-official-app-comparison.md`, `tools/emhid/`, `tools/inno661/`). None are scheduled; they graduate to TODO_LIST when a demand signal appears:

- **elink wireless protocol documentation** — ~90 command families from the Mac strings; community value for PIXY-Wireless owners.
- **EMEETLINK 5.8.5 (dead end, recorded 2026-09-19)** — downloaded + MD5-verified under `~/specimens/emeet-link/`: a different product line (conference-audio tooling, zero PIXY motor surface). Kept for the record only; do not re-download.
- **`EMVideoInput.dll`/`.plugin` inspection** — the official app's OBS↔app pipe vs our MJPEG stream; architecture depth.
- **usbmon cross-validation** — LARGELY RESOLVED statically (2026-09-19): the Beta.25 speed-query send site (`0x14017ecad`) computes the `mergeType(3,3)` = `0x63` dev byte at send time (literal `(dev<<5)|func` helper at `0x140179c30`), and every parser masks the echo with `& 0x1F` — the `0x63` routing is not 2.0.3-only and both echo forms validate. Remaining (optional): usbmon confirmation for the NON-motor families if the wired device ever behaves oddly.
- **`hidCmdSend`-style bounded retry** in our HID layer (the official impl retries with w4=50) + `hidCmdSend` retry-semantics decoding.
- **Privacy-trigger-time semantics** — the official app exposes a configurable privacy trigger delay; semantics unknown.
- **Motor-speed env default** — `#138` shipped the `speed` command plus persistence (`state.json` carries `speeds`, re-asserted on every move path and on device re-appear); the remaining follow-through is an `EMEET_PIXYD_MOTOR_SPEED` env default + real-unit clamping once the unit/limit is hardware-verified (#166).
- **`GET_DEVICE_MODE` authoritative query** — switch mode reads to the official head (`09 02 01 00`) or document why the empirical SET-head query stays (probe now exercises it).
- **`GET_FUNC_STA` bitfield decode** — turn the raw `func=` hex in `device` output into capability-gated UI.
- **`FuzzParseV2Response`** — parser-security parity with the uevent fuzzer once framing is hardware-pinned.
- **`EMEET_PIXYD_PRODUCT_IDS` env override** for future PIXY variants — YAGNI until a third model appears.
- **Device-DISAPPEAR reconcile semantics** — only device-appear is handled today; what should belief/state do on unplug (clean reset vs keep-last)?
- **`preset pull` (hardware → state), TODO #141 — IMPLEMENTED (2026-09-19, evidence-corrected)**: the inverse of `preset push`: sweep hardware motor slots into named software presets. Static decode (map doc §3.5a) shows the GET (`09 63 01 17` + slot byte) answers **mode-only** (one byte @8) in the Beta.25 build, while slot positions ride the `SET_MOTOR_PRESET_POS_MODE` echo (`[slot][mode][pan][tilt][zoom]`, floats gated on mode==1, min 0x16) — the original design's "GET returns mode@8 + floats" was a mis-attribution to the power-on-default parser. Shipped: `preset pull` queries slots `1..8` (the assumed cap), parses BOTH evidenced shapes via `pixy.ParseMotorPresetPosResponse`, stores full-shape occupied slots as additive `hw-<slot>` presets (rounded, clamped, never overwriting user names), counts mode-only slots as "set (position not exposed)", and gained a web Pull button (`POST /api/preset/pull`). Residual for the #166 hardware session: pin the real slot count and which response shape the wired firmware answers (a 2.0.3-era firmware may answer the GET with the full shape).

---

## Needs a design decision before it can be estimated

These are too design-heavy to be a TODO yet. Capture the decision (preferably as an ADR), then promote to `TODO_LIST.md`.

- **Structured command types** (former TODO #116): replace `handleCommand(string) string` + `strings.Fields` dispatch with typed command structs. High value (type safety, multi-word args) but high effort and touches the whole command surface. **ADR written** (`docs/adr/2026-09-18_structured-command-types.md`, recommends incremental typed registry) — awaiting Lars's decision.
- **Multi-word preset names via CLI** (former TODO #123): the web UI handles them, but CLI `strings.Fields` dispatch silently truncates at the first space. **ADR written** (`docs/adr/2026-09-18_multi-word-preset-names.md`, recommends join-remaining-parts; pinning test proves the bug live) — awaiting Lars's decision; the ~6-line implementation lands immediately after.
- **Re-assert AUDIO after power cycles too** — the reconcile (TODO #137, shipped) re-asserts the persisted camera mode but deliberately adopts audio/gesture from hardware; that boundary is documented. Changing it is a product decision, not a bug.

---

## Open questions (need a human answer, not a task)

- **Dependency-bump ownership for `website/`:** should bumps be dependabot-only (local sessions never bump; they `git sync` first)? This is the root-cause fix for the lockfile-conflict class (`2026-09-17_14-26` g2). Related: should the auto-commit daemon also push, or is manual `git sync` cadence intentional?
- **`/tmp` research raw materials — RESOLVED in practice (2026-09-19)**: the lost specimens were re-acquired from the Wayback Machine (CDX urlkey query + `id_` snapshot fetch — the durable re-acquisition recipe for any vendor-pulled file). Durable home: `~/specimens/emeet-studio/` (Beta.25 Win installer + full extraction); distilled derived data is committed (`tools/emhid/cmdtable.json` + `x64_heads.json`, `tools/inno661/data/parsed*.json`); the regeneration pipeline is `tools/emhid/extract_x64.py` (verified byte-identical). Residual: the 2.0.3 Mac installer and the Beta.25 Mac pkg are still un-re-acquired (first Wayback attempt 404'd; alternate snapshot forms untried) — only needed for a same-version parser cross-check, nothing is blocked on it.
- **Is the `buildflow --fix --semantic` daemon intentional?** It aggressively reverts in-flight edits during failed builds (see `2026-07-28_15-24` §g.2). If intentional, future sessions should work in a temp checkout; if not, killing it removes real churn. (Operational, not a code task.)
- **Push cadence:** sessions keep accumulating local commits while origin CI rots and dependabot diverges (`2026-09-17_14-26` §e4, `2026-09-17_17-34` f19). Should the daemon push per task, or is a manual cadence intentional?

---

## Decisions (won't-do)

Rejected ideas with rationale. Kept here so they are not re-proposed. (Former TODO items #79/#85/#86/#96/#98, plus the go-error-family scope decisions.)

| Idea                                                    | Decision           | Rationale                                                                                                                                                                                                                             |
| ------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remove `prometheus/client_golang` (#79/#96)             | Won't-do           | The OTel Prometheus exporter depends on it transitively; `promhttp.Handler()` is required for `/metrics`. Verified via `go mod graph`.                                                                                                |
| Move `main.go` → `cmd/emeet-pixyd/main.go` (#85)        | Won't-do           | This is a single-binary daemon with no subcommands by design. Root `main.go` is a defensible layout; the move would churn `flake.nix`/`package.nix`/CI for no behavior change. BuildFlow flags it, but it is a convention, not a bug. |
| Decompose the `Daemon` struct (#86)                     | Won't-do           | ~17 fields is manageable for a single-binary hardware daemon; splitting adds indirection without clarity.                                                                                                                             |
| Move `SSEEvent` to `internal/pixy` (#98)                | Won't-do           | `SSEEvent` is a transport-layer DTO; it belongs in `sse.go`, not the domain package.                                                                                                                                                  |
| Classify DataStar action handlers via go-error-family   | Won't-do           | DataStar SSE patches need HTTP 200 + a patch-elements/toast payload to render errors in-panel. Returning 4xx/5xx + JSON would break the UI.                                                                                           |
| Replace the HID circuit breaker with `IsRetryable()`    | Won't-do           | The existing `hidCircuitBreakerThreshold = 3` + re-probe logic is more nuanced than a binary retry flag.                                                                                                                              |
| Move `toastType` to `internal/pixy`                     | Done-differently   | `toastType` already lives in `web_types.go`; `SSEEvent` stays in `sse.go` (transport DTO).                                                                                                                                            |
| Convert keyboard shortcuts to `data-on:keydown__window` | Won't-do (for now) | The DataStar-native form needs awkward templ expression escaping and the PTZ arrow-key logic reads slider values; the ~80 lines of `app.js` work and are testable. Revisit only if `app.js` grows again.                              |
