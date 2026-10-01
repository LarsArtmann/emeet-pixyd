//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

type waybarJSON struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
	Model   string `json:"model,omitzero"`
	Battery string `json:"battery,omitzero"`
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

func (d *Daemon) waybarOutput(ctx context.Context) string {
	s := d.snapshot()

	// The bar is a human surface: collapse to offline when no video node is
	// present (the same collapse the CLI `status` line does), rather than
	// showing the desired mode for a camera that is not there. The machine
	// surface (/api/status) keeps camera=intent + online=false instead.
	camera := s.Camera
	if !s.Online {
		camera = pixy.StateOffline
	}

	info := waybarCameraStates[camera]
	class := info.class

	if s.InCall {
		class += " in-call"
	}

	var tooltip strings.Builder
	tooltip.Grow(tooltipInitSize)
	tooltip.WriteString("EMEET PIXY")

	if s.Model != "" {
		tooltip.WriteString(" (")
		tooltip.WriteString(string(s.Model))
		tooltip.WriteString(")")
	}

	tooltip.WriteString(": ")
	tooltip.WriteString(string(camera))
	tooltip.WriteString("\nAudio: ")
	tooltip.WriteString(string(s.Audio))
	tooltip.WriteString("\nAuto: ")
	tooltip.WriteString(s.Auto.String())

	if s.InCall {
		tooltip.WriteString("\nIn call: yes")
	}

	// Battery line (TODO #139) appears only when the device answers the
	// official HID battery queries; waybar output stays stable otherwise.
	// A reading additionally tags the class with charging/discharging so
	// user CSS can style the two states (ChargeSta semantics: {1,2}=charging).
	battery := ""

	reading, ok := d.powerStatus(ctx)

	if ok {
		battery = reading.String()

		stateClass := "discharging"
		if reading.Charging {
			stateClass = "charging"
		}

		class += " " + stateClass

		tooltip.WriteString("\nBattery: ")
		tooltip.WriteString(battery)
	}

	out := waybarJSON{
		Text:    info.icon + " " + info.text,
		Tooltip: tooltip.String(),
		Class:   "custom-camera " + class,
		Model:   string(s.Model),
		Battery: battery,
	}

	data, err := json.Marshal(out)
	if err != nil {
		return `{"text":"?","tooltip":"json marshal error","class":"custom-camera offline"}`
	}

	return string(data)
}
