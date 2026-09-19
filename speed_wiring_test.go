//go:build linux

package main

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the #138/#140/#141 wiring: persisted motor speeds re-asserted on
// every move path, the tracking variant surviving a daemon restart, and the
// preset-push web surface.

// --- #138: speed persistence ---

func TestSpeedCommand_PersistsAcrossRestart(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	sim, opt := withPixySimulator()
	first := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)
	first.config.StateDir = stateDir

	if result := first.handleCommand(t.Context(), "speed pan 50"); result.IsError() {
		t.Fatalf("speed pan 50 failed: %s", result.String())
	}

	if got := sim.MotorSpeed(pixy.MotorPan); got != 50 {
		t.Fatalf("simulator pan speed = %v, want 50 (command applied)", got)
	}

	second := newTestDaemon(t, pixy.StateTracking, "", "")

	second.config.StateDir = stateDir
	if !second.loadState() {
		t.Fatal("loadState = false, want true (state file was just written)")
	}

	if got := readState(second, func(s pixy.State) float32 { return s.Speeds.Pan }); got != 50 {
		t.Errorf("persisted pan speed after restart = %v, want 50", got)
	}

	if got := readState(second, func(s pixy.State) float32 { return s.Speeds.Tilt }); got != 0 {
		t.Errorf("unset tilt speed after restart = %v, want 0 (no preference)", got)
	}

	// The persisted speeds must reach the webStatus speed sliders.
	status := (&webServer{daemon: second}).getWebStatus(t.Context())
	if status.Speeds.Pan != 50 {
		t.Errorf("webStatus speed pan = %v, want 50 (sliders must not lie)", status.Speeds.Pan)
	}
}

// --- #138: re-assert on moves ---

func TestPTZMove_ReassertsConfiguredSpeed(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()

	var v4l2Calls []v4l2Call

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt, withCaptureV4L2(&v4l2Calls))

	if result := d.handleCommand(t.Context(), "speed pan 50"); result.IsError() {
		t.Fatalf("speed pan 50 failed: %s", result.String())
	}

	baseline := len(sim.SentReports())

	if result := d.handleCommand(t.Context(), "pan 30"); result.IsError() {
		t.Fatalf("pan 30 failed: %s", result.String())
	}

	reports := sim.SentReports()
	if got := len(reports) - baseline; got != 1 {
		t.Fatalf("pan move sent %d extra HID reports, want 1 (the speed re-assert)", got)
	}

	wantHead := pixy.V2SetMotorSpeed.WithIface(pixy.MotorMCUIface).Bytes()
	if string(reports[baseline][:4]) != string(wantHead) {
		t.Errorf("re-assert head = %x, want %x", reports[baseline][:4], wantHead)
	}

	if got := sim.MotorSpeed(pixy.MotorPan); got != 50 {
		t.Errorf("simulator pan speed = %v, want 50", got)
	}

	if len(v4l2Calls) != 1 || v4l2Calls[0].ctrl != "pan_absolute" {
		t.Errorf("v4l2 calls = %+v, want one pan_absolute set", v4l2Calls)
	}
}

func TestPTZMove_NoSpeedPreference_NoHIDReport(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()

	var v4l2Calls []v4l2Call

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt, withCaptureV4L2(&v4l2Calls))

	if result := d.handleCommand(t.Context(), "pan 30"); result.IsError() {
		t.Fatalf("pan 30 failed: %s", result.String())
	}

	if got := len(sim.SentReports()); got != 0 {
		t.Errorf("pan move without speeds sent %d HID reports, want 0", got)
	}
}

func TestPresetLoad_ReassertsConfiguredSpeeds(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()

	var v4l2Calls []v4l2Call

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt, withCaptureV4L2(&v4l2Calls))

	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{"home": {Pan: 30, Tilt: -10, Zoom: 120}}
	d.state.Speeds = pixy.SpeedValues{}.Set(pixy.AxisPan, 40).Set(pixy.AxisTilt, 60)
	d.mu.Unlock()

	if result := d.handleCommand(t.Context(), "preset load home"); result.IsError() {
		t.Fatalf("preset load home failed: %s", result.String())
	}

	reports := sim.SentReports()
	if len(reports) != 2 {
		t.Fatalf("preset load sent %d HID reports, want 2 (pan+tilt speed, zoom unset)", len(reports))
	}

	wantHead := pixy.V2SetMotorSpeed.WithIface(pixy.MotorMCUIface).Bytes()
	for i, motor := range []pixy.MotorType{pixy.MotorPan, pixy.MotorTilt} {
		if string(reports[i][:4]) != string(wantHead) || reports[i][4] != byte(motor) {
			t.Errorf("report %d = %x, want SetMotorSpeed for %s", i, reports[i], motor)
		}
	}

	if len(v4l2Calls) != 3 {
		t.Errorf("v4l2 calls = %d, want 3 (pan/tilt/zoom)", len(v4l2Calls))
	}
}

func TestCenter_ReassertsConfiguredSpeeds(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	centerCalls := 0
	d := newTestDaemon(
		t, pixy.StateTracking, testVideoDev, testHIDDev, opt,
		withCaptureCenter(&centerCalls),
	)

	d.mu.Lock()
	d.state.Speeds = pixy.SpeedValues{}.Set(pixy.AxisZoom, 25)
	d.mu.Unlock()

	if result := d.handleCommand(t.Context(), "center"); result.IsError() {
		t.Fatalf("center failed: %s", result.String())
	}

	reports := sim.SentReports()
	if len(reports) != 1 || reports[0][4] != byte(pixy.MotorZoom) {
		t.Fatalf("center reports = %x, want one zoom SetMotorSpeed", reports)
	}

	if centerCalls != 1 {
		t.Errorf("center calls = %d, want 1", centerCalls)
	}
}

func TestPresetPush_ReassertsConfiguredSpeedsBeforeMoves(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{"home": {Pan: 30, Tilt: -10, Zoom: 120}}
	d.state.Speeds = pixy.SpeedValues{}.Set(pixy.AxisPan, 15).Set(pixy.AxisTilt, 25).Set(pixy.AxisZoom, 35)
	d.mu.Unlock()

	if result := d.handleCommand(t.Context(), "preset push home"); result.IsError() {
		t.Fatalf("preset push home failed: %s", result.String())
	}

	reports := sim.SentReports()

	// 3 speed re-asserts + 3 SetMotorPos + 1 SetMotorPresetPos.
	if len(reports) != 7 {
		t.Fatalf("preset push sent %d reports, want 7 (3 speeds, 3 moves, 1 save)", len(reports))
	}

	speedHead := pixy.V2SetMotorSpeed.WithIface(pixy.MotorMCUIface).Bytes()
	for i, motor := range []pixy.MotorType{pixy.MotorPan, pixy.MotorTilt, pixy.MotorZoom} {
		if string(reports[i][:4]) != string(speedHead) || reports[i][4] != byte(motor) {
			t.Errorf("speed report %d = %x, want SetMotorSpeed for %s", i, reports[i], motor)
		}
	}
}

func TestReconcile_ReassertsPersistedSpeeds(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, "/dev/video0", "/dev/hidraw0", withSim)
	d.hadPersistedState = true

	d.mu.Lock()
	d.state.Speeds = pixy.SpeedValues{}.Set(pixy.AxisPan, 55)
	d.mu.Unlock()

	d.reconcileOnDeviceAppear(t.Context())

	if got := sim.MotorSpeed(pixy.MotorPan); got != 55 {
		t.Errorf("simulator pan speed after reconcile = %v, want 55 (re-asserted)", got)
	}

	// The camera mode already matched, so the ONLY Send must be the speed.
	if got := len(sim.SentReports()); got != 1 {
		t.Errorf("reconcile sent %d HID reports, want 1 (speed re-assert only)", got)
	}
}

// --- #140: tracking variant persistence ---

func TestTrackingVariant_PersistedAcrossRestart(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	sim, opt := withPixySimulator()
	first := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)
	first.config.StateDir = stateDir

	if result := first.handleCommand(t.Context(), "tracking fullbody"); result.IsError() {
		t.Fatalf("tracking fullbody failed: %s", result.String())
	}

	if mode, _ := sim.TargetTrack(); mode != byte(pixy.TrackFullBody) {
		t.Fatalf("simulator variant = %d, want %d (command applied)", mode, pixy.TrackFullBody)
	}

	// Online + tracking so the panel renders the variant picker.
	second := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev)

	second.config.StateDir = stateDir
	if !second.loadState() {
		t.Fatal("loadState = false, want true (state file was just written)")
	}

	if got := readState(second, func(s pixy.State) string { return s.TrackMode }); got != pixy.TrackFullBody.String() {
		t.Errorf("persisted track mode after restart = %q, want %q", got, pixy.TrackFullBody)
	}

	// The web picker must show the persisted variant, not reset to face.
	status := (&webServer{daemon: second}).getWebStatus(t.Context())
	if status.TrackMode != pixy.TrackFullBody.String() {
		t.Errorf("webStatus track mode = %q, want %q (picker must not lie)", status.TrackMode, pixy.TrackFullBody)
	}

	server := newTestWebServer(t, second)

	body := getPanelBody(t, server)
	if !strings.Contains(body, "/api/tracking/fullbody") {
		t.Error("panel does not render the fullbody variant segment")
	}
}

// --- #141: preset push web surface ---

func TestWebPresetPushEndpoint(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{"home": {Pan: 30, Tilt: -10, Zoom: 120}}
	d.mu.Unlock()

	server := newTestWebServer(t, d)

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		server.URL+"/api/preset/push/home",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	wantSaveHead := pixy.V2SetMotorPresetPos.WithIface(pixy.MotorMCUIface).Bytes()
	reports := sim.SentReports()

	if len(reports) != 4 || string(reports[3][:4]) != string(wantSaveHead) {
		t.Errorf("reports = %x, want 3 moves + slot save (no speeds configured)", reports)
	}
}

func TestWebPresetChip_RendersConfirmedPush(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, func(d *Daemon) {
		d.state.Presets = pixy.NewPresetMap()
		d.state.Presets["desk"] = pixy.PTZValues{Pan: 30, Tilt: -10, Zoom: 120}
	})
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		"preset-chip-push",
		"/api/preset/push/desk",
		"confirm(",
		"The camera will physically move",
		`aria-label="Push preset desk to camera hardware slot"`,
	})
}

// --- #138: lock order regression (v4l2Mu → hidMu) ---

// TestLockOrder_V4L2MoveWithHIDCommands pins the global lock order
// v4l2Mu → hidMu: PTZ move paths hold v4l2Mu and take hidMu underneath
// (reassertSpeeds), while HID-only commands take hidMu alone. If any path
// ever nests the locks the other way around, this mixed workload deadlocks
// and the watchdog fails the test instead of hanging the suite.
func TestLockOrder_V4L2MoveWithHIDCommands(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()

	var v4l2Calls []v4l2Call

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt, withCaptureV4L2(&v4l2Calls))

	// A persisted speed makes every move path nest hidMu under v4l2Mu.
	if result := d.handleCommand(t.Context(), "speed pan 40"); result.IsError() {
		t.Fatalf("speed pan 40: %s", result.String())
	}

	commands := []string{
		"pan 12", "tilt rel 5", "zoom 110",
		"speed tilt 30", "tracking halfbody", "status",
	}

	var wg sync.WaitGroup

	done := make(chan struct{})

	for worker := range 3 {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			for i := range 25 {
				_ = d.handleCommand(t.Context(), commands[(n+i)%len(commands)])
			}
		}(worker)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("mixed V4L2/HID workload deadlocked — lock order v4l2Mu → hidMu violated")
	}

	if got := sim.MotorSpeed(pixy.MotorTilt); got != 30 {
		t.Errorf("tilt speed = %v, want 30 (HID commands must still land)", got)
	}
}
