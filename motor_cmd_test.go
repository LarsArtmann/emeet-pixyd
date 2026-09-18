//go:build linux

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the `speed <axis> <value>` command (TODO #138): CLI-surface
// validation and the daemon→simulator round-trip over the official V2
// SetMotorSpeed report.

func TestHandleSpeedCommand_MissingArgs(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	for _, cmd := range []string{"speed", "speed pan"} {
		result := d.handleCommand(t.Context(), cmd)
		if !result.IsError() || result.String() != errorPrefix+respSpeedUsage {
			t.Errorf("handleCommand(%q) = %q, want usage error", cmd, result.String())
		}
	}
}

func TestHandleSpeedCommand_InvalidAxis(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	result := d.handleCommand(t.Context(), "speed sideways 10")
	if !result.IsError() || !strings.HasSuffix(result.String(), respSpeedUsage) {
		t.Errorf("invalid axis = %q, want usage error", result.String())
	}
}

func TestHandleSpeedCommand_InvalidValue(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	for _, val := range []string{"abc", "-5", "10001"} {
		result := d.handleCommand(t.Context(), "speed pan "+val)
		if !result.IsError() {
			t.Errorf("speed pan %s = %q, want error", val, result.String())
		}
	}
}

func TestHandleSpeedCommand_NoDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	result := d.handleCommand(t.Context(), "speed pan 45")
	if !result.IsError() {
		t.Fatalf("speed without device = %q, want error", result.String())
	}

	if !strings.Contains(result.String(), "not connected") {
		t.Errorf("error = %q, want not-connected", result.String())
	}
}

func TestHandleSpeedCommand_SimulatorRoundTrip(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	result := d.handleCommand(t.Context(), "speed pan 45")
	if result.IsError() {
		t.Fatalf("speed pan 45 failed: %s", result.String())
	}

	if got := sim.MotorSpeed(pixy.MotorPan); got != 45 {
		t.Fatalf("simulator pan speed = %v, want 45", got)
	}

	if want := "motor speed set: pan 45"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}

	// The sent report must be the official V2 head with the motor-MCU iface
	// byte, followed by the [motorType][speed f32] payload.
	reports := sim.SentReports()
	if len(reports) != 1 {
		t.Fatalf("expected 1 V2 report, got %d", len(reports))
	}

	wantHead := pixy.V2SetMotorSpeed.WithIface(pixy.MotorMCUIface).Bytes()
	if string(reports[0][:4]) != string(wantHead) {
		t.Errorf("head = %x, want %x", reports[0][:4], wantHead)
	}

	if reports[0][4] != byte(pixy.MotorPan) {
		t.Errorf("motorType byte = %#02x, want %#02x", reports[0][4], byte(pixy.MotorPan))
	}
}

func TestHandleSpeedCommand_AllAxes(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	for _, tc := range []struct {
		axis  string
		motor pixy.MotorType
		speed float32
	}{
		{"pan", pixy.MotorPan, 10},
		{"tilt", pixy.MotorTilt, 20},
		{"zoom", pixy.MotorZoom, 30},
	} {
		if result := d.handleCommand(t.Context(), "speed "+tc.axis+" 25"); result.IsError() {
			t.Fatalf("speed %s: %s", tc.axis, result.String())
		}

		if got := sim.MotorSpeed(tc.motor); got != 25 {
			t.Errorf("%s speed = %v, want 25", tc.axis, got)
		}
	}
}

func TestHandleSpeedCommand_CircuitBreakerOpen(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	d.mu.Lock()
	d.hidFailCount = hidCircuitBreakerThreshold
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "speed pan 45")
	if !result.IsError() || !strings.Contains(result.String(), "not connected") {
		t.Fatalf("speed with open breaker = %q, want not-connected", result.String())
	}

	if got := sim.MotorSpeed(pixy.MotorPan); got != 0 {
		t.Errorf("report sent despite open breaker: speed = %v", got)
	}
}

func TestHandleSpeedCommand_SetMotorSpeedFailureCountsTowardBreaker(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	sim.sendErr = context.DeadlineExceeded
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	if result := d.handleCommand(t.Context(), "speed pan 45"); !result.IsError() {
		t.Fatalf("speed with sendErr = %q, want error", result.String())
	}

	d.mu.RLock()
	failCount := d.hidFailCount
	d.mu.RUnlock()

	if failCount != 1 {
		t.Errorf("hidFailCount = %d, want 1 (failure must be accounted)", failCount)
	}
}
