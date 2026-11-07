//go:build linux

package main

import (
	"log/slog"
	"slices"
)

// DegradedReason is a stable degradation identifier surfaced by
// /api/health and /api/status (`reasons` array) and the web panel banner.
// The wire values are plain strings; the type only guards the mark/clear
// call sites and the reasons map against typo'd keys.
type DegradedReason string

// Degraded reason identifiers — stable wire values.
const (
	// degradedWebServer marks an HTTP listener death (ListenAndServe error
	// other than ErrServerClosed). The web surface is gone until restart.
	degradedWebServer DegradedReason = "web-server"
	// degradedSocketAccept marks the CLI socket accept loop hitting its
	// consecutive-failure ceiling: the daemon lives, but CLI commands may
	// not get served. Cleared again by a healthy accept.
	degradedSocketAccept DegradedReason = "socket-accept"
	// degradedExternalDepPrefix fronts one reason per optional helper
	// binary missing at startup (constructed by ExternalDepReason, e.g.
	// "extdep:v4l2-ctl"): each miss disables a real capability until the
	// binary is installed and the daemon restarts.
	degradedExternalDepPrefix = "extdep:"
)

// ExternalDepReason returns the degradation reason for a missing optional
// helper binary (e.g. "extdep:v4l2-ctl"), the single constructor for the
// extdep family so the prefix spelling cannot drift.
func ExternalDepReason(binary string) DegradedReason {
	return DegradedReason(degradedExternalDepPrefix + binary)
}

// markDegraded records a daemon-level degradation (guarded by d.mu). The map
// initializes lazily — a daemon that never degrades never allocates it.
func (d *Daemon) markDegraded(reason DegradedReason) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.degradedReasons == nil {
		d.degradedReasons = make(map[DegradedReason]struct{})
	}

	if _, exists := d.degradedReasons[reason]; exists {
		return
	}

	d.degradedReasons[reason] = struct{}{}

	slog.Warn("daemon degraded", "reason", reason)
}

// clearDegraded removes a degradation mark, logging the recovery only when
// the mark actually existed.
func (d *Daemon) clearDegraded(reason DegradedReason) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.degradedReasons[reason]; !exists {
		return
	}

	delete(d.degradedReasons, reason)

	slog.Info("degradation cleared", "reason", reason)
}

// degradedReasonsSortedLocked returns the sorted degradation reasons as
// their wire form ([]string — the /api contract). Callers must hold d.mu
// (read or write).
func (d *Daemon) degradedReasonsSortedLocked() []string {
	if len(d.degradedReasons) == 0 {
		return nil
	}

	reasons := make([]string, 0, len(d.degradedReasons))
	for reason := range d.degradedReasons {
		reasons = append(reasons, string(reason))
	}

	slices.Sort(reasons)

	return reasons
}
