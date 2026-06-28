//go:build linux

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestHandleCommandStatus(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	result := d.handleCommand(context.Background(), cmdStatus)
	assertStatusPrefix(t, result.String(), "camera=offline", "offline status")
	assertStatusContains(t, result.String(), "audio=", "offline status")
	assertStatusContains(t, result.String(), "auto=", "offline status")
}

func TestHandleCommandUnknown(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, "/dev/hidraw0")

	result := d.handleCommand(context.Background(), "foobar")
	assertEqual(t, result.String(), "error: unknown command: foobar")
}

func TestHandleCommandAutoToggle(t *testing.T) {
	t.Parallel()

	d := testDaemonWithDevice(t, pixy.StatePrivacy)
	d.config = defaultTestConfig(t.TempDir())

	result := d.handleCommand(context.Background(), "auto-off")
	assertEqual(t, result.String(), respAutoModeOff)

	if d.state.AutoMode != pixy.AutoOff {
		t.Error("expected auto mode to be false")
	}

	result = d.handleCommand(context.Background(), "auto-on")
	assertEqual(t, result.String(), "auto mode: full")

	assertAutoMode(t, d, pixy.AutoFull)
}

func TestHandleCommandAudioInvalid(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, "/dev/hidraw0")

	result := d.handleCommand(context.Background(), "audio xyz")
	if result.String() == "" || !strings.HasPrefix(result.String(), "error: audio xyz:") {
		t.Errorf("expected error starting with 'error: audio xyz:' for invalid mode, got: %s",
			result)
	}
}

func TestHandleCommandDeviceRequired(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateOffline, "", "")

	for _, cmd := range []string{cmdTrack, cmdIdle, cmdPrivacy, cmdTogglePrivacy, cmdCenter, cmdGestureOn, cmdGestureOff} {
		result := d.handleCommand(context.Background(), cmd)
		if result.String() == "" {
			t.Errorf("expected error response for '%s' with no device", cmd)
		}

		if len(result.String()) < 6 || result.String()[:6] != "error:" {
			t.Errorf("expected error: prefix for '%s' with no device, got: %s", cmd, result)
		}
	}
}

func TestWaybarOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		camera   pixy.CameraState
		inCall   bool
		expected string
	}{
		{pixy.StateTracking, false, testStrTracking},
		{pixy.StatePrivacy, false, testStrPrivacy},
		{pixy.StateIdle, false, "idle"},
		{pixy.StateOffline, false, "offline"},
		{pixy.StateTracking, true, "tracking in-call"},
	}

	for _, testCase := range tests {
		d := testDaemonWithState(t, testCase.camera, testCase.inCall)
		output := d.waybarOutput()

		var parsed map[string]string

		err := json.Unmarshal([]byte(output), &parsed)
		if err != nil {
			t.Fatalf("waybar output is not valid JSON: %s, err: %v", output, err)
		}

		if parsed["class"] != "custom-camera "+testCase.expected {
			t.Errorf(
				"expected class 'custom-camera %s', got '%s'",
				testCase.expected,
				parsed["class"],
			)
		}

		assertParsedField(t, parsed, "text")
		assertParsedField(t, parsed, "tooltip")
	}
}

func TestHandleCommandTogglePrivacy(t *testing.T) {
	t.Parallel()

	var captured []pixy.CameraState

	d := newTestDaemon(
		t,
		pixy.StatePrivacy, testVideoDev, "/dev/hidraw0",
		withCaptureTrackingSlice(&captured),
	)

	result := d.handleCommand(context.Background(), cmdTogglePrivacy)
	assertEqual(t, result.String(), respTrackingOn)

	if len(captured) != 1 || captured[0] != pixy.StateTracking {
		t.Errorf("expected tracking call with tracking, got %v", captured)
	}
}

func TestHandleCommandProbe(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateOffline, "", "")

	result := d.handleCommand(context.Background(), cmdProbe)

	if d.videoDev != "" {
		assertStatusPrefix(t, result.String(), "device found:", "PIXY connected")
	} else if result.String() != respDeviceNotFound {
		t.Errorf("expected 'device not found' when no PIXY connected, got: %s", result)
	}
}

func TestHandleCommandAudioCycleNoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, "", "")

	result := d.handleCommand(context.Background(), "audio")
	assertErrorPrefix(t, result.String())
}
