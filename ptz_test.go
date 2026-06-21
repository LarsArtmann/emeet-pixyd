//go:build linux

package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func assertV4L2CommandContains(t *testing.T, cmd *exec.Cmd, substrings []string) {
	t.Helper()

	cmdStr := cmd.String()
	for _, s := range substrings {
		if !strings.Contains(cmdStr, s) {
			t.Errorf("command should contain %q, got: %s", s, cmdStr)
		}
	}
}

func TestV4L2Set_CommandFormat(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(
		context.Background(), "v4l2-ctl", "-d", "/dev/video0",
		"--set-ctrl=pan_absolute=0",
	)
	assertV4L2CommandContains(t, cmd, []string{"v4l2-ctl", "/dev/video0"})
}

func TestParsePTZValues_InvalidDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	ptz := d.parsePTZValues(context.Background(), "/dev/nonexistent")
	if ptz.Pan != 0 || ptz.Tilt != 0 || ptz.Zoom != 0 {
		t.Errorf("expected zero values for nonexistent device, got pan=%d tilt=%d zoom=%d",
			ptz.Pan, ptz.Tilt, ptz.Zoom)
	}
}

func TestV4L2DegreesPerUnit(t *testing.T) {
	t.Parallel()

	if v4l2UnitsPerDegree != 3600 {
		t.Errorf("v4l2UnitsPerDegree = %d, want 3600", v4l2UnitsPerDegree)
	}
}
