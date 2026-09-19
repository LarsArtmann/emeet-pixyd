//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"

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

// reassertSpeeds re-sends the persisted per-axis motor speeds over
// SetMotorSpeed before a move (TODO #138 wiring). Firmware-side retention is
// untrustworthy (power cycles reset it; per-move retention is M27-verify), so
// every move path re-asserts instead of assuming. Axes whose persisted speed
// is zero carry no preference and are skipped; nothing is sent when no listed
// axis has a speed.
//
// Failures are logged, not returned: the speed is an enhancement to the move,
// never a precondition. No lock is taken here — callers on the V4L2 move
// paths (which hold v4l2Mu) rely on the global v4l2Mu → hidMu lock order.
func (d *Daemon) reassertSpeeds(ctx context.Context, axes ...pixy.Axis) {
	d.mu.RLock()
	configured := false

	for _, axis := range axes {
		if speed, ok := d.state.Speeds.Get(axis); ok && speed > 0 {
			configured = true

			break
		}
	}
	d.mu.RUnlock()

	if !configured {
		return
	}

	d.hidMu.Lock()
	err := d.reassertSpeedsLocked(ctx, axes...)
	d.hidMu.Unlock()

	if err != nil {
		slog.Warn("motor-speed re-assert failed, moving at firmware default speed", "error", err)
	}
}

// reassertSpeedsLocked is reassertSpeeds for callers that already hold
// d.hidMu (preset push, reconcile-on-appear). It stops at the first failing
// axis so one flaky device cannot triple-count toward the HID circuit
// breaker within a single re-assert.
//
// LOCK CONTRACT: caller holds d.hidMu.
func (d *Daemon) reassertSpeedsLocked(ctx context.Context, axes ...pixy.Axis) error {
	d.mu.RLock()
	speeds := d.state.Speeds
	d.mu.RUnlock()

	for _, axis := range axes {
		speed, ok := speeds.Get(axis)
		if !ok || speed <= 0 {
			continue
		}

		motor, ok := pixy.MotorTypeFromAxis(axis)
		if !ok {
			continue
		}

		if err := d.setMotorSpeed(ctx, motor, speed); err != nil {
			return fmt.Errorf("%s speed %g: %w", axis, speed, err)
		}
	}

	return nil
}

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
func (d *Daemon) sendV2Set(ctx context.Context, operation string, report []byte) error {
	d.mu.RLock()
	hidDev := d.hidDev
	circuitOpen := d.hidFailCount >= hidCircuitBreakerThreshold
	d.mu.RUnlock()

	if hidDev == nil {
		return fmt.Errorf("%s (no device): %w", operation, pixy.ErrPIXYNotConnected)
	}

	if circuitOpen {
		return fmt.Errorf("%s: %w", operation, pixy.ErrPIXYNotConnected)
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

		return fmt.Errorf("%s send: %w", operation, err)
	}

	d.mu.Lock()
	d.hidFailCount = 0
	d.mu.Unlock()

	return nil
}
