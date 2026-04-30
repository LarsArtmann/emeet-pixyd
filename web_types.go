//go:build linux

package main

type webStatus struct {
	Camera     string
	Audio      string
	Gesture    bool
	Pan        int
	Tilt       int
	Zoom       int
	InCall     bool
	Auto       bool
	Online     bool
	Device     string
	Error      string
	LastSynced string
	Toast      string
	ToastType  string
}
