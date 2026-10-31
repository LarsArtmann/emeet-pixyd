//go:build linux

package main

import (
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// formatSpeed renders a persisted motor speed for HTML attributes: shortest
// exact form, no trailing zeros (templ renders it into data-signals and
// value attributes).
func formatSpeed(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

type webStatus struct {
	pixy.PTZValues

	Camera       pixy.CameraState
	Audio        pixy.AudioMode
	Gesture      bool
	InCall       bool
	Auto         pixy.AutoMode
	Online       bool
	Controllable bool
	Device       string
	Model        string
	Error        string
	LastSynced   string
	Toast        string
	ToastType    toastType
	Version      string
	PresetNames  []string
	Battery      string // "" when the device does not answer battery queries
	TrackMode    string // active tracking variant (none/face/halfbody/fullbody)
	Speeds       pixy.SpeedValues
	// UnsupportedHint explains a recognized-but-unsupported EMEET device
	// ("" = nothing to show). Long-form text; space-constrained surfaces
	// render unsupportedHintShort of it.
	UnsupportedHint string

	Degraded        bool     // daemon-level degradation (dead listener, accept ceiling)
	DegradedReasons []string // stable reason keys (see degraded.go)
}

// toastType is a branded type for toast notification kinds (success, info, error).
type toastType string

const (
	toastTypeSuccess toastType = "success"
	toastTypeInfo    toastType = "info"
	toastTypeError   toastType = "error"
)

// unsupportedHintShort reduces the long-form CLI/log hint to a one-line
// label for space-constrained surfaces (web panel, Waybar tooltip): the
// clause before the first colon carries device + verdict ("EMEET C960
// (USB product ID 0x003f) is recognized but not controllable"); the full
// why-and-how text stays available in the panel tooltip and the CLI/log.
func unsupportedHintShort(hint string) string {
	if short, _, found := strings.Cut(hint, ":"); found {
		return short
	}

	return hint
}
