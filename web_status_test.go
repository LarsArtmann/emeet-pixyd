//go:build linux

package main

import (
	"encoding/json/v2"
	"net/http"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func decodeStatusResponse(t *testing.T, resp *http.Response) statusResponse {
	t.Helper()

	var got statusResponse

	if err := json.Unmarshal([]byte(getBody(t, resp)), &got); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}

	return got
}

// TestWeb_StatusEndpointOffline pins the widget contract while the camera is
// unplugged: the endpoint must stay 200 (so a bar widget can tell "daemon
// down" from "camera offline") and report online=false.
func TestWeb_StatusEndpointOffline(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/status")
	defer resp.Body.Close()

	assertStatusCode(t, resp, http.StatusOK)

	got := decodeStatusResponse(t, resp)

	if got.Online {
		t.Errorf("Online = true, want false for a daemon without a device")
	}

	if got.Camera != pixy.StatePrivacy {
		t.Errorf("Camera = %q, want %q", got.Camera, pixy.StatePrivacy)
	}

	if got.Device != "" {
		t.Errorf("Device = %q, want empty when offline", got.Device)
	}

	if got.Version != buildVersion {
		t.Errorf("Version = %q, want %q", got.Version, buildVersion)
	}

	if got.Auto != pixy.AutoFull {
		t.Errorf("Auto = %q, want %q", got.Auto, pixy.AutoFull)
	}

	if got.Audio != pixy.AudioNC {
		t.Errorf("Audio = %q, want %q", got.Audio, pixy.AudioNC)
	}
}

// TestWeb_StatusEndpointOnline pins the same contract with a device present:
// online=true and the device path echoed back for widget tooltips.
func TestWeb_StatusEndpointOnline(t *testing.T) {
	t.Parallel()
	daemon := newDaemonWithDevice(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/status")
	defer resp.Body.Close()

	assertStatusCode(t, resp, http.StatusOK)

	got := decodeStatusResponse(t, resp)

	if !got.Online {
		t.Errorf("Online = false, want true with a device present")
	}

	if got.Device != testVideoDev {
		t.Errorf("Device = %q, want %q", got.Device, testVideoDev)
	}
}

// TestWeb_StatusEndpointJSONShape guards the field names widget authors depend
// on. The battery key is omitted (not null) when the device does not answer
// battery queries.
func TestWeb_StatusEndpointJSONShape(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/status")
	defer resp.Body.Close()

	body := getBody(t, resp)

	for _, want := range []string{
		`"camera":"privacy"`,
		`"online":false`,
		`"inCall":false`,
		`"version":"dev"`,
	} {
		assertContains(t, body, want, "status JSON shape")
	}

	assertNotContains(t, body, `"battery"`, "battery omitted when unavailable")
}
