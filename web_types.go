//go:build linux

package main

import "github.com/LarsArtmann/emeet-pixyd/internal/pixy"

type webStatus struct {
	Camera     pixy.CameraState
	Audio      pixy.AudioMode
	Gesture    bool
	Pan        int
	Tilt       int
	Zoom       int
	InCall     bool
	Auto       pixy.AutoMode
	Online     bool
	Device     string
	Error      string
	LastSynced string
	Toast      string
	ToastType  string
	Version    string
}
