# EMEET PIXY Daemon — Feature Inventory

**Updated:** 2026-06-06

---

## Camera Control

| Feature               | Description                                                   | Status              |
| --------------------- | ------------------------------------------------------------- | ------------------- |
| **Face Tracking**     | AI face tracking via HID commands                             | ✅ FULLY_FUNCTIONAL |
| **Idle Mode**         | Camera powered on, no tracking                                | ✅ FULLY_FUNCTIONAL |
| **Privacy Mode**      | Physically blocks camera lens via HID                         | ✅ FULLY_FUNCTIONAL |
| **Toggle Privacy**    | Switches between tracking ↔ privacy                           | ✅ FULLY_FUNCTIONAL |
| **Center Camera**     | Resets pan=0, tilt=0, zoom=100 via `v4l2-ctl`                 | ✅ FULLY_FUNCTIONAL |
| **PTZ Sliders (Web)** | Pan (±170°), Tilt (±30°), Zoom (100–400×) with 300ms debounce | ✅ FULLY_FUNCTIONAL |
| **PTZ CLI**           | `pan/tilt/zoom <value>` via socket/CLI with clamping          | ✅ FULLY_FUNCTIONAL |
| **PTZ Relative Mode** | `pan+10`, `tilt-5` for relative adjustments via socket/CLI    | ✅ FULLY_FUNCTIONAL |
| **Keyboard PTZ**      | Arrow keys: pan/tilt ±5°, +/-: zoom ±10                       | ✅ FULLY_FUNCTIONAL |

## Audio

| Feature                       | Description                                          | Status              |
| ----------------------------- | ---------------------------------------------------- | ------------------- |
| **Audio Modes**               | NC / Live / Original via HID + web UI buttons        | ✅ FULLY_FUNCTIONAL |
| **Audio Cycle**               | `audio` without arg cycles NC → Live → Original → NC | ✅ FULLY_FUNCTIONAL |
| **PipeWire Source Switching** | `wpctl set-default` on call start (full auto mode)   | ✅ FULLY_FUNCTIONAL |

## Auto-Management

| Feature                | Description                                                            | Status              |
| ---------------------- | ---------------------------------------------------------------------- | ------------------- |
| **Call Detection**     | `/proc/*/fd` scanning every 2s, excludes self/descendants              | ✅ FULLY_FUNCTIONAL |
| **Auto Full**          | Call start → tracking + NC audio + PipeWire source; Call end → privacy | ✅ FULLY_FUNCTIONAL |
| **Auto Tracking-Only** | Call start → tracking; Call end → privacy                              | ✅ FULLY_FUNCTIONAL |
| **Auto Privacy-Only**  | Call start → nothing; Call end → privacy                               | ✅ FULLY_FUNCTIONAL |
| **Auto Off**           | Disables all automatic actions                                         | ✅ FULLY_FUNCTIONAL |
| **Debounce**           | Requires N consecutive polls (default 3) before state change           | ✅ FULLY_FUNCTIONAL |

## Gesture Control

| Feature            | Description                                 | Status              |
| ------------------ | ------------------------------------------- | ------------------- |
| **Gesture Toggle** | Enable/disable hand gesture control via HID | ✅ FULLY_FUNCTIONAL |

## Web UI (`http://127.0.0.1:8090`)

| Feature                      | Description                                                                     | Status              |
| ---------------------------- | ------------------------------------------------------------------------------- | ------------------- |
| **Live MJPEG Preview**       | ffmpeg → MJPEG stream, single-client semaphore                                  | ✅ FULLY_FUNCTIONAL |
| **Snapshot API**             | `GET /api/snapshot` returns last JPEG frame                                     | ✅ FULLY_FUNCTIONAL |
| **HTMX Auto-Refresh**        | Status panel polls `/panel` every 3s, pauses when tab hidden                    | ✅ FULLY_FUNCTIONAL |
| **Toast Notifications**      | In-browser success/info/error toasts, auto-dismiss 2.5s                         | ✅ FULLY_FUNCTIONAL |
| **Keyboard Shortcuts**       | T=Track, I=Idle, P=Privacy, C=Center (disabled in inputs)                       | ✅ FULLY_FUNCTIONAL |
| **Offline Banner**           | "Daemon unreachable" after 3 consecutive failures                               | ✅ FULLY_FUNCTIONAL |
| **PTZ Visual Feedback**      | Slider "sending" state, reverts to last-known-good on failure                   | ✅ FULLY_FUNCTIONAL |
| **Dark Glassmorphism Theme** | Dark CSS with glass cards, pulsing dots, responsive 2→1 col grid                | ✅ FULLY_FUNCTIONAL |
| **Security Headers**         | CSP, X-Frame-Options DENY, Referrer-Policy, X-Content-Type-Options, request IDs | ✅ FULLY_FUNCTIONAL |

## CLI / Unix Socket

| Feature                 | Description                                                             | Status              |
| ----------------------- | ----------------------------------------------------------------------- | ------------------- |
| **Unix Socket Control** | All commands via `/run/emeet-pixyd/control.sock`                        | ✅ FULLY_FUNCTIONAL |
| **Status**              | Full status string (camera, audio, gesture, PTZ, in-call, auto, device) | ✅ FULLY_FUNCTIONAL |
| **Sync**                | Queries hardware via HID, reconciles with daemon state                  | ✅ FULLY_FUNCTIONAL |
| **Probe**               | Re-scans sysfs for video4linux + hidraw devices                         | ✅ FULLY_FUNCTIONAL |
| **Device**              | Returns both `/dev/videoX` and `/dev/hidrawY` paths                     | ✅ FULLY_FUNCTIONAL |
| **Waybar Output**       | JSON with icon, class, tooltip for Waybar custom module                 | ✅ FULLY_FUNCTIONAL |
| **--version / --help**  | `emeet-pixyd --version` prints version, `--help` prints usage           | ✅ FULLY_FUNCTIONAL |

## Desktop Notifications

| Feature                   | Description                                            | Status              |
| ------------------------- | ------------------------------------------------------ | ------------------- |
| **Desktop Notifications** | `notify-send -a emeet-pixyd` for call start/end alerts | ✅ FULLY_FUNCTIONAL |

## Device Management

| Feature               | Description                                                     | Status              |
| --------------------- | --------------------------------------------------------------- | ------------------- |
| **Device Probing**    | sysfs walks matching vendor `328f`/product `00c0`               | ✅ FULLY_FUNCTIONAL |
| **Hotplug Detection** | Netlink uevent listener, re-probes + syncs on device appearance | ✅ FULLY_FUNCTIONAL |

## State Persistence

| Feature               | Description                                                  | Status              |
| --------------------- | ------------------------------------------------------------ | ------------------- |
| **State Persistence** | JSON at `{StateDir}/state.json`, atomic write (tmp + rename) | ✅ FULLY_FUNCTIONAL |
| **SIGHUP Save**       | Save state without shutdown                                  | ✅ FULLY_FUNCTIONAL |

## Monitoring & Observability

| Feature                 | Description                                                                                                                                                         | Status              |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------- |
| **Prometheus Metrics**  | OTel gauges + counters at `/metrics`: in_call, auto_mode, camera_state, command_total, probe_total, uevent_total, hid_failures_total, stream_duration, frames_total | ✅ FULLY_FUNCTIONAL |
| **pprof Debug**         | `/debug/pprof/*` gated behind `EMEET_PIXYD_DEBUG=true`                                                                                                              | ✅ FULLY_FUNCTIONAL |
| **Health Endpoint**     | `/api/health` returns JSON status + 503 when device offline                                                                                                         | ✅ FULLY_FUNCTIONAL |
| **systemd Integration** | `sd_notify` READY=1 + WATCHDOG=1 each poll tick                                                                                                                     | ✅ FULLY_FUNCTIONAL |

## HID Communication

| Feature                 | Description                                                    | Status              |
| ----------------------- | -------------------------------------------------------------- | ------------------- |
| **HID Config+Commit**   | 9-byte config + 4-byte commit via hidraw, 200ms sleep          | ✅ FULLY_FUNCTIONAL |
| **HID State Query**     | Generic `queryHIDState[T]` with 500ms timeout response parsing | ✅ FULLY_FUNCTIONAL |
| **HID Circuit Breaker** | Stops retrying after 3 consecutive failures, resets on success | ✅ FULLY_FUNCTIONAL |

## NixOS Module

| Feature               | Description                                                                 | Status              |
| --------------------- | --------------------------------------------------------------------------- | ------------------- |
| **NixOS Integration** | `hardware.emeet-pixy` options, udev rules, systemd user service, tmpfiles.d | ✅ FULLY_FUNCTIONAL |

## Nix Build

| Feature       | Description                                                            | Status              |
| ------------- | ---------------------------------------------------------------------- | ------------------- |
| **Nix Flake** | `nix build`, `nix run`, `nix flake check` with `proxyVendor` for templ | ✅ FULLY_FUNCTIONAL |

---

## Summary

- **Total features:** 49
- **Fully functional:** 49
- **Partially functional:** 0
- **Broken:** 0
- **Planned:** 0

All features are production-ready and fully implemented. The codebase has comprehensive test coverage for auto-management, process detection, commands, handlers, and HID protocol parsing.
