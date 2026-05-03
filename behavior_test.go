//go:build linux

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// ---------------------------------------------------------------------------
// BDD-style behavioral tests: full user-facing scenarios
// ---------------------------------------------------------------------------
//
// These tests verify complete user workflows end-to-end, using the
// Given/When/Then pattern. They complement the unit tests by exercising
// multi-step flows that span multiple components.
//
// Project convention: standard testing package, no ginkgo/testify.
// BDD structure is expressed through test naming and inline comments.

// ---------------------------------------------------------------------------
// Scenario: User starts a video call with full auto mode
// ---------------------------------------------------------------------------

func TestBehavior_FullAutoCallLifecycle(t *testing.T) {
	t.Parallel()

	// Given a daemon with a connected camera in privacy mode and full auto
	var (
		setSourceCalls []string
		notifyBodies   []string
	)

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, "", func(d *Daemon) {
		d.findSourceFn = func(_ context.Context) (string, error) { return "42", nil }
		d.setSourceFn = func(_ context.Context, id string) {
			setSourceCalls = append(setSourceCalls, id)
		}
		d.notifyFn = func(_ context.Context, _, body string) {
			notifyBodies = append(notifyBodies, body)
		}
		d.isCameraInUseFn = func(string) bool { return true }
		d.config.DebounceCount = 3
	})

	// When the camera is used for exactly 3 consecutive poll cycles
	d.autoManage(context.Background())
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then the call starts, PipeWire source switches, and user is notified
	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=true after 3 debounce cycles")
	}
	if len(setSourceCalls) == 0 || setSourceCalls[0] != "42" {
		t.Errorf("expected PipeWire source switch to 42, got: %v", setSourceCalls)
	}
	if len(notifyBodies) == 0 {
		t.Error("expected desktop notification")
	}

	// When the camera is released for 3 consecutive poll cycles
	d.isCameraInUseFn = func(string) bool { return false }
	notifyBodies = nil
	d.autoManage(context.Background())
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then the call ends and privacy notification is sent
	if readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=false after camera released")
	}
	if len(notifyBodies) == 0 {
		t.Error("expected notification on call end")
	}
	if !strings.Contains(notifyBodies[0], "privacy") {
		t.Errorf("expected privacy notification, got: %s", notifyBodies[0])
	}
}

// ---------------------------------------------------------------------------
// Scenario: User changes auto mode during an active call
// ---------------------------------------------------------------------------

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
	resp := d.handleAutoCommand([]string{"auto", "off"})

	// Then auto mode is off but the call state is preserved
	if resp != respAutoModeOff {
		t.Errorf("expected 'auto mode: off', got: %s", resp)
	}
	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("InCall should still be true after auto mode change")
	}
	camera := readState(d, func(s pixy.State) pixy.CameraState { return s.Camera })
	if camera != pixy.StateTracking {
		t.Errorf("camera should still be tracking, got: %s", camera)
	}

	// When camera is released, nothing happens (auto is off)
	d.isCameraInUseFn = func(string) bool { return false }
	d.config.DebounceCount = 1
	d.autoManage(context.Background())

	if readState(d, func(s pixy.State) bool { return s.InCall }) == false {
		t.Error("InCall should remain true because auto is off (no call end handling)")
	}
}

// ---------------------------------------------------------------------------
// Scenario: Camera flip-flops prevent false call detection
// ---------------------------------------------------------------------------

func TestBehavior_DebounceFlipFlop(t *testing.T) {
	t.Parallel()

	// Given a daemon with debounce count 3
	callStarted := false
	d := testAutoDaemon(func(d *Daemon) {
		d.config.DebounceCount = 3
		d.isCameraInUseFn = func(string) bool { return false }
		d.notifyFn = func(context.Context, string, string) {
			callStarted = true
		}
	})

	// When camera is used for 2 cycles, then idle for 1, then used for 2 again
	d.isCameraInUseFn = func(string) bool { return true }
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	inUse, idle := readDebounce(d)
	if inUse != 2 || idle != 0 {
		t.Errorf("after 2 in-use polls: inUse=%d idle=%d, want 2/0", inUse, idle)
	}

	d.isCameraInUseFn = func(string) bool { return false }
	d.autoManage(context.Background())

	inUse, idle = readDebounce(d)
	if inUse != 0 || idle != 1 {
		t.Errorf("after 1 idle poll: inUse=%d idle=%d, want 0/1", inUse, idle)
	}

	// Then no call was started
	if readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("should not be in call after flip-flop")
	}
	if callStarted {
		t.Error("no notification should have been sent")
	}

	// When camera is used for 2 more cycles (not enough for debounce)
	d.isCameraInUseFn = func(string) bool { return true }
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then still no call (counter reset, only 2 of 3)
	if readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("should not be in call after only 2 cycles post-reset")
	}
}

// ---------------------------------------------------------------------------
// Scenario: PTZ values are clamped and pan/tilt use degree multiplier
// ---------------------------------------------------------------------------

func TestBehavior_PTZClampingAndMultiplier(t *testing.T) {
	t.Parallel()

	var v4l2Calls []struct{ axis, val string }
	d := newPTZDaemon(func(d *Daemon) {
		d.v4l2SetFn = func(_ context.Context, _, axis, val string) error {
			v4l2Calls = append(v4l2Calls, struct{ axis, val string }{axis, val})
			return nil
		}
	})

	// When pan is set beyond the maximum (200 → clamp to 170)
	resp := d.handlePTZCommand(context.Background(), []string{"pan", "200"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	if len(v4l2Calls) != 1 {
		t.Fatalf("expected 1 v4l2 call, got %d", len(v4l2Calls))
	}
	if v4l2Calls[0].val != "612000" {
		t.Errorf("pan 200 should clamp to 170 and multiply: got %s, want 612000", v4l2Calls[0].val)
	}

	// When tilt is set beyond minimum (-50 → clamp to -30)
	v4l2Calls = nil
	resp = d.handlePTZCommand(context.Background(), []string{"tilt", "-50"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	if v4l2Calls[0].val != "-108000" {
		t.Errorf("tilt -50 should clamp to -30 and multiply: got %s, want -108000",
			v4l2Calls[0].val)
	}

	// When zoom is set beyond maximum (500 → clamp to 400, no multiplier)
	v4l2Calls = nil
	resp = d.handlePTZCommand(context.Background(), []string{"zoom", "500"})
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	if v4l2Calls[0].val != "400" {
		t.Errorf("zoom 500 should clamp to 400 without multiplier: got %s, want 400",
			v4l2Calls[0].val)
	}
}

// ---------------------------------------------------------------------------
// Scenario: Waybar output shows complete status in tooltip
// ---------------------------------------------------------------------------

func TestBehavior_WaybarTooltipContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		camera   pixy.CameraState
		audio    pixy.AudioMode
		autoMode pixy.AutoMode
		inCall   bool
	}{
		{
			"tracking with NC in call",
			pixy.StateTracking, pixy.AudioNC, pixy.AutoFull, true,
		},
		{
			"privacy with live not in call",
			pixy.StatePrivacy, pixy.AudioLive, pixy.AutoTrackingOnly, false,
		},
		{
			"idle with original auto-off",
			pixy.StateIdle, pixy.AudioOriginal, pixy.AutoOff, false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := testDaemonWithState(tc.camera, tc.inCall)
			d.state.Audio = tc.audio
			d.state.AutoMode = tc.autoMode

			output := d.waybarOutput()
			var parsed map[string]string
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				t.Fatalf("invalid JSON: %s", output)
			}

			if !strings.Contains(parsed["tooltip"], "EMEET PIXY") {
				t.Error("tooltip should contain device name")
			}
			if !strings.Contains(parsed["tooltip"], string(tc.camera)) {
				t.Errorf("tooltip should contain camera state %s", tc.camera)
			}
			if !strings.Contains(parsed["tooltip"], string(tc.audio)) {
				t.Errorf("tooltip should contain audio mode %s", tc.audio)
			}
			if !strings.Contains(parsed["tooltip"], string(tc.autoMode)) {
				t.Errorf("tooltip should contain auto mode %s", tc.autoMode)
			}

			if tc.inCall {
				if !strings.Contains(parsed["tooltip"], "In call: yes") {
					t.Error("tooltip should show in-call status when in call")
				}
			}

			if !strings.Contains(parsed["class"], "custom-camera") {
				t.Errorf("class should start with custom-camera, got: %s", parsed["class"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario: Error during auto call start doesn't prevent InCall flag
// ---------------------------------------------------------------------------

func TestBehavior_ErrorDuringCallStart_StillSetsInCall(t *testing.T) {
	t.Parallel()

	// Given a daemon with video device but no hidraw (setDeviceState returns early)
	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, "", func(d *Daemon) {
		d.isCameraInUseFn = func(string) bool { return true }
		d.config.DebounceCount = 1
	})

	// When a call starts (tracking fails due to no real HID device)
	d.autoManage(context.Background())

	// Then InCall is still set (we don't lose the call state just because HID failed)
	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("InCall should be set even when tracking activation fails")
	}
}

// ---------------------------------------------------------------------------
// Scenario: State survives daemon restart
// ---------------------------------------------------------------------------

func TestBehavior_StateSurvivesRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	//nolint:exhaustruct
	cfg := pixy.Config{
		StateDir:      dir,
		PollInterval:  2 * time.Second,
		DebounceCount: 3,
		WebAddr:       "127.0.0.1:0",
	}

	// Given a daemon with specific state
	original := pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioLive,
		Gesture:  true,
		InCall:   true,
		AutoMode: pixy.AutoTrackingOnly,
	}

	//nolint:exhaustruct
	d1 := &Daemon{
		mu:              sync.RWMutex{},
		config:          cfg,
		state:           original,
		streamSema:      make(chan struct{}, 1),
		isCameraInUseFn: func(string) bool { return false },
		findSourceFn:    func(context.Context) (string, error) { return "", nil },
		setSourceFn:     func(context.Context, string) {},
		notifyFn:        func(context.Context, string, string) {},
	}

	if err := d1.saveState(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// When a new daemon loads from the same state dir
	//nolint:exhaustruct
	d2 := &Daemon{
		mu:              sync.RWMutex{},
		config:          cfg,
		state:           pixy.DefaultState(),
		streamSema:      make(chan struct{}, 1),
		isCameraInUseFn: func(string) bool { return false },
		findSourceFn:    func(context.Context) (string, error) { return "", nil },
		setSourceFn:     func(context.Context, string) {},
		notifyFn:        func(context.Context, string, string) {},
	}
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

// ---------------------------------------------------------------------------
// Scenario: Audio mode cycles through all modes via command
// ---------------------------------------------------------------------------

func TestBehavior_AudioCycleCompletes(t *testing.T) {
	t.Parallel()

	var audioCalls []pixy.AudioMode
	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.state.Audio = pixy.AudioNC
		d.setAudioFn = func(_ context.Context, m pixy.AudioMode) error {
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

	finalAudio := readState(d, func(s pixy.State) pixy.AudioMode { return s.Audio })
	if finalAudio != pixy.AudioNC {
		t.Errorf("after full cycle, audio should be NC, got %s", finalAudio)
	}
}

// ---------------------------------------------------------------------------
// Scenario: Privacy toggle switches between tracking and privacy via command
// ---------------------------------------------------------------------------

func TestBehavior_PrivacyToggleRoundTrip(t *testing.T) {
	t.Parallel()

	var trackingCalls []pixy.CameraState
	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.setTrackingFn = func(_ context.Context, s pixy.CameraState) error {
			d.mu.Lock()
			d.state.Camera = s
			d.mu.Unlock()
			trackingCalls = append(trackingCalls, s)
			return nil
		}
	})

	// When user toggles privacy from privacy mode → should activate tracking
	resp := d.handleCommand(context.Background(), "toggle-privacy")
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	camera := readState(d, func(s pixy.State) pixy.CameraState { return s.Camera })
	if camera != pixy.StateTracking {
		t.Errorf("after toggle from privacy, expected tracking, got %s", camera)
	}

	// When user toggles again from tracking → should enter privacy
	resp = d.handleCommand(context.Background(), "toggle-privacy")
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	camera = readState(d, func(s pixy.State) pixy.CameraState { return s.Camera })
	if camera != pixy.StatePrivacy {
		t.Errorf("after toggle from tracking, expected privacy, got %s", camera)
	}

	if len(trackingCalls) != 2 {
		t.Errorf("expected 2 tracking calls, got %d", len(trackingCalls))
	}
}

// ---------------------------------------------------------------------------
// Scenario: Auto mode persists after state save
// ---------------------------------------------------------------------------

func TestBehavior_AutoModePersistsAfterSave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		//nolint:exhaustruct
		d.config = pixy.Config{
			StateDir:      dir,
			PollInterval:  2 * time.Second,
			DebounceCount: 3,
			WebAddr:       "127.0.0.1:0",
		}
	})

	// When user sets auto mode to tracking-only
	d.handleAutoCommand([]string{"auto", "tracking-only"})

	// Then state file contains the mode
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("state file not found: %v", err)
	}
	if !strings.Contains(string(data), "tracking-only") {
		t.Errorf("state file should contain 'tracking-only', got: %s", string(data))
	}
}

// ---------------------------------------------------------------------------
// Scenario: Tracking-only auto mode sets InCall and calls notify
// ---------------------------------------------------------------------------

func TestBehavior_TrackingOnlyAutoMode(t *testing.T) {
	t.Parallel()

	var notifyMessages []string
	d := testAutoDaemon(func(d *Daemon) {
		d.state.AutoMode = pixy.AutoTrackingOnly
		d.state.Camera = pixy.StatePrivacy
		d.state.Audio = pixy.AudioLive
		d.notifyFn = func(_ context.Context, _, body string) {
			notifyMessages = append(notifyMessages, body)
		}
		d.isCameraInUseFn = func(string) bool { return true }
		d.config.DebounceCount = 1
	})

	// When camera is used
	d.autoManage(context.Background())

	// Then InCall is set and notification sent with tracking-only mode
	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=true")
	}
	if len(notifyMessages) == 0 {
		t.Error("expected notification")
	}
	if !strings.Contains(notifyMessages[0], "tracking-only") {
		t.Errorf("notification should mention tracking-only, got: %s", notifyMessages[0])
	}
}

// ---------------------------------------------------------------------------
// Scenario: Privacy-only auto mode only acts on call end
// ---------------------------------------------------------------------------

func TestBehavior_PrivacyOnlyAutoMode(t *testing.T) {
	t.Parallel()

	var notifyMessages []string
	d := testAutoDaemon(func(d *Daemon) {
		d.state.AutoMode = pixy.AutoPrivacyOnly
		d.state.Camera = pixy.StateIdle
		d.notifyFn = func(_ context.Context, _, body string) {
			notifyMessages = append(notifyMessages, body)
		}
		d.isCameraInUseFn = func(string) bool { return true }
		d.config.DebounceCount = 1
	})

	// When camera is used (call start) — privacy-only should NOT activate tracking
	d.autoManage(context.Background())

	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=true")
	}
	// Tracking activation is NOT called because privacy-only mode doesn't activate tracking
	// The camera state stays as-is (or gets set to offline because HID fails, but that's fine)

	// When camera is released (call end) — privacy should activate
	notifyMessages = nil
	d.isCameraInUseFn = func(string) bool { return false }
	d.autoManage(context.Background())

	if readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=false after call end")
	}
	if len(notifyMessages) == 0 {
		t.Error("expected notification on call end")
	}
	if !strings.Contains(notifyMessages[0], "privacy") {
		t.Errorf("notification should mention privacy, got: %s", notifyMessages[0])
	}
}
