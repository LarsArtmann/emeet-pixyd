# ADR: Do Not Adopt kardianos/service

**Date:** 2026-06-08
**Status:** Rejected
**Context:** Evaluate whether to adopt `github.com/kardianos/service` for daemon lifecycle management.

## Decision

Do not adopt kardianos/service. The current `go-systemd/v22` + NixOS module approach is strictly superior.

## What kardianos/service Provides vs What emeet-pixyd Already Has

| Capability             | kardianos/service              | emeet-pixyd current                                                                           |
| ---------------------- | ------------------------------ | --------------------------------------------------------------------------------------------- |
| `sd_notify READY=1`    | Not supported                  | `go-systemd/v22` — already wired                                                              |
| `sd_notify WATCHDOG=1` | Not supported                  | Every 2s tick in eventLoop                                                                    |
| `sd_notify STOPPING=1` | Not supported                  | In `handleShutdown`                                                                           |
| `Type=notify` service  | Generates `Type=simple`        | NixOS module: `Type = "notify"`                                                               |
| `WatchdogSec=30`       | Not supported                  | NixOS module configures it                                                                    |
| Signal handling        | SIGTERM/SIGINT only            | SIGTERM + SIGINT + SIGHUP (state save)                                                        |
| Security hardening     | None (generic unit)            | `ProtectSystem`, `PrivateTmp`, `NoNewPrivileges`, `RestrictAddressFamilies`, `MemoryMax=256M` |
| Service installation   | Writes `.service` file to disk | Declarative NixOS module                                                                      |
| Cross-platform         | Windows, macOS, Linux          | Linux-only (`//go:build linux`) — dead weight                                                 |

## Rationale

### 1. Zero sd_notify support

This is the single most important systemd integration feature emeet-pixyd uses. `Type=notify` + `WatchdogSec=30` + `READY=1`/`STOPPING=1`/`WATCHDOG=1` are all absent from kardianos/service. Adopting it would **downgrade** the systemd integration.

### 2. Cross-platform abstraction for a Linux-only daemon

kardianos/service's value proposition is "write once, run on Windows/macOS/Linux." emeet-pixyd is `//go:build linux` — it probes `/proc`, talks to `/dev/hidraw*`, uses netlink uevents, calls `v4l2-ctl`, `wpctl`, `ffmpeg`. It will never run on Windows or macOS. The abstraction is pure overhead.

### 3. NixOS module is superior to generated unit files

The current NixOS module produces a hardened, properly-configured systemd unit declaratively. kardianos/service generates a generic unit file (no sandboxing, `RestartSec=120`, `WantedBy=multi-user.target` for a _user_ service) and would bypass NixOS's declarative service management entirely.

### 4. The `Run()` abstraction is trivial

kardianos/service's `Run()` calls `Start()`, waits for SIGTERM/SIGINT, calls `Stop()`. emeet-pixyd's `Run()` → `eventLoop()` is a 3-way select (signals, uevents, ticker) with rich lifecycle. Wrapping it in kardianos/service would mean fighting the abstraction — you'd set `optionRunWait` to your own signal handling, which is exactly what already exists.

### 5. Adds a dependency for negative value

It would replace a direct `go-systemd/v22` dependency (which provides sd_notify) with a thicker abstraction that provides less.

## When kardianos/service Would Make Sense

- Cross-platform CLI tools that need to run as services on Windows + Linux + macOS
- Projects that don't use sd_notify, watchdog, or systemd hardening
- Projects without a declarative service module generating the unit file

emeet-pixyd hits none of these.

## Consequences

- Keep `github.com/coreos/go-systemd/v22` for sd_notify
- Keep NixOS module for declarative service configuration
- No new dependency
- No abstraction layer between daemon and systemd
