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

//nolint:gochecknoglobals
var (
	promExporter      *prometheus.Exporter
	metricInCall      metric.Float64Gauge
	metricAutoMode    metric.Float64Gauge
	metricCameraState metric.Float64Gauge
	metricsRegistered sync.Once
)

func registerMetrics() {
	metricsRegistered.Do(func() {
		var err error
		promExporter, err = prometheus.New(
			prometheus.WithoutScopeInfo(),
			prometheus.WithoutTargetInfo(),
		)
		if err != nil {
			slog.Error("failed to create OTel Prometheus exporter", "error", err)
			return
		}
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(promExporter))
		meter := mp.Meter("emeet-pixyd")
		if metricInCall, err = meter.Float64Gauge(
			"emeet_pixyd_in_call",
			metric.WithDescription("Whether the camera is currently in a call (1=yes, 0=no)"),
		); err != nil {
			slog.Error("failed to create in_call gauge", "error", err)
		}
		if metricAutoMode, err = meter.Float64Gauge(
			"emeet_pixyd_auto_mode",
			metric.WithDescription("Whether auto-management mode is enabled (1=yes, 0=no)"),
		); err != nil {
			slog.Error("failed to create auto_mode gauge", "error", err)
		}
		if metricCameraState, err = meter.Float64Gauge(
			"emeet_pixyd_camera_state",
			metric.WithDescription("Current camera state as a gauge per state label (1=active)"),
		); err != nil {
			slog.Error("failed to create camera_state gauge", "error", err)
		}
	})
}

func updateMetrics(state pixy.State) {
	registerMetrics()
	ctx := context.Background()
	if state.InCall {
		metricInCall.Record(ctx, 1)
	} else {
		metricInCall.Record(ctx, 0)
	}
	if state.AutoMode.IsOff() {
		metricAutoMode.Record(ctx, 0)
	} else {
		metricAutoMode.Record(ctx, 1)
	}
	for _, s := range []pixy.CameraState{pixy.StatePrivacy, pixy.StateTracking, pixy.StateIdle} {
		stateAttr := metric.WithAttributes(attribute.String("state", string(s)))
		if state.Camera == s {
			metricCameraState.Record(ctx, 1, stateAttr)
		} else {
			metricCameraState.Record(ctx, 0, stateAttr)
		}
	}
}
