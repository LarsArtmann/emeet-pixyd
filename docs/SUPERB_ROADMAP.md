# emeet-pixyd — Superb Roadmap

**Generated:** 2026-04-20
**Current State:** 63.4% coverage, vet clean, race clean, 0 TODOs, 1.3M+ fuzz executions

---

## Current Metrics

| Metric | Value |
|--------|-------|
| Total lines (source) | ~3,752 (excl. generated) |
| Total lines (tests) | ~2,692 |
| Test-to-source ratio | 0.72:1 |
| Test functions | 120 (incl. 2 fuzz) |
| Coverage (main pkg) | 62.4% |
| Coverage (pixy pkg) | 89.7% |
| Coverage (total) | 63.4% |
| 100%-covered functions | 38 |
| 0%-covered functions | 18 |
| `go vet` | Clean |
| Race detector | Clean |
| Fuzz crashes | 0 / 1.3M execs |

### Source Files

| File | Lines | Role |
|------|-------|------|
| `main.go` | 845 | Daemon lifecycle, state, auto-management |
| `handlers.go` | 575 | HTTP handlers, MJPEG stream, Prometheus metrics |
| `commands.go` | 246 | Command dispatch |
| `hid.go` | 266 | HID I/O, protocol encoding |
| `process.go` | 144 | Process detection, PipeWire, notifications |
| `uevent.go` | 94 | Netlink hotplug parsing |
| `uevent_linux.go` | 31 | Linux netlink socket |
| `v4l2.go` | 84 | V4L2 control |
| `internal/pixy/pixy.go` | 227 | Core types, config, socket client |
| `templates_templ.go` | 720 | Generated HTML templates |

### Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| `a-h/templ` | v0.3.1001 | Type-safe HTML templates |
| `go-systemd/v22` | v22.7.0 | systemd watchdog notification |
| `prometheus/client_golang` | v1.23.2 | Metrics exposition |
| `golang.org/x/sys` | v0.35.0 | Unix system calls |

### Why Coverage is Capped at ~63%

The 18 functions at 0% coverage fall into three categories of hardware dependency:

| Category | Functions | Why Untestable |
|----------|-----------|----------------|
| **HID device** | `sdNotify`, `handleCallStart`, `handleCallEnd`, `autoManage`, `Run`, `main`, `exitWithDaemonError`, `sendCommand` | Require real `/dev/hidraw*` or are entry points |
| **Process/filesystem** | `ppidOf`, `isDescendantOf`, `findPixySource`, `setDefaultSource`, `notify` | Read `/proc`, shell out to `wpctl`/`notify-send` |
| **Kernel interfaces** | `listenUevents`, `unixSocketUevent`, `unixOpenNetlinkKobjectUevent`, `v4l2Set`, `v4l2SetMultiple` | Require netlink sockets, v4l2 devices |

Pushing past 63% requires dependency injection (interfaces) to mock these hardware interactions.

---

## Roadmap

### Phase 1: Dependency Injection for Testability

**Goal:** 80%+ coverage, testable without hardware

The core idea: extract interfaces for hardware interactions so they can be replaced with mocks in tests.

#### 1.1 Extract `Commander` interface for shell commands

Currently `process.go` calls `exec.CommandContext` directly for `wpctl`, `notify-send`, `ffmpeg`, and `v4l2-ctl`. Extract:

```go
type Commander interface {
    Run(ctx context.Context, name string, args ...string) error
    Output(ctx context.Context, name string, args ...string) ([]byte, error)
}
```

**Affected files:** `process.go`, `v4l2.go`, `handlers.go` (ffmpeg)
**Tests unlocked:** `findPixySource`, `setDefaultSource`, `notify`, `v4l2Set`, `handleStream`
**Effort:** Medium (interface + field on Daemon + 2 implementations)
**Coverage impact:** +8-10%

#### 1.2 Extract `HIDDevice` interface for HID I/O

Currently `hid.go` opens `/dev/hidraw*` directly. Extract:

```go
type HIDDevice interface {
    Send(report []byte) error
    SendRecv(ctx context.Context, report []byte) ([]byte, error)
}
```

**Affected files:** `hid.go`, `main.go`
**Tests unlocked:** `setTracking`, `setAudio`, `setGesture`, `centerCamera`, `syncState`, `queryTracking`, `queryAudio`, `queryGesture`, `handleCallStart`, `handleCallEnd`, `autoManage`
**Effort:** Medium-large (touches core Daemon methods)
**Coverage impact:** +15-20%

#### 1.3 Extract `ProcessInspector` interface for /proc traversal

```go
type ProcessInspector interface {
    PIDs() []int
    FDLinks(pid int) []string
    PPIDOf(pid int) int
}
```

**Affected files:** `process.go`
**Tests unlocked:** `ppidOf`, `isDescendantOf`, `isCameraInUse`
**Effort:** Small
**Coverage impact:** +3-5%

#### 1.4 Extract `UeventListener` interface for netlink

```go
type UeventListener interface {
    Listen(ctx context.Context) <-chan struct{}
}
```

**Affected files:** `uevent.go`, `uevent_linux.go`
**Tests unlocked:** `listenUevents`, `unixSocketUevent`, `unixOpenNetlinkKobjectUevent`
**Effort:** Small
**Coverage impact:** +2-3%

---

### Phase 2: Architecture Improvements

#### 2.1 Decompose `Run()` (104 lines → ~40)

`Run()` is the longest function at 104 lines (flagged by `funlen`). It does:
1. Signal handling setup
2. Uevent listener goroutine
3. Web server start
4. Unix socket listener
5. Main poll loop with ticker

Extract:
- `setupSignalHandler() <-chan os.Signal`
- `startUeventListener(ctx) chan struct{}`
- `startWebServer(ctx) error`
- `runPollLoop(ctx, ueventCh, pollTicker)`

**Effort:** Small
**Benefit:** Each piece becomes independently testable

#### 2.2 Eliminate `init()` for Prometheus metrics

The `init()` function in `handlers.go` registers global Prometheus metrics, making tests that exercise metrics non-hermetic. Instead:
- Accept a `prometheus.Registerer` in the web server constructor
- Register metrics during construction, not at init time
- Use `prometheus.NewPedanticRegistry()` in tests

**Effort:** Small
**Benefit:** Hermetic metric tests, no global state pollution

#### 2.3 Centralize `.golangci.yml` configuration

No `.golangci.yml` exists — the project relies on editor/LSP defaults. Create an explicit config that:
- Enables the linters currently reporting (gci, gofumpt, golines, goconst, revive, mnd, wrapcheck, funlen, errcheck, gosec, modernize)
- Disables irrelevant warnings for this project (revive package-comment, nlreturn)
- Sets project-specific thresholds (funlen: 100, mnd: exclude tests)

**Effort:** Trivial
**Benefit:** Consistent linting across CI, editors, `go vet`

---

### Phase 3: Robustness

#### 3.1 Graceful degradation for missing optional dependencies

Currently `handleStream` fails silently if `ffmpeg` is not installed (checked via `exec.LookPath`). Extend this pattern:
- `wpctl` missing → log once, disable audio switching (don't retry every poll)
- `notify-send` missing → log once, disable notifications
- `v4l2-ctl` missing → log once, disable PTZ controls

Cache availability at startup, check once, not per-request.

**Effort:** Small
**Benefit:** Cleaner logs, no repeated failures

#### 3.2 Circuit breaker for HID failures

If the HID device fails repeatedly, `setDeviceState` retries with `probeDevices` every call. Add a simple circuit breaker:
- After N consecutive HID failures, stop trying for a cooldown period
- Log at warning level once (not every failure)
- Reset on successful probe

**Effort:** Medium
**Benefit:** Prevents log spam when device is physically disconnected

#### 3.3 Stream health monitoring

The MJPEG stream (`handleStream`) silently stops if `extractJPEGFrame` fails once. Add:
- Frame counter per stream session (logged on close)
- Stream uptime metric (`emeet_pixyd_stream_duration_seconds`)
- Client disconnect detection (context cancellation)

**Effort:** Small
**Benefit:** Observability into stream reliability

#### 3.4 Structured error types for command responses

Commands return free-form strings like `"error: track: PIXY HID device not available"`. Define typed errors:

```go
type CommandError struct {
    Op   string
    Err  error
}

func (e *CommandError) Error() string { return fmt.Sprintf("%s: %v", e.Op, e.Err) }
```

**Effort:** Small
**Benefit:** Machine-parseable command responses, better error handling for CLI consumers

---

### Phase 4: Observability

#### 4.1 Additional Prometheus metrics

| Metric | Type | Purpose |
|--------|------|---------|
| `emeet_pixyd_stream_duration_seconds` | Histogram | Stream session duration |
| `emeet_pixyd_stream_frames_total` | Counter | Frames served |
| `emeet_pixyd_command_total` | Counter (label: command) | Command frequency |
| `emeet_pixyd_command_errors_total` | Counter (label: command) | Command failure rate |
| `emeet_pixyd_probe_total` | Counter | Device probe attempts |
| `emeet_pixyd_uevent_total` | Counter (label: action, subsystem) | Hotplug events |

**Effort:** Small per metric
**Benefit:** Production health monitoring via SigNoz

#### 4.2 Structured log levels audit

Current log levels are inconsistent:
- `slog.Debug` for errors that should be `slog.Warn` (stream pipe error, stream start error)
- `slog.Error` for expected conditions (device not found during probe)

Audit and standardize:
- `Debug`: internal details only useful during development
- `Info`: state changes (call start/end, device found/lost, config changes)
- `Warn`: recoverable failures (ffmpeg not found, wpctl missing, HID timeout)
- `Error`: unexpected failures (state file corruption, HID write failure)

**Effort:** Small
**Benefit:** Production log noise reduction

#### 4.3 `pprof` endpoint for production profiling

Add `net/http/pprof` under `/debug/pprof/` (behind the existing `securityMiddleware`). Useful for diagnosing goroutine leaks or memory issues in production.

**Effort:** Trivial (3 lines)
**Benefit:** Production debuggability

---

### Phase 5: Web UI

#### 5.1 WebSocket for live state updates

The web UI currently polls `/panel` via HTMX. Replace with:
- WebSocket endpoint at `/ws`
- Push state changes on every `handleCommand` call
- Client reconnects automatically on disconnect

**Effort:** Medium
**Benefit:** Instant UI updates, reduced polling overhead

#### 5.2 Keyboard shortcuts in web UI

Add keyboard shortcuts for common actions:
- `Space` — toggle privacy
- `T` — enable tracking
- `P` — enable privacy
- `A` — cycle audio
- `G` — toggle gesture

**Effort:** Small (HTML/JS only, `templates.templ`)
**Benefit:** Desktop-app-like UX

#### 5.3 Mobile-responsive layout

The current UI works on desktop but is not optimized for mobile. Add responsive breakpoints for the control buttons and PTZ sliders.

**Effort:** Small (CSS only, `static/`)
**Benefit:** Control camera from phone

---

### Phase 6: Testing Infrastructure

#### 6.1 Integration test harness with fake devices

Create a test harness that:
- Creates a fake `/dev/hidraw*` via `os.Pipe()` pair
- Creates a fake `/dev/video*` with a test pattern MJPEG stream
- Sets up a complete Daemon with these fakes
- Runs real HTTP requests against the web server

This would test the full stack: HTTP → command → HID → state → metrics → response.

**Effort:** Large
**Benefit:** End-to-end confidence without hardware

#### 6.2 Continuous fuzz in CI

The fuzz tests (`FuzzExtractJPEGFrame`, `FuzzParseHIDResponse`) are currently run manually. Add a CI job:
- Run each fuzz test for 60 seconds
- Store corpus in the repo under `testdata/fuzz/`
- Fail CI on any crash

**Effort:** Small
**Benefit:** Continuous safety net for byte-parsing code

#### 6.3 Benchmark suite

Add benchmarks for hot paths:
- `BenchmarkExtractJPEGFrame` — streaming performance
- `BenchmarkParseHIDResponse` — called per HID query
- `BenchmarkUpdateMetrics` — called every poll tick
- `BenchmarkHandleCommand` — command dispatch throughput

**Effort:** Small
**Benefit:** Performance regression detection

---

### Phase 7: Code Quality Nits

#### 7.1 Remaining linter suppressions

| Warning | Count | Recommended Action |
|---------|-------|--------------------|
| `nlreturn` (no blank line before return) | ~15 | Fix all — the linter is correct |
| `whitespace` (unnecessary leading newline) | ~10 | Fix all |
| `goconst` (`"idle"` repeated 3x) | 1 | Extract as constant |
| `perfsprint` (Sprintf → concatenation) | 1 | Fix |
| `modernize` (HasPrefix+TrimPrefix → CutPrefix) | 1 | Fix |
| `embeddedstructfieldcheck` (embedded field spacing) | 1 | Fix |

**Effort:** Trivial
**Benefit:** Clean linter output

#### 7.2 Remove `String()` method coverage gap

`CameraState.String()` and `AudioMode.String()` show 0% coverage because they're trivial (`return string(s)`) and never called directly in tests. Add trivial test calls.

**Effort:** Trivial
**Benefit:** 90%+ pixy package coverage

---

## Priority Matrix

| Item | Impact | Effort | Priority |
|------|--------|--------|----------|
| 2.1 Decompose `Run()` | Medium | Small | **P0** |
| 2.3 `.golangci.yml` config | Medium | Trivial | **P0** |
| 7.1 Fix linter warnings | Low | Trivial | **P0** |
| 7.2 `String()` coverage | Trivial | Trivial | **P0** |
| 4.3 pprof endpoint | Medium | Trivial | **P1** |
| 3.1 Graceful degradation | Medium | Small | **P1** |
| 4.2 Log level audit | Medium | Small | **P1** |
| 2.2 Eliminate `init()` | Medium | Small | **P1** |
| 4.1 Additional metrics | Medium | Small | **P1** |
| 3.2 Circuit breaker | Medium | Medium | **P2** |
| 3.4 Structured errors | Low | Small | **P2** |
| 5.2 Keyboard shortcuts | Low | Small | **P2** |
| 6.3 Benchmark suite | Low | Small | **P2** |
| 6.2 CI fuzz | Medium | Small | **P2** |
| 3.3 Stream monitoring | Low | Small | **P3** |
| 5.3 Mobile layout | Low | Small | **P3** |
| 5.1 WebSocket updates | Medium | Medium | **P3** |
| 1.1 Commander interface | High | Medium | **P3** |
| 1.2 HIDDevice interface | High | Medium-large | **P4** |
| 1.3 ProcessInspector | Medium | Small | **P4** |
| 1.4 UeventListener | Low | Small | **P4** |
| 6.1 Integration harness | High | Large | **P4** |

### Recommended Execution Order

1. **Quick wins** (P0): `.golangci.yml`, decompose `Run()`, fix linter nits, `String()` tests
2. **Observability** (P1): pprof, log levels, `init()` removal, additional metrics, graceful degradation
3. **Robustness** (P2): circuit breaker, structured errors, keyboard shortcuts, benchmarks, CI fuzz
4. **Architecture** (P3-P4): DI interfaces, WebSocket, integration harness

---

## What "Superb" Looks Like

| Metric | Current | Target |
|--------|---------|--------|
| Coverage | 63.4% | 80%+ |
| Test functions | 120 | 180+ |
| Linter warnings | ~73 | 0 |
| `go vet` | Clean | Clean |
| Race detector | Clean | Clean |
| Fuzz crashes | 0/1.3M | 0/10M+ |
| Longest function | 104 lines | <80 lines |
| Zero-value init | Yes | No |
| Hardware deps | Direct | Injected |
| CI fuzz | Manual | Automated |
| Benchmarks | None | 4+ |

The project is already **solid** — vet clean, race clean, all pure functions thoroughly tested with fuzz + unit coverage, real bugs fixed, dead code removed. The gap between "solid" and "superb" is primarily the DI refactor (Phase 1) which unlocks testability for the remaining 37% of code.
