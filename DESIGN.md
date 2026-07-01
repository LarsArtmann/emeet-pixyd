# Design

## Theme

Dark mode only. The user is at a desk, often in a dimly lit room, monitoring a webcam. A dark surface reduces eye strain and makes the live camera preview feel like a viewfinder. The ambient light is low; the screen should not emit glare.

## Color Strategy

Restrained: tinted neutrals with one accent (blue) and semantic state colors (green, yellow, red). No decorative gradients. No glassmorphism.

### Palette

| Role             | Value                      | Usage                                    |
| ---------------- | -------------------------- | ---------------------------------------- |
| Background       | `#0a0c10`                  | Page background, warm-tinted dark        |
| Surface          | `#13161d`                  | Cards, panels, solid blocks              |
| Surface Elevated | `#1a1e28`                  | Hover states, raised elements            |
| Border           | `#252a38`                  | Subtle dividers, card borders            |
| Border Strong    | `#303648`                  | Active borders, focus rings              |
| Text Primary     | `#e8ecf3`                  | Headings, primary labels                 |
| Text Secondary   | `#8a93a8`                  | Captions, meta, disabled                 |
| Accent           | `#5b8def`                  | Primary actions, focus, active selection |
| Accent Subtle    | `rgba(91, 141, 239, 0.15)` | Active button backgrounds                |
| Success          | `#4cc88a`                  | Tracking, online, positive state         |
| Warning          | `#e5a13d`                  | Idle, caution state                      |
| Error            | `#e85d5d`                  | Privacy, offline, negative state         |

Neutrals are tinted toward a cool blue-gray. No pure black (`#000`) or pure white (`#fff`).

## Typography

System font stack: `Inter, SF Pro Display, system-ui, -apple-system, sans-serif`.

| Level         | Size    | Weight | Usage                        |
| ------------- | ------- | ------ | ---------------------------- |
| Page title    | 1.25rem | 700    | Header brand                 |
| Section label | 0.65rem | 600    | Card titles, uppercase       |
| Body          | 0.85rem | 400    | Labels, toggle text          |
| Data          | 1.1rem  | 600    | State values, indicators     |
| Caption       | 0.72rem | 500    | Meta, shortcuts, last synced |

Line length capped at 75ch for any prose. Compact UI runs denser.

## Elevation

No blur-based elevation. Elevation is communicated through:

- Border color shifts (`--border` to `--border-strong`)
- Background shifts (`--surface` to `--surface-elevated`)
- Subtle box shadows for focus/active states only

## Components

### Buttons

- 1px border, rounded 8px
- Hover: background elevates, border brightens, slight translateY(-1px)
- Active: pressed state (translateY(0), no shadow)
- Active-state buttons: solid semantic color background, no glow effects

### Toggles

- Pill shape, 40x22px
- Track: muted background
- Thumb: solid off-white circle
- On state: solid semantic color, no glow

### Sliders

- 5px track, muted background
- 16px circular thumb in accent color
- Hover: thumb scales slightly

### Cards

- Solid surface background
- 1px border
- 12px radius
- No backdrop blur

### Preview

- Dark frame (`#050608`) with 1px border
- 16:9 aspect ratio
- LIVE indicator: solid red dot + text label

## Motion

- Duration: 150ms for state changes, 200ms for reveals
- Easing: `cubic-bezier(0.4, 0, 0.2, 1)` (ease-out)
- Only animate: opacity, transform, background-color, border-color
- No layout property animation
- Reduced motion: all transitions collapse to 0.01ms

## Layout

- Max-width 1020px container
- 2-column grid on desktop, single column on mobile (720px breakpoint)
- Generous padding (2rem top, 1.5rem sides)
- Preview card spans the visual weight of the left column

## Architecture Patterns

- **CommandResult**: All command handlers return a typed `CommandResult` struct (not raw strings). Provides `.String()` for backward compatibility and `.IsError()` for error checking.
- **Dependencies struct**: Nine DI function fields consolidated into single `Dependencies` struct. Production wiring in `NewDaemon()`, tests override individual fields.
- **HIDDevice interface**: Abstracts HID communication (`Send`, `SendRecv`). Production implementation wraps `/dev/hidraw*`, enabling test doubles.
- **Middleware**: Security headers, request logging, and request ID middleware are implemented locally in `http.go` (was briefly via `cqrs-htmx/v2`, removed). Uses a simple `chain()` function for composition.
- **Metrics**: OTel counters/gauges registered lazily via `sync.Once`. Includes command, probe, uevent, HID failure, stream duration, and frame counters. `promhttp.Handler()` (from `prometheus/client_golang`) serves the `/metrics` endpoint; all metric definitions use the OTel SDK.
- **Circuit breaker**: HID failures tracked via `hidFailCount`. After 3 consecutive failures, commands skip HID re-probe. Reset on success or successful device probe.
