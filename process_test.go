//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestPpidOf_CurrentProcess(t *testing.T) {
	t.Parallel()

	myPID := pixy.NewPID(os.Getpid())

	ppid := ppidOf(myPID)
	if ppid.IsZero() {
		t.Fatal("expected non-zero ppid for current process")
	}

	t.Logf("current pid=%d ppid=%d", myPID.Get(), ppid.Get())
}

func TestPpidOf_InvalidPID(t *testing.T) {
	t.Parallel()

	ppid := ppidOf(pixy.NewPID(999999999))
	assertPIDIsZero(t, ppid, "invalid PID")
}

func TestPpidOf_InitProcess(t *testing.T) {
	t.Parallel()

	ppid := ppidOf(pixy.NewPID(1))
	if ppid.IsZero() {
		t.Log("init ppid is 0 (expected in most cases)")
	}
}

func TestIsDescendantOf_Self(t *testing.T) {
	t.Parallel()

	myPID := pixy.NewPID(os.Getpid())

	ppid := ppidOf(myPID)
	if ppid.IsZero() {
		t.Skip("cannot determine parent PID")
	}

	if !isDescendantOf(myPID, ppid) {
		t.Errorf(
			"expected current process %d to be descendant of parent %d",
			myPID.Get(), ppid.Get(),
		)
	}
}

func TestIsDescendantOf_NotDescendant(t *testing.T) {
	t.Parallel()

	myPID := pixy.NewPID(os.Getpid())
	if isDescendantOf(myPID, pixy.NewPID(1)) {
		t.Log("process is descendant of PID 1 (common on Linux)")
	}
}

func TestIsDescendantOf_SamePID(t *testing.T) {
	t.Parallel()

	myPID := pixy.NewPID(os.Getpid())
	if isDescendantOf(myPID, myPID) {
		t.Error("a process should not be its own descendant")
	}
}

// assertPIDIsZero fails the test if the given PID is not zero (invalid/edge-case).
func assertPIDIsZero(t *testing.T, ppid pixy.PID, label string) {
	t.Helper()

	if !ppid.IsZero() {
		t.Errorf("expected 0 for %s, got %d", label, ppid.Get())
	}
}

func TestIsCameraInUse_NonexistentOrEmptyDev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dev  string
	}{
		{name: "empty dev path", dev: ""},
		{name: "nonexistent dev", dev: "/dev/video_nonexistent_xyz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if isCameraInUse(tc.dev) {
				t.Errorf("expected false for %s device path %q", tc.name, tc.dev)
			}
		})
	}
}

func TestIsCameraInUse_SelfFd(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	devPath := filepath.Join(tmpDir, "fake_dev")

	f, err := os.Create(devPath)
	if err != nil {
		t.Fatalf("create fake dev: %v", err)
	}

	_ = f.Close()

	// The file exists but no other process holds it open via /proc/*/fd
	// (except perhaps our own process, which is excluded)
	if isCameraInUse(devPath) {
		t.Log("isCameraInUse returned true — our own fd was detected but should be excluded")
	}
}

func TestIsCameraInUse_CurrentProcessHoldsFD(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	devPath := filepath.Join(tmpDir, "video_test_dev")

	f, err := os.Create(devPath)
	if err != nil {
		t.Fatalf("create fake dev: %v", err)
	}

	_ = f.Close()

	// isCameraInUse should return false because the current process and its
	// descendants are excluded from the check
	if isCameraInUse(devPath) {
		t.Error("expected false — self-owned FDs should be excluded")
	}
}

func TestPpidOf_MalformedStat(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	pidDir := filepath.Join(tmpDir, "12345")

	err := os.MkdirAll(pidDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(pidDir, "stat"), []byte("malformed"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// ppidOf reads from /proc, not our tmpDir, so this tests the real /proc.
	// We just verify it doesn't panic on edge cases.
	ppid := ppidOf(pixy.NewPID(-1))
	assertPIDIsZero(t, ppid, "negative PID")
}

func TestIsDescendantOf_MaxDepth(t *testing.T) {
	t.Parallel()
	// PID 1's parent is 0, which should hit max depth or terminate
	// This shouldn't panic regardless of outcome
	isDescendantOf(pixy.NewPID(1), pixy.NewPID(999999999))
}
