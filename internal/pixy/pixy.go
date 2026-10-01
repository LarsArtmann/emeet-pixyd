// Package pixy provides domain types, configuration, and IPC helpers for the
// EMEET PIXY webcam daemon (emeet-pixyd).
package pixy

import (
	"slices"
	"strings"
	"time"
	"unicode"

	errorfamily "github.com/larsartmann/go-error-family"
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
	ErrInvalidAudioMode error = errorfamily.NewRejection("audio.mode_invalid", "invalid audio mode")
	// ErrInvalidCameraState is returned when parsing an unknown camera state string.
	ErrInvalidCameraState error = errorfamily.NewRejection("camera.state_invalid", "invalid camera state")
	// ErrInvalidPresetName is returned when a preset name fails validation.
	ErrInvalidPresetName error = errorfamily.NewRejection("preset.name_invalid", "invalid preset name")
	// ErrHIDDeviceNotAvailable is returned when the HIDRAW device path is empty.
	ErrHIDDeviceNotAvailable error = errorfamily.NewInfrastructure(
		"hid.device_unavailable",
		"PIXY HID device not available",
	)
	// ErrPIXYNotConnected is returned when the V4L2 device path is empty.
	ErrPIXYNotConnected error = errorfamily.NewInfrastructure("pixy.not_connected", "PIXY not connected")
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

// ValidDesired reports whether s is a mode the user can actually choose.
// StateOffline is excluded: it is a connectivity observation projected at
// read time, never a persisted intent (see State.Camera).
func (s CameraState) ValidDesired() bool {
	switch s {
	case StateIdle, StateTracking, StatePrivacy:
		return true
	case StateOffline:
		return false
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
		return "", Wrapf(ErrInvalidAudioMode, "audio.parse_mode", "invalid audio mode: %q", rawInput)
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
		return AutoOff, Wrapf(
			ErrInvalidAutoMode,
			"auto.parse_mode",
			"invalid auto mode: %q (valid: off, full, tracking-only, privacy-only)",
			rawInput,
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
		return "", Wrapf(ErrInvalidCameraState, "camera.parse_state", "invalid camera state: %q", rawInput)
	}
}

// CurrentSchemaVersion is the state file format version. Increment when
// the JSON schema changes. Old state files with a lower version are
// logged as stale but still loaded (best-effort backward compatibility).
//
// v2 (2026-10-01): State.Camera is defined as *desired* mode only — the
// connectivity value StateOffline is no longer persisted (it was a
// split-brain with user intent). v1 files carrying "offline" are normalized
// to privacy on load.
const CurrentSchemaVersion = 2

// State holds the current runtime state of the PIXY daemon.
type State struct {
	SchemaVersion int `json:"v"`

	// Camera is the user's desired operating mode — one of idle, tracking,
	// or privacy. It is NEVER "offline": connectivity is observed at runtime
	// (a probe/device snapshot) and projected onto the display, so unplugging
	// the camera can never overwrite (or persist) the user's choice.
	Camera  CameraState `json:"camera"`
	Audio   AudioMode   `json:"audio"`
	Gesture bool        `json:"gesture"`

	// InCall is a runtime observation (is a call in progress?), not intent.
	// It is persisted for crash visibility but reset on daemon start: the
	// daemon cannot be in a call before it begins observing /proc.
	InCall   bool      `json:"inCall"`
	AutoMode AutoMode  `json:"autoMode"`
	Presets  PresetMap `json:"presets,omitempty"`

	// TrackMode is the persisted tracking variant (TODO #140): the canonical
	// TargetTrackMode string ("none"/"face"/"halfbody"/"fullbody"). Empty
	// means "never set" and reads back as the default (face).
	TrackMode string `json:"trackMode,omitempty"`

	// Speeds are the persisted per-axis motor speeds (TODO #138). Zero on an
	// axis means "no preference": the daemon leaves the firmware default in
	// effect and does not re-send it before moves.
	Speeds SpeedValues `json:"speeds,omitzero"`
}

// DefaultState returns the initial daemon state with privacy mode and auto-management enabled.
func DefaultState() State {
	return State{
		SchemaVersion: CurrentSchemaVersion,
		Camera:        StatePrivacy,
		Audio:         AudioNC,
		Gesture:       false,
		InCall:        false,
		AutoMode:      AutoFull,
		Presets:       NewPresetMap(),
		TrackMode:     "",            // never set — reads back as the default (face)
		Speeds:        SpeedValues{}, // no speed preference
	}
}

// Valid reports whether all enum fields contain recognized values.
func (s State) Valid() bool {
	if s.TrackMode != "" {
		if _, ok := ParseTargetTrackMode(s.TrackMode); !ok {
			return false
		}
	}

	return s.Camera.Valid() && s.Audio.Valid() && s.AutoMode.Valid()
}

// EffectiveTrackMode returns the persisted tracking variant, or the default
// (face) when none was ever set.
func (s State) EffectiveTrackMode() TargetTrackMode {
	if mode, ok := ParseTargetTrackMode(s.TrackMode); ok {
		return mode
	}

	return TrackFace
}

// SpeedValues holds the per-axis motor speed preference (TODO #138).
// Values are transmitted verbatim to the official SetMotorSpeed command;
// the physical unit is assumed degrees/second until hardware verification.
// Zero means "no preference" (firmware default stays in effect).
type SpeedValues struct {
	Pan  float32 `json:"pan,omitzero"`
	Tilt float32 `json:"tilt,omitzero"`
	Zoom float32 `json:"zoom,omitzero"`
}

// Get returns the speed for the given axis and true if the axis is
// recognized, or 0 and false if the axis is unknown.
func (s SpeedValues) Get(axis Axis) (float32, bool) {
	return axisGet(axis, s.Pan, s.Tilt, s.Zoom)
}

// Set returns a copy with the given axis set to speed.
func (s SpeedValues) Set(axis Axis, speed float32) SpeedValues {
	axisSet(axis, speed, &s.Pan, &s.Tilt, &s.Zoom)

	return s
}

// IsZero reports whether no axis carries a speed preference.
func (s SpeedValues) IsZero() bool { return s.Pan == 0 && s.Tilt == 0 && s.Zoom == 0 }

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
	return axisGet(axis, p.Pan, p.Tilt, p.Zoom)
}

// Set returns a copy with the given axis set to val.
func (p PTZValues) Set(axis Axis, val int) PTZValues {
	axisSet(axis, val, &p.Pan, &p.Tilt, &p.Zoom)

	return p
}

// axisGet returns the value held for axis among the pan/tilt/zoom fields,
// or the zero value and false when the axis is unrecognized. Shared by
// SpeedValues and PTZValues so the axis mapping has one definition.
func axisGet[V ~int | ~float32](axis Axis, pan, tilt, zoom V) (V, bool) {
	switch axis {
	case AxisPan:
		return pan, true
	case AxisTilt:
		return tilt, true
	case AxisZoom:
		return zoom, true
	default:
		return 0, false
	}
}

// axisSet writes value into the pan/tilt/zoom field matching axis;
// unrecognized axes are ignored.
func axisSet[V ~int | ~float32](axis Axis, value V, pan, tilt, zoom *V) {
	switch axis {
	case AxisPan:
		*pan = value
	case AxisTilt:
		*tilt = value
	case AxisZoom:
		*zoom = value
	}
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

// Preset limits.
const (
	// MaxPresets is the maximum number of named PTZ presets that can be stored.
	MaxPresets = 16
	// MaxPresetNameLength is the maximum length of a preset name in characters.
	MaxPresetNameLength = 32
)

// ValidatePresetName reports whether name is an acceptable preset label.
// A valid name is non-empty (after trimming), at most MaxPresetNameLength
// runes, contains no path separators, and contains no control characters.
func ValidatePresetName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Wrap(ErrInvalidPresetName, "preset.name_empty", "preset name is empty")
	}

	if len([]rune(trimmed)) > MaxPresetNameLength {
		return Wrapf(
			ErrInvalidPresetName,
			"preset.name_too_long",
			"preset name too long (%d > %d)",
			len([]rune(trimmed)),
			MaxPresetNameLength,
		)
	}

	for _, r := range trimmed {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return Wrapf(
				ErrInvalidPresetName,
				"preset.name_char",
				"preset name %q contains an illegal character",
				trimmed,
			)
		}
	}

	return nil
}

// PresetMap is a named collection of saved PTZ positions.
// It serializes as a plain JSON object (map), preserving the on-disk format.
type PresetMap map[string]PTZValues

// NewPresetMap returns an empty, initialized PresetMap.
func NewPresetMap() PresetMap { return make(PresetMap) }

// SortedNames returns the preset names sorted alphabetically.
func (p PresetMap) SortedNames() []string {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// Get returns the PTZ values for name, or false if not found.
func (p PresetMap) Get(name string) (PTZValues, bool) {
	v, ok := p[name]

	return v, ok
}

// IsFull reports whether the preset collection has reached MaxPresets.
func (p PresetMap) IsFull() bool { return len(p) >= MaxPresets }
