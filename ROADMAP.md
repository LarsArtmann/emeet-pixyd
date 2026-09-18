# emeet-pixyd — Roadmap

**Updated:** 2026-09-18
**Purpose:** Long-term direction and raw ideas **not yet refined into actionable tasks**. When an idea here becomes bounded and estimable, it graduates to `TODO_LIST.md`. When it is rejected, it moves to [Decisions (won't-do)](#decisions-wont-do) below.

> This is the project's living roadmap. The older `docs/SUPERB_ROADMAP.md` (archived 2026-06-05, metrics now stale) is retained only as a historical snapshot — read this file for current direction.

---

## Product vision

emeet-pixyd aims to be the **zero-touch Linux companion** for the EMEET PIXY: plug it in and the camera does the right thing (tracking when you're in a call, privacy when you're not), with a polished local web UI, full CLI/socket control, first-class NixOS integration, and no cloud dependency. Everything below either extends that vision or hardens what already ships.

The daemon is mature: 63/66 features `FULLY_FUNCTIONAL` (see `FEATURES.md`), build/test/lint/nix gates green. Roadmap work is therefore **enrichment and hardening**, not gap-filling.

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
- Extend the NixOS vmTest: fake sysfs + actually start the daemon inside the VM (after TODO #157 fixes the hang).
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
- Extend Waybar output: auto mode, pan/tilt, and (pending #139 verdict) battery.

### EMEET STUDIO research offshoots (intel worth keeping alive)

These came out of the 2026-09 official-app reverse-engineering (`docs/emeet-studio-official-app-comparison.md`, `tools/emhid/`, `tools/inno661/`). None are scheduled; they graduate to TODO_LIST when a demand signal appears:

- **elink wireless protocol documentation** — ~90 command families from the Mac strings; community value for PIXY-Wireless owners.
- **`EMVideoInput.dll`/`.plugin` inspection** — the official app's OBS↔app pipe vs our MJPEG stream; architecture depth.
- **usbmon cross-validation** — capture the official app on Windows to confirm the `mergeType` sub-device routing (`0x63` vs `3` iface byte).
- **`hidCmdSend`-style bounded retry** in our HID layer (the official impl retries with w4=50) + `hidCmdSend` retry-semantics decoding.
- **Privacy-trigger-time semantics** — the official app exposes a configurable privacy trigger delay; semantics unknown.
- **Motor-speed state persistence** — `state.json` schema v2 if #138 (PTZ speed) ships with a desired default.
- **`EMEET_PIXYD_PRODUCT_IDS` env override** for future PIXY variants — YAGNI until a third model appears.
- **Device-DISAPPEAR reconcile semantics** — only device-appear is handled today; what should belief/state do on unplug (clean reset vs keep-last)?
- **Motor-preset slot-count discovery** — `GetMotorPresetPosMode` slot sweep to learn how many hardware slots exist (design input for #141).

---

## Needs a design decision before it can be estimated

These are too design-heavy to be a TODO yet. Capture the decision (preferably as an ADR), then promote to `TODO_LIST.md`.

- **Structured command types** (former TODO #116): replace `handleCommand(string) string` + `strings.Fields` dispatch with typed command structs. High value (type safety, multi-word args) but high effort and touches the whole command surface. Design question: command-parser library vs. hand-rolled registry.
- **Multi-word preset names via CLI** (former TODO #123): the web UI handles them, but CLI `strings.Fields` dispatch silently truncates at the first space. Options: quote support / join-remaining-parts / structured commands / accept the limitation. Tied to the structured-commands decision above.
- **Re-assert AUDIO after power cycles too** — the reconcile (TODO #137, shipped) re-asserts the persisted camera mode but deliberately adopts audio/gesture from hardware; that boundary is documented. Changing it is a product decision, not a bug.

---

## Open questions (need a human answer, not a task)

- **Dependency-bump ownership for `website/`:** should bumps be dependabot-only (local sessions never bump; they `git sync` first)? This is the root-cause fix for the lockfile-conflict class (`2026-09-17_14-26` g2). Related: should the auto-commit daemon also push, or is manual `git sync` cadence intentional?
- **`/tmp` research raw materials** (348 MB Windows payload, 241 MB Mac binary, 104 MB disasm): archive somewhere durable, or let them die with reboot? The derived knowledge (cmdtable, format spec, parsed.json) is safely in-repo. (Asked 2026-09-17 18:43, still open.)
- **Is the `buildflow --fix --semantic` daemon intentional?** It aggressively reverts in-flight edits during failed builds (see `2026-07-28_15-24` §g.2). If intentional, future sessions should work in a temp checkout; if not, killing it removes real churn. (Operational, not a code task.)
- **Push cadence:** sessions keep accumulating local commits while origin CI rots and dependabot diverges (`2026-09-17_14-26` §e4, `2026-09-17_17-34` f19). Should the daemon push per task, or is a manual cadence intentional?

---

## Decisions (won't-do)

Rejected ideas with rationale. Kept here so they are not re-proposed. (Former TODO items #79/#85/#86/#96/#98, plus the go-error-family scope decisions.)

| Idea                                                  | Decision         | Rationale                                                                                                                                                                                                                             |
| ----------------------------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remove `prometheus/client_golang` (#79/#96)           | Won't-do         | The OTel Prometheus exporter depends on it transitively; `promhttp.Handler()` is required for `/metrics`. Verified via `go mod graph`.                                                                                                |
| Move `main.go` → `cmd/emeet-pixyd/main.go` (#85)      | Won't-do         | This is a single-binary daemon with no subcommands by design. Root `main.go` is a defensible layout; the move would churn `flake.nix`/`package.nix`/CI for no behavior change. BuildFlow flags it, but it is a convention, not a bug. |
| Decompose the `Daemon` struct (#86)                   | Won't-do         | ~17 fields is manageable for a single-binary hardware daemon; splitting adds indirection without clarity.                                                                                                                             |
| Move `SSEEvent` to `internal/pixy` (#98)              | Won't-do         | `SSEEvent` is a transport-layer DTO; it belongs in `sse.go`, not the domain package.                                                                                                                                                  |
| Classify DataStar action handlers via go-error-family | Won't-do         | DataStar SSE patches need HTTP 200 + a patch-elements/toast payload to render errors in-panel. Returning 4xx/5xx + JSON would break the UI.                                                                                           |
| Replace the HID circuit breaker with `IsRetryable()`  | Won't-do         | The existing `hidCircuitBreakerThreshold = 3` + re-probe logic is more nuanced than a binary retry flag.                                                                                                                              |
| Move `toastType` to `internal/pixy`                   | Done-differently | `toastType` already lives in `web_types.go`; `SSEEvent` stays in `sse.go` (transport DTO).                                                                                                                                            |
| Convert keyboard shortcuts to `data-on:keydown__window` | Won't-do (for now) | The DataStar-native form needs awkward templ expression escaping and the PTZ arrow-key logic reads slider values; the ~80 lines of `app.js` work and are testable. Revisit only if `app.js` grows again. |
