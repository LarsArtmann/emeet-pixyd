//go:build linux

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func newTestWebServer(t *testing.T, daemon *Daemon) *httptest.Server {
	t.Helper()

	webSrv := &webServer{daemon: daemon}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

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

func TestWeb_PanelShowsPresetsWhenOnline(t *testing.T) {
	t.Parallel()

	daemon := newDaemonWithDevice(t)
	daemon.mu.Lock()
	daemon.state.Presets = map[string]pixy.PTZValues{
		"home":    {Pan: 0, Tilt: 0, Zoom: 100},
		"wide":    {Pan: 30, Tilt: -10, Zoom: 120},
		"closeup": {Pan: -20, Tilt: 5, Zoom: 140},
	}
	daemon.mu.Unlock()

	srv := newTestWebServer(t, daemon)

	resp := get(t, srv.URL+"/panel")
	defer resp.Body.Close() //nolint:errcheck

	body := getBody(t, resp)

	assertContains(t, body, "preset-section", "preset section")
	assertContains(t, body, "preset-chip", "preset chips")
	assertContains(t, body, "/api/preset/load/home", "load home preset")
	assertContains(t, body, "/api/preset/delete/wide", "delete wide preset")
	assertContains(t, body, "preset-name-input", "save input")
	assertContains(t, body, "3/16", "preset count")
}

func TestWeb_PanelHidesPresetsWhenOffline(t *testing.T) {
	t.Parallel()

	daemon := newIntegrationDaemon(t)
	srv := newTestWebServer(t, daemon)

	resp := get(t, srv.URL+"/panel")
	defer resp.Body.Close() //nolint:errcheck

	body := getBody(t, resp)

	if strings.Contains(body, "preset-section") {
		t.Error("preset section should not appear when offline")
	}
}

func TestWeb_PresetSaveRejectsInvalidName(t *testing.T) {
	t.Parallel()

	daemon := newDaemonWithDevice(t)
	srv := newTestWebServer(t, daemon)

	longName := strings.Repeat("a", pixy.MaxPresetNameLength+1)

	resp := post(t, srv.URL+"/api/preset/save/"+longName, "application/json", nil)
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusBadRequest)
}
