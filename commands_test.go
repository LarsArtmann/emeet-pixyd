//go:build linux

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// ---------------------------------------------------------------------------
// CommandError tests
// ---------------------------------------------------------------------------

func TestCommandError_Error(t *testing.T) {
	t.Parallel()

	err := &CommandError{Op: "pan", Err: ErrInvalidValue}
	want := "error: pan: invalid value"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCommandError_Unwrap(t *testing.T) {
	t.Parallel()

	err := &CommandError{Op: "pan", Err: ErrInvalidValue}
	if got := err.Unwrap(); !errors.Is(got, ErrInvalidValue) {
		t.Errorf("Unwrap() = %v, want ErrInvalidValue", got)
	}
}

func TestIsCommandErrorResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s    string
		want bool
	}{
		{"error: pan: invalid value", true},
		{"error: zoom: device not found", true},
		{"tracking on", false},
		{"privacy on", false},
		{"error:", false},
		{"ERROR: pan: invalid value", false}, // wrong case
		{"error:pan", false},                 // no space after colon
	}

	for _, tc := range tests {
		t.Run(tc.s, func(t *testing.T) {
			t.Parallel()

			if got := IsCommandErrorResponse(tc.s); got != tc.want {
				t.Errorf("IsCommandErrorResponse(%q) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handlePTZCommand tests
// ---------------------------------------------------------------------------

func newPTZDaemon(opts ...testDaemonOption) *Daemon {
	allOpts := append([]testDaemonOption{func(d *Daemon) {
		d.v4l2SetFn = func(context.Context, string, string, string) error { return nil }
	}}, opts...)
	return newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", allOpts...)
}

func newPTZCaptureDaemon(opts ...testDaemonOption) (*Daemon, *[]struct{ axis, val string }) {
	var calls []struct{ axis, val string }
	d := newPTZDaemon(append(opts, func(d *Daemon) {
		d.v4l2SetFn = func(_ context.Context, _, axis, val string) error {
			calls = append(calls, struct{ axis, val string }{axis, val})
			return nil
		}
	})...)
	return d, &calls
}

func newAutoOffDaemon() *Daemon {
	return newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoOff
	})
}

func assertAutoModeEquals(t *testing.T, d *Daemon, want pixy.AutoMode) {
	t.Helper()
	d.mu.RLock()
	got := d.state.AutoMode
	d.mu.RUnlock()
	if got != want {
		t.Errorf("AutoMode = %s, want %s", got, want)
	}
}

func TestHandlePTZCommand_MissingArgs(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{"pan"})
	if !strings.HasPrefix(resp, "usage:") {
		t.Errorf("expected usage message, got: %s", resp)
	}
}

func TestHandlePTZCommand_InvalidValue(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{"pan", "not-a-number"})
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
	if !strings.Contains(resp, "pan") {
		t.Errorf("error should mention axis 'pan', got: %s", resp)
	}
}

func TestHandlePTZCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "", "", func(d *Daemon) {
		d.v4l2SetFn = func(context.Context, string, string, string) error { return nil }
	})

	resp := d.handlePTZCommand(context.Background(), []string{"pan", "10"})
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
	if !strings.Contains(resp, "device not found") {
		t.Errorf("error should mention 'device not found', got: %s", resp)
	}
}

func TestHandlePTZCommand_V4L2Error(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", func(d *Daemon) {
		d.v4l2SetFn = func(context.Context, string, string, string) error {
			return ErrInvalidValue
		}
	})

	resp := d.handlePTZCommand(context.Background(), []string{"pan", "10"})
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
}

func TestHandlePTZCommand_Success(t *testing.T) {
	t.Parallel()

	d, _ := newPTZCaptureDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{"pan", "10"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got error: %s", resp)
	}
	if !strings.Contains(resp, "pan") || !strings.Contains(resp, "10") {
		t.Errorf("unexpected response: %s", resp)
	}
}

func TestHandlePTZCommand_ZoomNoMultiplier(t *testing.T) {
	t.Parallel()

	d, setCalls := newPTZCaptureDaemon()

	d.handlePTZCommand(context.Background(), []string{"zoom", "200"})
	if len(*setCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*setCalls))
	}
	if (*setCalls)[0].val != "200" {
		t.Errorf("zoom value = %s, want 200 (no multiplier)", (*setCalls)[0].val)
	}
}

// ---------------------------------------------------------------------------
// handleCenterCommand tests
// ---------------------------------------------------------------------------

func TestHandleCenterCommand_Success(t *testing.T) {
	t.Parallel()

	var calls int
	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", func(d *Daemon) {
		d.centerCameraFn = func(context.Context) error { calls++; return nil }
	})

	resp := d.handleCenterCommand(context.Background())
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	if calls != 1 {
		t.Errorf("centerCamera called %d times, want 1", calls)
	}
}

func TestHandleCenterCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "", "")

	resp := d.handleCenterCommand(context.Background())
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error, got: %s", resp)
	}
}

// ---------------------------------------------------------------------------
// handleAutoCommand tests
// ---------------------------------------------------------------------------

func TestHandleAutoCommand_SetMode(t *testing.T) {
	t.Parallel()

	d := newAutoOffDaemon()

	resp := d.handleAutoCommand([]string{"auto", "full"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	if !strings.Contains(resp, "full") {
		t.Errorf("response should mention 'full', got: %s", resp)
	}

	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAutoCommand([]string{"auto", "invalid-mode"})
	if !strings.Contains(resp, "usage:") {
		t.Errorf("expected usage message, got: %s", resp)
	}
}

func TestHandleAutoCommand_ToggleOff(t *testing.T) {
	t.Parallel()

	d := newAutoOffDaemon()

	resp := d.handleAutoCommand([]string{"auto-on"})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_ToggleOn(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoFull
	})

	resp := d.handleAutoCommand([]string{"auto-off"})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoOff)
}

func TestHandleAutoCommand_ToggleAuto(t *testing.T) {
	t.Parallel()

	d := newAutoOffDaemon()

	resp := d.handleAutoCommand([]string{"toggle-auto"})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

// ---------------------------------------------------------------------------
// handleGestureCommand tests
// ---------------------------------------------------------------------------

func notError(t *testing.T, resp string) {
	t.Helper()
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
}

func TestHandleGestureCommand_On(t *testing.T) {
	t.Parallel()

	var called bool
	var enabledArg bool
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.setGestureFn = func(context.Context, bool) error { called = true; enabledArg = true; return nil }
	})

	resp := d.handleGestureCommand(context.Background(), "gesture-on")
	notError(t, resp)
	if !called || !enabledArg {
		t.Errorf("setGesture called=%v enabled=%v, want true/true", called, enabledArg)
	}
}

func TestHandleGestureCommand_Off(t *testing.T) {
	t.Parallel()

	var called bool
	var enabledArg bool
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.setGestureFn = func(context.Context, bool) error { called = true; enabledArg = false; return nil }
	})

	resp := d.handleGestureCommand(context.Background(), "gesture-off")
	notError(t, resp)
	if !called || enabledArg {
		t.Errorf("setGesture called=%v enabled=%v, want true/false", called, enabledArg)
	}
}

func TestHandleGestureCommand_ToggleOn(t *testing.T) {
	t.Parallel()

	var enabledArg bool
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.Gesture = false
		d.setGestureFn = func(_ context.Context, enabled bool) error { enabledArg = enabled; return nil }
	})

	resp := d.handleGestureCommand(context.Background(), "toggle-gesture")
	notError(t, resp)
	if !enabledArg {
		t.Errorf("toggle should enable gesture, got enabled=%v", enabledArg)
	}
}

func TestHandleGestureCommand_ToggleOff(t *testing.T) {
	t.Parallel()

	var enabledArg bool
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.Gesture = true
		d.setGestureFn = func(_ context.Context, enabled bool) error { enabledArg = enabled; return nil }
	})

	resp := d.handleGestureCommand(context.Background(), "toggle-gesture")
	notError(t, resp)
	if enabledArg {
		t.Errorf("toggle should disable gesture, got enabled=%v", enabledArg)
	}
}

// ---------------------------------------------------------------------------
// handleAudioCommand tests
// ---------------------------------------------------------------------------

func TestHandleAudioCommand_SetMode(t *testing.T) {
	t.Parallel()

	var modeArg pixy.AudioMode
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.setAudioFn = func(_ context.Context, m pixy.AudioMode) error { modeArg = m; return nil }
	})

	resp := d.handleAudioCommand(context.Background(), []string{"audio", "live"})
	notError(t, resp)
	if modeArg != pixy.AudioLive {
		t.Errorf("setAudio called with %s, want %s", modeArg, pixy.AudioLive)
	}
}

func TestHandleAudioCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAudioCommand(context.Background(), []string{"audio", "invalid"})
	if !strings.Contains(resp, "usage:") {
		t.Errorf("expected usage message, got: %s", resp)
	}
}

func TestHandleAudioCommand_NextMode(t *testing.T) {
	t.Parallel()

	var modeArg pixy.AudioMode
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.Audio = pixy.AudioNC
		d.setAudioFn = func(_ context.Context, m pixy.AudioMode) error { modeArg = m; return nil }
	})

	resp := d.handleAudioCommand(context.Background(), []string{"audio"})
	notError(t, resp)
	if modeArg != pixy.AudioLive {
		t.Errorf("next mode = %s, want %s", modeArg, pixy.AudioLive)
	}
}

// ---------------------------------------------------------------------------
// handleTrackingCommand tests
// ---------------------------------------------------------------------------

func TestHandleTrackingCommand_SetTracking(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.setTrackingFn = func(_ context.Context, s pixy.CameraState) error { stateArg = s; return nil }
	})

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, "track")
	notError(t, resp)
	if stateArg != pixy.StateTracking {
		t.Errorf("setTracking called with %s, want %s", stateArg, pixy.StateTracking)
	}
}

func TestHandleTrackingCommand_ErrorPath(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.setTrackingFn = func(context.Context, pixy.CameraState) error {
			return ErrInvalidValue
		}
	})

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, "track")
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
}

// ---------------------------------------------------------------------------
// actionToast tests
// ---------------------------------------------------------------------------

func TestActionToast_KnownCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmd  string
		want string
	}{
		{"track", "Tracking enabled"},
		{"idle", "Camera idle"},
		{"privacy", "Privacy mode on"},
		{"center", "Camera centered"},
		{"sync", "State synced"},
		{"probe", "Probed devices"},
		{"toggle-gesture", "Gesture toggled"},
		{"toggle-auto", "Auto mode toggled"},
	}

	for _, tc := range tests {
		t.Run(tc.cmd, func(t *testing.T) {
			t.Parallel()

			got, _ := actionToast(tc.cmd)
			if got != tc.want {
				t.Errorf("actionToast(%q) = %q, want %q", tc.cmd, got, tc.want)
			}
		})
	}
}

func TestActionToast_UnknownCommand(t *testing.T) {
	t.Parallel()

	got, gotType := actionToast("unknown-command")
	if got != "" || gotType != "" {
		t.Errorf("actionToast(unknown) = (%q, %q), want empty", got, gotType)
	}
}

// ---------------------------------------------------------------------------
// applyResponseToStatus tests
// ---------------------------------------------------------------------------

func TestApplyResponseToStatus_Error(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	status := webStatus{}
	applyResponseToStatus("error: pan: bad", &status, "ignored")
	if status.Error == "" {
		t.Error("Error should be set")
	}
	if status.Toast != "" {
		t.Error("Toast should not be set for error")
	}
}

func TestApplyResponseToStatus_Success(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	status := webStatus{}
	applyResponseToStatus("tracking on", &status, "Tracking enabled")
	if status.Error != "" {
		t.Error("Error should not be set")
	}
	if status.Toast != "Tracking enabled" {
		t.Errorf("Toast = %q, want %q", status.Toast, "Tracking enabled")
	}
}
