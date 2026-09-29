# ADR: Erraudit Debt — Site-Level `//nolint:erraudit` Directives Over a Silent Red Gate

**Date:** 2026-09-28
**Status:** Proposed — awaiting Lars's decision
**Context:** A 2026-09-28 triage of the buildflow erraudit step found 31 findings: 2 genuine (`%v`-on-err in `socket.go`, fixed to `%w` the same day) and 29 accepted-by-design — 22 `ignored` (best-effort `_ =` closes on cleanup paths), 2 `silent_swallow` (deliberate `/proc` scan races: a process may vanish mid-scan, skip-and-continue is the contract), 7 `sentinel_concrete_type` (`errors.Is` on go-error-family sentinels kept on purpose — classification by value identity; the go-error-modernization decision tree's explicit sentinel exception). The full rationale is recorded in `AGENTS.md` (Gotchas, "Erraudit baseline"). The open question is the *mechanism*: leave the buildflow gate permanently red (exit 69, documented as expected), or suppress at the sites.

**Tool facts (verified against `erraudit --help`, 2026-09-28):** erraudit honors `//nolint:erraudit` directives, and ships `erraudit nolint-audit` — a staleness auditor that flags directives whose underlying violation disappeared. Suppression at the site is therefore a first-class, auditable workflow, not a hack.

## Options

### Option A — Documented baseline, gate stays red

The current state: rationale in AGENTS.md, buildflow exits 69, everyone memorizes that this particular red is "expected". Rejected: a permanently red gate is alarm fatigue. The first *real* regression to arrive at finding #30 hides behind 29 documented ones, and every future session re-reads the gotcha to re-derive that the red is benign — the AGENTS.md note even had to say "do NOT re-triage from scratch" because the pull to re-check is real.

### Option B — `//nolint:erraudit` at the 29 sites — RECOMMENDED

One mechanical sweep: add the directive (with a brief reason, matching the golangci-lint call-site convention this repo already uses) at each of the 29 findings, then run `erraudit nolint-audit` to confirm zero stale directives. The buildflow gate returns to meaningful green: exit 0 means clean, non-zero means a NEW finding, which is exactly the signal a gate exists to give. Cost: one ~30–45m sweep; ongoing cost near zero because the staleness auditor replaces the manual "remove stale ones" discipline that golangci-lint directives need.

### Option C — Global config exclusion per finding type

Rejected. `--no-suppress` is for audits, not baselines, and a blanket "always ignore `ignored`/`silent_swallow`/`sentinel_concrete_type`" would also mute future genuine findings of those classes (the 2 genuine fixes this sweep WERE of classes otherwise accepted). Suppression belongs at the site, where the justification lives next to the code.

## Decision

Pending Lars's sign-off on Option B. Follow-up: the sweep itself (new TODO row; also un-blocks plan M07's "apply M04-Q3 outcome" step and makes the buildflow gate trustworthy again).

---

## Addendum 2026-09-29 — SUPERSEDED by the construction migration

Lars directed "accept nothing — make the error system as superb as possible instead", which replaces this ADR's premise (accept the 31, suppress at sites) with a fix-at-the-root migration: all error construction now goes through `go-error-family` (coded sentinels, family-inheriting per-family wraps, `Compose` joins), sentinels are widened to the `error` interface, and every previously discarded error is handled (logged, joined, or classified). Default-mode erraudit reports **0 violations** — this ADR's Option B sweep at the old 31 sites is moot and was never executed. The 21 residual maximal-audit `generic_return` advisories (warning severity only; `--no-suppress` bypasses directives by design) carry reasoned function-level `//nolint:erraudit` directives. Rationale and tool-behavior notes live in `AGENTS.md` (Gotchas, "Erraudit state").
