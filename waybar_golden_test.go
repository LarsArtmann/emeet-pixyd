//go:build linux

package main

import (
	"encoding/json/v2"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestWaybarGoldenJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		camera   pixy.CameraState
		inCall   bool
		expected waybarJSON
	}{
		{
			name:     "tracking_not_in_call",
			camera:   pixy.StateTracking,
			inCall:   false,
			expected: waybarJSON{Text: "\uf030 CAM", Tooltip: "EMEET PIXY: tracking\nAudio: nc\nAuto: full", Class: "custom-camera tracking"},
		},
		{
			name:     "tracking_in_call",
			camera:   pixy.StateTracking,
			inCall:   true,
			expected: waybarJSON{Text: "\uf030 CAM", Tooltip: "EMEET PIXY: tracking\nAudio: nc\nAuto: full\nIn call: yes", Class: "custom-camera tracking in-call"},
		},
		{
			name:     "privacy_not_in_call",
			camera:   pixy.StatePrivacy,
			inCall:   false,
			expected: waybarJSON{Text: "\uf011 OFF", Tooltip: "EMEET PIXY: privacy\nAudio: nc\nAuto: full", Class: "custom-camera privacy"},
		},
		{
			name:     "privacy_in_call",
			camera:   pixy.StatePrivacy,
			inCall:   true,
			expected: waybarJSON{Text: "\uf011 OFF", Tooltip: "EMEET PIXY: privacy\nAudio: nc\nAuto: full\nIn call: yes", Class: "custom-camera privacy in-call"},
		},
		{
			name:     "idle_not_in_call",
			camera:   pixy.StateIdle,
			inCall:   false,
			expected: waybarJSON{Text: "\uf03d IDLE", Tooltip: "EMEET PIXY: idle\nAudio: nc\nAuto: full", Class: "custom-camera idle"},
		},
		{
			name:     "idle_in_call",
			camera:   pixy.StateIdle,
			inCall:   true,
			expected: waybarJSON{Text: "\uf03d IDLE", Tooltip: "EMEET PIXY: idle\nAudio: nc\nAuto: full\nIn call: yes", Class: "custom-camera idle in-call"},
		},
		{
			name:     "offline_not_in_call",
			camera:   pixy.StateOffline,
			inCall:   false,
			expected: waybarJSON{Text: "\uf00d ---", Tooltip: "EMEET PIXY: offline\nAudio: nc\nAuto: full", Class: "custom-camera offline"},
		},
		{
			name:     "offline_in_call",
			camera:   pixy.StateOffline,
			inCall:   true,
			expected: waybarJSON{Text: "\uf00d ---", Tooltip: "EMEET PIXY: offline\nAudio: nc\nAuto: full\nIn call: yes", Class: "custom-camera offline in-call"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := testDaemonWithState(t, tc.camera, tc.inCall)
			output := d.waybarOutput()

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
