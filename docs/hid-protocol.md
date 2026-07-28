# EMEET PIXY HID Protocol — Reverse-Engineering Findings

> **Source:** Extracted from `hid.go`, `device.go`, and empirical testing against
> the EMEET PIXY dual-camera AI webcam (USB vendor `328f`, product `00c0`).
>
> **Last verified:** 2026-07-28 against the codebase.

---

## Overview

The EMEET PIXY exposes a HID interface over `/dev/hidraw*` for camera mode control
(tracking, privacy, idle), audio mode selection (noise cancellation, live, original),
and gesture toggle. All communication uses **config reports** — 9-byte command packets
followed by 4-byte commit packets, with a 200ms sleep between them.

PTZ (pan/tilt/zoom) is controlled separately via **V4L2** (`v4l2-ctl`), not HID.

---

## Device Identification

| Property | Value |
| -------- | ----- |
| USB Vendor ID | `328f` |
| USB Product ID | `00c0` |
| Video device | `/dev/video*` (V4L2) |
| HID device | `/dev/hidraw*` |
| Name matching | Contains `"EMEET"`, `"Pixy"`, or `"PIXY"` |

Device probing walks `/sys/class/video4linux` and `/sys/class/hidraw` to match vendor/product.

---

## Buffer Sizes

| Constant | Value | Purpose |
| -------- | ----- | ------- |
| `hidBufSize` | 32 bytes | Write buffer (padded to 32, only first 9 bytes carry data) |
| `hidRespBufSize` | 64 bytes | Read buffer for responses |
| `hidMinLen` | 9 bytes | Minimum meaningful response length |
| `hidDebugLen` | 16 bytes | Bytes logged in debug mode |
| `hidResponseMs` | 500 ms | Read timeout |
| `hidCommandSleepMs` | 200 ms | Sleep between config and commit |

---

## Config Report (Command Packet)

The `pixyConfig(iface, modeByte)` function builds a 9-byte config report:

```
Byte 0: 0x09 (cameraConfigPrefix — report ID)
Byte 1: Interface ID (see below)
Byte 2: 0x01 (cameraConfigMarker)
Byte 3: 0x00
Byte 4: 0x00
Byte 5: 0x01 (cameraConfigMarker)
Byte 6: 0x00
Byte 7: 0x01 (cameraConfigMarker)
Byte 8: Mode byte (varies by interface)
```

### Interface IDs

| Interface | Value | Purpose |
| --------- | ----- | ------- |
| Tracking | `0x01` (`hidInterfaceTracking`) | Camera tracking mode |
| Gesture | `0x04` (`hidInterfaceGesture`) | Hand gesture control on/off |
| Audio | `0x05` (`hidInterfaceAudio`) | Audio processing mode |

### Mode Bytes — Camera (Interface `0x01`)

| Mode | Byte Value | Description |
| ---- | ---------- | ----------- |
| Idle | `0x00` (`hidByteIdle`) | Camera powered, no tracking |
| Tracking | `0x01` (`hidByteTracking`) | Active face tracking |
| Privacy | `0x02` (`hidBytePrivacy`) | Lens physically blocked |

### Mode Bytes — Audio (Interface `0x05`)

| Mode | Byte Value | Description |
| ---- | ---------- | ----------- |
| Noise Cancel | `0x01` (`hidByteNC`) | Active noise cancellation |
| Live | `0x02` (`hidByteLive`) | Optimized for live/streaming |
| Original | `0x03` (`hidByteOriginal`) | Raw passthrough, no processing |

### Mode Bytes — Gesture (Interface `0x04`)

| State | Byte Value | Description |
| ----- | ---------- | ----------- |
| Disabled | `0x00` (`hidByteIdle`) | Gesture control off |
| Enabled | `0x01` (`gestureEnabledByte`) | Gesture control on |

---

## Commit Report

The `pixyCommit(iface)` function builds a 4-byte commit packet that applies
the preceding config:

```
Byte 0: 0x09 (cameraConfigPrefix)
Byte 1: Interface ID (same as the config report)
Byte 2: 0x01 (cameraConfigMarker)
Byte 3: Interface ID (repeated)
```

---

## Write Protocol

All commands use the `Send(report []byte)` method which:

1. Opens the hidraw device in write-only mode (`O_WRONLY`)
2. Pads the report to `hidBufSize` (32 bytes) with zeros
3. Writes the full 32-byte buffer
4. Closes the device

The full command sequence for any state change is:

```
1. Send pixyConfig(iface, modeByte)  →  32-byte write
2. Sleep 200ms (hidCommandSleepMs)
3. Send pixyCommit(iface)            →  32-byte write
```

---

## Query Protocol (Read State)

State queries use `SendRecv(ctx, report)` which does a bidirectional operation:

1. Opens hidraw in read-write mode (`O_RDWR`)
2. Writes the query payload
3. Reads a response (up to 64 bytes) with a 500ms timeout
4. Closes the device

### Query Payloads

**Tracking query:**
```
[0x09, 0x01, 0x01, 0x01]
```

**Audio query:**
```
[0x09, 0x05, 0x00, 0x04]
```

**Gesture query:**
```
[0x09, 0x04, 0x02, 0x01, 0x00, 0x01, 0x00, 0x01, 0x02]
```

### Response Parsing

Responses are parsed by `parseHIDResponse(data []byte)`:

```
Byte 0: Report prefix (must be 0x09)
Byte 1: Interface ID
Bytes 2-7: Markers/padding (ignored in parsing)
Byte 8: Mode byte (the actual state value)
```

The parser switches on `data[0] == 0x09` and `data[1]` to determine which
interface's response this is, then reads `data[8]` as the mode byte.

For gesture responses, the last byte (`data[len(data)-1]`) is checked
against `gestureEnabledByte` (0x01) for the on/off state.

If the response is shorter than `hidMinLen` (9 bytes) or the prefix/interface
combination is unrecognized, `Got` is set to `false` and the caller reports an error.

---

## Circuit Breaker

The daemon implements a HID circuit breaker with a threshold of 3 consecutive
failures. After 3 failures, `probeDevices()` is triggered to re-scan for the
device (hotplug recovery). The counter resets on the first success.

---

## V4L2 PTZ Controls (Separate from HID)

PTZ is controlled via V4L2 `v4l2-ctl`, not HID:

| Control | V4L2 Name | Range | Units |
| ------- | --------- | ----- | ----- |
| Pan | `pan_absolute` | -540000 to 540000 | 1/3600 degree |
| Tilt | `tilt_absolute` | -324000 to 324000 | 1/3600 degree |
| Zoom | `zoom_absolute` | 100 to 150 | Multiplier |

**Conversion:** User-facing degrees × 3600 = V4L2 units (for pan/tilt only; zoom is not multiplied).

**Sign convention:** Positive tilt = up (everywhere: V4L2 read, V4L2 write, web slider, keyboard).

---

## Unknown / Open Questions

- **Bytes 3-7 in config reports:** Their purpose is unknown. The current
  implementation follows the pattern observed from the official Windows driver
  packet captures. They may carry checksums, sequence numbers, or reserved fields.
- **Response bytes 2-7:** Ignored by the parser. May contain version info,
  timestamps, or device health data.
- **Boot/handshake sequence:** The official driver may send an initialization
  sequence on connect that the daemon does not replicate (the daemon works
  without it, suggesting it's optional).
- **Firmware update protocol:** Not reverse-engineered. Likely uses a different
  HID report ID.

---

## References

- `hid.go` — HID constants, buffer construction, response parsing
- `device.go` — High-level state-change functions (`setTracking`, `setAudio`, `setGesture`)
- `probe.go` — Device identification and sysfs walking
- `ptz.go` — V4L2 PTZ control (separate from HID)
