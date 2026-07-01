// Package pixy provides domain types, configuration, and IPC helpers for the
// EMEET PIXY webcam daemon (emeet-pixyd).
package pixy

import (
	"errors"
	"fmt"
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
// Input is case-insensitive.
func ParseAudioMode(rawInput string) (AudioMode, error) {
	switch strings.ToLower(strings.TrimSpace(rawInput)) {
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
			"invalid auto mode: %q (valid: off, full, tracking-only, privacy-only): %w",
			rawInput,
			ErrInvalidAutoMode,
		)
	}
}

// ParseCameraState maps a string to a CameraState.
// Input is case-insensitive.
func ParseCameraState(rawInput string) (CameraState, error) {
	switch strings.ToLower(strings.TrimSpace(rawInput)) {
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
	Camera   CameraState          `json:"camera"`
	Audio    AudioMode            `json:"audio"`
	Gesture  bool                 `json:"gesture"`
	InCall   bool                 `json:"inCall"`
	AutoMode AutoMode             `json:"autoMode"`
	Presets  map[string]PTZValues `json:"presets,omitempty"`
}

// DefaultState returns the initial daemon state with privacy mode and auto-management enabled.
func DefaultState() State {
	return State{
		Camera:   StatePrivacy,
		Audio:    AudioNC,
		Gesture:  false,
		InCall:   false,
		AutoMode: AutoFull,
		Presets:  make(map[string]PTZValues),
	}
}

// Valid reports whether all enum fields contain recognized values.
func (s State) Valid() bool {
	return s.Camera.Valid() && s.Audio.Valid() && s.AutoMode.Valid()
}

// PTZValues holds the current pan/tilt/zoom position of the camera.
type PTZValues struct {
	Pan  int `json:"pan"`
	Tilt int `json:"tilt"`
	Zoom int `json:"zoom"`
}

func (p PTZValues) Clamp() PTZValues {
	return PTZValues{
		Pan:  PanRange.Clamp(p.Pan),
		Tilt: TiltRange.Clamp(p.Tilt),
		Zoom: ZoomRange.Clamp(p.Zoom),
	}
}

// Get returns the PTZ value for the given axis and true if the axis is
// recognized, or 0 and false if the axis is unknown.
func (p PTZValues) Get(axis Axis) (int, bool) {
	switch axis {
	case AxisPan:
		return p.Pan, true
	case AxisTilt:
		return p.Tilt, true
	case AxisZoom:
		return p.Zoom, true
	default:
		return 0, false
	}
}

// Set returns a copy with the given axis set to val.
func (p PTZValues) Set(axis Axis, val int) PTZValues {
	switch axis {
	case AxisPan:
		p.Pan = val
	case AxisTilt:
		p.Tilt = val
	case AxisZoom:
		p.Zoom = val
	}

	return p
}

// Range pairs a minimum and maximum limit for a PTZ axis.
type Range struct {
	Min int
	Max int
}

// Clamp restricts v to the inclusive [Min, Max] range.
func (r Range) Clamp(v int) int {
	return max(r.Min, min(r.Max, v))
}

// PTZ axis limits in user-facing units (degrees for pan/tilt, multiplier for zoom).
// Hardware-verified against the EMEET PIXY V4L2 capabilities.
//
//nolint:gochecknoglobals,mnd // hardware constants, never mutated at runtime
var (
	PanRange  = Range{Min: -150, Max: 150}
	TiltRange = Range{Min: -90, Max: 90}
	ZoomRange = Range{Min: 100, Max: 150}
)

const (
	// ZoomDefault is the zoom value when the camera is centered/reset.
	ZoomDefault = 100
)

// Axis is a PTZ axis name: pan, tilt, or zoom.
// The branded type prevents accidental substitution of arbitrary strings
// into axis-keyed maps and functions.
type Axis string

// PTZ axis names used in CLI commands and HTTP routes.
const (
	AxisPan  Axis = "pan"
	AxisTilt Axis = "tilt"
	AxisZoom Axis = "zoom"
)
