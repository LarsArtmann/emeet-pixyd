# emeet-pixyd — TODO List

**Updated:** 2026-07-28 (docs-health audit — open work harvested from the five `2026-07-2x` reports and verified against the code)
**Source docs verified:** `docs/status/2026-07-2*` (5 files), `docs/status/2026-07-04_16-29_*`, all living docs. Every claim below was checked against the code on 2026-07-28.

> Completed work lives in `CHANGELOG.md` — it does NOT live here. Long-term ideas, design-heavy items, "decided won't-do" decisions, and open questions live in `ROADMAP.md`. This file is **open work only**.

---

## Status Legend

- ⬜ TODO — Not started
- 🔶 PARTIAL — Started but incomplete
- 🚫 BLOCKED — Waiting on an external unblock (permission, decision, upstream)

---

## Blocked (highest impact — needs an external unblock)

| #   | Status     | Task                                                                                                                                                                                            | Impact | Effort | Evidence                                                                                                                                                  |
| --- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 124 | 🚫 BLOCKED | **Publish `go-branded-id` v0.5.1** (binary-free), bump `emeet-pixyd` go.mod, remove the TEMPORARY `goBrandedSrc`/`replaceBrandedId` shims from `flake.nix` + `package.nix`, recompute `vendorHash` | 🔴 HIGH | S      | `go.mod` still `v0.5.0`; `flake.nix:77-94` + `package.nix:7-8,27` still carry the workaround; `2026-07-28_15-24` §b.1, §c.1, §f.1, §g.1; AGENTS.md gotcha |
| 127 | 🚫 BLOCKED | **Verify the live site reflects the website retrofit; deploy if not yet live** (`nix run .#deploy` / `firebase deploy --only hosting:emeet-pixyd`)                                                | MED    | S      | `2026-07-28_14-28` §c — retrofit was committed/build-green but reported not deployed. Needs a deploy go-ahead.                                             |

#124 is the single highest-impact debt in the project: it is **blocked on push permission** to the `go-branded-id` remote (the binary is already untracked upstream in `c29a034`; only the tag/publish + go.mod bump remain). #127 needs a deploy decision.

---

## High-impact quick wins

| #   | Status | Task                                                                                                            | Impact | Effort | Evidence                                                                                                                                                 |
| --- | ------ | --------------------------------------------------------------------------------------------------------------- | ------ | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 125 | ⬜ TODO | **Fix `.editorconfig`** to add `[*.{md,mdx}]` and `[*.{js,mjs,cjs}]` space rules, then re-run the formatter      | MED    | S      | `.editorconfig` has `[*] indent_style = tab` with no `.mdx`/`.mjs` override → those files indent with tabs; `2026-07-28_14-28` §d                          |
| 126 | ⬜ TODO | **Pin all GitHub Actions to commit SHAs** (9 tag pins across `go-test.yml`, `nix.yml`)                          | MED    | S      | `.github/workflows/*.yml`: `checkout@v4`, `setup-go@v5`, `golangci-lint-action@v7`, `cache@v4`, `nick-fields/retry@v3`, `nix-installer-action@v16`, `magic-nix-cache-action@v9`; `2026-07-23_21-27` §c.2 |
| 128 | ⬜ TODO | **Triage Dependabot alerts with `govulncheck`** (reported: `fast-uri` HIGH host-confusion; Astro MEDIUM XSS)    | MED    | S      | `2026-07-23_21-27` §c.3; `govulncheck` not on local PATH — run via CI or a nix shell                                                                       |

---

## UX / Accessibility

| #   | Status | Task                                                                       | Impact | Effort | Evidence               |
| --- | ------ | -------------------------------------------------------------------------- | ------ | ------ | ---------------------- |
| 106 | ⬜ TODO | `hx-on::after-swap` focus management for keyboard users (outerHTML swap loses focus) | MED | LOW  | `2026-07-04` §e.8       |
| 107 | ⬜ TODO | SSE connection status indicator (green/red dot) in the UI                  | MED    | LOW    | `2026-07-04` §e.9       |
| 109 | ⬜ TODO | Screen reader test pass (manual; document findings)                        | MED    | LOW    | `2026-07-04` §f.3       |
| 110 | ⬜ TODO | WCAG 2.1 AA audit                                                          | MED    | MED    | `2026-07-04` §f.15      |
| 111 | ⬜ TODO | Preset name autocomplete in web UI save input                              | LOW    | MED    | `2026-07-04` §f.21      |
| 112 | ⬜ TODO | Mobile device testing pass (real phone/iPad/landscape)                     | MED    | MED    | FEATURES.md (Mobile)   |

---

## Architecture

| #   | Status | Task                                                                          | Impact | Effort | Evidence                          |
| --- | ------ | ----------------------------------------------------------------------------- | ------ | ------ | --------------------------------- |
| 114 | ⬜ TODO | Consolidate `commandMsgError` into the `CommandError` pattern                 | LOW    | MED    | `2026-07-04` §e.2; `errors.go:35` |
| 115 | ⬜ TODO | Add `DisallowUnknownFields` to the state JSON decoder (strict schema)         | LOW    | LOW    | `2026-07-04` §f.17; absent in `state.go` |

> #116 (structured command types — HIGH/HIGH) and #123 (multi-word preset names through CLI dispatch) need design decisions first; both are in `ROADMAP.md`.

---

## Testing

| #   | Status | Task                                                            | Impact | Effort | Evidence           |
| --- | ------ | --------------------------------------------------------------- | ------ | ------ | ------------------ |
| 117 | ⬜ TODO | Snapshot testing for web panel HTML (`go-snaps`)                | MED    | MED    | `2026-07-04` §f.8  |
| 118 | ⬜ TODO | Property-based tests for `ValidatePresetName` and `Range.Clamp` | LOW    | MED    | `2026-07-04` §f.9  |
| 119 | ⬜ TODO | `go-snaps` snapshot test for waybar JSON output                 | LOW    | LOW    | `2026-07-04` §f.18 |
| 120 | ⬜ TODO | Integration test: full auto-manage lifecycle with fake devices  | MED    | MED    | `2026-07-04` §f.20 |
| 121 | ⬜ TODO | Add `wpctl` mock for PipeWire integration tests                 | LOW    | MED    | `2026-07-04` §f.24 |

---

## Docs

| #   | Status | Task                                               | Impact | Effort | Evidence          |
| --- | ------ | -------------------------------------------------- | ------ | ------ | ----------------- |
| 122 | ⬜ TODO | Document HID protocol reverse-engineering findings | LOW    | LOW    | `2026-07-04` §f.23 |

---

## Resolved in this audit (removed from the backlog)

These were open in the prior list and are now done or superseded — recorded here only so the numbering change is traceable. Full detail is in `CHANGELOG.md [Unreleased]`.

- ~~#108~~ Add gesture toggle to web UI — **DONE**: toggle is in `templates.templ:168` posting to `/api/gesture`; handler `handleGestureCommand` (`commands.go:253`).
- ~~#113~~ Wire `errors.Is` checks for the 9 sentinels — **SUPERSEDED** by go-error-family adoption: `errorfamily.go` registers all sentinels via `RegisterClassification` (which walks `errors.Is` chains), so callers get classification without per-site `errors.Is` wiring.

> The prior "Resolved History (archive)" (items #1–#105, Phases 1–10) was the project's completed backlog. It has been removed from this file — completed work lives in `CHANGELOG.md`, and the "decided won't-do" decisions (former #79, #85, #86, #96, #98) live in `ROADMAP.md` → "Decisions (won't-do)". That removal is the point: a TODO list is open work, not a trophy case.
