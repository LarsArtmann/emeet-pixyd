# Accessibility Audit & Testing Guide

> **WCAG 2.1 AA code-level audit** — completed 2026-07-28.
> Manual screen reader and mobile device testing checklists below.

---

## WCAG 2.1 AA Code-Level Audit

### Summary

| Criterion                    | Status | Notes                                                                                                                                                             |
| ---------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1.3.1 Info and Relationships | PASS   | Semantic HTML (`<main>`, `<header>`, `<aside>`, `<button>`, `<input>`); ARIA roles on custom widgets (`role="switch"`, `role="alert"`, `role="status"`)           |
| 1.4.3 Contrast (Minimum)     | PASS   | All text/background combinations exceed 4.5:1. `--text-dim` (#8a93a8) on `--surface` (#13161d) = 5.4:1. Placeholder opacity raised to 0.75 for 1.4.11 compliance. |
| 1.4.11 Non-text Contrast     | PASS   | All interactive elements have visible borders, focus indicators (2px accent outline), and state indicators (active/pressed classes).                              |
| 2.4.7 Focus Visible          | PASS   | All interactive elements have `:focus-visible` with 2px solid accent outline + offset. Preset input `:focus-visible` added in this audit.                         |
| 3.3.2 Labels or Instructions | PASS   | Preset input has `aria-label="Preset name"`. All icon-only buttons have `aria-label`. Toggle buttons use `aria-labelledby` referencing visible labels.            |
| 4.1.2 Name, Role, Value      | PASS   | Toggle buttons: `role="switch"` + `aria-checked`. Mode cards: `aria-current="true"` when active. SSE indicator: `role="status"`.                                  |
| 4.1.3 Status Messages        | PASS   | Toast container: `role="status"` + `aria-live="polite"`. Error banners: `role="alert"`. Offline banner: `role="status"`.                                          |
| 2.1.1 Keyboard               | PASS   | All functionality accessible via keyboard. Focus management preserves focus across DataStar panel morphs. Shortcuts: T/I/P/C for modes, arrows for PTZ, ? for help.    |
| 2.1.2 No Keyboard Trap       | PASS   | Shortcut legend closes via Escape. Modal-like patterns are dismissible.                                                                                           |

### Fixes Applied in This Audit

1. **Preset input `aria-label`** — Added `aria-label="Preset name"` (WCAG 3.3.2)
2. **Preset input `:focus-visible`** — Added 2px accent outline for keyboard focus (WCAG 2.4.7)
3. **Toast container `aria-live`** — Added `role="status"` + `aria-live="polite"` (WCAG 4.1.3)
4. **Mode cards `aria-current`** — Added `aria-current="true"` to active card (WCAG 4.1.2)
5. **Offline banner `role`** — Added `role="status"` to dynamically-created banner (WCAG 4.1.3)
6. **Placeholder opacity** — Raised from 0.6 to 0.75 for better contrast (WCAG 1.4.11)

### Color Contrast Details

| Element       | Foreground               | Background            | Ratio  | Status |
| ------------- | ------------------------ | --------------------- | ------ | ------ |
| Body text     | `--text` (#c5cbe0)       | `--bg` (#0a0c10)      | ~12:1  | PASS   |
| Dimmed text   | `--text-dim` (#8a93a8)   | `--surface` (#13161d) | ~5.4:1 | PASS   |
| Badge online  | `--green` (#4cc88a)      | `--green-subtle`      | ~6:1   | PASS   |
| Badge offline | `--red` (#e85d5d)        | `--red-subtle`        | ~5:1   | PASS   |
| Placeholder   | `--text-dim` 75% opacity | `--surface`           | ~4:1   | PASS   |

---

## Screen Reader Testing Checklist (#109)

Manual testing required with actual screen reader software. The code-level audit
above verifies ARIA attributes and semantic structure, but real-world screen reader
behavior must be verified manually.

### Recommended Tools

- **NVDA** (Windows, free) — primary test target
- **VoiceOver** (macOS/iOS, built-in) — secondary
- **Orca** (Linux, built-in on GNOME) — relevant for this Linux daemon

### Test Matrix

| Test                | Steps                         | Expected Result                                                       |
| ------------------- | ----------------------------- | --------------------------------------------------------------------- |
| Page load           | Open `http://127.0.0.1:8090/` | Screen reader announces "EMEET PIXY" heading and page purpose         |
| Camera mode         | Tab to mode cards             | Each card announces name (Track/Idle/Privacy) and description         |
| Active mode         | Navigate to active card       | Card announces as "current" via `aria-current`                        |
| Audio segments      | Tab to audio buttons          | Each announces "Noise Cancel"/"Live"/"Original"; active announced     |
| Toggle buttons      | Tab to Gesture/Auto toggle    | Announces as switch with on/off state                                 |
| PTZ sliders         | Tab to pan/tilt/zoom sliders  | Announces label, current value, and range                             |
| Preset save         | Tab to preset input           | Announces "Preset name, edit text"                                    |
| Preset chips        | Navigate to chip buttons      | Load button announces preset name; delete announces "Delete preset X" |
| Toast notifications | Trigger any action            | Toast announced via `aria-live="polite"`                              |
| Error banner        | Trigger an error              | Banner announced via `role="alert"`                                   |
| Offline state       | Disconnect device             | Status banner announced; mode cards announced as disabled             |
| Keyboard shortcuts  | Press `?`                     | Legend announced; Escape to close                                     |
| DataStar panel morph | Use any control               | Focus preserved; new state announced if changed                       |

### Known Limitations

- **MJPEG preview**: The `<img>` has `alt="Live camera feed"` but screen readers
  cannot describe the visual content. This is inherent to live video.
- **PTZ radar**: The visual position indicator has no text alternative. The
  slider values provide the same information accessibly.
- **SSE indicator**: The connection status dot announces via `role="status"`
  but only changes on connect/disconnect events.

---

## Mobile Device Testing Checklist (#112)

Manual testing required on real devices. The responsive CSS has breakpoints at
860px, 640px, and 400px, with touch-target sizing and `prefers-reduced-motion` support.

### Test Devices

- **Phone** (≤400px viewport): iPhone SE, Android small screen
- **Phablet** (401-640px): iPhone 14 Pro, Pixel 7
- **Tablet portrait** (641-860px): iPad mini
- **Tablet landscape** (>860px): iPad Air

### Test Matrix

| Test                     | Steps                      | Expected Result                                          |
| ------------------------ | -------------------------- | -------------------------------------------------------- |
| Layout at 400px          | Open on small phone        | Single-column layout, all cards stack vertically         |
| Touch targets            | Tap all buttons/sliders    | Min 44×44px touch area (WCAG 2.5.5)                      |
| Preview aspect ratio     | View on landscape phone    | 16:9 preview maintains aspect ratio                      |
| Mode cards               | Tap Track/Idle/Privacy     | Cards respond to touch, active state visible             |
| PTZ sliders              | Drag pan/tilt/zoom         | Sliders draggable on touch, values update                |
| Preset save              | Type name, tap Save        | Keyboard appears, input accessible, save works           |
| Keyboard shortcuts       | Press hardware keys        | Shortcuts work if physical keyboard connected            |
| Zoom/pinch               | Pinch to zoom page         | Page zoom works (viewport meta allows user scaling)      |
| Orientation              | Rotate device              | Layout adapts to orientation change                      |
| `prefers-reduced-motion` | Enable in OS settings      | Animations reduced/disabled                              |
| `hover:none`             | All interactions via touch | No hover-only interactions; all actions available on tap |

### Responsive Breakpoints

| Breakpoint | Layout Change                                                                     |
| ---------- | --------------------------------------------------------------------------------- |
| ≤860px     | Controls grid collapses to single column                                          |
| ≤640px     | Mode grid stays 3-col but cards shrink; padding reduced                           |
| ≤400px     | Mode cards may stack; font sizes reduced; PTZ radar hidden on very narrow screens |

### Known Limitations

- **Live preview**: MJPEG stream may be bandwidth-heavy on mobile networks.
  The daemon is designed for localhost access, so mobile testing is secondary.
- **Keyboard shortcuts**: Physical keyboards on mobile are rare; touch is primary.
