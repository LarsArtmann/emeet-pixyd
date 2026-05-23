package pixy

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func runValidTests[T any](
	t *testing.T,
	tests []struct {
		input T
		want  bool
	},
	typeName string,
	valid func(T) bool,
) {
	t.Helper()
	for _, tc := range tests {
		got := valid(tc.input)
		if got != tc.want {
			t.Errorf("%s(%v).Valid() = %v, want %v", typeName, tc.input, got, tc.want)
		}
	}
}

func TestCameraState_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input CameraState
		want  bool
	}{
		{StateIdle, true},
		{StateTracking, true},
		{StatePrivacy, true},
		{StateOffline, true},
		{CameraState("unknown"), false},
		{CameraState(""), false},
		{CameraState("IDLE"), false},
	}

	runValidTests(t, tests, "CameraState", func(v CameraState) bool { return v.Valid() })
}

func TestAudioMode_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input AudioMode
		want  bool
	}{
		{AudioNC, true},
		{AudioLive, true},
		{AudioOriginal, true},
		{AudioMode("unknown"), false},
		{AudioMode(""), false},
		{AudioMode("NC"), false},
	}

	runValidTests(t, tests, "AudioMode", func(v AudioMode) bool { return v.Valid() })
}

func TestAudioMode_Next(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input AudioMode
		want  AudioMode
	}{
		{AudioNC, AudioLive},
		{AudioLive, AudioOriginal},
		{AudioOriginal, AudioNC},
		{AudioMode("unknown"), AudioNC},
		{AudioMode(""), AudioNC},
	}

	for _, tc := range tests {
		if got := tc.input.Next(); got != tc.want {
			t.Errorf("AudioMode(%q).Next() = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestAudioMode_Next_CyclesThrough(t *testing.T) {
	t.Parallel()

	mode := AudioNC
	for _, want := range []AudioMode{AudioLive, AudioOriginal, AudioNC} {
		mode = mode.Next()
		if mode != want {
			t.Errorf("Next() = %v, want %v", mode, want)
		}
	}
}

func assertParseAudioMode(t *testing.T, input string, want AudioMode) {
	t.Helper()
	got, err := ParseAudioMode(input)
	if err != nil {
		t.Errorf("ParseAudioMode(%q) unexpected error: %v", input, err)
	}
	if got != want {
		t.Errorf("ParseAudioMode(%q) = %v, want %v", input, got, want)
	}
}

func assertParseCameraState(t *testing.T, input string, want CameraState) {
	t.Helper()
	got, err := ParseCameraState(input)
	if err != nil {
		t.Errorf("ParseCameraState(%q) unexpected error: %v", input, err)
	}
	if got != want {
		t.Errorf("ParseCameraState(%q) = %v, want %v", input, got, want)
	}
}

func expectParseCameraStateError(t *testing.T, input string) {
	t.Helper()
	_, err := ParseCameraState(input)
	if err == nil {
		t.Errorf("ParseCameraState(%q) expected error, got nil", input)
	}
	if !errors.Is(err, ErrInvalidCameraState) {
		t.Errorf("ParseCameraState(%q) error = %v, want ErrInvalidCameraState", input, err)
	}
}

func TestParseAudioMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    AudioMode
		wantErr bool
	}{
		{"nc", AudioNC, false},
		{"live", AudioLive, false},
		{"org", AudioOriginal, false},
		{"unknown", "", true},
		{"", "", true},
		{"NC", "", true},
		{"ORIGINAL", "", true},
	}

	for _, tc := range tests {
		if tc.wantErr {
			_, err := ParseAudioMode(tc.input)
			if err == nil {
				t.Errorf("ParseAudioMode(%q) expected error, got nil", tc.input)
			}

			if !errors.Is(err, ErrInvalidAudioMode) {
				t.Errorf("ParseAudioMode(%q) error = %v, want ErrInvalidAudioMode", tc.input, err)
			}

			continue
		}

		assertParseAudioMode(t, tc.input, tc.want)
	}
}

func TestParseAudioMode_OrgMapping(t *testing.T) {
	t.Parallel()

	got, err := ParseAudioMode("org")
	if err != nil {
		t.Fatalf("ParseAudioMode(\"org\") unexpected error: %v", err)
	}

	if got != AudioOriginal {
		t.Errorf("ParseAudioMode(\"org\") = %q, want %q", got, AudioOriginal)
	}

	if string(got) != "original" {
		t.Errorf("ParseAudioMode(\"org\").string() = %q, want %q", string(got), "original")
	}
}

func TestParseCameraState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    CameraState
		wantErr bool
	}{
		{"idle", StateIdle, false},
		{"tracking", StateTracking, false},
		{"privacy", StatePrivacy, false},
		{"offline", StateOffline, false},
		{"unknown", "", true},
		{"", "", true},
		{"IDLE", "", true},
		{"Offline", "", true},
	}

	for _, tc := range tests {
		if tc.wantErr {
			expectParseCameraStateError(t, tc.input)
			continue
		}
		assertParseCameraState(t, tc.input, tc.want)
	}
}

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

const testStateDir = "/tmp/test"

func TestConfig_StateFile(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	c := Config{StateDir: testStateDir}

	want := testStateDir + "/state.json"
	if got := c.StateFile(); got != want {
		t.Errorf("Config.StateFile() = %v, want %v", got, want)
	}
}

func TestConfig_SocketPath(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
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

	//nolint:exhaustruct
	valid := Config{
		StateDir:      testStateDir,
		PollInterval:  time.Second,
		DebounceCount: 3,
		WebAddr:       "127.0.0.1:8090",
	}

	t.Run("valid default config", func(t *testing.T) {
		t.Parallel()

		if err := DefaultConfig().Validate(); err != nil {
			t.Fatalf("DefaultConfig().Validate() = %v", err)
		}
	})

	t.Run("valid custom config", func(t *testing.T) {
		t.Parallel()

		if err := valid.Validate(); err != nil {
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
