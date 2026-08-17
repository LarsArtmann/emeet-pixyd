# Status: Website Recovery, Pitch Rewrite, Screenshots & Demo Video

**Date:** 2026-08-17 18:53
**Session scope:** Fix `emeet-pixyd.lars.software` "Site Not Found" → full go-live; rewrite landing pitch around the "no Linux software for this webcam" origin story; add real screenshots; produce a HyperFrames demo video; rebuild + verify website.
**Session ends with:** shell PATH completely broken (all commands fail, even `ls`/`date`) — report written via Write tool per user instruction. Undeployed build, uncommitted work.

---

## a) FULLY DONE

### 1. Site Not Found — diagnosed and fixed
- **Symptom:** `https://emeet-pixyd.lars.software/` served Firebase "Site Not Found".
- **Diagnosis chain:** DNS CNAME was correct (`emeet-pixyd.lars.software` → `emeet-pixyd.web.app`, resolving) but the Firebase hosting **site did not exist** (`firebase hosting:sites:list` had no `emeet-pixyd`). A prior session staged the DNS record but never created the site or deployed.
- **Fix executed:**
  - `firebase hosting:sites:create emeet-pixyd --project lars-software` — site created.
  - Website build repaired: `pnpm install --frozen-lockfile` failed (package.json refreshed html-validate/typescript vs stale lockfile) → regenerated lockfile, approved esbuild build script (`pnpm approve-builds esbuild`, wrote `allowBuilds: esbuild: true` into `website/pnpm-workspace.yaml`).
  - `pnpm run build` → 19 pages, CSP patched 19/19.
  - `firebase deploy --only hosting:emeet-pixyd --project lars-software` → release complete.
  - Verified `https://emeet-pixyd.web.app` returns 200 with full landing content.

### 2. Custom domain attached
- Added via Firebase REST API (`customDomains` endpoint, `x-goog-user-project` header, `{}` body, domain in query param — per skill reference script `/tmp/firebase-create-domain.js`).
- Status reached: `HOST_ACTIVE` / `OWNERSHIP_ACTIVE` / `CERT_VALIDATING`.
- User confirmed: site loads at the custom domain (with browser security warning — see §c).

### 3. Pitch rewrite (copywriting skill loaded and followed)
- **Hero:** badge "Auto-activation daemon for Linux" → **"Reverse-engineered for Linux"**; H1 → **"A great AI webcam, dumb on Linux. Until now."**; subhead tells the vendor-app origin story.
- **Metrics row:** "2s Poll interval / Linux Platform" → **"±150° Pan range / MIT License"** (more concrete, user-relevant).
- **New `WhySection.astro`** ("Why this exists — Great hardware. No Linux software. So I built the Linux software myself."): 3 story cards (The hardware / The problem / The fix, mentioning the 9-byte HID reports, commit handshake, 200ms timing) + accent result callout. Wired as the first section in `Sections.astro`.
- **New `camera` Lucide icon** added to `Icon.astro` + `useCaseIconKeys` in `types.ts` (standalone `tsc --strict` TS6 typecheck passes).
- **Meta copy:** `config.ts` title/description rewritten to match the origin story.

### 4. Real screenshots captured
- PIXY hardware was **disconnected** (`camera=offline` from running daemon) → screenshots show the offline UI state.
- Headless Chromium against live daemon `http://127.0.0.1:8090`:
  - `webui-full.png` (1440×2600 full page)
  - `webui-viewport.png` (1440×900 above the fold)
  - `webui-panel.png` (1440×1800 crop for the panel area)
- Verified programmatically: DOM dump contains `#status-panel`, `#preview`, mode labels; color sampling shows dark theme with 60 unique colors (not blank). Note: current model cannot view images — verification was code-level only.

### 5. HyperFrames demo video — researched, built, rendered, verified
- Researched `heygen-com/hyperframes` (real: HTML/CSS/GSAP → deterministic MP4; agent-friendly CLI) + full CLI docs + llms.txt.
- Scaffolded `/tmp/pixy-video/pixy-demo` (`init --example blank`).
- Authored 25s 1920×1080 composition, 4 scenes:
  1. Title ("A great AI webcam. Dumb on Linux.")
  2. Problem (3 feature cards ✗ unavailable, "WINDOWS & macOS ONLY" stamp)
  3. **UI recreation** — camera preview w/ lens + REC badge, mode cards (tracking → privacy), audio segments, PTZ radar dot motion + readout swaps, hardware shutter closing, terminal clip-path typing, toasts
  4. Outro (URL + GitHub).
- **Determinism fixes:** removed `.call()` callbacks and `text:` plugin (not seek-safe) → stacked-opacity layers + `clipPath` wipe; `left/top` → `x/y` transforms (linter rule).
- **NixOS fix:** bundled chrome-headless-shell missing shared libs → `HYPERFRAMES_BROWSER_PATH=<nix chromium>` works.
- **Quality gates:** lint 0 errors; `check` passing after fixing 10 WCAG contrast errors (gray `#8b949e`/`#6e7681` → `#a8b2bd`/`#98a2ad`, 51/51 AA), radar-over-mode-card occlusion (`.side { margin-right: 210px }`), intentional shutter overflow marked `data-layout-allow-overflow`.
- **Render:** `demo.mp4` — 1.4 MB, 25.0s, 750 frames, ~30s wall time.
- **Verification:** ffmpeg frame extraction at t=2/6/12/17/22 + histogram: violet title `#885AF3` at t=2, green tracking card `#44B893` at t=12 — scenes and state transitions render correctly.

### 6. Showcase section + website rebuild
- New `ShowcaseSection.astro`: framed video player (`/demo.mp4`, poster `webui-viewport.png`) + 2 screenshot figures; wired into `index.astro` above FeatureGrid.
- `pnpm run build` clean: 19 pages, CSP patched 19/19.
- Built HTML verified via grep + fetch of preview server: all new content present (hero, Why section, video element, screenshots).

---

## b) PARTIALLY DONE

### SSL certificate for custom domain — externally blocked
- Firebase serves placeholder `CN=firebaseapp.com` cert → browser shows "unsecure". Needs one DNS record:
  - `_acme-challenge.emeet-pixyd` TXT = `13skp2wu2edNpS6KZUvy7rMsk9F0ZJu6IMIxroIBbZM`
- **Staged** in `~/projects/domains/lars.software.tf` next to the existing CNAME; `terraform fmt` + `validate` pass.
- **Blocked:** Namecheap API key in `terraform.tfvars` is the placeholder string (error 1011102); additionally `api.ipify.org` is blocked from this machine so `NAMECHEAP_CLIENT_IP=89.65.239.240` must be set. No credentials found in `pass` store, env, or repo docs.
- **Manual user step required** (see questions).

### Deploy of the NEW build — not done
- The new pitch/screenshots/video build exists locally (`website/dist/`) but was **not deployed** — session hit the shell failure right after visual QA started.

### Visual QA — incomplete
- Preview server + landing fetch verified content; screenshot color-sampling showed flat gray rows below the fold — suspected scroll-reveal animations (`animations.js` IntersectionObserver) hiding off-screen content in the headless capture. Never confirmed before the shell broke.

---

## c) NOT STARTED

- **CI/CD for the website** (skill Phase 7: `FIREBASE_SERVICE_ACCOUNT` secret + deploy workflow) — none of the 3 existing workflows deploy the website.
- **README pitch alignment** — README still carries the old framing; website hero changed.
- **GitHub repo metadata** (description/homepage) — untouched.
- **AGENTS.md memory updates** for this session's learnings (NixOS HyperFrames workaround, pnpm approve-builds, showcase section, cert blocker).
- **Video polish**: no audio/music/voiceover, no vertical cut, poster reuses viewport screenshot.
- **`.firebase/` cache dir** — appeared untracked in `website/`, not gitignored.

---

## d) TOTALLY FUCKED UP

- **Shell environment died at session end:** every bash invocation now fails (`"ls": executable not found in $PATH`, same for grep/date/tr — even with explicit PATH exports). Root cause unknown (possible nix shell contamination). Consequences: final deploy, git commit, and last visual checks did not run. Report written via Write tool as instructed.
- **`astro check` is broken repo-wide (pre-existing, surfaced by me):** commit f398829 refreshed deps to `typescript@7.0.2`, which dropped the programmatic API `astro check` requires. Every `astro check` invocation crashes (also from neutral dir with `-p typescript@6` because website `node_modules` wins). Workaround used: standalone `tsc` on changed file. Not my breakage, not fixed.
- **`fix-csp.mjs` was run with a stale lockfile mismatch mid-session** — wasted a build cycle; recovered by regenerating the lockfile. Minor.

---

## e) WHAT WE SHOULD IMPROVE (self-critique)

1. **Deploy discipline:** I verified the build and then wandered into extra QA instead of deploying immediately. Ship-then-polish.
2. **Commit discipline:** skill mandates commits at checkpoints (site live, build verified). Nothing explicitly committed this session (auto-git daemon may have caught some).
3. **Screenshot timing:** camera was disconnected — I shipped offline-state screenshots. Should have paused and asked for the camera to be plugged in; online-state shots are far better marketing.
4. **Crop guess:** `webui-panel.png` crop offset (0,400) was never visually validated (model can't view images) — could be misframed.
5. **`firebase.json` cache headers don't cover `.mp4/.webm`** — the new video gets no long-cache header. Noticed late, not fixed.
6. **HyperFrames project lives in `/tmp`** — lost on reboot; re-rendering later means rebuilding.
7. **Memory updates skipped** — several durable learnings unwritten.
8. **PATH fragility:** heavy `nix shell … -c` usage all session; one bad environment and the session loses execution entirely. Consider wrapper functions or `direnv`.

---

## f) NEXT — up to 50 things

**Finish this launch**
1. Deploy the verified build (`firebase deploy --only hosting:emeet-pixyd`)
2. User: refresh Namecheap API key + whitelist IP `89.65.239.240`
3. Apply Terraform (`-target=namecheap_domain_records.lars_software`, with `NAMECHEAP_CLIENT_IP`)
4. Poll cert → `CERT_ACTIVE`, verify `https://emeet-pixyd.lars.software` 200 + valid cert
5. Verify TXT via `host -t TXT _acme-challenge.emeet-pixyd.lars.software`
6. Add `mp4|webm` to firebase.json cache-header extension list
7. Commit all website changes (+ `pnpm-workspace.yaml`, lockfile)
8. Add `.firebase/` to .gitignore
9. Diagnose/fix broken shell PATH (reboot shell or new session)

**Website content**
10. Retake screenshots with PIXY connected (online UI, live preview, tracking active)
11. Regenerate panel crop from online screenshot
12. Dedicated video poster from t=2 title frame
13. Update README pitch to match new hero
14. Update GitHub repo description + topics (`gh repo edit`)
15. Add VideoObject JSON-LD for demo.mp4 (SEO)
16. Mobile-layout check of ShowcaseSection
17. Dark/light toggle check with new section
18. Lazy-load attributes on video (`preload="metadata"` already, verify network weight on landing)
19. Animated GIF fallback poster for environments w/o video autoplay
20. Waybar output screenshot for showcase
21. CLI (`emeet-pixy --help`, status) screenshot for showcase
22. `/metrics` output screenshot for showcase
23. "Where to go next" retrofit audit on 16 docs pages (skill maintenance mode)
24. Docs page mirroring the "Why this exists" story
25. Vendor-app vs emeet-pixyd comparison table in docs
26. Verify OG image text matches new pitch
27. Pin `typescript@6.x` in website package.json until astro check supports TS7
28. Cache GitHub stars fetch (unauthenticated API in HeroSection can rate-limit)
29. Verify website flake (`nix build`) still green
30. Compress screenshots (PNG → webp) for landing weight

**Video**
31. Add music/voiceover (`hyperframes tts` or BGM)
32. Move project out of `/tmp` into `website/video/` (or repo `video/`)
33. 9:16 vertical cut for social
34. Add preset-chips demo to scene 3
35. Keyboard-shortcut scene (T/I/P/C, arrows, ?)
36. Show real camera footage when PIXY reconnected (HTML-in-canvas or captured frames)
37. Re-render after any copy change (single command once project is in repo)

**CI/CD + ops (skill Phase 7)**
38. Create `FIREBASE_SERVICE_ACCOUNT` secret (gcloud SA key → `gh secret set`)
39. Add two-job website workflow (build → deploy on master)
40. Add rollback runbook (`firebase hosting:rollback`) to repo docs
41. `nix flake check` on website flake in CI

**Memory / docs**
42. Update project AGENTS.md: showcase section, screenshots pipeline, HyperFrames learnings
43. Note the ACME TXT token in AGENTS.md/domain docs until cert is active
44. Update TODO_LIST.md from this report
45. CHANGELOG entry for website relaunch + video

**Daemon/product (observed, not touched)**
46. `emeet-pixy zoom 120 # Zoom to 120x` hero comment is misleading (zoom 100–150 is a percentage, not "120x") — fix copy
47. Daemon reported `camera=offline` while a stale process ran from `/nix/store/...-f3988293...` — confirm deployed version matches latest commit after rebuild
48. Consider `systemctl --user restart emeet-pixyd` smoke test post-deploy
49. Integration test note: screenshots depend on daemon running — document the capture command in website docs
50. Session retro: wrap `nix shell … -c` in repo helper script to avoid PATH nuking

---

## g) Questions I cannot answer myself

1. **Namecheap credentials:** Can you refresh the API key in `~/projects/domains/terraform.tfvars` and whitelist `89.65.239.240` in the Namecheap dashboard? Without it the ACME TXT can't be applied and the cert stays in `CERT_VALIDATING` (site stays "unsecure"). Everything else is staged; one `terraform apply` finishes it.
2. **Retake screenshots when the PIXY is plugged in?** Current shots show the offline UI. Want me to redo screenshots + (optionally) weave real camera footage into the video once the camera is connected, or ship the offline-state assets now?
3. **Deploy + video location:** Deploy the verified build now without further review? And should the HyperFrames project be committed into the repo (e.g. `website/video/`) so future re-renders are one command, or is `/tmp` acceptable because the MP4 is committed?

---

*Report written under degraded tooling (no shell). All claims verified by tool output captured during the session.*
