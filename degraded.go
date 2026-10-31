//go:build linux

package main

import (
	"log/slog"
	"slices"
)

// Degraded reason keys — stable identifiers surfaced by /api/health and
// /api/status (`reasons` array) and the web panel banner.
const (
	// degradedWebServer marks an HTTP listener death (ListenAndServe error
	// other than ErrServerClosed). The web surface is gone until restart.
	degradedWebServer = "web-server"
	// degradedSocketAccept marks the CLI socket accept loop hitting its
	// consecutive-failure ceiling: the daemon lives, but CLI commands may
	// not get served. Cleared again by a healthy accept.
	degradedSocketAccept = "socket-accept"
)

// markDegraded records a daemon-level degradation (guarded by d.mu). The map
// initializes lazily — a daemon that never degrades never allocates it.
func (d *Daemon) markDegraded(reason string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.degradedReasons == nil {
		d.degradedReasons = make(map[string]struct{})
	}

	if _, exists := d.degradedReasons[reason]; exists {
		return
	}

	d.degradedReasons[reason] = struct{}{}

	slog.Warn("daemon degraded", "reason", reason)
}

// clearDegraded removes a degradation mark, logging the recovery only when
// the mark actually existed.
func (d *Daemon) clearDegraded(reason string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.degradedReasons[reason]; !exists {
		return
	}

	delete(d.degradedReasons, reason)

	slog.Info("degradation cleared", "reason", reason)
}

// degradedReasonsSortedLocked returns the sorted degradation reasons. Callers
// must hold d.mu (read or write).
func (d *Daemon) degradedReasonsSortedLocked() []string {
	if len(d.degradedReasons) == 0 {
		return nil
	}

	reasons := make([]string, 0, len(d.degradedReasons))
	for reason := range d.degradedReasons {
		reasons = append(reasons, reason)
	}

	slices.Sort(reasons)

	return reasons
}
