//go:build linux

package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func shortSocketDir(t *testing.T) string {
	t.Helper()
	//nolint:usetesting // macOS t.TempDir() produces paths too long for Unix socket addresses
	dir, err := os.MkdirTemp("/tmp", "pxd-")
	if err != nil {
		t.Fatalf("create short temp dir: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}

func startSocketDaemon(t *testing.T) (*Daemon, pixy.Config) {
	t.Helper()
	cfg := pixy.Config{
		StateDir:      shortSocketDir(t),
		PollInterval:  2 * time.Second,
		DebounceCount: 3,
		WebAddr:       testWebAddr,
		AutoMode:      pixy.AutoFull,
		DefaultAudio:  pixy.AudioNC,
		Debug:         false,
	}

	daemon, daemonErr := NewDaemon(cfg)
	if daemonErr != nil {
		t.Fatalf("NewDaemon: %v", daemonErr)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		_ = daemon.listenUnix(ctx)
	}()

	for range 50 {
		if _, statErr := os.Stat(cfg.SocketPath()); statErr == nil {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	return daemon, cfg
}

func TestSocket_StatusCommand(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdStatus)
	assertSocketResponseHasPrefixes(t, resp, []string{"camera=", "audio=", "auto=", "device="})
}

func TestSocket_AutoToggleRoundTrip(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)

	resp := sendSC(t, cfg.SocketPath(), "auto-off")
	if resp != "auto mode: off" {
		t.Errorf("expected 'auto mode: off', got: %s", resp)
	}

	resp2 := sendSC(t, cfg.SocketPath(), "auto-on")
	if resp2 != "auto mode: full" {
		t.Errorf("expected 'auto mode: full', got: %s", resp2)
	}
}

func TestSocket_ProbeCommand(t *testing.T) {
	t.Parallel()
	daemon, cfg := startSocketDaemon(t)

	resp := sendSC(t, cfg.SocketPath(), cmdProbe)
	if daemon.videoDev != "" {
		if !strings.HasPrefix(resp, "device found:") {
			t.Errorf("expected 'device found: ...', got: %s", resp)
		}
	} else {
		if resp != "device not found" {
			t.Errorf("expected 'device not found', got: %s", resp)
		}
	}
}

func TestSocket_WaybarCommand(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdWaybar)
	assertSocketResponseHasPrefixes(t, resp, []string{`"text"`, `"tooltip"`, `"class"`})
}

func TestSocket_DeviceCommand(t *testing.T) {
	t.Parallel()
	daemon, cfg := startSocketDaemon(t)

	resp := sendSC(t, cfg.SocketPath(), cmdDevice)
	if daemon.videoDev != "" {
		if !strings.Contains(resp, daemon.videoDev) {
			t.Errorf("expected response containing %s, got: %s", daemon.videoDev, resp)
		}
	} else {
		if resp != "device not found" {
			t.Errorf("expected 'device not found', got: %s", resp)
		}
	}
}

func TestSocket_UnknownCommand(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), "foobar")
	assertSocketResponsePrefix(t, resp, "error: unknown command:", "socket response")
}

func TestSocket_StatusViaCommandReturnsStatus(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdStatus)
	assertSocketResponseContains(t, resp, "camera=", "socket response")
}

func TestSocket_CommandsNoDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		cmd string
	}{
		{cmdTrack, cmdTrack},

		{cmdPrivacy, cmdPrivacy},

		{cmdAudio, cmdAudio},

		{"gesture", cmdGestureOn},

		{cmdSync, cmdSync},

		{cmdCenter, cmdCenter},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			daemon, cfg := startSocketDaemon(t)

			if daemonHasDevices(daemon) {
				t.Skip("device connected")
			}

			resp := sendSC(t, cfg.SocketPath(), tc.cmd)
			assertSocketResponsePrefix(t, resp, "error:", "socket response")
		})
	}
}

func TestSocket_AudioInvalidMode(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)

	resp := sendSC(t, cfg.SocketPath(), "audio badmode")
	if !strings.HasPrefix(resp, "error: audio badmode:") {
		t.Errorf("expected error starting with 'error: audio badmode:', got: %s", resp)
	}
}

func TestSocket_AudioValidModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"nc", "live", "org"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			daemon, cfg := startSocketDaemon(t)

			if daemonHasDevices(daemon) {
				t.Skip("device connected, audio would succeed")
			}

			resp := sendSC(t, cfg.SocketPath(), "audio "+mode)
			assertSocketResponsePrefix(t, resp, "error:", "audio requires device")
		})
	}
}

func TestSocket_PanTiltZoom(t *testing.T) {
	t.Parallel()

	daemon, cfg := startSocketDaemon(t)
	if daemon.videoDev != "" {
		t.Skip("device connected")
	}

	for _, tc := range []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{"pan with value", "pan 10", true},
		{"tilt with value", "tilt -5", true},
		{"zoom with value", "zoom 200", true},
		{"pan missing value", pixy.AxisPan, false},
		{"tilt missing value", "tilt", false},
		{"zoom missing value", pixy.AxisZoom, false},
		{"pan invalid value", "pan abc", true},
		{"tilt invalid value", "tilt !", true},
		{"zoom invalid value", "zoom x", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := sendSC(t, cfg.SocketPath(), tc.cmd)
			if tc.wantErr {
				assertSocketResponsePrefix(t, resp, "error:", "socket response")
			} else if !strings.HasPrefix(resp, "usage:") {
				t.Errorf("expected usage for %q, got: %s", tc.cmd, resp)
			}
		})
	}
}

func TestSocket_TogglePrivacy(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdTogglePrivacy)
	assertCommandContainsAnyOf(t, resp,
		[]string{testStrPrivacy, testStrTracking}, "socket response")
}
