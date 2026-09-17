# HID Protocol — Official App Command Map

> Mapping between the official EMEET STUDIO 2.0.3 app's HID command surface and
> emeet-pixyd's implementation. Sources: mangled-symbol + debug-string extraction
> from the macOS binary (`/tmp/emeet/main_strings.txt`, 241 MB, richest source) and
> the Windows EXE (`win_strings.txt`), cross-checked against our `hid.go`,
> `device.go`, `commands.go`, and `docs/hid-protocol.md`.
>
> **2026-09-17.** Companion: `docs/emeet-studio-official-app-comparison.md` (§2.3 overlap, §4 gaps).

---

## 1. Official HID architecture

```
QML UI (EMBatteryLevel, PtzDeviceOptVM, ...)
  └── C++ controllers (EMDeviceViewController, HidConfigManage)
        └── EMHidCmdHelper          ← one hidCmdSend* per command, ~132 debug symbols
        │     ├── hidCmdSend(UsbHidCtrl*, const EMHidCmdV1Head&, const u8*, u16, bool, cb, u8)
        │     ├── hidCmdSend(UsbHidCtrl*, const EMHidCmdV2Head&, ...)      ← newer framing
        │     └── hidCmdSend(UsbHidCtrl*, HIDCMDTYPE, ...)                 ← generic
        ├── EMHidCmdV1RecvFsm / EMHidCmdV2RecvFsm  ← response state machines
        └── UsbHidCtrl                ← hidraw device wrapper (VideoDeviceManage::getUsbHidCtrlByDeviceInfo)
  └── EMNetCmdHelper (elink/WiFi)     ← same command set over the wireless transport
        └── lib_elink_* (motor_*, battery, ...)
```

- Two command headers exist: `EMHidCmdV1Head` and `EMHidCmdV2Head`, each with its
  receive FSM. Our 9-byte `0x09`-prefixed config/commit reports (hid-protocol.md)
  are one dialect of this family — likely V1-era.
- **`[HID_RACE_FIX]`**: the official app keeps a persistent HID *read thread* and
  pauses it (`setPause(true)`) before `hid_write` during firmware IC upgrades —
  they hit the same read/write interleaving hazard our single-shot `SendRecv`
  sidesteps by opening per operation.
- Response parsers are named `hidCmdParse<Cmd>(const u8* data, u8 len, <outs>)` —
  130+ of them, giving exact wire→struct layouts (§3).
- Every set command has a `_E` ("entry") and `_VAL` ("value") global
  (`EMHidCmdHelper::CMD_SET_MOTOR_SPEEDE`) — runtime-initialized command-ID
  constants. **The numeric command IDs are therefore NOT string-recoverable**;
  they live in .data, initialized at load (see §6).

## 2. Our implementation surface (today)

| Ours | Wire (hid-protocol.md) | Their equivalent |
| ---- | ---------------------- | ---------------- |
| Camera mode set (track/idle/privacy) | `0x09,0x01,0x01,0,0,1,0,1,mode` + commit `0x09,0x01,0x01,0x01` | `hidCmdSendSetDeviceMode(UsbHidCtrl*, DeviceMode, ...)` → `CMD_SET_DEVICE_MODE` |
| Audio mode set (nc/live/original) | iface `0x05`, mode 1/2/3 | `CMD_SET_AUDIO_INPUT_GAIN`-adjacent family (audio chain: AGC, input gain, denoise, music mode) — our 3-mode audio maps onto their denoise/music/AGC toggles, not one enum |
| Gesture toggle | iface `0x04`, on=1 | `hidCmdSendSetGestureRecogSta(UsbHidCtrl*, GestureType, u8, ...)` → `CMD_SET_GESTURE_RECOG_STA` (theirs is per-gesture-type) |
| Mode queries | `0x09,iface,...` reads | `hidCmdSendGetDeviceMode/GetGestureRecogSta(+/GestureType)/...` |
| PTZ | V4L2 (`v4l2-ctl`), not HID | **HID motor commands** (§3 MOTOR) + V4L2 UVC (`GetUvcSupportZoomParam`) |

## 3. Full command classification

Legend: ✅ implemented by us · 🔷 same semantic exists officially, our bytes known ·
🟨 official-only, parameter types known, **command bytes unknown** · ⬜ studio-feature, not applicable.

### Camera / modes

| Command (official) | Signature (send / parse-out) | Class |
| ------------------ | ---------------------------- | ----- |
| `CMD_SET/GET_DEVICE_MODE` | `DeviceMode` / `DeviceMode&` | ✅ (our iface 0x01) |
| `CMD_SET/GET_GESTURE_RECOG_STA` | `(GestureType, u8)` / `(GestureType&, u8&)` | ✅ (ours: single toggle) |
| `CMD_SET/GET_PRIVACY_TRIGGER_TIME` | `(i32)` / `u32&` — auto-privacy delay | 🟨 (PIXY-specific options, `initPrivacyTimeOptionsForPixyDevices`) |
| `CMD_SET/GET_REVERSE_STA` | `(ReverseType, u8)` — image flip | 🟨 |
| `CMD_SET/GET_HDR_STA` | `(u8)` | 🟨 |
| `CMD_GET_FUNC_STA` | `u32&` (bitfield of capabilities) | 🟨 — useful capability probe |
| `CMD_SET_FACTORY_RESET` / `CMD_SET_REBOOT` | — | 🟨 |

### Motor / PTZ (TODO #138, #141)

| Command | Signature | Class |
| ------- | --------- | ----- |
| `CMD_SET/GET_MOTOR_POS` | `(MotorType, f32)` / `(MotorType&, f32&, f32&)` | 🟨 — HID alternative to our V4L2 PTZ |
| `CMD_SET_MOTOR_RELATIVE_POS` | `(MotorType, f32)` | 🟨 — our `rel±` over V4L2 |
| `CMD_SET_MOTOR_RUNNING` | `(f32, f32, f32)` — one-key move pan/tilt/zoom ("yawPos" in logs) | 🟨 |
| `CMD_SET/GET_MOTOR_SPEED` | `(MotorType, f32)` / `(MotorType&, f32&, f32&)` | 🟨 — **#138**; speed is a float (likely °/s), per-axis |
| `CMD_SET_MOTOR_PRESET_POS` | `(u8 slot)` — save current to slot | 🟨 — **#141** |
| `CMD_SET/GET_MOTOR_PRESET_POS_MODE` | `(u8 slot, DefaultPosMode)` / `(u8&, DefaultPosMode&, f32&, f32&, f32&)` | 🟨 — **#141** (slot + mode + pan/tilt/zoom readback) |
| `CMD_SET/GET_MOTOR_POWER_ON_DEFAULT_POS_MODE` | `(DefaultPosMode)` / `(DefaultPosMode&, f32×3)` | 🟨 — **#141** (power-on default = last/custom preset) |
| `CMD_SET_MOTOR_DEFAULT_POS` | `(bool)` — save current as default | 🟨 |
| `CMD_GET_MOTOR_POS` (query) | `(MotorType&, f32&, f32&)` | 🟨 — hardware PTZ readback (our 500ms V4L2 readback equivalent) |

### Tracking (TODO #140)

| Command | Signature | Class |
| ------- | --------- | ----- |
| `CMD_SET/GET_TARGET_TRACK` | `(TargetTrackMode, f32, f32, f32)` / `(TargetTrackMode&, f32×3)` | 🟨 — face tracking; UI strings confirm **Face / HalfBody / FullBody** variants ("HalfBodyTracking", "FullBodyTracking", "Half Body Tracking") |
| `CMD_SET/GET_OBJECT_TRACK` | `(ObjectTrackMode, i8, f32×4)` / `(ObjectTrackMode&, i8&, f32×4)` | 🟨 — object tracking with target rect (4 floats) |
| `CMD_SET/GET_OBJECT_TRACK_REPORT` | `(u8)` | 🟨 |

### Power / battery (TODO #139, #144)

| Command | Signature | Class |
| ------- | --------- | ----- |
| `CMD_GET_BATTERY_LEVEL` | parse → `u8&` ("CMD_GET_BATTERY_LEVEL_VAL success.level") | 🟨 — **#139/#144**; HID form present in **Mac binary only** (Windows reaches battery via elink/`EMNetCmdHelper::getBatteryLevel` — see §5) |
| `CMD_GET_CHARGE_STA` | parse → `ChargeSta&` | 🟨 — companion to battery |
| `CMD_SET/GET_AUTO_POWER_ON` | `(u8)` | 🟨 |
| `CMD_SET/GET_AUTO_SHUTDOWN_TIME` | `(u8, i16)` / `(bool&, i16&)` | 🟨 |
| `CMD_SET/GET_POWER_MNG_MODE` | `(PowerMngMode)` | 🟨 |

### Audio (deeper than ours)

`CMD_GET/SET_AUDIO_AGC_STA` (`u8`), `AUDIO_INPUT_GAIN` (`i8`), `AUDIO_OUTPUT_VOLUME` (`u8`),
`CMD_GET/SET_DENOISE_STA` (`u8`), `CMD_GET/SET_MUSIC_MODE` (`MusicMode`), `CMD_GET_MIC_CUR`,
`CMD_SET/GET_MIC_MUTE_STA`, `CMD_SET_MIC_SWITCH` — all 🟨. Our 3-mode audio switch
(nc/live/original) is the UVC-level abstraction; theirs is the DSP-level chain.

### Identity / info

`CMD_GET_SN` (`LibDeviceType_e`, → QString), `CMD_GET_DEVICE_VER`/`CMD_GET_VER`
(→ u16), `CMD_GET_DEVICE_ALIAS` (set takes QString) — 🟨, cheap wins for `device` output.

### Studio features (⬜ not applicable to the wired PIXY daemon)

SD record family (12 cmds), HDMI dir/status, stream push RTMP/RTSP, WiFi
(SSID list/history/conn/AP/STA/dbm), remote pairing, RTC, LUT3D, light RGB
(brightness/color/switch/mode), multi-spectral, focus/meter/WB lock families,
OSD, key-short-func, firmware upgrade (`START/END_UPG_MASTER/DEVICE` — motor MCU
upgrade via `Fic760xUsbUpgradeDll` on Windows).

## 4. The four feature TODOs — design inputs

### #138 PTZ speed
`SetMotorSpeed(MotorType, f32)` per axis + `GetMotorSpeed` returning **two**
floats (value + limit). Motor has 3 axes (`MotorType` ∈ {pan, tilt, zoom} —
`SetMotorRunning(f32,f32,f32)` takes all three). V4L2 has no speed control, so
this must go over HID — the first genuinely new HID surface we would add.
Byte-level need: V2Head command ID + payload layout (f32 LE? scaled int?).

### #139/#144 Battery + charge
Read-only: level `u8` (percent), `ChargeSta` enum (strings show `Charging`,
`charge_status`). HID command exists in Mac build only — Windows uses
elink. De-risk (M7): send a candidate GET framing against the wired PIXY;
if no response, battery is PIXY-Wireless-only and #139 closes as wontfix.
The official UI has a dedicated `EMBatteryLevel` QML component and a
`batteryLight` LED behavior.

### #140 Tracking variants
`TargetTrackMode` enum with ≥3 values (Face/HalfBody/FullBody confirmed in UI
strings + settings keys `HalfBodyTracking`/`FullBodyTracking`). Set takes
`(mode, f32×3)` (sensitivity/box?). Current `CMD_SET_DEVICE_MODE` tracking
byte (0x01) is mode-less — the variants live one level deeper.

### #141 Motor presets
Hardware slots: `SetMotorPresetPos(u8 slot)` saves current position;
`GetMotorPresetPosMode(slot)` reads `(mode, pan, tilt, zoom)` back;
`SetMotorPowerOnDefaultPosMode(DefaultPosMode)` sets which slot (or
last-position) powers on. Our named presets (state.json) can mirror onto
slots 1..N. `DefaultPosMode` has a `DefaultPosModeDto` (u32 ctor) — likely
{last, preset_1..N}.

## 5. Platform string-diff (M13 resolution)

The Windows EXE carries **42** `hidCmdSend*` string hits vs the Mac binary's
**185**; zero Windows-only methods; the missing ones (all `Get*` queries:
motor pos/speed, battery, charge, target/object track, device mode, gesture
state, …) appear on Windows only as `EMNetCmdHelper::` (elink) forms or not at
all. Two consistent explanations, both non-blocking for us:

1. **Log-level stripping**: the Mac build ships debug-log format strings for
   every EMHidCmdHelper method (`EMHidCmdHelper::CMD_...`); the Windows linker
   (or release config) kept a subset. The `elink` 24-vs-1296 hit gap has the
   same shape — the Windows build links only the elink pieces it needs.
2. The shared codebase (same Qt 6.8.3, same Jenkins pipeline) makes true
   feature divergence unlikely; battery-over-HID may be newer than the
   Windows release snapshot.

Verdict: treat the **Mac binary as the protocol ground truth**; the Windows
diff is a build artifact, not a feature matrix. (Documented as resolved-open:
if hardware testing later shows the wired PIXY ignoring a HID battery query,
that — not the string diff — is the real signal.)

## 6. Getting the command bytes (implementation gate)

Everything above is name+type-complete but value-incomplete: the V2Head
command IDs are runtime-initialized globals. Three escalation paths, cheapest
first:

1. **Live sniff (M7-style)**: run the official app is impossible on Linux, but
   a Windows VM + Wireshark USB capture (or `usbmon` under the VM) gives the
   exact report bytes for any command we care about (speed, battery, presets).
2. **Ghidra on the Mac binary**: `__ZN14EMHidCmdHelper21CMD_GET_BATTERY_LEVELE`
   global's initializer reveals the ID constant. The binary is 241 MB but the
   symbols are intact (no stripping) — a targeted disassembly is feasible.
3. **Empirical probing**: the response FSM suggests a request/response ID
   scheme; with the PIXY attached, sweep candidate report IDs and watch for
   well-formed responses (bounded, cautious — the firmware tolerates unknown
   reports; our fuzz tests already exercise garbage reads).

Until one of these lands, feature TODOs #138–#141 stay **designed but not
wire-ready**; everything else in this doc is commit-grade documentation.
