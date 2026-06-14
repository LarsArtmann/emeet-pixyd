//go:build linux

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

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

func notError(t *testing.T, result CommandResult) {
	t.Helper()

	if result.IsError() {
		t.Errorf("expected success, got: %s", result.String())
	}
}

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

func TestHandleMutatingCommand_Unknown(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleMutatingCommand(context.Background(), []string{"unknown-cmd"})
	if !strings.HasPrefix(resp.String(), "error: unknown command:") {
		t.Errorf("expected unknown command response, got: %s", resp.String())
	}
}

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
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoOff)

	resp = d.handleCommand(context.Background(), cmdAutoOn)
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleCommand_EmptyCommandReturnsStatus(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	resp := d.handleCommand(context.Background(), "")
	assertCommandContains(t, resp.String(), "camera=", "status response")
}
