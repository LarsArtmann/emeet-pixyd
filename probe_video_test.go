//go:build linux

package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestProbeVideo4linux_PIXYFound(t *testing.T) {
	t.Parallel()

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, product: pixyUeventProduct, index: "0"},
		{name: testVideoDev2, product: pixyUeventProduct, index: "1"},
	})
}

func TestProbeVideo4linux_PIXY2KFound(t *testing.T) {
	t.Parallel()

	// PIXY 2K (328f:0118) — kernel compact hex product form (issue #6).
	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, product: "328f/118/0100", index: "0"},
	})
}

func TestProbeVideo4linux_PIXYOnlyCaptureNode(t *testing.T) {
	t.Parallel()

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, product: pixyUeventProduct, index: "0"},
	})
}

func TestProbeVideo4linux_PIXYNoIndexFile(t *testing.T) {
	t.Parallel()

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, product: pixyUeventProduct, index: ""},
	})
}

func TestProbeVideo4linux_NonPIXYSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		devices []fakeVideoDev
	}{
		{
			"NoPIXY",
			[]fakeVideoDev{
				{
					name:    "video1",
					product: "1511/402d/0100",
					index:   "0",
				},
			},
		},
		{
			"WrongVendorProduct",
			[]fakeVideoDev{
				{
					name:    testVideoDev0,
					product: "1234/5678/0001",
					index:   "0",
				},
			},
		},
		{"EmptyDir", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testV4L2ProbesNothing(t, tc.devices)
		})
	}
}

func TestProbeVideo4linux_NonexistentDir(t *testing.T) {
	t.Parallel()

	result, _, _ := probeVideo4linux("/nonexistent/path/video4linux", nil)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeVideo4linux_FixedEMEETRecognizedNotControllable(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with a fixed EMEET C960 (328f:003f, linux-hardware.org)
	root := t.TempDir()
	createFakeVideo4linux(t, root, []fakeVideoDev{
		{name: testVideoDev0, product: "328f/003f/100", index: "0"},
	})

	// When probing
	result, model, unsupported := probeVideo4linux(root, nil)

	// Then the C960 is NOT treated as a controllable device
	if result != "" {
		t.Errorf("expected no video device, got %s", result)
	}

	if model != "" {
		t.Errorf("expected no model, got %s", model)
	}

	// But it IS recognized with an actionable hint.
	for _, want := range []string{"EMEET C960", "0x003f", "/dev/video0"} {
		if !strings.Contains(unsupported, want) {
			t.Errorf("hint %q missing %q", unsupported, want)
		}
	}
}

func TestProbeVideo4linux_UnknownEMEETVendorHint(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with an EMEET-vendor device the registry does not know
	root := t.TempDir()
	createFakeVideo4linux(t, root, []fakeVideoDev{
		{name: testVideoDev0, product: "328f/0abc/100", index: "0"},
	})

	// When probing
	_, _, unsupported := probeVideo4linux(root, nil)

	// Then the hint names the vendor, the PID, and the opt-in env var
	for _, want := range []string{"unknown EMEET device", "0x0abc", "EMEET_PIXYD_EXTRA_PRODUCT_IDS"} {
		if !strings.Contains(unsupported, want) {
			t.Errorf("hint %q missing %q", unsupported, want)
		}
	}
}

func TestProbeVideo4linux_ExtraProductIDTreatedAsPIXY(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with an unregistered PIXY-family variant
	root := t.TempDir()
	createFakeVideo4linux(t, root, []fakeVideoDev{
		{name: testVideoDev0, product: "328f/0119/100", index: "0"},
	})

	// When probing with the PID explicitly opted in
	result, model, unsupported := probeVideo4linux(root, []int64{0x0119})

	// Then the device is controllable under a visible custom model name
	if result != "/dev/video0" {
		t.Errorf("expected /dev/video0, got %s", result)
	}

	if model != pixy.Model("PIXY (PID 0x0119)") {
		t.Errorf("model = %q, want %q", model, "PIXY (PID 0x0119)")
	}

	if unsupported != "" {
		t.Errorf("expected no unsupported hint, got %q", unsupported)
	}
}

func TestProbeVideo4linux_PIXYWinsOverFixedEMEET(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with both a fixed C960 and a controllable PIXY
	root := t.TempDir()
	createFakeVideo4linux(t, root, []fakeVideoDev{
		{name: testVideoDev0, product: "328f/003f/100", index: "0"},
		{name: testVideoDev2, product: pixyUeventProduct, index: "0"},
	})

	// When probing
	result, model, unsupported := probeVideo4linux(root, nil)

	// Then the PIXY wins and no unsupported hint leaks
	if result != "/dev/video2" {
		t.Errorf("expected /dev/video2, got %s", result)
	}

	if model != pixy.ModelOriginal {
		t.Errorf("model = %q, want %q", model, pixy.ModelOriginal)
	}

	if unsupported != "" {
		t.Errorf("expected no unsupported hint, got %q", unsupported)
	}
}

func TestUnsupportedEMEETFromUevent_ControllableDeviceYieldsEmpty(t *testing.T) {
	t.Parallel()

	uevent := []byte("PRODUCT=328f/00c0/2004\n")
	if hint := unsupportedEMEETFromUevent(uevent, "PRODUCT=", "/", 0, 1, "/dev/video0"); hint != "" {
		t.Errorf("expected empty hint for controllable PIXY, got %q", hint)
	}
}

func TestProbeVideo4linux_OBSCamIgnored(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	obsDir := filepath.Join(root, "video1")
	writeFakeFile(t, filepath.Join(obsDir, "name"), "OBS Cam")
	writeFakeFile(t, filepath.Join(obsDir, "index"), "0")

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, product: pixyUeventProduct, index: "0"},
	})
}

func TestProbeVideo4linux_MetadataNodeSkipped(t *testing.T) {
	t.Parallel()

	testV4L2ProbesNothing(t, []fakeVideoDev{
		{name: testVideoDev2, product: pixyUeventProduct, index: "1"},
	})
}

func TestProbeVideo4linux_MultipleCamerasPIXYSecond(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with another camera first, then PIXY
	root := t.TempDir()

	otherDir := filepath.Join(root, testVideoDev0)
	writeFakeFile(
		t,
		filepath.Join(otherDir, "device/modalias"),
		"usb:v1234p5678d0100dcEFdsc02dp01ic0Eisc01ip00in00",
	)
	writeFakeFile(t, filepath.Join(otherDir, "index"), "0")
	writeFakeFile(t, filepath.Join(otherDir, "name"), "Other Camera")

	createFakeVideo4linux(t, root, []fakeVideoDev{
		{
			name:    testVideoDev2,
			product: pixyUeventProduct,
			index:   "0",
		},
	})

	// When probing
	result, _, _ := probeVideo4linux(root, nil)

	// Then the PIXY is found even though it's not the first device
	if result != "/dev/video2" {
		t.Errorf("expected /dev/video2, got %s", result)
	}
}

func TestSetDeviceState_CircuitBreaker(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev)

	d.mu.Lock()
	d.hidDev = &failingHID{err: errors.New("device busy")}
	d.hidFailCount = hidCircuitBreakerThreshold
	d.mu.Unlock()

	err := d.setDeviceState(
		context.Background(),
		[]byte{0},
		[]byte{0},
		func(_ *Daemon) {},
	)
	if err == nil {
		t.Fatal("expected circuit-open error")
	}

	if !errors.Is(err, pixy.ErrPIXYNotConnected) {
		t.Errorf("circuit-open error should wrap ErrPIXYNotConnected, got: %v", err)
	}
}
