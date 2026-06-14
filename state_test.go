//go:build linux

package main

import (
	"os"
	"sync"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestStateDefaults(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()
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
		mu:     sync.RWMutex{},
		config: cfg,
		state: pixy.State{
			Camera:   pixy.StateTracking,
			Audio:    pixy.AudioLive,
			Gesture:  true,
			InCall:   true,
			AutoMode: pixy.AutoOff,
		},
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

	d := testDaemonNoDevice()

	d.config = cfg
	if loaded := d.loadState(); loaded {
		t.Error("expected loadState to return false for corrupt file")
	}

	if d.state.Camera != pixy.StatePrivacy {
		t.Errorf("expected state to remain unchanged on corrupt file, got %s", d.state.Camera)
	}
}

func TestStateFileMissing(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig("/nonexistent")
	d := testDaemonNoDevice()

	d.config = cfg
	if loaded := d.loadState(); loaded {
		t.Error("expected loadState to return false for missing file")
	}

	assertCameraState(t, d, pixy.StatePrivacy)
}

func TestStateFileValid(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig(t.TempDir())

	d := testDaemonNoDevice()
	d.config = cfg

	d.state = pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioLive,
		Gesture:  true,
		InCall:   true,
		AutoMode: pixy.AutoOff,
	}
	if saveErr := d.saveState(); saveErr != nil {
		t.Fatalf("saveState: %v", saveErr)
	}

	// Simulate a fresh daemon start with a different in-memory default.
	d2 := testDaemonNoDevice()
	d2.config = cfg
	d2.state = pixy.DefaultState()

	if loaded := d2.loadState(); !loaded {
		t.Error("expected loadState to return true for valid file")
	}

	if d2.state.AutoMode != pixy.AutoOff {
		t.Errorf("expected persisted AutoMode to win, got %s", d2.state.AutoMode)
	}
}
