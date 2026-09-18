//go:build linux

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// The battery/charge read surface (TODO #139). Heads are from the extracted
// official table; the RESPONSE framing is an assumption (head echo + payload)
// pinned at the hardware session (plan M27) — every read is best-effort and
// callers must degrade gracefully when the wired PIXY does not answer. The
// official app ships battery-over-HID in the macOS build only, so absence is
// an expected outcome, not an error condition.

// powerCacheTTL bounds how often status/waybar reads hit the HID device for
// battery data. One failed query per TTL is the steady-state cost on devices
// without battery-over-HID; successful readings cost the same.
const powerCacheTTL = time.Minute

type powerReading struct {
	Level    int  // percent, 0..100 (u8 on the wire)
	Charging bool // ChargeSta != 0 (enum values M27-verify)
}

type powerCache struct {
	mu        sync.RWMutex
	reading   powerReading
	expiresAt time.Time
}

func (c *powerCache) Get() (powerReading, bool) {
	now := time.Now()

	c.mu.RLock()
	valid := now.Before(c.expiresAt)
	reading := c.reading
	c.mu.RUnlock()

	return reading, valid
}

func (c *powerCache) Set(reading powerReading, ttl time.Duration) {
	c.mu.Lock()
	c.reading = reading
	c.expiresAt = time.Now().Add(ttl)
	c.mu.Unlock()
}

// powerStatus returns the cached reading if fresh, otherwise queries the
// device once and caches the result (successes AND authoritative failures —
// a device that does not answer battery queries would otherwise be re-probed
// on every status call). The bool reports whether a reading is available.
func (d *Daemon) powerStatus(ctx context.Context) (powerReading, bool) {
	if reading, ok := d.powerCache.Get(); ok {
		return reading, true
	}

	reading, err := d.queryPower(ctx)
	if err != nil {
		return powerReading{Level: 0, Charging: false}, false
	}

	d.powerCache.Set(reading, powerCacheTTL)

	return reading, true
}

// queryPower sends the battery and charge GET heads and parses the responses.
// Charge status is best-effort: a battery answer without a charge answer is
// still a valid reading.
func (d *Daemon) queryPower(ctx context.Context) (powerReading, error) {
	level, err := d.queryBatteryLevel(ctx)
	if err != nil {
		return powerReading{}, err
	}

	reading := powerReading{Level: level, Charging: false}

	charge, err := d.queryChargeStatus(ctx)
	if err == nil {
		reading.Charging = charge != 0
	}

	return reading, nil
}

//nolint:wrapcheck // pixy parse errors are already domain-wrapped
func (d *Daemon) queryBatteryLevel(ctx context.Context) (int, error) {
	resp, err := d.v2Read(ctx, pixy.V2GetBatteryLevel)
	if err != nil {
		return 0, err
	}

	return pixy.ParseBatteryLevel(resp)
}

//nolint:wrapcheck // pixy parse errors are already domain-wrapped
func (d *Daemon) queryChargeStatus(ctx context.Context) (byte, error) {
	resp, err := d.v2Read(ctx, pixy.V2GetChargeSta)
	if err != nil {
		return 0, err
	}

	return pixy.ParseChargeStatus(resp)
}

// v2Read sends a bare V2 GET head via SendRecv under the HID lock, with the
// same device/circuit guards as setMotorSpeed. It returns the RAW response;
// framing-specific parsing lives in internal/pixy so the simulator and the
// parsers evolve together when M27 pins the real framing.
func (d *Daemon) v2Read(ctx context.Context, head pixy.V2Head) ([]byte, error) {
	d.mu.RLock()
	hidDev := d.hidDev
	circuitOpen := d.hidFailCount >= hidCircuitBreakerThreshold
	d.mu.RUnlock()

	if hidDev == nil {
		return nil, fmt.Errorf("v2Read (no device): %w", pixy.ErrPIXYNotConnected)
	}

	if circuitOpen {
		return nil, fmt.Errorf("v2Read: %w", pixy.ErrPIXYNotConnected)
	}

	d.hidMu.Lock()
	resp, err := hidDev.SendRecv(ctx, head.Bytes())
	d.hidMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("v2Read %x: %w", head, err)
	}

	return resp, nil
}

// formatPower renders a reading for text surfaces: "87% (charging)" /
// "87% (discharging)".
func (r powerReading) String() string {
	state := "discharging"

	if r.Charging {
		state = "charging"
	}

	return fmt.Sprintf("%d%% (%s)", r.Level, state)
}

// appendPowerLine appends " battery=..." to a getStatus-style key=value
// string when a reading is available; status text stays stable otherwise.
func appendPowerLine(base string, reading powerReading, ok bool) string {
	if !ok {
		return base
	}

	return base + " battery=" + strings.ReplaceAll(reading.String(), " ", "")
}

// handleBatteryCommand implements `battery`: an explicit on-demand read of
// the battery/charge surface. Absence is reported as an explicit,
// non-alarming answer because battery-over-HID is not guaranteed on every
// PIXY (the official app ships it in the macOS build only).
func (d *Daemon) handleBatteryCommand(ctx context.Context) CommandResult {
	reading, ok := d.powerStatus(ctx)
	if !ok {
		return okResult(respBatteryUnavailable)
	}

	return okResult("battery: " + reading.String())
}

// Invalidate drops the cached reading so the next powerStatus call re-queries
// the device (used by tests and future pollers).
func (c *powerCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.expiresAt = time.Time{}
}
