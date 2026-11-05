# Status Report — State-Bug / Split-Brain Audit (Session 2026-10-01 ~04:1x CEST)

**Written:** 2026-10-01 04:21 CEST
**Scope:** This session only. The entire session was a **read-only audit**: find all state bugs, especially split-brains. **No code was changed, no tests run, no fixes applied.** The deliverable was the findings list reported in chat.
**End state:** Audit complete (11 findings, 4 high-value). Working tree untouched by me (auto-commit daemon may hold the report). Nothing pinned by a test; several findings are reasoned, not reproduced.

---

## Executive summary

I was asked to find all possible state bugs, especially split-brains. I loaded the `full-code-review` and `brutal-self-review` skills, read the state-holding surface (`main.go`, `state.go`, `device.go`, `process.go`, `auto.go`, `probe.go`, `commands.go`, `handlers.go`, `ptz.go`, `motor.go`, `power.go`, `circuitbreaker.go`, `cache.go`, `sse.go`, `hid.go`, `waybar.go`, `internal/pixy/pixy.go`, `web_types.go`, `deps.go`), and grepped every `d.state.* =` write site.

The central conclusion: **one struct (`pixy.State`) and one enum (`CameraState`) carry four different concepts** — user intent, observed hardware mode, device online/offline status, and session state. Almost every finding flows from that conflation. I produced 11 findings; I classify 4 as real bugs (the rest as design debt). **I did not fix or reproduce any of them**, and I did not cross-reference `TODO_LIST.md`/ADRs to check whether some are deliberate — that is the honest gap in this session.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| ~~a1~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Skill loading per rules: viewed `full-code-review/SKILL.md` and `brutal-self-review/SKILL.md` before touching the task~~ | ~~tool transcript~~ |
| ~~a2~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Mapped every state store: `d.state`, `ptzCache`, `lastFrame`, `powerCache`, debounce counters, `hidFailCount`, probe fields (`videoDev`/`hidrawDev`/`model`/`unsupportedHint`/`hidDev`), `hadPersistedState`, `lastSyncedAt`, `autoError`~~ | ~~full read of the files listed above~~ |
| ~~a3~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Exhaustive write-site inventory: grepped `d\.state\.[A-Za-z]+\s*=` and `InCall` (72 + 76 matches) to catch every mutation of persisted belief~~ | ~~grep output~~ |
| ~~a4~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Finding 1 (Camera overloaded with offline) traced across all three device-appear paths (startup, uevent, autoManage re-probe)~~ | ~~`probe.go:322-329`, `main.go:218-226`, `main.go:322-331`, `auto.go:98-108`~~ |
| ~~a5~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Finding 2 (`hadPersistedState` stale flag) confirmed set-once / never-updated~~ | ~~`main.go:121`, only other reads in `device.go:281,285` + tests~~ |
| ~~a6~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Finding 3 (`sync` HID I/O without `hidMu`) confirmed by tracing `handleCommand` → `handleQueryCommand` → `syncState` → `queryHIDState` → `SendRecv`, none taking `hidMu`~~ | ~~`commands.go:74-75,185-186`, `device.go:182-223`, `hid.go:317-351`~~ |
| ~~a7~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Finding 4 (`powerStatus` failure-caching) confirmed code contradicts its own doc~~ | ~~`power.go:39-55` vs comment `power.go:39-41`; call sites `handlers.go:101,197`, `waybar.go:80`~~ |
| ~~a8~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Findings 5–11 triangulated against the read paths that disagree (getStatus vs waybar vs web; ptzCache vs fresh read; sync's field coverage; metrics gauges)~~ | ~~`handlers.go`, `device.go`, `waybar.go`, `ptz.go`, `metrics.go`~~ |
| ~~a9~~ done — executed same day (remediation; CHANGELOG [0.5.0] Fixed L82-86) | ~~Reported all 11 findings with `file:line` references and severity grouping, split-brain findings first~~ | ~~chat deliverable~~ |

## b) PARTIALLY DONE

| # | Item | State | Remaining | Effort |
|---|------|-------|-----------|--------|
| ~~b1~~ done — reproduced (M2); ADR written; MaxInFlight proof; snapshot projection | ~~Findings reproduced~~ | ~~NONE reproduced — all are static-read conclusions~~ | ~~write a failing test per real bug (esp. #1, #3) to prove it before fixing~~ | ~~M~~ |
| ~~b2~~ done — reproduced (M2); ADR written; MaxInFlight proof; snapshot projection | ~~Cross-reference against `TODO_LIST.md` / `CHANGELOG.md` / ADRs~~ | ~~NOT done — I read `AGENTS.md` context but never checked whether #5 (hidden intent) or the breaker reset are deliberate~~ | ~~grep TODO_LIST/ADRs for `hadPersistedState`, `syncState`, `powerStatus`, breaker semantics~~ | ~~S~~ |
| ~~b3~~ done — reproduced (M2); ADR written; MaxInFlight proof; snapshot projection | ~~Finding 3 harm analysis~~ | ~~Reasoned (mis-paired HID responses silently accepted via interface-byte routing) but not demonstrated against the `pixySimulator`~~ | ~~simulator-based concurrency test to show response mis-pairing~~ | ~~M~~ |
| ~~b4~~ done — reproduced (M2); ADR written; MaxInFlight proof; snapshot projection | ~~`getStatus` vs `waybar` disagreement~~ | ~~Described as an edge case (hidraw-only device + successful track command); not traced to a concrete reachable state~~ | ~~verify with `withFakeDevices` harness~~ | ~~S~~ |
| ~~b5~~ done — reproduced (M2); ADR written; MaxInFlight proof; snapshot projection | ~~Formal report~~ | ~~Chat-only findings; no `docs/reviews/*.html` per the `full-code-review` skill's output contract~~ | ~~decide whether to produce the styled HTML review~~ | ~~S~~ |

## c) NOT STARTED (newly surfaced this session)

| # | Item | Why | Priority |
|---|------|-----|----------|
| ~~c1~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Fix Finding 1 — separate offline/online from `Camera` (e.g., a distinct field or an explicit `hasIntent` guard in `applyProbeResultLocked`)~~ | ~~not started~~ | ~~High~~ |
| ~~c2~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Fix Finding 2 — update `hadPersistedState` (or replace with "state file exists" derived at the point of use)~~ | ~~not started~~ | ~~High~~ |
| ~~c3~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Fix Finding 3 — take `hidMu` in `syncState` / route all HID through one serialized path~~ | ~~not started~~ | ~~High~~ |
| ~~c4~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Fix Finding 4 — cache authoritative failures per the doc, or correct the doc~~ | ~~not started~~ | ~~Medium~~ |
| ~~c5~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Unify the three device-appear paths behind one reconcile entry point (Finding 1/2 cousin)~~ | ~~not started~~ | ~~Medium~~ |
| ~~c6~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Decide `Online` semantics (video-present vs controllable) and align `getStatus`/`waybar`/`health` (Finding 5)~~ | ~~not started~~ | ~~Medium~~ |
| ~~c7~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Invalidate `ptzCache` on probe/device change (Finding 7)~~ | ~~not started~~ | ~~Low~~ |
| ~~c8~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Add `Offline` gauge / stop deriving gauges from the overloaded Camera (Finding 11)~~ | ~~not started~~ | ~~Low~~ |
| ~~c9~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Retire the ghost `ProcessInspector`/`procInspector` field or use it (Finding 10)~~ | ~~not started~~ | ~~Low~~ |
| ~~c10~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~File any of the above into `TODO_LIST.md`~~ | ~~not started~~ | ~~Medium~~ |
| ~~c11~~ done — resolved — see the report’s inline verdicts and CHANGELOG history | ~~Run `GOWORK=off go test -race -count=1 ./...` to baseline before any fix~~ | ~~not started~~ | ~~High (pre-fix gate)~~ |

## d) TOTALLY FUCKED UP (honest)

~~1. **I fixed nothing and proved nothing.** The session's output is static analysis only. Several claims (especially Finding 3's corruption mechanism and Finding 5's reachability) are reasoned from code, not reproduced. A finding that is not reproduced is a hypothesis, and I presented them with the confidence of facts.~~ done — STALE-RESOLVED: reproduced + fixed same day
~~2. **I did not check whether some "bugs" are intentional.** This project documents decisions heavily (AGENTS.md, ADRs, TODO_LIST). The breaker reset (`applyProbeResultLocked` zeroing `hidFailCount`) is explicitly described in AGENTS.md as intended ("Config-send failures trigger re-probe; only commit failures accumulate"). I flagged a related reset as Finding 10 without first checking the docs that already explain the design — that risks re-discovering a decision as a defect.~~ done — dispositions settled
~~3. **I ignored the skill's own output contract.** `full-code-review` mandates a styled HTML report at `docs/reviews/` and a Pareto plan; I delivered chat prose instead. Either the skill should not have been loaded for a focused question, or I should have produced the artifact. I did neither cleanly.~~ **Superseded — process notes.**
~~4. **I loaded two overlapping skills** (`full-code-review` + `brutal-self-review`) and then followed neither procedure. That is a process miss, not a code one.~~ **Superseded — process notes.**

## e) WHAT WE SHOULD IMPROVE

~~1. **Reproduce before reporting.** Every finding that claims misbehavior should have a failing test or a concrete reachable state before it is presented as a bug. Static-read findings must be labeled as such.~~ done — executed (M1/M2; ADR; lock table)
~~2. **Check intent before declaring a defect.** Grep `TODO_LIST.md`, `CHANGELOG.md`, and `docs/adr/` for each finding's keywords first; the project's own docs are the cheapest disambiguator between "bug" and "deliberate tradeoff".~~ done — executed (M1/M2; ADR; lock table)
~~3. **One `Camera`-shaped hole, one concept.** The root cause across findings 1, 2, 5, 6, 11 is that `CameraState` conflates intent with connectivity. A single data-model change (separate `online`/`controllable` from `desiredMode`) would dissolve several findings at once — a Pareto item, not eleven patches.~~ done — executed (M1/M2; ADR; lock table)
~~4. **`d.mu` is over-broad but `hidMu`/`v4l2Mu` are under-applied.** The concurrency story is only as strong as its weakest entry point; every HID caller must be audited for the lock it forgot (Finding 3).~~ done — executed (M1/M2; ADR; lock table)
~~5. **Report only what was actually done.** This session did analysis; the report template's "FULLY DONE / code changed" framing pushed me toward over-claiming. Keep the honest line: audit = findings, nothing shipped.~~ done — executed (M1/M2; ADR; lock table)

## f) UP TO 50 THINGS TO GET DONE NEXT

**Verification of this session's findings (do first)**
~~1. Baseline: `GOWORK=off go test -race -count=1 ./...` and `golangci-lint run` (expect green; nothing changed).~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~2. Write a failing test proving Finding 3: concurrent `syncState` + `autoManage` against `pixySimulator`, assert response mis-pairing.~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~3. Write a test proving Finding 1: simulate probe miss, assert `d.state.Camera` and the persisted file retain the user's mode.~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~4. Write a test proving Finding 2: fresh install → user sets mode → replug → assert mode survives.~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~5. Test-drive Finding 5: hidraw-only device + `track` command, compare `getStatus` vs `waybarOutput`.~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~6. Grep `TODO_LIST.md`/`CHANGELOG.md`/`docs/adr/` for each finding to separate intent from defect.~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)

**High-priority fixes (real bugs)**
~~7. Decouple offline/connectivity from `Camera` intent (Finding 1) — data-model change.~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~8. Make `hadPersistedState` reflect reality at use-time (Finding 2).~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~9. Serialize `syncState` HID access under `hidMu` (Finding 3).~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~10. Cache authoritative `powerStatus` failures, or fix the doc (Finding 4).~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~11. Unify the three device-appear paths into one reconcile entry (Finding 1/2).~~ done — executed (state_split_brain_test.go; persistedIntent; hidMu; power cache; reconcile unify; Online/Controllable; gauge; ghost retired; #182 filed; f12 write-side guard → TODO_LIST #196)
~~12. Add a compile-time guard so `StateOffline` can't be persisted as user intent (or exclude it from `saveState`).~~ **NOT-DO — TODO_LIST #196 (saveState guard).**
~~13. Make `sync` reconcile `Speeds`/`TrackMode`, or document why it must not.~~ **NOT-DO — TODO_LIST #182.**

**Medium**
~~14. Define `Online` vs "controllable" and align `getWebStatus`, `handleHealth`, `getStatus` (Finding 5).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~15. Reconcile `getStatus` (forces offline on no-video) with `waybarOutput` (uses raw belief).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~16. Invalidate `ptzCache` (and `powerCache`) on device appear/removal.~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~17. Distinguish PTZ read paths: one authoritative cache vs fresh reads.~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~18. Give the circuit breaker a reset policy that doesn't get erased by unrelated probe success (Finding 10).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~19. Add an `offline` gauge / decouple metrics from `Camera` (Finding 11).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~20. Retire or wire the ghost `ProcessInspector` (Finding 10).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~21. Decide whether `InCall` should be persisted at all (Finding 9).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~22. Audit every HID caller for missing `hidMu` (systematic, not just `sync`).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~23. Audit every v4l2 caller for missing `v4l2Mu` (`getStatus`, `getWebStatusWithPTZ` `parsePTZ`).~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)
~~24. Add a lock-contract test/doc table listing each entry point and the locks it must hold.~~ done — executed (ADR projection; breaker documented+tested; gauge; ghost; InCall decision; lock tables)

**Hygiene / process**
~~25. Decide the scope boundary of `full-code-review` output vs a quick audit; stop loading unused skills.~~ **Superseded — process notes; f27 done.**
~~26. Produce the HTML review artifact if a full review is actually wanted.~~ **Superseded — process notes; f27 done.**
~~27. Tile findings into `TODO_LIST.md` with IDs.~~ **Superseded — process notes; f27 done.**
~~28. Record the "conflated Camera" root cause in `ROADMAP.md` as a data-model item.~~ done — ADR + AGENTS + typed pair + projection + reports
~~29. Add an ADR if the Camera/intent decoupling is chosen.~~ done — ADR + AGENTS + typed pair + projection + reports
~~30. Update `AGENTS.md` Architecture bullet on state if the model changes.~~ done — ADR + AGENTS + typed pair + projection + reports
~~31. Add a `docs/DOMAIN_LANGUAGE.md` entry distinguishing intent vs connectivity vs session.~~ **NOT-DO — DOMAIN_LANGUAGE intent/connectivity entries — verified present (Intent/Connectivity rows).**
~~32. Verify auto-commit daemon committed this report.~~ done — ADR + AGENTS + typed pair + projection + reports
~~33. Run `templ generate`/`nix fmt` only if code changes land (not needed for a report).~~ done — ADR + AGENTS + typed pair + projection + reports

**Longer arc**
~~34. Introduce a typed `Connectivity`/`Controllable` pair in `internal/pixy` and thread it through probe → status → UI.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~35. Replace the per-surface status aggregation (`getStatus`/`waybar`/`getWebStatus`/`healthResponse`) with one canonical projection.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~36. Centralize all HID transport behind a single serialized client (the simulator already models the protocol; production should match).~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~37. Property test: any sequence of probe/sync/replug must never lose a persisted non-offline Camera.~~ done — regression suite covers; exact property subsumed
~~38. Property test: no two HID requests may be in flight simultaneously.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~39. Add metrics for HID request concurrency violations (defensive).~~ **NOT-DO — concurrency metrics never added (micro).**
~~40. Review `lastFrameCache` invalidation on stream stop/device change.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~41. Review `autoError` vs `state` lifecycle consistency.~~ **NOT-DO — autoError lifecycle unchecked (micro).**
~~42. Confirm `debounceInUse`/`debounceIdle` can't be observed inconsistent with `InCall`.~~ **NOT-DO — debounce/InCall unchecked (micro).**
~~43. Check `saveState` callers all hold `d.mu` (they do; add a comment/assert).~~ **NOT-DO — saveState d.mu comment unchecked (micro).**
~~44. Consider deriving `Online` in exactly one helper and reusing it everywhere.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~45. Add a golden test for the disconnect→replug→status matrix.~~ **NOT-DO — golden replug matrix never built (micro).**
~~46. Re-audit `applyProbeResultLocked` side effects (it mutates Camera, hidDev, hidFailCount) — split into pure vs mutating parts.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~47. Document the intended device-appear matrix explicitly (3 paths today).~~ **NOT-DO — appear matrix doc never added (AGENTS carries behavior).**
~~48. Delete dead abstractions surfaced (ghost fields) in one sweep.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~49. Add a `nolint`-free note if `procInspector` is removed.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)
~~50. Re-run this audit after fixes to confirm no split-brain regressions.~~ done — executed (devicePresence; snapshot; serialization; property-ish regression; metrics; cache; appear matrix; ghost; re-audit)

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (max 3)

~~1. **Intent:** Is `syncState` running HID I/O without `hidMu` deliberate (a documented belief that HID reads are safe concurrently), or an oversight? I can see the code but not the decision — and the answer changes Finding 3 from "bug" to "documented assumption".~~ done — ANSWERED by the remediation (fixes + ADR + #182)
~~2. **Design:** Should unplugging the camera ever modify persisted user intent? Concretely: should `CameraState` keep carrying "offline", or should connectivity become a separate field (dissolving findings 1, 2, 5, 6, 11 at once)? This is a data-model decision only you can set.~~ done — ANSWERED by the remediation (fixes + ADR + #182)
~~3. **Direction:** Do you want me to (choose one) reproduce-and-fix the 4 real findings now, or file all 11 into `TODO_LIST.md` and stop?~~ done — ANSWERED by the remediation (fixes + ADR + #182)
