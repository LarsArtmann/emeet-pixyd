package pixy

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
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

	// UpdateCheck enables the slow upstream release check (GitHub releases,
	// default on). When a newer release exists, every status surface shows a
	// small notice (web panel footer, /api/status, CLI version, Waybar
	// tooltip). Disable with EMEET_PIXYD_UPDATE_CHECK=off to keep the daemon
	// from reaching out to api.github.com at all.
	UpdateCheck bool

	// ExtraProductIDs are additional USB product IDs (hex, from	// EMEET_PIXYD_EXTRA_PRODUCT_IDS) treated as PIXY-family devices. They
	// let users onboard PIXY variants the registry does not know yet.
	ExtraProductIDs []USBProductID
}

// DefaultConfig returns the standard daemon configuration.
// State-related defaults (AutoMode, DefaultAudio) are derived from DefaultState()
// so env defaults and initial runtime state cannot drift apart.
func DefaultConfig() Config {
	state := DefaultState()

	return Config{
		StateDir:      DefaultStateDir,
		PollInterval:  DefaultPollInterval,
		DebounceCount: DefaultDebounceCount,
		WebAddr:       DefaultWebAddr,
		AutoMode:      state.AutoMode,
		DefaultAudio:  state.Audio,
		UpdateCheck:   true,
	}
}

// ConfigFromEnv returns a Config with defaults overridden by environment variables.
// Recognized variables: EMEET_PIXYD_STATE_DIR, EMEET_PIXYD_WEB_ADDR,
// EMEET_PIXYD_POLL_INTERVAL (Go duration), EMEET_PIXYD_DEBOUNCE_COUNT (int),
// EMEET_PIXYD_DEBUG (bool), EMEET_PIXYD_AUTO (off/full/tracking-only/privacy-only, or legacy true/1/false/0),
// EMEET_PIXYD_DEFAULT_AUDIO (nc/live/org),
// EMEET_PIXYD_EXTRA_PRODUCT_IDS (comma-separated hex USB product IDs),
// EMEET_PIXYD_UPDATE_CHECK (on/true/1 or off/false/0; default on).
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

	if v := os.Getenv("EMEET_PIXYD_EXTRA_PRODUCT_IDS"); v != "" {
		cfg.ExtraProductIDs = ParseExtraProductIDs(v)
	}

	if v := os.Getenv("EMEET_PIXYD_UPDATE_CHECK"); v != "" {
		enabled, parseErr := ParseBool(v)
		if parseErr == nil {
			cfg.UpdateCheck = enabled
		} else {
			slog.Warn("invalid EMEET_PIXYD_UPDATE_CHECK, using default", "value", v, "default", cfg.UpdateCheck)
		}
	}

	return cfg
}

// ParseBool parses the boolean env syntax shared by flag-like settings:
// on/true/1/enabled and off/false/0/disabled, case-insensitive. Numbers keep
// DEBUG's legacy "1" form working in both directions.
func ParseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "enabled":
		return true, nil
	case "off", "false", "0", "disabled":
		return false, nil
	default:
		return false, errorfamily.NewRejection("config.parse_bool",
			fmt.Sprintf("cannot parse %q as a boolean (on/off, true/false, 1/0)", value))
	}
}

// ParseUSBProductID parses a single hex USB product ID ("0x0119", "01ab",
// "0X01AB" — case-insensitive, optional prefix). Values that do not parse
// or are non-positive are rejected; ParseExtraProductIDs decides the
// skip-vs-abort policy per entry.
func ParseUSBProductID(raw string) (USBProductID, error) {
	id, parseErr := strconv.ParseInt(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "0x"), 16, 64)
	if parseErr != nil || id <= 0 {
		return 0, Wrapf(ErrInvalidProductID, "config.parse_product_id", "invalid USB product ID %q", raw)
	}

	return USBProductID(id), nil
}

// ParseExtraProductIDs parses a comma-separated list of hex USB product IDs
// (e.g. "0x0119,01ab"). Empty, unparseable, and non-positive entries are
// skipped with a warning so one bad value cannot disable the others.
func ParseExtraProductIDs(value string) []USBProductID {
	var ids []USBProductID

	for raw := range strings.SplitSeq(value, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		id, parseErr := ParseUSBProductID(raw)
		if parseErr != nil {
			slog.Warn("invalid EMEET_PIXYD_EXTRA_PRODUCT_IDS entry, skipping", "value", raw)

			continue
		}

		ids = append(ids, id)
	}

	return ids
}

// Config validation sentinel errors.
var (
	// ErrStateDirEmpty is returned when Config.StateDir is empty.
	ErrStateDirEmpty error = errorfamily.NewRejection("config.state_dir_empty", "state directory must not be empty")
	// ErrPollIntervalZero is returned when Config.PollInterval is not positive.
	ErrPollIntervalZero error = errorfamily.NewRejection("config.poll_interval", "poll interval must be positive")
	// ErrDebounceCountZero is returned when Config.DebounceCount is not positive.
	ErrDebounceCountZero error = errorfamily.NewRejection("config.debounce_count", "debounce count must be positive")
	// ErrWebAddrEmpty is returned when Config.WebAddr is empty.
	ErrWebAddrEmpty error = errorfamily.NewRejection("config.web_addr", "web address must not be empty")
	// ErrInvalidAutoMode is returned when Config.AutoMode is not a valid mode.
	ErrInvalidAutoMode error = errorfamily.NewRejection("config.auto_mode", "invalid auto mode in config")
	// ErrInvalidDefaultAudio is returned when Config.DefaultAudio is not a valid mode.
	ErrInvalidDefaultAudio error = errorfamily.NewRejection(
		"config.default_audio",
		"invalid default audio mode in config",
	)
	// ErrInvalidProductID is returned when a USB product ID fails to parse.
	ErrInvalidProductID error = errorfamily.NewRejection("config.product_id_invalid", "invalid USB product ID")
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
		return Wrapf(ErrInvalidAutoMode, "config.validate_auto", "config auto mode %q", c.AutoMode)
	}

	if !c.DefaultAudio.Valid() {
		return Wrapf(ErrInvalidDefaultAudio, "config.validate_audio", "config default audio %q", c.DefaultAudio)
	}

	return nil
}

// StateFile returns the path to the JSON state file within the state directory.
func (c Config) StateFile() string { return c.StateDir + "/state.json" }

// SocketPath returns the path to the Unix domain control socket within the state directory.
func (c Config) SocketPath() string { return c.StateDir + "/control.sock" }
