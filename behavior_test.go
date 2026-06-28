//go:build linux

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func assertDebounce(t *testing.T, d *Daemon, wantInUse, wantIdle int) {
	t.Helper()

	inUse, idle := readDebounce(d)
	if inUse != wantInUse || idle != wantIdle {
		t.Errorf("debounce counters: inUse=%d idle=%d, want %d/%d",
			inUse, idle, wantInUse, wantIdle)
	}
}

// assertTooltipContains fails if the waybar tooltip does not contain substring.
func assertTooltipContains(t *testing.T, parsed map[string]string, substr, label string) {
	t.Helper()

	if !strings.Contains(parsed["tooltip"], substr) {
		t.Errorf("tooltip should contain %s %q, got: %s", label, substr, parsed["tooltip"])
	}
}

func TestBehavior_FullAutoCallLifecycle(t *testing.T) {
	t.Parallel()

	// Given a daemon with a connected camera in privacy mode and full auto
	var (
		setSourceCalls []string
		notifyBodies   []string
	)

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, "", func(d *Daemon) {
		d.deps.findSource = func(_ context.Context) (pixy.SourceID, error) { return pixy.NewSourceID("42"), nil }
		d.deps.setSource = func(_ context.Context, id pixy.SourceID) {
			setSourceCalls = append(setSourceCalls, id.Get())
		}
		d.deps.notify = func(_ context.Context, _, body string) {
			notifyBodies = append(notifyBodies, body)
		}
		d.deps.isCameraInUse = cameraInUseFn
		d.config.DebounceCount = 3
	})

	// When the camera is used for exactly 3 consecutive poll cycles
	d.autoManage(context.Background())
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then the call starts, PipeWire source switches, and user is notified
	assertInCall(t, d, true)

	if len(setSourceCalls) == 0 || setSourceCalls[0] != "42" {
		t.Errorf("expected PipeWire source switch to 42, got: %v", setSourceCalls)
	}

	if len(notifyBodies) == 0 {
		t.Error("expected desktop notification")
	}

	// When the camera is released for 3 consecutive poll cycles
	d.deps.isCameraInUse = cameraNotInUseFn
	notifyBodies = nil

	d.autoManage(context.Background())
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then the call ends and privacy notification is sent
	assertInCall(t, d, false)

	if len(notifyBodies) == 0 {
		t.Error("expected notification on call end")
	}

	assertCommandContains(t, notifyBodies[0], "privacy", "notification")
}

func TestBehavior_DebounceFlipFlop(t *testing.T) {
	t.Parallel()

	// Given a daemon with debounce count 3
	callStarted := false
	d := testAutoDaemon(t, func(d *Daemon) {
		d.config.DebounceCount = 3
		d.deps.isCameraInUse = cameraNotInUseFn
		d.deps.notify = func(context.Context, string, string) {
			callStarted = true
		}
	})

	// When camera is used for 2 cycles, then idle for 1, then used for 2 again
	d.deps.isCameraInUse = cameraInUseFn
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	assertDebounce(t, d, 2, 0)

	d.deps.isCameraInUse = cameraNotInUseFn
	d.autoManage(context.Background())

	assertDebounce(t, d, 0, 1)

	// Then no call was started
	assertInCall(t, d, false)

	if callStarted {
		t.Error("no notification should have been sent")
	}

	// When camera is used for 2 more cycles (not enough for debounce)
	d.deps.isCameraInUse = cameraInUseFn
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then still no call (counter reset, only 2 of 3)
	assertInCall(t, d, false)
}

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

			d := testDaemonWithState(t, tc.camera, tc.inCall)
			d.state.Audio = tc.audio
			d.state.AutoMode = tc.autoMode

			output := d.waybarOutput()

			var parsed map[string]string

			err := json.Unmarshal([]byte(output), &parsed)
			if err != nil {
				t.Fatalf("invalid JSON: %s", output)
			}

			if !strings.Contains(parsed["tooltip"], "EMEET PIXY") {
				t.Error("tooltip should contain device name")
			}

			assertTooltipContains(t, parsed, string(tc.camera), "camera state")
			assertTooltipContains(t, parsed, string(tc.audio), "audio mode")
			assertTooltipContains(t, parsed, string(tc.autoMode), "auto mode")

			if tc.inCall {
				assertTooltipContains(t, parsed, "In call: yes", "in-call status")
			}

			if !strings.Contains(parsed["class"], "custom-camera") {
				t.Errorf("class should start with custom-camera, got: %s", parsed["class"])
			}
		})
	}
}

func TestBehavior_ErrorDuringCallStart_StillSetsInCall(t *testing.T) {
	t.Parallel()

	// Given a daemon with video device but no hidraw (setDeviceState returns early)
	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, "", func(d *Daemon) {
		d.deps.isCameraInUse = cameraInUseFn
		d.config.DebounceCount = 1
	})

	// When a call starts (tracking fails due to no real HID device)
	d.autoManage(context.Background())

	// Then InCall is still set (we don't lose the call state just because HID failed)
	assertInCall(t, d, true)
}
