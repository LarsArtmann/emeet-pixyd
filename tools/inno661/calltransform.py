#!/usr/bin/env python3
"""Port of Inno Setup's TransformCallInstructions (Compression.Base.pas, [Version 3]).

Converts relative addresses in x86/x64 CALL ($E8) and JMP ($E9) instructions
to absolute-buffer-relative addresses (Encode=True) or the inverse
(Encode=False). The installer decodes each file in <=64 KB blocks with a
cumulative wrapping AddrOffset — chunking is load-bearing, do not flatten.
"""

BLOCK = 65536
FLO_CALL_INSTRUCTION_OPTIMIZED = 4


def transform_block(buf: bytearray, encode: bool, addr_offset: int) -> None:
    size = len(buf)
    if size < 5:
        return
    size -= 4
    i = 0
    while i < size:
        op = buf[i]
        if op == 0xE8 or op == 0xE9:
            i += 1
            if buf[i + 3] == 0x00 or buf[i + 3] == 0xFF:
                addr = (addr_offset + i + 4) & 0xFFFFFF
                rel = buf[i] | (buf[i + 1] << 8) | (buf[i + 2] << 16)
                if not encode:
                    rel = (rel - addr) & 0xFFFFFFFF
                if rel & 0x800000:
                    buf[i + 3] = (~buf[i + 3]) & 0xFF
                if encode:
                    rel = (rel + addr) & 0xFFFFFFFF
                buf[i] = rel & 0xFF
                buf[i + 1] = (rel >> 8) & 0xFF
                buf[i + 2] = (rel >> 16) & 0xFF
            i += 4
        else:
            i += 1


def decode_stored(blob: bytes, call_optimized: bool) -> bytes:
    """Turn stored-chunk bytes into installed-on-disk bytes."""
    if not call_optimized:
        return blob
    out = bytearray(blob)
    addr_offset = 0
    for off in range(0, len(out), BLOCK):
        block = out[off:off + BLOCK]
        transform_block(block, encode=False, addr_offset=addr_offset)
        out[off:off + BLOCK] = block
        addr_offset = (addr_offset + len(block)) & 0xFFFFFFFF
    return bytes(out)
