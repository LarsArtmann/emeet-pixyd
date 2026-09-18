//go:build linux

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the battery/charge read surface (TODO #139). The simulator's
// fixture answers battery=87, charge=0; absence paths are exercised with a
// device-less daemon.

func TestBatteryCommand_SimulatorReading(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	result := d.handleCommand(t.Context(), "battery")
	if result.IsError() {
		t.Fatalf("battery failed: %s", result.String())
	}

	if want := "battery: 87% (discharging)"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}

	sim.state.mu.Lock()
	sim.state.batteryLevel = 50
	sim.state.chargeSta = 1
	sim.state.mu.Unlock()

	// Within the TTL the cached reading is served — no new HID query.
	result = d.handleCommand(t.Context(), "battery")
	if result.String() != "battery: 87% (discharging)" {
		t.Errorf("cached response = %q, want the first reading", result.String())
	}

	d.powerCache.Invalidate()

	result = d.handleCommand(t.Context(), "battery")
	if result.String() != "battery: 50% (charging)" {
		t.Errorf("post-TTL response = %q, want fresh reading", result.String())
	}
}

func TestBatteryCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	result := d.handleCommand(t.Context(), "battery")
	if result.IsError() {
		t.Fatalf("battery with no device errored: %s", result.String())
	}

	if want := respBatteryUnavailable; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}
}

func TestPowerStatus_AbsenceIsNotCachedForever(t *testing.T) {
	t.Parallel()

	// nil hidDev: every query fails fast (no device), so the power cache
	// must never claim a reading — the TTL expiry path for failures is a
	// no-op by construction because failures are simply not cached.
	d := testDaemonNoDevice(t)

	if _, ok := d.powerStatus(t.Context()); ok {
		t.Fatal("powerStatus reported a reading without a device")
	}
}

func TestPowerReading_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reading powerReading
		want    string
	}{
		{powerReading{Level: 87, Charging: false}, "87% (discharging)"},
		{powerReading{Level: 12, Charging: true}, "12% (charging)"},
	}

	for _, tc := range tests {
		if got := tc.reading.String(); got != tc.want {
			t.Errorf("String() = %q, want %q", got, tc.want)
		}
	}
}

func TestAppendPowerLine(t *testing.T) {
	t.Parallel()

	base := "camera=idle audio=nc device=/dev/video0"

	if got := appendPowerLine(base, powerReading{}, false); got != base {
		t.Errorf("absent reading changed the line: %q", got)
	}

	got := appendPowerLine(base, powerReading{Level: 42, Charging: true}, true)
	want := base + " battery=42%(charging)"

	if got != want {
		t.Errorf("line = %q, want %q", got, want)
	}
}

func TestPowerCache_Expiry(t *testing.T) {
	t.Parallel()

	c := powerCache{}

	if _, ok := c.Get(); ok {
		t.Fatal("empty cache returned a reading")
	}

	c.Set(powerReading{Level: 5}, -time.Second)

	if _, ok := c.Get(); ok {
		t.Fatal("expired entry returned a reading")
	}

	c.Set(powerReading{Level: 5}, time.Minute)

	if reading, ok := c.Get(); !ok || reading.Level != 5 {
		t.Fatalf("fresh entry: reading=%v ok=%v", reading, ok)
	}
}

func TestParseBatteryAndCharge(t *testing.T) {
	t.Parallel()

	batteryResp := append(pixy.V2GetBatteryLevel.Bytes(), 87)
	chargeResp := append(pixy.V2GetChargeSta.Bytes(), 1)

	level, err := pixy.ParseBatteryLevel(batteryResp)
	if err != nil || level != 87 {
		t.Errorf("battery = (%d, %v), want (87, nil)", level, err)
	}

	sta, err := pixy.ParseChargeStatus(chargeResp)
	if err != nil || sta != 1 {
		t.Errorf("charge = (%d, %v), want (1, nil)", sta, err)
	}

	if _, err := pixy.ParseBatteryLevel(pixy.V2GetBatteryLevel.Bytes()); err == nil {
		t.Error("empty battery payload accepted")
	}

	if _, err := pixy.ParseChargeStatus(pixy.V2GetChargeSta.Bytes()); err == nil {
		t.Error("empty charge payload accepted")
	}
}

func TestPresetPush_SimulatorSequence(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{"home": {Pan: 30, Tilt: -10, Zoom: 120}}
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset push home")
	if result.IsError() {
		t.Fatalf("preset push failed: %s", result.String())
	}

	if want := "preset pushed: home -> slot 1"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}

	// Three SetMotorPos reports (pan, tilt, zoom) then SetMotorPresetPos slot 1.
	reports := sim.SentReports()
	if len(reports) != 4 {
		t.Fatalf("expected 4 V2 reports, got %d", len(reports))
	}

	wantPosHead := pixy.V2SetMotorPos.WithIface(pixy.MotorMCUIface).Bytes()
	wantSaveHead := pixy.V2SetMotorPresetPos.WithIface(pixy.MotorMCUIface).Bytes()

	for i, axis := range []byte{byte(pixy.MotorPan), byte(pixy.MotorTilt), byte(pixy.MotorZoom)} {
		if string(reports[i][:4]) != string(wantPosHead) || reports[i][4] != axis {
			t.Errorf("report %d = %x, want SetMotorPos for axis %#02x", i, reports[i], axis)
		}
	}

	if string(reports[3][:4]) != string(wantSaveHead) || reports[3][4] != 1 {
		t.Errorf("save report = %x, want SetMotorPresetPos slot 1", reports[3])
	}
}

func TestPresetPush_UnknownPreset(t *testing.T) {
	t.Parallel()

	_, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	result := d.handleCommand(t.Context(), "preset push nope")
	if !result.IsError() || result.String() != errorPrefix+respPresetNotFound {
		t.Errorf("push unknown = %q, want %q", result.String(), respPresetNotFound)
	}
}

func TestPresetPush_SlotOverflowRejected(t *testing.T) {
	t.Parallel()

	_, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{}

	for i := range maxHardwarePresetSlots + 1 {
		d.state.Presets[fmt.Sprintf("p%02d", i)] = pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault}
	}

	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset push p08")
	if !result.IsError() || !strings.Contains(result.String(), "no hardware slot") {
		t.Errorf("push overflow = %q, want slot error", result.String())
	}
}

func TestPresetSave_MultiWordNameTruncates(t *testing.T) {
	t.Parallel()

	_, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	// PIN THE BUG (former TODO #123): strings.Fields dispatch means
	// `preset save my home` saves the preset "my" and silently drops
	// "home". The web UI (data-bind input) does not have this problem.
	// See docs/adr/2026-09-18_multi-word-preset-names.md.
	result := d.handleCommand(t.Context(), "preset save my home")
	if result.IsError() {
		t.Fatalf("preset save failed: %s", result.String())
	}

	d.mu.RLock()
	_, homeExists := d.state.Presets["my home"]
	_, myExists := d.state.Presets["my"]
	d.mu.RUnlock()

	if homeExists {
		t.Error("multi-word name saved; the truncation bug is fixed — update this test and the ADR")
	}

	if !myExists {
		t.Fatal("expected the truncated preset \"my\" to exist (pinning the bug)")
	}
}
