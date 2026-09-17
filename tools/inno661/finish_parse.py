#!/usr/bin/env python3
"""Drive the full setup-0 parse and dump parsed.json (files + locations with
SHA256 digests, icons, runs, uninstall runs, header facts).

Usage: finish_parse.py SETUP0.bin [parsed.json]
"""
import json
import sys

import setup0_parse as sp


def main() -> None:
    bin_path = sys.argv[1]
    out_path = sys.argv[2] if len(sys.argv) > 2 else "parsed.json"
    r = sp.Reader(open(bin_path, "rb").read())
    hdr, counts, langs = sp.parse_prefix(r)
    files = sp.parse_files(r, counts["File"])
    icons, runs, uruns, imgs = sp.parse_rest(r, counts)
    locs = sp.parse_locations(r, counts["FileLocation"])
    print(f"filelocations end at {r.pos} of {len(r.blob)} bytes")
    print(f"header: AppId={hdr['AppId']!r} languages={langs}")
    print(f"counts: {json.dumps(counts)}")
    for kind, lst in [("run", runs), ("uninstallrun", uruns)]:
        for i, (s, _t) in enumerate(lst):
            print(f"{kind}[{i}]: {s[0]!r} params={s[1]!r} verb={s[5]!r}")
    json.dump({
        "header": {"AppName": hdr["AppName"], "AppId": hdr["AppId"],
                   "AppVersion": hdr["AppVersion"], "DefaultDirName": hdr["DefaultDirName"],
                   "counts": counts, "languages": langs},
        "files": files, "locs": locs,
        "icons": [s for s, _t in icons], "runs": [s for s, _t in runs],
        "uruns": [s for s, _t in uruns], "wizard_images": imgs,
    }, open(out_path, "w"), indent=1)
    print(f"saved {out_path}")


if __name__ == "__main__":
    main()
