//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// TestNewUnixListener_FailsFastOnUnlistenablePath pins #191a: a startup
// socket bind failure must surface as an error (Run refuses to run READY=1
// with a dead CLI surface; systemd Restart=on-failure recovers) with a
// non-zero exit code.
func TestNewUnixListener_FailsFastOnUnlistenablePath(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev)

	// Poison the state dir path with a regular file so MkdirAll cannot
	// succeed and the socket path can never exist.
	notDir := filepath.Join(t.TempDir(), "statedir")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("poison state dir: %v", err)
	}

	d.config.StateDir = notDir

	_, err := d.newUnixListener()
	if err == nil {
		t.Fatal("expected listen failure for unlistenable socket path")
	}

	if code := errorfamily.ExitCode(err); code == 0 {
		t.Errorf("expected non-zero exit code for startup listen failure, got %d", code)
	}
}

// TestNewUnixListener_HealthyBindRoundTrips proves the split kept the
// healthy path: bind, permissions, then a client connects and speaks one
// command through the accept loop.
func TestNewUnixListener_HealthyBindRoundTrips(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev)
	d.config.StateDir = t.TempDir()

	listener, err := d.newUnixListener()
	if err != nil {
		t.Fatalf("expected healthy bind, got: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	go d.acceptUnixLoop(ctx, listener)

	resp, err := pixy.SendCommand(ctx, d.config.SocketPath(), "status")
	if err != nil {
		t.Fatalf("send command over bound socket: %v", err)
	}

	if resp == "" {
		t.Error("expected non-empty status response")
	}
}
