# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

This is a Linux-only Go daemon built with Nix. The toolchain (Go ≥ 1.27.1, golangci-lint, templ) is pinned in `flake.nix` — enter the dev shell first:

```bash
nix develop            # provides go 1.27, golangci-lint, templ; installs the pre-commit lint hook
templ generate         # required after editing templates.templ (generated files are committed)
```

Build, test, and lint (GOWORK=off is required — a parent directory's go.work does not include this project):

```bash
nix build                                     # build the daemon
GOWORK=off go test -race -count=1 ./...       # full test suite (CI runs this)
GOWORK=off golangci-lint run --timeout 2m ./...  # lint (CI runs this; 0 issues is the bar)
```

Commits touching `.go`/`.templ` files are gated by the pre-commit hook, which runs whole-module golangci-lint.

## Reporting Issues

Please use GitHub Issues to report bugs and request features. For device-detection problems, include the output of `emeet-pixy device` and `lsusb | grep 328f` (PIXY `328f:00c0`, PIXY 2K `328f:0118`).
