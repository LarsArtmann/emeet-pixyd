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
- **`[HID_RACE_FIX]`**: the official app keeps a persistent HID _read thread_ and
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

| Ours                                 | Wire (hid-protocol.md)                                         | Their equivalent                                                                                                                                                          |
| ------------------------------------ | -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Camera mode set (track/idle/privacy) | `0x09,0x01,0x01,0,0,1,0,1,mode` + commit `0x09,0x01,0x01,0x01` | `hidCmdSendSetDeviceMode(UsbHidCtrl*, DeviceMode, ...)` → `CMD_SET_DEVICE_MODE`                                                                                           |
| Audio mode set (nc/live/original)    | iface `0x05`, mode 1/2/3                                       | `CMD_SET_AUDIO_INPUT_GAIN`-adjacent family (audio chain: AGC, input gain, denoise, music mode) — our 3-mode audio maps onto their denoise/music/AGC toggles, not one enum |
| Gesture toggle                       | iface `0x04`, on=1                                             | `hidCmdSendSetGestureRecogSta(UsbHidCtrl*, GestureType, u8, ...)` → `CMD_SET_GESTURE_RECOG_STA` (theirs is per-gesture-type)                                              |
| Mode queries                         | `0x09,iface,...` reads                                         | `hidCmdSendGetDeviceMode/GetGestureRecogSta(+/GestureType)/...`                                                                                                           |
| PTZ                                  | V4L2 (`v4l2-ctl`), not HID                                     | **HID motor commands** (§3 MOTOR) + V4L2 UVC (`GetUvcSupportZoomParam`)                                                                                                   |

## 3. Full command classification

Legend: ✅ implemented by us · 🔷 same semantic exists officially, head+payload bytes
known (`tools/emhid/cmdtable.json`, 2026-09-18 extraction) · ⬜ studio-feature, not
applicable. Former 🔷 rows ("bytes unknown") were all upgraded after the V2Head
table extraction; response _framing_ and enum _values_ remain M27-verify items (§6).

### Camera / modes

| Command (official)                         | Signature (send / parse-out)                | Class                                                              |
| ------------------------------------------ | ------------------------------------------- | ------------------------------------------------------------------ |
| `CMD_SET/GET_DEVICE_MODE`                  | `DeviceMode` / `DeviceMode&`                | ✅ (our iface 0x01)                                                |
| `CMD_SET/GET_GESTURE_RECOG_STA`            | `(GestureType, u8)` / `(GestureType&, u8&)` | ✅ (ours: single toggle)                                           |
| `CMD_SET/GET_PRIVACY_TRIGGER_TIME`         | `(i32)` / `u32&` — auto-privacy delay       | 🔷 (PIXY-specific options, `initPrivacyTimeOptionsForPixyDevices`) |
| `CMD_SET/GET_REVERSE_STA`                  | `(ReverseType, u8)` — image flip            | 🔷                                                                 |
| `CMD_SET/GET_HDR_STA`                      | `(u8)`                                      | 🔷                                                                 |
| `CMD_GET_FUNC_STA`                         | `u32&` (bitfield of capabilities)           | 🔷 — useful capability probe                                       |
| `CMD_SET_FACTORY_RESET` / `CMD_SET_REBOOT` | —                                           | 🔷                                                                 |

### Motor / PTZ (TODO #138, #141)

| Command                                       | Signature                                                                | Class                                                           |
| --------------------------------------------- | ------------------------------------------------------------------------ | --------------------------------------------------------------- |
| `CMD_SET/GET_MOTOR_POS`                       | `(MotorType, f32)` / `(MotorType&, f32&, f32&)`                          | 🔷 — HID alternative to our V4L2 PTZ                            |
| `CMD_SET_MOTOR_RELATIVE_POS`                  | `(MotorType, f32)`                                                       | 🔷 — our `rel±` over V4L2                                       |
| `CMD_SET_MOTOR_RUNNING`                       | `(f32, f32, f32)` — one-key move pan/tilt/zoom ("yawPos" in logs)        | 🔷                                                              |
| `CMD_SET/GET_MOTOR_SPEED`                     | `(MotorType, f32)` / `(MotorType&, f32&, f32&)`                          | 🔷 — **#138**; speed is a float (likely °/s), per-axis          |
| `CMD_SET_MOTOR_PRESET_POS`                    | `(u8 slot)` — save current to slot                                       | 🔷 — **#141**                                                   |
| `CMD_SET/GET_MOTOR_PRESET_POS_MODE`           | `(u8 slot, DefaultPosMode)` / `(u8&, DefaultPosMode&, f32&, f32&, f32&)` | 🔷 — **#141** (slot + mode + pan/tilt/zoom readback)            |
| `CMD_SET/GET_MOTOR_POWER_ON_DEFAULT_POS_MODE` | `(DefaultPosMode)` / `(DefaultPosMode&, f32×3)`                          | 🔷 — **#141** (power-on default = last/custom preset)           |
| `CMD_SET_MOTOR_DEFAULT_POS`                   | `(bool)` — save current as default                                       | 🔷                                                              |
| `CMD_GET_MOTOR_POS` (query)                   | `(MotorType&, f32&, f32&)`                                               | 🔷 — hardware PTZ readback (our 500ms V4L2 readback equivalent) |

### Tracking (TODO #140)

| Command                           | Signature                                                         | Class                                                                                                                                         |
| --------------------------------- | ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `CMD_SET/GET_TARGET_TRACK`        | `(TargetTrackMode, f32, f32, f32)` / `(TargetTrackMode&, f32×3)`  | 🔷 — face tracking; UI strings confirm **Face / HalfBody / FullBody** variants ("HalfBodyTracking", "FullBodyTracking", "Half Body Tracking") |
| `CMD_SET/GET_OBJECT_TRACK`        | `(ObjectTrackMode, i8, f32×4)` / `(ObjectTrackMode&, i8&, f32×4)` | 🔷 — object tracking with target rect (4 floats)                                                                                              |
| `CMD_SET/GET_OBJECT_TRACK_REPORT` | `(u8)`                                                            | 🔷                                                                                                                                            |

### Power / battery (TODO #139, #144)

| Command                          | Signature                                                 | Class                                                                                                                                      |
| -------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `CMD_GET_BATTERY_LEVEL`          | parse → `u8&` ("CMD_GET_BATTERY_LEVEL_VAL success.level") | 🔷 — **#139/#144**; HID form present in **Mac binary only** (Windows reaches battery via elink/`EMNetCmdHelper::getBatteryLevel` — see §5) |
| `CMD_GET_CHARGE_STA`             | parse → `ChargeSta&`                                      | 🔷 — companion to battery                                                                                                                  |
| `CMD_SET/GET_AUTO_POWER_ON`      | `(u8)`                                                    | 🔷                                                                                                                                         |
| `CMD_SET/GET_AUTO_SHUTDOWN_TIME` | `(u8, i16)` / `(bool&, i16&)`                             | 🔷                                                                                                                                         |
| `CMD_SET/GET_POWER_MNG_MODE`     | `(PowerMngMode)`                                          | 🔷                                                                                                                                         |

### Audio (deeper than ours)

`CMD_GET/SET_AUDIO_AGC_STA` (`u8`), `AUDIO_INPUT_GAIN` (`i8`), `AUDIO_OUTPUT_VOLUME` (`u8`),
`CMD_GET/SET_DENOISE_STA` (`u8`), `CMD_GET/SET_MUSIC_MODE` (`MusicMode`), `CMD_GET_MIC_CUR`,
`CMD_SET/GET_MIC_MUTE_STA`, `CMD_SET_MIC_SWITCH` — all 🔷. Our 3-mode audio switch
(nc/live/original) is the UVC-level abstraction; theirs is the DSP-level chain.

### Identity / info

`CMD_GET_SN` (`LibDeviceType_e`, → QString), `CMD_GET_DEVICE_VER`/`CMD_GET_VER`
(→ u16), `CMD_GET_DEVICE_ALIAS` (set takes QString) — 🔷, cheap wins for `device` output.

### Studio features (⬜ not applicable to the wired PIXY daemon)

SD record family (12 cmds), HDMI dir/status, stream push RTMP/RTSP, WiFi
(SSID list/history/conn/AP/STA/dbm), remote pairing, RTC, LUT3D, light RGB
(brightness/color/switch/mode), multi-spectral, focus/meter/WB lock families,
OSD, key-short-func, firmware upgrade (`START/END_UPG_MASTER/DEVICE` — motor MCU
upgrade via `Fic760xUsbUpgradeDll` on Windows).

## 3.5 The extracted V2Head command table (2026-09-18 breakthrough)

**Source:** static-initializer disassembly of the macOS binary — every
`EMHidCmdHelper::CMD_*_E` global is constructed as `EMHidCmdV2Head(u8,u8,u8,u8)`;
joining the four immediate `mov w1..w4` operands with the GOT slot → symbol binds
yielded **162 command heads**. Tool: `tools/emhid/extract_cmdtable.py`; data:
`tools/emhid/cmdtable.json` (single source of truth for the full list — the
grouped tables below are the human-readable view).

**Head format `[b0, b1, b2, b3]`:**

- `b0` — report head: `0x09` (V2 era) or `0x07` (legacy `HIDOldCmdHelper*`/V1 commands).
- `b1` — logical sub-device / interface. Ten devices (0x00–0x0a, no 0x07).
- `b2` — category (`mergeType` component). `mergeType(dev, func) = (dev<<5) | func`
  explains the large legacy values (e.g. 0x63 = (3<<5)|3).
- `b3` — command ID within (device, category). SET/GET are distinct IDs
  (e.g. `SET_MOTOR_SPEED`=3 vs `GET_MOTOR_SPEED`=19), not a flag bit.

**Motor-MCU routing:** motor `SET` commands overwrite the iface byte on the wire
with `0x63` (motor-MCU sub-device, `mergeType(3,3)`); the table records the
logical `0x03`. Probes should try **both** `b1=0x03` and `b1=0x63` (M3 probe does).

**Payload layouts (from controller call sites + log format strings):**

| Command                         | Payload after head                  |
| ------------------------------- | ----------------------------------- |
| `CMD_SET_DEVICE_MODE`           | `[mode:u8]`                         |
| `CMD_SET_MOTOR_SPEED`           | `[motorType:u8][speed:f32]` (len 5) |
| `CMD_SET_MOTOR_PRESET_POS`      | `[slot:u8]`                         |
| `CMD_SET_MOTOR_PRESET_POS_MODE` | `[slot:u8][mode:u8]`                |
| `CMD_SET_TARGET_TRACK`          | `[mode:u8][f32×3]` (len 13)         |
| GETs                            | bare 4-byte heads, no payload       |

**Known query heads** (battery probe / read paths): battery `09 00 00 02`,
charge `09 00 00 06`, motor speed `09 03 01 13`, motor pos `09 03 01 02`,
target track `09 04 01 02`, device mode `09 02 01 00`, func status `09 01 00 0d`,
SN `09 01 00 04`, ver `09 01 00 05`, device ver `09 01 00 0f`.

Extraction collision note: two symbols share head `07 06 70 00`
(`CMD_GET_SD_RECORD_SUPPORT_VIDEO_PARAM` and a V1 `SET_FOCUS_LOCK_STA`) — the
legacy `0x07` region reuses (device, category, id) triples; irrelevant for the
V2 `0x09` surface we implement against.

### Power / battery (dev 0x00) — 8 commands

| Head bytes    | Command                      |
| ------------- | ---------------------------- |
| `09 00 00 01` | `CMD_SET_REBOOT`             |
| `09 00 00 02` | `CMD_GET_BATTERY_LEVEL`      |
| `09 00 00 03` | `CMD_SET_AUTO_SHUTDOWN_TIME` |
| `09 00 00 05` | `CMD_GET_AUTO_SHUTDOWN_TIME` |
| `09 00 00 06` | `CMD_GET_CHARGE_STA`         |
| `09 00 00 07` | `CMD_SET_AUTO_POWER_ON`      |
| `09 00 00 08` | `CMD_GET_AUTO_POWER_ON`      |
| `09 00 00 09` | `CMD_SET_POWER_MANAGE_MODE`  |

### Device identity & mode (dev 0x01) — 10 commands

| Head bytes    | Command                     |
| ------------- | --------------------------- |
| `09 01 00 03` | `CMD_GET_POWER_MANAGE_MODE` |
| `09 01 00 04` | `CMD_GET_SN`                |
| `09 01 00 05` | `CMD_GET_VER`               |
| `09 01 00 08` | `CMD_SET_FACTORY_RESET`     |
| `09 01 00 0d` | `CMD_GET_FUNC_STA`          |
| `09 01 00 0e` | `CMD_SET_DEVICE_VER`        |
| `09 01 00 0f` | `CMD_GET_DEVICE_VER`        |
| `09 01 00 10` | `CMD_SET_DEVICE_ALIAS`      |
| `09 01 01 00` | `CMD_GET_DEVICE_ALIAS`      |
| `09 01 01 01` | `CMD_SET_DEVICE_MODE`       |

### Privacy, light, key-func (dev 0x02) — 16 commands

| Head bytes    | Command                          |
| ------------- | -------------------------------- |
| `07 02 09 00` | `CMD_SET_VERTICAL_STA_OLD`       |
| `09 02 01 00` | `CMD_GET_DEVICE_MODE`            |
| `09 02 01 01` | `CMD_SET_PRIVACY_TRIGGER_TIME`   |
| `09 02 02 00` | `CMD_GET_PRIVACY_TRIGGER_TIME`   |
| `09 02 02 01` | `CMD_SET_LIGHT_RGB_SWITCH`       |
| `09 02 02 02` | `CMD_GET_LIGHT_RGB_SWITCH`       |
| `09 02 02 03` | `CMD_SET_LIGHT_RGB_COLOR`        |
| `09 02 02 04` | `CMD_GET_LIGHT_RGB_COLOR`        |
| `09 02 02 05` | `CMD_SET_LIGHT_RGB_BRIGHTNESS`   |
| `09 02 02 06` | `CMD_GET_LIGHT_RGB_BRIGHTNESS`   |
| `09 02 02 07` | `CMD_SET_LIGHT_MODE`             |
| `09 02 02 08` | `CMD_GET_LIGHT_MODE`             |
| `09 02 02 0b` | `CMD_GET_LIGHT_ADJUST_ALLOW_STA` |
| `09 02 02 0c` | `CMD_SET_LIGHT_LIMIT_BRIGHTNESS` |
| `09 02 04 00` | `CMD_GET_LIGHT_LIMIT_BRIGHTNESS` |
| `09 02 04 01` | `CMD_SET_KEY_SHORT_FUNC`         |

### Motor / PTZ (dev 0x03; SETs go on the wire with iface byte 0x63) — 14 commands

| Head bytes    | Command                                   |
| ------------- | ----------------------------------------- |
| `09 03 01 00` | `CMD_GET_KEY_SHORT_FUNC`                  |
| `09 03 01 01` | `CMD_SET_MOTOR_POS`                       |
| `09 03 01 02` | `CMD_GET_MOTOR_POS`                       |
| `09 03 01 03` | `CMD_SET_MOTOR_SPEED`                     |
| `09 03 01 13` | `CMD_GET_MOTOR_SPEED`                     |
| `09 03 01 14` | `CMD_SET_MOTOR_POWER_ON_DEFAULT_POS_MODE` |
| `09 03 01 15` | `CMD_GET_MOTOR_POWER_ON_DEFAULT_POS_MODE` |
| `09 03 01 16` | `CMD_SET_MOTOR_PRESET_POS_MODE`           |
| `09 03 01 17` | `CMD_GET_MOTOR_PRESET_POS_MODE`           |
| `09 03 01 18` | `CMD_SET_MOTOR_DEFAULT_POS`               |
| `09 03 01 19` | `CMD_SET_MOTOR_PRESET_POS`                |
| `09 03 01 20` | `CMD_SET_MOTOR_RELATIVE_POS`              |
| `09 03 04 03` | `CMD_SET_MOTOR_RUNNING`                   |
| `09 03 04 04` | `CMD_SET_REMOTE_PAIRING_STA`              |

### Optics, tracking, gesture (dev 0x04) — 48 commands

| Head bytes    | Command                           |
| ------------- | --------------------------------- |
| `09 04 00 01` | `CMD_GET_REMOTE_PAIRING_STA`      |
| `09 04 00 02` | `CMD_SET_FOCUS_MODE`              |
| `09 04 00 03` | `CMD_GET_FOCUS_MODE`              |
| `09 04 00 04` | `CMD_SET_METER_MODE`              |
| `09 04 00 05` | `CMD_GET_METER_MODE`              |
| `09 04 00 06` | `CMD_SET_OSD_STA`                 |
| `09 04 00 07` | `CMD_GET_OSD_STA`                 |
| `09 04 00 08` | `CMD_GET_REVERSE_STA`             |
| `09 04 00 09` | `CMD_SET_REVERSE_STA`             |
| `09 04 00 0a` | `CMD_SET_WB_LOCK_STA`             |
| `09 04 00 0b` | `CMD_GET_WB_LOCK_STA`             |
| `09 04 00 0c` | `CMD_SET_EV_LOCK_STA`             |
| `09 04 00 0d` | `CMD_GET_EV_LOCK_STA`             |
| `09 04 00 0e` | `CMD_SET_FOCUS_LOCK_STA`          |
| `09 04 00 0f` | `CMD_GET_FOCUS_LOCK_STA`          |
| `09 04 00 10` | `CMD_GET_RIO_INFO`                |
| `09 04 00 11` | `CMD_GET_IMAGE_SCOPE`             |
| `09 04 00 12` | `CMD_GET_UVCARGS_STATUS`          |
| `09 04 00 13` | `CMD_SET_HDR_STA`                 |
| `09 04 00 14` | `CMD_GET_HDR_STA`                 |
| `09 04 00 15` | `CMD_SET_MANNUAL_EV_TIME`         |
| `09 04 00 16` | `CMD_GET_MANNUAL_EV_TIME`         |
| `09 04 00 17` | `CMD_SET_MANNUAL_WB_INFO`         |
| `09 04 00 18` | `CMD_GET_MANNUAL_WB_INFO`         |
| `09 04 00 19` | `CMD_GET_UVC_SUPPORT_VIDEO_PARAM` |
| `09 04 00 1a` | `CMD_SET_MULTI_SPECTRAL`          |
| `09 04 00 1b` | `CMD_GET_MULTI_SPECTRAL`          |
| `09 04 00 1c` | `CMD_SET_MULTI_SPECTRAL_FUNC`     |
| `09 04 00 1d` | `CMD_GET_MULTI_SPECTRAL_FUNC`     |
| `09 04 00 1e` | `CMD_SET_UVC_VIDEO_DIR`           |
| `09 04 00 21` | `CMD_GET_UVC_VIDEO_DIR`           |
| `09 04 00 22` | `CMD_SET_POWER_LINE_MODE`         |
| `09 04 00 27` | `CMD_GET_POWER_LINE_MODE`         |
| `09 04 00 28` | `CMD_SET_LUT3D_MODE`              |
| `09 04 00 29` | `CMD_GET_LUT3D_MODE`              |
| `09 04 00 2a` | `CMD_GET_UVC_SUPPORT_ZOOM_PARAM`  |
| `09 04 00 2b` | `CMD_SET_HDMI_VIDEO_DIR`          |
| `09 04 00 2c` | `CMD_GET_HDMI_VIDEO_DIR`          |
| `09 04 00 31` | `CMD_GET_HDMI_CONNECT_STATUS`     |
| `09 04 00 32` | `CMD_SET_WB_FINE_TUNE`            |
| `09 04 01 00` | `CMD_GET_WB_FINE_TUNE`            |
| `09 04 01 01` | `CMD_SET_TARGET_TRACK`            |
| `09 04 01 02` | `CMD_GET_TARGET_TRACK`            |
| `09 04 01 03` | `CMD_SET_OBJECT_TRACK`            |
| `09 04 01 04` | `CMD_GET_OBJECT_TRACK`            |
| `09 04 01 05` | `CMD_SET_OBJECT_TRACK_REPORT`     |
| `09 04 02 00` | `CMD_GET_OBJECT_TRACK_REPORT`     |
| `09 04 02 01` | `CMD_SET_GESTURE_RECOG_STA`       |

### Audio DSP (dev 0x05) — 13 commands

| Head bytes    | Command                     |
| ------------- | --------------------------- |
| `09 05 00 00` | `CMD_GET_GESTURE_RECOG_STA` |
| `09 05 00 01` | `CMD_SET_DENOISE_STA`       |
| `09 05 00 02` | `CMD_GET_DENOISE_STA`       |
| `09 05 00 03` | `CMD_SET_MIC_SWITCH`        |
| `09 05 00 04` | `CMD_SET_MUSIC_MODE`        |
| `09 05 00 05` | `CMD_GET_MUSIC_MODE`        |
| `09 05 00 06` | `CMD_GET_MIC_CUR`           |
| `09 05 00 07` | `CMD_SET_MIC_MUTE_STA`      |
| `09 05 00 08` | `CMD_GET_MIC_MUTE_STA`      |
| `09 05 00 09` | `CMD_SET_AUDIO_AGC_STA`     |
| `09 05 00 0a` | `CMD_GET_AUDIO_AGC_STA`     |
| `09 05 00 0b` | `CMD_SET_AUDIO_INPUT_GAIN`  |
| `09 05 01 00` | `CMD_GET_AUDIO_INPUT_GAIN`  |

### Volume, WiFi, stream push (dev 0x06) + legacy V1 (head 0x07) — 35 commands

| Head bytes    | Command                                 |
| ------------- | --------------------------------------- |
| `07 06 33 00` | `CMD_SET_FOCUS_LOCK_STA_OLD`            |
| `07 06 34 00` | `CMD_GET_HORIZONTAL_STA_OLD`            |
| `07 06 35 00` | `CMD_SET_HORIZONTAL_STA_OLD`            |
| `07 06 36 00` | `CMD_GET_VERTICAL_STA_OLD`              |
| `07 06 6e 00` | `CMD_SET_EV_LOCK_STA_OLD`               |
| `07 06 6f 00` | `CMD_GET_WB_LOCK_STA_OLD`               |
| `07 06 70 00` | `CMD_GET_SD_RECORD_SUPPORT_VIDEO_PARAM` |
| `07 06 70 00` | `CMD_SET_FOCUS_LOCK_STA`                |
| `07 06 71 00` | `CMD_GET_EV_LOCK_STA_OLD`               |
| `07 06 72 00` | `CMD_SET_WB_LOCK_STA_OLD`               |
| `07 06 73 00` | `CMD_GET_FOCUS_LOCK_STA_OLD`            |
| `07 06 74 00` | `CMD_GET_VER_OLD`                       |
| `09 06 00 00` | `CMD_SET_AUDIO_OUTPUT_VOLUME`           |
| `09 06 00 01` | `CMD_GET_HAS_PASSWD`                    |
| `09 06 00 02` | `CMD_SET_CONFIG_PASSWD`                 |
| `09 06 01 00` | `CMD_SET_VERIFY_PASSWD`                 |
| `09 06 01 01` | `CMD_GET_WIFI_LIST`                     |
| `09 06 01 02` | `CMD_SET_CONN_WIFI`                     |
| `09 06 01 03` | `CMD_GET_CUR_WIFI_DBM`                  |
| `09 06 01 04` | `CMD_SET_DEL_CUR_WIFI`                  |
| `09 06 01 05` | `CMD_SET_DISCONN_CUR_WIFI`              |
| `09 06 01 06` | `CMD_GET_AP_PARAM`                      |
| `09 06 01 07` | `CMD_SET_AP_PARAM`                      |
| `09 06 01 08` | `CMD_GET_AP_STA`                        |
| `09 06 01 09` | `CMD_SET_AP_STA`                        |
| `09 06 01 0a` | `CMD_GET_SSID_HISTORY`                  |
| `09 06 01 0b` | `CMD_GET_STA_INFO`                      |
| `09 06 01 0c` | `CMD_GET_AP_INFO`                       |
| `09 06 01 0d` | `CMD_SET_COUNTRY_CODE`                  |
| `09 06 02 00` | `CMD_GET_COUNTRY_CODE`                  |
| `09 06 02 01` | `CMD_SET_START_RTMP_PUSH`               |
| `09 06 02 02` | `CMD_SET_START_RTSP_PUSH`               |
| `09 06 02 03` | `CMD_SET_STREAM_PUSH_OFF`               |
| `09 06 02 04` | `CMD_GET_STREAM_PUSH_STA`               |
| `09 06 02 05` | `CMD_GET_STREAM_PUSH_URL`               |

### Stream info / RTC (dev 0x08) — 2 commands

| Head bytes    | Command                    |
| ------------- | -------------------------- |
| `09 08 01 06` | `CMD_GET_STREAM_PUSH_INFO` |
| `09 08 01 07` | `CMD_SET_RTC_TIME`         |

### RTC / firmware upgrade (dev 0x09) — 5 commands

| Head bytes    | Command                    |
| ------------- | -------------------------- |
| `09 09 00 00` | `CMD_GET_RTC_TIME`         |
| `09 09 00 01` | `CMD_SET_START_UPG_DEVICE` |
| `09 09 00 04` | `CMD_SET_START_UPG_MASTER` |
| `09 09 00 05` | `CMD_SET_END_UPG_MASTER`   |
| `09 09 00 06` | `CMD_SET_END_UPG_DEVICE`   |

### SD card (dev 0x0a) — 11 commands

| Head bytes    | Command                          |
| ------------- | -------------------------------- |
| `09 0a 00 00` | `CMD_GET_DEVICE_UPG_STA`         |
| `09 0a 00 01` | `CMD_GET_SD_INFO`                |
| `09 0a 00 02` | `CMD_SET_SD_FORMAT`              |
| `09 0a 00 03` | `CMD_SET_SD_RECORD_INFO`         |
| `09 0a 00 04` | `CMD_GET_SD_RECORD_INFO`         |
| `09 0a 00 05` | `CMD_SET_SD_RECORD_STA`          |
| `09 0a 00 06` | `CMD_GET_SD_RECORD_STA`          |
| `09 0a 00 07` | `CMD_GET_SD_RECORD_ERR_INFO`     |
| `09 0a 00 08` | `CMD_GET_SD_RECORD_RUN_DURATION` |
| `09 0a 00 09` | `CMD_SET_SD_RECORD_MAX_DURATION` |
| `09 0a 00 0a` | `CMD_GET_SD_RECORD_MAX_DURATION` |

## 4. The four feature TODOs — design inputs

### #138 PTZ speed

`SetMotorSpeed(MotorType, f32)` per axis + `GetMotorSpeed` returning **two**
floats (value + limit). Motor has 3 axes (`MotorType` ∈ {pan, tilt, zoom} —
`SetMotorRunning(f32,f32,f32)` takes all three). V4L2 has no speed control, so
this must go over HID — the first genuinely new HID surface we would add.
Byte-level: SOLVED — `SET_MOTOR_SPEED` head `09 03 01 03`, payload
`[motorType:u8][speed:f32]`; `GET_MOTOR_SPEED` head `09 03 01 13`. `MotorType`
values (pan/tilt/zoom → 0/1/2?) remain M27-verify.

### #139/#144 Battery + charge

Read-only: level `u8` (percent), `ChargeSta` enum (strings show `Charging`,
`charge_status`). HID command exists in Mac build only — Windows uses
elink. De-risk (M7): send a candidate GET framing against the wired PIXY;
if no response, battery is PIXY-Wireless-only and #139 closes as wontfix.
Exact heads known: `GET_BATTERY_LEVEL` `09 00 00 02`, `GET_CHARGE_STA` `09 00 00 06`
(dev 0x00 — no motor-MCU routing).
The official UI has a dedicated `EMBatteryLevel` QML component and a
`batteryLight` LED behavior.

### #140 Tracking variants

`TargetTrackMode` enum with ≥3 values (Face/HalfBody/FullBody confirmed in UI
strings + settings keys `HalfBodyTracking`/`FullBodyTracking`). Set takes
`(mode, f32×3)` (sensitivity/box?). Current `CMD_SET_DEVICE_MODE` tracking
byte (0x01) is mode-less — the variants live one level deeper. Bytes: `SET_TARGET_TRACK`
`09 04 01 01` payload `[mode:u8][f32×3]`, `GET_TARGET_TRACK` `09 04 01 02`.
`TargetTrackMode` values (Face/HalfBody/FullBody → 0/1/2 in UI order?) M27-verify.

### #141 Motor presets

Hardware slots: `SetMotorPresetPos(u8 slot)` saves current position;
`GetMotorPresetPosMode(slot)` reads `(mode, pan, tilt, zoom)` back;
`SetMotorPowerOnDefaultPosMode(DefaultPosMode)` sets which slot (or
last-position) powers on. Our named presets (state.json) can mirror onto
slots 1..N. `DefaultPosMode` has a `DefaultPosModeDto` (u32 ctor) — likely
{last, preset_1..N}. Bytes: `SET_MOTOR_PRESET_POS` `09 03 01 19` `[slot:u8]`,
`SET/GET_MOTOR_PRESET_POS_MODE` `09 03 01 16`/`17` (`[slot][mode]`),
`SET/GET_MOTOR_POWER_ON_DEFAULT_POS_MODE` `09 03 01 14`/`15`.

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

## 6. Getting the command bytes (implementation gate — BROKEN OPEN)

**2026-09-18: path 2 succeeded.** The full 162-command V2Head table was extracted
from the macOS binary by disassembling the `EMHidCmdV2Head(u8,u8,u8,u8)` static
initializers and joining them with GOT→symbol binds (`tools/emhid/`). Every head
byte in this document comes from that table (§3.5). Escalation history preserved:

1. ~~Live sniff (Windows VM + usbmon)~~ — not needed for heads; remains the
   cross-validation option for the `0x03`-vs-`0x63` motor iface byte.
2. ~~Ghidra/objdump on the Mac binary~~ — **DONE** (llvm-objdump, 162 heads +
   payload layouts from call sites/log strings).
3. ~~Empirical probing~~ — superseded; the M3 probe now sweeps only the _exact_
   known heads × {`0x03`, `0x63`} iface variants instead of generic candidates.

**What is still open (blocked 2026-09-18, needs the binary re-downloaded — the
`/tmp` raw materials died with reboot and no download URL is committed):**

- Enum **values** for `MotorType`, `TargetTrackMode`, `DefaultPosMode`,
  `ChargeSta` (UI order strongly suggests 0/1/2 for the first two; M27-verify).
- **Response framing** (`EMHidCmdV2RecvFsm::onDataRecv`, `hidCmdParseGet*`
  bodies: head echo? sequence byte? length prefix?) — read paths implement
  best-effort parse + raw-hex fallback until pinned.
- Windows x86_64 cross-verification of the table (plan M25).
