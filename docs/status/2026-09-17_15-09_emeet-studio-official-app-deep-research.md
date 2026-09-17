# EMEET Studio Official-App Deep Research — Status

**2026-09-17 15:09 CEST** — requested by user. All times CEST.

## Executive summary

Researching what the official EMEET STUDIO apps (Windows EXE + macOS PKG, v2.0.3,
downloaded from emeetstudio S3) actually do, to compare against `emeet-pixyd`.
**The macOS app is fully analyzed and yields a near-complete feature picture.**
The Windows installer uses Inno Setup **6.6.1** — brand-new format, unsupported by
innoextract — so I reverse-engineered the format from the open-source Inno Setup
repo and patched a private innoextract build. That patch is **one bug away from
working** (header parse misaligns late in the struct). The final deliverable —
the comparison report — is **not yet written**.

## Completed work items

### 1. Downloads — DONE

- `/tmp/emeet/EMEET_STUDIO_V2.0.3_hotfix_intl_Win.exe` (138 MB, PE32, Inno Setup 6.6.1)
- `/tmp/emeet/EMEET_STUDIO_V2.0.3_intl_mac_20260903_110248.pkg` (322 MB, xar)

### 2. macOS app — DONE (fully analyzed)

Extracted via `xar` → gzip-cpio payload → `EMEET STUDIO.app` (581 MB uncompressed).

**What it is:** a Qt 6.8.3 / QML monolith (241 MB binary, `com.emeet.studio.next`)
that **embeds OBS Studio 31.1.2** — it is a virtual-camera studio suite, not a
control daemon. Key bundled components:

| Component                                                                   | Purpose                                                                    |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `libobs.framework` + obs-transitions/image-source/perspective plugins       | OBS-based scene composition                                                |
| `obs-volcengine-beauty.plugin`                                              | ByteDance Volcengine Effect SDK (`bef_effect_ai_*`) beauty/gesture filters |
| `EMVideoInput.plugin`                                                       | OBS pipe source "EMEET STUDIO video source"                                |
| `Library/SystemExtensions/…mac-camera-extension` (DriverKit CMIO, v0.3)     | macOS 12.3+ virtual camera                                                 |
| `VirtualAudioPlugin2.driver` (CoreAudio HAL, `/Library/Audio/Plug-Ins/HAL`) | virtual audio device                                                       |
| FFmpeg/mbedtls/librist/libsrt/libhidapi/libusb-1.0                          | streaming, TLS, wireless transport, HID                                    |

**HID command surface** (from strings, ~150 `CMD_GET/SET_*`): tracking modes
(Face/Half/Full-body + manual ObjectTrack with x,y,w,h), audio (AGC, input gain,
mute, denoise, music mode = nc/live/original), PTZ motor (position, relative,
**speed**, **on-device presets**, power-on default position), privacy trigger
time, gesture recognition per type, image controls (HDR, EV/WB/focus locks, WB
fine-tune, metering, power-line, LUT3D/color grading, OSD, flip/reverse,
multi-spectral), LED RGB light (color/brightness/mode/tally), SD-card recording
(format/start/stop/segments), HDMI output orientation, WiFi/AP/SSID config,
RTMP/RTSP/RIST push streaming, firmware upgrade (MCU/IC, from `fw.emeet.ai` +
`emeet.ai` JSON manifests — both now 404), factory reset, auto power-on/off,
RTC/NTP sync, battery/charge status, device alias/SN/password.

**"elink" wireless protocol** (~90 command families, `lib_elink_*`): full remote
control of PIXY-Wireless over WiFi incl. image params, streaming, SD, power.

**Multi-device support:** Pixy, Pixy **2K**, Pixy **Dual**, Piko, Piko+, Piko
Dual + wireless variants. Account system (login/registration), multi-device
manager, remote-control pairing, mini floating window, canvas (OBS-like) window,
AI "inspiration" pages, firmware-upgrade UI.

Also noted: their own HID concurrency bug tag `[HID_RACE_FIX]`; `UsbHidCtrl`
wraps hidapi; `EMHidCmdV2Head` V2 protocol header; Jenkins path
`op-EMEET_STUDIO_2.0_macOS_master/NPI2024_Audio_Video_App`.

### 3. Windows installer — IN PROGRESS (≈90% done)

- innoextract 1.10-dev (nixpkgs) fails: "Unexpected setup loader revision: 2"
- 7-Zip 26.02: no Inno handler (my misconception — wasted a step)
- wine 32-bit: installer is x64-only → aborts; wine64: no wow64 → fails;
  **wineWow64 background job still running** (shell 072, ~1h, no output yet)
- **Patched innoextract** (clone at `/tmp/emeet/innoextract-src`, builds green)
  against the Inno Setup 6.6.1 source (tags `is-6_4_1` vs `is-6_6_1`):
  - loader offset table revision 2 (i64 fields) — validated byte-exact vs hexdump
  - `Inno Setup Setup Data (6.6.1)` version entry
  - 49-byte `TSetupEncryptionHeader` skip after SetupID (size found by CRC32
    brute force; my Pascal size math gave 45 — nested unpacked `TSetupEncryptionNonce`
    is 24B due to alignment)
  - header: 34 strings, `NumISSigKeyEntries`, wizard-field reorder, password
    fields removed (moved to encryption header), wizard colors (4×u32+u8)
  - language/file/data entry layouts for 6.6.1; ISSig-key entry skip;
    4 wizard-image groups (−1 marker); SevenZip DLL skip
- **Current blocker:** header parse still misaligns late in the struct
  (enum garbage values 103/0x6c/105/115 = ASCII "glis" — stream lands in string
  data). `UninstallLogMode` now parses; `DirExistsWarning`/privileges/language
  detection are garbage. Next step (was mid-flight): temporary per-field offset
  logging in `header.cpp` to pinpoint the exact divergence, then fix the field
  guess, rebuild, extract.

### 4. Comparison report — NOT STARTED

The actual deliverable. All input data exists from the macOS analysis
(`FEATURES.md` read on our side; full official feature surface mapped).

## What I forgot / did badly

- **Chased the rabbit hole.** The user nudged twice ("better way?"). The Mac app
  already answers ~95% of "what do the official apps do"; Windows file-level
  extraction mainly adds driver-binary details. I should have timeboxed the
  Inno 6.6.1 reverse-engineering and written the report first.
- **Three wine attempts** (minimal/win64/wow) burned ~30+ min before switching
  to the (correct) source-driven innoextract patching. Inno Setup being open
  source made that the obvious first move.
- **Background job hygiene:** wine64 job ran ~50 min before I checked it;
  wineWow (072) has been running ~1h with no monitoring.
- Firmware manifest URLs all 404 — didn't try alternate paths (e.g. with
  version prefixes) before moving on.

## Top 3 next tasks

1. **Finish the Windows extraction** (≤1h): add offset logging to `header.cpp`,
   fix the late-struct misalignment, rebuild, `innoextract -d win …`; cross-check
   with wineWow job 072 output. Kill 072 once either path succeeds.
2. **Write the comparison report** (the deliverable): official capabilities vs
   `FEATURES.md` — same-parity areas (tracking/idle/privacy, PTZ, audio modes,
   gesture, presets, firmware info) and official-only areas (beauty/OBS virtual
   camera, virtual audio, wireless+streaming, SD recording, LED, image controls,
   battery, firmware upgrade, account system) with a recommendation of which
   gaps are worth closing on Linux.
3. **Persist findings** in the repo: HID command catalogue (extending
   `docs/hid-protocol.md`) + link the report from `docs/status/`.

## Open questions (max 3)

1. Is finishing the Windows file-level extraction worth ~1 more hour, or should
   I write the comparison report now from the macOS analysis and treat Windows
   as "same Qt app + drivers (confirmed by installer metadata)"?
2. Where should the final comparison live — chat only, `docs/` in this repo, or
   the website?
3. Should official-app-only capabilities (e.g. PTZ speed, on-device presets,
   battery status, tracking-mode variants, privacy trigger time) be triaged into
   `TODO_LIST.md`/`ROADMAP.md` as candidates, or is that out of scope for this task?
