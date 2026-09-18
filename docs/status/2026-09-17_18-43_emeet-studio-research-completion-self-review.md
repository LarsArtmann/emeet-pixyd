# EMEET STUDIO Deep Research — Completion & Self-Review Status

**2026-09-17 18:43 CEST** — follow-up to `2026-09-17_15-09_emeet-studio-official-app-deep-research.md`.
Covers only this session's run: finishing the Windows extraction, the analysis, the
comparison report, and an honest look at what I did badly.

**Deliverable:** `docs/emeet-studio-official-app-comparison.md` (committed in `61ce62b`).
**Working artifacts:** `/tmp/emeet/` (ephemeral! see §d/§f).

**Resolution:** ~~f) verification & preservation items 1–5~~ all done (#142 `tools/inno661/` + SHA256 proof, #147 /tmp cleanup + AGENTS block, CHANGELOG entry, AGENTS pointer); protocol follow-ups 6/9/13–15/18 done (#143/#145, map doc §5, `hid-protocol.md` cross-check). STILL OPEN: f7 hardware battery verdict (#144/#139), f10 privacy-trigger-time (ROADMAP), f11 elink doc (ROADMAP), f12 EMVideoInput (ROADMAP), f16 innoextract PR (#148), f17 public page (ROADMAP question). Questions g1 (payload archival) and g3 (public page) remain Lars's calls.

---

## a) FULLY DONE

1. **Windows installer fully cracked — pure Python, no C++.** Abandoned the
   one-bug-away patched innoextract and decoded Inno Setup 6.6.1 from the authoritative
   `jrsoftware/issrc@is-6_6_1` sources + CRC brute-force verification:
   - Setup-0 layout: SetupID[64] → u32 enc-header CRC → 49-byte TSetupEncryptionHeader
     (EncryptionUse=0, KDF iters 220000) → blocks `[u32 hdrCRC][u32 StoredSize][u8 Comp]`
     with per-≤4096-byte chunk CRCs; each block an independent LZMA1 stream with 5-byte
     props (`5d 00 00 80 00`, 8 MB dict, no size field → `lzma.FORMAT_RAW`).
   - TSetupHeader 6.6.1 byte-perfect (34 strings UTF-16LE, 17 counts incl. ISSigKey,
     63-byte tail, 6-byte options). AppId `{{C32F7F8G-973B-5G0B-B472-22F5633048CB}`,
     v2.0.3, x64-only, 9 languages, 24.7 KB compiled setup script.
   - **All entries parsed**: 9 languages, 123 custom messages, 11 dirs, 2,212 file
     entries, 4 icons, 1 Run (launch app), 3 UninstallRun (taskkill, vcam unregister,
     VirtualAudio uninstall), 4 wizard-image groups, 2,211 file locations.
   - **Corrected a wrong handoff assumption**: stream 2 contains _only_ FileLocation
     entries (89 bytes each, ends exactly at block-2 boundary); Icons/Run live in
     stream 1 right after File entries.
   - **Payload location found**: one solid LZMA chunk at **Offset1 + 9**
     (`Setup.FileExtractor.pas:249` — `Seek(SetupLdrOffset1 + FL.StartOffset)`); no
     `idskb32` slice marker exists in 6.6.1. 347.8 MB original → 136.2 MB stored.
2. **All 2,211 payload files extracted** to `/tmp/emeet/win/` in 8 s (8 MB-dict LZMA
   sequential pass, files located by ChunkSuboffset/OriginalSize).
3. **Windows analysis complete**:
   - vcam = user-mode DirectShow COM filter (`emeet_virtual_camera_x86/x64.dll`,
     CLSID `{4A197D07-…}`, regsvr32, dual-registry-key verification logic in their bat).
   - VirtualAudio = signed kernel WDM driver (`EmeetAudioDriver.sys`, devcon install on
     `ROOT\EmeetVirtualAudio`, RefCount-shared across app versions).
   - Embedded OBS: `obs.dll`, `libobs-d3d11/opengl/winrt.dll`, `obs-frontend-api.dll`,
     `obs-scripting.dll` (Lua); plugins EMVideoInput, image-source, obs-transitions,
     obs-volcengine-beauty **with the full ByteDance model/resource tree**.
   - Qt 6.8.3 on Windows = identical to Mac → one codebase, two Jenkins branches
     (`_EMEET_STUDIO_2.0-Windows_master` / `op-EMEET_STUDIO_2.0_macOS_master`, both
     `NPI2024_Audio_Video_App`).
   - Windows supports _more_ devices (Nova4K, C960Ultra, C63E4KDual, C60E4K, S600Light,
     E3164/E3165/E7002…); Windows HID command names ⊂ Mac's (96 vs ~150, zero win-only).
   - `Fic760xUsbUpgradeDll.dll` (WiFi-chip firmware flasher), fw API
     `fw.emeet.ai/api/v3/firmware/models/{model}`, eMeetLink self-update JSON, Kuaishou
     OAuth endpoints, `[HID_RACE_FIX]` present on Windows too.
4. **Comparison report written** (`docs/emeet-studio-official-app-comparison.md`):
   architecture, driver mechanics, parity table, official-only vs pixyd-only features,
   ranked gap recommendations, reusable Inno 6.6.1 reverse-engineering notes.
5. **Triage landed**: TODO_LIST #138–#141 (PTZ speed, battery status, tracking-mode
   variants, motor presets); header date updated; old status report closed with a
   resolution note. Auto-commit daemon picked everything up (`61ce62b`).

## b) PARTIALLY DONE

- **Extraction integrity**: verified structurally only (`file(1)` recognizes PE headers
  on the spot-checked binaries; file count and byte-total match the location table).
  The location SHA256 digests were parsed but **discarded** — no cryptographic
  verification of the 2,211 files was performed.
- **Windows string diff vs Mac**: command-set subset proof is solid, but two anomalies
  were noticed and hand-waved, not resolved: `TargetTrack/ObjectTrack` = 0 hits in the
  Windows exe (Mac: 8/32), `elink` only 24 hits (Mac: 1,296). Probably string/QML
  embedding differences — unproven.
- **Device list**: firmware-model endpoints were conflated with device-class names
  (`Emeet*`); the report's "E3164/E3165/E7002…" line carries an ellipsis I never
  resolved into a clean final list.

## c) NOT STARTED (from this session's own findings — all optional follow-ups)

- Mapping the official `CMD_GET_/CMD_SET_` names onto our `hid.go`/`commands.go`
  implementation (the actionable protocol-gap table that would power TODO #138–#141).
- Reading out the 2 extracted wizard images; probing `fw.emeet.ai` for any live model;
  documenting `elink`; extracting/inspecting `EMVideoInput.dll` pipe protocol.
- Preserving the Inno 6.6.1 Python toolchain beyond `/tmp` (reboot = gone).
- AGENTS.md pointer to the comparison doc (memory protocol says enduring context
  belongs there — I skipped it).
- CHANGELOG entry for the research deliverable.

## d) TOTALLY FUCKED UP (honest list)

1. **Wine/Xvfb retry was a waste**: launched it before checking that `nix shell
   nixpkgs#wineWowPackages.stable` wasn't cached — it tried to _build wine-wow from
   source_ and died (exit 2). No harm, but it was blind optimism; the Python path made
   it redundant within minutes.
2. **Sloppy offset arithmetic burned ~5 debug loops**: (i) a botched `cat >>` draft
   append with broken digest handling had to be trimmed out; (ii) a 40-vs-20-byte
   version-field skip bug in the refactored parser; (iii) a Python one-liner with
   unterminated string quoting; (iv) misread FILETIME anchors (j−79 vs j−76) that
   briefly made me distrust the _correct_ 89-byte location grid; (v) several minutes
   flip-flopping on field orders by eyeballing hexdumps instead of reading the Pascal
   record. The lesson I keep re-learning: compute offsets in code; hexdump-eyeballing
   is where errors breed.
3. **Extraction wrote backslash-literal filenames** (`bin\64bit\foo.dll` as a single
   filename) — predictable, and I fixed it after, but the right call was normalizing
   paths in the extractor from the start.
4. **Unverified claim shipped in TODO #139**: I wrote "PIXY has an internal battery the
   official app reports" — plausible only for **PIXY-Wireless**; the wired PIXY may
   not answer battery queries at all. The TODO should have carried that caveat.
5. **Digest data thrown away** (see §b) — I had the SHA256s in hand at parse time and
   dropped them from `locations.json`.
6. **No explicit cleanup**: job 099 died on its own, but three stale wine prefixes
   (`wineprefix`, `wineprefix64`, `wineprefixwow`, `wineprefixxvfb`) are squatting in
   `/tmp` (possibly GBs), and the half-patched `innoextract-src` tree sits abandoned
   mid-bug with no written abandon/finish decision.

## e) WHAT WE SHOULD IMPROVE (process, from this run)

- **Preserve ephemeral research tooling**: anything valuable in `/tmp` (parsers,
  parsed.json, strings dumps) should be committed or archived _at the moment it works_,
  not left for a reboot to eat. Same class of loss as TODO #130 (HyperFrames sources).
- **Verify with the data you already have**: when a format hands you checksums, use
  them — cheap cryptographic proof beats spot-checks.
- **Write caveats into tickets, not just reports**: #139's battery claim needed its
  "wireless-only?" question inline, where a future implementer will read it.
- **AGENTS.md discipline**: I updated TODO_LIST but skipped the memory file; enduring
  discoveries (Inno 6.6.1 spec location, comparison doc, artifact paths) should have
  landed there same-session.
- **Kill speculative side-quests faster**: the wine retry cost little, but it was
  launched on hope rather than a 10-second feasibility check (`nix path-info`).

## f) NEXT — up to 50, ranked (this session's scope only)

**Verification & preservation (do first, cheap):**

1. Re-run location parse keeping SHA256s; checksum all 2,211 extracted files (S).
2. Copy the Inno 6.6.1 toolchain (`extract_setup0.py`, `setup0_parse.py`,
   `finish_parse.py`, `extract_files.py`, `parsed.json`) into `tools/inno661/` or
   `docs/research/` before reboot (S).
3. Archive or trash the wine prefixes + dead innoextract tree; record the abandon
   decision (S).
4. Add AGENTS.md pointer to the comparison doc + `/tmp/emeet` inventory (S).
5. CHANGELOG entry for the research (S).

**Protocol follow-ups (power TODO #138–#141):**
6. Build the official-CMD ↔ our-HID mapping table against `hid-protocol.md` + `hid.go` (M).
7. Hardware test: does the wired PIXY answer `CMD_GET_BATTERY_LEVEL`? De-risk #139 (S).
8. Enumerate the exact MOTOR preset/speed command names from Mac strings; design the
sniff plan for #138/#141 (S).
9. Resolve the Windows `TargetTrack`/`ObjectTrack` string absence (feature-gated?
stripped? different naming?) (S/M).
10. Investigate `privacy trigger time` semantics from Mac strings (S).

**Deeper intel (optional):**
11. Document `elink` protocol shape from Mac strings (~90 families) (M).
12. Extract + inspect `EMVideoInput.dll` (their OBS↔app pipe) vs our MJPEG approach (M).
13. Clean final device matrix (device classes vs firmware models, both platforms) (S).
14. View the 2 wizard images (branding) (XS).
15. Probe `fw.emeet.ai/api/v3/firmware/models/*` for any live model + response shape (S).

**Upstream / public (needs Lars's call — see questions):**
16. Port Inno 6.6.1 support to innoextract upstream (I hold a verified format spec +
Python reference; innoextract is THE tool and currently blind to 6.6.1) (M/L).
17. Publish a distilled comparison (or roadmap deltas) on the website (S/M).
18. `hid-protocol.md`: add the official framing names (`EMHidCmdV2Head`, `_OLD`
variants, `EMHidCmdRecvFsm`, `[HID_RACE_FIX]`) as external validation notes (S).

_(Deliberately stopping at 18 — padding to 50 with unrelated work would violate the
"only this session" rule.)_

## g) Questions for Lars (cannot be answered from here)

1. **Preservation scope:** the full 348 MB extracted Windows payload and the ~1.27M-line
   strings dumps live only in `/tmp/emeet/`. Commit the _tooling + parsed JSON_ to the
   repo and let the payload itself die with `/tmp` — or archive the payload somewhere
   durable too (it's their copyrighted binaries; repo-fitting 348 MB is questionable)?
2. **innoextract upstream PR:** want me to turn the Inno 6.6.1 findings into an
   upstream contribution (C++ port of the verified format)? It's real OSS value but a
   half-day-plus of work in someone else's codebase — your time budget, your call.
3. **Public or internal:** the comparison doc reverse-engineers their installers,
   drivers, and firmware API. Keep it internal, or distill a public "how emeet-pixyd
   relates to EMEET STUDIO" page for the website (positioning win, but it documents
   their internals publicly)?

---

**Bottom line:** the research goal is fully delivered — both official apps unpacked and
analyzed, comparison written, gaps triaged. The failures this session were process, not
outcome: wasted wine retry, sloppy offset arithmetic, dropped checksums, an
unverified battery claim in a ticket, and everything valuable still living in `/tmp`.

---

## Resolution (2026-09-18)

All session work shipped; this report is retained as a point-in-time snapshot. See `CHANGELOG.md` and the successor reports in this directory for the durable record.
