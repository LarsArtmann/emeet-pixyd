//go:build linux

package main

import (
	"context"
	"strings"
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
		{"-30", -30, true, false},
		{"+45", 45, true, false},
		{"+0", 0, true, false},
		{"-0", 0, true, false},
		{"0", 0, false, false},
		{"", 0, false, true},
		{"+", 0, false, true},
		{"-", 0, false, true},
		{"abc", 0, false, true},
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

func newPTZDaemon(opts ...testDaemonOption) *Daemon {
	allOpts := append([]testDaemonOption{func(_ *Daemon) {}}, opts...)

	return newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", allOpts...)
}

func newPTZCaptureDaemon(opts ...testDaemonOption) (*Daemon, *[]struct{ axis, val string }) {
	var calls []struct{ axis, val string }

	d := newPTZDaemon(append(opts, func(d *Daemon) {
		d.deps.v4l2Set = func(_ context.Context, _, axis, val string) error {
			calls = append(calls, struct{ axis, val string }{axis, val})

			return nil
		}
	})...)

	return d, &calls
}

func TestHandlePTZCommand_MissingArgs(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{pixy.AxisPan})
	if !strings.HasPrefix(resp.String(), "usage:") {
		t.Errorf("expected usage message, got: %s", resp.String())
	}
}

func TestHandlePTZCommand_InvalidValue(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{pixy.AxisPan, "not-a-number"})
	if !resp.IsError() {
		t.Errorf("expected error response, got: %s", resp.String())
	}

	assertCommandContains(t, resp.String(), "pan", "error")
}

func TestHandlePTZCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "", "")

	resp := d.handlePTZCommand(context.Background(), []string{pixy.AxisPan, "10"})
	if !resp.IsError() {
		t.Errorf("expected error response, got: %s", resp.String())
	}

	assertCommandContains(t, resp.String(), "device not found", "error")
}

func TestHandlePTZCommand_V4L2Error(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", func(d *Daemon) {
		d.deps.v4l2Set = func(context.Context, string, string, string) error {
			return ErrInvalidValue
		}
	})

	resp := d.handlePTZCommand(context.Background(), []string{pixy.AxisPan, "10"})
	if !resp.IsError() {
		t.Errorf("expected error response, got: %s", resp.String())
	}
}

func TestHandlePTZCommand_Success(t *testing.T) {
	t.Parallel()

	d, _ := newPTZCaptureDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{pixy.AxisPan, "10"})
	if resp.IsError() {
		t.Errorf("expected success, got error: %s", resp.String())
	}

	assertCommandContainsAnyOf(t, resp.String(), []string{"pan", "10"}, "response")
}

func TestHandlePTZCommand_ZoomNoMultiplier(t *testing.T) {
	t.Parallel()

	d, setCalls := newPTZCaptureDaemon()

	d.handlePTZCommand(context.Background(), []string{pixy.AxisZoom, "200"})

	if len(*setCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*setCalls))
	}

	if (*setCalls)[0].val != "200" {
		t.Errorf("zoom value = %s, want 200 (no multiplier)", (*setCalls)[0].val)
	}
}

func TestHandleCenterCommand_Success(t *testing.T) {
	t.Parallel()

	var calls int

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", withCaptureCenter(&calls))

	resp := d.handleCenterCommand(context.Background())
	if resp.IsError() {
		t.Errorf("expected success, got: %s", resp.String())
	}

	if calls != 1 {
		t.Errorf("centerCamera called %d times, want 1", calls)
	}
}

func TestHandleCenterCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "", "")

	resp := d.handleCenterCommand(context.Background())
	if !resp.IsError() {
		t.Errorf("expected error, got: %s", resp.String())
	}
}
