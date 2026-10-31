//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TestNewUnixListener_FailsFastOnUnlistenablePath pins #191a: a startup
// socket bind failure must surface as an error (Run refuses to run READY=1
// with a dead CLI surface; systemd Restart=on-failure recovers) with a
// non-zero exit code. The healthy path is covered end-to-end by
// startSocketDaemon in socket_test.go, which now binds synchronously.
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
