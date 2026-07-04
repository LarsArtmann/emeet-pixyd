//go:build linux

package main

import "github.com/LarsArtmann/emeet-pixyd/internal/pixy"

type webStatus struct {
	pixy.PTZValues

	Camera      pixy.CameraState
	Audio       pixy.AudioMode
	Gesture     bool
	InCall      bool
	Auto        pixy.AutoMode
	Online      bool
	Device      string
	Error       string
	LastSynced  string
	Toast       string
	ToastType   toastType
	Version     string
	PresetNames []string
}

// toastType is a branded type for toast notification kinds (success, info, error).
type toastType string

const (
	toastTypeSuccess toastType = "success"
	toastTypeInfo    toastType = "info"
	toastTypeError   toastType = "error"
)
