//go:build linux

package main

import (
	"os"
	"sync"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// inCallState returns a pixy.State representing an active call scenario
// shared across multiple state persistence tests.
func inCallState() pixy.State {
	return pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioLive,
		Gesture:  true,
		InCall:   true,
		AutoMode: pixy.AutoOff,
	}
}

// assertLoadStateFalse fails if loadState returns true (used for corrupt/missing files).
func assertLoadStateFalse(t *testing.T, d *Daemon, label string) {
	t.Helper()

	if loaded := d.loadState(); loaded {
		t.Errorf("expected loadState to return false for %s file", label)
	}
}

func TestStateDefaults(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)
	assertCameraState(t, d, pixy.StatePrivacy)

	if d.state.Audio != pixy.AudioNC {
		t.Errorf("expected default audio to be nc, got %s", d.state.Audio)
	}

	assertAutoMode(t, d, pixy.AutoFull)

	if d.state.InCall != false {
		t.Error("expected in_call to be false by default")
	}
}

func TestStateSaveLoad(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig(t.TempDir())

	d := &Daemon{
		mu:            sync.RWMutex{},
		config:        cfg,
		state:         inCallState(),
		videoDev:      "",
		hidrawDev:     "",
		debounceInUse: 0,
		debounceIdle:  0,
	}

	saveErr := d.saveState()
	if saveErr != nil {
		t.Fatalf("saveState: %v", saveErr)
	}

	d2 := &Daemon{
		mu:     sync.RWMutex{},
		config: cfg,
		state: pixy.State{
			Camera:   pixy.StateIdle,
			Audio:    pixy.AudioNC,
			Gesture:  false,
			InCall:   false,
			AutoMode: pixy.AutoFull,
		},
		videoDev:      "",
		hidrawDev:     "",
		debounceInUse: 0,
		debounceIdle:  0,
	}
	d2.loadState()

	if d2.state.Camera != pixy.StateTracking {
		t.Errorf("expected camera=tracking, got %s", d2.state.Camera)
	}

	if d2.state.Audio != pixy.AudioLive {
		t.Errorf("expected audio=live, got %s", d2.state.Audio)
	}

	if d2.state.Gesture != true {
		t.Error("expected gesture=true")
	}

	if d2.state.InCall != true {
		t.Error("expected in_call=true")
	}

	if d2.state.AutoMode != pixy.AutoOff {
		t.Error("expected auto_mode=false")
	}
}

func TestStateFileCorrupt(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig(t.TempDir())

	err := os.WriteFile(cfg.StateFile(), []byte("not json"), pixy.PermissionStateFile)
	if err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	d := testDaemonNoDevice(t)

	d.config = cfg
	assertLoadStateFalse(t, d, "corrupt")

	if d.state.Camera != pixy.StatePrivacy {
		t.Errorf("expected state to remain unchanged on corrupt file, got %s", d.state.Camera)
	}
}

func TestStateFileMissing(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig("/nonexistent")
	d := testDaemonNoDevice(t)

	d.config = cfg
	assertLoadStateFalse(t, d, "missing")

	assertCameraState(t, d, pixy.StatePrivacy)
}

func TestStateFileValid(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig(t.TempDir())

	d := testDaemonNoDevice(t)
	d.config = cfg

	d.state = inCallState()
	if saveErr := d.saveState(); saveErr != nil {
		t.Fatalf("saveState: %v", saveErr)
	}

	// Simulate a fresh daemon start with a different in-memory default.
	d2 := testDaemonNoDevice(t)
	d2.config = cfg
	d2.state = pixy.DefaultState()

	if loaded := d2.loadState(); !loaded {
		t.Error("expected loadState to return true for valid file")
	}

	if d2.state.AutoMode != pixy.AutoOff {
		t.Errorf("expected persisted AutoMode to win, got %s", d2.state.AutoMode)
	}
}
