// Package pixy provides domain types, configuration, and IPC helpers for the
// EMEET PIXY webcam daemon (emeet-pixyd).
package pixy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Default paths, intervals, and permission bits for the daemon.
const (
	// DefaultStateDir is the runtime state directory for socket and state file.
	DefaultStateDir      = "/run/emeet-pixyd"
	// DefaultPollInterval is how often the auto-manager checks camera usage.
	DefaultPollInterval  = 2 * time.Second
	// DefaultDebounceCount is the number of consecutive polls before triggering a state change.
	DefaultDebounceCount = 3
	// DefaultWebAddr is the default listen address for the web UI.
	DefaultWebAddr       = "127.0.0.1:8090"

	// DefaultSocketTimeout is the connect timeout for the Unix control socket client.
	DefaultSocketTimeout = 2 * time.Second
	// DefaultWriteTimeout is the I/O timeout for the Unix control socket client.
	DefaultWriteTimeout  = 2 * time.Second
	// SocketBufSize is the read buffer size for the Unix control socket.
	SocketBufSize        = 256
	// ConnBufSize is the read buffer size for the Unix control socket client.
	ConnBufSize          = 4096

	// PermissionStateDir is the os.FileMode for the state directory.
	PermissionStateDir  = 0o750
	// PermissionStateFile is the os.FileMode for the state JSON file.
	PermissionStateFile = 0o600
	// PermissionSocket is the os.FileMode for the Unix control socket.
	PermissionSocket    = 0o600
)

var (
	// ErrInvalidAudioMode is returned when parsing an unknown audio mode string.
	ErrInvalidAudioMode      = errors.New("invalid audio mode")
	// ErrInvalidCameraState is returned when parsing an unknown camera state string.
	ErrInvalidCameraState    = errors.New("invalid camera state")
	// ErrHIDDeviceNotAvailable is returned when the HIDRAW device path is empty.
	ErrHIDDeviceNotAvailable = errors.New("PIXY HID device not available")
	// ErrPIXYNotConnected is returned when the V4L2 device path is empty.
	ErrPIXYNotConnected      = errors.New("PIXY not connected")
)

// CameraState represents the current operating mode of the PIXY camera.
type CameraState string

// Camera operating states.
const (
	// StateIdle means the camera is powered on but not actively tracking.
	StateIdle     CameraState = "idle"
	// StateTracking means the camera is actively tracking faces.
	StateTracking CameraState = "tracking"
	// StatePrivacy means the camera lens is physically blocked.
	StatePrivacy  CameraState = "privacy"
	// StateOffline means no PIXY device is detected.
	StateOffline  CameraState = "offline"
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
	AudioNC       AudioMode = "nc"
	// AudioLive is optimized for live / streaming audio.
	AudioLive     AudioMode = "live"
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
	case "org":
		return AudioOriginal, nil
	default:
		return "", fmt.Errorf("invalid audio mode: %q: %w", rawInput, ErrInvalidAudioMode)
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
	AutoMode bool        `json:"autoMode"`
}

// DefaultState returns the initial daemon state with privacy mode and auto-management enabled.
func DefaultState() State {
	return State{
		Camera:   StatePrivacy,
		Audio:    AudioNC,
		Gesture:  false,
		InCall:   false,
		AutoMode: true,
	}
}

// Config holds daemon configuration parameters.
type Config struct {
	StateDir      string
	PollInterval  time.Duration
	DebounceCount int
	WebAddr       string
}

// DefaultConfig returns the standard daemon configuration.
func DefaultConfig() Config {
	return Config{
		StateDir:      DefaultStateDir,
		PollInterval:  DefaultPollInterval,
		DebounceCount: DefaultDebounceCount,
		WebAddr:       DefaultWebAddr,
	}
}

// Config validation sentinel errors.
var (
	// ErrStateDirEmpty is returned when Config.StateDir is empty.
	ErrStateDirEmpty     = errors.New("state directory must not be empty")
	// ErrPollIntervalZero is returned when Config.PollInterval is not positive.
	ErrPollIntervalZero  = errors.New("poll interval must be positive")
	// ErrDebounceCountZero is returned when Config.DebounceCount is not positive.
	ErrDebounceCountZero = errors.New("debounce count must be positive")
	// ErrWebAddrEmpty is returned when Config.WebAddr is empty.
	ErrWebAddrEmpty      = errors.New("web address must not be empty")
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
		return fmt.Errorf("setDeadline: %w", err)
	}

	return nil
}

// SendCommand sends a command string over a Unix socket and returns the response.
func SendCommand(ctx context.Context, socketPath, cmd string) (string, error) {
	dialer := net.Dialer{Timeout: DefaultSocketTimeout}

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("sendCommand dial: %w", err)
	}

	defer func() { _ = conn.Close() }()

	deadlineErr := SetDeadline(conn, DefaultWriteTimeout)
	if deadlineErr != nil {
		return "", deadlineErr
	}

	_, writeErr := conn.Write([]byte(cmd))
	if writeErr != nil {
		return "", fmt.Errorf("sendCommand write: %w", writeErr)
	}

	buf := make([]byte, ConnBufSize)

	n, readErr := conn.Read(buf)
	if readErr != nil {
		return "", fmt.Errorf("sendCommand read: %w", readErr)
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
