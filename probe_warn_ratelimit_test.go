//go:build linux

package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestProbeVideo4linux_UeventWarnRateLimited proves the absent-device probe
// WARN fires exactly once per path across repeated probes: while the PIXY is
// absent, autoManage re-probes every PollInterval and previously re-logged
// the same ENOENT line on every tick (~160/day in production, 2026-09-02).
func TestProbeVideo4linux_UeventWarnRateLimited(t *testing.T) { //nolint:paralleltest // mutates global slog + limiter
	var buf bytes.Buffer

	prev := slog.Default()
	prevLimiter := ueventWarnLimiter

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	ueventWarnLimiter = newWarnLimiter(time.Hour)

	t.Cleanup(func() {
		slog.SetDefault(prev)

		ueventWarnLimiter = prevLimiter
	})

	root := t.TempDir()

	// video0: a capture node (index 0) with NO device/uevent — the exact
	// shape that triggers the rate-limited warning.
	if err := os.MkdirAll(filepath.Join(root, testVideoDev0), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, testVideoDev0, "index"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		if got, _ := probeVideo4linux(root); got != "" {
			t.Fatalf("expected empty probe result, got %q", got)
		}
	}

	if got := strings.Count(buf.String(), "failed to read uevent"); got != 1 {
		t.Fatalf("expected exactly 1 rate-limited warn across 3 probes, got %d (buffer: %q)", got, buf.String())
	}
}

// TestWarnInaccessibleDevices_HintsOnPermissionDenied proves a device node
// that exists but cannot be opened produces an actionable warning naming the
// udev fix, instead of a later cryptic "Permission denied" HID error.
func TestWarnInaccessibleDevices_HintsOnPermissionDenied(t *testing.T) { //nolint:paralleltest // mutates global slog
	if os.Geteuid() == 0 {
		t.Skip("running as root — permission bits do not block opens")
	}

	var buf bytes.Buffer

	prev := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	locked := filepath.Join(t.TempDir(), "hidraw9")
	if err := os.WriteFile(locked, nil, 0o000); err != nil {
		t.Fatal(err)
	}

	warnInaccessibleDevices(probeResult{HidrawDev: locked})

	out := buf.String()
	if !strings.Contains(out, "not accessible") {
		t.Errorf("expected accessibility warning, got %q", out)
	}

	if !strings.Contains(out, "udev") {
		t.Errorf("warning lacks the udev fix hint: %q", out)
	}
}

// TestWarnInaccessibleDevicesLimited_RateLimitsHotplug proves the
// uevent-appear path warns once per node per interval: a flapping USB
// connection re-triggers the appear branch, and an un-limited warning would
// flood the journal on replug storms.
func TestWarnInaccessibleDevicesLimited_RateLimitsHotplug(t *testing.T) { //nolint:paralleltest // mutates global slog
	if os.Geteuid() == 0 {
		t.Skip("running as root — permission bits do not block opens")
	}

	var buf bytes.Buffer

	prev := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	t.Cleanup(func() { slog.SetDefault(prev) })

	locked := filepath.Join(t.TempDir(), "hidraw9")

	if err := os.WriteFile(locked, nil, 0o000); err != nil {
		t.Fatal(err)
	}

	limiter := newWarnLimiter(time.Hour)
	now := time.Unix(0, 0)
	limiter.now = func() time.Time { return now }

	probe := probeResult{HidrawDev: locked}

	warnInaccessibleDevicesLimited(probe, limiter) // first appear: warns

	now = now.Add(time.Minute)

	warnInaccessibleDevicesLimited(probe, limiter) // replug within interval: silent

	now = now.Add(2 * time.Hour)

	warnInaccessibleDevicesLimited(probe, limiter) // much later: warns again

	got := strings.Count(buf.String(), "not accessible")
	if got != 2 {
		t.Fatalf("expected exactly 2 warns (initial + post-interval), got %d (buffer: %q)", got, buf.String())
	}
}
