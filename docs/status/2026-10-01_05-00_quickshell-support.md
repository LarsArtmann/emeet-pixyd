# Status Report — Quickshell / DankMaterialShell Support

**Date:** 2026-10-01 05:00
**Session scope:** Add first-class Quickshell support to emeet-pixyd and make the user's `SystemNix/platforms/nixos/desktop/quickshell.nix` camera widget actually work.
**Author:** Crush (session)
**Repos touched:** `emeet-pixyd` (primary), `SystemNix` (one widget file)

---

## TL;DR

The user's DMS/Quickshell camera widget was polling `GET /api/status`, which
**did not exist** — the daemon only had `/api/health`. The widget therefore
always rendered "off". I added the `/api/status` JSON endpoint, documented a
Quickshell integration (guide + ready-to-paste widget + Home Manager wiring),
and rewired the SystemNix widget to the new contract.

Everything I authored builds, lints (0 issues), and the new tests pass. The
website builds 21 pages. **The full Go suite is NOT green**, but both remaining
failures are unrelated to this work (see §d).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Diagnosed the real gap: widget polls `/api/status`; daemon only serves `/api/health` | route table in `handlers.go`; widget `CameraWidget.qml:19` |
| 2 | Added `GET /api/status` (`statusResponse` + `handleStatusJSON`) | `handlers.go:165`–`212`, route `handlers.go:445` |
| 3 | Endpoint always returns `200` while the daemon runs (widget can distinguish daemon-down vs camera-offline) | `handleStatusJSON` writes `http.StatusOK` unconditionally |
| 4 | Lean stable contract: `camera, device, model, online, inCall, auto, audio, gesture, battery, version` | `statusResponse` struct, json tags |
| 5 | Reused `getWebStatus` → same state + TTL-cached battery reading as web panel/Waybar (DRY) | `handleStatusJSON` calls `s.getWebStatus` |
| 6 | Battery omitted when unavailable (`json:"battery,omitzero"`, json/v2) | shape test asserts `"battery"` absent |
| 7 | Tests: offline, online, JSON shape | `web_status_test.go` (3 tests, all pass) |
| 8 | Method-guard test extended: `/api/status` rejects POST | `web_audio_test.go` `TestWeb_GETEndpointsRejectPOST` |
| 9 | Website guide `guides/quickshell.mdx` (contract table, DMS widget, Home Manager snippet, click actions) | new file |
| 10 | Sidebar entry added | `website/astro.config.mjs` |
| 11 | Cross-links: Waybar guide + Related Tools + architecture endpoint table | 3 files |
| 12 | README: "Who is this for" + Features table row | `README.md` |
| 13 | Architecture endpoint table made accurate: added `/api/status`, and the previously-undocumented `/api/speed`, `/api/tracking`, `/api/preset/push`, `/api/preset/pull` | `architecture/overview.mdx` |
| 14 | Memory docs updated: `CHANGELOG.md` [Unreleased], `FEATURES.md`, `AGENTS.md` | 3 files |
| 15 | SystemNix widget rewired to `camera` + state colors (`privacy`→error, `idle`→warning, `tracking`→primary, unreachable→distinct `off`) | `SystemNix/pkgs/dms-plugins/systemnix-camera/CameraWidget.qml` |
| 16 | Verified: `go build`, `go vet`, `golangci-lint run` (0 issues), new tests, website `pnpm run build` (21 pages, CSP patched) | command output |
| 17 | Confirmed the pre-existing motor test failure predates this session | git worktree at `938f6e5` reproduced it |

---

## b) PARTIALLY DONE

| # | Item | What's missing |
|---|------|----------------|
| 1 | Endpoint contract | No PTZ (`pan/tilt/zoom`), `trackMode`, `speeds`, `error`, `lastSynced`. Deliberately lean, but richer widgets want more. |
| 2 | Test coverage | No "battery **present**" test; no test that `device`/`model` reflect a real probe; no benchmark for the new handler. |
| 3 | SystemNix widget | Rewritten but **never run or QML-linted**; theme tokens (`Theme.error`, `Theme.warning`) inferred from other plugins, not proven valid for this context. |
| 4 | Docs | Guide's example widget and the deployed SystemNix widget diverged slightly (`statusLabel`/`stateColor` vs `statusText`/`statusColor`). Should be identical. |
| 5 | Interpretation of "support quickshell.nix" | Executed on the most plausible reading without confirming. If the user meant a Home-Manager **module** or a SystemNix-only change, scope was mis-targeted. |
| 6 | Verification of full suite | Ran it, but the repo was being mutated concurrently by another session, so "green" could not be established. |

---

## c) NOT STARTED

| # | Item | Why it matters |
|---|------|----------------|
| 1 | Ship a reference DMS plugin **in the emeet-pixyd repo** (`contrib/quickshell/`) | First-class artifact vs "documented example only". |
| 2 | ADR for the `/api/status` public contract | Repo uses ADRs for interface/decision commitments; a "stable public interface" deserves one. |
| 3 | Live smoke test (`nix run` + `curl /api/status`) | Tests use httptest only; no end-to-end proof. |
| 4 | `nix build` (production gate) | AGENTS says `nix build` is the preferred gate; I only ran `go build`. |
| 5 | `nix flake check` / treefmt on the changed tree | CI gate not run locally. |
| 6 | CHANGELOG entry in SystemNix | That repo documents plugin changes too. |
| 7 | Fuzz target for the new JSON encoder path | Not warranted (no parsing), but noted. |
| 8 | Prometheus metric for `/api/status` hits | Optional; other read endpoints aren't counted either. |

---

## d) TOTALLY FUCKED UP / RED FLAGS

Nothing catastrophic, but be honest:

1. **I did not ask which interpretation was intended.** "Support quickshell.nix"
   has at least three readings (daemon endpoint / in-repo widget+module / edit
   the SystemNix file). I picked one and ran. It is the *most* plausible, but it
   was a guess.
2. **The full test suite is red while I worked** — captured:
   - `TestHandleSpeedCommand_SetMotorSpeedFailureCountsTowardBreaker` — **pre-existing** (reproduced at `938f6e5`); looks like the "passes on CI, fails on hardware-bearing machines" class AGENTS warns about (the run logs `found PIXY device ... hidraw9`).
   - `TestSSEEndpoint_SurvivesServerWriteTimeout` — from **another session's uncommitted `sse_test.go`**.
   - A transient `zz_wt_test.go` (`TestWriteTimeoutEnforced`) appeared and vanished mid-run — another agent writing files concurrently.
   I reported "unrelated" but could not *prove* my diff is innocent of the SSE one (it obviously is — I only added a GET JSON route — but proof beats assertion).
3. **Concurrent sessions mutating the repo** made verification unreliable and
   means the auto-commit daemon committed my work under opaque
   `chore: auto-commit N changed file(s)` messages — history for this feature is
   effectively unreviewable.
4. **The SystemNix widget is unverified at runtime.** If a theme token is wrong,
   the widget throws at load and the shell loses the pill. I should have at
   least checked DMS's `Theme.qml` token list.

---

## e) WHAT WE SHOULD IMPROVE — answering the three prompts

### What did I forget?
- Run the daemon and `curl /api/status` for a real end-to-end proof.
- Verify the DMS `Theme` tokens I used actually exist.
- Keep the guide widget and the deployed widget byte-identical.
- Add a battery-present test.
- Confirm the interpretation of the request before implementing.
- Note the auto-commit-mangled history in the report *at the time*, not after.

### What could I have done better?
- **Ask first** on a genuinely ambiguous, cross-repo request (the rules permit it for "multiple valid approaches with big tradeoffs").
- Run `nix build` (repo's preferred gate) rather than only `go build`.
- Give the endpoint an ADR and a richer, versioned contract from the start.
- Prove the unrelated failures by running the suite on a clean worktree at the same moment, not by reasoning.

### What could I still improve?
- Ship the widget in-repo (`contrib/quickshell/`) so it isn't trapped in SystemNix.
- Extend the contract (PTZ, trackMode) behind a documented stability promise.
- Add a benchmark and a battery-present test.
- Add a small `scripts/` or docs mention so the endpoint is discoverable from the CLI reference.

---

## f) UP TO 50 THINGS TO DO NEXT (ranked roughly by value)

1. **Confirm the request interpretation** — daemon endpoint vs in-repo module vs SystemNix-only. (blocks further scope)
2. Live smoke test: `nix run` → `curl -s localhost:8090/api/status` → eyeball JSON.
3. `nix build` to exercise the production derivation.
4. Run `nix flake check --no-build` + treefmt locally.
5. Verify DMS `Theme.error` / `Theme.warning` exist (grep upstream `Theme.qml`); otherwise swap tokens.
6. Make the guide widget and the SystemNix widget identical (single source of truth).
7. Add a battery-present test (simulator + `withNoop*`) asserting the `battery` field appears.
8. Add a benchmark for `handleStatusJSON` (mirror `BenchmarkGetWebStatus`).
9. Write `docs/adr/2026-10-01_status-api-contract.md` declaring the field names stable.
10. Ship `contrib/quickshell/` in emeet-pixyd (plugin.json + QML + README).
11. Add a CLI hint: `emeet-pixy --help`/README pointer to `/api/status`.
12. Document `/api/status` in the CLI Reference / a new "HTTP API" doc page.
13. Add `trackMode` and `speeds` to the contract (typed, additive).
14. Consider `pan`/`tilt`/`zoom` (cost: a V4L2 read; gate behind a query flag `?ptz=1`).
15. Add `error` and `lastSynced` fields for parity with `webStatus`.
16. Add a JSON-schema or golden file pinning the exact field set (regression guard).
17. Add CORS/`Content-Type` assertion test.
18. Pin `Cache-Control: no-store` on `/api/status` (widgets must not cache).
19. Add an `ETag`/conditional-get option for high-frequency pollers.
20. Add a Prometheus counter for `/api/status` requests.
21. Document recommended poll interval (8 s in the widget) and rationale.
22. Add a Waybar↔Quickshell parity note (same state, two transports).
23. Add click-action wiring example (QML `MouseArea` → `curl -X POST /api/toggle-privacy`).
24. Verify the widget re-fire path (daemon restart) shows `off` then recovers.
25. Add an offline→online transition note in the guide.
26. Consider SSE for widgets (`/api/events`) instead of polling — document tradeoff.
27. Add a `systemnix-camera` plugin README in SystemNix.
28. Add SystemNix CHANGELOG entry for the widget change.
29. Add a DMS plugin test/tooltip if the plugin API supports one (verify `PluginComponent` props).
30. Surface `device`/`model` in the widget tooltip if the API allows.
31. Add a `/api/status?fields=` projection to keep payloads tiny.
32. Version the endpoint (`/api/v1/status`) if the contract may break.
33. Add `omitempty` review: confirm json/v2 `omitzero` semantics for all bool/typed fields.
34. Confirm `pixy.CameraState`/`AudioMode`/`AutoMode` marshal as the raw strings widgets expect.
35. Add a test that `camera` is never empty (widget fallback safety).
36. Add a migration note if any consumer depended on `/api/health` shape.
37. Add the endpoint to `website/.../metrics.mdx` cross-reference (health vs status).
38. Ensure `website` CI page-count guard still satisfied (21 ≥ 19 — yes).
39. Run `astro check` (strict TS gate) locally.
40. Add an `og:image`/description for the new guide page (SEO pass, TODO #175 territory).
41. Add the guide to the landing page "integrations" section if one exists.
42. Benchmark JSON payload size.
43. Add an integration test tagged `-tags=integration` that hits a running daemon.
44. Consider consolidating `healthResponse` + `statusResponse` field naming.
45. Add a lint guard/`exhaustruct` check for future response structs.
46. Re-run the full suite on a clean worktree to establish a true green baseline.
47. Investigate the pre-existing motor-breaker test failure (is it hardware-dependent? `newTestDaemon` noop gap per AGENTS).
48. Coordinate with the concurrent session — two agents in one repo is corrupting verification and history.
49. Fix the auto-commit history problem (opaque messages) — TODO #176 territory already tracks gating it on green builds.
50. Re-read this report's §f and file the survivors into `TODO_LIST.md`.

---

## g) QUESTIONS I CANNOT ANSWER MYSELF (max 3)

1. **What did "support quickshell.nix" mean?** (a) the daemon endpoint I built,
   (b) a Home-Manager/DMS **module shipped by emeet-pixyd** that your
   `quickshell.nix` imports, or (c) just fix the SystemNix widget? I built (a)
   and did (c); if you meant (b), that is net-new work.
2. **Should the DMS widget live in the emeet-pixyd repo** (`contrib/quickshell/`,
   first-class artifact) or stay a SystemNix-private plugin that we merely
   document? This decides whether I add shipped QML to the daemon repo.
3. **How much should `/api/status` expose?** Lean (current: no PTZ) vs rich
   (PTZ, trackMode, speeds — at the cost of a V4L2 read per poll). Which do you
   want as the stable contract?

---

## Verification summary (this session)

| Gate | Result |
|------|--------|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `golangci-lint run --timeout 2m ./...` | **0 issues** |
| New tests (`TestWeb_Status*`) | pass |
| Full `go test ./...` | **red** — 1 pre-existing (motor breaker) + 1 from another session's `sse_test.go` |
| `website` `pnpm run build` | pass — 21 pages, CSP patched |
| `nix build` | **not run** |
| SystemNix widget runtime | **not run / not linted** |
