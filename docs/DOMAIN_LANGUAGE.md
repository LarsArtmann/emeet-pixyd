# Domain Language

A **Ubiquitous Language** for **emeet-pixyd** — shared across users, developers,
and AI agents. Inspired by Domain-Driven Design (DDD).

Every term below means the **same thing** to everyone who reads it. The daemon
code (`internal/pixy/`) is the source of truth for these definitions.

## Glossary

| Term            | Definition                                                                               | Context / Where used              |
| --------------- | ---------------------------------------------------------------------------------------- | --------------------------------- |
| PIXY            | The EMEET PIXY dual-camera AI webcam (USB `328f:00c0`). The hardware this daemon drives. | Product name, udev rules, probing |
| Daemon          | The long-running `emeet-pixyd` background service.                                       | systemd unit, lifecycle           |
| Call            | A video call or camera session. Detected when any process opens `/dev/video*`.           | Auto-management, call detection   |
| In-Call         | State where the camera device is open by a non-daemon process.                           | `State.InCall`, auto.go           |
| Tracking        | Camera actively following faces via the on-device AI.                                    | `CameraState`, HID                |
| Privacy         | Camera lens physically blocked by the hardware shutter.                                  | `CameraState`, HID                |
| Idle            | Camera powered on but not tracking and not blocked.                                      | `CameraState`                     |
| Offline         | No PIXY device detected (unplugged or not probed).                                       | `CameraState`                     |
| PTZ             | Pan / Tilt / Zoom — the motorized camera position axes.                                  | `Axis`, `PTZValues`, ptz.go       |
| Preset          | A named, saved PTZ position that can be recalled later.                                  | `PresetMap`, state.json           |
| Snapshot        | A single still JPEG frame captured from the live stream.                                 | `/api/snapshot`, stream.go        |
| Debounce        | N consecutive polling cycles required before a state transition is committed.            | `DebounceCount`, auto.go          |
| Probe           | Scan sysfs (`/sys/class/video4linux`, `/sys/class/hidraw`) to find the PIXY.             | `probeDevices()`, probe.go        |
| Hotplug         | USB plug/unplug event detected via netlink uevent, triggering a re-probe.                | uevent.go                         |
| HID             | Human Interface Device protocol over `/dev/hidraw*` used for camera control.             | hid.go                            |
| Config + Commit | The two-phase HID write: a 9-byte config report followed by a 4-byte commit report.      | hid.go, 200ms inter-report delay  |
| Circuit Breaker | HID failure tracker: 3 consecutive failures trigger a device re-probe.                   | device.go                         |
| PipeWire Source | The PIXY microphone as a PipeWire audio source, switched via `wpctl`.                    | process.go, `SourceID`            |
| Waybar          | Status-bar integration producing JSON for a custom Waybar module.                        | waybar.go                         |

## Entities

Objects with identity and lifecycle.

| Term   | Definition                                                                  | Identity / Lifecycle                                          |
| ------ | --------------------------------------------------------------------------- | ------------------------------------------------------------- |
| Daemon | The running process owning the device, web UI, socket, and polling loop.    | Started by systemd; runs until signal/shutdown                |
| State  | The persisted runtime state of the daemon (camera mode, audio, presets...). | Persisted in `state.json`; schema-versioned (`SchemaVersion`) |
| Preset | A named PTZ position saved in state, recalled by name.                      | Created/loaded/deleted by name; max 16 (`MaxPresets`)         |

## Value Objects

Immutable objects defined by their attributes.

| Term        | Definition                                                          | Type / Values                                                |
| ----------- | ------------------------------------------------------------------- | ------------------------------------------------------------ |
| CameraState | The operating mode of the camera lens/AI.                           | `idle` \| `tracking` \| `privacy` \| `offline`               |
| AudioMode   | The microphone noise-cancellation strategy.                         | `nc` \| `live` \| `original`                                 |
| AutoMode    | The automatic camera-management strategy applied on call start/end. | `off` \| `full` \| `tracking-only` \| `privacy-only`         |
| Axis        | A PTZ axis name (branded type to prevent string mixing).            | `pan` \| `tilt` \| `zoom`                                    |
| PTZValues   | The pan/tilt/zoom position triple.                                  | `PTZValues{Pan, Tilt, Zoom int}`                             |
| Range       | An inclusive `[Min, Max]` limit for one PTZ axis, with `Clamp()`.   | Pan ±150°, Tilt ±90°, Zoom 100-150× (hardware-verified)      |
| PID         | Branded process ID (phantom-typed, prevents mixing with SourceID).  | `pixy.PID` — used in `/proc` scanning                        |
| SourceID    | Branded PipeWire source ID (phantom-typed).                         | `pixy.SourceID` — used in `wpctl` switching                  |
| Config      | Daemon configuration derived from environment variables.            | `StateDir`, `WebAddr`, `PollInterval`, `DebounceCount`, etc. |

## Events

Things that happen in the domain.

| Event            | When it happens                                         | Effect                                          |
| ---------------- | ------------------------------------------------------- | ----------------------------------------------- |
| Call Start       | Camera device opened by a process (after debounce).     | Tracking + audio + source switch (per AutoMode) |
| Call End         | Camera device released (after debounce).                | Privacy mode (per AutoMode)                     |
| Hotplug (uevent) | USB add/remove for the PIXY.                            | Re-probe devices, update state                  |
| State Changed    | Any mutation to camera/audio/gesture/PTZ/in-call state. | Broadcast to all SSE clients, persist to disk   |

## Commands

Actions the system can perform (via Unix socket, CLI, or web UI).

| Command                 | Effect                                                |
| ----------------------- | ----------------------------------------------------- |
| track / idle / privacy  | Set camera mode via HID.                              |
| toggle-privacy          | Switch between tracking and privacy.                  |
| center                  | Reset pan=0, tilt=0, zoom=100 via V4L2.               |
| pan/tilt/zoom \<value\> | Set a PTZ axis (absolute, or `rel+/-N` for relative). |
| audio [mode]            | Set or cycle audio mode (NC → Live → Original → NC).  |
| gesture-on / off        | Toggle hand-gesture control via HID.                  |
| auto [mode]             | Set or report the auto-management strategy.           |
| preset save/load/delete | Manage named PTZ presets.                             |
| sync                    | Query hardware via HID, reconcile daemon state.       |
| probe                   | Re-scan sysfs for the PIXY.                           |
| status / device         | Report current state / device paths.                  |
| waybar                  | Emit JSON for a Waybar module.                        |

## Bounded Contexts

| Context          | Description                                                                 | Boundary                              |
| ---------------- | --------------------------------------------------------------------------- | ------------------------------------- |
| Camera Control   | HID-driven lens/AI behavior (tracking, privacy, audio, gesture).            | hid.go, device.go                     |
| PTZ / Position   | V4L2-driven motor control (pan, tilt, zoom, presets, readback).             | ptz.go, v4l2 via `v4l2-ctl`           |
| Auto-Management  | Call detection + state transitions driven by `/proc` scanning and debounce. | auto.go, process.go                   |
| Web UI           | HTTP/HTMX/SSE presentation layer and user interactions.                     | handlers.go, templates.templ, static/ |
| Persistence      | Schema-versioned JSON state, atomic write, socket IPC.                      | state.go, socket.go                   |
| Device Discovery | sysfs probing and netlink hotplug.                                          | probe.go, uevent.go                   |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
