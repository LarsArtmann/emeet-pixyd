---
format: 1920x1080
duration: 25s
message: "A great AI webcam, dumb on Linux — until emeet-pixyd gives Linux every feature, automatically."
arc: Hook → Problem → Solution → CTA
audience: Linux users owning an EMEET PIXY
mode: autonomous
---

# Demo video reconstruction (lost-source insurance)

Reconstruction of the committed `website/public/demo.mp4` (25s) whose HyperFrames
composition source was lost in `/tmp` (plan M20 / TODO #130). Scene timings,
copy, palette, and layout were read back from the rendered artifact frame by
frame. The committed mp4 remains the reference; this project exists so
re-renders become one command.

## Frame 1 — Title

- status: outline
- duration: 6s
- transition_in: fade
- scene: Hook — the pitch line
- poster: 4

Dark near-black background (#0d0a09 family) with a large, very dim violet radial
glow behind center. Copy, centered:

- Eyebrow (violet #8b5cf6, mono, letterspaced, small): `EMEET PIXY × LINUX`
- H1 line 1 (off-white, huge, bold): `A great AI webcam.`
- H1 line 2 (violet, same size): `Dumb on Linux.`
- Sub (gray, appears ~2s in): `The vendor ships software for Windows and macOS only.`

Motion: text fades/rises in staggered (eyebrow → line 1 → line 2 → sub);
background glow breathes slowly. No narration (the original was unnarrated).

## Frame 2 — Problem

- status: outline
- duration: 5s
- transition_in: crossfade
- scene: the three locked features
- poster: 3

Same background. Centered white H2: `Everything smart about it, locked away.`
Below: three dark feature cards in a row (Face Tracking / Privacy Shutter /
Noise Cancelling), each with a red mono `× unavailable` line. Under the row, a
red-outlined mono badge: `WINDOWS & macOS ONLY`.

Motion: cards pop in staggered; badge border draws/brightens last.

## Frame 3 — UI recreation

- status: outline
- duration: 9s
- transition_in: crossfade
- scene: the daemon's control panel, recreated
- poster: 5

The heaviest frame: a faithful HTML recreation of the daemon's dark web panel
inside one large rounded card on the dark background:

- Left pane: red mono `CALL_ACTIVE` label, dark video viewport with a pulsing
  violet glow dot (the tracked face), rounded corners.
- Right pane, top: mono label `CAMERA MODE` and three mode cards —
  `Tracking / Face tracking on` (active, green glow), `Idle / Standby`,
  `Privacy / Lens blocked`.
- Right pane, middle: mono `AUDIO` label with a three-segment control —
  `Noise Canc.` (active violet), `Live`, `Original`.
- Far right: PTZ radar (concentric circles + crosshair + violet position dot)
  with mono readouts `pan +42°`, `tilt +18°`, `zoom 120`.
- Bottom: an empty status-bar strip across the card.

Motion: the face dot pulses (scale/opacity loop with finite repeats), the radar
dot eases between positions, pan/tilt/zoom numbers tick, mode cards light up
staggered. This mirrors the real UI's violet-on-dark glassmorphism.

## Frame 4 — Outro

- status: outline
- duration: 5s
- transition_in: crossfade
- scene: CTA
- poster: 3

Centered: H1 `Linux gets every feature.` (with `every feature` in violet) /
`Automatically.` (white), then a violet-bordered rounded chip with mono
violet text: `emeet-pixyd.lars.software`. Gentle rise-in; background glow as
Frame 1.
