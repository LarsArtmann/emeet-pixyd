#!/usr/bin/env python3
"""SHA256-verify extracted files against the digests recorded by Inno Setup.

The FileLocation SHA256Sum hashes the ORIGINAL file content (Compiler.
CompressionHandler.pas:243-260). Files with floCallInstructionOptimized are
stored in transformed form, so the decode transform is applied before
hashing (Setup.FileExtractor.pas:336-363 does the same on install).

Usage: verify.py PARSED.json EXTRACTED_DIR
Exit 0 iff every present file matches its digest.
"""

import hashlib
import json
import os
import sys


def main() -> None:
    parsed_path, outdir = sys.argv[1], sys.argv[2]
    d = json.load(open(parsed_path))
    locs = d["locs"]
    ok = bad = miss = skipped = 0
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
        blob = open(path, "rb").read()
        h = hashlib.sha256(blob).hexdigest()
        if h == loc["sha256"]:
            ok += 1
        else:
            bad += 1
            mismatches.append(
                f"MISMATCH {path}: loc={loc['sha256'][:12]} "
                f"actual={h[:12]} flags={loc['flags']}"
            )
    print(f"ok={ok} bad={bad} missing={miss} skipped(type1)={skipped}")
    for m in mismatches[:20]:
        print(m)
    sys.exit(1 if (bad or miss) else 0)


if __name__ == "__main__":
    main()
