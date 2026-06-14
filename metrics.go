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
	promExporter *prometheus.Exporter

	inCall         metric.Float64Gauge
	autoMode       metric.Float64Gauge
	cameraState    metric.Float64Gauge
	commands       metric.Int64Counter
	uevents        metric.Int64Counter
	probes         metric.Int64Counter
	hidFailures    metric.Int64Counter
	streamDuration metric.Float64Histogram
	framesTotal    metric.Int64Counter
}

var (
	metricsInstance   *daemonMetrics //nolint:gochecknoglobals // lazy init via sync.Once
	metricsRegistered sync.Once      //nolint:gochecknoglobals // lazy init, runs once per process
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

		metricsInstance = &daemonMetrics{
			promExporter: provider,
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
				"Current camera state as a gauge per state label (1=active)",
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

//nolint:ireturn
func mustFloat64Gauge(meter metric.Meter, name, desc string) metric.Float64Gauge {
	g, err := meter.Float64Gauge(name, metric.WithDescription(desc))
	if err != nil {
		slog.Error("failed to create gauge", "name", name, "error", err)
	}

	return g
}

//nolint:ireturn
func mustInt64Counter(meter metric.Meter, name, desc string) metric.Int64Counter {
	c, err := meter.Int64Counter(name, metric.WithDescription(desc))
	if err != nil {
		slog.Error("failed to create counter", "name", name, "error", err)
	}

	return c
}

//nolint:ireturn
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

func updateMetrics(state pixy.State) {
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
