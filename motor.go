//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// The official V2 motor-command family (docs/hid-protocol-official-map.md
// §3.5). Unlike the config+commit dialect used for tracking/audio/gesture,
// V2 SETs are single reports — no two-step, no hidCommandSleepMs pause.

// setMotorSpeed sends SetMotorSpeed for one axis. The report goes out with
// the motor-MCU iface byte (0x63), matching the official app's on-wire
// behavior; which routing the wired PIXY firmware answers is a pending
// hardware verification (plan M27) — flip pixy.MotorMCUIface usage there if
// it proves wrong.
//
// The value is transmitted verbatim as float32. The physical unit is ASSUMED
// to be degrees/second (map doc §4/#138) and is not hardware-verified yet,
// so callers apply only the sanity bound from the command layer — there is
// no hardware range to clamp to until M27 pins it.
//
// Failure accounting mirrors setDeviceState (circuit breaker + re-probe on
// send failure) without touching the working config+commit path.
func (d *Daemon) setMotorSpeed(ctx context.Context, motor pixy.MotorType, speed float32) error {
	report := append(
		pixy.V2SetMotorSpeed.WithIface(pixy.MotorMCUIface).Bytes(),
		pixy.MotorSpeedPayload(motor, speed)...,
	)

	d.mu.RLock()
	hidDev := d.hidDev
	circuitOpen := d.hidFailCount >= hidCircuitBreakerThreshold
	d.mu.RUnlock()

	if hidDev == nil {
		return fmt.Errorf("setMotorSpeed (no device): %w", pixy.ErrPIXYNotConnected)
	}

	if circuitOpen {
		return fmt.Errorf("setMotorSpeed: %w", pixy.ErrPIXYNotConnected)
	}

	if err := hidDev.Send(report); err != nil {
		d.mu.Lock()
		d.hidFailCount++

		recordHIDFailure(ctx)

		if d.hidFailCount < hidCircuitBreakerThreshold {
			d.applyProbeResultLocked(probeDevices()) //nolint:contextcheck
		}
		d.mu.Unlock()
		d.broadcastStateChanged()

		return fmt.Errorf("setMotorSpeed send: %w", err)
	}

	d.mu.Lock()
	d.hidFailCount = 0
	d.mu.Unlock()

	return nil
}


