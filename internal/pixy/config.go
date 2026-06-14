package pixy

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds daemon configuration parameters.
// Values are read from environment variables by ConfigFromEnv().
type Config struct {
	StateDir      string
	PollInterval  time.Duration
	DebounceCount int
	WebAddr       string
	Debug         bool
	AutoMode      AutoMode
	DefaultAudio  AudioMode
}

// DefaultConfig returns the standard daemon configuration.
// State-related defaults (AutoMode, DefaultAudio) are derived from DefaultState()
// so env defaults and initial runtime state cannot drift apart.
func DefaultConfig() Config {
	state := DefaultState()

	//nolint:exhaustruct
	return Config{
		StateDir:      DefaultStateDir,
		PollInterval:  DefaultPollInterval,
		DebounceCount: DefaultDebounceCount,
		WebAddr:       DefaultWebAddr,
		AutoMode:      state.AutoMode,
		DefaultAudio:  state.Audio,
	}
}

// ConfigFromEnv returns a Config with defaults overridden by environment variables.
// Recognized variables: EMEET_PIXYD_STATE_DIR, EMEET_PIXYD_WEB_ADDR,
// EMEET_PIXYD_POLL_INTERVAL (Go duration), EMEET_PIXYD_DEBOUNCE_COUNT (int),
// EMEET_PIXYD_DEBUG (bool), EMEET_PIXYD_AUTO (off/full/tracking-only/privacy-only, or legacy true/1/false/0),
// EMEET_PIXYD_DEFAULT_AUDIO (nc/live/org).
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("EMEET_PIXYD_STATE_DIR"); v != "" {
		cfg.StateDir = v
	}

	if v := os.Getenv("EMEET_PIXYD_WEB_ADDR"); v != "" {
		cfg.WebAddr = v
	}

	if v := os.Getenv("EMEET_PIXYD_POLL_INTERVAL"); v != "" {
		d, parseErr := time.ParseDuration(v)
		if parseErr == nil {
			cfg.PollInterval = d
		} else {
			slog.Warn("invalid EMEET_PIXYD_POLL_INTERVAL, using default", "value", v, "default", cfg.PollInterval)
		}
	}

	if v := os.Getenv("EMEET_PIXYD_DEBOUNCE_COUNT"); v != "" {
		n, parseErr := strconv.Atoi(v)
		if parseErr == nil {
			cfg.DebounceCount = n
		} else {
			slog.Warn("invalid EMEET_PIXYD_DEBOUNCE_COUNT, using default", "value", v, "default", cfg.DebounceCount)
		}
	}

	if v := os.Getenv("EMEET_PIXYD_DEBUG"); v != "" {
		cfg.Debug = strings.EqualFold(v, "true") || v == "1"
	}

	if v := os.Getenv("EMEET_PIXYD_AUTO"); v != "" {
		m, parseErr := ParseAutoMode(v)
		if parseErr == nil {
			cfg.AutoMode = m
		} else {
			slog.Warn("invalid EMEET_PIXYD_AUTO, using default", "value", v, "default", cfg.AutoMode)
		}
	}

	if v := os.Getenv("EMEET_PIXYD_DEFAULT_AUDIO"); v != "" {
		m, parseErr := ParseAudioMode(v)
		if parseErr == nil {
			cfg.DefaultAudio = m
		} else {
			slog.Warn("invalid EMEET_PIXYD_DEFAULT_AUDIO, using default", "value", v, "default", cfg.DefaultAudio)
		}
	}

	return cfg
}

// Config validation sentinel errors.
var (
	// ErrStateDirEmpty is returned when Config.StateDir is empty.
	ErrStateDirEmpty = errors.New("state directory must not be empty")
	// ErrPollIntervalZero is returned when Config.PollInterval is not positive.
	ErrPollIntervalZero = errors.New("poll interval must be positive")
	// ErrDebounceCountZero is returned when Config.DebounceCount is not positive.
	ErrDebounceCountZero = errors.New("debounce count must be positive")
	// ErrWebAddrEmpty is returned when Config.WebAddr is empty.
	ErrWebAddrEmpty = errors.New("web address must not be empty")
	// ErrInvalidAutoMode is returned when Config.AutoMode is not a valid mode.
	ErrInvalidAutoMode = errors.New("invalid auto mode in config")
	// ErrInvalidDefaultAudio is returned when Config.DefaultAudio is not a valid mode.
	ErrInvalidDefaultAudio = errors.New("invalid default audio mode in config")
)

// Validate checks that all required config fields are set and sane.
func (c Config) Validate() error {
	if c.StateDir == "" {
		return ErrStateDirEmpty
	}

	if c.PollInterval <= 0 {
		return ErrPollIntervalZero
	}

	if c.DebounceCount <= 0 {
		return ErrDebounceCountZero
	}

	if c.WebAddr == "" {
		return ErrWebAddrEmpty
	}

	if !c.AutoMode.Valid() {
		return fmt.Errorf("config auto mode %q: %w", c.AutoMode, ErrInvalidAutoMode)
	}

	if !c.DefaultAudio.Valid() {
		return fmt.Errorf("config default audio %q: %w", c.DefaultAudio, ErrInvalidDefaultAudio)
	}

	return nil
}

// StateFile returns the path to the JSON state file within the state directory.
func (c Config) StateFile() string { return c.StateDir + "/state.json" }

// SocketPath returns the path to the Unix domain control socket within the state directory.
func (c Config) SocketPath() string { return c.StateDir + "/control.sock" }
