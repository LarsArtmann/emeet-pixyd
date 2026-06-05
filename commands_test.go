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

	err := &CommandError{Op: pixy.AxisPan, Err: ErrInvalidValue}

	want := "error: pan: invalid value"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCommandError_Unwrap(t *testing.T) {
	t.Parallel()

	err := &CommandError{Op: pixy.AxisPan, Err: ErrInvalidValue}

	got := err.Unwrap()
	if !errors.Is(got, ErrInvalidValue) {
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
		d.deps.v4l2Set = func(_ context.Context, _, axis, val string) error {
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

// ---------------------------------------------------------------------------
// handleCenterCommand tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// handleAutoCommand tests
// ---------------------------------------------------------------------------

func TestHandleAutoCommand_SetMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{cmdAuto, "full"})
	if resp.IsError() {
		t.Errorf("expected success, got: %s", resp.String())
	}

	assertCommandContains(t, resp.String(), "full", "response")

	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAutoCommand([]string{"auto", "invalid-mode"})
	assertCommandContains(t, resp.String(), "usage:", "response")
}

func TestHandleAutoCommand_ToggleOff(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{"auto-on"})
	notError(t, resp.String())
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_ToggleOn(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoFull
	})

	resp := d.handleAutoCommand([]string{cmdAutoOff})
	notError(t, resp.String())
	assertAutoModeEquals(t, d, pixy.AutoOff)
}

func TestHandleAutoCommand_ToggleAuto(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{cmdToggleAuto})
	notError(t, resp.String())
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_BareAutoShowsCurrentMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoTrackingOnly
	})

	resp := d.handleAutoCommand([]string{cmdAuto})
	notError(t, resp.String())
	assertCommandContains(t, resp.String(), "tracking-only", "response")
	assertAutoModeEquals(t, d, pixy.AutoTrackingOnly)
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
	notError(t, resp.String())

	if !called || !enabledArg {
		t.Errorf("setGesture called=%v enabled=%v, want true/true", called, enabledArg)
	}
}

func TestHandleGestureCommand_Off(t *testing.T) {
	t.Parallel()

	var called, enabledArg bool

	d := newTestDaemon(pixy.StatePrivacy, "", "", withCaptureGesture(&called, &enabledArg))

	resp := d.handleGestureCommand(context.Background(), cmdGestureOff)
	notError(t, resp.String())

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
	notError(t, resp.String())

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
	notError(t, resp.String())

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
	notError(t, resp.String())

	if modeArg != pixy.AudioLive {
		t.Errorf("setAudio called with %s, want %s", modeArg, pixy.AudioLive)
	}
}

func TestHandleAudioCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAudioCommand(context.Background(), []string{"audio", "invalid"})
	assertCommandContains(t, resp.String(), "error:", "response")
}

func TestHandleAudioCommand_NextMode(t *testing.T) {
	t.Parallel()

	var modeArg pixy.AudioMode

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.Audio = pixy.AudioNC
	}, withCaptureAudio(&modeArg))

	resp := d.handleAudioCommand(context.Background(), []string{cmdAudio})
	notError(t, resp.String())

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
	notError(t, resp.String())

	if stateArg != pixy.StateTracking {
		t.Errorf("setTracking called with %s, want %s", stateArg, pixy.StateTracking)
	}
}

func TestHandleTrackingCommand_ErrorPath(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.deps.setTracking = func(context.Context, pixy.CameraState) error {
			return ErrInvalidValue
		}
	})

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, cmdTrack)
	if !resp.IsError() {
		t.Errorf("expected error response, got: %s", resp.String())
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
// applyResultToStatus tests
// ---------------------------------------------------------------------------

func TestApplyResultToStatus_Error(t *testing.T) {
	t.Parallel()

	status := webStatus{}
	applyResultToStatus(errResult("pan", ErrInvalidValue), &status, "ignored", toastTypeError)

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

func TestApplyResultToStatus_Success(t *testing.T) {
	t.Parallel()

	status := webStatus{}
	applyResultToStatus(okResult(respTrackingOn), &status, "Tracking enabled", toastTypeSuccess)

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

func TestApplyResultToStatus_InfoToast(t *testing.T) {
	t.Parallel()

	status := webStatus{}
	applyResultToStatus(okResult("ok"), &status, "Toggled", toastTypeInfo)

	if status.ToastType != toastTypeInfo {
		t.Errorf("ToastType = %q, want %q", status.ToastType, toastTypeInfo)
	}
}

func TestApplyResultToStatus_ErrorOverridesToast(t *testing.T) {
	t.Parallel()

	status := webStatus{}
	applyResultToStatus(errResultMsg("something failed"), &status, "Success msg", toastTypeSuccess)

	if status.Error == "" {
		t.Error("Error should be set for error result")
	}

	if status.Toast != "" {
		t.Error("Toast should not be set when result is an error")
	}

	if status.ToastType != "" {
		t.Error("ToastType should not be set when result is an error")
	}
}

// ---------------------------------------------------------------------------
// handleQueryCommand tests
// ---------------------------------------------------------------------------

func TestHandleQueryCommand_Version(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleQueryCommand(context.Background(), []string{cmdVersion})
	if !strings.HasPrefix(resp.String(), "emeet-pixyd ") {
		t.Errorf("expected version prefix, got: %s", resp.String())
	}
}

func TestHandleQueryCommand_Waybar(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleQueryCommand(context.Background(), []string{cmdWaybar})
	if !strings.Contains(resp.String(), `"class"`) {
		t.Errorf("expected JSON with 'class' key, got: %s", resp.String())
	}
}

func TestHandleQueryCommand_Device_NoDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleQueryCommand(context.Background(), []string{cmdDevice})
	if resp.String() != respDeviceNotFound {
		t.Errorf("expected %q, got: %s", respDeviceNotFound, resp.String())
	}
}

func TestHandleQueryCommand_Device_WithDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonWithDevice(pixy.StateTracking)

	resp := d.handleQueryCommand(context.Background(), []string{cmdDevice})
	if !strings.Contains(resp.String(), "/dev/video") {
		t.Errorf("expected device path, got: %s", resp.String())
	}
}

func TestHandleQueryCommand_Sync_NoDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleQueryCommand(context.Background(), []string{cmdSync})
	if !resp.IsError() {
		t.Errorf("expected error response for sync without device, got: %s", resp.String())
	}
}

func TestHandleQueryCommand_Probe_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateOffline, "", "")

	resp := d.handleQueryCommand(context.Background(), []string{cmdProbe})
	// probeDevices() scans real sysfs, so this test is flaky when a PIXY is connected.
	if resp.String() == respDeviceNotFound {
		return
	}

	if !strings.Contains(resp.String(), "/dev/video") {
		t.Errorf("expected device path or %q, got: %s", respDeviceNotFound, resp.String())
	}
}

// ---------------------------------------------------------------------------
// handleTogglePrivacy tests
// ---------------------------------------------------------------------------

func TestHandleTogglePrivacy_FromPrivacy(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(
		pixy.StatePrivacy, "/dev/video0", "/dev/hidraw7",
		withCaptureTracking(&stateArg),
	)

	resp := d.handleTogglePrivacy(context.Background())
	notError(t, resp.String())

	if stateArg != pixy.StateTracking {
		t.Errorf("toggle from privacy should set tracking, got: %s", stateArg)
	}
}

func TestHandleTogglePrivacy_FromTracking(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(
		pixy.StateTracking, "/dev/video0", "/dev/hidraw7",
		withCaptureTracking(&stateArg),
	)

	resp := d.handleTogglePrivacy(context.Background())
	notError(t, resp.String())

	if stateArg != pixy.StatePrivacy {
		t.Errorf("toggle from tracking should set privacy, got: %s", stateArg)
	}
}

func TestHandleTogglePrivacy_FromIdle(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(
		pixy.StateIdle, "/dev/video0", "/dev/hidraw7",
		withCaptureTracking(&stateArg),
	)

	resp := d.handleTogglePrivacy(context.Background())
	notError(t, resp.String())

	if stateArg != pixy.StatePrivacy {
		t.Errorf("toggle from idle should set privacy, got: %s", stateArg)
	}
}

// ---------------------------------------------------------------------------
// handleMutatingCommand tests
// ---------------------------------------------------------------------------

func TestHandleMutatingCommand_Unknown(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleMutatingCommand(context.Background(), []string{"unknown-cmd"})
	if !strings.HasPrefix(resp.String(), "error: unknown command:") {
		t.Errorf("expected unknown command response, got: %s", resp.String())
	}
}

// ---------------------------------------------------------------------------
// handleCommand routing tests (query vs mutating)
// ---------------------------------------------------------------------------

func TestHandleCommand_QueryNoLock(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	tests := []struct {
		cmd    string
		substr string
	}{
		{cmdVersion, "emeet-pixyd "},
		{cmdWaybar, `"class"`},
		{cmdDevice, respDeviceNotFound},
		{cmdStatus, "camera="},
	}
	for _, tc := range tests {
		t.Run(tc.cmd, func(t *testing.T) {
			t.Parallel()

			resp := d.handleCommand(context.Background(), tc.cmd)
			assertCommandContains(t, resp.String(), tc.substr, "response")
		})
	}
}

func TestHandleCommand_MutatingRequiresLock(t *testing.T) {
	t.Parallel()

	d := testDaemonWithDevice(pixy.StatePrivacy)
	d.config = defaultTestConfig(t.TempDir())

	resp := d.handleCommand(context.Background(), cmdAutoOff)
	notError(t, resp.String())
	assertAutoModeEquals(t, d, pixy.AutoOff)

	resp = d.handleCommand(context.Background(), cmdAutoOn)
	notError(t, resp.String())
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleCommand_EmptyCommandReturnsStatus(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleCommand(context.Background(), "")
	assertCommandContains(t, resp.String(), "camera=", "status response")
}
