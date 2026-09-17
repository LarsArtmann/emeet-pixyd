//go:build linux

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// TestReconcile_FreshStateAdoptsHardware pins the fresh-install semantics:
// with no persisted state, hardware is the source of truth and the daemon
// adopts its camera mode instead of trusting its default belief.
func TestReconcile_FreshStateAdoptsHardware(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0", withSim)
	d.hadPersistedState = false

	d.reconcileOnDeviceAppear(context.Background())

	d.mu.RLock()
	camera := d.state.Camera
	d.mu.RUnlock()

	if camera != pixy.StateIdle {
		t.Errorf("camera after fresh reconcile = %q, want %q (adopted from hardware)", camera, pixy.StateIdle)
	}

	if got := sim.Tracking(); got != pixy.StateIdle {
		t.Errorf("hardware camera = %q, want %q (fresh install must not write)", got, pixy.StateIdle)
	}

	if writes := len(sim.SentReports()); writes != 0 {
		t.Errorf("fresh reconcile wrote %d HID reports, want 0 (adopt-only)", writes)
	}
}

// TestReconcile_PersistedModeReasserted pins the persistence semantics:
// when a state file existed and the hardware reports a different camera
// mode (power cycle / replug), the persisted mode is re-asserted so
// privacy survives reboots.
func TestReconcile_PersistedModeReasserted(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StatePrivacy, "/dev/video0", "/dev/hidraw0", withSim)
	d.hadPersistedState = true

	if hw := sim.Tracking(); hw == pixy.StatePrivacy {
		t.Fatalf("test setup: hardware already in privacy, want different (got %q)", hw)
	}

	d.reconcileOnDeviceAppear(context.Background())

	d.mu.RLock()
	camera := d.state.Camera
	d.mu.RUnlock()

	if camera != pixy.StatePrivacy {
		t.Errorf("believed camera after reconcile = %q, want %q (persisted mode must win)", camera, pixy.StatePrivacy)
	}

	if got := sim.Tracking(); got != pixy.StatePrivacy {
		t.Errorf("hardware camera after reconcile = %q, want %q (re-asserted)", got, pixy.StatePrivacy)
	}
}

// TestReconcile_MatchingHardwareNoWrite verifies the re-assert is
// write-free when belief and hardware already agree.
func TestReconcile_MatchingHardwareNoWrite(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, "/dev/video0", "/dev/hidraw0", withSim)
	d.hadPersistedState = true

	d.reconcileOnDeviceAppear(context.Background())

	if writes := len(sim.SentReports()); writes != 0 {
		t.Errorf("reconcile wrote %d HID reports, want 0 (belief already matched hardware)", writes)
	}
}

// TestReconcile_QueryFailureKeepsBelief verifies the failure path: when the
// hardware cannot be queried, the daemon keeps its belief, stays silent,
// and does not crash.
func TestReconcile_QueryFailureKeepsBelief(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StatePrivacy, "/dev/video0", "/dev/hidraw0", withSim)
	d.hadPersistedState = true

	sim.sendRecvErr = errors.New("simulated query failure")

	d.reconcileOnDeviceAppear(context.Background())

	d.mu.RLock()
	camera := d.state.Camera
	d.mu.RUnlock()

	if camera != pixy.StatePrivacy {
		t.Errorf("camera after failed reconcile = %q, want %q (belief must be kept)", camera, pixy.StatePrivacy)
	}

	if got := sim.Tracking(); got == pixy.StatePrivacy {
		t.Errorf("hardware camera = %q after failed reconcile; no write should have occurred", got)
	}
}

// TestAutoOff_ManualCameraModeSurvivesTicks pins the guarantee issue #6
// asked for: with auto=off, the /proc monitor is skipped entirely and a
// manually-set camera mode survives any number of autoManage ticks.
func TestAutoOff_ManualCameraModeSurvivesTicks(t *testing.T) {
	t.Parallel()

	var monitorCalled bool

	d := newTestDaemon(
		t,
		pixy.StateTracking,
		"/dev/video0",
		"/dev/hidraw0",
		withAutoOff(),
		withNoopTracking(),
		withNoopAudio(),
	)

	d.deps.isCameraInUse = func(string) bool {
		monitorCalled = true

		return true
	}

	for range 3 {
		d.autoManage(context.Background())
	}

	d.mu.RLock()
	camera := d.state.Camera
	inCall := d.state.InCall
	d.mu.RUnlock()

	if camera != pixy.StateTracking {
		t.Errorf("camera after 3 ticks with auto=off = %q, want %q", camera, pixy.StateTracking)
	}

	if inCall {
		t.Error("in_call became true with auto=off, want untouched")
	}

	if monitorCalled {
		t.Error("/proc camera-usage monitor ran with auto=off, want it skipped entirely")
	}
}

// TestStateRoundTrip_PreservedCameraMode pins the state.json round-trip:
// a manually-set camera mode and auto=off survive a daemon restart.
func TestStateRoundTrip_PreservedCameraMode(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	first := newTestDaemon(t, pixy.StateTracking, "", "", withAutoOff())
	first.config.StateDir = stateDir
	first.state.Camera = pixy.StateTracking
	first.state.AutoMode = pixy.AutoOff

	first.mu.Lock()
	first.saveStateOrLog("save for round-trip")
	first.mu.Unlock()

	second := newTestDaemon(t, pixy.StateOffline, "", "")
	second.config.StateDir = stateDir

	second.hadPersistedState = second.loadState()
	if !second.hadPersistedState {
		t.Fatal("loadState = false, want true (state file was just written)")
	}

	if second.state.Camera != pixy.StateTracking {
		t.Errorf("camera after restart = %q, want %q", second.state.Camera, pixy.StateTracking)
	}

	if !second.state.AutoMode.IsOff() {
		t.Errorf("auto mode after restart = %q, want %q", second.state.AutoMode, pixy.AutoOff)
	}

	if !second.hadPersistedState {
		t.Error("loadState did not report persisted state")
	}
}

// TestDeviceCommandIncludesModel pins the `device` command surfacing the
// detected model next to the device paths (issue #6 support debugging).
func TestDeviceCommandIncludesModel(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateIdle, "/dev/video0", "/dev/hidraw0")
	d.mu.Lock()
	d.model = pixy.Model2K
	d.mu.Unlock()

	result := d.handleCommand(context.Background(), "device")

	if result.Err != nil {
		t.Fatalf("device command failed: %v", result.Err)
	}

	if want := "/dev/video0 /dev/hidraw0 PIXY 2K"; result.Message != want {
		t.Errorf("device output = %q, want %q", result.Message, want)
	}
}

// TestWebStatusCarriesModel pins the model surfacing in the web status.
func TestWebStatusCarriesModel(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateIdle, "/dev/video0", "")
	d.mu.Lock()
	d.model = pixy.Model2K
	d.mu.Unlock()

	status := (&webServer{daemon: d}).getWebStatus()

	if status.Model != string(pixy.Model2K) {
		t.Errorf("webStatus.Model = %q, want %q", status.Model, pixy.Model2K)
	}
}
