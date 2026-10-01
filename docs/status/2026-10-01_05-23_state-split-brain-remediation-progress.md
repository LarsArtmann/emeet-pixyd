# State / Split-Brain Remediation — Status Report

**Written:** 2026-10-01 05:23 CEST
**Plan:** `docs/planning/2026-10-01_04-24_SUPERB-state-split-brain-remediation-plan.md`
**Audit:** `docs/status/2026-10-01_04-21_state-bug-split-brain-audit.md`
**Author:** Crush (executing agent)

> **Read this first.** The work is **mid-implementation**, not finished. The
> production build compiles; the **test suite does not compile yet**, and no
> behavior has been verified end-to-end. Treat everything below as "in flight".

---

## 0. Honest snapshot (the truth right now)

```
$ GOWORK=off go build ./...          -> exit 0   (production compiles)
$ GOWORK=off go test -run x ./...    -> BUILD FAILED
    power_test.go:77: undefined: errors
    zz_repro_test.go:24/33: d.hadPersistedState undefined
$ git status                         -> 14 files modified, tree DIRTY
```

- The test binary does **not** compile (1 missing import, 1 stale scratch test).
- `templates.templ` was edited but `templ generate` has **not** been re-run, so
  `templates_templ.go` is stale relative to the `.templ` source.
- The **auto-commit daemon** has been committing my changes in chunks
  (`1a787a1` ... `8c7b0ee`); several earlier edits are already in `HEAD`, the
  latest 14 files are uncommitted. History is now noisy (`chore: auto-commit N
  changed file(s)`), not a clean feature commit.

---

## 1. What I set out to do

Execute the whole SUPERB remediation plan: verify and fix the 11 state /
split-brain findings from the audit, and the 97 fine-grained tasks (F1-F97)
grouped under 24 medium tasks (M1-M24), without *verschlimmbessern*.

---

## 2. What is DONE (implemented, **unverified**)

Present tense is deliberately weak: "written" does not equal "proven". None of
these has a green test run behind it yet.

| ID | Task | State | Evidence |
|----|------|-------|----------|
| M1 | Baseline + intent cross-check | **done** | baseline was green before edits: `go test -race` ok, `golangci-lint` 0 issues; grep of `TODO_LIST`/`CHANGELOG`/`docs/adr` ran (no hits for `hadPersistedState`/`syncState`/`powerStatus`; `CHANGELOG` only documents the breaker policy) |
| M2 | Reproduce Findings 1 & 2 | **done (red proven)** | temp test printed `BUG REPRODUCED: camera after probe miss = "offline", want tracking` and `hadPersistedState still false after saving user intent` |
| M3 | Decouple connectivity from `CameraState` | **written** | `applyProbeResultLocked` no longer writes camera; `CameraState.ValidDesired()`; `State.Camera` documented as intent-only; `CurrentSchemaVersion` 1->2; `loadState` normalizes legacy `offline`->`privacy`; `devicePresence`/`displayCamera` projection |
| M4 | Derive persisted-intent truth | **written** | `persistedIntent atomic.Bool` set by `saveState()` on success and by `loadState()` at startup; `reconcileOnDeviceAppear` reads it |
| M5 | Serialize `syncState` under `hidMu` | **written** | new `syncState` takes `hidMu`; `syncStateLocked` for the reconcile path; added missing nil-HID guard (also closes a latent nil-deref panic on a video-only partial device) |
| M6 | `powerStatus` failure caching | **written** | `powerCacheEntry{reading, available}`; failures cached for `powerFailureCacheTTL` (10s), successes for `powerCacheTTL` (1m) |
| M7 | Unify device-appear paths | **partly** | `autoManage` re-probe now calls `reconcileOnDeviceAppear` (startup + uevent already did); not yet extracted into a single named entry helper |
| M10 | Online vs Controllable semantics | **partly** | `webStatus.Controllable`, `statusResponse.Controllable`; template gates HID controls on `Controllable`; PTZ/preview still `Online` |
| M11 | Canonical status projection | **partly** | single `displayCamera(online, desired)` used by `getStatus`, `waybar`, `getWebStatus`, `handleHealth`, metrics - but no `projectStatus()` struct as the plan named |
| M12 | Invalidate caches on device change | **written** | `applyProbeResultLocked` invalidates `ptzCache`+`powerCache` on presence change |
| M15 | Offline metric gauge | **written** | `updateMetrics(state, online)` projects camera and now emits the `offline` label |
| M16 | Retire ghost `ProcessInspector` | **written** | interface + `procInspector` + `noopProcessInspector` + fake removed; `isCameraInUse` dep kept |
| M17 | InCall persistence | **partly** | decided: keep persisted for crash-visibility, but **reset to false on daemon start** in `NewDaemon`; the "move it out of `State`" refactor was rejected (see section 7) |
| M23 | `lastFrameCache` audit | **written** | added `Clear()`; called when the camera goes offline |

---

## 3. What is PARTIALLY done (started, not finished)

- **Test migration** for changed behavior - not completed. Known-broken tests:
  `probe_hidraw_test.go:267` (asserts `state.Camera == offline` after a probe
  miss), `auto_manage_test.go:62` (same assertion via `autoManage`),
  `power_test.go:77` (missing `errors` import), `zz_repro_test.go` (scratch,
  must be replaced by the real regression suite).
- **`templ generate`** - required after `.templ` edits; not run.
- **M7** - reconcile is now called from all three paths, but there is no single
  extracted `handleDeviceAppear(ctx, probe)` entry, and each path is not yet
  covered by its own test (F39).
- **M10/M11** - fields exist and are used, but the plan's fuller "one canonical
  `projectStatus()`" abstraction was not built; I chose a smaller pure function
  instead.

---

## 4. What is NOT started

- **M8** - property test: intent survives probe/sync/replug sequences.
- **M9** - HID concurrency proof (simulator "no overlapping request" assert,
  `sync`+`autoManage` race test).
- **M13** - decide whether `sync` should reconcile `Speeds`/`TrackMode` (my
  position: it cannot until hardware readback is verified; needs documenting).
- **M14** - breaker reset policy (my analysis says the current behavior is
  *intentional* per `CHANGELOG`; needs documenting, not changing).
- **M18** - lock-contract audit + lock table in `AGENTS.md`.
- **M19** - file findings/tasks into `TODO_LIST.md`.
- **M20** - ADR for the connectivity/intent decoupling.
- **M21** - update `AGENTS.md` + `docs/DOMAIN_LANGUAGE.md`.
- **M22** - retro intent-check (prune non-bugs).
- **M24** - final verification (build, race suite, lint, `templ generate`
  diff, re-audit).
- The proper regression test file (`state_split_brain_test.go`) is **not yet
  written**; only the throwaway `zz_repro_test.go` exists.

---

## 5. What is TOTALLY FUCKED UP (my own mistakes this session)

1. **I left the tree non-compiling for tests.** I paused with `power_test.go`
   missing an import and a scratch repro test in the root. A green build was
   the checkpoint I skipped.
2. **I trusted the plan's framing too readily.** The plan assumed a data-model
   change needing a schema bump; my actual fix keeps the JSON shape identical
   and only reinterprets `camera`. I bumped the version anyway for honesty,
   which will log a spurious "schema mismatch" once per existing user - a cost
   I introduced without a migration need.
3. **`zz_repro_test.go` is litter.** It never had `t.Parallel()`, it hard-codes
   the old field name, and it breaks the build until deleted. It should have
   been replaced by the real test immediately after proving red.
4. **I did not re-run `templ generate`,** so `templates_templ.go` no longer
   matches `templates.templ`. The UI change (gate controls on `Controllable`)
   is therefore half-wired.
5. **I let the auto-commit daemon shred my history.** Eight `chore: auto-commit`
   commits interleave my logically-grouped edits; bisecting a bug will be
   worthless. I should have worked on a branch or committed atomically as I
   went.
6. **I accidentally created a junk file.** A malformed tool call wrote a file
   literally named `{"command": "cd ` into the repo root. I found and
   `trash`ed it, but it existed for several minutes and is exactly the kind of
   mess I should never produce.
7. **I did not verify half of what I "wrote".** M3/M4/M5/M6/M7 are the highest
   stakes changes in the project (state model + lock discipline) and **none of
   them has a passing test**. Claiming them "done" would be trophy-case
   behavior.

---

## 6. Diff surface (14 uncommitted files; more already auto-committed)

| File | Change |
|------|--------|
| `auto.go` | `updateMetrics(state, online)`; autoManage re-probe now reconciles |
| `commander.go` | removed `noopProcessInspector`, dropped `pixy` import |
| `deps.go` | removed ghost `procInspector` field |
| `process.go` | removed `ProcessInspector` interface + `procInspector` impl |
| `main.go` | `hadPersistedState` -> `persistedIntent atomic.Bool`; reset `InCall` on start |
| `metrics.go` | `updateMetrics(state, online)`; offline gauge label |
| `power.go` | negative-cache entry type + dual TTLs |
| `power_test.go` | negative-cache assertions (needs `errors` import) |
| `main_test.go`, `fake_device_test.go`, `pixy_simulator_test.go` | fake/import cleanup |
| `reconcile_test.go`, `speed_wiring_test.go`, `handlers_test.go` | migrated to new API |
| *(already in HEAD)* `device.go`, `probe.go`, `state.go`, `handlers.go`, `waybar.go`, `web_types.go`, `cache.go`, `templates.templ`, `internal/pixy/pixy.go` | the core decoupling + projections |

---

## 7. Design decisions I made (and why)

- **Connectivity != intent, decided via projection, not a new persisted field.**
  `State.Camera` means *desired mode only*; connectivity is observed via
  `devicePresence` and projected with `displayCamera`. This dissolves findings
  1, 2, 6 and keeps the JSON shape compatible.
- **Kept `StateOffline` in the enum** as a projection value; removed it from
  what gets *persisted* (legacy files normalized on load). Removing the enum
  value would have rippled through waybar/metrics/tests for no gain.
- **`InCall` stays in `State`** (JSON unchanged) but is reset at startup. Moving
  it to a runtime field collides with `json.RejectUnknownMembers(true)` and
  would make old state files rejected wholesale - a data-loss regression.
- **Breaker reset (M14) is intentional.** `applyProbeResultLocked`'s
  `hidFailCount = 0` is exactly the documented "config-send failures re-probe
  and reset; only commit failures accumulate" policy. I will document, not
  rewire it.
- **`sync` cannot reconcile speeds/track-mode (M13)** until hardware readback
  is verified (#166); it should be documented deliberately.

---

## 8. Verification state (the only thing that matters)

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go test` compile | **FAILS** (`errors` import; `zz_repro_test.go`) |
| `go test -race` | not run since edits |
| `golangci-lint` | not run since edits |
| `templ generate` diff | not run (stale!) |
| `nix build` | not run |
| Re-audit (no regressions) | not run |

**Nothing is verified. The "done" column in section 2 means written, not proven.**

---

## 9. Fastest path to green (proposed, not yet executed)

1. Delete `zz_repro_test.go`; add `errors` import to `power_test.go`.
2. Run `templ generate`; run full `go test -race`; fix the two offline-assertion
   tests (`probe_hidraw_test.go:267`, `auto_manage_test.go:62`).
3. Write the real regression suite (M2/M8/M9): intent clobber, persist-across-
   replug, negative-cache, sync+autoManage race.
4. Then M13/M14/M18/M19/M20/M21/M22 (docs/ADR/TODO), then M24 verification.

---

## 10. Three questions I still cannot answer without you

1. **Schema bump:** keep `CurrentSchemaVersion = 2` (honest, one spurious
   warning per existing user), or revert to 1 (silent, since the JSON is
   byte-compatible)?
2. **`InCall`:** is reset-on-start enough, or do you want the full "move it out
   of persisted `State`" refactor (with the `RejectUnknownMembers` migration
   cost)?
3. **M13/M14:** confirm "document as intentional" is the disposition you want,
   or do you want code changes despite the `CHANGELOG`-documented intent?

---

## 11. Bottom line

The **core fix is written** - camera intent is no longer clobbered by
connectivity and no longer persisted as `offline`; `sync` is now lock-safe;
the battery HID storm is cached; the ghost abstraction is gone. But the
**test suite does not compile, `templ generate` is stale, and nothing is
verified.** The honest status is **about 60% of the plan implemented, 0%
verified, tree dirty, history shredded by the auto-committer.**

I am stopping here and waiting for your instructions, per the request.