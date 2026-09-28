//go:build linux

package main

import (
	"context"
)

// The HID circuit breaker lives here: the single definitions of the guard
// snapshot, failure accounting, and the threshold policy they share. Every
// HID transport path (config+commit in device.go, V2 single-report SETs in
// motor.go, V2 GETs in power.go) goes through these instead of restating the
// lock dance and threshold comparison.

// hidSendGuard snapshots the HID device handle and the circuit-breaker state
// under a read lock (acquire → copy → release); callers act on the copies.
// The boolean is true once hidFailCount has reached the breaker threshold.
func (d *Daemon) hidSendGuard() (HIDDevice, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.hidDev, d.hidFailCount >= hidCircuitBreakerThreshold
}

// recordHIDSendFailure is the single definition of a failed HID send: one
// breaker strike plus metrics, and — until the breaker trips — an immediate
// re-probe so a yanked device is replaced before the next command. The state
// change is broadcast after the lock is released.
func (d *Daemon) recordHIDSendFailure(ctx context.Context) {
	d.mu.Lock()
	d.hidFailCount++

	recordHIDFailure(ctx)

	if d.hidFailCount < hidCircuitBreakerThreshold {
		d.applyProbeResultLocked(probeDevices(d.config.ExtraProductIDs)) //nolint:contextcheck
	}
	d.mu.Unlock()
	d.broadcastStateChanged()
}
