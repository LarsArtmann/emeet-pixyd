//go:build linux

package main

import (
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
