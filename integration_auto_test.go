//go:build linux

package main

import (
	"context"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// TestIntegration_AutoManageFullLifecycle exercises the complete auto-manage
// state machine: idle → call-start (tracking+audio+source) → call-end (privacy),
// using the fake device harness so no real hardware is touched.
func TestIntegration_AutoManageFullLifecycle(t *testing.T) {
	t.Parallel()

	var trackingCalls []pixy.CameraState

	var audioCalls []pixy.AudioMode

	sourceSet := false

	cameraInUse := false

	d := newTestDaemon(
		t,
		pixy.StatePrivacy,
		testVideoDev,
		testHIDDev,
		withDebounceCount(),
		withFakeDevices(),
		withFindSource("pixy-source-42"),
		func(d *Daemon) {
			d.deps.isCameraInUse = func(_ string) bool { return cameraInUse }

			d.deps.setTracking = func(_ context.Context, state pixy.CameraState) error {
				d.mu.Lock()
				d.state.Camera = state
				d.mu.Unlock()

				trackingCalls = append(trackingCalls, state)

				return nil
			}

			d.deps.setAudio = func(_ context.Context, mode pixy.AudioMode) error {
				d.mu.Lock()
				d.state.Audio = mode
				d.mu.Unlock()

				audioCalls = append(audioCalls, mode)

				return nil
			}

			d.deps.setSource = func(_ context.Context, _ pixy.SourceID) {
				sourceSet = true
			}
		},
	)

	ctx := context.Background()

	// Phase 1: camera idle, debounce should accumulate but not trigger
	for range 2 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, false)
	assertCameraState(t, d, pixy.StatePrivacy)

	if len(trackingCalls) != 0 {
		t.Fatalf("expected 0 tracking calls before debounce, got %d", len(trackingCalls))
	}

	// Phase 2: camera in use, debounce (3 polls) then call start
	cameraInUse = true

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, true)
	assertCameraState(t, d, pixy.StateTracking)

	if len(trackingCalls) != 1 || trackingCalls[0] != pixy.StateTracking {
		t.Errorf("expected tracking activation, got %v", trackingCalls)
	}

	if len(audioCalls) != 1 || audioCalls[0] != pixy.AudioNC {
		t.Errorf("expected NC audio activation, got %v", audioCalls)
	}

	if !sourceSet {
		t.Error("expected PipeWire source to be set on call start")
	}

	// Phase 3: camera released, debounce (3 polls) then call end → privacy
	cameraInUse = false

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, false)
	assertCameraState(t, d, pixy.StatePrivacy)

	if len(trackingCalls) != 2 || trackingCalls[1] != pixy.StatePrivacy {
		t.Errorf("expected privacy after call end, got %v", trackingCalls)
	}

	// Phase 4: rapid flip-flop — camera in use again
	cameraInUse = true

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, true)
	assertCameraState(t, d, pixy.StateTracking)
}

// TestIntegration_AutoManageTrackingOnlyMode verifies that tracking-only mode
// activates tracking but does NOT switch the PipeWire source.
func TestIntegration_AutoManageTrackingOnlyMode(t *testing.T) {
	t.Parallel()

	sourceSet := false

	cameraInUse := false

	d := newTestDaemon(
		t,
		pixy.StatePrivacy,
		testVideoDev,
		testHIDDev,
		withDebounceCount(),
		withFakeDevices(),
		func(d *Daemon) {
			d.state.AutoMode = pixy.AutoTrackingOnly

			d.deps.isCameraInUse = func(_ string) bool { return cameraInUse }

			d.deps.setSource = func(_ context.Context, _ pixy.SourceID) {
				sourceSet = true
			}
		},
	)

	ctx := context.Background()

	cameraInUse = true

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, true)
	assertCameraState(t, d, pixy.StateTracking)

	if sourceSet {
		t.Error("tracking-only mode should NOT switch PipeWire source")
	}

	// Call end → privacy
	cameraInUse = false

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, false)
	assertCameraState(t, d, pixy.StatePrivacy)
}

// TestIntegration_AutoManagePrivacyOnlyMode verifies that privacy-only mode
// does NOT activate tracking on call start, but DOES enter privacy on call end.
func TestIntegration_AutoManagePrivacyOnlyMode(t *testing.T) {
	t.Parallel()

	var trackingCalls []pixy.CameraState

	cameraInUse := false

	d := newTestDaemon(
		t,
		pixy.StatePrivacy,
		testVideoDev,
		testHIDDev,
		withDebounceCount(),
		withFakeDevices(),
		func(d *Daemon) {
			d.state.AutoMode = pixy.AutoPrivacyOnly

			d.deps.isCameraInUse = func(_ string) bool { return cameraInUse }

			d.deps.setTracking = func(_ context.Context, state pixy.CameraState) error {
				d.mu.Lock()
				d.state.Camera = state
				d.mu.Unlock()

				trackingCalls = append(trackingCalls, state)

				return nil
			}
		},
	)

	ctx := context.Background()

	cameraInUse = true

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, true)
	assertCameraState(t, d, pixy.StatePrivacy)

	if len(trackingCalls) != 0 {
		t.Errorf("privacy-only mode should NOT activate tracking on call start, got %v", trackingCalls)
	}

	cameraInUse = false

	for range 3 {
		d.autoManage(ctx)
	}

	assertInCall(t, d, false)
	assertCameraState(t, d, pixy.StatePrivacy)

	if len(trackingCalls) != 1 || trackingCalls[0] != pixy.StatePrivacy {
		t.Errorf("expected privacy activation on call end, got %v", trackingCalls)
	}
}
