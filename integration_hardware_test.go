//go:build integration

// Integration tests require a real EMEET PIXY device connected.
// Run with: go test -tags=integration -run TestIntegration ./...
// These tests are skipped in normal CI — they exercise real HID/V4L2 hardware paths.

package main

import (
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestIntegration_DeviceProbed(t *testing.T) {
	d := newTestDaemon(t, pixy.StatePrivacy, "", "")

	probeResult := probeDevices()
	d.applyProbeResultLocked(probeResult)

	d.mu.RLock()
	videoDev := d.videoDev
	hidrawDev := d.hidrawDev
	d.mu.RUnlock()

	if videoDev == "" {
		t.Skip("no PIXY video device found — connect hardware to run this test")
	}

	if hidrawDev == "" {
		t.Skip("no PIXY hidraw device found — connect hardware to run this test")
	}

	t.Logf("PIXY found: video=%s hidraw=%s", videoDev, hidrawDev)
}

func TestIntegration_PTZRoundTrip(t *testing.T) {
	probeResult := probeDevices()

	if probeResult.VideoDev == "" {
		t.Skip("no PIXY video device found — connect hardware to run this test")
	}

	// newTestDaemon wires real parsePTZValues by default — DON'T inject fake stubs
	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)

	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	// Set pan to a known value
	result := d.handleCommand(t.Context(), "pan 30")
	if result.IsError() {
		t.Fatalf("pan 30 failed: %v", result)
	}

	// Wait for motor settle + delayed readback
	time.Sleep(700 * time.Millisecond)

	// Read back actual position
	values := d.deps.parsePTZ(t.Context(), videoDev)

	if v, _ := values.Get(pixy.AxisPan); v != 30 {
		t.Errorf("pan readback = %d, want 30", v)
	}

	// Reset to center
	centerResult := d.handleCommand(t.Context(), "center")
	if centerResult.IsError() {
		t.Fatalf("center failed: %v", centerResult)
	}
}

func TestIntegration_HIDTrackingToggle(t *testing.T) {
	probeResult := probeDevices()

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StatePrivacy, probeResult.VideoDev, probeResult.HidrawDev)

	// Toggle tracking on
	result := d.handleCommand(t.Context(), "track")
	if result.IsError() {
		t.Fatalf("track failed: %v", result)
	}

	d.mu.RLock()
	camera := d.state.Camera
	d.mu.RUnlock()

	if camera != pixy.StateTracking {
		t.Errorf("camera state = %s, want tracking", camera)
	}

	// Toggle privacy back on
	_ = d.handleCommand(t.Context(), "privacy")
}

func TestIntegration_AudioCycle(t *testing.T) {
	probeResult := probeDevices()

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)

	for _, want := range []pixy.AudioMode{pixy.AudioNC, pixy.AudioLive, pixy.AudioOriginal} {
		result := d.handleCommand(t.Context(), "audio "+string(want))
		if result.IsError() {
			t.Errorf("audio %s failed: %v", want, result)
		}

		d.mu.RLock()
		got := d.state.Audio
		d.mu.RUnlock()

		if got != want {
			t.Errorf("audio state = %s, want %s", got, want)
		}
	}
}

// TestIntegration_BatteryProbe explores whether the wired PIXY answers a HID
// battery query (official app: CMD_GET_BATTERY_LEVEL → u8 level, only the macOS
// build carries the HID form — see docs/hid-protocol-official-map.md §4/#139).
// The exact query framing is unknown, so this sweeps the known 4-byte query
// dialect ([0x09, iface, X, Y] — the shape of the tracking/audio queries) over
// all interfaces and logs every well-formed response. It never sends the
// 9-byte config + commit form, so it cannot change device state; privacy is
// re-asserted afterwards as a belt-and-suspenders restore.
//
// Verdict table: any response with 0x09 prefix from a non-{0x01,0x04,0x05}
// interface is a candidate battery/info surface → feed findings into TODO #144.
func TestIntegration_BatteryProbe(t *testing.T) {
	probeResult := probeDevices()

	if probeResult.HidrawDev == "" {
		t.Skip("no PIXY hidraw device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StatePrivacy, probeResult.VideoDev, probeResult.HidrawDev)
	dev := newHIDRawDevice(probeResult.HidrawDev)

	knownIfaces := map[byte]bool{
		hidInterfaceTracking: true,
		hidInterfaceGesture:  true,
		hidInterfaceAudio:    true,
	}

	for iface := byte(0x00); iface <= 0x07; iface++ {
		for _, tail := range [][2]byte{{0x01, 0x01}, {0x00, 0x04}, {0x02, 0x01}} {
			resp, err := dev.SendRecv(t.Context(), []byte{cameraConfigPrefix, iface, tail[0], tail[1]})
			if err != nil {
				t.Logf("iface %#02x tail %v: error: %v", iface, tail, err)
				continue
			}

			if len(resp) == 0 {
				continue
			}

			status := "UNKNOWN-IFACE"
			if knownIfaces[iface] {
				status = "known-iface"
			}

			t.Logf("iface %#02x tail %v: %s resp=%x", iface, tail, status, resp)
		}
	}

	// Restore a known camera state regardless of what the probe stirred up.
	if result := d.handleCommand(t.Context(), "privacy"); result.IsError() {
		t.Logf("privacy restore failed: %v", result)
	}
}
