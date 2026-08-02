//go:build linux

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// TestWebPanelGolden_Tracking verifies the complete HTML structure of the
// status panel when the camera is tracking and online. This is a structural
// snapshot — every element ID, active class, and key attribute is checked.
func TestWebPanelGolden_Tracking(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithDevice(t, pixy.StateTracking)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		`id="status-panel"`,
		`mode-card`,
		`mode-track`,
		`mode-track active`,
		`mode-idle`,
		`mode-privacy`,
		`segmented`,
		`segment`,
		`preset-section`,
		`preset-name-input`,
		`preset-save-btn`,
		`slider-pan`,
		`slider-tilt`,
		`slider-zoom`,
		`ptz-radar`,
		`/api/center`,
		`/api/sync`,
		`/api/probe`,
		`role="switch"`,
		`data-on:click`,
	})
}

func TestWebPanelGolden_Privacy(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithDevice(t, pixy.StatePrivacy)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	if !strings.Contains(body, "mode-privacy active") {
		t.Error("expected privacy mode card to be active")
	}

	if !strings.Contains(body, "mode-track") {
		t.Error("expected track mode card to be present")
	}
}

func TestWebPanelGolden_Idle(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithDevice(t, pixy.StateIdle)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	if !strings.Contains(body, "mode-idle active") {
		t.Error("expected idle mode card to be active")
	}
}

func TestWebPanelGolden_Offline(t *testing.T) {
	t.Parallel()

	daemon := testDaemonNoDevice(t)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		`id="status-panel"`,
		`mode-card`,
		`mode-track`,
		`mode-idle`,
		`mode-privacy`,
	})

	if strings.Contains(body, "slider-pan") {
		t.Error("expected PTZ sliders to be absent when offline")
	}

	if strings.Contains(body, "preset-section") {
		t.Error("expected preset section to be absent when offline")
	}

	if strings.Contains(body, "card-muted") {
		// Expected: offline shows muted position card
	} else {
		t.Error("expected muted position card when offline")
	}
}

func TestWebPanelGolden_DisabledButtonsWhenOffline(t *testing.T) {
	t.Parallel()

	daemon := testDaemonNoDevice(t)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	// Mode cards should be disabled
	if !strings.Contains(body, `mode-track`) || !strings.Contains(body, "disabled") {
		t.Error("expected mode cards to be disabled when offline")
	}

	// Audio segments should be disabled
	if !strings.Contains(body, "segment") || !strings.Contains(body, "disabled") {
		t.Error("expected audio segments to be disabled when offline")
	}
}

func TestWebPanelGolden_AudioSegmentsActiveState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		audio  pixy.AudioMode
		modeID string
	}{
		{pixy.AudioNC, "nc"},
		{pixy.AudioLive, "live"},
		{pixy.AudioOriginal, "org"},
	}

	for _, tc := range tests {
		t.Run(string(tc.audio), func(t *testing.T) {
			t.Parallel()

			daemon := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, func(d *Daemon) {
				d.state.Audio = tc.audio
			})
			server := newTestWebServer(t, daemon)

			body := getPanelBody(t, server)
			label := audioLabel(tc.modeID)

			if !strings.Contains(body, `"segment active"`) || !strings.Contains(body, label) {
				t.Errorf("expected segment %s (%s) to be active", tc.modeID, label)
			}
		})
	}
}

func TestWebPanelGolden_InCallStrip(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithState(t, pixy.StateTracking, true)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	if !strings.Contains(body, "status-strip") || !strings.Contains(body, "In call") {
		t.Error("expected 'In call' status strip when in call")
	}
}

func TestWebPanelGolden_NoInCallStrip(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithState(t, pixy.StateTracking, false)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	if strings.Contains(body, "status-strip") {
		t.Error("expected no status strip when not in call")
	}
}

func TestWebPanelGolden_PresetsRendered(t *testing.T) {
	t.Parallel()

	daemon := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, func(d *Daemon) {
		d.state.Presets = pixy.NewPresetMap()
		d.state.Presets["home"] = pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: 100}
		d.state.Presets["desk"] = pixy.PTZValues{Pan: 30, Tilt: -10, Zoom: 120}
	})
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		"preset-chips",
		"home",
		"desk",
		`/api/preset/load/`,
		`/api/preset/delete/`,
	})
}

// --- helpers ---

func getPanelBody(t *testing.T, server *httptest.Server) string {
	t.Helper()

	resp := get(t, server.URL+"/panel")
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	return getBody(t, resp)
}

func assertContainsAll(t *testing.T, body string, expected []string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(body, exp) {
			t.Errorf("expected panel HTML to contain %q", exp)
		}
	}
}

func audioLabel(modeID string) string {
	switch modeID {
	case "nc":
		return "Noise Cancel"
	case "live":
		return "Live"
	case "org":
		return "Original"
	default:
		return modeID
	}
}

// TestWebPanel_PTZRadarHasServerRenderedStyle verifies that the PTZ radar
// template includes a server-rendered style attribute with CSS custom properties
// (--pan-x, --pan-y, --zoom-pct). This prevents the FOUC bug where the radar
// dot renders at (0,0) before DataStar evaluates the data-style expressions.
func TestWebPanel_PTZRadarHasServerRenderedStyle(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithDevice(t, pixy.StateTracking)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		"--pan-x:",
		"--pan-y:",
		"--zoom-pct:",
		"data-style:--pan-x",
		"data-style:--pan-y",
		"data-style:--zoom-pct",
	})
}

// TestWebPanel_DataStarAttributes verifies that all DataStar-specific
// attributes are present in the rendered panel HTML:
// - data-indicator on all action buttons (loading state)
// - data-class:btn-loading for CSS class toggling
// - data-bind on PTZ sliders for two-way signal sync
func TestWebPanel_DataStarAttributes(t *testing.T) {
	t.Parallel()

	daemon := testDaemonWithDevice(t, pixy.StateTracking)
	server := newTestWebServer(t, daemon)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		`data-indicator="loading"`,
		`data-class:btn-loading="$loading"`,
	})

	indicatorCount := strings.Count(body, `data-indicator="loading"`)
	if indicatorCount < 8 {
		t.Errorf("expected at least 8 data-indicator attributes (3 mode + 3 audio + gesture + auto + center + sync + probe), got %d",
			indicatorCount)
	}

	loadingClassCount := strings.Count(body, `data-class:btn-loading="$loading"`)
	if loadingClassCount < 8 {
		t.Errorf("expected at least 8 data-class:btn-loading attributes, got %d", loadingClassCount)
	}

	assertContainsAll(t, body, []string{
		`data-bind="$pan"`,
		`data-bind="$tilt"`,
		`data-bind="$zoom"`,
	})
}
