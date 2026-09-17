#!/usr/bin/env python3
"""Extract the EMEET STUDIO V2Head HID command-ID table from the macOS binary.

The official app constructs every HID command from a static EMHidCmdV2Head
global (C++ ctor `EMHidCmdV2Head::EMHidCmdV2Head(u8,u8,u8,u8)`), initialized
by static initializers that pass four immediate constants. This script:

  1. carves the arm64 slice out of the fat Mach-O,
  2. dumps the disassembly + chained-fixup binds once (llvm-objdump),
  3. joins every `bl EMHidCmdV2HeadC1Ehhhh` site with its four `mov w1..w4`
     immediates and the GOT slot -> CMD_*_E symbol it initializes.

Output: cmdtable.json = { "CMD_...": [b0, b1, b2, b3] }.

Usage (NixOS): nix shell nixpkgs#llvm --command python3 extract_cmdtable.py <PATH-TO-MAC-BINARY>
Requires ~500 MB scratch and a few minutes.
"""
import json
import re
import struct
import subprocess
import sys
import tempfile
from pathlib import Path

ARM64_CPUTYPE = 0x0100000C
CTOR_ADDR = "0x1005cde50"  # EMHidCmdV2Head::EMHidCmdV2Head(u8,u8,u8,u8) - slice-specific, verify via llvm-nm


def carve_arm64(binary: str, out: Path) -> None:
    with open(binary, "rb") as f:
        head = f.read(4096)
    magic, nfat = struct.unpack(">II", head[:8])
    if magic != 0xCAFEBAbe & 0xFFFFFFFF:  # noqa: PLR0124 - readability
        raise SystemExit("not a fat Mach-O; expected 0xcafebabe")
    off = 8
    for _ in range(nfat):
        cputype, _sub, offset, size, _align = struct.unpack(">IIIII", head[off:off + 20])
        if cputype == ARM64_CPUTYPE:
            with open(binary, "rb") as f:
                f.seek(offset)
                out.write(f.read(size))
            return
        off += 20
    raise SystemExit("no arm64 slice found")


def run(cmd: list[str]) -> str:
    return subprocess.run(cmd, capture_output=True, text=True, check=True).stdout


def main() -> None:
    binary = sys.argv[1]
    here = Path(__file__).parent
    arm64 = here / "arm64.bin"
    carve_arm64(binary, arm64)

    nm = run(["llvm-nm", str(arm64)])
    m = re.search(r"^([0-9a-f]+) T __ZN14EMHidCmdV2HeadC1Ehhhh$", nm, re.M)
    if not m:
        raise SystemExit("EMHidCmdV2Head ctor symbol not found - binary layout changed")
    ctor = m.group(1)

    fixups = run(["llvm-objdump", "--macho", "--chained-fixups", "--dyld-info", str(arm64)])
    slot2sym = {}
    for line in fixups.splitlines():
        if "__got" not in line or "bind" not in line:
            continue
        mm = re.match(r"__DATA_CONST __got\s+(0x[0-9A-Fa-f]+)\s+\S+\s+bind.*?(__Z\S+)", line)
        if mm:
            slot2sym[int(mm.group(1), 16)] = mm.group(2)

    disasm = run(["llvm-objdump", "--disassemble", str(arm64)]).splitlines()
    pat_ldr = re.compile(r"ldr\tx0, \[x0, #(0x[0-9a-f]+)\]")
    pat_mov = re.compile(r"mov\tw([1-4]), #(0x[0-9a-f]+|-0x[0-9a-f]+)")
    pat_bl = re.compile(rf"bl\s+0x{int(ctor, 16):x}\b")

    out = {}
    for i, line in enumerate(disasm):
        if not pat_bl.search(line):
            continue
        ldr_slot, movs = None, {}
        for j in range(max(0, i - 30), i):
            ml = pat_ldr.search(disasm[j])
            if ml and ldr_slot is None:
                ldr_slot = int(ml.group(1), 16)
            mm = pat_mov.search(disasm[j])
            if mm:
                movs[int(mm.group(1))] = int(mm.group(2), 16)
        if ldr_slot is None or not all(k in movs for k in (1, 2, 3, 4)):
            continue
        sym = slot2sym.get(0x106a9c000 + ldr_slot)  # __DATA_CONST __got base
        if sym is None:
            continue
        if sym.startswith("__ZGVN"):  # guard slot -> object symbol
            sym = "__ZN" + sym[len("__ZGVN"):]
        name = re.sub(r"^__ZN14EMHidCmdHelper\d+", "", sym)
        name = name[:-1] if name.endswith("E") else name
        out[name] = [movs[k] for k in (1, 2, 3, 4)]

    table = {k: v for k, v in sorted(out.items())}
    json.dump(table, open(here / "cmdtable.json", "w"), indent=1)
    print(f"extracted {len(table)} commands -> cmdtable.json")
    arm64.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
