//go:build linux

package main

import (
	"context"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// withShortReassertDelay shortens the privacy-trap re-assert delay for the
// duration of one test and restores it after. Tests touching it run serially
// (no t.Parallel) because the delay is package state.
func withShortReassertDelay(tb testing.TB, delay time.Duration) {
	tb.Helper()

	original := privacyTrapReassertDelay
	privacyTrapReassertDelay = delay

	tb.Cleanup(func() { privacyTrapReassertDelay = original })
}

// countTrackingConfigs counts 9-byte config reports for the tracking
// interface carrying the given mode byte (commits are 4 bytes).
func countTrackingConfigs(sim *pixySimulator, modeByte byte) int {
	count := 0

	for _, report := range sim.SentReports() {
		if len(report) == hidMinLen &&
			report[0] == cameraConfigPrefix && report[1] == hidInterfaceTracking &&
			report[8] == modeByte {
			count++
		}
	}

	return count
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal(msg)
}

// TestSetTracking_ReassertsModeAfterTrapRace pins the core fix: a track/idle
// write issued while the privacy-trap window is armed gets ONE deferred
// re-assert, so the user's click lands even when the firmware eats the
// original write mid-transition.
//nolint:paralleltest // mutates the package-level re-assert delay
func TestSetTracking_ReassertsModeAfterTrapRace(t *testing.T) {
	withShortReassertDelay(t, 30*time.Millisecond)

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, withSim)

	d.armPrivacyTrap()

	if err := d.setTracking(context.Background(), pixy.StateTracking); err != nil {
		t.Fatalf("setTracking: %v", err)
	}

	waitForCondition(t, 2*time.Second,
		func() bool { return countTrackingConfigs(sim, hidByteTracking) == 2 },
		"expected exactly one deferred re-assert write after the trap race")

	if got := sim.Tracking(); got != pixy.StateTracking {
		t.Errorf("simulator mode after re-assert = %s, want tracking", got)
	}

	if got := readCameraState(d); got != pixy.StateTracking {
		t.Errorf("believed mode after re-assert = %s, want tracking", got)
	}
}

// TestSetTracking_NoReassertOutsideTrapWindow: without an armed trap window
// a mode write goes out exactly once — the re-assert never fires for normal
// clicks.
//nolint:paralleltest // mutates the package-level re-assert delay
func TestSetTracking_NoReassertOutsideTrapWindow(t *testing.T) {
	withShortReassertDelay(t, 30*time.Millisecond)

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, withSim)

	if err := d.setTracking(context.Background(), pixy.StateTracking); err != nil {
		t.Fatalf("setTracking: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	if got := countTrackingConfigs(sim, hidByteTracking); got != 1 {
		t.Errorf("tracking config writes = %d, want exactly 1 (no trap armed)", got)
	}
}

// TestSetTracking_ReassertSkippedAfterIntentChange: when the user's intent
// moves on before the re-assert fires (privacy clicked meanwhile), the stale
// re-assert is dropped instead of fighting the newer command.
//nolint:paralleltest // mutates the package-level re-assert delay
func TestSetTracking_ReassertSkippedAfterIntentChange(t *testing.T) {
	withShortReassertDelay(t, 500*time.Millisecond)

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, withSim)

	d.armPrivacyTrap()

	if err := d.setTracking(context.Background(), pixy.StateTracking); err != nil {
		t.Fatalf("setTracking(tracking): %v", err)
	}

	if err := d.setTracking(context.Background(), pixy.StatePrivacy); err != nil {
		t.Fatalf("setTracking(privacy): %v", err)
	}

	// The re-assert fires 500ms after the tracking write — by then the
	// privacy write (>=200ms config+commit) has completed and belief moved on.
	time.Sleep(1200 * time.Millisecond)

	if got := countTrackingConfigs(sim, hidByteTracking); got != 1 {
		t.Errorf("tracking config writes = %d, want 1 (re-assert must be skipped)", got)
	}

	if got := countTrackingConfigs(sim, hidBytePrivacy); got != 1 {
		t.Errorf("privacy config writes = %d, want 1", got)
	}

	if got := sim.Tracking(); got != pixy.StatePrivacy {
		t.Errorf("simulator mode = %s, want privacy", got)
	}
}

// TestSetTracking_PrivacyWriteNeverReasserts: entering privacy is never
// racing the trap cover — the deferred re-assert applies to uncovering
// writes only.
//nolint:paralleltest // mutates the package-level re-assert delay
func TestSetTracking_PrivacyWriteNeverReasserts(t *testing.T) {
	withShortReassertDelay(t, 30*time.Millisecond)

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, withSim)

	d.armPrivacyTrap()

	if err := d.setTracking(context.Background(), pixy.StatePrivacy); err != nil {
		t.Fatalf("setTracking(privacy): %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	if got := countTrackingConfigs(sim, hidBytePrivacy); got != 1 {
		t.Errorf("privacy config writes = %d, want exactly 1", got)
	}
}

// TestHandleCommand_TrackDuringTrapSchedulesReassert covers the full command
// routing: the web UI's POST /api/track lands in handleCommand, whose
// hidMu-scoped write must not deadlock against the re-assert goroutine taking
// hidMu afterwards.
//nolint:paralleltest // mutates the package-level re-assert delay
func TestHandleCommand_TrackDuringTrapSchedulesReassert(t *testing.T) {
	withShortReassertDelay(t, 30*time.Millisecond)

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, withSim)

	d.armPrivacyTrap()

	result := d.handleCommand(context.Background(), cmdTrack)
	if result.IsError() {
		t.Fatalf("track command failed: %v", result)
	}

	waitForCondition(t, 2*time.Second,
		func() bool { return countTrackingConfigs(sim, hidByteTracking) == 2 },
		"expected the re-assert write through the full command path")
}

// TestHandlePTZCommand_ArmsPrivacyTrap pins which moves arm the race window.
func TestHandlePTZCommand_ArmsPrivacyTrap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cmd  string
		arm  bool
	}{
		{"tilt at trap boundary", "tilt -85", true},
		{"tilt below trap boundary", "tilt -90", true},
		{"tilt just above trap boundary", "tilt -84", false},
		{"tilt mid range", "tilt -20", false},
		{"pan move does not arm", "pan -100", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev,
				withCaptureV4L2(&[]v4l2Call{}), withNoopParsePTZ())

			if result := d.handleCommand(context.Background(), tc.cmd); result.IsError() {
				t.Fatalf("%s failed: %v", tc.cmd, result)
			}

			if got := d.privacyTrapArmed(); got != tc.arm {
				t.Errorf("privacyTrapArmed() after %q = %v, want %v", tc.cmd, got, tc.arm)
			}
		})
	}
}

// TestSchedulePTZReadback_RearmsTrapOnArrival: the delayed hardware readback
// re-arms the window when the head actually landed in the trap zone — slow
// travels from far positions keep the protection alive.
func TestSchedulePTZReadback_RearmsTrapOnArrival(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, withNoopParsePTZ())

	d.deps.parsePTZ = func(_ context.Context, _ string) pixy.PTZValues {
		return pixy.PTZValues{Pan: 0, Tilt: -88, Zoom: pixy.ZoomDefault}
	}

	d.schedulePTZReadback(context.Background(), testVideoDev)

	waitForCondition(t, 2*time.Second, d.privacyTrapArmed,
		"expected the readback to re-arm the trap window for tilt -88")
}

// TestPrivacyTrapArmedWindowExpiry pins that the armed state is a window,
// not a latch.
func TestPrivacyTrapArmedWindowExpiry(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, withNoopParsePTZ())

	d.armPrivacyTrap()

	if !d.privacyTrapArmed() {
		t.Fatal("trap must be armed right after armPrivacyTrap")
	}

	d.mu.Lock()
	d.trapArmedUntil = time.Now().Add(-time.Second)
	d.mu.Unlock()

	if d.privacyTrapArmed() {
		t.Error("trap must not stay armed past its window")
	}
}
