//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// withUnsupportedHint seeds the daemon with an unsupported-device hint the
// way a probe that found a recognized-but-not-controllable EMEET webcam would.
func withUnsupportedHint(hint string) testDaemonOption {
	return func(d *Daemon) {
		d.mu.Lock()
		d.applyProbeResultLocked(probeResult{UnsupportedHint: hint})
		d.mu.Unlock()
	}
}

const testUnsupportedHint = "EMEET C960 (USB product ID 0x003f) is recognized but not controllable: " +
	"fixed EMEET webcams are fully driven by the standard Linux UVC driver, " +
	"and emeet-pixyd automates only the PIXY family's vendor features"

func TestUnsupportedHintShort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hint string
		want string
	}{
		{
			name: "recognized_device_short_is_first_clause",
			hint: testUnsupportedHint,
			want: "EMEET C960 (USB product ID 0x003f) is recognized but not controllable",
		},
		{
			name: "unknown_device_short_is_first_clause",
			hint: "unknown EMEET device (USB product ID 0x0abc) is not supported yet: set EMEET_PIXYD_EXTRA_PRODUCT_IDS",
			want: "unknown EMEET device (USB product ID 0x0abc) is not supported yet",
		},
		{
			name: "no_colon_returns_whole_hint",
			hint: "plain hint without colon",
			want: "plain hint without colon",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := unsupportedHintShort(tc.hint); got != tc.want {
				t.Errorf("unsupportedHintShort = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestUnsupportedHint_AbsentWhenUnset pins the additive contract (ADR
// 2026-09-28, Option B): with no hint, none of the surfaces carry the field —
// existing consumers keep parsing unchanged output.
func TestUnsupportedHint_AbsentWhenUnset(t *testing.T) {
	t.Parallel()

	d := testDaemonWithState(t, pixy.StateIdle, false)

	var waybarOut map[string]any
	if err := json.Unmarshal([]byte(d.waybarOutput(t.Context())), &waybarOut); err != nil {
		t.Fatalf("unmarshal waybar JSON: %v", err)
	}

	if _, present := waybarOut["unsupportedHint"]; present {
		t.Errorf("waybar unsupportedHint present without a hint, want omitted: %v", waybarOut["unsupportedHint"])
	}

	srv := newTestWebServer(t, d)

	var status struct {
		UnsupportedHint string `json:"unsupportedHint"`
	}
	getJSON(t, srv.URL+"/api/status", &status)

	if status.UnsupportedHint != "" {
		t.Errorf("/api/status unsupportedHint = %q without a hint, want empty", status.UnsupportedHint)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	body := getBody(t, doGet(t, ctx, srv.URL+"/panel")) //nolint:bodyclose // getBody closes the body after reading
	if strings.Contains(body, "unsupported-hint") {
		t.Error("panel renders the unsupported-hint note without a hint")
	}
}

// TestUnsupportedHint_SurfacesOnAllHumanSurfaces pins Option B: the Waybar
// tooltip gets the short clause, the additive JSON key carries the full hint,
// /api/status carries it additively, and the panel note shows the short
// clause inside #status-panel with the full text in the title tooltip.
func TestUnsupportedHint_SurfacesOnAllHumanSurfaces(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t,
		pixy.StateIdle,
		"",
		"",
		withConfig(t.TempDir()),
		withUnsupportedHint(testUnsupportedHint),
	)

	var out waybarJSON
	if err := json.Unmarshal([]byte(d.waybarOutput(t.Context())), &out); err != nil {
		t.Fatalf("unmarshal waybar JSON: %v", err)
	}

	if !strings.Contains(out.Tooltip, "is recognized but not controllable") {
		t.Errorf("tooltip = %q, want the short unsupported clause", out.Tooltip)
	}

	if out.UnsupportedHint != testUnsupportedHint {
		t.Errorf("waybar unsupportedHint = %.80s…, want the full hint", out.UnsupportedHint)
	}

	srv := newTestWebServer(t, d)

	var status struct {
		UnsupportedHint string `json:"unsupportedHint"`
	}
	getJSON(t, srv.URL+"/api/status", &status)

	if status.UnsupportedHint != testUnsupportedHint {
		t.Errorf("/api/status unsupportedHint = %.80s…, want the full hint", status.UnsupportedHint)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := doGet(t, ctx, srv.URL+"/panel")
	defer resp.Body.Close()

	panelBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read panel body: %v", readErr)
	}

	body := string(panelBody)

	short := unsupportedHintShort(testUnsupportedHint)
	if !strings.Contains(body, short) {
		t.Errorf("panel lacks the short unsupported clause %q", short)
	}

	if !strings.Contains(body, "unsupported-hint") {
		t.Error("panel lacks the unsupported-hint note element")
	}
}
