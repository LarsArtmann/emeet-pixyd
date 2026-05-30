# Product

## Register

product

## Users

Linux desktop users with an EMEET PIXY dual-camera AI webcam. Technical, likely developers or power users. Using the web UI to monitor and control their webcam state during video calls or general desktop use. Context: sitting at a desk, possibly in a dim room, glancing at the control panel to verify camera status or adjust settings.

## Product Purpose

A local daemon control panel for the EMEET PIXY webcam. Provides real-time camera preview, PTZ control, audio mode switching, gesture toggles, and auto-management. Success means the user can instantly see camera state, make adjustments without friction, and trust the daemon is managing their camera correctly.

## Brand Personality

Precise. Capable. Unobtrusive. Three words: "tool-like", "reliable", "focused".

## Anti-references

- Generic SaaS dashboards with card grids and gradient hero sections
- Glassmorphism-heavy designs that feel decorative rather than functional
- Consumer webcam software with bubbly icons and playful animations
- Overly technical or terminal-native aesthetics that feel hostile

## Design Principles

- The interface should feel like a hardware control panel, not a marketing site
- State visibility is paramount: the user must know camera status at a glance
- Controls should feel tactile and responsive, with immediate feedback
- No decorative motion: every animation conveys state change
- Density is acceptable for a tool the user interacts with frequently

## Accessibility & Inclusion

- WCAG 2.1 AA target
- Keyboard shortcuts for all primary actions
- Reduced motion support via `prefers-reduced-motion`
- Sufficient contrast for state indicators (color + shape/shadow)
