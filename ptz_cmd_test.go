//go:build linux

package main

import (
	"context"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestParsePTZValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantVal int
		wantRel bool
		wantErr bool
	}{
		{"50", 50, false, false},
		{"-30", -30, false, false},
		{"-90", -90, false, false},
		{"+45", 45, false, false},
		{"0", 0, false, false},
		{"rel+10", 10, true, false},
		{"rel-5", -5, true, false},
		{"rel0", 0, true, false},
		{"rel", 0, false, true},
		{"", 0, false, true},
		{"abc", 0, false, true},
		{"relabc", 0, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			val, relative, err := parsePTZValue(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parsePTZValue(%q): expected error, got nil", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("parsePTZValue(%q): unexpected error: %v", tc.input, err)
			}

			if val != tc.wantVal {
				t.Errorf("parsePTZValue(%q) val = %d, want %d", tc.input, val, tc.wantVal)
			}

			if relative != tc.wantRel {
				t.Errorf("parsePTZValue(%q) relative = %v, want %v", tc.input, relative, tc.wantRel)
			}
		})
	}
}

func newPTZDaemon(tb testing.TB, opts ...testDaemonOption) *Daemon {
	tb.Helper()

	// PTZ behavior tests must be hardware-independent: relative-mode commands
	// (e.g. "tilt -30") read current position via parsePTZ. Wire a deterministic
	// stub so relative math is reproducible regardless of real /dev/video0 state.
	opts = append([]testDaemonOption{withNoopParsePTZ()}, opts...)

	return newTestDaemon(tb, pixy.StateTracking, "/dev/video0", "/dev/hidraw7", opts...)
}

func newPTZCaptureDaemon(tb testing.TB, opts ...testDaemonOption) (*Daemon, *[]v4l2Call) {
	tb.Helper()

	var calls []v4l2Call

	d := newPTZDaemon(tb, append(opts, withCaptureV4L2(&calls))...)

	return d, &calls
}

func TestHandlePTZCommand_MissingArgs(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon(t)

	resp := d.handlePTZCommand(context.Background(), []string{string(pixy.AxisPan)})
	assertStatusPrefix(t, resp.String(), "usage:", "usage message")
}

func TestHandlePTZCommand_InvalidValue(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon(t)

	resp := d.handlePTZCommand(context.Background(), []string{string(pixy.AxisPan), "not-a-number"})
	expectError(t, resp)

	assertCommandContains(t, resp.String(), "pan", "error")
}

func TestHandlePTZCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, "", "")

	resp := d.handlePTZCommand(context.Background(), []string{string(pixy.AxisPan), "10"})
	expectError(t, resp)

	assertCommandContains(t, resp.String(), "device not found", "error")
}

func TestHandlePTZCommand_V4L2Error(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw7", func(d *Daemon) {
		d.deps.v4l2Set = func(context.Context, string, string, string) error {
			return ErrInvalidValue
		}
	})

	resp := d.handlePTZCommand(context.Background(), []string{string(pixy.AxisPan), "10"})
	expectError(t, resp)
}

func TestHandlePTZCommand_Success(t *testing.T) {
	t.Parallel()

	d, _ := newPTZCaptureDaemon(t)

	resp := d.handlePTZCommand(context.Background(), []string{string(pixy.AxisPan), "10"})
	notError(t, resp)

	assertCommandContainsAnyOf(t, resp.String(), []string{"pan", "10"}, "response")
}

func TestHandlePTZCommand_ZoomNoMultiplier(t *testing.T) {
	t.Parallel()

	d, setCalls := newPTZCaptureDaemon(t)

	d.handlePTZCommand(context.Background(), []string{string(pixy.AxisZoom), "125"})

	if len(*setCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*setCalls))
	}

	if (*setCalls)[0].val != "125" {
		t.Errorf("zoom value = %s, want 125 (no multiplier)", (*setCalls)[0].val)
	}
}

func TestHandleCenterCommand_Success(t *testing.T) {
	t.Parallel()

	var calls int

	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw7", withCaptureCenter(&calls))

	resp := d.handleCenterCommand(context.Background())
	notError(t, resp)

	if calls != 1 {
		t.Errorf("centerCamera called %d times, want 1", calls)
	}
}

func TestHandleCenterCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, "", "")

	resp := d.handleCenterCommand(context.Background())
	expectError(t, resp)
}
