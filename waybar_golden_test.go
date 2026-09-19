//go:build linux

package main

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestWaybarGoldenJSON(t *testing.T) {
	t.Parallel()

	trackingText := "\uf030 CAM"
	privacyText := "\uf011 OFF"
	idleText := "\uf03d IDLE"
	offlineText := "\uf00d ---"

	tests := []struct {
		name     string
		camera   pixy.CameraState
		inCall   bool
		expected waybarJSON
	}{
		{
			name: "tracking_not_in_call", camera: pixy.StateTracking, inCall: false,
			expected: waybarJSON{
				Text: trackingText, Tooltip: "EMEET PIXY: tracking\nAudio: nc\nAuto: full",
				Class: "custom-camera tracking",
			},
		},
		{
			name: "tracking_in_call", camera: pixy.StateTracking, inCall: true,
			expected: waybarJSON{
				Text:    trackingText,
				Tooltip: "EMEET PIXY: tracking\nAudio: nc\nAuto: full\nIn call: yes",
				Class:   "custom-camera tracking in-call",
			},
		},
		{
			name: "privacy_not_in_call", camera: pixy.StatePrivacy, inCall: false,
			expected: waybarJSON{
				Text: privacyText, Tooltip: "EMEET PIXY: privacy\nAudio: nc\nAuto: full",
				Class: "custom-camera privacy",
			},
		},
		{
			name: "privacy_in_call", camera: pixy.StatePrivacy, inCall: true,
			expected: waybarJSON{
				Text:    privacyText,
				Tooltip: "EMEET PIXY: privacy\nAudio: nc\nAuto: full\nIn call: yes",
				Class:   "custom-camera privacy in-call",
			},
		},
		{
			name: "idle_not_in_call", camera: pixy.StateIdle, inCall: false,
			expected: waybarJSON{
				Text: idleText, Tooltip: "EMEET PIXY: idle\nAudio: nc\nAuto: full",
				Class: "custom-camera idle",
			},
		},
		{
			name: "idle_in_call", camera: pixy.StateIdle, inCall: true,
			expected: waybarJSON{
				Text:    idleText,
				Tooltip: "EMEET PIXY: idle\nAudio: nc\nAuto: full\nIn call: yes",
				Class:   "custom-camera idle in-call",
			},
		},
		{
			name: "offline_not_in_call", camera: pixy.StateOffline, inCall: false,
			expected: waybarJSON{
				Text: offlineText, Tooltip: "EMEET PIXY: offline\nAudio: nc\nAuto: full",
				Class: "custom-camera offline",
			},
		},
		{
			name: "offline_in_call", camera: pixy.StateOffline, inCall: true,
			expected: waybarJSON{
				Text:    offlineText,
				Tooltip: "EMEET PIXY: offline\nAudio: nc\nAuto: full\nIn call: yes",
				Class:   "custom-camera offline in-call",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := testDaemonWithState(t, tc.camera, tc.inCall)

			output := d.waybarOutput(t.Context())

			expectedJSON, err := json.Marshal(tc.expected)
			if err != nil {
				t.Fatalf("marshal expected: %v", err)
			}

			if output != string(expectedJSON) {
				t.Errorf("golden JSON mismatch:\nwant: %s\ngot:  %s", expectedJSON, output)
			}
		})
	}
}

func TestWaybarModelInTooltipAndJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		model         pixy.Model
		wantTooltip   string
		wantModelJSON string
	}{
		{
			name:          "original_pixy",
			model:         pixy.ModelOriginal,
			wantTooltip:   "EMEET PIXY (PIXY): ",
			wantModelJSON: "PIXY",
		},
		{
			name:          "pixy_2k",
			model:         pixy.Model2K,
			wantTooltip:   "EMEET PIXY (PIXY 2K): ",
			wantModelJSON: "PIXY 2K",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := testDaemonWithState(t, pixy.StateTracking, false)
			d.model = tc.model

			var out waybarJSON

			if err := json.Unmarshal([]byte(d.waybarOutput(t.Context())), &out); err != nil {
				t.Fatalf("unmarshal waybar JSON: %v", err)
			}

			if !strings.Contains(out.Tooltip, tc.wantTooltip) {
				t.Errorf("tooltip = %q, want containing %q", out.Tooltip, tc.wantTooltip)
			}

			if out.Model != tc.wantModelJSON {
				t.Errorf("model field = %q, want %q", out.Model, tc.wantModelJSON)
			}
		})
	}
}

func TestWaybarModelOmittedWhenUnknown(t *testing.T) {
	t.Parallel()

	d := testDaemonWithState(t, pixy.StateIdle, false)

	var out map[string]any

	if err := json.Unmarshal([]byte(d.waybarOutput(t.Context())), &out); err != nil {
		t.Fatalf("unmarshal waybar JSON: %v", err)
	}

	if _, present := out["model"]; present {
		t.Errorf("model field present for unknown model, want omitted: %v", out["model"])
	}
}

// TestWaybarBatteryClass pins the charging/discharging class tag (TODO #139
// remainder): a battery reading appends the ChargeSta-derived state class so
// user CSS can style it; no reading leaves the class untouched.
func TestWaybarBatteryClass(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	var out waybarJSON

	if err := json.Unmarshal([]byte(d.waybarOutput(t.Context())), &out); err != nil {
		t.Fatalf("unmarshal waybar JSON: %v", err)
	}

	if !strings.Contains(out.Class, "discharging") || out.Battery != "87% (discharging)" {
		t.Errorf("discharging output = class %q battery %q, want discharging class + 87%%", out.Class, out.Battery)
	}

	sim.state.mu.Lock()
	sim.state.chargeSta = 2 // the second official charging value must tag identically
	sim.state.mu.Unlock()

	d.powerCache.Invalidate()

	out = waybarJSON{}
	if err := json.Unmarshal([]byte(d.waybarOutput(t.Context())), &out); err != nil {
		t.Fatalf("unmarshal waybar JSON: %v", err)
	}

	if !strings.Contains(out.Class, "charging") {
		t.Errorf("charging output class = %q, want charging tag", out.Class)
	}

	if strings.Contains(out.Class, "discharging") {
		t.Errorf("charging output class = %q, must not also tag discharging", out.Class)
	}
}
