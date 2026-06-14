//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestBehavior_AutoModeChangeMidCall(t *testing.T) {
	t.Parallel()

	// Given a daemon in a call with full auto mode
	d := testAutoDaemon(
		withInCall(true),
		func(d *Daemon) {
			d.state.Camera = pixy.StateTracking
			d.state.Audio = pixy.AudioNC
			d.state.AutoMode = pixy.AutoFull
		},
	)

	// When the user disables auto mode via command
	resp := d.handleAutoCommand([]string{cmdAutoOff})

	// Then auto mode is off but the call state is preserved
	if resp.String() != respAutoModeOff {
		t.Errorf("expected 'auto mode: off', got: %s", resp)
	}

	assertInCall(t, d, true)

	camera := readCameraState(d)
	if camera != pixy.StateTracking {
		t.Errorf("camera should still be tracking, got: %s", camera)
	}

	// When camera is released, nothing happens (auto is off)
	d.deps.isCameraInUse = cameraNotInUseFn
	d.config.DebounceCount = 1
	d.autoManage(context.Background())

	assertInCall(t, d, true)
}

func TestBehavior_StateSurvivesRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := defaultTestConfig(dir)

	// Given a daemon with specific state
	original := pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioLive,
		Gesture:  true,
		InCall:   true,
		AutoMode: pixy.AutoTrackingOnly,
	}

	d1 := newDaemonForStateTest(cfg, original)

	err := d1.saveState()
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// When a new daemon loads from the same state dir
	d2 := newDaemonForStateTest(cfg, pixy.DefaultState())
	d2.loadState()

	// Then all fields match the original
	if d2.state.Camera != original.Camera {
		t.Errorf("camera: got %s, want %s", d2.state.Camera, original.Camera)
	}

	if d2.state.Audio != original.Audio {
		t.Errorf("audio: got %s, want %s", d2.state.Audio, original.Audio)
	}

	if d2.state.Gesture != original.Gesture {
		t.Errorf("gesture: got %v, want %v", d2.state.Gesture, original.Gesture)
	}

	if d2.state.InCall != original.InCall {
		t.Errorf("inCall: got %v, want %v", d2.state.InCall, original.InCall)
	}

	if d2.state.AutoMode != original.AutoMode {
		t.Errorf("autoMode: got %s, want %s", d2.state.AutoMode, original.AutoMode)
	}
}

func TestBehavior_AudioCycleCompletes(t *testing.T) {
	t.Parallel()

	var audioCalls []pixy.AudioMode

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.state.Audio = pixy.AudioNC
		d.deps.setAudio = func(_ context.Context, m pixy.AudioMode) error {
			d.mu.Lock()
			d.state.Audio = m
			d.mu.Unlock()

			audioCalls = append(audioCalls, m)

			return nil
		}
	})

	// When user cycles audio 3 times (NC → Live → Original → NC)
	d.handleCommand(context.Background(), "audio")
	d.handleCommand(context.Background(), "audio")
	d.handleCommand(context.Background(), "audio")

	// Then we've cycled through all 3 modes and returned to NC
	want := []pixy.AudioMode{pixy.AudioLive, pixy.AudioOriginal, pixy.AudioNC}

	if len(audioCalls) != 3 {
		t.Fatalf("expected 3 audio calls, got %d", len(audioCalls))
	}

	for i, w := range want {
		if audioCalls[i] != w {
			t.Errorf("cycle %d: got %s, want %s", i, audioCalls[i], w)
		}
	}

	finalAudio := readAudioState(d)
	if finalAudio != pixy.AudioNC {
		t.Errorf("after full cycle, audio should be NC, got %s", finalAudio)
	}
}

func TestBehavior_PrivacyToggleRoundTrip(t *testing.T) {
	t.Parallel()

	var trackingCalls []pixy.CameraState

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.deps.setTracking = func(_ context.Context, s pixy.CameraState) error {
			d.mu.Lock()
			d.state.Camera = s
			d.mu.Unlock()

			trackingCalls = append(trackingCalls, s)

			return nil
		}
	})

	// When user toggles privacy from privacy mode → should activate tracking
	resp := d.handleCommand(context.Background(), cmdTogglePrivacy)
	if resp.IsError() {
		t.Errorf("expected success, got: %s", resp)
	}

	assertCameraState(t, d, pixy.StateTracking)

	// When user toggles again from tracking → should enter privacy
	resp = d.handleCommand(context.Background(), cmdTogglePrivacy)
	if resp.IsError() {
		t.Errorf("expected success, got: %s", resp)
	}

	assertCameraState(t, d, pixy.StatePrivacy)

	if len(trackingCalls) != 2 {
		t.Errorf("expected 2 tracking calls, got %d", len(trackingCalls))
	}
}

func TestBehavior_AutoModePersistsAfterSave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d := newTestDaemon(pixy.StatePrivacy, "", "", withConfig(dir))

	// When user sets auto mode to tracking-only
	d.handleAutoCommand([]string{"auto", "tracking-only"})

	// Then state file contains the mode
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("state file not found: %v", err)
	}

	assertCommandContains(t, string(data), "tracking-only", "state file")
}

func TestBehavior_TrackingOnlyAutoMode(t *testing.T) {
	t.Parallel()

	var notifyMessages []string

	d := testAutoDaemon(withNotifyMessages(&notifyMessages), func(d *Daemon) {
		d.state.AutoMode = pixy.AutoTrackingOnly
		d.state.Camera = pixy.StatePrivacy
		d.state.Audio = pixy.AudioLive
		d.deps.isCameraInUse = cameraInUseFn
		d.config.DebounceCount = 1
	})

	// When camera is used
	d.autoManage(context.Background())

	// Then InCall is set and notification sent with tracking-only mode
	assertInCall(t, d, true)

	if len(notifyMessages) == 0 {
		t.Error("expected notification")
	}

	assertNotifyContains(t, notifyMessages, "tracking-only")
}

func TestBehavior_PrivacyOnlyAutoMode(t *testing.T) {
	t.Parallel()

	var notifyMessages []string

	d := testAutoDaemon(withNotifyMessages(&notifyMessages), func(d *Daemon) {
		d.state.AutoMode = pixy.AutoPrivacyOnly
		d.state.Camera = pixy.StateIdle
		d.deps.isCameraInUse = cameraInUseFn
		d.config.DebounceCount = 1
	})

	// When camera is used (call start) — privacy-only should NOT activate tracking
	d.autoManage(context.Background())

	assertInCall(t, d, true)
	// Tracking activation is NOT called because privacy-only mode doesn't activate tracking
	// The camera state stays as-is (or gets set to offline because HID fails, but that's fine)

	// When camera is released (call end) — privacy should activate
	notifyMessages = nil
	d.deps.isCameraInUse = cameraNotInUseFn
	d.autoManage(context.Background())

	assertInCall(t, d, false)

	if len(notifyMessages) == 0 {
		t.Error("expected notification on call end")
	}

	assertNotifyContains(t, notifyMessages, "privacy")
}
