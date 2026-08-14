# Status: MIT Migration, Docs Accuracy, CSP, OG Images — Round 2

**Date:** 2026-07-14 02:45
**Session goal:** Fix all issues identified in the previous status report (license split brain, missing OG images, missing CSP, docs inaccuracy, broken icon, missing AGENTS.md website section)

---

## A) FULLY DONE

1. **LICENSE changed to MIT** — `LICENSE` file replaced with standard MIT text. Previously was Proprietary.

2. **License consistency achieved across all surfaces**:
   - `LICENSE` file: MIT
   - `README.md`: Changed from "Proprietary" to "MIT", added MIT badge
   - `website/package.json`: Was already "MIT" (correct from session 1)
   - `website/src/components/Footer.astro`: Changed from "Linux-only" to "MIT License"
   - `website/src/layouts/LandingLayout.astro` (JSON-LD): Added `"license": "https://opensource.org/licenses/MIT"`
   - `flake.nix`: Was already `MIT` (correct)
   - `package.nix`: Was already `lib.licenses.mit` (correct)

3. **OG image generation added** — `astro-og-canvas` dependency added, `src/pages/og/[...slug].ts` generates 19 per-page OG images (one per page) with violet border (`[139, 92, 246]`). OG meta tags added to `LandingLayout.astro` (`og:image`, `twitter:image`, `twitter:card` upgraded to `summary_large_image`).

4. **CSP added** — Three-layer implementation:
   - Astro `security.csp` config in `astro.config.mjs` (`script-src 'self'`, `style-src 'self' 'unsafe-inline'`)
   - `scripts/fix-csp.mjs` post-build script that hashes inline scripts and injects SHA-256 hashes (ported from gogenfilter)
   - `Content-Security-Policy` header in `firebase.json`
   - Build script updated: `astro build && node scripts/fix-csp.mjs`
   - Verified: CSP fix patches 19/19 HTML files on build

5. **Plug icon fixed** — Replaced the nonsensical two-horizontal-lines SVG with a proper USB plug path (vertical cable with connector prongs).

6. **Docs accuracy verified and corrected against Go source code**:
   - **Metrics page**: Rewrote entirely. Removed invented metrics (`probe_duration_seconds`, `command_duration_seconds`). Added real metrics: `emeet_pixyd_auto_mode`, `emeet_pixyd_hid_failures_total`. Corrected types (Float64Gauge, Int64Counter, Float64Histogram).
   - **Waybar page**: Rewrote with actual JSON output format (Font Awesome icons `\uf030`/`\uf03d`/`\uf011`/`\uf00d`, text values `CAM`/`IDLE`/`OFF`/`---`, class prefix `custom-camera`, tooltip format `EMEET PIXY: <state>`, conditional `In call: yes` line).
   - **Architecture/API endpoints**: Fixed from 18 to 24 endpoints. Added missing: `/panel`, `/api/stream`, `/api/toggle-privacy`, `/api/sync`, `/api/probe`, `/api/gesture` (corrected to toggle), `/debug/pprof/*` (conditional). Removed nonexistent `/api/status`.
   - **Web UI keyboard shortcuts**: Fixed step sizes — pan/tilt step is 5 degrees (not generic "step"), zoom step is 10 (not generic). Added `=`/`_` alternative keys. Added note about shortcuts being ignored in input fields.

7. **AGENTS.md updated**:
   - Updated date to 2026-07-14
   - Added `website/` to File Responsibilities table
   - Added full "Website" section with file table + conventions (accent color, CSP, OG images, build/deploy commands, Node version)
   - Added MIT license consistency note in Gotchas section

8. **Final build verified**: 0 errors, 0 warnings, 19 pages + 19 OG images, CSP patched 19/19 files, typecheck clean.

---

## B) PARTIALLY DONE

1. **Docs accuracy verification** — Fixed 4 docs pages (metrics, waybar, API endpoints, keyboard shortcuts) but did NOT verify ALL 16 docs pages. The following were NOT cross-referenced against source:
   - `cli-reference.mdx` — CLI commands were verified in session 1 and appear correct
   - `configuration.mdx` — Env vars were verified in session 1 and appear correct
   - `ptz-control.mdx` — Ranges verified in session 1 (correct: -150/150, -90/90, 100/150)
   - `auto-modes.mdx` — Mode table verified in session 1 (correct)
   - `hid-protocol.mdx` — General description, may have minor inaccuracies
   - `call-detection.mdx` — General description, may have minor inaccuracies
   - `presets.mdx` — MaxPresets=16 and MaxPresetNameLength=32 verified (correct)
   - `architecture/overview.mdx` — Source layout table may be slightly stale

2. **Website deployment** — All code ready but not deployed. Same blockers as session 1: Firebase site creation + terraform apply + firebase deploy all need manual execution.

3. **README content** — License fixed, but the README still doesn't mention the website URL in the body text (only in badge + footer). Could add a "Documentation" section.

---

## C) NOT STARTED

1. **Firebase hosting site creation** (`firebase hosting:sites:create emeet-pixyd`)
2. **Firebase custom domain attachment** (`emeet-pixyd.lars.software`)
3. **`terraform apply`** in domains repo
4. **`firebase deploy`** of the website
5. **ACME challenge TXT record** (needed after Firebase provisions SSL cert)
6. **GitHub Actions workflow** for website CI/deploy
7. **Git commit** of any changes (website, LICENSE, README, AGENTS.md, domains)
8. **`package-lock.json` update** — The new `astro-og-canvas` dependency was installed but the lockfile should be committed

---

## D) TOTALLY FUCKED UP

1. **First edit attempt on LICENSE** — The `write` tool failed with "File has been modified since it was last read" because I tried to write before reading. Had to re-read then re-write. Minor waste of a tool call.

2. **AGENTS.md edit collision** — When adding the license note before `## NixOS Module`, I accidentally consumed the `## NixOS Module` header in the replacement. Had to immediately fix it with another edit. Sloppy context matching.

3. **CSP header may be too strict** — The `firebase.json` CSP header is `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'`. But the actual CSP hashes are injected by `fix-csp.mjs` into the HTML `<meta>` tags (Astro's CSP approach). Having BOTH a `firebase.json` header AND an HTML meta CSP could conflict — the more restrictive one wins. The `firebase.json` CSP doesn't include the inline script hashes, so it would block them. **This is likely broken on actual deployment** — the meta CSP (with hashes) and the header CSP (without hashes) will conflict. Need to either: (a) remove CSP from `firebase.json` and rely solely on Astro's meta CSP, or (b) remove Astro CSP and put everything in `firebase.json` (but then fix-csp.mjs can't inject hashes).

---

## E) WHAT WE SHOULD IMPROVE

1. **Fix the CSP double-source conflict** — The `firebase.json` CSP header and Astro's meta CSP will conflict. Pick one approach. Recommendation: remove the CSP header from `firebase.json` and rely on Astro's meta CSP (which includes the injected hashes from `fix-csp.mjs`). The gogenfilter reference website does NOT have a CSP header in `firebase.json` — it relies solely on the meta CSP.

2. **Verify ALL 16 docs pages against source** — 4 were verified this session; 12 remain with varying confidence levels.

3. **Add website URL to README body** — Currently only in badge + footer link. A "Documentation" section or inline mention would improve discoverability.

4. **Add Starlight custom OG image to docs pages** — Starlight pages get OG images from the `[...slug].ts` route, but Starlight's own head doesn't include `og:image` meta tags by default. Need to verify the docs pages actually reference the OG images in their HTML.

5. **The `connect-src 'self'` in CSP** — SSE connections go to the same origin, so this should be fine. But worth verifying.

6. **Website `.gitignore`** — Should be verified that `package-lock.json` is NOT ignored (it should be committed).

7. **No `html-validate` config** — gogenfilter had this; we have the devDependency but no config file.

---

## F) UP TO 50 THINGS TO GET DONE NEXT

### Critical (must fix before deployment)

1. **Fix CSP conflict** — Remove `Content-Security-Policy` header from `website/firebase.json` (let Astro meta CSP handle it)
2. Verify OG images are referenced in docs page HTML (not just the landing page)
3. Verify `package-lock.json` is not gitignored
4. Git commit all changes (LICENSE, README, AGENTS.md, website/, domains/)
5. Create Firebase hosting site: `firebase hosting:sites:create emeet-pixyd`
6. Add custom domain in Firebase console
7. Run `terraform apply` in domains repo
8. Deploy website: `nix run .#deploy` (or `pnpm run build && firebase deploy --only hosting`)
9. Add `_acme-challenge.emeet-pixyd` TXT record once Firebase provides it
10. Verify `emeet-pixyd.lars.software` resolves and serves correctly

### Docs accuracy (remaining 12 pages)

11. Verify `cli-reference.mdx` against `commands.go` (spot-check all commands)
12. Verify `configuration.mdx` env var defaults against `internal/pixy/` config code
13. Verify `ptz-control.mdx` V4L2 details against `ptz.go`
14. Verify `auto-modes.mdx` call lifecycle against `auto.go`
15. Verify `hid-protocol.mdx` byte structure against `hid.go`
16. Verify `call-detection.mdx` debounce logic against `auto.go`
17. Verify `presets.mdx` validation rules against source
18. Verify `architecture/overview.mdx` source layout is current
19. Verify `architecture/hid-protocol.mdx` generic function signature
20. Verify `architecture/call-detection.mdx` process exclusion logic
21. Verify `getting-started/installation.mdx` NixOS module instructions
22. Verify `getting-started/quick-start.mdx` output format matches actual `status` command
23. Verify `troubleshooting.mdx` solutions are accurate
24. Verify `changelog.mdx` — currently generic, should reflect actual commits
25. Verify `contributing.mdx` commands are correct
26. Verify `related-tools.mdx` descriptions are fair/accurate

### Quality improvements

27. Remove CSP header from `firebase.json` (fix conflict with meta CSP)
28. Add `html-validate` config file
29. Add `jscpd` dedup check (gogenfilter has it)
30. Add GitHub Actions workflow for website build + deploy
31. Add website URL mention to README body text
32. Test mobile responsive layout on actual device
33. Test light mode rendering
34. Verify Pagefind search works on deployed site
35. Add real screenshot of web UI to landing page
36. Add architecture diagram (D2 or Mermaid) to docs
37. Improve favicon design (current is basic camera lens)
38. Add `CHANGELOG.md` entry for the MIT license change
39. Update `FEATURES.md` if it mentions license
40. Verify `TODO_LIST.md` doesn't reference old proprietary license

### Infrastructure

41. Commit `package-lock.json` for reproducible builds
42. Verify website nix flake builds correctly (`nix build` in `website/`)
43. Add `.nix` file to `website/.gitignore` (result symlink)
44. Consider adding `pre-commit` hooks for website (astro check, html-validate)
45. Add Firebase deploy to GitHub Actions (with `FIREBASE_TOKEN` secret)
46. Consider adding `treefmt-nix` check to website CI

### Polish

47. Add a "Documentation" section to README with link to website
48. Add badge for website status (uptime monitoring)
49. Consider adding Discord/community links to website footer
50. Add animated demo/GIF of auto-activation flow to landing page

---

## G) TOP 2 QUESTIONS I CANNOT ANSWER MYSELF

### 1. Should I remove the CSP header from `firebase.json` to avoid the double-CSP conflict?

The website now has CSP in two places:

- **Astro meta CSP** (in HTML `<head>`) — includes SHA-256 hashes for inline scripts, injected by `fix-csp.mjs`
- **Firebase header CSP** (in `firebase.json`) — static, does NOT include inline script hashes

When both are present, browsers apply the MORE RESTRICTIVE union. The Firebase header CSP (`script-src 'self'`) would block all inline scripts (theme-init, header, animations, copy-code) because it lacks the hashes. **This is almost certainly broken.**

My recommendation: remove the CSP header from `firebase.json` and rely solely on Astro's meta CSP (which the gogenfilter reference also does). But I want to confirm before making the change, because I might be wrong about how browser CSP conflict resolution works, or you might want a defense-in-depth approach.

### 2. Should I commit and push everything now, or wait until the Firebase infrastructure is ready?

The website code is complete and builds cleanly. The domains DNS change is in Terraform config but not applied. The LICENSE/README/AGENTS.md changes are ready. Should I:

- **(a)** Commit everything now (so the public repo reflects MIT license immediately), and handle deployment separately?
- **(b)** Wait until Firebase hosting is set up, deploy the website, verify it works, THEN commit everything together?

Option (a) seems safer (the MIT license change should be public ASAP), but I want your call since I shouldn't push without explicit approval.

---

## Appendix: CSP Conflict Fix (post-report)

**Date:** 2026-07-14 02:58

### Problem

The CSP header in `firebase.json` (`script-src 'self'`) would have blocked all inline scripts (theme-init, header, animations, copy-code) on actual deployment because it lacked the SHA-256 hashes that `fix-csp.mjs` injects into the HTML `<meta>` tag. When both a header CSP and a meta CSP exist, the browser applies the MORE RESTRICTIVE union — the static header would have won and broken all inline JS.

### Fix

- **Removed** the `Content-Security-Policy` header from `website/firebase.json`
- CSP is now handled solely by Astro's meta CSP (`security.csp` config + `fix-csp.mjs` post-build)
- This matches the gogenfilter reference website (0 CSP headers in `firebase.json`)

### Verification

- Landing page (`dist/index.html`): meta CSP present with 14 SHA-256 inline script hashes
- Docs page (`dist/guides/metrics/index.html`): meta CSP present
- `firebase.json`: 0 occurrences of `Content-Security-Policy`
- Full build: 19 pages + 19 OG images, CSP patched 19/19 HTML files, 0 errors
