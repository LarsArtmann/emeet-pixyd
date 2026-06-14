//go:build linux

package pixy

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestDefaultState(t *testing.T) {
	t.Parallel()

	s := DefaultState()

	if s.Camera != StatePrivacy {
		t.Errorf("DefaultState().Camera = %v, want %v", s.Camera, StatePrivacy)
	}

	if s.Audio != AudioNC {
		t.Errorf("DefaultState().Audio = %v, want %v", s.Audio, AudioNC)
	}

	if s.Gesture {
		t.Error("DefaultState().Gesture = true, want false")
	}

	if s.InCall {
		t.Error("DefaultState().InCall = true, want false")
	}

	if s.AutoMode != AutoFull {
		t.Errorf("DefaultState().AutoMode = %q, want %q", s.AutoMode, AutoFull)
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	c := DefaultConfig()

	if c.StateDir != DefaultStateDir {
		t.Errorf("DefaultConfig().StateDir = %v, want %v", c.StateDir, DefaultStateDir)
	}

	if c.PollInterval != DefaultPollInterval {
		t.Errorf("DefaultConfig().PollInterval = %v, want %v", c.PollInterval, DefaultPollInterval)
	}

	if c.DebounceCount != DefaultDebounceCount {
		t.Errorf(
			"DefaultConfig().DebounceCount = %v, want %v",
			c.DebounceCount,
			DefaultDebounceCount,
		)
	}

	if c.WebAddr != DefaultWebAddr {
		t.Errorf("DefaultConfig().WebAddr = %v, want %v", c.WebAddr, DefaultWebAddr)
	}
}

func TestConfig_SocketPath(t *testing.T) {
	t.Parallel()

	c := Config{StateDir: testStateDir}

	want := testStateDir + "/control.sock"
	if got := c.SocketPath(); got != want {
		t.Errorf("Config.SocketPath() = %v, want %v", got, want)
	}
}

func TestSetDeadline(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()

	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})

	err := SetDeadline(client, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SetDeadline() unexpected error: %v", err)
	}
}

func TestSendCommand_DialFailure(t *testing.T) {
	t.Parallel()

	_, err := SendCommand(context.Background(), "/tmp/nonexistent-socket-path-test.sock", "status")
	if err == nil {
		t.Fatal("SendCommand() expected error for nonexistent socket, got nil")
	}
}

func TestSendCommand_EndToEnd(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	socketPath := tmpDir + "/test.sock"

	var lc net.ListenConfig

	listener, err := lc.Listen(context.Background(), "unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		buf := make([]byte, SocketBufSize)

		n, readErr := conn.Read(buf)
		if readErr != nil {
			return
		}

		_, _ = conn.Write([]byte("response:" + string(buf[:n])))
	}()

	got, err := SendCommand(context.Background(), socketPath, "hello")
	if err != nil {
		t.Fatalf("SendCommand() unexpected error: %v", err)
	}

	if got != "response:hello" {
		t.Errorf("SendCommand() = %q, want %q", got, "response:hello")
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := Config{
		StateDir:      testStateDir,
		PollInterval:  time.Second,
		DebounceCount: 3,
		WebAddr:       "127.0.0.1:8090",
		AutoMode:      AutoFull,
		DefaultAudio:  AudioNC,
	}

	t.Run("valid default config", func(t *testing.T) {
		t.Parallel()

		err := DefaultConfig().Validate()
		if err != nil {
			t.Fatalf("DefaultConfig().Validate() = %v", err)
		}
	})

	t.Run("valid custom config", func(t *testing.T) {
		t.Parallel()

		err := valid.Validate()
		if err != nil {
			t.Fatalf("Validate() = %v", err)
		}
	})

	tests := []struct {
		name   string
		mutate func(*Config)
		want   error
	}{
		{"empty state dir", func(c *Config) { c.StateDir = "" }, ErrStateDirEmpty},
		{"zero poll interval", func(c *Config) { c.PollInterval = 0 }, ErrPollIntervalZero},
		{"negative poll interval", func(c *Config) { c.PollInterval = -1 }, ErrPollIntervalZero},
		{"zero debounce", func(c *Config) { c.DebounceCount = 0 }, ErrDebounceCountZero},
		{"negative debounce", func(c *Config) { c.DebounceCount = -1 }, ErrDebounceCountZero},
		{"empty web addr", func(c *Config) { c.WebAddr = "" }, ErrWebAddrEmpty},
		{
			"invalid auto mode",
			func(c *Config) { c.AutoMode = AutoMode("bogus") },
			ErrInvalidAutoMode,
		},
		{
			"invalid default audio",
			func(c *Config) { c.DefaultAudio = AudioMode("bogus") },
			ErrInvalidDefaultAudio,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			tc.mutate(&cfg)

			err := cfg.Validate()
			if !errors.Is(err, tc.want) {
				t.Errorf("Validate() = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestConfigFromEnv_DefaultsWhenUnset(t *testing.T) {
	t.Parallel()

	cfg := ConfigFromEnv()
	def := DefaultConfig()

	if cfg != def {
		t.Errorf("ConfigFromEnv() with no env vars = %+v, want %+v", cfg, def)
	}
}

func TestConfigFromEnv_OverridesFromEnv(t *testing.T) {
	t.Setenv("EMEET_PIXYD_STATE_DIR", "/tmp/custom-state")
	t.Setenv("EMEET_PIXYD_WEB_ADDR", "0.0.0.0:9999")
	t.Setenv("EMEET_PIXYD_POLL_INTERVAL", "5s")
	t.Setenv("EMEET_PIXYD_DEBOUNCE_COUNT", "7")
	t.Setenv("EMEET_PIXYD_DEBUG", "true")

	cfg := ConfigFromEnv()

	if cfg.StateDir != "/tmp/custom-state" {
		t.Errorf("StateDir = %q, want %q", cfg.StateDir, "/tmp/custom-state")
	}

	if cfg.WebAddr != "0.0.0.0:9999" {
		t.Errorf("WebAddr = %q, want %q", cfg.WebAddr, "0.0.0.0:9999")
	}

	if cfg.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 5*time.Second)
	}

	if cfg.DebounceCount != 7 {
		t.Errorf("DebounceCount = %d, want 7", cfg.DebounceCount)
	}

	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestConfigFromEnv_DebugAcceptsOne(t *testing.T) {
	t.Setenv("EMEET_PIXYD_DEBUG", "1")

	cfg := ConfigFromEnv()

	if !cfg.Debug {
		t.Error("Debug = false, want true for '1'")
	}
}

func TestConfigFromEnv_IgnoresInvalidValues(t *testing.T) {
	t.Setenv("EMEET_PIXYD_POLL_INTERVAL", "not-a-duration")
	t.Setenv("EMEET_PIXYD_DEBOUNCE_COUNT", "not-a-number")

	cfg := ConfigFromEnv()
	def := DefaultConfig()

	if cfg.PollInterval != def.PollInterval {
		t.Errorf(
			"PollInterval = %v, want default %v (invalid env ignored)",
			cfg.PollInterval, def.PollInterval,
		)
	}

	if cfg.DebounceCount != def.DebounceCount {
		t.Errorf(
			"DebounceCount = %d, want default %d (invalid env ignored)",
			cfg.DebounceCount, def.DebounceCount,
		)
	}
}

func TestConfigFromEnv_PartialOverride(t *testing.T) {
	t.Setenv("EMEET_PIXYD_STATE_DIR", "/custom")

	cfg := ConfigFromEnv()

	if cfg.StateDir != "/custom" {
		t.Errorf("StateDir = %q, want %q", cfg.StateDir, "/custom")
	}

	if cfg.WebAddr != DefaultWebAddr {
		t.Errorf("WebAddr = %q, want default %q", cfg.WebAddr, DefaultWebAddr)
	}
}

func TestDefaultConfig_NewFields(t *testing.T) {
	t.Parallel()

	c := DefaultConfig()

	if c.AutoMode != AutoFull {
		t.Errorf("DefaultConfig().AutoMode = %q, want %q", c.AutoMode, AutoFull)
	}

	if c.DefaultAudio != AudioNC {
		t.Errorf("DefaultConfig().DefaultAudio = %q, want %q", c.DefaultAudio, AudioNC)
	}
}

func TestDefaultConfig_DoesNotDriftFromDefaultState(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	state := DefaultState()

	if cfg.AutoMode != state.AutoMode {
		t.Errorf(
			"DefaultConfig().AutoMode (%q) != DefaultState().AutoMode (%q): defaults drifted",
			cfg.AutoMode, state.AutoMode,
		)
	}

	if cfg.DefaultAudio != state.Audio {
		t.Errorf(
			"DefaultConfig().DefaultAudio (%q) != DefaultState().Audio (%q): defaults drifted",
			cfg.DefaultAudio, state.Audio,
		)
	}
}

func TestConfigFromEnv_AutoAndAudio(t *testing.T) {
	t.Setenv("EMEET_PIXYD_AUTO", "false")
	t.Setenv("EMEET_PIXYD_DEFAULT_AUDIO", "live")

	cfg := ConfigFromEnv()

	if cfg.AutoMode != AutoOff {
		t.Errorf("AutoMode = %q, want %q", cfg.AutoMode, AutoOff)
	}

	if cfg.DefaultAudio != AudioLive {
		t.Errorf("DefaultAudio = %q, want %q", cfg.DefaultAudio, AudioLive)
	}
}

func TestConfigFromEnv_InvalidAudioIgnored(t *testing.T) {
	t.Setenv("EMEET_PIXYD_DEFAULT_AUDIO", "invalid")

	cfg := ConfigFromEnv()

	if cfg.DefaultAudio != AudioNC {
		t.Errorf("DefaultAudio = %q, want default %q", cfg.DefaultAudio, AudioNC)
	}
}
