//go:build linux

package main

import (
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func assertAutoModeEquals(t *testing.T, d *Daemon, want pixy.AutoMode) {
	t.Helper()
	d.mu.RLock()
	got := d.state.AutoMode
	d.mu.RUnlock()

	if got != want {
		t.Errorf("AutoMode = %s, want %s", got, want)
	}
}

func TestHandleAutoCommand_SetMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{cmdAuto, "full"})
	notError(t, resp)

	assertCommandContains(t, resp.String(), "full", "response")

	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_InvalidMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	resp := d.handleAutoCommand([]string{"auto", "invalid-mode"})
	assertCommandContains(t, resp.String(), "usage:", "response")
}

func TestHandleAutoCommand_ToggleOff(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{"auto-on"})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_ToggleOn(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoFull
	})

	resp := d.handleAutoCommand([]string{cmdAutoOff})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoOff)
}

func TestHandleAutoCommand_ToggleAuto(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", withAutoOff())

	resp := d.handleAutoCommand([]string{cmdToggleAuto})
	notError(t, resp)
	assertAutoModeEquals(t, d, pixy.AutoFull)
}

func TestHandleAutoCommand_BareAutoShowsCurrentMode(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.state.AutoMode = pixy.AutoTrackingOnly
	})

	resp := d.handleAutoCommand([]string{cmdAuto})
	notError(t, resp)
	assertCommandContains(t, resp.String(), "tracking-only", "response")
	assertAutoModeEquals(t, d, pixy.AutoTrackingOnly)
}
