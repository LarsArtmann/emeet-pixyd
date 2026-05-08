//go:build linux

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
// Test helpers
// ---------------------------------------------------------------------------

// postPTZFormValue posts a form-encoded PTZ value and returns the response and HTML.
func postPTZFormValue(
	t *testing.T,
	server *httptest.Server,
	path, value string,
) (*http.Response, string) {
	t.Helper()
	body := strings.NewReader("value=" + value)
	req, reqErr := http.NewRequestWithContext(
		context.Background(), http.MethodPost, server.URL+path, body,
	)
	if reqErr != nil {
		t.Fatalf("create request: %v", reqErr)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		t.Fatalf("POST %s: %v", path, respErr)
	}
	defer resp.Body.Close() //nolint:errcheck
	respBody, _ := io.ReadAll(resp.Body)
	return resp, string(respBody)
}

// assertV4L2Call asserts exactly one v4l2 call with the expected value.
func assertV4L2Call(t *testing.T, v4l2Calls []struct{ axis, val string }, wantVal string) {
	t.Helper()
	if len(v4l2Calls) != 1 {
		t.Fatalf("expected 1 v4l2 call, got %d", len(v4l2Calls))
	}
	if v4l2Calls[0].val != wantVal {
		t.Errorf("v4l2 call val = %s, want %s", v4l2Calls[0].val, wantVal)
	}
}

// assertDebounce asserts the debounce counters match the expected values.
func assertDebounce(t *testing.T, d *Daemon, wantInUse, wantIdle int) {
	t.Helper()
	inUse, idle := readDebounce(d)
	if inUse != wantInUse || idle != wantIdle {
		t.Errorf("debounce counters: inUse=%d idle=%d, want %d/%d",
			inUse, idle, wantInUse, wantIdle)
	}
}

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
		d.findSourceFn = func(_ context.Context) (pixy.SourceID, error) { return pixy.NewSourceID("42"), nil }
		d.setSourceFn = func(_ context.Context, id pixy.SourceID) {
			setSourceCalls = append(setSourceCalls, id.String())
		}
		d.notifyFn = func(_ context.Context, _, body string) {
			notifyBodies = append(notifyBodies, body)
		}
		d.isCameraInUseFn = cameraInUseFn
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
	d.isCameraInUseFn = cameraNotInUseFn
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
	resp := d.handleAutoCommand([]string{cmdAutoOff})

	// Then auto mode is off but the call state is preserved
	if resp != respAutoModeOff {
		t.Errorf("expected 'auto mode: off', got: %s", resp)
	}
	assertInCall(t, d, true)
	camera := readCameraState(d)
	if camera != pixy.StateTracking {
		t.Errorf("camera should still be tracking, got: %s", camera)
	}

	// When camera is released, nothing happens (auto is off)
	d.isCameraInUseFn = cameraNotInUseFn
	d.config.DebounceCount = 1
	d.autoManage(context.Background())

	assertInCall(t, d, true)
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
		d.isCameraInUseFn = cameraNotInUseFn
		d.notifyFn = func(context.Context, string, string) {
			callStarted = true
		}
	})

	// When camera is used for 2 cycles, then idle for 1, then used for 2 again
	d.isCameraInUseFn = cameraInUseFn
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	assertDebounce(t, d, 2, 0)

	d.isCameraInUseFn = cameraNotInUseFn
	d.autoManage(context.Background())

	assertDebounce(t, d, 0, 1)

	// Then no call was started
	assertInCall(t, d, false)
	if callStarted {
		t.Error("no notification should have been sent")
	}

	// When camera is used for 2 more cycles (not enough for debounce)
	d.isCameraInUseFn = cameraInUseFn
	d.autoManage(context.Background())
	d.autoManage(context.Background())

	// Then still no call (counter reset, only 2 of 3)
	assertInCall(t, d, false)
}

// ---------------------------------------------------------------------------
// Scenario: PTZ values are clamped and pan/tilt use degree multiplier
// ---------------------------------------------------------------------------

func TestBehavior_PTZClampingAndMultiplier(t *testing.T) {
	t.Parallel()

	d, v4l2Calls := newPTZCaptureDaemon()

	// When pan is set beyond the maximum (200 → clamp to 170)
	resp := d.handlePTZCommand(context.Background(), []string{axisPan, "200"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, "612000")

	// When tilt is set beyond minimum (-50 → clamp to -30)
	*v4l2Calls = nil
	resp = d.handlePTZCommand(context.Background(), []string{"tilt", "-50"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, "-108000")

	// When zoom is set beyond maximum (500 → clamp to 400, no multiplier)
	*v4l2Calls = nil
	resp = d.handlePTZCommand(context.Background(), []string{axisZoom, "500"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, "400")
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
		d.isCameraInUseFn = cameraInUseFn
		d.config.DebounceCount = 1
	})

	// When a call starts (tracking fails due to no real HID device)
	d.autoManage(context.Background())

	// Then InCall is still set (we don't lose the call state just because HID failed)
	assertInCall(t, d, true)
}

// ---------------------------------------------------------------------------
// Scenario: State survives daemon restart
// ---------------------------------------------------------------------------

func TestBehavior_StateSurvivesRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := testConfig(dir)

	// Given a daemon with specific state
	original := pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioLive,
		Gesture:  true,
		InCall:   true,
		AutoMode: pixy.AutoTrackingOnly,
	}

	d1 := newDaemonForStateTest(cfg, original)

	if err := d1.saveState(); err != nil {
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

	finalAudio := readAudioState(d)
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
	resp := d.handleCommand(context.Background(), cmdTogglePrivacy)
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	assertCameraState(t, d, pixy.StateTracking)

	// When user toggles again from tracking → should enter privacy
	resp = d.handleCommand(context.Background(), cmdTogglePrivacy)
	if IsCommandErrorResponse(resp) {
		t.Errorf("expected success, got: %s", resp)
	}
	assertCameraState(t, d, pixy.StatePrivacy)

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

// ---------------------------------------------------------------------------
// Scenario: Tracking-only auto mode sets InCall and calls notify
// ---------------------------------------------------------------------------

func TestBehavior_TrackingOnlyAutoMode(t *testing.T) {
	t.Parallel()

	var notifyMessages []string
	d := testAutoDaemon(withNotifyMessages(&notifyMessages), func(d *Daemon) {
		d.state.AutoMode = pixy.AutoTrackingOnly
		d.state.Camera = pixy.StatePrivacy
		d.state.Audio = pixy.AudioLive
		d.isCameraInUseFn = cameraInUseFn
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

// ---------------------------------------------------------------------------
// Scenario: Privacy-only auto mode only acts on call end
// ---------------------------------------------------------------------------

func TestBehavior_PrivacyOnlyAutoMode(t *testing.T) {
	t.Parallel()

	var notifyMessages []string
	d := testAutoDaemon(withNotifyMessages(&notifyMessages), func(d *Daemon) {
		d.state.AutoMode = pixy.AutoPrivacyOnly
		d.state.Camera = pixy.StateIdle
		d.isCameraInUseFn = cameraInUseFn
		d.config.DebounceCount = 1
	})

	// When camera is used (call start) — privacy-only should NOT activate tracking
	d.autoManage(context.Background())

	assertInCall(t, d, true)
	// Tracking activation is NOT called because privacy-only mode doesn't activate tracking
	// The camera state stays as-is (or gets set to offline because HID fails, but that's fine)

	// When camera is released (call end) — privacy should activate
	notifyMessages = nil
	d.isCameraInUseFn = cameraNotInUseFn
	d.autoManage(context.Background())

	assertInCall(t, d, false)
	if len(notifyMessages) == 0 {
		t.Error("expected notification on call end")
	}
	assertNotifyContains(t, notifyMessages, "privacy")
}

// ---------------------------------------------------------------------------
// Scenario: PTZ slider via web returns user's input value, not stale cache
// ---------------------------------------------------------------------------

func TestBehavior_PTZWebSliderReflectsUserInput(t *testing.T) {
	t.Parallel()

	// Given a daemon with a device and a web server (cache has stale pan=0)
	d := newTestDaemon(
		pixy.StateTracking,
		"/dev/video0",
		"/dev/hidraw7",
		withNoopV4L2(),
		func(d *Daemon) {
			d.ptzCache.values = pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: 100}
			d.ptzCache.expiresAt = time.Now().Add(ptzCacheTTL)
		},
	)
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	defer server.Close()

	// When user sets pan to 50 via the web interface
	resp, html := postPTZFormValue(t, server, "/api/ptz/pan", "50")
	resp.Body.Close() //nolint:errcheck

	// Then the response slider contains the user's value (50), not the stale cache value (0)
	assertCommandContains(t, html, `value="50"`, "slider response")
	if strings.Contains(html, `value="0"`) {
		t.Error("slider response should NOT contain stale cache value 0")
	}

	// And a success toast is shown
	assertCommandContains(t, html, "Pan set to 50", "response")

	// And the PTZ cache is invalidated
	d.ptzCache.mu.RLock()
	expired := time.Now().After(d.ptzCache.expiresAt)
	d.ptzCache.mu.RUnlock()
	if !expired {
		t.Error("PTZ cache should be invalidated after successful set")
	}
}

// ---------------------------------------------------------------------------
// Scenario: PTZ slider via web shows error toast on failure
// ---------------------------------------------------------------------------

func TestBehavior_PTZWebSliderShowsErrorOnFailure(t *testing.T) {
	t.Parallel()

	// Given a daemon with no device
	d := newTestDaemon(pixy.StateOffline, "", "", withNoopV4L2())
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	defer server.Close()

	// When user tries to set pan
	resp, html := postPTZFormValue(t, server, "/api/ptz/pan", "50")
	defer resp.Body.Close() //nolint:errcheck
	assertHTTPStatusOK(t, resp)

	// Then an error toast is shown
	assertCommandContains(t, html, "toast-error", "response")
	assertCommandContains(t, html, "error:", "response")
}
