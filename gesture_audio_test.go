//go:build linux

package main

import (
	"context"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// assertAudioLive fails the test if the captured audio mode is not AudioLive.
func assertAudioLive(t *testing.T, modeArg pixy.AudioMode, scenario string) {
	t.Helper()

	if modeArg != pixy.AudioLive {
		t.Errorf("%s = %s, want %s", scenario, modeArg, pixy.AudioLive)
	}
}

func TestHandleGestureCommand_On(t *testing.T) {
	t.Parallel()

	var called, enabledArg bool

	d := newTestDaemon(t, pixy.StatePrivacy, "", "", withCaptureGesture(&called, &enabledArg))

	resp := d.handleGestureCommand(context.Background(), cmdGestureOn)
	notError(t, resp)

	if !called || !enabledArg {
		t.Errorf("setGesture called=%v enabled=%v, want true/true", called, enabledArg)
	}
}

func TestHandleGestureCommand_Off(t *testing.T) {
	t.Parallel()

	var called, enabledArg bool

	d := newTestDaemon(t, pixy.StatePrivacy, "", "", withCaptureGesture(&called, &enabledArg))

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
		t,
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
		t,
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

func TestHandleAudioCommand_SetMode(t *testing.T) {
	t.Parallel()

	var modeArg pixy.AudioMode

	d := newTestDaemon(t, pixy.StatePrivacy, "", "", withCaptureAudio(&modeArg))

	resp := d.handleAudioCommand(context.Background(), []string{cmdAudio, string(pixy.AudioLive)})
	notError(t, resp)

	assertAudioLive(t, modeArg, "setAudio called with")
}

func TestHandleAudioCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, "", "")

	resp := d.handleAudioCommand(context.Background(), []string{"audio", "invalid"})
	assertCommandContains(t, resp.String(), "error:", "response")
}

func TestHandleAudioCommand_NextMode(t *testing.T) {
	t.Parallel()

	var modeArg pixy.AudioMode

	d := newTestDaemon(t, pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.Audio = pixy.AudioNC
	}, withCaptureAudio(&modeArg))

	resp := d.handleAudioCommand(context.Background(), []string{cmdAudio})
	notError(t, resp)

	assertAudioLive(t, modeArg, "next mode")
}

func TestHandleTrackingCommand_SetTracking(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(t, pixy.StatePrivacy, "", "", withCaptureTracking(&stateArg))

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, cmdTrack)
	notError(t, resp)

	if stateArg != pixy.StateTracking {
		t.Errorf("setTracking called with %s, want %s", stateArg, pixy.StateTracking)
	}
}

func TestHandleTrackingCommand_ErrorPath(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.deps.setTracking = func(context.Context, pixy.CameraState) error {
			return ErrInvalidValue
		}
	})

	resp := d.handleTrackingCommand(context.Background(), pixy.StateTracking, cmdTrack)
	expectError(t, resp)
}

func TestHandleTogglePrivacy_FromPrivacy(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(
		t,
		pixy.StatePrivacy, "/dev/video0", "/dev/hidraw7",
		withCaptureTracking(&stateArg),
	)

	resp := d.handleTogglePrivacy(context.Background())
	notError(t, resp)

	if stateArg != pixy.StateTracking {
		t.Errorf("toggle from privacy should set tracking, got: %s", stateArg)
	}
}

func TestHandleTogglePrivacy_FromTracking(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(
		t,
		pixy.StateTracking, "/dev/video0", "/dev/hidraw7",
		withCaptureTracking(&stateArg),
	)

	resp := d.handleTogglePrivacy(context.Background())
	notError(t, resp)

	if stateArg != pixy.StatePrivacy {
		t.Errorf("toggle from tracking should set privacy, got: %s", stateArg)
	}
}

func TestHandleTogglePrivacy_FromIdle(t *testing.T) {
	t.Parallel()

	var stateArg pixy.CameraState

	d := newTestDaemon(
		t,
		pixy.StateIdle, "/dev/video0", "/dev/hidraw7",
		withCaptureTracking(&stateArg),
	)

	resp := d.handleTogglePrivacy(context.Background())
	notError(t, resp)

	if stateArg != pixy.StatePrivacy {
		t.Errorf("toggle from idle should set privacy, got: %s", stateArg)
	}
}
