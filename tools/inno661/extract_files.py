#!/usr/bin/env python3
"""Extract all payload files from the Inno Setup 6.6.1 solid chunk.

Inno 6.6.1 stores file data as ONE solid chunk at loader Offset1
(Setup.FileExtractor.pas:249: Seek(SetupLdrOffset1 + FL.StartOffset)).
At Offset1: 'zlb\\x1a' magic + 5 LZMA props bytes + a single LZMA1 raw
stream (8 MB dict). Files are located by their FileLocation ChunkSuboffset
/ OriginalSize inside the decompressed stream, extracted sequentially.

Usage: extract_files.py INSTALLER.exe PARSED.json OFFSETS.json [OUTDIR]
"""
import json
import lzma
import os
import sys
import time

from calltransform import FLO_CALL_INSTRUCTION_OPTIMIZED, decode_stored


def main() -> None:
    installer, parsed_path, offsets_path = sys.argv[1:4]
    outdir = sys.argv[4] if len(sys.argv) > 4 else "extracted"
    offs = json.load(open(offsets_path))
    chunk_start = offs["offset1"] + 9      # skip 'zlb\x1a' + 5 props bytes
    chunk_end = offs["offset0"]            # data ends where setup-0 begins

    data = open(installer, "rb").read()
    chunk_in = data[chunk_start:chunk_end]
    d = json.load(open(parsed_path))
    locs = d["locs"]
    files = d["files"]

    targets = []
    for i, f in enumerate(files):
        if f["type"] == 1:
            continue  # type 1 = uninstall-only file not present in the chunk
        loc = locs[f["loc"]]
        dest = f["dest"].replace("{app}", "").replace("{tmp}", "_tmp")
        path = os.path.join(outdir, dest.lstrip("\\"))
        targets.append((loc["sub"], loc["orig"], loc["flags"], path, i))
    targets.sort()
    print(f"{len(targets)} files, total {sum(t[1] for t in targets) / 1e6:.1f} MB",
          flush=True)
    dec = lzma.LZMADecompressor(
        format=lzma.FORMAT_RAW,
        filters=[{"id": lzma.FILTER_LZMA1, "dict_size": 0x800000}])
    in_pos = 0
    in_step = 8 * 1024 * 1024
    pending = b""
    t0 = time.time()
    written = 0
    for sub, orig, flags, path, idx in targets:
        os.makedirs(os.path.dirname(path), exist_ok=True)
        while (len(pending) < orig and not dec.eof and in_pos < len(chunk_in)):
            pending += dec.decompress(chunk_in[in_pos:in_pos + in_step])
            in_pos += in_step
        blob, pending = pending[:orig], pending[orig:]
        if flags & FLO_CALL_INSTRUCTION_OPTIMIZED:
            blob = decode_stored(blob, call_optimized=True)
        with open(path, "wb") as out:
            out.write(blob)
        written += len(blob)
        if len(blob) < orig:
            print(f"TRUNCATED file {idx}: got {len(blob)} of {orig}")
            sys.exit(1)
        if idx % 200 == 0:
            print(f"  file {idx}: {written / 1e6:.1f} MB, {time.time() - t0:.0f}s",
                  flush=True)
    print(f"DONE: {written / 1e6:.1f} MB in {time.time() - t0:.0f}s -> {outdir}")


if __name__ == "__main__":
    main()
