#!/usr/bin/env python3
"""SHA256-verify extracted files against the digests recorded by Inno Setup.

Compares each extracted file against BOTH digest candidates:
  - the File entry digest (content hash of the source file)
  - the FileLocation entry digest (chunk-content hash)

Usage: verify.py PARSED.json EXTRACTED_DIR
Exit 0 iff every present file matches at least one digest.
"""
import hashlib
import json
import os
import sys


def main() -> None:
    parsed_path, outdir = sys.argv[1], sys.argv[2]
    d = json.load(open(parsed_path))
    locs = d["locs"]
    ok = miss = bad = skipped = 0
    mismatches = []
    for i, f in enumerate(d["files"]):
        if f["type"] == 1:
            skipped += 1
            continue
        loc = locs[f["loc"]]
        dest = f["dest"].replace("{app}", "").replace("{tmp}", "_tmp")
        path = os.path.join(outdir, dest.lstrip("\\"))
        if not os.path.exists(path):
            miss += 1
            mismatches.append(f"MISSING {path}")
            continue
        h = hashlib.sha256(open(path, "rb").read()).hexdigest()
        if h == f["sha256"] or h == loc["sha256"]:
            ok += 1
        else:
            bad += 1
            mismatches.append(f"MISMATCH {path}: file={f['sha256'][:12]} "
                              f"loc={loc['sha256'][:12]} actual={h[:12]}")
    print(f"ok={ok} bad={bad} missing={miss} skipped(type1)={skipped}")
    for m in mismatches[:20]:
        print(m)
    sys.exit(1 if (bad or miss) else 0)


if __name__ == "__main__":
    main()
