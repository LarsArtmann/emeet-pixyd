# emeet-pixyd → Standalone Project Extraction

**Date:** 2026-04-30 20:04
**Status:** COMPLETE

---

## Summary

Extracted `emeet-pixyd` from `SystemNix/pkgs/emeet-pixyd/` into its own dedicated project at `/home/lars/projects/emeet-pixyd` with its own Git repository, Nix flake, NixOS module, and CI pipeline.

---

## A) FULLY DONE

| Item | Details |
|------|---------|
| Source code migration | All 30+ Go source files copied, module path updated from `github.com/larsartmann/systemnix/emeet-pixyd` → `github.com/LarsArtmann/emeet-pixyd` |
| Go build verification | `go build ./...` and `go vet ./...` pass |
| Standalone flake.nix | `nix build` produces working binary with `emeet-pixyd` + `emeet-pixy` symlink |
| Standalone package.nix | `buildGoModule` derivation with src filter |
| NixOS module | `modules/nixos.nix` — udev rules, systemd user service, hardware.emeet-pixy options |
| Overlay | `overlays.default` exposes `emeet-pixyd` package |
| GitHub repository | `LarsArtmann/emeet-pixyd` — pushed and live |
| SystemNix flake input | SSH-based `git+ssh://git@github.com/LarsArtmann/emeet-pixyd?ref=master` with `nixpkgs.follows` |
| SystemNix overlay swap | `emeetPixyOverlay` delegates to `emeet-pixyd.overlays.default` |
| SystemNix NixOS module | Loaded via `inputs.emeet-pixyd.nixosModules.default` |
| Old source removal | `pkgs/emeet-pixyd/` (34 files), `pkgs/emeet-pixyd.nix`, `platforms/nixos/hardware/emeet-pixy.nix` deleted |
| CI — Go tests | `.github/workflows/go-test.yml` in emeet-pixyd |
| CI — Nix build | `.github/workflows/nix.yml` with DeterminateSystems installer + magic-nix-cache |
| CI — SystemNix | Removed emeet-pixyd from `go-test.yml` (now runs in its own repo) |
| README fixes | Corrected `StateDir` (`/run/emeet-pixyd`), `WebAddr` (`127.0.0.1:8090`), `SocketPath` (`control.sock`); replaced `just` commands with `nix` workflow |
| templates_templ.go | Force-tracked in git (required for `buildGoModule` — does not run `templ generate`) |
| .envrc | Added for direnv + nix-direnv |
| NixOS build verified | `nix build .#nixosConfigurations.evo-x2` passes end-to-end |
| Both repos pushed | `LarsArtmann/emeet-pixyd` and `LarsArtmann/SystemNix` both pushed to GitHub |

---

## B) PARTIALLY DONE

| Item | Status | Next Step |
|------|--------|-----------|
| None | — | — |

---

## C) NOT STARTED (Future Improvements)

| # | Item | Effort | Impact | Notes |
|---|------|--------|--------|-------|
| 1 | Type model: use `enums` package for `CameraState`/`AudioMode` with `String()`/`Parse()` codegen | Medium | Medium | Current string-based enums work fine but could use `go-enum` or similar for less boilerplate |
| 2 | Config from env/flags via `koanf` or `envconfig` | Medium | Low | Current `DefaultConfig()` is simple and sufficient for a single-deployment daemon |
| 3 | Integration test in CI (needs hardware) | High | Medium | Integration tests require the actual webcam — keep local-only for now |
| 4 | Add `nix flake check` to emeet-pixyd CI | Low | Low | Already runs `nix flake check --no-build` in nix.yml |
| 5 | goreleaser for GitHub releases | Medium | Medium | Automate versioned releases with cross-compilation |
| 6 | Add `docs/` folder from SystemNix (status reports, roadmap) | Low | Low | Already copied; may want to prune/archive |
| 7 | Separate `internal/pixy` into its own Go module | Medium | Low | Would allow reuse but current monorepo is simpler |
| 8 | Add OpenAPI spec for web UI endpoints | Medium | Low | Web UI is HTMX-based, not a public API |
| 9 | Structured logging with `slog` handlers (JSON for production) | Low | Medium | Already uses `slog`; just needs JSON handler config |
| 10 | Health check endpoint for k8s/monitoring | Low | Medium | Already has Prometheus metrics; `/healthz` would be trivial |

---

## D) TOTALLY FUCKED UP

Nothing fucked up. Both repos build, push, and integrate correctly.

**Close call:** `templates_templ.go` was initially gitignored (both locally and globally in `~/.config/git/ignore`). The Nix build would have failed on clean clones since `buildGoModule` doesn't run `templ generate`. Fixed by removing `*_templ.go` from `.gitignore` and force-tracking the file.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture Observations

1. **Type model is clean but manual** — `CameraState` and `AudioMode` are string-based enums with hand-written `Valid()`, `String()`, `Parse*()` methods. A code generator like `github.com/abice/go-enum` would eliminate the boilerplate, but the current approach is explicit and easy to understand.

2. **State management is mutex-heavy** — `Daemon` has `sync.RWMutex` + `sync.Mutex` + nested lock structs. Consider extracting state into a `atomic.Value` or `sync/atomic`-based store for the hot path (call detection polling).

3. **Config is not reloadable** — Config is set at startup. A SIGHUP handler exists but only saves state, doesn't reload config. Low priority since the daemon runs as a systemd service with declarative config.

4. **Well-established libs already in use** — `templ`, `prometheus/client_golang`, `go-systemd/v22` are all appropriate choices. No obvious missing dependencies.

5. **The `process.go` call detection** scans `/proc/*/fd` which is Linux-specific but correct. Could use `inotify` on `/dev/video*` for event-driven detection instead of polling.

---

## F) Top 25 Things to Get Done Next

Sorted by **impact × effort** (high-to-low):

| # | Task | Effort | Impact | Priority |
|---|------|--------|--------|----------|
| 1 | Deploy the new SystemNix config to evo-x2 (`nh os switch`) | 5min | Critical | P0 |
| 2 | Verify emeet-pixyd service starts correctly after deploy | 5min | Critical | P0 |
| 3 | Test webcam waybar integration post-deploy | 5min | High | P0 |
| 4 | Tag v0.2.0 release on emeet-pixyd GitHub | 2min | Medium | P1 |
| 5 | Add goreleaser for versioned GitHub releases | 30min | Medium | P2 |
| 6 | Add `golangci-lint` to CI workflow (already have `.golangci.yml`) | 5min | Medium | P2 |
| 7 | Configure GitHub branch protection (require CI pass) | 5min | Medium | P2 |
| 8 | Update SystemNix AGENTS.md with new architecture info | 10min | Medium | P2 |
| 9 | Add JSON structured logging for production | 15min | Medium | P3 |
| 10 | Add `/healthz` endpoint | 5min | Medium | P3 |
| 11 | Extract `CameraState`/`AudioMode` enums with codegen | 30min | Low | P3 |
| 12 | Replace polling-based call detection with inotify | 1hr | Medium | P4 |
| 13 | Add snapshot endpoint tests | 15min | Low | P4 |
| 14 | Add V4L2 PTZ control tests (mock) | 30min | Low | P4 |
| 15 | Add web UI E2E tests | 1hr | Low | P5 |
| 16 | Document HID protocol reverse-engineering notes | 30min | Low | P5 |
| 17 | Add GRPC/CLI client library for programmatic control | 2hr | Low | P6 |
| 18 | Support multiple PIXY devices simultaneously | 2hr | Low | P6 |
| 19 | Add Airplay/NDI output detection (not just V4L2) | 3hr | Low | P7 |
| 20 | Package for nixpkgs upstream submission | 2hr | Low | P7 |
| 21 | Add macOS support (CoreMedia/IOKit instead of V4L2/hidraw) | 1week | Low | P8 |
| 22 | Add WebSocket streaming for live camera preview | 4hr | Low | P8 |
| 23 | Container image (Docker/Nixpacks) for non-NixOS users | 2hr | Low | P9 |
| 24 | Home Assistant integration | 4hr | Low | P9 |
| 25 | Multi-language SDK (Python, TypeScript) | 1week | Low | P10 |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we deploy this to evo-x2 now?** The migration is complete and verified in `nix build`, but I cannot run `nh os switch` or verify the actual webcam hardware behavior. The emeet-pixyd service, udev rules, and waybar integration need real hardware testing to confirm nothing broke during the module path change.

---

## Migration Stats

- **Lines removed from SystemNix:** 12,133
- **Lines added to emeet-pixyd:** ~800 (new flake, module, CI, README updates)
- **Files deleted from SystemNix:** 34
- **New files in emeet-pixyd:** 6 (flake.nix, package.nix, modules/nixos.nix, .envrc, .github/workflows/nix.yml, .gitignore)
- **Build time impact:** None — same binary cache hits via `nixpkgs.follows`
