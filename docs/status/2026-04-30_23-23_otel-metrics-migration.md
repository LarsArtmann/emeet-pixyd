# Status Report — 2026-04-30 23:23

## Session Goal

Migrate from `prometheus/client_golang` direct usage to `go.opentelemetry.io/otel/exporters/prometheus` (unified OpenTelemetry observability).

---

## A) FULLY DONE

### Metrics Migration to OpenTelemetry

**Commits:** `6f22af7`, `9db2b3d`, `971240c`

| What                          | Detail                                                                                                                                            |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `handlers.go` imports         | Replaced `prometheus` + `promhttp` with OTel `exporters/prometheus`, `metric`, `sdk/metric`, `attribute`. Kept `promhttp` for `/metrics` handler. |
| `handlers.go` metric vars     | `prometheus.NewGauge` / `NewGaugeVec` → `metric.Float64Gauge` instruments created via OTel `MeterProvider`                                        |
| `handlers.go` registerMetrics | Creates `prometheus.Exporter` (OTel), `sdkmetric.NewMeterProvider`, registers 3 gauges via `meter.Float64Gauge()`                                 |
| `handlers.go` updateMetrics   | `.Set(v)` → `.Record(ctx, v)` with `metric.WithAttributes()` for labels                                                                           |
| `handlers_test.go` assertions | Replaced `testutil.ToFloat64()` with `requireGaugeValue()` using `promExporter.Collect()` + `metricdata.Gauge[float64]`                           |
| `handlers_test.go` matchAttrs | `attribute.Set.Value(Key)` for clean attribute matching                                                                                           |
| `go.mod`                      | Added `go.opentelemetry.io/otel` v1.43.0, `exporters/prometheus` v0.65.0, `sdk/metric` v1.43.0, `metric` v1.43.0 + transitive deps                |
| Lint                          | 0 issues (golines line-length fixed with `stateAttr` extraction)                                                                                  |
| Tests                         | All pass with `-race -count=1`                                                                                                                    |
| AGENTS.md                     | Updated file responsibilities, testing section, gotchas                                                                                           |

### Test Refactoring (unstaged, not yet committed)

| File                         | Change                                                                                                                                                     |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `integration_test.go`        | Extracted `daemonHasDevices()`, consolidated 3 PTZ test functions into 1 table-driven `TestSocket_PanTiltZoom`, removed `assertSocketCommandsHavePrefix`   |
| `internal/pixy/pixy_test.go` | Replaced `assertValid(any)` with generic `runValidTests[T]()` — type-safe, no type switch                                                                  |
| `main_test.go`               | Merged `TestProbeDevices_SetsStateToOfflineWhenNoVideo` + `TestProbeDevices_RecoversFromOffline` into single table-driven `TestProbeDevices_ProbeBehavior` |

---

## B) PARTIALLY DONE

### Unstaged test refactoring

The 3 test files above have refactoring changes that **build and pass tests** but have **2 lint issues**:

- `gocritic`: `else if` suggestion in `integration_test.go:853`
- `gofumpt`: formatting in `internal/pixy/pixy_test.go`

These need fixing before commit.

---

## C) NOT STARTED

| #   | Item                                                                                                                      |
| --- | ------------------------------------------------------------------------------------------------------------------------- |
| 1   | Push commits to remote                                                                                                    |
| 2   | `.golangci.yml` — remove `prometheus_client` suppression rules (if any were added)                                        |
| 3   | Remove `prometheus/client_golang` from go.mod entirely (requires replacing `promhttp.Handler()` with OTel-native handler) |
| 4   | OTel tracing integration (not requested, but natural next step)                                                           |
| 5   | OTel resource attributes (service.name, service.version) for the Prometheus exporter                                      |

---

## D) TOTALLY FUCKED UP

| Issue                                                     | What happened                                                                                             | Root cause                                                                              | Resolution                                                         |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| LSP reverted `handlers.go` edits 3x                       | Gopls auto-save reverted OTel imports to old prometheus imports because `go.mod` didn't have the deps yet | Editing source before ensuring `go.mod` was updated                                     | Fixed by running `go get` for OTel deps FIRST, then editing source |
| `matchAttrs` used `Value.Equal()`                         | `attribute.Value` has no `Equal()` method                                                                 | Assumed API without checking                                                            | Fixed with `AsString()` comparison                                 |
| `requireGaugeValue("emeet_pixyd_in_call after reset", 0)` | Metric name doesn't match — `"after reset"` appended to name                                              | Carried over label-style naming from old `requireMetric`                                | Fixed to use exact metric name                                     |
| `updateMetrics` nil panic in tests                        | `metricInCall` is nil until `registerMetrics()` runs                                                      | Old prometheus gauges initialized at package level; OTel instruments need explicit init | Added `registerMetrics()` call in `TestUpdateMetrics`              |

---

## E) WHAT WE SHOULD IMPROVE

### Process

1. **Always ensure go.mod has deps before editing imports** — LSP reverts broken imports
2. **Run lint immediately after file changes** — catch golines issues before committing
3. **Verify edit persistence after LSP restart** — check `head -30` after edits

### Architecture

4. **`promhttp.Handler()` still pulls in `prometheus/client_golang`** — could use OTel exporter's `ServeHTTP` directly if available, or a custom handler
5. **`registerMetrics()` uses `sync.Once` + `slog.Error`** — errors in metric creation silently degrade. Consider failing fast in NewDaemon
6. **`context.Background()` in `updateMetrics()`** — should accept ctx parameter for consistency
7. **Metric names are duplicated** — string literals in both `registerMetrics()` and tests. Could be constants

### Testing

8. **`requireGaugeValue` calls `registerMetrics()` every invocation** — the `sync.Once` makes this harmless but redundant. Call once in test setup
9. **Test coverage for `/metrics` endpoint** — no integration test verifies the actual Prometheus text output
10. **`matchAttrs` only checks string equality** — works for current use case but not generic. Consider `AsInterface()` comparison

---

## F) Top 25 Next Actions

### High Priority (do next session)

| #   | Action                                                                                              | Effort | Impact |
| --- | --------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Fix 2 lint issues in unstaged test files, commit, push                                              | Low    | High   |
| 2   | Push all commits to remote                                                                          | Low    | High   |
| 3   | Add integration test for `/metrics` endpoint — verify OTel metrics appear in Prometheus text format | Medium | High   |
| 4   | Extract metric name constants (`metricNameInCall`, etc.) to avoid duplication                       | Low    | Medium |
| 5   | Pass `context.Context` to `updateMetrics()` instead of `context.Background()`                       | Low    | Medium |

### Medium Priority (next sprint)

| #   | Action                                                                         | Effort | Impact |
| --- | ------------------------------------------------------------------------------ | ------ | ------ |
| 6   | Remove `prometheus/client_golang` direct dep — replace `promhttp.Handler()`    | Medium | Medium |
| 7   | Add OTel resource attributes (service.name=emeet-pixyd, version)               | Low    | Medium |
| 8   | Make `registerMetrics()` return error instead of `slog.Error` + silent failure | Low    | Medium |
| 9   | `flake.nix` — verify OTel deps don't bloat closure size                        | Low    | Medium |
| 10  | Add `go.opentelemetry.io/otel` to `.golangci.yml` depguard allowlist if needed | Low    | Low    |
| 11  | Table-driven test for `actionToast()`                                          | Low    | Low    |
| 12  | Fuzz test for `matchAttrs`                                                     | Low    | Low    |
| 13  | Add Prometheus metric for device online/offline state                          | Low    | Medium |
| 14  | Add Prometheus histogram for HID command latency                               | Medium | High   |
| 15  | Add Prometheus counter for call detection events                               | Low    | Medium |

### Lower Priority (backlog)

| #   | Action                                                   | Effort | Impact |
| --- | -------------------------------------------------------- | ------ | ------ |
| 16  | OTel tracing for HID commands                            | Medium | High   |
| 17  | OTel tracing for auto-manage cycle                       | Medium | Medium |
| 18  | structured logging correlation with trace IDs            | Medium | Medium |
| 19  | Export metrics via OTLP in addition to Prometheus        | Medium | Medium |
| 20  | Health check endpoint (`/healthz`)                       | Low    | Medium |
| 21  | Readiness endpoint (`/readyz`) — checks device connected | Low    | Medium |
| 22  | Graceful OTel SDK shutdown on daemon stop                | Low    | Medium |
| 23  | Config option for metrics endpoint disable               | Low    | Low    |
| 24  | Grafana dashboard JSON for PIXY metrics                  | Medium | Medium |
| 25  | Alert rules for device offline, HID failures             | Medium | Medium |

---

## G) Top #1 Question

**How should we handle `promhttp.Handler()`?** The OTel Prometheus exporter registers on the default Prometheus registry, so `promhttp.Handler()` works. But this keeps `prometheus/client_golang` as a direct dependency. Options:

1. **Keep it** — simple, works, `promhttp` is lightweight
2. **Use `promhttp.HandlerFor(prometheus.DefaultGatherer)`** — same behavior, same dep
3. **Build a custom handler** using `promExporter.Collect()` + `text.WriteToTextfile` — removes `prometheus/client_golang` entirely but more code

I recommend **option 1** (keep it) — the `promhttp` import is minimal and the OTel exporter is designed to work with it. Removing it adds complexity for no functional gain.

---

## Build & Test Status

```
Tests:  PASS (all, -race -count=1)
Lint:   0 issues (committed code), 2 issues (unstaged test refactoring)
Build:  PASS
Deps:   OTel v1.43.0 / exporters/prometheus v0.65.0 added, godebug removed
```

## Git Log (this session)

```
971240c docs: update AGENTS.md with OTel metrics migration notes
9db2b3d refactor: migrate metrics tests to OTel SDK and fix lint
6f22af7 refactor: migrate Prometheus metrics to OpenTelemetry SDK with Prometheus exporter
```

## Unstaged Changes

```
modified:   integration_test.go      (test consolidation, daemonHasDevices extraction)
modified:   internal/pixy/pixy_test.go (generic runValidTests[T])
modified:   main_test.go             (merged probe tests into table-driven)
```
