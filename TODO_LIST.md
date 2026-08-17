# emeet-pixyd — TODO List

**Updated:** 2026-08-18 (website relaunch deployed + cert active — see `CHANGELOG.md` for details)

> Completed work lives in `CHANGELOG.md` — it does NOT live here. Long-term ideas, design-heavy items, "decided won't-do" decisions, and open questions live in `ROADMAP.md`. This file is **open work only**.

---

## Status Legend

- ⬜ TODO — Not started
- 🔶 PARTIAL — Started but incomplete
- 🚫 BLOCKED — Waiting on an external unblock (permission, decision, upstream)

---

## Blocked (highest impact — needs an external unblock)

| #   | Status     | Task                                                                                                                                                                                                                                | Impact  | Effort | Evidence                                                                                                                                                  |
| --- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 124 | 🚫 BLOCKED | **Publish `go-branded-id` v0.5.1** (binary-free), bump `emeet-pixyd` go.mod, remove the TEMPORARY `goBrandedSrc`/`replaceBrandedId` shims from `flake.nix` + `package.nix`, recompute `vendorHash`                                  | 🔴 HIGH | S      | `go.mod` still `v0.5.0`; `flake.nix:77-94` + `package.nix:7-8,27` still carry the workaround; `2026-07-28_15-24` §b.1, §c.1, §f.1, §g.1; AGENTS.md gotcha |
| 129 | 🚫 BLOCKED | **Retake web UI screenshots with the PIXY connected** (online state: live MJPEG preview, tracking active) — current shipped screenshots show the offline UI. Regenerate `webui-panel.png` crop + video poster from the online shots | MED     | S      | `2026-08-17_18-53` §a.4/§e.3 — camera was disconnected during capture; files in `website/public/screenshots/`                                             |

#124 is blocked on push permission to the `go-branded-id` remote (the binary is already untracked upstream in `c29a034`; only the tag/publish + go.mod bump remain). #129 needs the camera plugged in.

---

## TODO

| #   | Status  | Task                                                                                                                                                                                                                                                                      | Impact | Effort | Evidence                                                                                                                   |
| --- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | -------------------------------------------------------------------------------------------------------------------------- |
| 130 | ⬜ TODO | **Rebuild the HyperFrames demo-video composition inside the repo** (e.g. `website/video/`) — the source lived in `/tmp/pixy-video` and was lost on reboot; only the rendered `demo.mp4` is committed. Rebuilding enables future re-renders (copy tweaks, 9:16 cut, audio) | MED    | M      | `2026-08-17_18-53` §a.5, §e.6 — `/tmp/pixy-video/pixy-demo/` gone; render command + NixOS env vars documented in AGENTS.md |
| 131 | ⬜ TODO | **Align README pitch with the new hero origin story** ("no Linux software for this webcam") — README still carries the old framing while the website hero changed                                                                                                         | MED    | S      | `2026-08-17_18-53` §c; compare `README.md` intro vs live hero                                                              |
| 132 | ⬜ TODO | **Update GitHub repo metadata** (`gh repo edit` — description, homepage `https://emeet-pixyd.lars.software`, topics: linux, webcam, hid, nixos, go)                                                                                                                       | LOW    | S      | `2026-08-17_18-53` §c                                                                                                      |
| 133 | ⬜ TODO | **Website CI/CD**: `FIREBASE_SERVICE_ACCOUNT` secret + deploy workflow (build → deploy on master) so the website stops depending on manual `firebase deploy`                                                                                                              | MED    | M      | `2026-08-17_18-53` §c; website-launch skill Phase 7; no existing workflow deploys the website                              |
| 134 | ⬜ TODO | **Pin `typescript@6.x` in website `package.json`** until `astro check` supports TS7 (typescript@7.0.2 from dep refresh f398829 dropped the programmatic API `astro check` needs — it crashes repo-wide)                                                                   | LOW    | S      | `2026-08-17_18-53` §d; workaround was standalone `tsc --strict`                                                            |
| 135 | ⬜ TODO | **Unify hero terminal code** — `website/src/data/hero-code.ts` (copy-button source) and the hardcoded `highlightedCode` string in `HeroSection.astro` duplicate the same terminal content; they drift silently (survived one near-miss: "Zoom to 120x" fix needed both)   | LOW    | S      | `HeroSection.astro:24-39` + `hero-code.ts`; found while fixing the zoom copy                                               |
| 136 | ⬜ TODO | **Landing polish**: `VideoObject` JSON-LD for `demo.mp4` (SEO), PNG→webp screenshot compression, mobile + dark/light QA of ShowcaseSection, dedicated poster frame from t=2 instead of reusing the viewport screenshot                                                    | LOW    | M      | `2026-08-17_18-53` §f (15, 12, 16-17, 30)                                                                                  |

---

All items completed before 2026-07-28 (#106–#128) and the 2026-08-17/18 website relaunch (#127 deploy + cert) are in `CHANGELOG.md`. Full 50-item backlog from the 2026-08-17 session: `docs/status/2026-08-17_18-53_website-recovery-pitch-screenshots-video.md` §f.

> #116 (structured command types — HIGH/HIGH) and #123 (multi-word preset names through CLI dispatch) need design decisions first; both are in `ROADMAP.md`.
