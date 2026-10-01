//go:build linux

package main

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// applyProbe drives a probe result into the daemon the way production does
// (write lock held for applyProbeResultLocked).
func applyProbe(d *Daemon, r probeResult) {
	d.mu.Lock()
	d.applyProbeResultLocked(r)
	d.mu.Unlock()
}

// TestIntentSurvivesProbeMissAndReplug is the Finding 1 regression property:
// no sequence of probe miss → appear → reconcile may overwrite the user's
// persisted camera intent. Connectivity is observed, never written.
func TestIntentSurvivesProbeMissAndReplug(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev, withSim)
	d.persistedIntent.Store(true)

	// Device disappears: presence drops, intent must not.
	applyProbe(d, probeResult{})

	if got := readCameraState(d); got != pixy.StatePrivacy {
		t.Fatalf("intent after probe miss = %q, want %q", got, pixy.StatePrivacy)
	}

	if d.presence().Online {
		t.Error("presence().Online = true after probe miss, want false")
	}

	// Device reappears: intent still wins; reconcile re-asserts it on hardware.
	applyProbe(d, probeResult{VideoDev: testVideoDev, HidrawDev: testHIDDev})

	// applyProbeResultLocked installs a real hidraw handle for the reappeared
	// node; point it back at the simulator so reconcile has a working transport.
	d.mu.Lock()
	d.hidDev = sim
	d.mu.Unlock()

	if got := readCameraState(d); got != pixy.StatePrivacy {
		t.Fatalf("intent after replug = %q, want %q", got, pixy.StatePrivacy)
	}

	if !d.presence().Online || !d.presence().Controllable {
		t.Errorf("presence after replug = %+v, want online+controllable", d.presence())
	}

	d.hidMu.Lock()
	d.reconcileOnDeviceAppear(context.Background())
	d.hidMu.Unlock()

	if got := sim.Tracking(); got != pixy.StatePrivacy {
		t.Errorf("hardware camera after replug reconcile = %q, want %q (intent re-asserted)", got, pixy.StatePrivacy)
	}
}

// TestProbeResultNeverWritesIntent is a direct guard on the fixed root cause:
// applyProbeResultLocked must not touch the camera field for any presence
// transition.
func TestProbeResultNeverWritesIntent(t *testing.T) {
	t.Parallel()

	for _, wants := range []pixy.CameraState{pixy.StatePrivacy, pixy.StateTracking, pixy.StateIdle} {
		d := newTestDaemon(t, wants, "", "")

		applyProbe(d, probeResult{VideoDev: testVideoDev, HidrawDev: testHIDDev})
		applyProbe(d, probeResult{})
		applyProbe(d, probeResult{VideoDev: testVideoDev, HidrawDev: testHIDDev})

		if got := readCameraState(d); got != wants {
			t.Errorf("intent = %q, want %q (probe transitions must not write it)", got, wants)
		}
	}
}

// TestLoadState_NormalizesLegacyOffline pins the v1→v2 migration: a state file
// that persisted the connectivity value "offline" as the camera mode is
// normalized to privacy, so a past outage can never masquerade as intent.
func TestLoadState_NormalizesLegacyOffline(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	legacy := `{"v":1,"camera":"offline","audio":"nc","gesture":false,"inCall":false,"autoMode":"full"}`

	if err := os.WriteFile(stateDir+"/state.json", []byte(legacy), 0o600); err != nil {
		t.Fatalf("seed legacy state: %v", err)
	}

	d := newTestDaemon(t, pixy.StateTracking, "", "")
	d.config.StateDir = stateDir

	if !d.loadState() {
		t.Fatal("loadState = false, want true for a valid legacy file")
	}

	if got := readCameraState(d); got != pixy.StatePrivacy {
		t.Errorf("legacy offline normalized to %q, want %q", got, pixy.StatePrivacy)
	}
}

// TestProbeResultInvalidatesCachesOnPresenceChange pins Finding 7: cached PTZ
// and battery readings (and the last MJPEG frame) belong to a specific device,
// so a presence transition must drop them while an unchanged presence keeps
// them.
func TestProbeResultInvalidatesCachesOnPresenceChange(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev)

	d.ptzCache.Set(pixy.PTZValues{Pan: 10}, ptzCacheTTL)
	d.powerCache.Set(
		powerCacheEntry{reading: powerReading{Level: 42, Charging: false}, available: true},
		powerCacheTTL,
	)
	d.lastFrame.Set([]byte{0xFF, 0xD8})

	applyProbe(d, probeResult{VideoDev: testVideoDev, HidrawDev: testHIDDev})

	if _, ok := d.ptzCache.Get(); !ok {
		t.Error("ptzCache invalidated without a presence change")
	}

	if _, ok := d.powerCache.Get(); !ok {
		t.Error("powerCache invalidated without a presence change")
	}

	applyProbe(d, probeResult{})

	if _, ok := d.ptzCache.Get(); ok {
		t.Error("ptzCache not invalidated on device removal")
	}

	if _, ok := d.powerCache.Get(); ok {
		t.Error("powerCache not invalidated on device removal")
	}

	if frame := d.lastFrame.Get(); len(frame) != 0 {
		t.Errorf("lastFrame not cleared on device removal: %d bytes", len(frame))
	}
}

// TestSimulatorInFlightDetection guards the concurrency proof against being
// vacuous: the high-water mark must rise on overlap and never fall back.
func TestSimulatorInFlightDetection(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	sim.trackEnter()
	sim.trackEnter()

	if got := sim.MaxInFlight(); got != 2 {
		t.Errorf("MaxInFlight after two overlapping entries = %d, want 2", got)
	}

	sim.trackExit()
	sim.trackExit()

	if got := sim.MaxInFlight(); got != 2 {
		t.Errorf("MaxInFlight after drain = %d, want 2 (high-water mark must persist)", got)
	}
}

// TestHIDSerialization_ConcurrentSyncAndAutoManage is the Finding 3 proof:
// every HID transport path is serialized under hidMu, so no two transport
// calls are ever in flight at once (a mis-paired hidraw read would be silently
// accepted by the interface-byte router otherwise).
func TestHIDSerialization_ConcurrentSyncAndAutoManage(t *testing.T) {
	t.Parallel()

	sim, withSim := withPixySimulator()
	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev, withSim)

	var wait sync.WaitGroup

	for range 10 {
		wait.Add(2)

		go func() {
			defer wait.Done()

			d.syncState(context.Background())
		}()

		go func() {
			defer wait.Done()

			d.autoManage(context.Background())
		}()
	}

	wait.Wait()

	if got := sim.MaxInFlight(); got > 1 {
		t.Errorf("observed %d concurrent HID transport calls, want <= 1 (hidMu must serialize)", got)
	}
}
