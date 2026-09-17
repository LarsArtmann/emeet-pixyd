#!/usr/bin/env python3
"""Inno Setup 6.6.1 setup-0 parser — single source of truth.

Parses the decompressed setup-0 stream (see extract_setup0.py) per the
record layouts in jrsoftware/issrc@is-6_6_1 `Shared.Struct.pas`.

Entry orders:
  stream 1: header prefix (34 UTF-16 + 4 ansi strings + 17 i32 counts +
    20B versions + 63B tail + 6B options) -> Language -> CustomMessage ->
    Permission -> Type -> Component -> Task -> Dir -> ISSigKey -> File ->
    Icon -> Ini -> Registry -> InstallDelete -> UninstallDelete -> Run ->
    UninstallRun -> 4x wizard-image groups -> optional DLLs
  stream 2: FileLocation entries ONLY (89 bytes each).

File entry tail after 15 strings + 1 ansi (77 bytes):
  SHA256[32] + Typ[1] + versions[20] + loc[4] + attribs[4] + extsize[8]
  + perms[2] + opts[5] + type[1]
FileLocation entry (89 bytes):
  first[4] + last[4] + start[8] + suboff[8] + orig[8] + comp[8]
  + SHA256[32] + FILETIME[8] + verms[4] + verls[4] + flags[1]
"""
import struct

STR_NAMES = [
    "AppName", "AppVerName", "AppId", "AppCopyright", "AppPublisher",
    "AppPublisherURL", "AppSupportPhone", "AppSupportURL", "AppUpdatesURL",
    "AppVersion", "DefaultDirName", "DefaultGroupName", "BaseFilename",
    "UninstallFilesDir", "UninstallDisplayName", "UninstallDisplayIcon",
    "AppMutex", "DefaultUserInfoName", "DefaultUserInfoOrg",
    "DefaultUserInfoSerial", "AppReadmeFile", "AppContact", "AppComments",
    "AppModifyPath", "CreateUninstallRegKey", "Uninstallable",
    "CloseApplicationsFilter", "SetupMutex", "ChangesEnvironment",
    "ChangesAssociations", "ArchitecturesAllowed",
    "ArchitecturesInstallIn64BitMode", "CloseApplicationsFilterExcludes",
    "SevenZipLibraryName",
]
COUNT_NAMES = [
    "Language", "CustomMessage", "Permission", "Type", "Component", "Task",
    "Dir", "ISSigKey", "File", "FileLocation", "Icon", "Ini", "Registry",
    "InstallDelete", "UninstallDelete", "Run", "UninstallRun",
]


class Reader:
    def __init__(self, blob: bytes):
        self.blob = blob
        self.pos = 0

    def u8(self) -> int:
        v = self.blob[self.pos]
        self.pos += 1
        return v

    def u16(self) -> int:
        v, = struct.unpack_from("<H", self.blob, self.pos)
        self.pos += 2
        return v

    def i32(self) -> int:
        v, = struct.unpack_from("<i", self.blob, self.pos)
        self.pos += 4
        return v

    def i64(self) -> int:
        v, = struct.unpack_from("<q", self.blob, self.pos)
        self.pos += 8
        return v

    def rdstr(self):
        n, = struct.unpack_from("<I", self.blob, self.pos)
        self.pos += 4
        if n >= 0xFFFFFFF0:
            return None
        raw = self.blob[self.pos:self.pos + n]
        self.pos += n
        return raw.decode("utf-16le", "replace")

    def rdansi(self):
        n, = struct.unpack_from("<I", self.blob, self.pos)
        self.pos += 4
        if n >= 0xFFFFFFF0:
            return None
        raw = self.blob[self.pos:self.pos + n]
        self.pos += n
        return raw

    def raw(self, n: int) -> bytes:
        b = self.blob[self.pos:self.pos + n]
        self.pos += n
        return b

    def rdentry(self, nstr: int, tail: int):
        strs = [self.rdstr() for _ in range(nstr)]
        return strs, self.raw(tail)


def parse_prefix(r: Reader):
    hdr = {}
    for n in STR_NAMES:
        hdr[n] = r.rdstr()
    for n in ["LicenseText", "InfoBeforeText", "InfoAfterText", "CompiledCodeText"]:
        hdr[n] = r.rdansi()
    counts = {}
    for n in COUNT_NAMES:
        counts[n] = r.i32()
    r.raw(20)   # MinVersion + OnlyBelowVersion
    r.raw(56)   # tail: WizardSize(8)+DarkStyle(1)+AlphaFormat(1)+BackColors(8)
                # +DynDark(8)+Opacity(1)+ExtraDiskSpace(8)+SlicesPerDisk(4)
                # +7 enums(7)+DisableDir/GroupPage(2)+UninstallDisplaySize(8)
    hdr["Options"] = int.from_bytes(r.raw(6), "little")
    langs = []
    for _ in range(counts["Language"]):
        name = r.rdstr()
        for _ in range(3):
            r.rdstr()
        for _ in range(4):
            r.rdansi()
        r.u16()
        r.raw(17)  # dfs(4)+sh(4)+sw(4)+wfs(4)+rtl(1)
        langs.append(name)
    for _ in range(counts["CustomMessage"]):
        r.rdstr()
        r.rdstr()
        r.i32()
    for _ in range(counts["Permission"]):
        r.rdansi()
    for _ in range(counts["Type"]):
        for _ in range(4):
            r.rdstr()
        r.raw(20 + 1 + 1 + 8)
    for _ in range(counts["Component"]):
        for _ in range(5):
            r.rdstr()
        r.raw(8 + 4 + 1 + 20 + 1 + 8)
    for _ in range(counts["Task"]):
        for _ in range(6):
            r.rdstr()
        r.raw(4 + 1 + 20 + 1)
    for _ in range(counts["Dir"]):
        for _ in range(7):
            r.rdstr()
        r.raw(4 + 20 + 2 + 1)
    for _ in range(counts["ISSigKey"]):
        for _ in range(3):
            r.rdstr()
    return hdr, counts, langs


def parse_files(r: Reader, count: int):
    files = []
    for _ in range(count):
        strs = [r.rdstr() for _ in range(15)]
        allowed = r.rdansi()
        digest = r.raw(32)
        r.u8()          # version typ
        r.raw(20)       # versions
        loc = r.i32()
        attribs = r.i32()
        extsize = r.i64()
        r.u16()         # perms
        opts = r.raw(5)
        ftype = r.u8()
        files.append({
            "src": strs[0], "dest": strs[1], "loc": loc,
            "sha256": digest.hex(),
            "opts": int.from_bytes(opts, "little"),
            "type": ftype, "extsize": extsize,
        })
    return files


def parse_rest(r: Reader, counts):
    icons = [r.rdentry(13, 48) for _ in range(counts["Icon"])]
    for _ in range(counts["Ini"]):
        r.rdentry(10, 21)   # 20B versions + Options(1)
    for _ in range(counts["Registry"]):
        r.rdentry(9, 29)    # 20B versions + RootKey(4) + Permissions(2) + Typ(1) + Options(2)
    for _ in range(counts["InstallDelete"]):
        r.rdentry(7, 21)    # 20B versions + DeleteType(1)
    for _ in range(counts["UninstallDelete"]):
        r.rdentry(7, 21)
    runs = [r.rdentry(13, 27) for _ in range(counts["Run"])]
    uruns = [r.rdentry(13, 27) for _ in range(counts["UninstallRun"])]
    imgs = []
    for g in range(4):
        cnt = r.i32()
        if cnt == -1:
            imgs.append({"group": g, "count": "same-as-regular"})
            continue
        sizes = []
        for _ in range(cnt):
            n = r.i32()
            r.raw(max(n, 0))
            sizes.append(n)
        imgs.append({"group": g, "count": cnt, "sizes": sizes})
    return icons, runs, uruns, imgs


def parse_locations(r: Reader, count: int):
    locs = []
    for _ in range(count):
        first, last = r.i32(), r.i32()
        start, suboff, orig, comp = r.i64(), r.i64(), r.i64(), r.i64()
        sha = r.raw(32)
        r.raw(8)   # FILETIME
        r.raw(8)   # verms + verls
        flags = r.u8()
        locs.append({
            "first": first, "last": last, "start": start, "sub": suboff,
            "orig": orig, "comp": comp, "sha256": sha.hex(), "flags": flags,
        })
    return locs
