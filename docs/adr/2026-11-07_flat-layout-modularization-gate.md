# ADR: Flat layout stays until a named second consumer exists

**Date:** 2026-11-07
**Status:** Accepted
**Context:** Four consecutive questions in one session ("why so much code in root?", "can't we have a single main.go?", "sub folders make no sense or what?"), assessed with the go-modularize skill's Unix gate. Full analysis: `docs/status/2026-11-07_06-40_architecture-layout-qa-session.md`, visual review: `docs/reviews/2026-11-07_data-model-and-layout-review.html`.

## Decision

emeet-pixyd stays a **single Go module with the daemon in the root `package main`** and the domain types in `internal/pixy`. No `cmd/` split, no sub-package decomposition, no second go.mod — until a **named second consumer** exists (gate 2 evidence bar: a real importer or a committed plan row/ADR/TODO stating what it composes and the trigger that fires it).

## Gate verdict (go-modularize, 2026-11-07)

| Package | Purpose | Direction | Surface | Pipe test | Solo build | Verdict |
|---|---|---|---|---|---|---|
| `internal/pixy` (domain) | domain types for PIXY control | ✓ zero daemon infra (only go-branded-id, go-error-family, stdlib — verified via `go list`) | ✓ `internal/` | ✗ only consumer is `main`; no named second consumer (NixOS module consumes the binary, `tools/` is Python) | n/a (same module) | keep as package; do NOT promote to module |
| root `main` (daemon) | run the PIXY daemon | ✓ | ✓ package main = all private | n/a (leaf) | ✓ | cohesive; FM#11 (god-module, 14+ packages) not triggered |

## Options considered and rejected

1. **Single `main.go`** — rejected. 33 source files ≈ 7,172 hand-written lines (verified `wc -l`, 2026-11-07) plus the generated `templates_templ.go`; ~72 test files (~17.9k lines) remain regardless. A god-file is FM#11 at file granularity; filenames are the architecture index (AGENTS.md table).
2. **Cosmetic `cmd/emeet-pixyd/` + `internal/daemon/` move** — viable but deferred: zero semantic gain, real churn (flake `subPackages`, AGENTS.md table, path pins in CI/`.crushrc` — unverified at proposal time). Re-open if root listing noise becomes a daily pain.
3. **Real sub-packages (`internal/hid|stream|web|process`)** — blocked: every file operates on `*Daemon`'s private state and the `hidMu`→`v4l2Mu` lock-order contract is pinned by same-package tests. Precondition is a god-struct decomposition (`hidTransport`/`streamServer`/`callDetector` types) — the data-model migrations in the review HTML are the first step of that preparation.

## Consequences

- Future sessions must not relitigate this without new gate-2 evidence; cite this ADR.
- The extraction trigger is precise: a second binary importing the HID/V4L2 layer, or an external consumer of `internal/pixy` types (which first requires graduating them out of `internal/`).
- When that fires, the seam already points the right direction (`internal/pixy` imports zero daemon infrastructure), making the extraction mechanical.
