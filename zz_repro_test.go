//go:build linux

package main

import (
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestZZRepro_ProbeMissClobbersIntent(t *testing.T) {
	d := newTestDaemon(t, pixy.StateTracking, "", "")
	d.mu.Lock()
	d.applyProbeResultLocked(probeResult{VideoDev: "", HidrawDev: ""})
	d.mu.Unlock()

	if got := readCameraState(d); got != pixy.StateTracking {
		t.Fatalf("BUG REPRODUCED: camera after probe miss = %q, want tracking", got)
	}
}

func TestZZRepro_HadPersistedStale(t *testing.T) {
	d := newTestDaemon(t, pixy.StatePrivacy, "", "")
	if d.hadPersistedState {
		t.Fatal("test setup: expected fresh flag false")
	}

	d.mu.Lock()
	d.state.Camera = pixy.StateTracking
	d.saveStateOrLog("repro")
	d.mu.Unlock()

	if !d.hadPersistedState {
		t.Fatalf("BUG REPRODUCED: hadPersistedState still false after saving user intent")
	}
}
