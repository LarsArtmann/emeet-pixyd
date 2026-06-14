//go:build linux

package main

import (
	"context"
	"errors"
	"path/filepath"
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

	result := probeVideo4linux("/nonexistent/path/video4linux")
	if result != "" {
		t.Errorf("expected empty, got %s", result)
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
	result := probeVideo4linux(root)

	// Then the PIXY is found even though it's not the first device
	if result != "/dev/video2" {
		t.Errorf("expected /dev/video2, got %s", result)
	}
}

func TestSetDeviceState_CircuitBreaker(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateIdle, testVideoDev, testHIDDev)

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
