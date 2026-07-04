//go:build linux

package main

import (
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// assertAxisNotInMap fails if the given axis exists in ptzAxes.
// Used to verify the map rejects unknown axis names.
func assertAxisNotInMap(t *testing.T, axis pixy.Axis) {
	t.Helper()

	if _, ok := ptzAxes[axis]; ok {
		t.Errorf("unknown axis %q should not be in ptzAxes", axis)
	}
}

func TestPTZAxisLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		axis pixy.Axis
		want string
	}{
		{pixy.AxisPan, "Pan"},
		{pixy.AxisTilt, "Tilt"},
		{pixy.AxisZoom, "Zoom"},
	}
	for _, tc := range tests {
		got := ptzAxes[tc.axis].Label
		if got != tc.want {
			t.Errorf("ptzAxes[%q].Label = %q, want %q", tc.axis, got, tc.want)
		}
	}

	assertAxisNotInMap(t, "unknown")
}

func TestPTZAxisUnit(t *testing.T) {
	t.Parallel()

	if got := ptzAxes[pixy.AxisPan].Unit; got != "\u00b0" {
		t.Errorf("pan unit = %q, want °", got)
	}

	if got := ptzAxes[pixy.AxisTilt].Unit; got != "\u00b0" {
		t.Errorf("tilt unit = %q, want °", got)
	}

	if got := ptzAxes[pixy.AxisZoom].Unit; got != "x" {
		t.Errorf("zoom unit = %q, want x", got)
	}
}

func TestPTZAxisValue(t *testing.T) {
	t.Parallel()

	status := webStatus{PTZValues: pixy.PTZValues{Pan: -10, Tilt: 5, Zoom: 200}}
	if got, ok := status.Get(pixy.AxisPan); got != -10 || !ok {
		t.Errorf("pan value = (%d, %v), want (-10, true)", got, ok)
	}

	if got, ok := status.Get(pixy.AxisTilt); got != 5 || !ok {
		t.Errorf("tilt value = (%d, %v), want (5, true)", got, ok)
	}

	if got, ok := status.Get(pixy.AxisZoom); got != 200 || !ok {
		t.Errorf("zoom value = (%d, %v), want (200, true)", got, ok)
	}

	if got, ok := status.Get("unknown"); got != 0 || ok {
		t.Errorf("unknown axis = (%d, %v), want (0, false)", got, ok)
	}
}

func TestRangeClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		r    pixy.Range
		v    int
		want int
	}{
		{"within", pixy.Range{Min: 0, Max: 10}, 5, 5},
		{"below", pixy.Range{Min: 0, Max: 10}, -5, 0},
		{"above", pixy.Range{Min: 0, Max: 10}, 15, 10},
		{"pan center", pixy.PanRange, 0, 0},
		{"pan below", pixy.PanRange, -200, pixy.PanRange.Min},
		{"pan above", pixy.PanRange, 200, pixy.PanRange.Max},
		{"zoom min", pixy.ZoomRange, pixy.ZoomRange.Min, pixy.ZoomRange.Min},
		{"zoom mid", pixy.ZoomRange, 125, 125},
		{"zoom above", pixy.ZoomRange, 500, pixy.ZoomRange.Max},
	}

	for _, tc := range tests {
		got := tc.r.Clamp(tc.v)
		if got != tc.want {
			t.Errorf("%s: Clamp(%d) = %d, want %d", tc.name, tc.v, got, tc.want)
		}
	}
}

func TestPTZLimits(t *testing.T) {
	t.Parallel()

	if info := ptzAxes[pixy.AxisPan]; info.Range != pixy.PanRange {
		t.Errorf("pan range: got %+v, want %+v", info.Range, pixy.PanRange)
	}

	if info := ptzAxes[pixy.AxisTilt]; info.Range != pixy.TiltRange {
		t.Errorf("tilt range: got %+v, want %+v", info.Range, pixy.TiltRange)
	}

	if info := ptzAxes[pixy.AxisZoom]; info.Range != pixy.ZoomRange {
		t.Errorf("zoom range: got %+v, want %+v", info.Range, pixy.ZoomRange)
	}

	assertAxisNotInMap(t, pixy.Axis("unknown"))
}

func TestPTZAxisValid(t *testing.T) {
	t.Parallel()

	if !ptzAxisValid(pixy.AxisPan) {
		t.Error("pan should be valid")
	}

	if !ptzAxisValid(pixy.AxisTilt) {
		t.Error("tilt should be valid")
	}

	if !ptzAxisValid(pixy.AxisZoom) {
		t.Error("zoom should be valid")
	}

	if ptzAxisValid("unknown") {
		t.Error("unknown should be invalid")
	}
}

func TestFormatLastSynced(t *testing.T) {
	t.Parallel()

	if result := formatLastSynced(time.Time{}); result != "" {
		t.Errorf("zero time should return empty, got %q", result)
	}

	if result := formatLastSynced(time.Now()); result != "just now" {
		t.Errorf("recent time should return 'just now', got %q", result)
	}

	if result := formatLastSynced(time.Now().Add(-2 * time.Minute)); result != "2m ago" {
		t.Errorf("2 min ago should return '2m ago', got %q", result)
	}

	old := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)

	result := formatLastSynced(old)
	if len(result) != 5 {
		t.Errorf("old time should return HH:MM format, got %q", result)
	}
}

func BenchmarkFormatLastSynced(b *testing.B) {
	t := time.Now()

	b.ResetTimer()

	for b.Loop() {
		formatLastSynced(t)
	}
}
