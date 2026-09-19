//go:build linux

package main

import (
	"os"
	"strings"
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

func TestStateFileRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig(t.TempDir())

	data := []byte(`{"v":1,"camera":"tracking","audio":"nc","gesture":false,` +
		`"inCall":false,"autoMode":"full","bogusField":"evil"}`)

	err := os.WriteFile(cfg.StateFile(), data, pixy.PermissionStateFile)
	if err != nil {
		t.Fatalf("write state file: %v", err)
	}

	d := testDaemonNoDevice(t)
	d.config = cfg

	if loaded := d.loadState(); loaded {
		t.Error("expected loadState to reject unknown fields, but it loaded successfully")
	}

	if d.state.Camera != pixy.StatePrivacy {
		t.Errorf("expected state to remain at default, got %s", d.state.Camera)
	}
}

// TestStateRoundTrip_SpeedsOmitZero_TrackModeNone pins the on-disk JSON
// shape: zero speeds omit the whole "speeds" key (the file stays clean for
// users who never configured speeds), and the corrected "none" variant
// round-trips like any other.
func TestStateRoundTrip_SpeedsOmitZero_TrackModeNone(t *testing.T) {
	t.Parallel()

	cfg := defaultTestConfig(t.TempDir())

	d := &Daemon{
		mu:     sync.RWMutex{},
		config: cfg,
		state:  pixy.DefaultState(),
	}

	d.state.TrackMode = pixy.TrackNone.String()
	d.state.Speeds = pixy.SpeedValues{}.Set(pixy.AxisPan, 42.5)

	if err := d.saveState(); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	raw, err := os.ReadFile(cfg.StateFile())
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	if !strings.Contains(string(raw), `"speeds":{"pan":42.5}`) {
		t.Errorf("state file missing speeds object: %s", raw)
	}

	if !strings.Contains(string(raw), `"trackMode":"none"`) {
		t.Errorf("state file missing trackMode none: %s", raw)
	}

	reloaded := &Daemon{
		mu:     sync.RWMutex{},
		config: cfg,
		state:  pixy.DefaultState(),
	}

	if !reloaded.loadState() {
		t.Fatal("loadState = false, want true (state file was just written)")
	}

	if got, _ := reloaded.state.Speeds.Get(pixy.AxisPan); got != 42.5 {
		t.Errorf("pan speed after reload = %v, want 42.5", got)
	}

	if got := reloaded.state.EffectiveTrackMode(); got != pixy.TrackNone {
		t.Errorf("effective track mode after reload = %v, want none", got)
	}

	clean := &Daemon{
		mu:     sync.RWMutex{},
		config: cfg,
		state:  pixy.DefaultState(),
	}

	if err := clean.saveState(); err != nil {
		t.Fatalf("saveState clean: %v", err)
	}

	raw, err = os.ReadFile(cfg.StateFile())
	if err != nil {
		t.Fatalf("re-read state file: %v", err)
	}

	if strings.Contains(string(raw), "speeds") {
		t.Errorf("zero speeds still wrote a speeds key: %s", raw)
	}
}
