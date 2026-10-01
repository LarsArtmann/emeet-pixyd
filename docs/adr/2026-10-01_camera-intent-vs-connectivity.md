# ADR: Camera State Is Intent — Connectivity Is Observed, Never Persisted

**Date:** 2026-10-01
**Status:** Accepted (engineering decision; closes findings 1, 2, 5, 6, 11 of the 2026-10-01 state/split-brain audit)
**Context:** `pixy.CameraState` was carrying four different concepts at once: the user's desired mode (intent), the observed hardware mode, device online/offline status, and — via `State` — session state. `applyProbeResultLocked` wrote `StateOffline` into the persisted `state.Camera` on every probe miss, so a transient probe miss could overwrite and persist the user's chosen mode; `hadPersistedState` was a boolean snapshot taken once at startup and never updated, so the "persisted intent wins on re-appear" reconcile ran against stale truth.

## Options

### Option A — Add a second persisted field (`Online bool` / `DesiredCamera`)

Rejected. It changes the on-disk JSON schema, needs a real migration, and duplicates information the daemon already observes at runtime (`d.videoDev`, `d.hidrawDev`). A persisted `Online` would also be a new split-brain: the file could disagree with the bus.

### Option B — Remove `StateOffline` from the enum entirely

Rejected. `StateOffline` is still the correct display value on the CLI `status` line (`camera=offline` while unplugged, a pre-existing contract) and in the Waybar map. Removing it would ripple through every surface for no gain.

### Option C — `Camera` is intent only; connectivity is observed and projected per surface — RECOMMENDED (shipped)

`State.Camera` is defined as the user's **desired** mode: one of `idle`, `tracking`, `privacy`. It is never `offline`; `CameraState.ValidDesired()` encodes that, `loadState` normalizes a legacy persisted `"offline"` to `privacy`, and `applyProbeResultLocked` no longer writes the field at all. Connectivity lives where it is observed — `devicePresence{Online, Controllable}` snapshot from `d.videoDev`/`d.hidrawDev` — and each surface reads it separately:

- `/api/status` and the web panel report `camera` (intent) plus `online`/`controllable` booleans.
- The CLI `status` line keeps its pre-existing collapse (prints `offline` when no video node).
- Metrics carry a dedicated `emeet_pixyd_online` gauge; `emeet_pixyd_camera_state` now reports the desired mode only.

`hadPersistedState bool` is replaced by `d.persistedIntent atomic.Bool`, set true by every successful `saveState()`. `syncState` takes `hidMu` (via `syncStateLocked` for callers that already hold it).

## Decision

Option C. The JSON shape is byte-compatible (the `Camera` field keeps its name and tags); `CurrentSchemaVersion` bumps to 2 only to record the *semantic* change — the file's `camera` no longer ever holds `offline`, and a v1 file carrying `offline` is normalized to `privacy` with a warning. Unplugging the camera can no longer modify or persist user intent, and the persisted-intent flag is derived from disk truth at the moment it is written.

## Consequences

- Findings 1 (probe-miss clobber), 2 (stale `hadPersistedState`), 5/6 (surface disagreement), and 11 (metrics overload) dissolve at one root rather than five patches.
- `StateOffline` survives as a display/projection value; `ValidDesired()` will reject it if anyone ever tries to persist it again.
- `sync` still does not reconcile `Speeds`/`TrackMode` (no verified hardware readback — TODO #166); that remains documented, not implemented.
