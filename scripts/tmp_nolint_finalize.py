#!/usr/bin/env python3
"""Finalize the nolint sweep: delete //x placeholders, restore the 9
directives the line-mapper wrongly dropped (issues fire >1 line away:
func bodies / var blocks)."""
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

# file -> {lineno: original directive comment to restore (None = delete line)}
RESTORE = {
    "power.go": {96: "//nolint:wrapcheck // pixy parse errors are already domain-wrapped",
                 106: "//nolint:wrapcheck // pixy parse errors are already domain-wrapped"},
    "motor.go": {153: "//nolint:wrapcheck // pixy parse errors are already domain-wrapped"},
    "identity.go": {63: "//nolint:wrapcheck // pixy parse errors are already domain-wrapped",
                    76: "//nolint:wrapcheck // pixy parse errors are already domain-wrapped",
                    89: "//nolint:wrapcheck // pixy parse errors are already domain-wrapped"},
    "hid.go": {145: "//nolint:nonamedreturns // named err is required for the deferred close-error aggregation"},
    "internal/pixy/pixy.go": {432: "//nolint:gochecknoglobals,mnd // hardware constants, never mutated at runtime"},
    "internal/pixy/v2head.go": {71: "//nolint:gochecknoglobals // protocol constants; [4]byte cannot be Go consts"},
}

restored = deleted = 0
for rel, fixes in RESTORE.items():
    path = ROOT / rel
    lines = path.read_text().splitlines(keepends=True)
    for lineno, directive in fixes.items():
        line = lines[lineno - 1]
        assert "//x" in line, f"{rel}:{lineno} has no placeholder: {line!r}"
        code = line.split("//x")[0]
        if code.strip():
            lines[lineno - 1] = f"{code}{directive}\n"
        else:
            lines[lineno - 1] = f"{directive}\n"
        restored += 1
    path.write_text("".join(lines))

for path in ROOT.rglob("*.go"):
    if "_templ.go" in path.name or ".git" in path.parts:
        continue
    text = path.read_text()
    if "//x" not in text:
        continue
    out = []
    for line in text.splitlines(keepends=True):
        if line.strip() == "//x\n" or line.strip() == "//x":
            deleted += 1
            continue
        if "//x" in line:
            line = line.split("//x")[0].rstrip() + "\n"
            deleted += 1
        out.append(line)
    path.write_text("".join(out))

print(f"restored {restored} directives, deleted {deleted} placeholders")
