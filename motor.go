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

	return d.sendV2Set(ctx, "setMotorSpeed", report)
}

// setTargetTrack sends the official V2 SetTargetTrack command for a tracking
// variant (TODO #140): single report, head 09 04 01 01 + [mode:u8][f32x3]
// with zeroed floats. Failure accounting mirrors setMotorSpeed. Callers hold
// d.hidMu (command dispatcher contract, same as setTracking).
func (d *Daemon) setTargetTrack(ctx context.Context, mode pixy.TargetTrackMode) error {
	report := append(pixy.V2SetTargetTrack.Bytes(), pixy.TargetTrackPayload(mode)...)

	d.mu.RLock()
	hidDev := d.hidDev
	circuitOpen := d.hidFailCount >= hidCircuitBreakerThreshold
	d.mu.RUnlock()

	if hidDev == nil {
		return fmt.Errorf("setTargetTrack (no device): %w", pixy.ErrPIXYNotConnected)
	}

	if circuitOpen {
		return fmt.Errorf("setTargetTrack: %w", pixy.ErrPIXYNotConnected)
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

		return fmt.Errorf("setTargetTrack send: %w", err)
	}

	d.mu.Lock()
	d.hidFailCount = 0
	d.mu.Unlock()

	return nil
}

// maxHardwarePresetSlots is the assumed motor-preset slot count. The official
// GetMotorPresetPosMode sweep that pins the real count (and per-slot validity)
// runs in the hardware session (plan M27 / TODO #141); the guard only prevents
// sending slots the firmware is unlikely to have.
const maxHardwarePresetSlots = 8

// setMotorPos moves one axis to an absolute position over the official V2
// SetMotorPos command (head+payload single report, motor-MCU iface). The
// position unit is assumed to be the same degrees/multiplier we present
// everywhere else (M27-verify). LOCK CONTRACT: caller holds d.hidMu.
func (d *Daemon) setMotorPos(ctx context.Context, motor pixy.MotorType, pos float32) error {
	report := append(
		pixy.V2SetMotorPos.WithIface(pixy.MotorMCUIface).Bytes(),
		pixy.MotorSpeedPayload(motor, pos)...,
	)

	return d.sendV2Set(ctx, "setMotorPos", report)
}

// setMotorPresetPos saves the CURRENT position into a 1-based hardware slot.
// LOCK CONTRACT: caller holds d.hidMu.
func (d *Daemon) setMotorPresetPos(ctx context.Context, slot byte) error {
	report := append(pixy.V2SetMotorPresetPos.WithIface(pixy.MotorMCUIface).Bytes(), slot)

	return d.sendV2Set(ctx, "setMotorPresetPos", report)
}

// sendV2Set is the shared transport for single-report V2 SET commands:
// device/circuit guards, Send, and setDeviceState-style failure accounting.
// LOCK CONTRACT: the caller holds d.hidMu (the command dispatcher holds it
// for HID commands; multi-step callers take it around their whole sequence —
// taking it here would deadlock against the dispatcher).
func (d *Daemon) sendV2Set(ctx context.Context, op string, report []byte) error {
	d.mu.RLock()
	hidDev := d.hidDev
	circuitOpen := d.hidFailCount >= hidCircuitBreakerThreshold
	d.mu.RUnlock()

	if hidDev == nil {
		return fmt.Errorf("%s (no device): %w", op, pixy.ErrPIXYNotConnected)
	}

	if circuitOpen {
		return fmt.Errorf("%s: %w", op, pixy.ErrPIXYNotConnected)
	}

	err := hidDev.Send(report)

	if err != nil {
		d.mu.Lock()
		d.hidFailCount++

		recordHIDFailure(ctx)

		if d.hidFailCount < hidCircuitBreakerThreshold {
			d.applyProbeResultLocked(probeDevices()) //nolint:contextcheck
		}
		d.mu.Unlock()
		d.broadcastStateChanged()

		return fmt.Errorf("%s send: %w", op, err)
	}

	d.mu.Lock()
	d.hidFailCount = 0
	d.mu.Unlock()

	return nil
}
