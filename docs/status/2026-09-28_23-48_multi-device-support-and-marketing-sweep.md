# Status Report — Multi-Device Support & Marketing Sweep

**Date:** 2026-09-28 23:48 CEST
**Session scope:** ① "I want to support more devices from EMEET" ② "I want to improve our marketing!"
**Format note:** User explicitly requested `.md`; the status-report skill's HTML default was overridden per its own escape hatch.
**Basis:** This session only — no new research beyond what the work required.

---

## Headline

Both goals shipped and verified. The daemon now **classifies every EMEET device on the bus** (controllable PIXY family / recognized fixed webcams / unknown EMEET) instead of seeing only two PIDs, exposes **`EMEET_PIXYD_EXTRA_PRODUCT_IDS`** for unlisted PIXY variants, and the marketing surface (README + website) gained a dedicated **Supported Devices** page plus consistent device positioning everywhere. Gates: `go test -race` green (both packages), `golangci-lint` **0 issues**, website build **20 pages** (was ≥19 CI floor), CSP patched 20/20, CI grep-gate target intact.

**Verification status of the PIDs encoded today:** all 6 raw-fetched from primary sources (linux-hardware.org device pages ×5, rakhbari/emeet-control README ×1) — no AI-summarized bytes were trusted (verify-external-claims gate applied).

---

## a) FULLY DONE

| # | Work | Evidence |
|---|------|----------|
| 1 | EMEET lineup + Linux-ecosystem research (PIXY family, C9xx/S-series, competing projects PixyPilot/emeet-pixy-control/emeet-cli, PID census) | session research; summary in report only |
| 2 | External PID verification — 003f=C960 (31 probes), 2013=C960 old (33), 007c=C960 2K (5), 0073=C950 (23), 002d=C970 (3), 00EF=S600L (README) | raw fetches, 2026-09-28 |
| 3 | `pixy.Family` (`FamilyPIXY` / `FamilyFixedUVC`) + `DeviceProfile` + static registries + `ResolveProductID` (registry order: PIXY → extras → fixed; extra outranks fixed as explicit opt-in) + `ExtraModelName` (`PIXY (PID 0x…)`) + `UnsupportedDeviceHint` | `internal/pixy/model.go` |
| 4 | `EMEET_PIXYD_EXTRA_PRODUCT_IDS` env config — comma-separated hex, invalid/non-positive entries skipped with warning, `Config.ExtraProductIDs` plumbed | `internal/pixy/config.go` |
| 5 | Probe recognition: `parseUeventLine`, `unsupportedEMEETFromUevent`, `probeResult.UnsupportedHint`, rate-limited (1h) startup/hotplug log, all 7 production `probeDevices` call sites threaded with `d.config.ExtraProductIDs` | `probe.go`, `main.go`, `commands.go`, `device.go`, `motor.go`, `auto.go`, `circuitbreaker.go` |
| 6 | `device` + `probe` commands surface the hint instead of a bare "no device found" | `commands.go` |
| 7 | Tests: `TestResolveProductID` (11 cases), `TestModelFromProductID`, `TestUnsupportedDeviceHint`, `TestParseExtraProductIDs` (8), `TestConfigFromEnv_ExtraProductIDs`, 5 probe recognition tests, device-hint command test, 2 fuzz seeds (003f, 0abc) | `internal/pixy/model_test.go`, `config_test.go`, `probe_video_test.go`, `commands_test.go`, `uevent_fuzz_test.go` |
| 8 | Full verification: `go vet`, `go build`, `GOWORK=off go test -race -count=1 ./...` (green ×2 runs), `golangci-lint` (2.13.2) **0 issues** after fixing my 2 findings + 1 adjacent | session runs |
| 9 | Genuine lint fixes: socket errors now name the path (`create state dir %s`, `listen on %s`); `parseUeventLine` + `hidSendGuard` named returns removed (nonamedreturns) | `socket.go`, `probe.go`, `circuitbreaker.go` |
| 10 | Website **Supported Devices** page (matrix, honest fixed-cam explanation, extra-ID how-to, caution callout, contribution CTA) + sidebar entry | `website/src/content/docs/getting-started/supported-devices.mdx`, `astro.config.mjs` |
| 11 | README: Supported Devices table, sharper "When NOT to use", 2 troubleshooting rows, env-var row; accidental deletion of `## Comparison` heading caught and restored same-pass | `README.md` |
| 12 | Website consistency sweep: changelog entry, troubleshooting + installation (also fixed stale `x86_64`-only → `x86_64 + aarch64`, flake-confirmed) + hid-protocol + configuration mentions; landing "USB Hotplug" card carries the trust line | 6 website files |
| 13 | Project docs closed out: CHANGELOG [Unreleased] (Added + Changed), FEATURES (2 new rows + drift fix `isPixyProductID` → reality + counts 73→75), ROADMAP (PRODUCT_IDS idea → implemented; PIXY Wireless intel row), DOMAIN_LANGUAGE (5 terms), AGENTS.md (file table, multi-device behavior, registry order) | 5 docs |
| 14 | Website build verified twice: 20 pages, pagefind indexed, CSP 20/20, CI sentinel-grep target (`Corrections from official-app evidence`) present in built HTML — including a rebuild AFTER the last changelog.mdx edit | `pnpm run build` runs 23:41 + 23:42 |
| 15 | Erraudit triage of 31 buildflow findings: 2 genuine context_loss fixed; 29 reviewed and consciously left (22 deliberate `_ =` best-effort, 2 by-design `/proc` race skips, 7 go-error-family sentinels — per go-error-modernization decision tree, NOT cargo-culted to zero) | triage in-session |

## b) PARTIALLY DONE

| # | Item | Gap |
|---|------|-----|
| 1 | Fixed-EMEET recognition on the **video4linux walk only** | hidraw-side walk still matches PIXY-family PIDs only; a fixed cam exposing hidraw but not video4linux (rare) is unrecognized |
| 2 | Unsupported-hint surfaces | CLI (`device`/`probe`) + logs yes; **web UI status panel and Waybar do not** — offline just looks offline. Deliberate scope cut, but never recorded as a decision |
| 3 | Lint-debt rationale | The 29-finding triage (why each class is left) lives in the chat transcript only — not persisted in AGENTS.md/ROADMAP, so the next session will re-litigate it |
| 4 | Device-fact consistency sweep | README + website updated, but repo-internal docs still carry the old two-PID story: `docs/hid-protocol.md`, `CONTRIBUTING.md` (device mention at :33), and the **supported-device matrix in `docs/emeet-studio-official-app-comparison.md` is now stale** vs the new registry |
| 5 | Test coverage of new paths | `probe` command hint branch untested (only `device` is); the rate-limited log branch in `probeDevices` untested; `TestHandleQueryCommand_Probe_NoDevice` still relies on real-sysfs tolerance |
| 6 | Landing page positioning | Only the "USB Hotplug" card updated; ROADMAP's older idea (mirror "Who is this for?"/"When NOT to use" fully onto the landing) untouched |
| 7 | Nix gates | `nix build` / `nix flake check` / vmTest NOT run locally (no dep changes → vendorHash untouched, risk low but unverified; udev-rule vmTest assertions unaffected since nix files untouched) |
| 8 | buildflow gate closure | erraudit 31→29 (2 fixed); final full `buildflow` run not re-executed to confirm counts + exit semantics with remaining accepted findings |
| 9 | Website deployment | Not deployed (correct — no push without ask). The auto-daemon's commits + `website.yml` will carry it on next push; OG image for the new page unverified visually |

## c) NOT STARTED

| # | Item | Where it lives |
|---|------|----------------|
| 1 | PIXY Wireless support | Needs a public USB PID + hardware; ROADMAP row created today (opt-in path ready) |
| 2 | Web UI / Waybar unsupported-device surfacing | New — belongs in ROADMAP until a design pass |
| 3 | NixOS module option for `extraProductIds` (declarative env) | Considered, rejected as YAGNI today — revisit on demand |
| 4 | Pre-existing TODO_LIST rows (#129 online screenshots, #138–#141/#150 hardware bundle, #148 upstream PR, #154–#156) | Untouched — hardware/Lars-gated, correctly out of scope |
| 5 | All other ROADMAP themes (OTel tracing, SSE heartbeat, structured command types, multi-word CLI presets, koanf, …) | Untouched |

## d) TOTALLY FUCKED UP

**Nothing at "fucked" severity.** Verified: tests green, lint 0 issues, website builds, no data loss, no reverts of others' work, parallel-agent file (`circuitbreaker.go`) edited forward, not destroyed.

Near-misses worth logging (self-caught, zero residual damage):

1. **README heading swallowed**: a multiedit replaced `## Comparison` without restoring it — the comparison table dangled under "Supported Devices" for exactly one tool-call cycle before I caught it via a headings sweep.
2. **Formatter/edit-tool churn**: a formatter (treefmt-style hook) rewrote files between my reads, causing ~4 failed edits and one literal `\n` typo in a `switch` before I switched to read-after-sed discipline. Wasted cycles, no corruption — but it cost maybe 6–8 tool calls.
3. **`Config` comparability break**: adding `ExtraProductIDs []int64` silently made `Config` non-comparable, breaking `TestConfigFromEnv_DefaultsWhenUnset` (`cfg != def`) — caught by vet, fixed with `reflect.DeepEqual` (the "slice field breaks `==`" class is now a live lesson).
4. **Drift carried forward**: I nearly propagated the phantom `isPixyProductID()` helper (FEATURES.md claimed it; it never existed in code) into the new row — caught by a 5-second grep, fixed to cite `pixy.ResolveProductID`.

## e) WHAT WE SHOULD IMPROVE

1. **Persist lint-debt decisions** — "29 erraudit findings accepted, here's why per class" belongs in AGENTS.md (gotcha) so no session burns an hour re-triaging.
2. **Split-brain sweep before declaring done** — when a fact changes (device support), grep *all* docs (repo + website) for the old claim; I swept website + README but missed repo-internal `docs/` until writing this report.
3. **Formatter-race discipline** — after any `sed -i`, re-`view` before `edit`; better: prefer `lsp_replace_symbol` for signature changes (would have avoided the 3-value destructuring sweep across 8 test files by hand).
4. **Slice-field anticipation** — adding a slice/map to a compared struct: update equality tests in the same commit as the field.
5. **Run the nix gate when touching probe/udev-adjacent code** — even when nix files are untouched, `nix build` is the CI truth; it was skipped for time.
6. **Decision records for scope cuts** — "web UI doesn't show the hint" was decided silently; one ROADMAP bullet would have cost 30 seconds.
7. **E2E smoke** — `daemon + C960-fake` would ideally be exercised via the vmTest/fake-sysfs path rather than unit tests only.

## f) UP TO 50 THINGS TO GET DONE NEXT

*(Brainstorm-ranked by impact; harvest with routing rigor — most are ROADMAP fuel, ~8 are TODO_LIST-grade.)*

**Session leftovers (highest leverage, do first)**
1. Record the erraudit-debt rationale (29 accepted findings, per-class why) in AGENTS.md gotchas.
2. Update the stale supported-device matrix in `docs/emeet-studio-official-app-comparison.md` to the new registry.
3. Update `docs/hid-protocol.md` device-identification section (two-PID story → family registry).
4. Update `CONTRIBUTING.md` device mention + add "report your EMEET PID" flow (ties into the new funnel).
5. Add TODO/ROADMAP rows: web-UI + Waybar hint surfacing (design-light, high UX).
6. Add ROADMAP row: hidraw-side fixed-device recognition (or document video-only as the decision).
7. Test the `probe` command hint branch + the rate-limited log branch.
8. Verify OG image renders for `/getting-started/supported-devices/`.
9. Run `nix build` + `nix flake check` locally to close the nix-gate gap.
10. Re-run full `buildflow` to confirm erraudit 31→29 and record the exit-gate posture.

**NixOS / packaging**
11. Decide: declarative NixOS `extraProductIds` option vs env-only (ADR-lite).
12. vmTest extension: fake sysfs + start the daemon inside the VM (existing ROADMAP item, now more valuable — it would exercise recognition).
13. Consider a udev *snippet* doc for users who want the extra-ID devices group-accessible (documented tradeoff, default stays no-rules).

**Device support (demand-gated)**
14. PIXY Wireless: capture PID on first community report; one-line registry entry + opt-in docs already exist.
15. S600L fill-light offshoot (rakhbari protocol) — research-only ROADMAP row, needs Linux verification.
16. Community PID-classification workflow: issue template with `lsusb | grep 328f` prefill.
17. `GET_FUNC_STA` bitfield decode → capability-gated UI (existing ROADMAP).
18. `FuzzParseV2Response` (existing ROADMAP, hardware-pin dependent).
19. Device-disappear reconcile semantics (existing ROADMAP question).
20. Shared V2 query helper (TODO #172, trigger = 4th GET family).

**Hardware bundle (existing, unchanged)**
21. #166 wired session (speed unit, MotorType, preset shapes, battery verdict) — still the single highest-impact hardware task.
22. #129 online screenshots (bundle with #166).
23. #138/#139/#140/#141 hardware follow-throughs.
24. #150 MotorType/DefaultPosMode pinning (+ optional Windows usbmon capture).
25. #154 close issue #6 loop; #155 release cadence call (v0.4.1 vs v0.5); #156 branch protection.

**Marketing / website**
26. Deploy the website (push → `website.yml`) and verify the new page live.
27. Add per-page feedback links + reading time (Starlight config; existing ROADMAP web idea).
28. Mirror "Who is this for?"/"When NOT to use" fully onto the landing page (existing ROADMAP idea).
29. Lars's standing call: public EMEET-STUDIO comparison/RE-methodology page (positioning win vs internals-exposure risk).
30. `prettier`/`dprint` config for `.mdx`/`.mjs` (existing ROADMAP idea) to end formatter churn class.
31. Screenshot refresh pipeline: scripted headless-chromium captures so #129 stops being manual.
32. SEO pass: per-page `description` frontmatter audit + sitemap ping after deploy.
33. Demo video: mention multi-device honesty line (optional reshoot trigger — cheap to defer).
34. Starlight callouts sweep (`:::tip`/`:::caution`) for buried prose notes (existing ROADMAP).

**Code health**
35. Error-wrapping `%v`→`%w` audit (existing ROADMAP; breaks `errors.Is` chains).
36. `errorfamily.LogError()` expansion to remaining `slog.Error` sites (existing ROADMAP).
37. Structured command types (ADR awaiting Lars's decision, TODO-#116 lineage).
38. Multi-word preset names via CLI (ADR awaiting Lars's approval, ~6 lines).
39. CI guard: fail if the go-modules FOD references store paths (existing ROADMAP).
40. `EMEET_PIXYD_MOTOR_SPEED` env default (existing ROADMAP, post-#138-unit).
41. `nix flake update` cadence automation (Renovate/scheduled; existing ROADMAP).

**Docs health**
42. HARVEST this report's (f) section into TODO_LIST/ROADMAP with routing rigor (docs-health rule — do NOT entomb).
43. `FEATURES.md` mobile/a11y partial rows: actually execute the two documented checklists (screen-reader, real-device).
44. Annotate/sweep older `docs/status/` reports per docs-health ANNOTATE (completeness grep).
45. `docs/DOMAIN_LANGUAGE.md`: consider "Extra Product ID" cross-link from README glossary-ish section (tiny).

**Testing**
46. Property test for `ResolveProductID` registry-order invariants (extra > fixed, PIXY > all; 200 seeded cases, house style).
47. Simulator: fixed-EMEET HIDDevice profile (refuse-and-log behavior) to pin "no vendor bytes to non-PIXY" at the HID layer.
48. Benchmark for `probeVideo4linux` with the added recognition pass (regression guard; expected negligible).

**Ops**
49. Push cadence decision (existing open question) — local commits are piling while origin CI idles.
50. Dependency-bump ownership for `website/` (existing open question; prevents the lockfile-conflict class).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Unsupported-hint surfacing:** should the web UI (offline panel) and Waybar also show the "recognized but not controllable: EMEET C960…" message, or is CLI + logs the intended surface? (Determines whether I add a `webStatus`/tooltip field or close the idea.)
2. **NixOS module:** do you want a declarative `hardware.emeet-pixy.extraProductIds` option (list-of-strings → env), or is `EMEET_PIXYD_EXTRA_PRODUCT_IDS` as a user-set env var the intended interface forever?
3. **Accepted-lint-debt policy:** the 29 remaining buildflow erraudit findings (22 deliberate `_ =`, 2 by-design `/proc` continues, 7 go-error-family sentinels) — record-and-accept as a documented baseline, or do you want `//nolint` suppressions at the sites so the buildflow gate can exit green?

---

**Waiting for instructions.** Per the harness contract I did not commit this report manually — the auto-commit daemon picks it up.
