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
	if got := ptzAxisValue(pixy.AxisPan, status); got != -10 {
		t.Errorf("pan value = %d, want -10", got)
	}

	if got := ptzAxisValue(pixy.AxisTilt, status); got != 5 {
		t.Errorf("tilt value = %d, want 5", got)
	}

	if got := ptzAxisValue(pixy.AxisZoom, status); got != 200 {
		t.Errorf("zoom value = %d, want 200", got)
	}

	if got := ptzAxisValue("unknown", status); got != 0 {
		t.Errorf("unknown axis = %d, want 0", got)
	}
}

func TestClampInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v, lo, hi, want int
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{0, pixy.PanRange.Min, pixy.PanRange.Max, 0},
		{-200, pixy.PanRange.Min, pixy.PanRange.Max, pixy.PanRange.Min},
		{200, pixy.PanRange.Min, pixy.PanRange.Max, pixy.PanRange.Max},
		{pixy.ZoomRange.Min, pixy.ZoomRange.Min, pixy.ZoomRange.Max, pixy.ZoomRange.Min},
		{125, pixy.ZoomRange.Min, pixy.ZoomRange.Max, 125},
		{500, pixy.ZoomRange.Min, pixy.ZoomRange.Max, pixy.ZoomRange.Max},
	}

	for _, tc := range tests {
		got := clampInt(tc.v, tc.lo, tc.hi)
		if got != tc.want {
			t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestPTZLimits(t *testing.T) {
	t.Parallel()

	if info := ptzAxes[pixy.AxisPan]; info.Min != pixy.PanRange.Min || info.Max != pixy.PanRange.Max {
		t.Errorf("pan limits: got %d,%d, want %d,%d", info.Min, info.Max, pixy.PanRange.Min, pixy.PanRange.Max)
	}

	if info := ptzAxes[pixy.AxisTilt]; info.Min != pixy.TiltRange.Min || info.Max != pixy.TiltRange.Max {
		t.Errorf("tilt limits: got %d,%d, want %d,%d", info.Min, info.Max, pixy.TiltRange.Min, pixy.TiltRange.Max)
	}

	if info := ptzAxes[pixy.AxisZoom]; info.Min != pixy.ZoomRange.Min || info.Max != pixy.ZoomRange.Max {
		t.Errorf("zoom limits: got %d,%d, want %d,%d", info.Min, info.Max, pixy.ZoomRange.Min, pixy.ZoomRange.Max)
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
