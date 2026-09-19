# emhid — EMEET STUDIO V2Head command-table extractor

Extracts the complete HID command table from the official EMEET STUDIO
macOS binary: **162 command heads** (`cmdtable.json`), each a 4-byte
`[report, device, category, id]` V2Head constructed by static initializers.
This table is the implementation gate for every new HID feature in
emeet-pixyd (PTZ speed, battery, tracking variants, motor presets).

Consumed by: `docs/hid-protocol-official-map.md` §3.5 (annotated view),
`docs/hid-protocol.md` (V2Head section).

## Usage

```sh
# 1. Obtain the EMEET STUDIO macOS installer (.pkg) from emeet.ai and
#    extract the .app bundle (pkg uses xar; `xar -xf file.pkg`).
#    The analyzed build was: EMEET_STUDIO_V2.0.3_intl_mac_20260903_110248.pkg
#    (fat Mach-O, ~241 MB arm64 slice, symbols intact).

# 2. Run the extractor (llvm-objdump + llvm-nm required):
nix shell nixpkgs#llvm --command python3 extract_cmdtable.py /path/to/EMEET\ STUDIO

# 3. Result: ./cmdtable.json  =  { "CMD_...": [b0, b1, b2, b3], ... }
```

Requires ~500 MB scratch (`arm64.bin` carve, deleted at exit) and a few
minutes for the full `--disassemble` pass.

## How it works

1. Carve the arm64 slice out of the fat Mach-O (`0x0100000C` cputype).
2. `llvm-nm` finds `EMHidCmdV2Head::EMHidCmdV2Head(u8,u8,u8,u8)` dynamically.
3. `llvm-objdump --chained-fixups --dyld-info` builds a GOT-slot → `CMD_*_E`
   symbol map.
4. The full disassembly is scanned for `bl <ctor>` sites; each site's four
   `mov w1..w4 #imm` operands (searched backwards ≤30 instructions) plus the
   `ldr x0, [x0, #slot]` GOT load identify one command.

## Caveats

- **Hardcoded GOT base (`0x106a9c000`)**: the script derives the ctor address
  and all immediates dynamically, but the `__DATA_CONST __got` segment base is
  a literal (line with `0x106A9C000 + ldr_slot`). A different binary build may
  relocate it — re-derive via
  `llvm-objdump --macho --chained-fixups <arm64.bin> | grep __got` and update
  the constant before trusting an empty extraction result.
- **Symbol-name collisions in the legacy `0x07` region**: two symbols share
  head `07 06 70 00`; the JSON keeps both (one overwrites the other in
  insertion order). Irrelevant for the V2 `0x09` surface.
- **Windows x86_64 cross-verification is not implemented** (the PE has a
  different initializer shape; tracked as TODO #152).
- The extractor reads **heads only** — payload layouts were derived separately
  from controller call sites and log format strings and are documented in the
  map doc.

## Table excerpt

| Head          | Command                    | Meaning                        |
| ------------- | -------------------------- | ------------------------------ |
| `09 00 00 02` | `CMD_GET_BATTERY_LEVEL`    | battery percent (u8)           |
| `09 00 00 06` | `CMD_GET_CHARGE_STA`       | charge status enum             |
| `09 03 01 03` | `CMD_SET_MOTOR_SPEED`      | `[motorType:u8][speed:f32]`    |
| `09 03 01 13` | `CMD_GET_MOTOR_SPEED`      | value + limit (two f32)        |
| `09 03 01 19` | `CMD_SET_MOTOR_PRESET_POS` | `[slot:u8]`                    |
| `09 04 01 01` | `CMD_SET_TARGET_TRACK`     | `[mode:u8][f32×3]`             |
| `09 01 01 01` | `CMD_SET_DEVICE_MODE`      | **our** tracking config+commit |
