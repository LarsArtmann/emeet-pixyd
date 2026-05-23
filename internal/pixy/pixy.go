// Package pixy provides domain types, configuration, and IPC helpers for the
// EMEET PIXY webcam daemon (emeet-pixyd).
package pixy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default paths, intervals, and permission bits for the daemon.
const (
	// DefaultStateDir is the runtime state directory for socket and state file.
	DefaultStateDir = "/run/emeet-pixyd"
	// DefaultPollInterval is how often the auto-manager checks camera usage.
	DefaultPollInterval = 2 * time.Second
	// DefaultDebounceCount is the number of consecutive polls before triggering a state change.
	DefaultDebounceCount = 3
	// DefaultWebAddr is the default listen address for the web UI.
	DefaultWebAddr = "127.0.0.1:8090"

	// DefaultSocketTimeout is the connect timeout for the Unix control socket client.
	DefaultSocketTimeout = 2 * time.Second
	// DefaultWriteTimeout is the I/O timeout for the Unix control socket client.
	DefaultWriteTimeout = 2 * time.Second
	// SocketBufSize is the read buffer size for the Unix control socket.
	SocketBufSize = 256
	// ConnBufSize is the read buffer size for the Unix control socket client.
	ConnBufSize = 4096

	// PermissionStateDir is the os.FileMode for the state directory.
	PermissionStateDir = 0o750
	// PermissionStateFile is the os.FileMode for the state JSON file.
	PermissionStateFile = 0o600
	// PermissionSocket is the os.FileMode for the Unix control socket.
	PermissionSocket = 0o600
)

var (
	// ErrInvalidAudioMode is returned when parsing an unknown audio mode string.
	ErrInvalidAudioMode = errors.New("invalid audio mode")
	// ErrInvalidCameraState is returned when parsing an unknown camera state string.
	ErrInvalidCameraState = errors.New("invalid camera state")
	// ErrHIDDeviceNotAvailable is returned when the HIDRAW device path is empty.
	ErrHIDDeviceNotAvailable = errors.New("PIXY HID device not available")
	// ErrPIXYNotConnected is returned when the V4L2 device path is empty.
	ErrPIXYNotConnected = errors.New("PIXY not connected")
)

// CameraState represents the current operating mode of the PIXY camera.
type CameraState string

// Camera operating states.
const (
	// StateIdle means the camera is powered on but not actively tracking.
	StateIdle CameraState = "idle"
	// StateTracking means the camera is actively tracking faces.
	StateTracking CameraState = "tracking"
	// StatePrivacy means the camera lens is physically blocked.
	StatePrivacy CameraState = "privacy"
	// StateOffline means no PIXY device is detected.
	StateOffline CameraState = "offline"
)

func (s CameraState) String() string { return string(s) }

// Valid reports whether the camera state is one of the known values.
func (s CameraState) Valid() bool {
	switch s {
	case StateIdle, StateTracking, StatePrivacy, StateOffline:
		return true
	default:
		return false
	}
}

// AudioMode represents the noise cancellation mode of the PIXY camera microphone.
type AudioMode string

// Audio noise-cancellation modes.
const (
	// AudioNC enables noise cancellation (default for calls).
	AudioNC AudioMode = "nc"
	// AudioLive is optimized for live / streaming audio.
	AudioLive AudioMode = "live"
	// AudioOriginal passes through raw microphone audio without processing.
	AudioOriginal AudioMode = "original"
)

func (m AudioMode) String() string { return string(m) }

// Valid reports whether the audio mode is one of the known values.
func (m AudioMode) Valid() bool {
	switch m {
	case AudioNC, AudioLive, AudioOriginal:
		return true
	default:
		return false
	}
}

// Next returns the audio mode that follows m in the NC → Live → Original → NC cycle.
func (m AudioMode) Next() AudioMode {
	switch m {
	case AudioNC:
		return AudioLive
	case AudioLive:
		return AudioOriginal
	case AudioOriginal:
		return AudioNC
	default:
		return AudioNC
	}
}

// ParseAudioMode maps a CLI shorthand ("nc", "live", "org") to an AudioMode.
func ParseAudioMode(rawInput string) (AudioMode, error) {
	switch rawInput {
	case "nc":
		return AudioNC, nil
	case "live":
		return AudioLive, nil
	case "org", string(AudioOriginal):
		return AudioOriginal, nil
	default:
		return "", fmt.Errorf("invalid audio mode: %q: %w", rawInput, ErrInvalidAudioMode)
	}
}

// AutoMode represents the automatic camera management strategy.
type AutoMode string

// Auto-management modes.
const (
	// AutoOff disables all automatic management.
	AutoOff AutoMode = "off"
	// AutoFull enables tracking + noise cancellation on call start, privacy on call end.
	AutoFull AutoMode = "full"
	// AutoTrackingOnly enables face tracking on call start, privacy on call end.
	AutoTrackingOnly AutoMode = "tracking-only"
	// AutoPrivacyOnly enables privacy mode on call end, but does not activate tracking on call start.
	AutoPrivacyOnly AutoMode = "privacy-only"
)

func (m AutoMode) String() string { return string(m) }

// Valid reports whether the auto mode is one of the known values.
func (m AutoMode) Valid() bool {
	switch m {
	case AutoOff, AutoFull, AutoTrackingOnly, AutoPrivacyOnly:
		return true
	default:
		return false
	}
}

// IsOff reports whether auto-management is completely disabled.
func (m AutoMode) IsOff() bool { return m == AutoOff }

// Toggle returns AutoFull if auto is off, or AutoOff if auto is on.
func (m AutoMode) Toggle() AutoMode {
	if m.IsOff() {
		return AutoFull
	}
	return AutoOff
}

// ActivatesTracking reports whether this mode activates face tracking on call start.
func (m AutoMode) ActivatesTracking() bool {
	return m == AutoFull || m == AutoTrackingOnly
}

// ActivatesAudio reports whether this mode activates noise cancellation on call start.
func (m AutoMode) ActivatesAudio() bool {
	return m == AutoFull
}

// ActivatesPrivacy reports whether this mode switches to privacy on call end.
func (m AutoMode) ActivatesPrivacy() bool {
	return m == AutoFull || m == AutoTrackingOnly || m == AutoPrivacyOnly
}

// SwitchesSource reports whether this mode switches PipeWire source on call start.
func (m AutoMode) SwitchesSource() bool {
	return m == AutoFull
}

// ParseAutoMode maps a string to an AutoMode. Accepts both the enum values
// ("off", "full", "tracking-only", "privacy-only") and legacy booleans
// ("true"/"1" → full, "false"/"0" → off).
func ParseAutoMode(rawInput string) (AutoMode, error) {
	switch strings.ToLower(strings.TrimSpace(rawInput)) {
	case "off":
		return AutoOff, nil
	case "full":
		return AutoFull, nil
	case "tracking-only":
		return AutoTrackingOnly, nil
	case "privacy-only":
		return AutoPrivacyOnly, nil
	case "true", "1":
		return AutoFull, nil
	case "false", "0":
		return AutoOff, nil
	default:
		return AutoOff, fmt.Errorf(
			"invalid auto mode: %q (valid: off, full, tracking-only, privacy-only)",
			rawInput,
		)
	}
}

// ParseCameraState maps a string to a CameraState.
func ParseCameraState(rawInput string) (CameraState, error) {
	switch rawInput {
	case string(StateIdle):
		return StateIdle, nil
	case string(StateTracking):
		return StateTracking, nil
	case string(StatePrivacy):
		return StatePrivacy, nil
	case string(StateOffline):
		return StateOffline, nil
	default:
		return "", fmt.Errorf("invalid camera state: %q: %w", rawInput, ErrInvalidCameraState)
	}
}

// State holds the current runtime state of the PIXY daemon.
type State struct {
	Camera   CameraState `json:"camera"`
	Audio    AudioMode   `json:"audio"`
	Gesture  bool        `json:"gesture"`
	InCall   bool        `json:"inCall"`
	AutoMode AutoMode    `json:"autoMode"`
}

// DefaultState returns the initial daemon state with privacy mode and auto-management enabled.
func DefaultState() State {
	return State{
		Camera:   StatePrivacy,
		Audio:    AudioNC,
		Gesture:  false,
		InCall:   false,
		AutoMode: AutoFull,
	}
}

// Valid reports whether all enum fields contain recognized values.
func (s State) Valid() bool {
	return s.Camera.Valid() && s.Audio.Valid() && s.AutoMode.Valid()
}

// PTZValues holds the current pan/tilt/zoom position of the camera.
type PTZValues struct {
	Pan  int
	Tilt int
	Zoom int
}

// PTZ axis limits in user-facing units (degrees for pan/tilt, multiplier for zoom).
const (
	PanMin  = -170
	PanMax  = 170
	TiltMin = -30
	TiltMax = 30
	ZoomMin = 100
	ZoomMax = 400
	// ZoomDefault is the zoom value when the camera is centered/reset.
	ZoomDefault = 100
)

// Config holds daemon configuration parameters.
// Fields marked with env tags are read from environment variables by ConfigFromEnv().
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
func DefaultConfig() Config {
	//nolint:exhaustruct
	return Config{
		StateDir:      DefaultStateDir,
		PollInterval:  DefaultPollInterval,
		DebounceCount: DefaultDebounceCount,
		WebAddr:       DefaultWebAddr,
		AutoMode:      AutoFull,
		DefaultAudio:  AudioNC,
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
		if d, err := time.ParseDuration(v); err == nil {
			cfg.PollInterval = d
		}
	}

	if v := os.Getenv("EMEET_PIXYD_DEBOUNCE_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DebounceCount = n
		}
	}

	if v := os.Getenv("EMEET_PIXYD_DEBUG"); v != "" {
		cfg.Debug = strings.EqualFold(v, "true") || v == "1"
	}

	if v := os.Getenv("EMEET_PIXYD_AUTO"); v != "" {
		if m, err := ParseAutoMode(v); err == nil {
			cfg.AutoMode = m
		}
	}

	if v := os.Getenv("EMEET_PIXYD_DEFAULT_AUDIO"); v != "" {
		if m, err := ParseAudioMode(v); err == nil {
			cfg.DefaultAudio = m
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

// SetDeadline sets a read/write deadline on the connection relative to now.
func SetDeadline(conn net.Conn, timeout time.Duration) error {
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return fmt.Errorf("setDeadline (timeout=%v): %w", timeout, err)
	}

	return nil
}

// SendCommand sends a command string over a Unix socket and returns the response.
func SendCommand(ctx context.Context, socketPath, cmd string) (string, error) {
	//nolint:exhaustruct
	dialer := net.Dialer{Timeout: DefaultSocketTimeout}

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("sendCommand dial %s: %w", socketPath, err)
	}

	defer func() { _ = conn.Close() }()

	deadlineErr := SetDeadline(conn, DefaultWriteTimeout)
	if deadlineErr != nil {
		return "", fmt.Errorf("sendCommand %s deadline: %w", socketPath, deadlineErr)
	}

	_, writeErr := conn.Write([]byte(cmd))
	if writeErr != nil {
		return "", fmt.Errorf("sendCommand %s write: %w", socketPath, writeErr)
	}

	buf := make([]byte, ConnBufSize)

	n, readErr := conn.Read(buf)
	if readErr != nil {
		return "", fmt.Errorf("sendCommand %s read: %w", socketPath, readErr)
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
