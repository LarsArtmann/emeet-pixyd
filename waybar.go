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

const tooltipInitSize = 64

type waybarCameraInfo struct {
	icon  string
	class string
	text  string
}

//nolint:gochecknoglobals
var waybarCameraStates = map[pixy.CameraState]waybarCameraInfo{
	pixy.StateTracking: {icon: "\uf030", class: string(pixy.StateTracking), text: "CAM"},
	pixy.StatePrivacy:  {icon: "\uf011", class: string(pixy.StatePrivacy), text: "OFF"},
	pixy.StateIdle:     {icon: "\uf03d", class: string(pixy.StateIdle), text: "IDLE"},
	pixy.StateOffline:  {icon: "\uf00d", class: string(pixy.StateOffline), text: "---"},
}

func (d *Daemon) waybarOutput() string {
	d.mu.RLock()
	camera := d.state.Camera
	audio := d.state.Audio
	inCall := d.state.InCall
	autoMode := d.state.AutoMode
	d.mu.RUnlock()

	info := waybarCameraStates[camera]
	class := info.class

	if inCall {
		class += " in-call"
	}

	var tooltip strings.Builder
	tooltip.Grow(tooltipInitSize)
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
		Text:    info.icon + " " + info.text,
		Tooltip: tooltip.String(),
		Class:   "custom-camera " + class,
	}

	data, err := json.Marshal(out)
	if err != nil {
		return `{"text":"?","tooltip":"json marshal error","class":"custom-camera offline"}`
	}

	return string(data)
}
