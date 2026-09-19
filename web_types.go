//go:build linux

package main

import (
	"strconv"

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

	Camera      pixy.CameraState
	Audio       pixy.AudioMode
	Gesture     bool
	InCall      bool
	Auto        pixy.AutoMode
	Online      bool
	Device      string
	Model       string
	Error       string
	LastSynced  string
	Toast       string
	ToastType   toastType
	Version     string
	PresetNames []string
	Battery     string // "" when the device does not answer battery queries
	TrackMode   string // active tracking variant (face/halfbody/fullbody)
	Speeds      pixy.SpeedValues
}

// toastType is a branded type for toast notification kinds (success, info, error).
type toastType string

const (
	toastTypeSuccess toastType = "success"
	toastTypeInfo    toastType = "info"
	toastTypeError   toastType = "error"
)
