# emeet-pixyd — TODO List

**Updated:** 2026-07-28 (all actionable items completed — see `CHANGELOG.md` for details)

> Completed work lives in `CHANGELOG.md` — it does NOT live here. Long-term ideas, design-heavy items, "decided won't-do" decisions, and open questions live in `ROADMAP.md`. This file is **open work only**.

---

## Status Legend

- ⬜ TODO — Not started
- 🔶 PARTIAL — Started but incomplete
- 🚫 BLOCKED — Waiting on an external unblock (permission, decision, upstream)

---

## Blocked (highest impact — needs an external unblock)

| #   | Status     | Task                                                                                                                                                                                               | Impact  | Effort | Evidence                                                                                                                                                  |
| --- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 124 | 🚫 BLOCKED | **Publish `go-branded-id` v0.5.1** (binary-free), bump `emeet-pixyd` go.mod, remove the TEMPORARY `goBrandedSrc`/`replaceBrandedId` shims from `flake.nix` + `package.nix`, recompute `vendorHash` | 🔴 HIGH | S      | `go.mod` still `v0.5.0`; `flake.nix:77-94` + `package.nix:7-8,27` still carry the workaround; `2026-07-28_15-24` §b.1, §c.1, §f.1, §g.1; AGENTS.md gotcha |
| 127 | 🚫 BLOCKED | **Verify the live site reflects the website retrofit; deploy if not yet live** (`nix run .#deploy` / `firebase deploy --only hosting:emeet-pixyd`)                                                 | MED     | S      | `2026-07-28_14-28` §c — retrofit was committed/build-green but reported not deployed. Needs a deploy go-ahead.                                            |

#124 is the single highest-impact debt in the project: it is **blocked on push permission** to the `go-branded-id` remote (the binary is already untracked upstream in `c29a034`; only the tag/publish + go.mod bump remain). #127 needs a deploy decision.

---

All other items (#106, #107, #109, #110, #111, #112, #114, #115, #117, #118, #119, #120, #121, #122, #125, #126, #128) were completed on 2026-07-28. See `CHANGELOG.md` for details.

> #116 (structured command types — HIGH/HIGH) and #123 (multi-word preset names through CLI dispatch) need design decisions first; both are in `ROADMAP.md`.
