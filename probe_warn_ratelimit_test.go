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
func TestProbeVideo4linux_UeventWarnRateLimited(t *testing.T) {
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
		if got := probeVideo4linux(root); got != "" {
			t.Fatalf("expected empty probe result, got %q", got)
		}
	}

	if got := strings.Count(buf.String(), "failed to read uevent"); got != 1 {
		t.Fatalf("expected exactly 1 rate-limited warn across 3 probes, got %d (buffer: %q)", got, buf.String())
	}
}
