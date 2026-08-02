//go:build linux

package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestWeb_AudioWithValidModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"nc", audioModeLive, audioModeOrg} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			daemon := newIntegrationDaemon(t)

			server := newTestWebServer(t, daemon)

			resp := post(t, server.URL+"/api/audio/"+mode, "application/json", strings.NewReader("{}"))

			defer resp.Body.Close() //nolint:errcheck

			assertStatusCode(t, resp, http.StatusOK)
		})
	}
}

func TestWeb_AudioInvalidMode(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := post(t, server.URL+"/api/audio/blorp", "application/json", strings.NewReader("{}"))
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusOK)
	assertResponseContains(t, resp, "status-panel", "still returns panel even on invalid mode")
}

func TestWeb_PTZEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path   string
		body   string
		status int
	}{
		{"/api/ptz/", "", http.StatusBadRequest},
		{"/api/ptz/pan", "", http.StatusBadRequest},
		{"/api/ptz/pan", `{"pan":10}`, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			testPTZEndpoint(t, tc.path, tc.body, tc.status)
		})
	}
}

func TestWeb_SnapshotNoDevice(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/snapshot")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusServiceUnavailable)
	assertResponseContains(t, resp, "no frame available", "503 body")
}

func TestWeb_StreamNoDevice(t *testing.T) {
	t.Parallel()
	testGETEndpoint503(t, "/api/stream")
}

func TestWeb_POSTEndpointsRejectGET(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	endpoints := []string{
		"/api/track",

		"/api/idle",

		"/api/privacy",

		"/api/toggle-privacy",

		"/api/audio/nc",

		"/api/gesture",

		"/api/auto",

		"/api/center",

		"/api/sync",

		"/api/probe",
	}
	assertEndpointsReturnNonOK(t, server.URL, "GET", endpoints)
}

func TestWeb_GETEndpointsRejectPOST(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	endpoints := []string{"/", "/panel", "/api/snapshot", "/api/stream"}
	assertEndpointsReturnNonOK(t, server.URL, "POST", endpoints)
}

func TestWeb_UnknownRouteReturns404(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/nonexistent")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusNotFound)
}

func TestWeb_HealthEndpoint(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/health")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusServiceUnavailable)
	body := getBody(t, resp)
	assertContains(t, body, `"status":"offline"`, "health response")
	assertContains(t, body, `"version":"dev"`, "health version")
}

func TestWeb_HealthEndpointOnline(t *testing.T) {
	t.Parallel()
	daemon := newDaemonWithDevice(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+"/api/health")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusOK)
	body := getBody(t, resp)
	assertContains(t, body, `"status":"ok"`, "health response")
}

func TestWeb_WebStatusOfflineNoDevice(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	webSrv := &webServer{daemon: daemon}
	status := webSrv.getWebStatus()
	assertWebStatus(t, status)
}

func TestWeb_WebStatusOnlineWithDevice(t *testing.T) {
	t.Parallel()
	daemon := newDaemonWithDevice(t)
	daemon.state.Camera = pixy.StateTracking
	daemon.state.Audio = pixy.AudioLive
	daemon.state.Gesture = true
	daemon.state.InCall = true
	webSrv := &webServer{daemon: daemon}
	status := webSrv.getWebStatus()

	assertWebStatusField(t, status, webStatusCheck{
		Camera: ptr(pixy.StateTracking),

		Audio: ptr(pixy.AudioLive),

		Gesture: new(true),

		Auto: ptr(pixy.AutoFull),

		InCall: new(true),

		Online: new(true),

		Device: new("/dev/video0"),

		Zoom: new(100),
	})
}

func TestWeb_WebStatusAllCameraStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		camera pixy.CameraState
	}{
		{pixy.StateTracking},

		{pixy.StatePrivacy},

		{pixy.StateIdle},

		{pixy.StateOffline},
	}
	for _, tc := range tests {
		t.Run(string(tc.camera), func(t *testing.T) {
			t.Parallel()
			daemon := newIntegrationDaemon(t)

			daemon.videoDev = "/dev/video0"

			daemon.state.Camera = tc.camera

			webSrv := &webServer{daemon: daemon}

			status := webSrv.getWebStatus()

			if status.Camera != tc.camera {
				t.Errorf("expected camera=%s, got %s", tc.camera, status.Camera)
			}
		})
	}
}

func TestWeb_WebStatusAllAudioModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		audio pixy.AudioMode
	}{
		{pixy.AudioNC},

		{pixy.AudioLive},

		{pixy.AudioOriginal},
	}
	for _, tc := range tests {
		t.Run(string(tc.audio), func(t *testing.T) {
			t.Parallel()
			daemon := newIntegrationDaemon(t)

			daemon.state.Audio = tc.audio

			webSrv := &webServer{daemon: daemon}

			status := webSrv.getWebStatus()

			if status.Audio != tc.audio {
				t.Errorf("expected audio=%s, got %s", tc.audio, status.Audio)
			}
		})
	}
}
