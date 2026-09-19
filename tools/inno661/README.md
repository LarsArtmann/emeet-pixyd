# Inno Setup 6.6.1 extractor (pure Python)

Toolchain that reverses the EMEET STUDIO 2.0.3 Windows installer
(`EMEET_STUDIO_V2.0.3_hotfix_intl_Win.exe`), written from the authoritative
Pascal sources (`jrsoftware/issrc@is-6_6_1`, `Projects/Src/Shared.Struct.pas`).
Created 2026-09-17 because innoextract did not support 6.6.1; every layer is
CRC/SHA256-verified. Analysis context:
[`docs/emeet-studio-official-app-comparison.md`](../../docs/emeet-studio-official-app-comparison.md) §5.

**Verification status: all 2,211 payload files SHA256-match the digests Inno
itself recorded** (`verify.py` exit 0, 2026-09-17).

**Verified on a second installer**: the toolchain transferred unchanged to
`EMEET_STUDIO_V2.0.0-Beta.25_cn_Win.exe` (2026-09-19, re-acquired via the
Wayback Machine; installer + full extraction live in
`~/specimens/emeet-studio/`). Its parse data is committed as
`data/parsed-beta25.json` + `data/setup0_offsets-beta25.json`; the Beta.25
payload extraction (`work/out/`, incl. the disassembled x64 `EMEET STUDIO
2.exe`) is regenerable from the durable installer with the commands below.

## Usage

```bash
mkdir -p data OUTDIR   # the scripts do not create directories
python3 extract_setup0.py INSTALLER.exe data/setup0.bin   # loader table + setup-0 de-chunk
python3 finish_parse.py    data/setup0.bin data/parsed.json
python3 extract_files.py   INSTALLER.exe data/parsed.json data/setup0_offsets.json OUTDIR/
python3 verify.py          data/parsed.json OUTDIR/       # exit 0 = all digests match
```

`data/parsed.json` (pre-parsed for this installer: 2,212 file entries,
2,211 file locations with SHA256 digests, icons, run/uninstall-run entries)
and `data/setup0_offsets.json` are committed so the format knowledge survives
even if the installer itself is lost.

## Format summary (Inno Setup 6.6.1)

- **Loader offset table** at the `rDlPtS` magic (near EOF): ID[12] + u32 ver=2
  - i64 TotalSize + i64 OffsetEXE + u32 UncSizeEXE + i32 CRCEXE
  - i64 **Offset0** (setup-0 header stream) + i64 **Offset1** (file data)
  - u32 pad + i32 CRC. For this installer: Offset0=`0x82c9df0`,
    Offset1=`0xdfa00`.
- **Setup-0** (at Offset0): SetupID[64] + u32 CRC (covers the 49-byte
  encryption header that follows) + TSetupEncryptionHeader{u8 EncryptionUse=0,
  KDFSalt[16], u32 KDFIterations=220000, BaseNonce[12], PasswordTest[4]},
  then blocks.
- **Block framing**: `[u32 hdrCRC][u32 StoredSize][u8 Compressed]`
  (hdrCRC = CRC32 of the 5 header bytes); data arrives as chunks
  `[u32 chunkCRC][<=4096B]`. Each block is an independent LZMA1 raw stream:
  5 props bytes (`5d 00 00 80 00`, 8 MB dict) then data — **no size field**
  (`lzma.FORMAT_RAW` after stripping the props). End of blocks is detected by
  a header-CRC mismatch on trailing padding.
- **TSetupHeader** (in the first decompressed block): 34 UTF-16LE length-
  prefixed strings (u32 byte-len, >=0xFFFFFFF0 = NULL) + 4 ansi + 17 i32
  counts + 20B versions + **56B tail** + 6B Options.
- **Stream 1 entry order**: Language(4str+4ansi+19B) → CustomMessage(2str+4B)
  → Permission(ansi) → Type(4str+30B) → Component(5str+42B) → Task(6str+26B)
  → Dir(7str+27B) → ISSigKey(3str) → **File(15str+1ansi+77B)** →
  Icon(13str+48B incl. 16B TGUID) → Ini(10str+21B) → Registry(9str+29B) →
  InstallDelete/UninstallDelete(7str+21B) → Run/UninstallRun(13str+27B) →
  4x wizard-image groups → optional DLLs.
  File tail: SHA256[32] (all-zero in this installer) + Typ[1] + vers[20] +
  loc[4] + attribs[4] + extsize[8] + perms[2] + opts[5] + type[1].
- **Stream 2 = FileLocation ONLY** (89B each): first[4]+last[4] (slices) +
  start[8]+sub[8]+orig[8]+comp[8] + SHA256[32] + FILETIME[8] + ver[8] +
  flags[1]. `comp` is the whole solid chunk's compressed size (shared);
  `sub` locates the file inside the **decompressed** solid stream.
- **File data**: ONE solid chunk at Offset1 (`Setup.FileExtractor.pas:249`):
  `zlb\x1a` + 5 LZMA props bytes + a single LZMA1 raw stream (8 MB dict).
  No `idskb32` marker in 6.6.1. Layout on disk: `[loader][solid chunk][setup-0]`.
- **Call-instruction optimization**: PE files flagged
  `floCallInstructionOptimized` (flags bit 2) are stored with E8/E9 rel32s
  transformed (`Compression.Base.pas` TransformCallInstructions, 64KB blocks
  with cumulative wrapping AddrOffset). `extract_files.py` decodes them so the
  output equals what the installer writes to disk; the FileLocation SHA256
  hashes the decoded original. `calltransform.py` is the port.

## Files

| File                       | Purpose                                              |
| -------------------------- | ---------------------------------------------------- |
| `extract_setup0.py`        | Locate loader table, de-chunk + decompress setup-0   |
| `setup0_parse.py`          | Parser module: header, all entry types, locations    |
| `finish_parse.py`          | Drive full parse → `parsed.json`                     |
| `extract_files.py`         | Extract + call-decode all payload files              |
| `calltransform.py`         | TransformCallInstructions port                       |
| `verify.py`                | SHA256-verify extracted tree against Inno digests    |
| `data/parsed.json`         | Pre-parsed entries for this installer (with digests) |
| `data/setup0_offsets.json` | Loader offsets for this installer                    |

## Notes

- The extraction that informed `docs/emeet-studio-official-app-comparison.md`
  (in `/tmp/emeet/win`, now superseded) was in _stored_ (encoded) form; for
  strings analysis both forms are equivalent — the transform only rewrites
  rel32 operands of CALL/JMP instructions.
- Installer facts: AppId `{{C32F7F8G-973B-5G0B-B472-22F5633048CB}`, 9
  languages, 2,212 file entries (2,211 present + 1 uninstall-only type-1),
  4 icons, 1 Run (launch app), 3 UninstallRun (taskkill, vcam unregister,
  VirtualAudio uninstall).
