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

	err := &CommandError{Op: axisPan, Err: ErrInvalidValue}
	want := "error: pan: invalid value"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCommandError_Unwrap(t *testing.T) {
	t.Parallel()

	err := &CommandError{Op: axisPan, Err: ErrInvalidValue}
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
		{respTrackingOn, false},
		{respPrivacyOn, false},
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
	allOpts := append([]testDaemonOption{func(_ *Daemon) {}}, opts...)
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

	resp := d.handlePTZCommand(context.Background(), []string{axisPan})
	if !strings.HasPrefix(resp, "usage:") {
		t.Errorf("expected usage message, got: %s", resp)
	}
}

func TestHandlePTZCommand_InvalidValue(t *testing.T) {
	t.Parallel()

	d := newPTZDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{axisPan, "not-a-number"})
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
	assertCommandContains(t, resp, "pan", "error")
}

func TestHandlePTZCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "", "")

	resp := d.handlePTZCommand(context.Background(), []string{axisPan, "10"})
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
	assertCommandContains(t, resp, "device not found", "error")
}

func TestHandlePTZCommand_V4L2Error(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", func(d *Daemon) {
		d.v4l2SetFn = func(context.Context, string, string, string) error {
			return ErrInvalidValue
		}
	})

	resp := d.handlePTZCommand(context.Background(), []string{axisPan, "10"})
	if !IsCommandErrorResponse(resp) {
		t.Errorf("expected error response, got: %s", resp)
	}
}

func TestHandlePTZCommand_Success(t *testing.T) {
	t.Parallel()

	d, _ := newPTZCaptureDaemon()

	resp := d.handlePTZCommand(context.Background(), []string{axisPan, "10"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got error: %s", resp)
	}
	assertCommandContainsAnyOf(t, resp, []string{"pan", "10"}, "response")
}

func TestHandlePTZCommand_ZoomNoMultiplier(t *testing.T) {
	t.Parallel()

	d, setCalls := newPTZCaptureDaemon()

	d.handlePTZCommand(context.Background(), []string{axisZoom, "200"})
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
	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7", withCaptureCenter(&calls))

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

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{cmdAuto, "full"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	assertCommandContains(t, resp, "full", "response")

	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAutoCommand([]string{"auto", "invalid-mode"})
	assertCommandContains(t, resp, "usage:", "response")
}

func TestHandleAutoCommand_ToggleOff(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{"auto-on"})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_ToggleOn(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoFull
	})

	resp := d.handleAutoCommand([]string{cmdAutoOff})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoOff)
}

func TestHandleAutoCommand_ToggleAuto(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{cmdToggleAuto})
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

	var called, enabledArg bool
	d := newTestDaemon(pixy.StatePrivacy, "", "", withCaptureGesture(&called, &enabledArg))

	resp := d.handleGestureCommand(context.Background(), cmdGestureOn)
	notError(t, resp)
	if !called || !enabledArg {
		t.Errorf("setGesture called=%v enabled=%v, want true/true", called, enabledArg)
	}
}

func TestHandleGestureCommand_Off(t *testing.T) {
	t.Parallel()

	var called, enabledArg bool
	d := newTestDaemon(pixy.StatePrivacy, "", "", withCaptureGesture(&called, &enabledArg))

	resp := d.handleGestureCommand(context.Background(), cmdGestureOff)
	notError(t, resp)
	if !called || enabledArg {
		t.Errorf("setGesture called=%v enabled=%v, want true/false", called, enabledArg)
	}
}

func TestHandleGestureCommand_ToggleOn(t *testing.T) {
	t.Parallel()

	var enabledArg bool
	d := newTestDaemon(
		pixy.StatePrivacy, "", "",
		func(d *Daemon) { d.state.Gesture = false },
		withCaptureGestureArg(&enabledArg),
	)

	resp := d.handleGestureCommand(context.Background(), cmdToggleGesture)
	notError(t, resp)
	if !enabledArg {
		t.Errorf("toggle should enable gesture, got enabled=%v", enabledArg)
	}
}

func TestHandleGestureCommand_ToggleOff(t *testing.T) {
	t.Parallel()

	var enabledArg bool
	d := newTestDaemon(
		pixy.StatePrivacy, "", "",
		func(d *Daemon) { d.state.Gesture = true },
		withCaptureGestureArg(&enabledArg),
	)

	resp := d.handleGestureCommand(context.Background(), cmdToggleGesture)
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
	d := newTestDaemon(pixy.StatePrivacy, "", "", withCaptureAudio(&modeArg))

	resp := d.handleAudioCommand(context.Background(), []string{cmdAudio, string(pixy.AudioLive)})
	notError(t, resp)
	if modeArg != pixy.AudioLive {
		t.Errorf("setAudio called with %s, want %s", modeArg, pixy.AudioLive)
	}
}

func TestHandleAudioCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAudioCommand(context.Background(), []string{"audio", "invalid"})
	assertCommandContains(t, resp, "error:", "response")
}

func TestHandleAudioCommand_NextMode(t *testing.T) {
	t.Parallel()

	var modeArg pixy.AudioMode
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.Audio = pixy.AudioNC
	}, withCaptureAudio(&modeArg))

	resp := d.handleAudioCommand(context.Background(), []string{cmdAudio})
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
	d := newTestDaemon(pixy.StatePrivacy, "", "", withCaptureTracking(&stateArg))

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, cmdTrack)
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

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, cmdTrack)
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
		{cmdTrack, toastTrackingEnabled},
		{cmdIdle, toastCameraIdle},
		{cmdPrivacy, toastPrivacyOn},
		{cmdCenter, toastCameraCentered},
		{cmdSync, toastStateSynced},
		{cmdProbe, toastProbedDevices},
		{cmdToggleGesture, "Gesture toggled"},
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
	applyResponseToStatus("error: pan: bad", &status, "ignored", toastTypeError)
	if status.Error == "" {
		t.Error("Error should be set")
	}
	if status.Toast != "" {
		t.Error("Toast should not be set for error")
	}
	if status.ToastType != "" {
		t.Error("ToastType should not be set for error")
	}
}

func TestApplyResponseToStatus_Success(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	status := webStatus{}
	applyResponseToStatus(respTrackingOn, &status, "Tracking enabled", toastTypeSuccess)
	if status.Error != "" {
		t.Error("Error should not be set")
	}
	if status.Toast != "Tracking enabled" {
		t.Errorf("Toast = %q, want %q", status.Toast, "Tracking enabled")
	}
	if status.ToastType != toastTypeSuccess {
		t.Errorf("ToastType = %q, want %q", status.ToastType, toastTypeSuccess)
	}
}

func TestApplyResponseToStatus_InfoToast(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	status := webStatus{}
	applyResponseToStatus("ok", &status, "Toggled", toastTypeInfo)
	if status.ToastType != toastTypeInfo {
		t.Errorf("ToastType = %q, want %q", status.ToastType, toastTypeInfo)
	}
}

func TestApplyResponseToStatus_ErrorOverridesToast(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	status := webStatus{}
	applyResponseToStatus("error: something failed", &status, "Success msg", toastTypeSuccess)
	if status.Error == "" {
		t.Error("Error should be set for error response")
	}
	if status.Toast != "" {
		t.Error("Toast should not be set when response is an error")
	}
	if status.ToastType != "" {
		t.Error("ToastType should not be set when response is an error")
	}
}
