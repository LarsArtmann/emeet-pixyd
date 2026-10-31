# Architecture Review: Service Orientation, Composability, Resilience, Self-Health

**Date:** 2026-10-31 14:17
**Scope:** The `samber/do` + health-checks question, then the four architecture dimensions it really asks about.
**Series:** follows `2026-06-06_22-19_modularity.html` (many of its recommendations — `CommandRunner`, `UeventListener`, DataStar migration, typed `CommandResult` — have since shipped; its "17 open TODOs" list is stale and superseded by `TODO_LIST.md`).
**Method:** source-verified only. Every claim below cites `file:line` from this session's reads.

---

## 0. Executive verdict — the premise is wrong, and that is the correct outcome

> **Q: "Are we using samber/do v2 in combination with Health Checks superbly?!"
> **A: No — we use no samber/do at all, and the project is better for it.**

- `go.mod` carries **zero samber dependencies** (`go.mod:5-18`: templ, go-systemd, go-humanize, go-branded-id, go-error-family, prometheus, datastar, otel, x/sys).
- The rejection is **documented and dated**: `docs/planning/archive/2026-05-01_04-32_quality-sweep-round2.md:83` — *"samber/do: 1 struct, ~6K LOC. DI framework is absurd overkill."*
- The same philosophy rejected `kardianos/service` (ROADMAP: zero sd_notify value for a Linux-only daemon) and decomposing the `Daemon` struct (ROADMAP #86: "~17 fields is manageable for a single-binary hardware daemon; splitting adds indirection without clarity").

This review therefore audits **what the project does instead** against (a) the samber/do DO-1→DO-6 anti-pattern rules — the durable part of that library's best practice — and (b) the four dimensions asked about. The conclusion up front:

> **The architecture is proper for its scope.** It satisfies every DO rule *by construction* (not by container), its health story is genuinely layered (systemd watchdog → liveness endpoint → always-200 status contract → metrics → degradation-by-omission), and its resilience engineering (circuit breaker, privacy-trap countermeasure, reconcile-with-persisted-intent) is beyond what any DI container gives you. Two medium findings (subprocess deadlines, silent listener degradation) and three minor ones are the honest residue.

---

## 1. The DI audit — `Dependencies` struct vs samber/do's DO-1→DO-6

The project's composition root is `NewDaemon` (`main.go:102-157`): it wires a single `Dependencies` value (`deps.go:13-27`) whose fields are **function values and two interfaces** (`CommandRunner`, `UeventListener`). Tests override via functional options; production wiring is the only place real implementations appear.

| DO rule | samber/do smell | emeet-pixyd status | Evidence |
| --- | --- | --- | --- |
| DO-1 | `Must*` panics on runtime paths | **Clean by construction** — no container, no `MustInvoke` anywhere; DI is struct assignment | `main.go:116-130` |
| DO-2 | `do.New()` without `Shutdown()` | **Equivalent guarantee, different shape** — there is no container to leak; every owned resource (socket listener, HTTP server, uevent fd, broadcaster) has an explicit shutdown path in `handleShutdown` / `listenUnix` / `uevent_linux.go` | `main.go:277-297`, `socket.go:48-53` |
| DO-3 | `Override*` outside tests | **Clean** — no override mechanism exists in production; test options are the only mutation path (163 usages across `*_test.go`) | test-option builders |
| DO-4 | package-level injector global | **Clean** — `Dependencies` is a field of `Daemon`, resolved once, passed by ownership; no global | `main.go:87` |
| DO-5 | `Invoke` inside loops | **Clean** — nothing resolves inside loops; deps are captured closures | `deps.go` consumers |
| DO-6 | `Shutdown()` reaching into other services | **Clean** — `handleShutdown` is a flat, ordered teardown (state → socket → HTTP), no cross-service calls | `main.go:277-297` |
| Service-locator smell | injector passed deep into business logic | **Clean** — `Daemon` holds resolved deps; nothing re-resolves | `deps.go` |
| Storing the container in a service | service holds injector | **Partial hit** — see F6: `Daemon` *is* both container and primary service (`d.deps.setTracking = d.setTracking`, `main.go:125-130`, the "circular ref" note) | F6 below |

**What samber/do would have added here:** unified lifecycle interfaces (`do.ShutdownerWithError`, `do.Healthchecker`), named-service plumbing, and scope machinery. **What it would have cost:** a runtime reflection container for a graph with exactly **one node type** (`Daemon`), plus a dependency the Nix FOD must vendor. The 2026-05-01 call was right, and nothing in the current ~30-file layout has outgrown it. Verdict on "superb DI": **yes, relative to need — the graph is trivial and it is handled trivially, which is the superb choice.**

The one place the hand-rolled approach is *weaker* than the container canon: lifecycle and health are **per-component conventions** (each goroutine owner rolls its own shutdown handling) rather than a uniform interface set. At one node, conventions are readable; they would not scale to ten services. That threshold is nowhere in sight (ROADMAP #86 pins the monolith deliberately).

---

## 2. Self-health audit — the layered model

Health here is not one endpoint; it is five deliberate layers, each answering a different question:

| Layer | Question it answers | Mechanism | Evidence |
| --- | --- | --- | --- |
| **1. Supervisor** | "Is the process alive and looping?" | systemd `Type=notify`, `WatchdogSec=30`, `Restart=on-failure`, `RestartSec=3`; `READY=1` after HTTP start, `WATCHDOG=1` every poll tick, `STOPPING=1` on shutdown | `modules/nixos.nix:116-120`, `main.go:222,278,356` |
| **2. Liveness probe** | "Is the *camera* usable?" | `GET /api/health` → 200 `{status,camera,version}` when video node present, **503 when offline** — the monitoring contract (alert on 503, not on connection refused) | `handlers.go:139-154`, `website/.../monitoring.mdx` |
| **3. Daemon-alive contract** | "Is the *daemon* up, regardless of camera?" | `GET /api/status` **always 200 while running** — widgets distinguish connection-refused (daemon down) from `online:false` (camera unplugged) without parsing error bodies | `handlers.go:162-180` |
| **4. Metrics** | "What has it been doing?" | 9 OTel instruments on `/metrics`: `online` gauge, `hid_failures_total`, `probes_total`, `commands_total`, `uevents_total`, `stream_duration_seconds`, `frames_total`, `in_call`, `auto_mode`, `camera_state` | `metrics.go:17-100` |
| **5. Degradation by omission** | "What still works?" | Battery/PTZ lines vanish from status/Waybar when the device can't answer; `checkExternalDeps` warns at startup *with impact per missing binary* ("MJPEG streaming unavailable") | `power.go:24-33`, `main.go:159-174` |

Layer 3 is the quietly excellent one: most local daemons conflate "daemon healthy" with "hardware present," forcing widgets to guess. The intent/connectivity split (ADR `2026-10-01_camera-intent-vs-connectivity`) carries the same discipline into the state model — `State.Camera` is intent-only, `devicePresence{Online,Controllable}` is observed connectivity, and a probe miss *cannot* corrupt user intent (`state_split_brain_test.go` pins it).

**Gaps in the layer model** are F2 and F5 below: the layers observe the *camera*, not the daemon's own components (socket listener, web listener). A daemon that lost its Unix socket still reports `READY`, still pets the watchdog, and still answers `/api/health` 200 — see §5.

---

## 3. Resilience audit

| Mechanism | Design | Evidence |
| --- | --- | --- |
| **HID circuit breaker** | Threshold 3; strikes only on commit failures; a strike below threshold triggers immediate re-probe (yanked device replaced before next command); at threshold commands fail fast with `ErrPIXYNotConnected` | `circuitbreaker.go:18-40`, `main.go:90` |
| **HID response timeout** | 500ms bounded read on every response wait — no unbounded hidraw reads | `hid.go:54,66,213-231` |
| **Privacy-trap countermeasure** | Hardware-evidenced firmware race (mode write eaten + interface poisoned near tilt −85°) countered in software: 6s arm window, 3s-deferred re-assert with idle bounce, belief-check before firing | `privacy_trap.go`, ADR-grade evidence in AGENTS.md |
| **Reconcile-on-appear** | Persisted intent wins over hardware (power cycles reset the camera; privacy must survive replug); fresh install adopts hardware; **every failure path logs and keeps belief — nothing fatal** | `device.go:403-462` |
| **Debounced call detection** | Counter clamps at `DebounceCount`; `auto=off` disables the monitor entirely (test-pinned) | `auto.go:120-146` |
| **Caches with negative results** | PTZ cache, power cache (60s TTL *and* remembered-absence), last-frame cache — sick devices cost at most one query per window | `power.go:24-33`, `cache.go` |
| **SSE fan-out** | Non-blocking sends, slow clients drop events, never stall the daemon | `sse.go:56-70` |
| **Uevent listener** | Transient read errors `continue` (never kill the listener); context-cancellable; fd closed on shutdown | `uevent_linux.go` |
| **Lock contract** | Documented lock-order table (`v4l2Mu → hidMu`, never reversed), pinned by a deadlock-watchdog test and a HID-serialization high-water-mark test | AGENTS.md Concurrency, `TestLockOrder_V4L2MoveWithHIDCommands` |
| **Client-side bounding** | Socket CLI: dial + read/write deadlines; server side: 5s per-connection IO deadlines — a slow daemon cannot hang a CLI caller | `internal/pixy/ipc.go:26-62`, `socket.go:19,90,105` |

The resilience posture is: **degrade narrowly, never lie about state, and let the supervisor be the last resort.** That last clause is load-bearing — it is why the watchdog coupling (F4) is a feature with an invariant to respect, not a bug.

---

## 4. Service orientation & composability

**Service orientation in-the-large does not apply — by design.** This is a single-binary local hardware daemon; the ROADMAP's won't-do list (#85 cmd/ layout, #86 Daemon decomposition) pins that deliberately. Judged in-the-small — interfaces at every hardware/OS boundary, one command core behind multiple transports — it is strong:

- Every external world touch is an interface or injectable function: `CommandRunner` (subprocesses), `UeventListener` (netlink), `HIDDevice` (hidraw), plus 11 function-field deps for V4L2/PipeWire/notify (`deps.go:13-27`).
- One command core (`handleCommand`, `commands.go`) serves **three facades**: Unix socket (CLI/scripts/Waybar), HTTP/DataStar (browser), and the internal auto-manage paths — no logic duplication across transports.
- `internal/pixy` is a clean domain kernel: branded IDs (`PID`, `SourceID`), validated enums with `ValidDesired()` excluding `StateOffline`, no infrastructure imports leaking in.

**Composability** is proven by the test suite's shape: 163 usages of the `with*` DI options, a protocol-faithful hardware simulator (`pixySimulator`) that validates every outgoing byte and tracks transport concurrency, and hardware tests that run the same code paths against the real device under `-tags=integration`. Production code never branches on "am I under test" — the seams are real.

The composability debt is F6: `Daemon` is simultaneously the container (owns `deps`), the state store, and the biggest service. Rewiring any concern means touching the same struct. At ~17 fields + documented lock contract this is manageable and explicitly won't-do'd; it is noted as the first thing to revisit if a second consumer of the domain kernel ever appears.

---

## 5. Findings (severity-ordered)

### F1 — MEDIUM: no per-subprocess deadline on external tools
`realCommandRunner` runs `exec.CommandContext` with the *daemon-lifetime* context on socket paths (request contexts on HTTP, which carry no handler deadline either). A wedged `v4l2-ctl`/`ffmpeg`/`wpctl` (sick USB stack) holds `v4l2Mu`/`hidMu` indefinitely; the command stalls, other PTZ commands queue behind the mutex, and only `WatchdogSec=30` recovers the daemon via restart. Clients are deadline-bounded so nothing hangs *outside* the daemon — the residue is the mutex hold and the restart blast radius.
**Evidence:** `commander.go:24-48` (no timeout wrapper), `socket.go:103` (daemon ctx into `handleCommand`), contrast `hid.go:66` (HID *is* bounded).
**Fix:** per-call `context.WithTimeout` in `v4l2Set`/`parsePTZValues`/stream/notify wrappers (a `withTimeout(ctx, v4l2CmdTimeout)` helper; ~15 lines + tests). The tools are interactive CLI utilities; 3-5s bounds are generous.

### F2 — MEDIUM: listener failures degrade silently
If `listenUnix` fails (state dir uncreatable, socket path occupied) or `ListenAndServe` errors, the failure is **log-only** (`main.go:212-217`, `main.go:265-272`); the daemon still sends `READY=1`, still pets the watchdog, and runs indefinitely without its CLI surface (socket) or web surface. `/api/health` keeps answering 200 with the camera online. Nothing observes the daemon's own components.
**Fix (two-tier):** fail-fast on *startup* socket listen failure (return the error from `Run` → non-zero exit → systemd `Restart=on-failure` does its job), and add a `degraded` dimension to `/api/health`/`/api/status` for runtime losses (web listener died, socket accept loop dead). Keeps layer semantics: supervisor for process health, health endpoint for component health.

### F3 — LOW: `trapReasserts` never drained in production shutdown
The WaitGroup exists so "shutdown (and tests) can wait for their HID writes to finish" (`main.go:60-62`), but only tests call `.Wait()` (`privacy_trap_test.go:66`). `handleShutdown` does not. Cosmetic in production (process death reaps them), yet it was already identified in `docs/status/2026-10-01_05-23_privacy-trap-fixed-hardware-session.md` item 19 — and never harvested into `TODO_LIST.md`. **Fix:** `d.trapReasserts.Wait()` before state save in `handleShutdown` (2 lines) or consciously reclassify as won't-do.

### F4 — LOW: watchdog petting is coupled to the poll tick — document the invariant
`WATCHDOG=1` fires only in the ticker branch (`main.go:354-356`), *after* `autoManage` completes. One loop iteration = poll + `/proc` scan + up to several HID round trips (each with 200ms sleeps) + possible reconcile. Comfortably inside 30s today, but the budget is implicit. **Fix:** a comment on `WatchdogSec` in `modules/nixos.nix` stating the invariant (`WatchdogSec ≥ ~10 × PollInterval + worst-case autoManage`), so a future `POLL_INTERVAL=15s` config doesn't silently starve the watchdog.

### F5 — LOW: `/api/health` has no daemon-component dimension
Deliberate (the monitoring doctrine scopes it to camera presence), but it is exactly why F2 can hide. Solved together with F2's `degraded` field; no separate work.

### F6 — INFO: `Daemon` is container and service at once
`d.deps.setTracking = d.setTracking` (`main.go:125`) — the composition root wires the service into its own dependency graph. Fine at this scale (ROADMAP #86), noted as the growth trigger.

---

## 6. Scores

| Dimension | Score | Justification |
| --- | --- | --- |
| Service orientation | **4.5 / 5** | Interfaces at every boundary, one command core behind three facades; in-the-large N/A by documented design |
| Composability | **4.5 / 5** | 163 test-override usages, protocol-faithful simulator, pure probe; minus F6 |
| Resilience | **4 / 5** | Breaker, trap countermeasure, reconcile, debounce, bounded clients; minus F1 (unbounded subprocess holds) and F2 |
| Self-health | **4 / 5** | Five deliberate layers, intent/connectivity split; minus F2/F5 (daemon components unobserved) |
| DI quality (DO-1→DO-6 lens) | **Pass by construction** | Zero violations; the one structural smell (F6) is a documented won't-do |

## 7. Action roadmap

| # | Action | Priority | Impact | Effort |
| --- | --- | --- | --- | --- |
| 1 | F1: per-subprocess timeouts (`v4l2Set`, `parsePTZValues`, stream, notify) | **P1** | MED | S |
| 2 | F2: fail-fast startup socket-listen failure + `degraded` field in health/status | **P1** | MED | S-M |
| 3 | F3: drain `trapReasserts` in `handleShutdown` (or reclassify won't-do) | P3 | LOW | S |
| 4 | F4: document the `WatchdogSec`/`PollInterval` invariant in `modules/nixos.nix` | P3 | LOW | S |

Harvested into `TODO_LIST.md` as #190-#193 (per docs-health HARVEST — roadmaps do not live in timestamped reports alone).

## 8. Would samber/do improve anything? (the honest tradeoff)

| samber/do feature | Would it help here? |
| --- | --- |
| Composition root + lazy providers | No — the graph is one node; `NewDaemon` already *is* the composition root |
| `do.Shutdowner`/`Healthchecker` interfaces | Marginal — would formalize what F2/F5 surface as gaps; but those gaps are fixable in ~50 lines without a container |
| Named services / scopes | No — zero multi-implementation or request-scope needs |
| `do.Healthchecker` + `do.Health` aggregation | This is the *one* genuinely attractive idea (a uniform health-check registry feeding `/api/health`). Worth stealing **as a pattern** (a `map[string]func() error` walked by the health handler), not as a dependency |
| Audit hooks (`samber-do-auditlog`) | No — OTel metrics + slog already cover registration/invocation observability |

**Bottom line:** adopting samber/do today would add a runtime container, a FOD-vendored dependency, and reflection indirection to a codebase whose DI needs are fully met by a 14-line struct — to buy one pattern that is stealable in an afternoon. The 2026-05-01 rejection remains correct. If F2's `degraded` work ever grows to per-component health checks (socket, web, HID, probe), implement the plain-map registry; revisit the container only if a second long-lived service type appears.

---

*Point-in-time snapshot. When superseded, annotate — do not rewrite. Findings harvested to `TODO_LIST.md` #190-#193.*

---

## Addendum (2026-10-31, session-hygiene pass F092–F097)

### §6 re-scored against the assessment rubric (F094)

The original §6 table scored four review-chosen dimensions. Re-scored against the architecture-review skill's seven-dimension rubric (1–5 each, evidence-cited):

| Rubric dimension | Score | Justification | Delta vs original §6 |
| --- | --- | --- | --- |
| Coupling | **4** | Interfaces at every transport boundary (`HIDDevice`, `UeventListener`, `CommandRunner`, DI function fields); the `Daemon` struct is the one concrete hub all files share — by design for a single-process daemon, but it is concrete-dependence coupling | new |
| Cohesion | **5** | Every file owns one concern (probe/HID/PTZ/stream/metrics each named for it); `internal/pixy` carries only domain types; no utils/common packages | new |
| Modularity | **4** | Boundaries align with domain boundaries and are test-pinnable (lock-contract table, serialization tests); `templates_templ.go` is generated-into-the-package rather than isolated — minor | new |
| Composability | **4.5** | same evidence as original ("163 test-override usages" — grep-sum of option-name occurrences, not distinct call sites; simulator; pure probe); minus F6 | 0 (was 4.5) |
| Scalability | **4** | single-device daemon by design; new features extend via the command registry + DI without restructuring (speed/presets/tracking landed without refactors) | new |
| Service orientation | **4.5** | same evidence as original; in-the-large N/A by documented design | 0 (was 4.5) |
| Dependency direction | **4** | domain (`internal/pixy`) depends on nothing infrastructure; root package depends on domain — correct direction; HID/v4l2 access is behind interfaces the daemon injects | new |

**Average: 4.29 — "Good" band (3.5–4.4), upper edge.** The rubric makes explicit what the original table implied: the architecture's strength is cohesion + service orientation; its ceiling is the concrete `Daemon` hub (coupling/modularity 4s), which is the accepted trade for a single-binary daemon.

### June-review open TODOs — exhaustive audit (F097)

The `2026-06-06_22-19_modularity.html` review left 13 open rows (the "17" figure in this report's header over-counted; audited against `TODO_LIST.md` at `e0db422`, 2026-06-07). One-line verdicts:

- #14 log-levels audit — **absorbed** (leveled slog discipline throughout; no dedicated row survived)
- #20 continuous fuzz in CI — **shipped** (`go-test.yml` fuzz targets + corpus cache + list-vs-CI assert)
- #21 `CommandRunner` interface — **shipped** (`commander.go`/`deps.go`)
- #23 `ProcessInspector` — **retired** (2026-10-01b: plain functions behind `isCameraInUse`)
- #24 `UeventListener` — **shipped** (`uevent.go`)
- #26 mobile-responsive layout — **shipped** (5 `@media` blocks, `static/style.css`)
- #27 WebSocket live updates — **superseded** (DataStar SSE with `openWhenHidden` reconnect; WebSocket deliberately gone)
- #30 camera presets — **shipped** (full preset family + hardware mirroring)
- #31 fake-device test harness — **shipped** (`pixySimulator`, byte-faithful)
- #32 stream/process/hid coverage — **shipped** (fuzz + simulator + timeout tests)
- #34 MJPEG reconnection — **shipped differently** (client retry with `STREAM_MAX_RETRIES` + connection-lost banner)
- #35 real-hardware integration tests — **shipped** (`-tags=integration` suite)
- #42 PTZ readback accuracy — **shipped** (readback scheduling, `ptz.go`)
- (#59 `encoding/json/v2` SKIP — **landed anyway** with Go 1.27; the codebase imports it today)

### Health-layers diagram (F096)

The five-layer model of §2 is now a D2 diagram in the series directory: `2026-10-31_14-17_health-layers.d2` (+ `.svg`), including the F2/F5 gap note and its #191 closure.
