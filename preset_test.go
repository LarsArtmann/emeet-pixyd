//go:build linux

package main

import (
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestPreset_SaveLoadDelete(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
		withFakeParsePTZ(10, -5, 120),
	)

	// Save a preset — should read from fake parsePTZ (10,-5,120)
	result := d.handleCommand(t.Context(), "preset save home")
	if result.IsError() {
		t.Fatalf("preset save: %v", result)
	}

	// Verify it was saved in state
	d.mu.RLock()
	saved, ok := d.state.Presets["home"]
	d.mu.RUnlock()

	if !ok {
		t.Fatal("preset 'home' not found in state")
	}

	if saved.Pan != 10 || saved.Tilt != -5 || saved.Zoom != 120 {
		t.Errorf("preset saved = %+v, want {Pan:10, Tilt:-5, Zoom:120}", saved)
	}

	// List presets
	listResult := d.handleCommand(t.Context(), "preset list")
	if listResult.IsError() {
		t.Fatalf("preset list: %v", listResult)
	}

	// Delete the preset
	delResult := d.handleCommand(t.Context(), "preset delete home")
	if delResult.IsError() {
		t.Fatalf("preset delete: %v", delResult)
	}

	// Verify it was deleted
	d.mu.RLock()
	_, ok = d.state.Presets["home"]
	d.mu.RUnlock()

	if ok {
		t.Error("preset 'home' should have been deleted")
	}
}

func TestPreset_LoadNotFound(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
	)

	result := d.handleCommand(t.Context(), "preset load nonexistent")
	if !result.IsError() {
		t.Error("expected error for nonexistent preset")
	}
}

func TestPreset_DeleteNotFound(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
	)

	result := d.handleCommand(t.Context(), "preset delete nonexistent")
	if !result.IsError() {
		t.Error("expected error for deleting nonexistent preset")
	}
}

func TestPreset_ListEmpty(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
	)

	result := d.handleCommand(t.Context(), "preset list")
	if result.IsError() {
		t.Fatalf("preset list empty: %v", result)
	}

	if result.String() != "no presets saved" {
		t.Errorf("preset list empty = %q, want %q", result.String(), "no presets saved")
	}
}

func TestPreset_Usage(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
	)

	result := d.handleCommand(t.Context(), "preset")
	if result.String() != respPresetUsage {
		t.Errorf("preset usage = %q, want %q", result.String(), respPresetUsage)
	}
}

func TestPreset_SaveNoDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)
	result := d.handleCommand(t.Context(), "preset save test")

	if !result.IsError() {
		t.Error("expected error when saving preset without device")
	}
}

func TestPreset_LoadSetsPTZCache(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
	)

	// Pre-populate a preset
	d.mu.Lock()
	d.state.Presets = map[string]pixy.PTZValues{
		"custom": {Pan: 42, Tilt: -20, Zoom: 130},
	}
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset load custom")
	if result.IsError() {
		t.Fatalf("preset load: %v", result)
	}

	// Verify PTZ cache was updated
	cached, valid := d.ptzCache.Get()
	if !valid {
		t.Fatal("PTZ cache should be valid after preset load")
	}

	if cached.Pan != 42 || cached.Tilt != -20 || cached.Zoom != 130 {
		t.Errorf("PTZ cache after load = %+v, want {Pan:42, Tilt:-20, Zoom:130}", cached)
	}
}

func TestPreset_PresetsPersistInState(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t, pixy.StateTracking, "/dev/video0", "/dev/hidraw0",
		withFakeDevices(),
		withFakeParsePTZ(5, 5, 110),
	)

	// Save a preset
	_ = d.handleCommand(t.Context(), "preset save pos1")

	// Save state
	d.mu.Lock()
	d.saveStateOrLog("test save")
	d.mu.Unlock()

	// Load state into a new daemon
	d2 := newTestDaemon(
		t, pixy.StatePrivacy, "", "",
		withFakeDevices(),
	)
	d2.config.StateDir = d.config.StateDir

	loaded := d2.loadState()
	if !loaded {
		t.Fatal("loadState should succeed")
	}

	d2.mu.RLock()
	pos, ok := d2.state.Presets["pos1"]
	d2.mu.RUnlock()

	if !ok {
		t.Fatal("preset 'pos1' should have persisted")
	}

	if pos.Pan != 5 || pos.Tilt != 5 || pos.Zoom != 110 {
		t.Errorf("persisted preset = %+v, want {Pan:5, Tilt:5, Zoom:110}", pos)
	}
}
