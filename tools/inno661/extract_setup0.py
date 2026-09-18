#!/usr/bin/env python3
"""Locate the Inno Setup 6.6.1 loader offset table and de-chunk the setup-0 stream.

The loader's offset table sits at the `rDlPtS` magic near the end of the file:

    ID[12] ("rDlPtS" + version bytes) + u32 ver=2 + i64 TotalSize
    + i64 OffsetEXE + u32 UncSizeEXE + i32 CRCEXE
    + i64 Offset0 (setup-0 header stream) + i64 Offset1 (setup-1 file data)
    + u32 pad + i32 CRC

At Offset0 the setup-0 stream begins:

    SetupID[64] + u32 CRC (covers the 49-byte encryption header)
    + TSetupEncryptionHeader{u8 EncryptionUse=0, KDFSalt[16],
      u32 KDFIterations, BaseNonce[12], PasswordTest[4]} + blocks

Each block: [u32 hdrCRC][u32 StoredSize][u8 Compressed] where hdrCRC = CRC32 of
the 5 header bytes; the stored data is chunks of [u32 chunkCRC][<=4096 bytes].
Each block decompresses as an independent LZMA1 raw stream (5-byte props
prefix, typically 5d 00 00 80 00 = lc3/lp0/pb2, 8 MB dict).

Usage: extract_setup0.py INSTALLER.exe [OUT.bin]
"""

import json
import lzma
import struct
import sys
import zlib

LOADER_MAGIC = b"rDlPtS"


def find_loader_table(data: bytes) -> dict:
    idx = data.rfind(LOADER_MAGIC)
    if idx < 0:
        raise SystemExit("loader magic rDlPtS not found")
    tbl = idx + len(LOADER_MAGIC) + 6  # skip 6 version bytes
    (ver,) = struct.unpack_from("<I", data, tbl)
    (total,) = struct.unpack_from("<q", data, tbl + 4)
    (off_exe,) = struct.unpack_from("<q", data, tbl + 12)
    (unc_exe,) = struct.unpack_from("<I", data, tbl + 20)
    (crc_exe,) = struct.unpack_from("<i", data, tbl + 24)
    (off0,) = struct.unpack_from("<q", data, tbl + 28)
    (off1,) = struct.unpack_from("<q", data, tbl + 36)
    return {
        "table_offset": idx,
        "version": ver,
        "total_size": total,
        "offset_exe": off_exe,
        "unc_size_exe": unc_exe,
        "crc_exe": crc_exe,
        "offset0": off0,
        "offset1": off1,
    }


def dechunk_setup0(data: bytes, base: int, verbose: bool = True) -> bytes:
    s = data[base:]
    pos = 64 + 4 + 49  # setup id + header crc + encryption header
    out = bytearray()
    while pos + 9 <= len(s):
        (hdr_crc,) = struct.unpack("<I", s[pos : pos + 4])
        stored, compressed = struct.unpack("<IB", s[pos + 4 : pos + 9])
        if stored == 0 and compressed == 0:
            break
        calc = zlib.crc32(s[pos + 4 : pos + 9]) & 0xFFFFFFFF
        if calc != hdr_crc:
            # Trailing padding after the last block fails the header CRC —
            # that mismatch IS the end-of-stream marker (verified empirically).
            if verbose:
                print(f"end of blocks at {pos:#x} (header CRC mismatch on padding)")
            break
        pos += 9
        blob = bytearray()
        remaining = stored
        while remaining > 0:
            (ccrc,) = struct.unpack("<I", s[pos : pos + 4])
            clen = min(remaining - 4, 4096)
            chunk = s[pos + 4 : pos + 4 + clen]
            if zlib.crc32(chunk) & 0xFFFFFFFF != ccrc:
                raise SystemExit(f"chunk CRC mismatch at {pos:#x}")
            blob += chunk
            pos += 4 + clen
            remaining -= 4 + clen
        if compressed:
            props = bytes(blob[:5])
            (dict_size,) = struct.unpack("<I", props[1:5])
            filters = [{"id": lzma.FILTER_LZMA1, "dict_size": dict_size}]
            dec = lzma.LZMADecompressor(format=lzma.FORMAT_RAW, filters=filters)
            blob = dec.decompress(bytes(blob[5:]))
        out += blob
        if verbose:
            print(
                f"block: stored={stored} compressed={compressed} "
                f"out={len(blob)} total={len(out)}"
            )
    return bytes(out)


def main() -> None:
    installer = sys.argv[1]
    out_bin = sys.argv[2] if len(sys.argv) > 2 else "setup0.bin"
    data = open(installer, "rb").read()
    offs = find_loader_table(data)
    print(f"loader table: {offs}")
    with open(out_bin.replace(".bin", "_offsets.json"), "w") as f:
        json.dump(offs, f, indent=1)
    blob = dechunk_setup0(data, offs["offset0"])
    open(out_bin, "wb").write(blob)
    print(f"wrote {len(blob)} bytes to {out_bin}")


if __name__ == "__main__":
    main()
