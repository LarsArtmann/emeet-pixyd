//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"

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

	daemon := newTestDaemon(t, pixy.StatePrivacy, testVideoDev, "")
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

	daemon := testDaemonWithState(t, pixy.StateTracking, false)

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

		daemon := newTestDaemon(t, pixy.StateTracking, "", "")

		result := daemon.handleCommand(context.Background(), cmdStatus)
		assertStatusPrefix(t, result.String(), "camera=offline", "tracking intent without video node")
		assertStatusContains(t, result.String(), "device=", "offline status line")
	})

	t.Run("video_without_hid_keeps_intent", func(t *testing.T) {
		t.Parallel()

		daemon := newTestDaemon(t, pixy.StateTracking, testVideoDev, "", withNoopParsePTZ())

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

	daemon := newTestDaemon(t, pixy.StateTracking, testVideoDev, "", withNoopParsePTZ())
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	if !strings.Contains(body, "disabled") {
		t.Error("expected HID-backed controls to render disabled without the HID node")
	}

	assertNotContains(t, body, "tracking-variants", "tracking-variant strip requires controllable")

	assertContains(t, body, "slider-pan", "position card renders while video node present")
}
