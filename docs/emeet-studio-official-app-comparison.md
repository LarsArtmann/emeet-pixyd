# EMEET STUDIO (Official App) vs emeet-pixyd — Deep Research Comparison

> **Research date:** 2026-09-17 · **Analyst:** Crush (for Lars)
> **Sources:** Official EMEET STUDIO 2.0.3 installers, fully unpacked and analyzed:
>
> - Windows: `EMEET_STUDIO_V2.0.3_hotfix_intl_Win.exe` (Inno Setup 6.6.1 — format reverse-engineered, all 2,211 payload files extracted)
> - macOS: `EMEET_STUDIO_V2.0.3_intl_mac_20260903_110248.pkg` (xar → PKG → app bundle, strings-analyzed)
> - Our side: `FEATURES.md` (code-verified inventory)
>   **Artifacts:** `/tmp/emeet/` (extracted Windows payload in `win/`, Mac bundle in `mac/`, strings dumps, parsers)

---

## 1. TL;DR

**The official app and emeet-pixyd are complementary, not competing.** EMEET STUDIO is a
~350 MB cross-platform _virtual-camera studio suite_ (an OBS fork with beauty filters,
virtual camera + virtual microphone outputs) that also happens to be the device-control
app. emeet-pixyd is a 15 MB headless daemon that automates the _device control_ subset —
and does several things the official app **never does** (call detection, auto mode
transitions, Waybar, socket CLI, remote web UI).

On the narrow overlap — camera/audio/PTZ control of the PIXY over HID — we are at rough
feature parity for the core modes, with both sides having unique extras. Their own binary
contains a `[HID_RACE_FIX]` tag validating the exact concurrency concern our `hidMu`
mutex design addresses.

---

## 2. What the Official App Actually Is

### 2.1 One Qt codebase, two build branches

Both binaries are the same project — `NPI2024_Audio_Video_App` — built per platform:

|                                            | Windows                                                                                                                                                                               | macOS                                                                               |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| Build path (from embedded PDB/log strings) | `C:\Users\cheny\jenkins\workspace\_EMEET_STUDIO_2.0-Windows_master\NPI2024_Audio_Video_App\src\...`                                                                                   | `op-EMEET_STUDIO_2.0_macOS_master/NPI2024_Audio_Video_App`                          |
| Main binary                                | `EMEET STUDIO.exe` — 122 MB PE32+                                                                                                                                                     | `EMEET STUDIO` — 241 MB Mach-O                                                      |
| UI framework                               | Qt 6.8.3 / QML (Quick Controls, FluentWinUI3 style)                                                                                                                                   | Qt 6.8.3 / QML                                                                      |
| Embedded OBS                               | `obs.dll`, `libobs-d3d11/opengl/winrt.dll`, `obs-frontend-api.dll`, `obs-scripting.dll` (Lua)                                                                                         | libobs 31.1.2 + plugins                                                             |
| OBS plugins shipped                        | EMVideoInput, image-source, obs-transitions, **obs-volcengine-beauty** (+ full model resource tree)                                                                                   | same + obs-perspective                                                              |
| Beauty/gesture AI                          | `effect.dll` 63.6 MB (ByteDance Volcengine Effect SDK, `bef_effect_ai_*`)                                                                                                             | obs-volcengine-beauty.plugin                                                        |
| Media stack                                | FFmpeg 7.x DLLs (avcodec-61…), libx264-164, **librist** (RIST streaming), datachannel (WebRTC)                                                                                        | FFmpeg, librist, libsrt, mbedtls                                                    |
| Device I/O                                 | `hidapi.dll`/`libhidapi-0.dll`, `libusb-1.0.dll`                                                                                                                                      | libhidapi, libusb                                                                   |
| Extras                                     | `Fic760xUsbUpgradeDll.dll` (WiFi-chip firmware flasher), `emeet_virtual_camera_x64.dll`                                                                                               | DriverKit CMIO camera extension, CoreAudio HAL virtual audio driver                 |
| Virtual camera                             | **User-mode DirectShow filter** (`emeet_virtual_camera_x86/x64.dll`, CLSID `{4A197D07-74F1-4730-AA47-3267604D0857}`, registered via `regsvr32`)                                       | SystemExtension `com.emeet.studio.next.mac-camera-extension` (CMIO)                 |
| Virtual microphone                         | **Kernel driver** `EmeetAudioDriver.sys` (WDM wave/mixer, signed `.cat`, installed via bundled `devcon.exe`, hardware id `ROOT\EmeetVirtualAudio`, RefCount-protected shared install) | CoreAudio HAL plugin (`VirtualAudioPlugin2.driver` → `/Library/Audio/Plug-Ins/HAL`) |
| Installer                                  | Inno Setup 6.6.1, x64-only, 9 languages, 2,212 file entries, 347.8 MB payload                                                                                                         | pkg with 3 sub-packages (app + virtual audio + uninstaller)                         |

The Windows app's OBS beauty plugin ships ByteDance model assets (face meshes,
reshape/whiten "ComposeMakeup" trees, shaders, `ttfacemodel` algo models) — this is a
real commercial beauty pipeline, not a toy.

### 2.2 Device support matrix (from `fw.emeet.ai` model strings)

- **Both platforms:** EmeetPixy, EmeetPixy2K, EmeetPixyDual, EmeetPiko, EmeetPikoDual (+ Piko+ on Mac), PIXY-Wireless
- **Windows-only:** EmeetNova4K, EmeetC960Ultra, EmeetC63E4KDual, EmeetC60E4K, EmeetS600Light, E3164/E3165/E7002… — the Windows app is also the control suite for EMEET's _budget UVC webcam_ lineup (no motor, no HID protocol of PIXY's class).

### 2.3 Their device-control surface (the part that overlaps with us)

From ~96 distinct `CMD_GET_/CMD_SET_` HID names in the Windows binary (a strict subset of
the Mac binary's ~150; every Windows command exists on Mac — one codebase):

- Tracking: TargetTrack + ObjectTrack (Face/HalfBody/FullBody; object box x,y,w,h floats)
- Mode: device mode, privacy trigger time
- Audio: mode (nc/live/original — **same three modes we expose**), AGC, input gain, mute, denoise, music mode
- PTZ: motor position (absolute/relative), **speed**, preset positions, power-on default position
- Image: HDR, EV/WB/focus locks, WB fine-tune, meter mode, power-line mode, LUT3D/color grading, OSD, reverse/flip, multi-spectral
- LED: RGB color/brightness/mode/tally
- Recording: SD card info/format/start/stop/segments/errors
- Video I/O: HDMI in/out, UVC direction
- Wireless (PIXY-Wireless): WiFi AP/STA/SSID/password, `elink` JSON network protocol (~90 command families)
- Streaming from camera: RTMP/RTSP/RIST push
- System: firmware upgrade (MCU + WiFi FIC760x), factory reset, auto power-on/auto-shutdown, RTC/NTP, battery/charge, SN/alias/country-code/password auth, key remapping, auto-rotate

Firmware endpoints: `https://fw.emeet.ai/api/v3/firmware/models/{model}` (currently 404 for
PIXY models) and `https://emeet.ai/software/eMeetLink/software_upgrade_config.json`
(app self-update config, S3 mirror). The `device_upgrade_*.json` URLs found in the Mac
binary (pixy, pixy_2k, pixy_dual, piko, piko_dual, pikoplus) all 404 today.

**Protocol-level validation of our implementation:** their binaries wrap hidapi with
`UsbHidCtrl` + `EMHidCmdRecvFsm` and an `EMHidCmdV2Head` header (plus `_OLD` legacy
variants — i.e., a v2 protocol header evolution, matching our config+commit report
framing). Their log tag **`[HID_RACE_FIX] HID read thread paused, safe to start
hid_write`** shows they hit (and patched) the same hidapi concurrent read/write race our
`hidMu` serialization prevents by design.

---

## 3. Feature Comparison

### 3.1 Core device control — parity table

| Capability                      | Official app                        | emeet-pixyd                                           | Verdict                                                                                       |
| ------------------------------- | ----------------------------------- | ----------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| Tracking / Idle / Privacy modes | ✅ full UI                          | ✅ (`cmdTrack/Idle/Privacy`)                          | **Parity**                                                                                    |
| Audio modes NC/Live/Original    | ✅                                  | ✅ (`AudioMode`, CLI shorthand `org`)                 | **Parity**                                                                                    |
| Gesture toggle                  | ✅ per-gesture-type config          | ✅ single toggle                                      | Official ahead (granularity)                                                                  |
| PTZ absolute/relative           | ✅ + **speed control**              | ✅ absolute+relative (`rel±n`)                        | Official ahead (speed); our relative UX is first-class                                        |
| PTZ presets                     | ✅ motor presets + power-on default | ✅ named presets (16), web chips + CLI, persisted     | **We win on UX** (naming/management); they store presets _on the motor_ (survives host swaps) |
| PTZ limits                      | hardware                            | hardware-verified `Range.Clamp` (±150°/±90°/100–150×) | Parity                                                                                        |
| PTZ readback                    | ✅                                  | ✅ delayed readback correcting cache                  | Parity                                                                                        |
| Center camera                   | ✅                                  | ✅                                                    | Parity                                                                                        |
| HID protocol                    | EMHidCmdV2 config+commit            | config+commit (200 ms), hidraw                        | **Same wire behavior**                                                                        |
| Hotplug detection               | (app UI event)                      | ✅ netlink uevents → re-probe, circuit breaker        | We win (automatic, headless)                                                                  |
| State persistence               | app config                          | `state.json` w/ schema versioning                     | Parity                                                                                        |

### 3.2 Official-only (out of scope for a daemon — mostly)

- **OBS virtual-camera studio**: scenes/sources/transitions, image source, Lua scripting
- **Beauty filters**: ByteDance Volcengine face reshape/whiten/makeup/sharpen pipelines
- **Virtual camera output** (DirectShow/CMIO) and **virtual microphone** (kernel/CoreAudio driver)
- **Live-streaming integration**: RTMP/RIST/WebRTC push, **Kuaishou OAuth** (`emeettest.emeet.ai/authuser/api/kuaishou/*`)
- **SD-card recording management**, **LED/tally control**, **battery/charge status**
- **Image tuning**: HDR, EV/WB/focus locks, LUT3D, meter/power-line modes
- **Wireless PIXY support** (`elink` network protocol, WiFi provisioning, FIC760x WiFi firmware flash)
- **Firmware upgrade UI** (MCU + WiFi IC), **accounts/login**, **multi-device support** (incl. budget UVC cams)
- Auto power-on/shutdown, RTC/NTP, key remapping, privacy-trigger time

### 3.3 emeet-pixyd-only (the official app has _none_ of this)

| Capability                                                                          | Why the official app can't                          | Ours                                               |
| ----------------------------------------------------------------------------------- | --------------------------------------------------- | -------------------------------------------------- |
| **Call detection** (`/proc/*/fd` scan + debounce)                                   | It's a foreground studio app; no automation concept | ✅ core design                                     |
| **Auto mode transitions** (call start → tracking+NC+PipeWire source; end → privacy) | ditto                                               | ✅ 4 modes (`full/tracking-only/privacy-only/off`) |
| **PipeWire default-source switching** (`wpctl`)                                     | Windows/macOS app; no Linux audio story at all      | ✅                                                 |
| **Headless daemon** (systemd, sd_notify watchdog, socket CLI)                       | GUI app                                             | ✅                                                 |
| **Web UI** (DataStar, live SSE state, reactive PTZ radar)                           | Qt desktop UI only                                  | ✅ remote-controllable                             |
| **Waybar integration**                                                              | n/a                                                 | ✅                                                 |
| **Snapshot capture** to JPEG download                                               | in-app only                                         | ✅ web                                             |
| Desktop notifications on call start/end                                             | —                                                   | ✅ `notify-send`                                   |
| Metrics (OTel/Prometheus `/metrics`)                                                | —                                                   | ✅                                                 |

**Positioning:** they own the _content-creation_ side (look good in calls/streams); we own
the _Linux automation_ side (right mode at the right time, zero interaction). The PIXY's
headline motor/auto-tracking tricks are accessible from both.

**Zoom (and call apps) specifically:** the official app has **zero call-app awareness** —
no `zoom.us`/`Zoom.exe`/`InMeeting`/process-detection strings on either platform; every
"zoom" hit is camera zoom (`softZoom`/`hardwareZoom`, UVC zoom params, `PreviewZoom.qml`,
`remote status:ZoomState` = wireless camera zoom state; Teams/Skype/Discord: 0 hits).
Their model: you manually pick "EMEET STUDIO Virtual Camera" inside Zoom; the app never
knows a call is happening. emeet-pixyd detects Zoom generically — `isCameraInUse`
(`process.go:95`) flags *any* non-self process holding `/dev/videoX`: native Zoom opens
the device directly; Flatpak Zoom routes via xdg-desktop-portal/PipeWire, where the
PipeWire daemon holds the fd while streaming, so it is detected too (attribution differs,
detection does not). Same mechanism covers Teams-web, browsers, and everything else.

---

## 4. Actionable Intelligence for emeet-pixyd

Ranked by value-to-effort for a Linux daemon:

1. **PTZ speed control** — the motor exposes a speed parameter we never set. Cheap HID
   add, visible smoothness win for preset recall/relative moves.
2. **Battery/charge status query** (`CMD_GET_BATTERY_LEVEL`, `CMD_GET_CHARGE_STA`) —
   read-only, trivially surfaced in `status`/Waybar/web.
3. **Privacy trigger time** (auto-privacy after N minutes idle?) — likely maps to our
   auto-privacy philosophy; investigate semantics via HID query.
4. **Motor-side presets + power-on default position** — we could mirror our named presets
   onto the motor's own preset slots so they survive host changes / power cycles.
5. **Tracking-mode variants** (Face/HalfBody/FullBody) — if the protocol exposes the
   selector, a `tracking face|halfbody|fullbody` command is a small, high-visibility add.
6. **Firmware version query** (`CMD_GET_DEVICE_VER`) — display-only; full firmware
   upgrade stays out of scope (their MCU flash flow is risky and the fw API currently
   404s).
7. **Image controls** (HDR toggle, flip/reverse) — HID-settable; nice-to-have.

Explicitly **not** worth chasing: beauty/OBS/virtual-device stack (wrong problem for a
daemon), wireless/elink (hardware we don't have), SD/LED/HDMI (studio features),
Kuaishou/accounts.

---

## 5. Reverse-Engineering Notes (reusable)

### 5.1 Inno Setup 6.6.1 (unsupported by innoextract 1.10-dev)

Format fully decoded from `jrsoftware/issrc` tag `is-6_6_1` sources + CRC-verified Python
parsing (no C++ needed in the end):

- Loader offset table at `rDlPtS` magic (file offset `0xdec48`), revision 2: ID[12] +
  u32 ver=2 + i64 TotalSize + i64 OffsetEXE + u32 UncSizeEXE + i32 CRCEXE + i64 Offset0 +
  i64 **Offset1** + u32 pad + i32 tableCRC.
- **Offset1 = start of the file-data chunk** (`FSourceF.Seek(SetupLdrOffset1 + FL.StartOffset)`
  in `Setup.FileExtractor.pas:249`) — this is where the payload LZMA stream lives, _no
  `idskb32` slice marker exists anymore_.
- Setup-0 at Offset0: SetupID[64] → u32 CRC (= CRC of the 49-byte encryption header, not
  the decompressed header) → TSetupEncryptionHeader[49] → compressed blocks.
- Block framing: `[u32 hdrCRC][u32 StoredSize][u8 Compressed]` (hdrCRC = CRC32 of the
  5-byte header) then data as chunks `[u32 chunkCRC][≤4096 bytes]`; each block = one
  independent LZMA1 stream with 5-byte props (`5d 00 00 80 00`, 8 MB dict) read from
  stream start (`Compression.LZMA1SmallDecompressor.pas` — no size field → Python
  `lzma.FORMAT_RAW` with matching dict).
- TSetupHeader 6.6.1: 34 strings + 4 ansi + 17 counts (ISSigKey count between Dir and
  File) + 2×10 version + 63 bytes of tail ints/enums + 6-byte options set (46 flags).
  Strings are u32-length + UTF-16LE.
- Entry order stream 1: Language(4str+4ansi+19B) → CustomMessage(2str+4B) → Permission →
  Type/Component/Task → Dir(7str+27B) → ISSigKey(3str) → **File**(15str+1ansi+77B incl.
  SHA256+verification enum) → **Icon**(13str+48B) → Ini → Registry → Deletes →
  Run(13str+27B) → 4× wizard-image groups (i32 count, −1 = reuse) → optional DLLs.
  Stream 2: **FileLocation only** (89B: slice ints, 4×i64 offsets/sizes, SHA256, FILETIME,
  file version, 5-flag set) — _not_ interleaved with Icons (this corrected an earlier
  assumption).
- All 2,211 files live in **one solid LZMA chunk** (347.8 MB original → 136.2 MB stored)
  at `Offset1+9`; `ChunkSuboffset`/`OriginalSize` locate each file inside it.
- Working Python toolchain in `/tmp/emeet/`: `extract_setup0.py`, `setup0_parse.py`,
  `finish_parse.py`, `extract_files.py`, outputs `parsed.json` / `locations.json` /
  `win/` (full payload).

### 5.2 Windows driver install flow (their own bats, heavily commented)

- vcam: `regsvr32` of a DirectShow filter; they verify _both_ the CLSID key and the
  DirectShow video-input category instance key (security software wipes the latter,
  making the camera invisible while "registered").
- Virtual mic: `devcon install` of a root-enumerated kernel driver; a registry RefCount
  (`HKLM\SOFTWARE\EMEET\VirtualAudio`) shared across app versions decides real removal.
- Their `.bat` files are ASCII-only by policy (CJK comments break on GBK/CP932/UTF-8
  consoles and _execute as commands_) — a war story worth remembering.

### 5.3 Quirks found

- The audio driver INF's DeviceDesc is literally **"EMEET Virtual Camera"** (copy-paste
  legacy; their own install bat comments apologize for it — the .cat signature forbids
  editing the INF).
- `KDFIterations = 220000` (Argon2-ish KDF params in the unused encryption header) even
  for unencrypted installs.
- App self-update config: `https://emeet.ai/software/eMeetLink/software_upgrade_config.json`.

---

## 6. Conclusion

The official EMEET STUDIO app validates emeet-pixyd's core thesis: on Linux, the
device-control half of the PIXY experience deserves a first-class _automation_ citizen,
not a 350 MB OBS fork. Our HID implementation matches their wire protocol; our automation
layer (call detection, auto modes, PipeWire, Waybar, web UI) is genuine differentiation
they don't attempt on any platform. The highest-value gaps worth closing are small HID
surface additions (PTZ speed, battery status, tracking-mode variants, motor presets),
enumerated in §4.
