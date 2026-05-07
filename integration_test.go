//go:build linux

package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	testWebAddr   = "127.0.0.1:0"
	audioModeLive = "live"
	audioModeOrg  = "org"
)

func newIntegrationDaemon(t *testing.T) *Daemon {
	t.Helper()
	return newTestDaemon(pixy.StatePrivacy, "", "", func(d *Daemon) {
		d.config = testConfig(t.TempDir())
	})
}

func newDaemonWithDevice(t *testing.T) *Daemon {
	t.Helper()
	return newTestDaemon(pixy.StatePrivacy, "/dev/video0", "/dev/hidraw7", func(d *Daemon) {
		d.config = testConfig(t.TempDir())
	})
}

func newTestWebServer(t *testing.T, daemon *Daemon) *httptest.Server {
	t.Helper()
	webSrv := &webServer{daemon: daemon}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func getBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func assertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("expected %d, got %d", expected, resp.StatusCode)
	}
}

func assertContains(t *testing.T, haystack, needle, label string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected body to contain %q", label, needle)
	}
}

func assertNotContains(t *testing.T, haystack, needle, label string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("%s: expected body NOT to contain %q", label, needle)
	}
}

func assertResponseContains(t *testing.T, resp *http.Response, substr, label string) {
	t.Helper()
	assertContains(t, getBody(t, resp), substr, label)
}

func post(t *testing.T, url, contentType string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, body)
	if err != nil {
		t.Fatalf("new POST request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new GET request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func sendSC(t *testing.T, socketPath, cmd string) string {
	t.Helper()
	resp, err := pixy.SendCommand(context.Background(), socketPath, cmd)
	if err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	return resp
}

func daemonHasDevices(d *Daemon) bool {
	return d.videoDev != "" || d.hidrawDev != ""
}

func assertEndpointsReturnNonOK(t *testing.T, serverURL, method string, endpoints []string) {
	t.Helper()
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			var (
				resp *http.Response

				err error
			)

			if method == "GET" {
				req, reqErr := http.NewRequestWithContext(
					context.Background(),
					http.MethodGet,
					serverURL+ep,
					nil,
				)
				if reqErr != nil {
					t.Fatalf("new GET request: %v", reqErr)
				}
				resp, err = http.DefaultClient.Do(req)
			} else {
				req, reqErr := http.NewRequestWithContext(
					context.Background(),
					http.MethodPost,
					serverURL+ep,
					nil,
				)
				if reqErr != nil {
					t.Fatalf("new POST request: %v", reqErr)
				}
				resp, err = http.DefaultClient.Do(req)
			}

			if err != nil {
				t.Fatalf("%s %s: %v", method, ep, err)
			}

			resp.Body.Close() //nolint:errcheck

			if resp.StatusCode == http.StatusOK {
				t.Errorf("%s %s should not be 200, got %d", method, ep, resp.StatusCode)
			}
		})
	}
}

// assertWebStatusOffline verifies all fields match offline/no-device state.
func assertWebStatusOffline(t *testing.T, status webStatus) {
	t.Helper()
	assertWebStatusField(t, status, webStatusCheck{
		Camera: ptr(pixy.StatePrivacy),

		Audio: ptr(pixy.AudioNC),

		Gesture: new(false),

		Auto: ptr(pixy.AutoFull),

		InCall: new(false),

		Online: new(false),

		Device: new(""),

		Pan: new(0), Tilt: new(0), Zoom: new(0),
	})
}

type webStatusCheck struct {
	Camera  *pixy.CameraState
	Audio   *pixy.AudioMode
	Gesture *bool
	Auto    *pixy.AutoMode
	InCall  *bool
	Online  *bool
	Device  *string
	Pan     *int
	Tilt    *int
	Zoom    *int
}

func assertPtrEqual[T comparable](t *testing.T, name string, got T, want *T) {
	t.Helper()
	if want != nil && got != *want {
		t.Errorf("expected %s=%v, got %v", name, *want, got)
	}
}

func assertWebStatusField(t *testing.T, status webStatus, check webStatusCheck) {
	t.Helper()
	assertPtrEqual(t, "camera", status.Camera, check.Camera)
	assertPtrEqual(t, "audio", status.Audio, check.Audio)
	assertPtrEqual(t, "gesture", status.Gesture, check.Gesture)
	assertPtrEqual(t, "auto", status.Auto, check.Auto)
	assertPtrEqual(t, "inCall", status.InCall, check.InCall)
	assertPtrEqual(t, "online", status.Online, check.Online)
	assertPtrEqual(t, "device", status.Device, check.Device)
	assertPtrEqual(t, "pan", status.Pan, check.Pan)
	assertPtrEqual(t, "tilt", status.Tilt, check.Tilt)
	assertPtrEqual(t, "zoom", status.Zoom, check.Zoom)
}

func assertWebStatus(t *testing.T, status webStatus) {
	t.Helper()
	assertWebStatusOffline(t, status)
}

func assertSocketResponseContains(t *testing.T, resp, substr, label string) {
	t.Helper()
	if !strings.Contains(resp, substr) {
		t.Errorf("%s: expected %q in response, got: %s", label, substr, resp)
	}
}

func assertSocketResponsePrefix(t *testing.T, resp, prefix, label string) {
	t.Helper()
	if !strings.HasPrefix(resp, prefix) {
		t.Errorf("%s: expected prefix %q, got: %s", label, prefix, resp)
	}
}

func assertSocketResponseHasPrefixes(t *testing.T, resp string, prefixes []string) {
	t.Helper()
	for _, p := range prefixes {
		assertSocketResponseContains(t, resp, p, "socket response")
	}
}

func assertCommandContainsAnyOf(t *testing.T, resp string, substrs []string, label string) {
	t.Helper()
	for _, s := range substrs {
		if strings.Contains(resp, s) {
			return
		}
	}
	t.Errorf("expected one of %v in %s, got: %s", substrs, label, resp)
}

func assertCommandResponse(t *testing.T, cmd string, substr, label string) {
	t.Helper()
	d := newDaemonWithDevice(t)
	resp := d.handleCommand(context.Background(), cmd)
	assertCommandContains(t, resp, substr, label)
}

// ---------- Index page ----------

func TestWeb_IndexReturnsHTML(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+"/")
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
	body := getBody(t, resp)
	assertContains(t, body, "<!doctype html>", "index page")
	assertContains(t, body, "EMEET PIXY", "index page title")
	assertContains(t, body, "status-panel", "index has status panel")
}

func TestWeb_IndexShowsOfflineWhenNoDevice(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+"/")
	defer resp.Body.Close() //nolint:errcheck
	body := getBody(t, resp)
	assertContains(t, body, "Offline", "offline badge")
	assertContains(t, body, "Camera offline", "camera offline text")
}

func TestWeb_IndexShowsOnlineWithDevice(t *testing.T) {
	t.Parallel()
	daemon := newDaemonWithDevice(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+"/")
	defer resp.Body.Close() //nolint:errcheck
	body := getBody(t, resp)
	assertContains(t, body, "Online", "online badge")
	assertNotContains(t, body, "Offline", "should not show offline")
}

// ---------- Status panel (HTMX partial) ----------

func TestWeb_PanelReturnsHTMLFragment(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+"/panel")
	defer resp.Body.Close() //nolint:errcheck
	body := getBody(t, resp)
	assertContains(t, body, "status-panel", "panel has status-panel div")
	assertContains(t, body, "Track", "panel has track button")
	assertContains(t, body, "Privacy", "panel has privacy button")
	assertContains(t, body, "Noise Cancel", "panel has audio buttons")
}

func TestWeb_PanelReflectsDaemonState(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+"/panel")
	defer resp.Body.Close() //nolint:errcheck
	body := getBody(t, resp)
	assertContains(t, body, "privacy", "panel shows privacy state")
	assertContains(t, body, "gesture", "panel has gesture control")
}

// ---------- Auto toggle ----------

func TestWeb_AutoToggleOff(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/auto", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
	daemon.mu.Lock()
	isAuto := daemon.state.AutoMode
	daemon.mu.Unlock()
	if !isAuto.IsOff() {
		t.Errorf("expected auto=off after toggle, got %q", isAuto)
	}
}

func TestWeb_AutoToggleOn(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	daemon.state.AutoMode = pixy.AutoOff
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/auto", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
	daemon.mu.Lock()
	isAuto := daemon.state.AutoMode
	daemon.mu.Unlock()
	if isAuto.IsOff() {
		t.Errorf("expected auto!=off after toggle, got %q", isAuto)
	}
}

func TestWeb_AutoToggleRoundTrip(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/auto", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	daemon.mu.Lock()
	if !daemon.state.AutoMode.IsOff() {
		t.Fatal("first toggle should turn auto off")
	}
	daemon.mu.Unlock()
	resp2 := post(t, server.URL+"/api/auto", "", nil)
	defer resp2.Body.Close() //nolint:errcheck
	daemon.mu.Lock()
	if daemon.state.AutoMode.IsOff() {
		t.Fatal("second toggle should turn auto back on")
	}
	daemon.mu.Unlock()
}

// ---------- Gesture toggle ----------

func TestWeb_GestureToggleEndpoint(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/gesture", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
}

func TestWeb_GestureToggleReturnsPanel(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/gesture", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertResponseContains(t, resp, "status-panel", "gesture response is panel fragment")
}

// ---------- Audio endpoint ----------

func TestWeb_AudioWithValidModes(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"nc", audioModeLive, audioModeOrg} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			daemon := newIntegrationDaemon(t)

			server := newTestWebServer(t, daemon)

			resp := post(

				t,

				server.URL+"/api/audio",

				"application/x-www-form-urlencoded",

				strings.NewReader("mode="+mode),
			)

			defer resp.Body.Close() //nolint:errcheck

			assertStatusCode(t, resp, http.StatusOK)
		})
	}
}

func TestWeb_AudioInvalidMode(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(

		t,

		server.URL+"/api/audio",

		"application/x-www-form-urlencoded",

		strings.NewReader("mode=blorp"),
	)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
	assertResponseContains(t, resp, "status-panel", "still returns panel even on invalid mode")
}

func TestWeb_AudioNoModeParam(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/audio", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
}

// ---------- PTZ endpoint ----------

func testPTZEndpoint(t *testing.T, path, body string, expectedStatus int) {
	t.Helper()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+path, "application/x-www-form-urlencoded", strings.NewReader(body))
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, expectedStatus)
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
		{"/api/ptz/pan", "value=10", http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			testPTZEndpoint(t, tc.path, tc.body, tc.status)
		})
	}
}

// ---------- Track/Idle/Privacy ----------

func testWebEndpointReturnsOK(t *testing.T, endpoint string) {
	t.Helper()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+endpoint, "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
}

func TestWeb_TrackEndpointNoDevice(t *testing.T) {
	t.Parallel()
	testWebEndpointReturnsOK(t, "/api/track")
}

func TestWeb_IdleEndpointNoDevice(t *testing.T) {
	t.Parallel()
	testWebEndpointReturnsOK(t, "/api/idle")
}

func TestWeb_PrivacyEndpointNoDevice(t *testing.T) {
	t.Parallel()
	testWebEndpointReturnsOK(t, "/api/privacy")
}

func TestWeb_TogglePrivacyEndpointNoDevice(t *testing.T) {
	t.Parallel()
	testWebEndpointReturnsOK(t, "/api/toggle-privacy")
}

func TestWeb_CenterEndpointNoDevice(t *testing.T) {
	t.Parallel()
	testWebEndpointReturnsOK(t, "/api/center")
}

// ---------- Probe/Sync ----------

func TestWeb_ProbeEndpoint(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/probe", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
	assertResponseContains(t, resp, "status-panel", "probe returns panel")
}

func TestWeb_SyncEndpointNoDevice(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := post(t, server.URL+"/api/sync", "", nil)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusOK)
}

// ---------- Snapshot/Stream (require device) ----------

func testGETEndpoint503(t *testing.T, path string) {
	t.Helper()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+path)
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusServiceUnavailable)
	assertResponseContains(t, resp, "no camera device", "503 body")
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

// ---------- Method enforcement ----------

func TestWeb_POSTEndpointsRejectGET(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	endpoints := []string{
		"/api/track",

		"/api/idle",

		"/api/privacy",

		"/api/toggle-privacy",

		"/api/audio",

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

// ---------- 404 for unknown routes ----------

func TestWeb_UnknownRouteReturns404(t *testing.T) {
	t.Parallel()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)
	resp := get(t, server.URL+"/api/nonexistent")
	defer resp.Body.Close() //nolint:errcheck
	assertStatusCode(t, resp, http.StatusNotFound)
}

// ---------- webStatus mapping ----------

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
	//nolint:exhaustruct
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

func shortSocketDir(t *testing.T) string {
	t.Helper()
	//nolint:usetesting // macOS t.TempDir() produces paths too long for Unix socket addresses
	dir, err := os.MkdirTemp("/tmp", "pxd-")
	if err != nil {
		t.Fatalf("create short temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// ---------- Daemon unix socket integration ----------

func startSocketDaemon(t *testing.T) (*Daemon, pixy.Config) {
	t.Helper()
	//nolint:exhaustruct
	cfg := pixy.Config{
		StateDir: shortSocketDir(t),

		PollInterval: 2 * time.Second,

		DebounceCount: 3,

		WebAddr: testWebAddr,
	}
	daemon, daemonErr := NewDaemon(cfg)
	if daemonErr != nil {
		t.Fatalf("NewDaemon: %v", daemonErr)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		_ = daemon.listenUnix(ctx)
	}()
	for range 50 {

		if _, statErr := os.Stat(cfg.SocketPath()); statErr == nil {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}
	return daemon, cfg
}

func TestSocket_StatusCommand(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdStatus)
	assertSocketResponseHasPrefixes(t, resp, []string{"camera=", "audio=", "auto=", "device="})
}

func TestSocket_AutoToggleRoundTrip(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), "auto-off")
	if resp != "auto mode: off" {
		t.Errorf("expected 'auto mode: off', got: %s", resp)
	}
	resp2 := sendSC(t, cfg.SocketPath(), "auto-on")
	if resp2 != "auto mode: full" {
		t.Errorf("expected 'auto mode: full', got: %s", resp2)
	}
}

func TestSocket_ProbeCommand(t *testing.T) {
	t.Parallel()
	daemon, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdProbe)
	if daemon.videoDev != "" {
		if !strings.HasPrefix(resp, "device found:") {
			t.Errorf("expected 'device found: ...', got: %s", resp)
		}
	} else {
		if resp != "device not found" {
			t.Errorf("expected 'device not found', got: %s", resp)
		}
	}
}

func TestSocket_WaybarCommand(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdWaybar)
	assertSocketResponseHasPrefixes(t, resp, []string{`"text"`, `"tooltip"`, `"class"`})
}

func TestSocket_DeviceCommand(t *testing.T) {
	t.Parallel()
	daemon, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdDevice)
	if daemon.videoDev != "" {
		if resp != daemon.videoDev {
			t.Errorf("expected %s, got: %s", daemon.videoDev, resp)
		}
	} else {
		if resp != "device not found" {
			t.Errorf("expected 'device not found', got: %s", resp)
		}
	}
}

func TestSocket_UnknownCommand(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), "foobar")
	assertSocketResponsePrefix(t, resp, "unknown command:", "socket response")
}

func TestSocket_StatusViaCommandReturnsStatus(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdStatus)
	assertSocketResponseContains(t, resp, "camera=", "socket response")
}

func TestSocket_CommandsNoDevice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string

		cmd string
	}{
		{cmdTrack, cmdTrack},

		{cmdPrivacy, cmdPrivacy},

		{cmdAudio, cmdAudio},

		{"gesture", cmdGestureOn},

		{cmdSync, cmdSync},

		{cmdCenter, cmdCenter},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			daemon, cfg := startSocketDaemon(t)

			if daemonHasDevices(daemon) {
				t.Skip("device connected")
			}

			resp := sendSC(t, cfg.SocketPath(), tc.cmd)
			assertSocketResponsePrefix(t, resp, "error:", "socket response")
		})
	}
}

func TestSocket_AudioInvalidMode(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), "audio badmode")
	if resp != "usage: audio [nc|live|org]" {
		t.Errorf("expected usage message, got: %s", resp)
	}
}

func TestSocket_AudioValidModes(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"nc", "live", "org"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			daemon, cfg := startSocketDaemon(t)

			if daemonHasDevices(daemon) {
				t.Skip("device connected, audio would succeed")
			}

			resp := sendSC(t, cfg.SocketPath(), "audio "+mode)
			assertSocketResponsePrefix(t, resp, "error:", "audio requires device")
		})
	}
}

func TestSocket_PanTiltZoom(t *testing.T) {
	t.Parallel()
	daemon, cfg := startSocketDaemon(t)
	if daemon.videoDev != "" {
		t.Skip("device connected")
	}

	for _, tc := range []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{"pan with value", "pan 10", true},
		{"tilt with value", "tilt -5", true},
		{"zoom with value", "zoom 200", true},
		{"pan missing value", axisPan, false},
		{"tilt missing value", "tilt", false},
		{"zoom missing value", axisZoom, false},
		{"pan invalid value", "pan abc", true},
		{"tilt invalid value", "tilt !", true},
		{"zoom invalid value", "zoom x", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := sendSC(t, cfg.SocketPath(), tc.cmd)
			if tc.wantErr {
				assertSocketResponsePrefix(t, resp, "error:", "socket response")
			} else if !strings.HasPrefix(resp, "usage:") {
				t.Errorf("expected usage for %q, got: %s", tc.cmd, resp)
			}
		})
	}
}

func TestSocket_TogglePrivacy(t *testing.T) {
	t.Parallel()
	_, cfg := startSocketDaemon(t)
	resp := sendSC(t, cfg.SocketPath(), cmdTogglePrivacy)
	assertCommandContainsAnyOf(t, resp,
		[]string{testStrPrivacy, testStrTracking}, "socket response")
}

func TestWeb_IndexContainsCameraButtons(t *testing.T) {
	t.Parallel()

	daemon := newDaemonWithDevice(t)
	srv := newTestWebServer(t, daemon)

	resp := get(t, srv.URL+"/")
	defer resp.Body.Close() //nolint:errcheck

	body := getBody(t, resp)
	html := body

	for _, want := range []string{
		`hx-post="/api/track"`,
		`hx-post="/api/idle"`,
		`hx-post="/api/privacy"`,
		`hx-post="/api/audio"`,
		`hx-post="/api/gesture"`,
		`hx-post="/api/auto"`,
		`hx-post="/api/center"`,
		`hx-post="/api/sync"`,
		`hx-post="/api/probe"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index HTML missing %q", want)
		}
	}

	bad := []string{`hx-post="endpoint"`, `aria-label="ariaLabel"`}
	for _, b := range bad {
		if strings.Contains(html, b) {
			t.Errorf("index HTML contains literal template variable: %q", b)
		}
	}
}

func TestWeb_PanelEndpointReturnsStatusPanel(t *testing.T) {
	t.Parallel()

	daemon := newDaemonWithDevice(t)
	srv := newTestWebServer(t, daemon)

	resp := get(t, srv.URL+"/panel")
	defer resp.Body.Close() //nolint:errcheck

	assertHTTPStatusOK(t, resp)

	body := getBody(t, resp)
	html := body

	if !strings.Contains(html, `id="status-panel"`) {
		t.Error("panel response missing #status-panel div")
	}
	if strings.Contains(html, `hx-trigger="every 3s, refresh from:body, load"`) {
		t.Error("panel still has 'load' trigger (infinite loop bug)")
	}
}

func TestHandleCommand_UnknownCommand(t *testing.T) {
	t.Parallel()
	assertCommandResponse(t, "foobar", "unknown command", "response")
}

func TestHandleCommand_StatusFormat(t *testing.T) {
	t.Parallel()

	d := newDaemonWithDevice(t)
	resp := d.handleCommand(context.Background(), "")
	assertCommandContainsAnyOf(t, resp, []string{"camera=", "audio="}, "status response")
}

func TestHandleCommand_AudioUsage(t *testing.T) {
	t.Parallel()
	assertCommandResponse(t, "audio badmode", "usage:", "response for bad audio mode")
}

func TestHandleCommand_PTZUsage(t *testing.T) {
	t.Parallel()
	assertCommandResponse(t, "pan", "usage:", "response for pan without value")
}

func TestHandleCommand_Device(t *testing.T) {
	t.Parallel()

	d := newDaemonWithDevice(t)
	resp := d.handleCommand(context.Background(), cmdDevice)
	assertCommandContains(t, resp, "/dev/video", "response")
}

func TestHandleCommand_DeviceNotFound(t *testing.T) {
	t.Parallel()

	d := newIntegrationDaemon(t)
	resp := d.handleCommand(context.Background(), cmdDevice)
	if resp != respDeviceNotFound {
		t.Errorf("expected device not found, got: %s", resp)
	}
}
