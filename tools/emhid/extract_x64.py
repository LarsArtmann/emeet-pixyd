#!/usr/bin/env python3
"""Extract EMHidCmdV2Head command heads from the EMEET STUDIO Windows x64 binary.

In the x64 build the EMHidCmdV2Head globals in .data are zeroed in the file
image; the CRT static initializers construct them at load time. Each
initializer thunk passes the four head bytes to the ctor as register
immediates:

    sub  rsp,0x38
    mov  r9b,<cat>                     ; or `xor r9d,r9d` for cat 0
    mov  BYTE PTR [rsp+0x20],<cmd>     ; ctor reads it back at [rsp+0x28]
    mov  r8b,<iface>                   ; or `xor r8d,r8d` for iface 0
    lea  rcx,[rip+...]   # <head slot in .data>
    mov  dl,0x9                        ; report prefix
    call <EMHidCmdV2Head ctor>
    lea  rcx,[rip+...]   # <callback / registration target>
    jmp  <common registration tail>

This script disassembles .text once, joins every ctor call site with its
immediates, deduplicates by head tuple, and cross-verifies the result against
cmdtable.json (the 2.0.3 macOS extraction).

Output: x64_heads.json = { "9,dev,cat,id": { "name": ...|null, "slots": [...],
"callbacks": [...] } }. Heads of (9,0,0,0) are zero-initializer artifacts and
are excluded (counted in the summary instead).

Usage:
  python3 extract_x64.py <EMEET-STUDIO-2.exe> [--asm text.asm] [--ctor 0x140179370]
                          [--cmdtable cmdtable.json] [--out x64_heads.json]

--asm reuses an existing `objdump -d -M intel -j .text` dump instead of
re-disassembling (the full dump is ~160 MB; regeneration takes a few minutes).

Verified against EMEET_STUDIO_V2.0.0-Beta.25_cn_Win.exe: 109 unique heads,
108 matching the 2.0.3 Mac table byte-for-byte by exact tuple match
(2026-09-19: an earlier "version-shift" reading — 2.0.3 inserting
SET_REBOOT/GET_MOTOR_SPEED and shifting IDs — is retracted; the matches need
no shift, see docs/hid-protocol-official-map.md).
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

CTOR_DEFAULT = "0x140179370"  # EMHidCmdV2Head ctor, Beta.25 x64; verify per build

# Instruction patterns inside one initializer thunk.
RE_SLOT = re.compile(r"lea\s+rcx,\[rip\+0x[0-9a-f]+\]\s+#\s+(0x[0-9a-f]+)")
RE_CALLBACK = RE_SLOT
RE_REPORT = re.compile(r"\bmov\s+dl,(0x[0-9a-f]+)")
RE_IFACE = re.compile(r"\bmov\s+r8b,(0x[0-9a-f]+)")
RE_IFACE_ZERO = re.compile(r"\bxor\s+r8d,r8d")
RE_CAT = re.compile(r"\bmov\s+r9b,(0x[0-9a-f]+)")
RE_CAT_ZERO = re.compile(r"\bxor\s+r9d,r9d")
RE_CMD = re.compile(r"mov\s+BYTE PTR \[rsp\+0x[0-9a-f]+\],(0x[0-9a-f]+)")

# Thunk boundaries: ctor calls sit between int3 padding, so a backwards scan
# that stops at int3/ret never crosses into a neighboring function.
RE_BOUNDARY = re.compile(r"\b(int3|ret)\b")

WINDOW = 12  # max instructions examined before a ctor call


def disassemble(binary: str) -> str:
    print(
        f"disassembling .text of {binary} (objdump, a few minutes)...", file=sys.stderr
    )
    return subprocess.run(
        ["objdump", "-d", "-M", "intel", "-j", ".text", binary],
        capture_output=True,
        text=True,
        check=True,
    ).stdout


def parse_instruction(line: str) -> tuple[str, str] | None:
    # "  140001094:\t41 b1 01 \tmov    r9b,0x1" -> ("mov", "r9b,0x1")
    m = re.match(r"\s*[0-9a-f]+:\s+(?:[0-9a-f]{2} )+\s*([a-z0-9]+)\s*(.*)$", line)
    if not m:
        return None
    return m.group(1), m.group(2).strip()


def load_instructions(disasm: str) -> list[tuple[str, str]]:
    instructions: list[tuple[str, str]] = []
    for raw in disasm.splitlines():
        if parsed := parse_instruction(raw):
            instructions.append(parsed)
    return instructions


def ctor_sites(instructions: list[tuple[str, str]], ctor: str) -> list[int]:
    target = f"0x{ctor}"
    return [
        i
        for i, (mnemonic, operands) in enumerate(instructions)
        if mnemonic == "call" and operands == target
    ]


def sweep(
    instructions: list[tuple[str, str]], ctors: list[int]
) -> tuple[dict[str, dict], int]:
    heads: dict[str, dict] = {}
    partial = 0

    for site in ctors:
        window = instructions[max(0, site - WINDOW) : site]
        slot = report = iface = cat = cmd = None
        iface_zero = cat_zero = False

        for mnemonic, operands in window:
            text = f"{mnemonic} {operands}"
            if m := RE_SLOT.search(text):
                slot = m.group(1)
            if m := RE_REPORT.search(text):
                report = int(m.group(1), 16)
            if m := RE_IFACE.search(text):
                iface = int(m.group(1), 16)
            if RE_IFACE_ZERO.search(text):
                iface_zero = True
            if m := RE_CAT.search(text):
                cat = int(m.group(1), 16)
            if RE_CAT_ZERO.search(text):
                cat_zero = True
            if m := RE_CMD.search(text):
                cmd = int(m.group(1), 16)

        if iface is None and iface_zero:
            iface = 0
        if cat is None and cat_zero:
            cat = 0

        if None in (slot, report, iface, cat, cmd) or report != 9:
            partial += 1
            continue

        key = f"9,{iface},{cat},{cmd}"
        entry = heads.setdefault(key, {"name": None, "slots": [], "callbacks": []})
        if slot not in entry["slots"]:
            entry["slots"].append(slot)

    return heads, partial


def attach_callbacks(
    instructions: list[tuple[str, str]], ctors: list[int], heads: dict[str, dict]
) -> None:
    """Fill each head's callback list from the lea AFTER each ctor call."""
    slot_to_key: dict[str, str] = {}
    for key, entry in heads.items():
        for slot in entry["slots"]:
            slot_to_key.setdefault(slot, key)

    for site in ctors:
        slot = None
        for mnemonic, operands in instructions[max(0, site - WINDOW) : site]:
            if m := RE_SLOT.search(f"{mnemonic} {operands}"):
                slot = m.group(1)  # last lea before the call is the slot

        if slot is None:
            continue

        for mnemonic, operands in instructions[site + 1 : site + 6]:
            if m := RE_CALLBACK.search(f"{mnemonic} {operands}"):
                callback = m.group(1)
                key = slot_to_key.get(slot)
                if key and callback not in heads[key]["callbacks"]:
                    heads[key]["callbacks"].append(callback)
                break
            if RE_BOUNDARY.search(mnemonic):
                break


def cross_verify(heads: dict[str, dict], cmdtable: dict[str, list[int]]) -> dict:
    table_by_tuple = {",".join(map(str, v)): name for name, v in cmdtable.items()}

    matched = artifact = 0
    for key, entry in heads.items():
        if key == "9,0,0,0":
            artifact += 1
            continue
        if name := table_by_tuple.get(key):
            entry["name"] = name
            matched += 1

    x64_only = sorted(
        k for k, e in heads.items() if k != "9,0,0,0" and e["name"] is None
    )
    mac_only = sorted(k for k in table_by_tuple if k not in heads)

    return {
        "matched": matched,
        "artifact_zero_heads": artifact,
        "x64_only": x64_only,
        "mac_only": mac_only,
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("binary", help="EMEET STUDIO 2.exe (Windows x64 build)")
    parser.add_argument("--asm", help="reuse an existing objdump -d -M intel dump")
    parser.add_argument(
        "--ctor",
        default=CTOR_DEFAULT,
        help=f"EMHidCmdV2Head ctor address (default {CTOR_DEFAULT})",
    )
    parser.add_argument(
        "--cmdtable", default=str(Path(__file__).parent / "cmdtable.json")
    )
    parser.add_argument("--out", default=str(Path(__file__).parent / "x64_heads.json"))
    args = parser.parse_args()

    ctor = args.ctor.removeprefix("0x").lower()
    disasm = Path(args.asm).read_text() if args.asm else disassemble(args.binary)

    instructions = load_instructions(disasm)
    ctors = ctor_sites(instructions, ctor)

    heads, partial = sweep(instructions, ctors)
    attach_callbacks(instructions, ctors, heads)

    cmdtable = json.loads(Path(args.cmdtable).read_text())
    stats = cross_verify(heads, cmdtable)

    clean = {k: v for k, v in sorted(heads.items()) if k != "9,0,0,0"}
    Path(args.out).write_text(json.dumps(clean, indent=1) + "\n")

    print(
        f"unique heads: {len(clean)} (+{stats['artifact_zero_heads']} zero-head artifact)"
        f" | incomplete thunks skipped: {partial}"
    )
    print(
        f"cmdtable match: {stats['matched']}/{len(clean)}"
        f" | x64-only: {len(stats['x64_only'])}"
        f" | mac-only: {len(stats['mac_only'])}"
    )
    if stats["x64_only"]:
        print(f"x64-only heads: {', '.join(stats['x64_only'])}")
    print(f"wrote {args.out}")


if __name__ == "__main__":
    main()
