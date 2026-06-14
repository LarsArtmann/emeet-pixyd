//go:build linux

package main

import (
	"errors"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestConfigPaths(t *testing.T) {
	t.Parallel()

	cfg := pixy.Config{
		StateDir:      "/tmp/test-pixyd",
		PollInterval:  pixy.DefaultPollInterval,
		DebounceCount: pixy.DefaultDebounceCount,
	}
	if cfg.StateFile() != "/tmp/test-pixyd/state.json" {
		t.Errorf("unexpected StateFile: %s", cfg.StateFile())
	}

	if cfg.SocketPath() != "/tmp/test-pixyd/control.sock" {
		t.Errorf("unexpected SocketPath: %s", cfg.SocketPath())
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := pixy.DefaultConfig()
	if cfg.StateDir != pixy.DefaultStateDir {
		t.Errorf("expected StateDir=%s, got %s", pixy.DefaultStateDir, cfg.StateDir)
	}

	if cfg.PollInterval != pixy.DefaultPollInterval {
		t.Errorf("expected PollInterval=%v, got %v", pixy.DefaultPollInterval, cfg.PollInterval)
	}

	if cfg.DebounceCount != pixy.DefaultDebounceCount {
		t.Errorf("expected DebounceCount=%d, got %d", pixy.DefaultDebounceCount, cfg.DebounceCount)
	}
}

func TestParseAudioMode(t *testing.T) {
	t.Parallel()

	tests := []parseTestCase[pixy.AudioMode]{
		{"nc", pixy.AudioNC, false},
		{audioModeLive, pixy.AudioLive, false},
		{audioModeOrg, pixy.AudioOriginal, false},
		{"original", pixy.AudioOriginal, false},
		{"NC", pixy.AudioNC, false},
		{"LIVE", pixy.AudioLive, false},
		{testStrUnknown, "", true},
		{"", "", true},
	}
	runParseTests(t, "pixy.ParseAudioMode", pixy.ParseAudioMode, tests)
}

func TestParseCameraState(t *testing.T) {
	t.Parallel()

	tests := []parseTestCase[pixy.CameraState]{
		{"idle", pixy.StateIdle, false},
		{"tracking", pixy.StateTracking, false},
		{"privacy", pixy.StatePrivacy, false},
		{"offline", pixy.StateOffline, false},
		{testStrUnknown, "", true},
		{"", "", true},
		{"PRIVACY", pixy.StatePrivacy, false},
		{"Tracking", pixy.StateTracking, false},
	}
	runParseTests(t, "pixy.ParseCameraState", pixy.ParseCameraState, tests)
}

func TestDefaultStateValues(t *testing.T) {
	t.Parallel()

	s := pixy.DefaultState()
	if s.Camera != pixy.StatePrivacy {
		t.Errorf("expected default camera=privacy, got %s", s.Camera)
	}

	if s.Audio != pixy.AudioNC {
		t.Errorf("expected default audio=nc, got %s", s.Audio)
	}

	if s.Gesture != false {
		t.Error("expected default gesture=false")
	}

	if s.InCall != false {
		t.Error("expected default inCall=false")
	}

	if s.AutoMode != pixy.AutoFull {
		t.Error("expected default autoMode=true")
	}
}

func TestSetDeadlineError(t *testing.T) {
	t.Parallel()

	conn := &mockConn{setDeadlineErr: errors.New("deadline error")}

	err := pixy.SetDeadline(conn, time.Second)
	if err == nil {
		t.Error("expected error from SetDeadline with failing conn")
	}
}
