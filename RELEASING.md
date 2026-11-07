# RELEASING.md — Release Runbook

How to cut a release of emeet-pixyd. The human act is exactly one step: **cut the
annotated tag** (auto-tagging is deliberately deleted). Everything mechanical
after the tag push is automated by `.github/workflows/release.yml` and
`.goreleaser.yaml`.

The pipeline was adopted 2026-11-07 (first release through it: v0.6.0). Source
plan: `docs/planning/2026-11-07_06-48_SUPERB-release-pipeline-hardening.md`.

---

## 1. Version scheme — three shapes, one contract

| Build source          | `--version` renders                    | Set by                                                        |
| --------------------- | -------------------------------------- | ------------------------------------------------------------- |
| GoReleaser release    | `v0.6.0` (the tag)                     | `-X main.buildVersion=v{{ .Version }}` in `.goreleaser.yaml`  |
| `nix build` / NixOS   | git commit SHA (`<rev>`, `-dirty` aware) | `flake.nix` `version = self.rev or self.dirtyRev or "dev"`  |
| `go build` / `go run` | `dev`                                  | `main.go` `var buildVersion = "dev"` default                  |

**The update checker's contract** (`update.go`, doc comment at the top):

- Release binaries (`vX.Y.Z`) are compared **numerically** (semver cores) against
  the latest GitHub release tag. The `v`/`V` prefix is stripped before parsing.
- Nix builds (bare git SHA) are resolved via the **GitHub compare API** — only a
  commit strictly behind the release tag counts.
- `dev` builds can never be compared and **never touch the network**.
- Off switch: `EMEET_PIXYD_UPDATE_CHECK=off` / `hardware.emeet-pixy.updateCheck = false`.

The `v` prefix in the ldflags template is LOAD-BEARING for the numeric channel —
never change `-X main.buildVersion=v{{ .Version }}` to a bare `{{ .Version }}`.

Surfaces that render the version: `emeet-pixyd --version`, the `version` socket
command (appends update-available info), `/api/status` `version` field, Waybar
tooltip, web panel footer.

## 2. Pre-tag checklist (all must be green)

1. **CHANGELOG curated** — `[Unreleased]` content moved into
   `[X.Y.Z] - YYYY-MM-DD`, empty `[Unreleased]` placeholder left behind, compare
   link refs at the bottom flipped.
2. **CI green at HEAD on master** — `gh run list --branch master --limit 5`;
   Go tests, Nix, Website build (incl. `lint-workflows`), and Release `validate`
   all green. Red master forbids tagging.
3. **Local snapshot build passes**:
   ```bash
   GOWORK=off goreleaser release --snapshot --clean --skip=publish --skip=sign
   ```
   Gotcha: `--snapshot --skip=publish` does **NOT** skip signing (cosign runs
   locally only when installed; in CI the validate job must pass `--skip=sign`
   explicitly because cosign is only installed in the release job).
4. **`GOWORK=off goreleaser check`** — config lint.
5. Optional but recommended: `GOWORK=off go test -race -count=1 ./...` locally
   (CI's `verify` job re-runs it at the tag).

## 3. Cut the release

```bash
# 1. Commit the CHANGELOG curation
git add CHANGELOG.md && git commit -m "docs: cut v0.6.0 changelog"

# 2. Annotated tag — headline features, not every commit
git tag -a v0.6.0 -m "Release v0.6.0 — <tagline>

Key changes:
- <headline 1>
- <headline 2>
- <headline 3>"

# 3. Verify the tag points where you think
git tag --points-at HEAD
git show v0.6.0:go.mod | head -3

# 4. Push master first, then the tag
git push origin master
git push origin v0.6.0
```

Tag message convention (matches v0.4.0/v0.5.0): first line
`Release vX.Y.Z — <one-line identity>`, blank line, `Key changes:` + a handful
of user-facing bullets.

A tag push triggers, in order:

1. **`verify`** — `go vet` + `go test -race -count=1` at the tagged commit.
2. **`release`** — GoReleaser builds linux amd64+arm64, archives (`README` +
   `LICENSE`), `checksums.txt`, keyless cosign bundle, creates the GitHub
   Release with commit-derived notes; then `attest-build-provenance` attests
   `dist/checksums.txt`.
3. **`smoke`** — downloads the just-published amd64 tarball, `sha256sum -c`
   verifies it, runs `--version`, asserts it equals the tag. (A smoke failure is
   a red X *after* a completed publish — benign, fix-forward.)
4. **`Nix`** — nix.yml also triggers on `v*` tags, proving the flake builds at
   the exact release commit.

## 4. Post-tag verification

```bash
gh run watch <release-run-id>   # or gh run list --workflow release.yml
gh release view v0.6.0          # assets: 2 tarballs, checksums.txt, .bundle.json, provenance
```

End-to-end proof (the `smoke` job automates the first half; do the cosign part
once per release until the smoke step also covers it):

```bash
cd "$(mktemp -d)"
base="https://github.com/LarsArtmann/emeet-pixyd/releases/download/v0.6.0"
curl -fsSL -O "${base}/checksums.txt" -O "${base}/emeet-pixyd_linux_amd64.tar.gz"
sha256sum -c checksums.txt --ignore-missing
tar -xzf emeet-pixyd_linux_amd64.tar.gz && ./emeet-pixyd --version   # want: emeet-pixyd v0.6.0

nix shell nixpkgs#cosign -c cosign verify-blob checksums.txt \
  --bundle checksums.txt.bundle.json \
  --certificate-identity-regexp '^https://github\.com/LarsArtmann/emeet-pixyd/\.github/workflows/release\.yml@refs/tags/' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'
```

Archives are version-less (`emeet-pixyd_linux_amd64.tar.gz`), so stable
`/releases/latest/download/…` URLs always work.

## 5. Mutability rules

- **Tags are immutable.** Once pushed (and once the Go module proxy notices —
  irrelevant for a binary-only repo, but the discipline holds), a tag is never
  moved, re-cut, or deleted. A bad release is **fix-forward**: cut vX.Y.Z+1.
- The **GitHub Release object** (notes text, draft state) is not the tag and MAY
  be edited/redrafted after the fact.
- Never re-point an existing tag at a different commit to "fix" artifacts.

## 6. Prerelease (rc) channel

`release.prerelease: auto` in `.goreleaser.yaml`: any tag carrying a pre-release
suffix (`v0.7.0-rc.1`, `-beta.2`) is automatically marked **pre-release** on
GitHub and is excluded from the "Latest" designation — update checker and
`/releases/latest` URLs keep pointing at the last stable.

rc cut process: same as §3 but tag `vX.Y.Z-rc.N`. Verify the release page shows
the pre-release badge. When the rc is blessed, cut the final `vX.Y.Z` tag (rc
tags are never mutated into finals).

## 7. Gotchas (each one bit a real session)

- **`GOWORK=off` for every Go/goreleaser command** — a parent `go.work` exists
  that does not include this project.
- **Snapshot ≠ publish-skip for signing** — `--skip=publish` alone still runs
  the cosign step; the validate job passes `--skip=sign` too.
- **Linux-only by construction** — `GOOS=darwin go build ./...` fails on the
  Linux-only error-family wrappers (`undefined: Wrapf` in `internal/pixy`), the
  uevent netlink listener, and hidraw. Linux amd64 + arm64 are the shipped
  targets; there is intentionally no darwin build.
- **Single-tag assumption** — GoReleaser runs on tag pushes only and reads
  `GORELEASER_CURRENT_TAG` from that single tag; do not add multi-tag
  (changelog-per-module) setups here.
- **`GOWORK` env is set at the workflow level** in release.yml; don't remove it
  to "simplify" — CI would inherit the runner's (absent) workspace state.
- **Auto-commit daemon** may sweep intermediate states into `chore:` commits
  while you work; the release build gate (go build + vet) holds broken trees,
  but nix/website breakage only surfaces at their own gates.

## 8. Supply-chain posture

- **Cosign keyless** (sigstore, GitHub OIDC) signs `checksums.txt`; the bundle
  lands as `checksums.txt.bundle.json` next to it. Verify per §4 — the identity
  regexp is pinned to THIS workflow path at tag refs.
- **Provenance**: `actions/attest-build-provenance` with
  `subject-checksums: ./dist/checksums.txt` — the attestation covers
  checksums.txt, which transitively covers every tarball (each archive's sha256
  is a line in the attested file). No per-artifact attestations needed.
- **GoReleaser version: floating `~> v2` (deliberate).** The `validate` job on
  every master push/PR continuously proves the config against the latest v2
  release (drift guard), while dependabot's github-actions ecosystem does NOT
  cover the `version:` input — an exact pin would rot silently until a
  tag-time failure. Re-decide only if the validate job ever proves insufficient.
- SBOM: deliberately deferred (trigger + rationale in ROADMAP).

## 9. After the release

- Read the auto-generated release notes; if `ci:`/`build:` noise slipped through,
  tune `changelog.filters` in `.goreleaser.yaml` (fix-forward, affects the next
  release only).
- Re-read `.goreleaser.yaml` against the real output once — prune anything the
  first three real releases prove unused (calendar note: after v0.8.0).
- The `smoke` job green is the consumability proof — no manual download needed
  unless you want the cosign check of §4.
