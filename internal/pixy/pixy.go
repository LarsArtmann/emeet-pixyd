package pixy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	DefaultStateDir      = "/run/emeet-pixyd"
	DefaultPollInterval  = 2 * time.Second
	DefaultDebounceCount = 3
	DefaultWebAddr       = "127.0.0.1:8090"

	DefaultSocketTimeout = 2 * time.Second
	DefaultWriteTimeout  = 2 * time.Second
	SocketBufSize        = 256
	ConnBufSize          = 4096

	PermissionStateDir  = 0o750
	PermissionStateFile = 0o600
	PermissionSocket    = 0o600
)

var (
	ErrInvalidAudioMode      = errors.New("invalid audio mode")
	ErrInvalidCameraState    = errors.New("invalid camera state")
	ErrHIDDeviceNotAvailable = errors.New("PIXY HID device not available")
	ErrPIXYNotConnected      = errors.New("PIXY not connected")
)

// CameraState represents the current operating mode of the PIXY camera.
type CameraState string

const (
	StateIdle     CameraState = "idle"
	StateTracking CameraState = "tracking"
	StatePrivacy  CameraState = "privacy"
	StateOffline  CameraState = "offline"
)

func (s CameraState) String() string { return string(s) }

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

const (
	AudioNC       AudioMode = "nc"
	AudioLive     AudioMode = "live"
	AudioOriginal AudioMode = "original"
)

func (m AudioMode) String() string { return string(m) }

func (m AudioMode) Valid() bool {
	switch m {
	case AudioNC, AudioLive, AudioOriginal:
		return true
	default:
		return false
	}
}

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

var (
	ErrStateDirEmpty     = errors.New("state directory must not be empty")
	ErrPollIntervalZero  = errors.New("poll interval must be positive")
	ErrDebounceCountZero = errors.New("debounce count must be positive")
	ErrWebAddrEmpty      = errors.New("web address must not be empty")
)

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
