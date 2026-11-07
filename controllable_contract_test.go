//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// The controllable contract (retro 2026-10-01_11-37 c3/f6-f10): `online`
// means the video node is present, `controllable` means the HID node is
// present — the two degrade independently (a uvcvideo node can enumerate
// while the vendor hidraw node is missing or the udev rule never fired).
// The machine surface (/api/status) reports both flags verbatim; the human
// surfaces (CLI status line, Waybar) collapse the camera mode to offline
// only when the VIDEO node is absent, never because HID is gone. These pins
// keep the field-table row in
// website/src/content/docs/guides/quickshell.mdx honest.

// TestControllableContract_StatusVideoWithoutHID pins the /api/status shape
// in the split state: video node present, HID node absent. The endpoint must
// stay 200, report online=true + controllable=false, and keep camera as the
// desired intent.
func TestControllableContract_StatusVideoWithoutHID(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, pixy.DesiredPrivacy, testVideoDev, "")
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/status")
	defer resp.Body.Close()

	assertStatusCode(t, resp, http.StatusOK)

	body := getBody(t, resp)

	var got statusResponse

	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}

	if !got.Online {
		t.Error("Online = false, want true (video node present)")
	}

	if got.Controllable {
		t.Error("Controllable = true, want false (HID node absent)")
	}

	if got.Camera != pixy.StatePrivacy {
		t.Errorf("Camera = %q, want %q (intent must survive the HID loss)", got.Camera, pixy.StatePrivacy)
	}

	assertContains(t, body, `"controllable":false`, "status JSON controllable shape")
}

// TestControllableContract_WaybarVideoWithoutHID pins the human surface:
// with the video node present the bar keeps showing the desired mode even
// though HID (and therefore battery/mode control) is gone — the collapse is
// keyed on the video node only.
func TestControllableContract_WaybarVideoWithoutHID(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithState(t, pixy.DesiredTracking, false)

	output := daemon.waybarOutput(t.Context())

	assertContains(t, output, `"class":"custom-camera tracking"`, "waybar class keeps intent while video node present")
	assertContains(t, output, `EMEET PIXY: tracking`, "waybar tooltip keeps intent while video node present")
}

// TestControllableContract_StatusLine pins the CLI collapse behavior on both
// sides of the video-node boundary: no video node collapses the mode to
// offline even when the persisted intent is tracking; video-without-HID
// keeps the intent and still echoes the device path.
func TestControllableContract_StatusLine(t *testing.T) {
	t.Parallel()

	t.Run("no_video_node_collapses_to_offline", func(t *testing.T) {
		t.Parallel()

		daemon := newTestDaemon(t, pixy.DesiredTracking, "", "")

		result := daemon.handleCommand(context.Background(), cmdStatus)
		assertStatusPrefix(t, result.String(), "camera=offline", "tracking intent without video node")
		assertStatusContains(t, result.String(), "device=", "offline status line")
	})

	t.Run("video_without_hid_keeps_intent", func(t *testing.T) {
		t.Parallel()

		daemon := newTestDaemon(t, pixy.DesiredTracking, testVideoDev, "", withNoopParsePTZ())

		result := daemon.handleCommand(context.Background(), cmdStatus)
		assertStatusPrefix(t, result.String(), "camera=tracking", "tracking intent with video node present")
		assertStatusContains(t, result.String(), "device="+testVideoDev, "online status line")
	})
}

// TestControllableContract_PanelVideoWithoutHID pins the panel degradation:
// HID-backed controls (mode cards, audio segments, gesture switch) render
// disabled, the tracking-variant strip disappears (it needs HID), while the
// V4L2-backed position card still renders because the video node is present.
func TestControllableContract_PanelVideoWithoutHID(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, pixy.DesiredTracking, testVideoDev, "", withNoopParsePTZ())
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	if !strings.Contains(body, "disabled") {
		t.Error("expected HID-backed controls to render disabled without the HID node")
	}

	assertNotContains(t, body, "tracking-variants", "tracking-variant strip requires controllable")

	assertContains(t, body, "slider-pan", "position card renders while video node present")
}

// TestStatusContract_BatteryFieldsAdditive pins the 2026-11-07 additive
// battery split in /api/status: the legacy `battery` display string is
// byte-identical to what widgets already parse, and the new
// batteryPercent/batteryCharging keys appear (and are typed) only when the
// device answered a battery query — absent otherwise, never null.
func TestStatusContract_BatteryFieldsAdditive(t *testing.T) {
	t.Parallel()

	d := testDaemonWithState(t, pixy.DesiredPrivacy, false)
	d.powerCache.Set(
		powerCacheEntry{reading: powerReading{Level: 87, Charging: true}, available: true},
		time.Minute,
	)

	srv := &webServer{daemon: d}
	mux := newWebMux(srv)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp := get(t, server.URL+"/api/status")
	body := getBody(t, resp)
	resp.Body.Close()

	var parsed map[string]any

	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}

	if got, ok := parsed["battery"].(string); !ok || got != "87% (charging)" {
		t.Errorf("battery = %q (%v), want the legacy display string \"87%% (charging)\"", parsed["battery"], ok)
	}

	if got, ok := parsed["batteryPercent"].(float64); !ok || got != 87 {
		t.Errorf("batteryPercent = %v (%v), want 87 as a JSON number", parsed["batteryPercent"], ok)
	}

	if got, ok := parsed["batteryCharging"].(bool); !ok || !got {
		t.Errorf("batteryCharging = %v (%v), want true", parsed["batteryCharging"], ok)
	}

	assertContains(t, body, `"batteryPercent":87`, "status JSON batteryPercent shape")
	assertContains(t, body, `"batteryCharging":true`, "status JSON batteryCharging shape")
}
