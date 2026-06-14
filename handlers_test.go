//go:build linux

package main

import (
	"context"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func requireGaugeValue(t *testing.T, name string, want float64, attrs ...attribute.KeyValue) {
	t.Helper()
	registerMetrics()

	var rm metricdata.ResourceMetrics

	err := collectMetrics(context.Background(), &rm)
	if err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			gauge, ok := m.Data.(metricdata.Gauge[float64])
			if !ok {
				t.Fatalf("%s: not a float64 gauge", name)
			}

			for _, dp := range gauge.DataPoints {
				if matchAttrs(dp.Attributes, attrs) {
					if dp.Value != want {
						t.Errorf("%s = %v, want %v", name, dp.Value, want)
					}

					return
				}
			}
		}
	}

	t.Errorf("gauge %s with attrs %v not found", name, attrs)
}

func matchAttrs(set attribute.Set, wanted []attribute.KeyValue) bool {
	if len(wanted) == 0 {
		return set.Len() == 0
	}

	for _, w := range wanted {
		v, ok := set.Value(w.Key)
		if !ok || v.AsString() != w.Value.AsString() {
			return false
		}
	}

	return true
}

//nolint:paralleltest
func TestUpdateMetrics(t *testing.T) {
	registerMetrics()

	state := pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioNC,
		InCall:   true,
		AutoMode: pixy.AutoOff,
	}

	updateMetrics(state)

	requireGaugeValue(t, "emeet_pixyd_in_call", 1)
	requireGaugeValue(t, "emeet_pixyd_auto_mode", 0)

	for _, s := range []pixy.CameraState{pixy.StatePrivacy, pixy.StateTracking, pixy.StateIdle} {
		want := 0.0
		if state.Camera == s {
			want = 1.0
		}

		requireGaugeValue(t, "emeet_pixyd_camera_state", want, attribute.String("state", string(s)))
	}

	updateMetrics(pixy.State{
		Camera:   pixy.StatePrivacy,
		InCall:   false,
		AutoMode: pixy.AutoFull,
	})

	requireGaugeValue(t, "emeet_pixyd_in_call", 0)
	requireGaugeValue(t, "emeet_pixyd_auto_mode", 1)
}
