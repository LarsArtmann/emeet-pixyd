//go:build linux

package main

import (
	"context"
	"log/slog"
	"sync"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type daemonMetrics struct {
	inCall         metric.Float64Gauge
	autoMode       metric.Float64Gauge
	cameraState    metric.Float64Gauge
	online         metric.Float64Gauge
	breakerState   metric.Float64Gauge
	commands       metric.Int64Counter
	uevents        metric.Int64Counter
	probes         metric.Int64Counter
	hidFailures    metric.Int64Counter
	autoErrors     metric.Int64Counter
	acceptFailures metric.Int64Counter
	streamDuration metric.Float64Histogram
	framesTotal    metric.Int64Counter
}

var (
	metricsInstance   *daemonMetrics //x
	metricsRegistered sync.Once      //x
	// testPromExporter is set during registerMetrics() so tests can collect
	// metric snapshots without adding production-only state to daemonMetrics.
	testPromExporter *prometheus.Exporter //x
)

func registerMetrics() {
	metricsRegistered.Do(func() {
		provider, err := prometheus.New(
			prometheus.WithoutScopeInfo(),
			prometheus.WithoutTargetInfo(),
		)
		if err != nil {
			slog.Error("failed to create OTel Prometheus exporter", "error", err)

			return
		}

		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(provider))
		meter := mp.Meter("emeet-pixyd")
		testPromExporter = provider

		metricsInstance = &daemonMetrics{
			inCall: mustFloat64Gauge(
				meter,
				"emeet_pixyd_in_call",
				"Whether the camera is currently in a call (1=yes, 0=no)",
			),
			autoMode: mustFloat64Gauge(
				meter,
				"emeet_pixyd_auto_mode",
				"Whether auto-management mode is enabled (1=yes, 0=no)",
			),
			cameraState: mustFloat64Gauge(
				meter,
				"emeet_pixyd_camera_state",
				"Desired camera state as a gauge per state label (1=active)",
			),
			online: mustFloat64Gauge(
				meter,
				"emeet_pixyd_online",
				"Whether a video device is currently present (1=online, 0=offline)",
			),
			breakerState: mustFloat64Gauge(
				meter,
				"emeet_pixyd_breaker_state",
				"HID circuit breaker state (1=tripped, 0=closed)",
			),
			commands: mustInt64Counter(meter, "emeet_pixyd_commands_total", "Total number of commands processed"),
			uevents: mustInt64Counter(
				meter,
				"emeet_pixyd_uevents_total",
				"Total number of relevant uevents received",
			),
			probes: mustInt64Counter(meter, "emeet_pixyd_probes_total", "Total number of device probes"),
			hidFailures: mustInt64Counter(
				meter,
				"emeet_pixyd_hid_failures_total",
				"Total number of HID send failures",
			),
			autoErrors: mustInt64Counter(
				meter,
				"emeet_pixyd_auto_errors_total",
				"Total number of failed auto-manage operations (track, audio, source, privacy)",
			),
			acceptFailures: mustInt64Counter(
				meter,
				"emeet_pixyd_socket_accept_failures_total",
				"Total number of CLI socket accept failures (10 consecutive mark the daemon degraded)",
			),
			streamDuration: mustFloat64Histogram(
				meter,
				"emeet_pixyd_stream_duration_seconds",
				"Duration of MJPEG stream sessions in seconds",
				"s",
			),
			framesTotal: mustInt64Counter(
				meter,
				"emeet_pixyd_frames_total",
				"Total number of JPEG frames served via MJPEG stream",
			),
		}
	})
}

//x
func mustFloat64Gauge(meter metric.Meter, name, desc string) metric.Float64Gauge {
	g, err := meter.Float64Gauge(name, metric.WithDescription(desc))
	if err != nil {
		slog.Error("failed to create gauge", "name", name, "error", err)
	}

	return g
}

//x
func mustInt64Counter(meter metric.Meter, name, desc string) metric.Int64Counter {
	c, err := meter.Int64Counter(name, metric.WithDescription(desc))
	if err != nil {
		slog.Error("failed to create counter", "name", name, "error", err)
	}

	return c
}

//x
func mustFloat64Histogram(meter metric.Meter, name, desc, unit string) metric.Float64Histogram {
	h, err := meter.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit(unit))
	if err != nil {
		slog.Error("failed to create histogram", "name", name, "error", err)
	}

	return h
}

func recordHIDFailure(ctx context.Context) {
	if metricsInstance == nil {
		return
	}

	metricsInstance.hidFailures.Add(ctx, 1)
}

// recordBreakerState mirrors the HID circuit breaker onto its gauge so the
// tripped/closed state is scrapeable without inferring from failure counts.
func recordBreakerState(tripped bool) {
	if metricsInstance == nil {
		return
	}

	value := 0.0
	if tripped {
		value = 1
	}

	metricsInstance.breakerState.Record(context.Background(), value)
}

// recordAutoError counts one failed auto-manage operation by its op label.
func recordAutoError(ctx context.Context, operation string) {
	if metricsInstance == nil {
		return
	}

	metricsInstance.autoErrors.Add(
		ctx, 1,
		metric.WithAttributes(attribute.String("op", operation)),
	)
}

func recordSocketAcceptFailure(ctx context.Context) {
	if metricsInstance == nil {
		return
	}

	metricsInstance.acceptFailures.Add(ctx, 1)
}

func recordProbe() {
	if metricsInstance == nil {
		return
	}

	metricsInstance.probes.Add(context.Background(), 1)
}

func recordStreamDuration(ctx context.Context, seconds float64) {
	if metricsInstance == nil {
		return
	}

	metricsInstance.streamDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("source", "mjpeg")))
}

func recordFrame(ctx context.Context) {
	if metricsInstance == nil {
		return
	}

	metricsInstance.framesTotal.Add(ctx, 1)
}

func recordUevent(action, subsystem string) {
	if metricsInstance == nil {
		return
	}

	metricsInstance.uevents.Add(
		context.Background(), 1,
		metric.WithAttributes(
			attribute.String("action", action),
			attribute.String("subsystem", subsystem),
		),
	)
}

// updateMetrics records the current state gauges. online is the runtime
// connectivity snapshot: it feeds the dedicated online gauge, while
// camera_state reports the persisted desired mode (which is never offline).
// breakerTripped derives the breaker gauge from the authoritative strike
// count (hidFailCount >= threshold); the trip path also records immediately
// so a scrape right after the third failure never sees the closed state.
func updateMetrics(state pixy.State, online bool, breakerTripped bool) {
	if metricsInstance == nil {
		return
	}

	ctx := context.Background()
	if state.InCall {
		metricsInstance.inCall.Record(ctx, 1)
	} else {
		metricsInstance.inCall.Record(ctx, 0)
	}

	if state.AutoMode.IsOff() {
		metricsInstance.autoMode.Record(ctx, 0)
	} else {
		metricsInstance.autoMode.Record(ctx, 1)
	}

	if online {
		metricsInstance.online.Record(ctx, 1)
	} else {
		metricsInstance.online.Record(ctx, 0)
	}

	if breakerTripped {
		metricsInstance.breakerState.Record(ctx, 1)
	} else {
		metricsInstance.breakerState.Record(ctx, 0)
	}

	for _, s := range []pixy.CameraState{pixy.StatePrivacy, pixy.StateTracking, pixy.StateIdle} {
		stateAttr := metric.WithAttributes(attribute.String("state", string(s)))
		if state.Camera == s {
			metricsInstance.cameraState.Record(ctx, 1, stateAttr)
		} else {
			metricsInstance.cameraState.Record(ctx, 0, stateAttr)
		}
	}
}

func recordCommandMetric(ctx context.Context, cmd string, result CommandResult) {
	if metricsInstance == nil {
		return
	}

	resultStr := "success"
	if result.IsError() {
		resultStr = "error"
	}

	metricsInstance.commands.Add(
		ctx, 1,
		metric.WithAttributes(
			attribute.String("command", cmd),
			attribute.String("result", resultStr),
		),
	)
}
