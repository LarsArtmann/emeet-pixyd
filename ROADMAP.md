# emeet-pixyd — Roadmap

**Updated:** 2026-07-28
**Purpose:** Long-term direction and raw ideas **not yet refined into actionable tasks**. When an idea here becomes bounded and estimable, it graduates to `TODO_LIST.md`. When it is rejected, it moves to [Decisions (won't-do)](#decisions-wont-do) below.

> This is the project's living roadmap. The older `docs/SUPERB_ROADMAP.md` (archived 2026-06-05, metrics now stale) is retained only as a historical snapshot — read this file for current direction.

---

## Product vision

emeet-pixyd aims to be the **zero-touch Linux companion** for the EMEET PIXY: plug it in and the camera does the right thing (tracking when you're in a call, privacy when you're not), with a polished local web UI, full CLI/socket control, first-class NixOS integration, and no cloud dependency. Everything below either extends that vision or hardens what already ships.

The daemon is mature: 59/60 features `FULLY_FUNCTIONAL` (see `FEATURES.md`), build/test/lint/nix gates green. Roadmap work is therefore **enrichment and hardening**, not gap-filling.

---

## Themes & raw ideas

### Error handling (build on go-error-family)

go-error-family is adopted at the boundaries that matter (HTTP status derivation, CLI exit codes, daemon-init logging). These are incremental enhancements, not commitments:

- Expand `errorfamily.LogError()` beyond the single daemon-init site to the remaining `slog.Error` call sites that flow through classified sentinels (`state.go`, `process.go`, `uevent.go`, `socket.go`).
- Register `MessageTemplate`s for key error codes to enable structured user-facing messages at CLI/API boundaries.
- Adopt `errorfamily.HTTPHandler()` for the JSON-shaped endpoints (`/api/health`, `/api/snapshot`) — they currently use plain `http.Error`.
- Adopt `errorfamilytest.Assert*` helpers to cut classification-test boilerplate.
- Surface error-family counts in Prometheus (errors by family) and consider per-family error budgets.
- Write an ADR capturing the scoped-adoption decision (why HTMX handlers and the circuit breaker stay outside classification).

### Build & release hardening

- Replace the TEMPORARY `go-branded-id` in-sandbox `replace` with a published binary-free version (this is `TODO_LIST #124` once unblocked — the single biggest debt).
- Generalize the FOD guard: strip **all** root-level ELF artifacts in `postFetch` rather than hardcoding `namer`.
- Add a NixOS VM test that exercises `nixosModules.default` end-to-end (today only the overlay delegation is checked).
- Add a CI step that fails if the go-modules FOD references store paths (regression test for the binary-poisoning class of bug).
- `nix flake update` cadence / automation (Renovate or nvfetcher for `goBrandedSrc`).

### Web presence

- Deploy/verify the retrofitted docs site (`TODO_LIST #127`).
- Add more Starlight callouts (`:::tip`/`:::note`/`:::caution`) for notes currently buried in prose.
- Per-page feedback links (issue tracker with pre-filled title); enable reading time.
- Consider mirroring "Who is this for?" / "When NOT to use this" / the comparison matrix onto the Astro landing page (currently README-only).
- Add a `prettier`/`prettierd` config for `.mdx`/`.mjs` to prevent future formatter wars (pairs with the `.editorconfig` fix in `TODO_LIST #125`).

### Observability & UX (lower-priority enhancements)

- OpenTelemetry **tracing** (not just metrics) — e.g. trace PTZ command latency.
- SSE heartbeat (prevent proxy idle kills) + `LastEventID` replay after reconnect.
- HTTP panic-recovery middleware.
- Camera diagnostics endpoint (full V4L2 control dump).
- PTZ patrol/sweep mode; configurable home position; movement-speed control.
- `koanf` layered config (file + env, replacing env-only).
- Add `pan`/`tilt` to Waybar tooltip output.

---

## Needs a design decision before it can be estimated

These are too design-heavy to be a TODO yet. Capture the decision (preferably as an ADR), then promote to `TODO_LIST.md`.

- **Structured command types** (former TODO #116): replace `handleCommand(string) string` + `strings.Fields` dispatch with typed command structs. High value (type safety, multi-word args) but high effort and touches the whole command surface. Design question: command-parser library vs. hand-rolled registry.
- **Multi-word preset names via CLI** (former TODO #123): the web UI handles them, but CLI `strings.Fields` dispatch silently truncates at the first space. Options: quote support / join-remaining-parts / structured commands / accept the limitation. Tied to the structured-commands decision above.

---

## Open questions (need a human answer, not a task)

- **Is the `buildflow --fix --semantic` daemon intentional?** It aggressively reverts in-flight edits during failed builds (see `2026-07-28_15-24` §g.2). If intentional, future sessions should work in a temp checkout; if not, killing it removes real churn. (Operational, not a code task.)

---

## Decisions (won't-do)

Rejected ideas with rationale. Kept here so they are not re-proposed. (Former TODO items #79/#85/#86/#96/#98, plus the go-error-family scope decisions.)

| Idea | Decision | Rationale |
| --- | --- | --- |
| Remove `prometheus/client_golang` (#79/#96) | Won't-do | The OTel Prometheus exporter depends on it transitively; `promhttp.Handler()` is required for `/metrics`. Verified via `go mod graph`. |
| Move `main.go` → `cmd/emeet-pixyd/main.go` (#85) | Won't-do | This is a single-binary daemon with no subcommands by design. Root `main.go` is a defensible layout; the move would churn `flake.nix`/`package.nix`/CI for no behavior change. BuildFlow flags it, but it is a convention, not a bug. |
| Decompose the `Daemon` struct (#86) | Won't-do | ~17 fields is manageable for a single-binary hardware daemon; splitting adds indirection without clarity. |
| Move `SSEEvent` to `internal/pixy` (#98) | Won't-do | `SSEEvent` is a transport-layer DTO; it belongs in `sse.go`, not the domain package. |
| Classify HTMX action handlers via go-error-family | Won't-do | HTMX `outerHTML` swaps need HTTP 200 + an HTML toast to render errors in-panel. Returning 4xx/5xx + JSON would break the UI. |
| Replace the HID circuit breaker with `IsRetryable()` | Won't-do | The existing `hidCircuitBreakerThreshold = 3` + re-probe logic is more nuanced than a binary retry flag. |
| Move `toastType` to `internal/pixy` | Done-differently | `toastType` already lives in `web_types.go`; `SSEEvent` stays in `sse.go` (transport DTO). |
