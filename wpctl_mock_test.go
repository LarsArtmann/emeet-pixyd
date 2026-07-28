//go:build linux

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// wpctlMock is a test double for CommandRunner that simulates PipeWire
// wpctl behavior. It returns configurable `wpctl status` output and records
// `wpctl set-default` calls.
type wpctlMock struct {
	statusOutput string // simulated `wpctl status` output
	setDefaultID string // last ID passed to `wpctl set-default`
	lookPathOk   bool   // whether LookPath reports the binary as found
}

func newWpctlMock(includePixy bool) *wpctlMock {
	var status string
	if includePixy {
		status = strings.Join([]string{
			"PipeWire 'pipewire-0' [1.2.7]",
			"Audio",
			"  Sinks:",
			"    * 41. Built-in Audio Stereo",
			"  Sources:",
			"    * 42. EMEET PIXY Conference Camera",
			"    43. Built-in Microphone",
			"",
		}, "\n")
	} else {
		status = strings.Join([]string{
			"PipeWire 'pipewire-0' [1.2.7]",
			"Audio",
			"  Sources:",
			"    * 43. Built-in Microphone",
			"",
		}, "\n")
	}

	return &wpctlMock{statusOutput: status, lookPathOk: true}
}

func (w *wpctlMock) Run(_ context.Context, name string, args ...string) error {
	if name == wpctl && len(args) >= 2 && args[0] == "set-default" {
		w.setDefaultID = args[1]
	}

	return nil
}

func (w *wpctlMock) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if name == wpctl && len(args) > 0 && args[0] == "status" {
		return []byte(w.statusOutput), nil
	}

	return nil, nil
}

func (w *wpctlMock) LookPath(string) (string, error) {
	if w.lookPathOk {
		return "/usr/bin/wpctl", nil
	}

	return "", errLookPathFailed
}

var errLookPathFailed = strings.TrimSpace("wpctl not found")

// withWpctlMock wires a wpctlMock into the daemon's DI and sets findSource/setSource
// to use the REAL daemon methods (which call commander.Output/Run internally).
func withWpctlMock(mock *wpctlMock) testDaemonOption {
	return func(d *Daemon) {
		d.deps.commander = mock
		d.deps.findSource = d.findPixySource
		d.deps.setSource = d.setDefaultSource
	}
}

func TestWpctlMock_FindPixySource_ParsesStatus(t *testing.T) {
	t.Parallel()

	mock := newWpctlMock(true)
	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev, withWpctlMock(mock))

	source, err := d.findPixySource(context.Background())
	if err != nil {
		t.Fatalf("expected source found, got: %v", err)
	}

	if source.Get() != "42" {
		t.Errorf("expected source ID 42, got %s", source.Get())
	}
}

func TestWpctlMock_FindPixySource_NotFound(t *testing.T) {
	t.Parallel()

	mock := newWpctlMock(false)
	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev, withWpctlMock(mock))

	_, err := d.findPixySource(context.Background())
	if err == nil {
		t.Error("expected ErrAudioSourceNotFound when no PIXY in wpctl status")
	}
}

func TestWpctlMock_SetDefaultSource_RecordsCall(t *testing.T) {
	t.Parallel()

	mock := newWpctlMock(true)
	d := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, testHIDDev, withWpctlMock(mock))

	d.setDefaultSource(context.Background(), pixy.NewSourceID("42"))

	if mock.setDefaultID != "42" {
		t.Errorf("expected setDefaultID=42, got %s", mock.setDefaultID)
	}
}

func TestWpctlMock_LookPath(t *testing.T) {
	t.Parallel()

	mock := newWpctlMock(true)

	path, err := mock.LookPath(wpctl)
	if err != nil {
		t.Errorf("expected LookPath to succeed: %v", err)
	}

	if path == "" {
		t.Error("expected non-empty path from LookPath")
	}
}

func TestWpctlMock_StatusFormatMatchesParser(t *testing.T) {
	t.Parallel()

	mock := newWpctlMock(true)

	var found bool

	for line := range strings.SplitSeq(mock.statusOutput, "\n") {
		if isPixyName(line) {
			found = true

			break
		}
	}

	if !found {
		t.Error("expected mock wpctl status to contain a PIXY device name")
	}
}
