//go:build linux

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Hardware-evidenced privacy trap on the wired PIXY (2026-10-01 session, see
// integration_privacy_hardware_test.go): arriving at tilt <= -85° while in
// tracking or idle covers the lens (mean frame luma drops to ~7-24 vs ~130+
// open). The cover is transient and positional — leaving the tilt zone, or
// any camera-mode write that lands after the firmware's ~1.5-3s transition,
// re-opens the lens even while still parked in the zone. The failure users
// see: a track/idle write sent within the transition is ACKed but silently
// dropped, so clicking Track right after tilting down appears to do nothing.
const (
	// privacyTrapTilt is the tilt angle at (or below) which the trap engages.
	privacyTrapTilt = -85
)

// privacyTrapReassertDelay re-sends the commanded camera mode this long after
// the original write, safely past the measured ~1.5-3s transition window.
// A var (not const) purely so unit tests can shorten it.
//
//nolint:gochecknoglobals
var privacyTrapReassertDelay = 3 * time.Second

// privacyTrapArmWindow is how long a tilt arrival in the trap zone keeps the
// race protection armed. Measured from the tilt command and re-armed by the
// delayed PTZ readback, so slow travels from far positions stay covered.
const privacyTrapArmWindow = 6 * time.Second

// armPrivacyTrap records that the head just moved into the trap zone. Mode
// writes issued while armed get one deferred re-assert (see setTracking).
func (d *Daemon) armPrivacyTrap() {
	d.mu.Lock()
	d.trapArmedUntil = time.Now().Add(privacyTrapArmWindow)
	d.mu.Unlock()
}

// privacyTrapArmed reports whether a trap transition may still be in flight.
func (d *Daemon) privacyTrapArmed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return time.Now().Before(d.trapArmedUntil)
}

// schedulePrivacyTrapReassert re-sends a camera mode whose write may have
// been eaten by the firmware's privacy-trap transition. Skipped when the
// believed mode has moved on (the user changed intent meanwhile) — only the
// still-current command is re-asserted.
func (d *Daemon) schedulePrivacyTrapReassert(ctx context.Context, mode pixy.CameraState) {
	reassertCtx := context.WithoutCancel(ctx)

	go func() {
		timer := time.NewTimer(privacyTrapReassertDelay)
		defer timer.Stop()

		select {
		case <-reassertCtx.Done():
			return
		case <-timer.C:
		}

		d.mu.RLock()
		believed := d.state.Camera
		d.mu.RUnlock()

		if believed != mode {
			slog.Debug("privacy trap re-assert skipped, intent changed",
				"commanded", mode, "believed", believed)

			return
		}

		d.hidMu.Lock()
		err := d.writeTracking(reassertCtx, mode)
		d.hidMu.Unlock()

		if err != nil {
			slog.Warn("privacy trap re-assert failed", "mode", mode, "error", err)

			return
		}

		slog.Info("privacy trap re-asserted camera mode", "mode", mode)
	}()
}
