#!/usr/bin/env python3
"""Empirical nolint liveness check (T16 / #181 chunk 2).

Strips every non-erraudit //nolint directive to a placeholder, runs
golangci-lint once, maps fired issues back to the original directives,
and restores only the LIVE ones. Stale ones stay stripped.
"""
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
NOLINT = re.compile(r"//\s*nolint(:[a-zA-Z0-9_,\s]+)?\s*(//.*)?$")
ISSUE = re.compile(r"^(.+?):(\d+):\d+: (\w+):")

records = []  # (file, lineno, comment, stripped_line, original_line)


def process_file(path: Path) -> list:
    lines = path.read_text().splitlines(keepends=True)
    recs = []
    changed = False
    for i, line in enumerate(lines):
        if "nolint" not in line:
            continue
        if "erraudit" in line:
            continue
        m = NOLINT.search(line)
        if not m:
            continue
        comment = m.group(0)
        stripped = line[: m.start()] + "//x\n"
        if stripped == line:
            continue
        recs.append((i + 1, comment, stripped, line))
        lines[i] = stripped
        changed = True
    if changed:
        path.write_text("".join(lines))
    return recs


def main() -> None:
    go_files = [
        p
        for p in ROOT.rglob("*.go")
        if "_templ.go" not in p.name and ".git" not in p.parts
    ]
    for f in go_files:
        for lineno, comment, stripped, original in process_file(f):
            records.append((f, lineno, comment, stripped, original))

    print(f"stripped {len(records)} directives")

    result = subprocess.run(
        ["golangci-lint", "run", "--timeout", "5m", "./..."],
        capture_output=True,
        text=True,
        cwd=ROOT,
        env={
            "PATH": "/nix/store/wnjaiawf798z57if1yf59i73abakkxwk-golangci-lint-2.14.0/bin:/nix/store/pn86sk63a2lpmwx16vj4s5871slkyzfw-go-1.27.1/bin:/usr/bin:/bin",
            "GOWORK": "off",
            "HOME": str(Path.home()),
            "GOTMPDIR": str(Path.home() / ".cache/gotmp"),
            "TMPDIR": str(Path.home() / ".cache/gotmp"),
        },
    )
    issues = []
    for line in (result.stdout + result.stderr).splitlines():
        m = ISSUE.match(line)
        if m:
            issues.append((ROOT / m.group(1), int(m.group(2)), m.group(3)))

    print(f"lint fired {len(issues)} issues on stripped tree")

    live_keys = set()
    for ifile, iline, ilinter in issues:
        for f, lineno, comment, _, _ in records:
            if f == ifile and linter_in(comment, ilinter) and lineno in (iline, iline - 1):
                live_keys.add((f, lineno))

    restored, dropped = 0, 0
    by_file: dict[Path, dict[int, str]] = {}
    for f, lineno, comment, stripped, original in records:
        if (f, lineno) in live_keys:
            by_file.setdefault(f, {})[lineno] = original
            restored += 1
        else:
            by_file.setdefault(f, {})[lineno] = stripped
            dropped += 1

    for f, replacements in by_file.items():
        lines = f.read_text().splitlines(keepends=True)
        for lineno, newline in replacements.items():
            lines[lineno - 1] = newline
        f.write_text("".join(lines))

    print(f"restored {restored} LIVE, dropped {dropped} STALE")
    for f, lineno, comment, _, _ in records:
        if (f, lineno) not in live_keys:
            print(f"STALE {f.relative_to(ROOT)}:{lineno}: {comment.strip()}")
    for ifile, iline, ilinter in issues:
        if not any(
            f == ifile and lineno in (iline, iline - 1) and linter_in(c, ilinter)
            for f, lineno, c, _, _ in records
        ):
            print(f"UNMATCHED-ISSUE {ifile.relative_to(ROOT)}:{iline}: {ilinter}")


def linter_in(comment: str, linter: str) -> bool:
    m = re.match(r"//\s*nolint:([a-zA-Z0-9_,\s]+)", comment)
    if not m:
        return True  # bare //nolint suppresses everything
    names = [n.strip() for n in m.group(1).split(",")]
    return "all" in names or linter in names


if __name__ == "__main__":
    main()
