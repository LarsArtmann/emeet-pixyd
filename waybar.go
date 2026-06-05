//go:build linux

package main

import (
	"encoding/json"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

type waybarJSON struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

func (d *Daemon) waybarOutput() string {
	d.mu.RLock()
	camera := d.state.Camera
	audio := d.state.Audio
	inCall := d.state.InCall
	autoMode := d.state.AutoMode
	d.mu.RUnlock()

	icon := ""
	class := ""
	text := ""

	switch camera {
	case pixy.StateTracking:
		icon = "\uf030"
		class = string(pixy.StateTracking)
		text = "CAM"
	case pixy.StatePrivacy:
		icon = "\uf011"
		class = string(pixy.StatePrivacy)
		text = "OFF"
	case pixy.StateIdle:
		icon = "\uf03d"
		class = string(pixy.StateIdle)
		text = "IDLE"
	case pixy.StateOffline:
		icon = "\uf00d"
		class = string(pixy.StateOffline)
		text = "---"
	}

	if inCall {
		class += " in-call"
	}

	var tooltip strings.Builder
	tooltip.Grow(64)
	tooltip.WriteString("EMEET PIXY: ")
	tooltip.WriteString(string(camera))
	tooltip.WriteString("\nAudio: ")
	tooltip.WriteString(string(audio))
	tooltip.WriteString("\nAuto: ")
	tooltip.WriteString(autoMode.String())
	if inCall {
		tooltip.WriteString("\nIn call: yes")
	}

	out := waybarJSON{
		Text:    icon + " " + text,
		Tooltip: tooltip.String(),
		Class:   "custom-camera " + class,
	}

	data, err := json.Marshal(out)
	if err != nil {
		return `{"text":"?","tooltip":"json marshal error","class":"custom-camera offline"}`
	}

	return string(data)
}
