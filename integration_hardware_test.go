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

// TestIntegration_BatteryProbe explores whether the wired PIXY answers HID
// read queries using the EXACT heads extracted from the official EMEET STUDIO
// binary (tools/emhid/cmdtable.json — see docs/hid-protocol-official-map.md
// §3.5). Queries are bare 4-byte heads with no payload; the official protocol
// has no commit step for them, so this cannot change device state — the only
// writes here are the privacy re-assertion at the end.
//
// Probed read-only heads (plan M3):
//
//	battery        09 00 00 02   (CMD_GET_BATTERY_LEVEL → u8 percent)
//	charge         09 00 00 06   (CMD_GET_CHARGE_STA → ChargeSta enum)
//	motor speed    09 03 01 13   (CMD_GET_MOTOR_SPEED → f32 value + f32 limit)
//	motor pos      09 03 01 02   (CMD_GET_MOTOR_POS → f32 pan/tilt/zoom)
//	target track   09 04 01 02   (CMD_GET_TARGET_TRACK → mode + f32×3)
//	device mode    09 02 01 00   (CMD_GET_DEVICE_MODE — authoritative vs our
//	                              SET-head query on 09 01 01 01)
//	func status    09 01 00 0d   (CMD_GET_FUNC_STA → u32 capability bitfield)
//	serial number  09 01 00 04   (CMD_GET_SN)
//	version        09 01 00 05   (CMD_GET_VER → u16)
//	device version 09 01 00 0f   (CMD_GET_DEVICE_VER → u16)
//
// Motor heads are probed with BOTH iface-byte variants — the logical 0x03 and
// the motor-MCU mergeType 0x63 (mergeType(3,3) = (3<<5)|3) — because the
// official app overwrites the iface byte on the wire for motor commands and
// which one the wired PIXY firmware answers is exactly what this probe
// answers. Response FRAMING is still unpinned (parse bodies were in the
// deleted /tmp disasm), so every response is logged as raw hex: the M27
// hardware session reads this log and feeds TODO #144 / #139.
func TestIntegration_BatteryProbe(t *testing.T) {
	probeResult := probeDevices()

	if probeResult.HidrawDev == "" {
		t.Skip("no PIXY hidraw device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StatePrivacy, probeResult.VideoDev, probeResult.HidrawDev)
	dev := newHIDRawDevice(probeResult.HidrawDev)

	// name → head; motor heads get a second entry with the 0x63 iface byte.
	probes := []struct {
		name string
		head []byte
	}{
		{"battery", []byte{0x09, 0x00, 0x00, 0x02}},
		{"charge", []byte{0x09, 0x00, 0x00, 0x06}},
		{"motor-speed-iface3", []byte{0x09, 0x03, 0x01, 0x13}},
		{"motor-speed-iface63", []byte{0x09, 0x63, 0x01, 0x13}},
		{"motor-pos-iface3", []byte{0x09, 0x03, 0x01, 0x02}},
		{"motor-pos-iface63", []byte{0x09, 0x63, 0x01, 0x02}},
		{"target-track", []byte{0x09, 0x04, 0x01, 0x02}},
		{"device-mode-official", []byte{0x09, 0x02, 0x01, 0x00}},
		{"func-status", []byte{0x09, 0x01, 0x00, 0x0d}},
		{"serial-number", []byte{0x09, 0x01, 0x00, 0x04}},
		{"version", []byte{0x09, 0x01, 0x00, 0x05}},
		{"device-version", []byte{0x09, 0x01, 0x00, 0x0f}},
	}

	for _, p := range probes {
		resp, err := dev.SendRecv(t.Context(), p.head)
		if err != nil {
			t.Logf("%-22s head=%x: error: %v", p.name, p.head, err)
			continue
		}
		if len(resp) == 0 {
			t.Logf("%-22s head=%x: no response (timeout)", p.name, p.head)
			continue
		}
		t.Logf("%-22s head=%x: %d bytes: %x", p.name, p.head, len(resp), resp)
	}

	// Restore a known camera state regardless of what the probe stirred up.
	if result := d.handleCommand(t.Context(), "privacy"); result.IsError() {
		t.Logf("privacy restore failed: %v", result)
	}
}
