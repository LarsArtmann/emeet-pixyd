//go:build linux

package pixy

import "testing"

func TestPTZValues_Clamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ptz  PTZValues
		want PTZValues
	}{
		{"within limits", PTZValues{Pan: 0, Tilt: 0, Zoom: 125}, PTZValues{Pan: 0, Tilt: 0, Zoom: 125}},
		{"pan over max", PTZValues{Pan: 500, Tilt: 0, Zoom: 100}, PTZValues{Pan: PanRange.Max, Tilt: 0, Zoom: 100}},
		{"pan under min", PTZValues{Pan: -500, Tilt: 0, Zoom: 100}, PTZValues{Pan: PanRange.Min, Tilt: 0, Zoom: 100}},
		{"tilt over max", PTZValues{Pan: 0, Tilt: 100, Zoom: 100}, PTZValues{Pan: 0, Tilt: TiltRange.Max, Zoom: 100}},
		{"tilt under min", PTZValues{Pan: 0, Tilt: -100, Zoom: 100}, PTZValues{Pan: 0, Tilt: TiltRange.Min, Zoom: 100}},
		{"zoom under min", PTZValues{Pan: 0, Tilt: 0, Zoom: 0}, PTZValues{Pan: 0, Tilt: 0, Zoom: ZoomRange.Min}},
		{"zoom over max", PTZValues{Pan: 0, Tilt: 0, Zoom: 500}, PTZValues{Pan: 0, Tilt: 0, Zoom: ZoomRange.Max}},
		{
			"all clamped",
			PTZValues{Pan: -999, Tilt: 999, Zoom: 999},
			PTZValues{Pan: PanRange.Min, Tilt: TiltRange.Max, Zoom: ZoomRange.Max},
		},
	}
	for _, tc := range tests {
		got := tc.ptz.Clamp()
		if got != tc.want {
			t.Errorf("%s: Clamp() = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestPTZValues_Get(t *testing.T) {
	t.Parallel()

	ptz := PTZValues{Pan: 10, Tilt: -5, Zoom: 200}

	if got := ptz.Get(AxisPan); got != 10 {
		t.Errorf("Get(pan) = %d, want 10", got)
	}

	if got := ptz.Get(AxisTilt); got != -5 {
		t.Errorf("Get(tilt) = %d, want -5", got)
	}

	if got := ptz.Get(AxisZoom); got != 200 {
		t.Errorf("Get(zoom) = %d, want 200", got)
	}

	if got := ptz.Get(Axis("unknown")); got != 0 {
		t.Errorf("Get(unknown) = %d, want 0", got)
	}
}

func TestPTZValues_Set(t *testing.T) {
	t.Parallel()

	ptz := PTZValues{Pan: 1, Tilt: 2, Zoom: 3}

	if got := ptz.Set(AxisPan, 42); got != (PTZValues{Pan: 42, Tilt: 2, Zoom: 3}) {
		t.Errorf("Set(pan, 42) = %+v, want {Pan:42, Tilt:2, Zoom:3}", got)
	}

	if got := ptz.Set(AxisTilt, -10); got != (PTZValues{Pan: 1, Tilt: -10, Zoom: 3}) {
		t.Errorf("Set(tilt, -10) = %+v, want {Pan:1, Tilt:-10, Zoom:3}", got)
	}

	if got := ptz.Set(AxisZoom, 300); got != (PTZValues{Pan: 1, Tilt: 2, Zoom: 300}) {
		t.Errorf("Set(zoom, 300) = %+v, want {Pan:1, Tilt:2, Zoom:300}", got)
	}

	if got := ptz.Set(Axis("unknown"), 999); got != ptz {
		t.Errorf("Set(unknown, 999) should return unchanged copy")
	}
}
