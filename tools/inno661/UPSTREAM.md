# Upstream contribution draft — innoextract + Inno Setup 6.6.1

**Status: STAGED, NOT SENT.** Filing is Lars's call (plan M26 / TODO #148).
This file contains everything needed to fire: the issue text, the PR outline,
and the evidence pointers. Target project: `dscharrer/innoextract`.

## Why this belongs upstream

innoextract 1.10-dev (current master) rejects Inno Setup 6.6.1 installers with
an unsupported-version error. The format changes in 6.6.1 are small and fully
characterized below; a working pure-Python implementation exists that extracts
**every payload byte-identically** (all 2,211 files SHA256-match the digests
Inno itself records in its own headers), so the format claims are verifiable
against a real-world installer, not just the Pascal sources.

## Evidence package (all committed in LarsArtmann/emeet-pixyd)

- `tools/inno661/README.md` — the distilled 6.6.1 format spec (below).
- `tools/inno661/*.py` — reference implementation: `extract_setup0.py`
  (loader table + de-chunking), `finish_parse.py` (header parse),
  `extract_files.py` (payload extraction incl. the call-instruction
  transform), `verify.py` (SHA256 proof).
- `tools/inno661/data/parsed.json` + `data/setup0_offsets.json` — parse
  output for the specimen installer.
- Test specimen: `EMEET_STUDIO_V2.0.3_hotfix_intl_Win.exe` (Inno 6.6.1,
  unnencrypted, 2,211 files, solid LZMA1).

## Suggested issue text

Title: "Inno Setup 6.6.1 support (format deltas characterized; reference
implementation with byte-identical extraction)"

Body:

> innoextract 1.10-dev cannot read Inno Setup 6.6.1 installers. I
> reverse-engineered the 6.6.1 deltas against the authoritative Pascal
> sources (jrsoftware/issrc tag is-6_6_1) and validated a reference
> implementation on a real 6.6.1 installer: all 2,211 extracted files match
> the SHA256 digests Inno records in its own headers, byte-identically.
>
> The deltas vs the last supported version, condensed (full spec at
> tools/inno661/README.md in LarsArtmann/emeet-pixyd):
>
> 1. **Loader offset table** unchanged in shape (rDlPtS magic, ID[12] +
>    version u32) — 6.6.1 writes version 2 with the same field layout.
> 2. **Setup-0 encryption header**: `TSetupEncryptionHeader` now sits
>    between SetupID and the block stream — `{u8 EncryptionUse, KDFSalt[16],
>    u32 KDFIterations, BaseNonce[12], PasswordTest[4]}` (49 bytes), with a
>    u32 CRC covering it. Unencrypted installers still write the full struct
>    with `EncryptionUse=0`.
> 3. **Block framing** unchanged: `[u32 hdrCRC][u32 StoredSize][u8
>    Compressed]`, per-block independent LZMA1 raw streams with 5 props
>    bytes and no size field; end detection via header-CRC mismatch on
>    trailing padding.
> 4. **TSetupHeader**: 34 UTF-16LE length-prefixed strings, 4 ansi, 17 i32
>    counts, 20B versions, **56-byte tail**, 6B options; NULL string marker
>    is `>= 0xFFFFFFF0`.
> 5. **File entry** (stream 1): 15 UTF-16 strings + 1 ansi + 77-byte tail
>    ending in `SHA256[32]` (observed all-zero in the specimen),
>    `Typ[1] vers[20] loc[4] attribs[4] extsize[8] perms[2] opts[5]
>    type[1]`.
> 6. **FileLocation** (stream 2): 89 bytes/entry — slices `[4][4]`, offsets
>    `start[8] sub[8] orig[8] comp[8]`, `SHA256[32]`, `FILETIME[8]`,
>    `ver[8]`, `flags[1]`. `comp` is the compressed size of the WHOLE solid
>    chunk (shared across files); `sub` locates the file inside the
>    DECOMPRESSED solid stream.
> 7. **Stored-vs-installed transform**: files with the call-optimization
>    flag are stored with `E8/E9` rel32 operands adjusted by a cumulative
>    `AddrOffset` (64KB blocks) — `Compression.Base.pas`
>    `floCallInstructionOptimized`. Extraction must invert this or the
>    affected binaries corrupt silently. This is the trap most likely to
>    bite a naive implementation: without the inverse transform the
>    extraction LOOKS successful.
> 8. **File data**: one solid chunk at the loader's Offset1: `zlb\x1a` +
>    5 LZMA props bytes (8 MB dict) + a single LZMA1 raw stream.
>
> A pure-Python reference (extract → parse → extract files → verify) lives
> at tools/inno661/ in LarsArtmann/emeet-pixyd; happy to distill any part
> into a patch against innoextract's setup/ directory.

## PR outline (if the maintainers want code)

- `src/setup/version.cpp`: accept 6.6.1 (`INNO_VERSION(6, 6, 1)`).
- `src/setup/header.cpp`: the 56-byte header tail + encryption-header skip.
- `src/setup/file.cpp` / `src/stream.cpp`: the 89-byte FileLocation row and
  the solid-chunk `sub`-offset model (files are slices of ONE decompressed
  stream, not independent compressed blobs).
- `src/util/loadcalltransform` (or equivalent): the E8/E9 `AddrOffset`
  inverse — port of `tools/inno661/calltransform.py`, which is itself a
  port of `Compression.Base.pas`.
- Tests: run `verify.py` flow against the specimen; the SHA256 proof makes
  the test binary-pass/fail unambiguous.

## Checklist before sending

1. Lars's go (this file is the whole ask).
2. Re-run the toolchain end-to-end if the specimen installer is re-downloaded
   (`/tmp` copy died with reboot; emeet.ai hosts it).
3. Check innoextract master for movement on 6.6.x (issue search "6.6").
4. Trim the issue body if a maintainer prefers short PRs; the spec link
   carries the detail.
