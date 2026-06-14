//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestProbeHidraw_PIXYFound(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with a PIXY hidraw device
	root := t.TempDir()
	createFakeHidraw(t, root, []fakeHidrawDev{
		{
			name:    "hidraw7",
			hidID:   "0003:0000328F:000000C0",
			hidName: "EMEET PIXY",
		},
	})

	// When probing
	result := probeHidraw(root)

	// Then the PIXY hidraw is found
	if result != testHIDDev {
		t.Errorf("expected /dev/hidraw7, got %s", result)
	}
}

func TestProbeHidraw_NoPIXY(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with only non-PIXY hidraw devices
	root := t.TempDir()
	createFakeHidraw(t, root, []fakeHidrawDev{
		{
			name:    "hidraw0",
			hidID:   "0003:00003151:0000402D",
			hidName: "2.4G Wireless Mouse",
		},
		{
			name:    "hidraw3",
			hidID:   "0003:00001A2C:00004852",
			hidName: "SEMICO USB Gaming Keyboard",
		},
	})

	// When probing
	result := probeHidraw(root)

	// Then nothing is found
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeHidraw_EmptyDir(t *testing.T) {
	t.Parallel()

	// Given an empty hidraw sysfs directory
	root := t.TempDir()

	// When probing
	result := probeHidraw(root)

	// Then nothing is found
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeHidraw_NonexistentDir(t *testing.T) {
	t.Parallel()

	// Given a nonexistent sysfs path
	result := probeHidraw("/nonexistent/path/hidraw")

	// Then nothing is found
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeHidraw_MixedDevices(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with mouse, keyboard, and PIXY
	root := t.TempDir()
	createFakeHidraw(t, root, []fakeHidrawDev{
		{
			name:    "hidraw0",
			hidID:   "0003:00003151:0000402D",
			hidName: "2.4G Wireless Mouse",
		},
		{
			name:    "hidraw3",
			hidID:   "0003:00001A2C:00004852",
			hidName: "SEMICO USB Gaming Keyboard",
		},
		{
			name:    "hidraw7",
			hidID:   "0003:0000328F:000000C0",
			hidName: "EMEET PIXY",
		},
		{
			name:    "hidraw8",
			hidID:   "0003:0000043E:00009A39",
			hidName: "LG Electronics Inc. LG Monitor Controls",
		},
	})

	// When probing
	result := probeHidraw(root)

	// Then the PIXY is found
	if result != testHIDDev {
		t.Errorf("expected /dev/hidraw7, got %s", result)
	}
}

func TestProbeHidraw_NoUeventFile(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with a directory but no uevent file
	root := t.TempDir()

	dirErr := os.MkdirAll(filepath.Join(root, "hidraw0", "device"), 0o755)
	if dirErr != nil {
		t.Fatalf("mkdir: %v", dirErr)
	}

	// When probing
	result := probeHidraw(root)

	// Then nothing is found (graceful skip)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestHasPixyProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		uevent  string
		matches bool
	}{
		{
			"compact hex (kernel format)",
			"DEVTYPE=usb_interface\nPRODUCT=328f/c0/2004\n",
			true,
		},
		{
			"leading zeros",
			"DEVTYPE=usb_interface\nPRODUCT=328f/00c0/2004\n",
			true,
		},
		{
			"uppercase hex",
			"DEVTYPE=usb_interface\nPRODUCT=328F/C0/2004\n",
			true,
		},
		{
			"wrong vendor",
			"DEVTYPE=usb_interface\nPRODUCT=1234/c0/2004\n",
			false,
		},
		{
			"wrong product",
			"DEVTYPE=usb_interface\nPRODUCT=328f/00c1/2004\n",
			false,
		},
		{
			"no PRODUCT line",
			"DEVTYPE=usb_interface\nDRIVER=uvcvideo\n",
			false,
		},
		{
			"empty uevent",
			"",
			false,
		},
		{
			"malformed PRODUCT (one field)",
			"PRODUCT=328f\n",
			false,
		},
		{
			// The malformed first PRODUCT line used to make matchesPixyID
			// return false immediately. A valid PRODUCT line later in the
			// file should still be honored.
			"malformed PRODUCT followed by valid one",
			"PRODUCT=328f\nPRODUCT=328f/00c0/2004\n",
			true,
		},
		{
			// Reverse order: valid first, malformed second — must still
			// match on the first valid line.
			"valid PRODUCT followed by malformed",
			"PRODUCT=328f/00c0/2004\nPRODUCT=328f\n",
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := matchesPixyID([]byte(tc.uevent), "PRODUCT=", "/", 0, 1)
			if got != tc.matches {
				t.Errorf("matchesPixyID(%q) = %v, want %v", tc.uevent, got, tc.matches)
			}
		})
	}
}

func TestProbeDevices_SetsStateToOfflineWhenNoVideo(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		initialCamera pixy.CameraState
	}{
		{"from non-offline state", pixy.StatePrivacy},
		{"from offline state", pixy.StateOffline},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := newTestDaemon(tc.initialCamera, "", "")
			d.applyProbeResultLocked(probeDevices())

			hasDev := d.videoDev != ""

			isOffline := d.state.Camera == pixy.StateOffline
			if hasDev && isOffline {
				t.Error("camera should not be offline when video device is found")
			}

			if !hasDev && !isOffline {
				t.Errorf("expected offline when no video device, got %s", d.state.Camera)
			}
		})
	}
}
