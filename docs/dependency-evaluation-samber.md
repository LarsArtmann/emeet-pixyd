# Dependency Evaluation: samber/lo & samber/ro

**Date:** 2026-05-01\
**Verdict:** Adopt neither.

---

## 1. samber/lo — Lodash-style Go Utilities

### What it is

Generic utility library (500+ functions) for slice/map/string manipulation: `Filter`, `Map`, `Reduce`, `Contains`, `Ternary`, `Coalesce`, `ToPtr`, `Try0`, etc.

### Candidate sites found

| Pattern                        | Sites | `lo` function        |
| ------------------------------ | ----- | -------------------- |
| `boolStr` ternary helper       | 1     | `lo.Ternary`         |
| `ptr[T]` in tests              | 4     | `lo.ToPtr`           |
| `ptzAxisValid` switch          | 1     | `lo.Contains`        |
| `_ = someFunc()`               | 16    | `lo.Try0`            |
| `ConfigFromEnv` env overrides  | 7     | `lo.CoalesceOrEmpty` |
| `isRelevantUevent` membership  | 1     | `lo.Contains`        |
| `isPixyName` OR-chain          | 1     | `lo.SomeBy`          |
| `setGesture` ternary assign    | 1     | `lo.Ternary`         |
| `requestIDMiddleware` coalesce | 1     | `lo.CoalesceOrEmpty` |

### Why each one fails

- **`lo.Ternary` replacing `boolStr`** — `boolStr` is 5 lines, called 4 times. `lo.Ternary` **eagerly evaluates both branches** (classic footgun). Current `if/else` is lazy and idiomatic Go.
- **`lo.ToPtr` replacing `ptr[T]`** — Only used in tests. Adding a production dependency to simplify test helpers is backwards. `ptr` is 1 line.
- **`lo.Contains` replacing `ptzAxisValid`/`isRelevantUevent`** — 3-element membership checks. `lo.Contains([]string{"pan","tilt","zoom"}, axis)` allocates a slice on every call. Switch is zero-alloc.
- **`lo.Try0` replacing `_ = f.Close()`** — Go convention. `lo.Try0` adds a function call and a dependency to express the same intent. Every Go developer reads `_ = f.Close()` instantly.
- **`lo.CoalesceOrEmpty` replacing `ConfigFromEnv`** — The 7 env blocks each have different parsing logic (duration, int, bool, AudioMode). `CoalesceOrEmpty` only works for the string-as-is case. The `lo` version would be _more_ complex.
- **`lo.SomeBy` replacing `isPixyName`** — `strings.Contains` OR-chain is 3 short-circuit evaluations. `lo.SomeBy` allocates a slice, creates a closure, and iterates all 3 unconditionally. Slower.
- **`lo.Ternary` for `setGesture` assign** — `var mark byte = hidByteIdle; if enabled { mark = gestureEnabledByte }` is perfectly clear Go. The `lo.Ternary` version is the same length but with a function call.

### The deeper problem

This codebase is **imperative I/O**, not data transformation:

- **System calls and file I/O** — `/proc` walks, `hidraw` reads, `v4l2-ctl` subprocesses, netlink sockets
- **Early returns and `continue` on errors** — `lo.Filter`/`lo.Map` can't express these; they require pure predicates
- **Go 1.26 iterator sequences** — `strings.SplitSeq`, `strings.FieldsSeq` are already lazy, zero-alloc. `lo` versions regress to slice-allocating.
- **Byte-level HID protocol** — switch-on-constant is the right tool; `lo.SliceToMap` lookup tables would be obscure

### Cost summary

| Factor             | Impact                                                                             |
| ------------------ | ---------------------------------------------------------------------------------- |
| New dependency     | +1 in minimal hardware daemon with 6 direct deps                                   |
| Style divergence   | `lo.Ternary`/`lo.Contains` signal "JS/Python thinking", break Go readability norms |
| Test coverage gain | None — every replacement is 1:1 semantics                                          |
| Performance        | `lo.Contains`, `lo.SomeBy`, `lo.Ternary` allocate where current code doesn't       |
| Eager evaluation   | `lo.Ternary` evaluates both branches — footgun for side-effecting code             |

---

## 2. samber/ro — ReactiveX for Go

### What it is

Go implementation of ReactiveX: Observables, Observers, Subjects, pipeable operators (`Filter`, `Map`, `DebounceTime`, `Retry`, `Catch`, `SwitchMap`, etc.), and a plugin ecosystem (signals, HTTP, WebSocket, process execution).

### The daemon's event model today

```go
// main.go Run() — the entire event loop
for {
    select {
    case sig := <-sigs:        // signal → save state / shutdown
    case <-ueventCh:           // hotplug → re-probe devices
    case <-ticker.C:           // poll → autoManage
    }
}
```

Three event sources. One select. ~40 lines total.

### Mapping to ro

| Event source | ro replacement                 | Problem                                                                                                                                                                                        |
| ------------ | ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ticker.C`   | `ro.Interval`                  | Saves 0 lines over `time.NewTicker`                                                                                                                                                            |
| `ueventCh`   | `ro.FromChannel` + `ro.Filter` | Still need the goroutine reading raw netlink; ro can only wrap the channel                                                                                                                     |
| `sigs`       | `rosignal.NewSignalCatcher()`  | Best fit — saves ~5 lines. Not worth a dependency.                                                                                                                                             |
| **Debounce** | `ro.DebounceTime`              | **Semantic mismatch** — current debounce is count-based (3 consecutive same-direction polls), not time-based. No standard Rx operator models the flip-flop counter pattern in `auto.go:84-98`. |
| **State**    | `BehaviorSubject[pixy.State]`  | Requires rewriting all 15+ state mutation sites from `d.mu.Lock(); d.state.X = v; d.mu.Unlock()` to `subject.Next(newState)`. Full concurrency model change.                                   |

### The deeper problems

**1. Side effects everywhere.** `autoManage` → `handleCallStart` → HID writes, `v4l2-ctl` subprocess, `wpctl` call, `notify-send`. Rx operators are supposed to be pure functions. Putting I/O in `ro.Map` is an Rx anti-pattern; you'd need `ro.Tap` or subscribe-side effects — imperative code wearing a mask.

**2. Error model mismatch.** The daemon's model: "log and continue." A failed HID write triggers a re-probe, a failed notification is silently dropped, a failed state save is logged. Rx's model: **errors terminate the stream.** You'd need `ro.Catch` at every I/O boundary, adding boilerplate instead of removing it.

**3. Split concurrency brain.** The daemon uses `d.mu` + `d.cmdMu` + per-struct locks. Rx wants state to flow through the stream, not be shared. Mixing both creates the worst of both worlds — you still need locks for the 15+ mutation sites, AND you now have subscription lifecycle to manage. You can't half-adopt Rx; you'd need to commit to pushing all state through subjects, which means rewriting `setTracking`, `setAudio`, `setGesture`, `syncState`, `probeDevices`, `handleCallStart`, `handleCallEnd`, `handleAutoCommand`, `loadState` — essentially every method that touches `d.state`.

**4. Shutdown complexity.** Current: `cancel()` + deferred `os.Remove` + `httpSrv.Shutdown`. With ro: unsubscribe all subscriptions, wait for completion, handle the case where a subscription is mid-operator-chain. The current 10-line shutdown becomes a distributed cleanup problem.

**5. Debugging.** When a reactive pipeline breaks, stack traces go through `Pipe` → `Map` → `Filter` → `Subscribe` internals. The current imperative code gives you a direct line number.

**6. The complexity budget.** This is a 7-file hardware daemon with 3 event sources. ro is designed for systems with 10+ event sources, complex merging/splitting, windowing, and backpressure. The daemon doesn't have that problem. The select loop is already the simplest thing that could work.

### The one genuinely interesting idea

`BehaviorSubject[pixy.State]` would elegantly solve the "read current state without locks" problem — the web UI's `getWebStatus` could subscribe once and always have the latest value. But realizing that benefit requires committing the entire concurrency model to the stream paradigm. That's a rewrite, not a refactor, and the current lock-based model is correct and tested.

---

## 3. When would adoption make sense?

### samber/lo

- Codebase with heavy slice/map pipeline transformations: `Filter → Map → Reduce → GroupBy` chains on data collections
- Pure functions without early returns or side effects
- Pre-Go 1.21 codebase (lo provides `min`/`max`/`contains` that are now builtins)
- Heavy JSON/struct mapping layers with many `if v != "" { use v } else { use default }` patterns

### samber/ro

- 10+ event sources requiring complex composition (merge, switch, window, buffer)
- Backpressure management between producers and consumers
- Multicasting to many independent subscribers
- UI frameworks where reactive state propagation is the dominant pattern
- Systems where the event graph is the primary complexity, not I/O

### Neither fits this daemon

The dominant complexity is **hardware I/O and protocol handling** (HID byte-level commands, V4L2 PTZ control, `/proc` filesystem scanning, netlink sockets). These are inherently imperative, side-effecting, and error-resilient. Functional and reactive paradigms add indirection without reducing complexity — and they create new categories of problems (eager evaluation, stream lifecycle, error propagation, debugging opacity) that the current code doesn't have.
