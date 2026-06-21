//go:build linux

package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func postPTZFormValue(
	t *testing.T,
	server *httptest.Server,
	path, value string,
) (*http.Response, string) {
	t.Helper()

	body := strings.NewReader("value=" + value)

	req, reqErr := http.NewRequestWithContext(
		context.Background(), http.MethodPost, server.URL+path, body,
	)
	if reqErr != nil {
		t.Fatalf("create request: %v", reqErr)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		t.Fatalf("POST %s: %v", path, respErr)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, _ := io.ReadAll(resp.Body)

	return resp, string(respBody)
}

func assertSingleV4L2Call(t *testing.T, calls []v4l2Call) v4l2Call {
	t.Helper()

	if len(calls) != 1 {
		t.Fatalf("expected 1 v4l2 call, got %d", len(calls))
	}

	return calls[0]
}

func assertV4L2Call(t *testing.T, calls []v4l2Call, wantVal string) {
	t.Helper()

	call := assertSingleV4L2Call(t, calls)

	if call.val != wantVal {
		t.Errorf("v4l2 call val = %s, want %s", call.val, wantVal)
	}
}

func assertV4L2CallFull(t *testing.T, calls []v4l2Call, wantDev, wantCtrl, wantVal string) {
	t.Helper()

	call := assertSingleV4L2Call(t, calls)

	if call.dev != wantDev {
		t.Errorf("v4l2 dev = %q, want %q", call.dev, wantDev)
	}

	if call.ctrl != wantCtrl {
		t.Errorf("v4l2 ctrl = %q, want %q", call.ctrl, wantCtrl)
	}

	if call.val != wantVal {
		t.Errorf("v4l2 val = %q, want %q", call.val, wantVal)
	}
}

func assertPTZCacheUpdated(t *testing.T, d *Daemon, axis pixy.Axis, want int) {
	t.Helper()

	cached, valid := d.ptzCache.Get()
	if !valid {
		t.Fatal("PTZ cache should be valid (updated) after successful set, not invalidated")
	}

	if got := cached.Get(axis); got != want {
		t.Errorf("PTZ cache %s = %d, want %d", axis, got, want)
	}
}

// withSeededPTZCache returns a testDaemonOption that primes the PTZ cache
// with a known starting state (pan=0, tilt=0, zoom=100) so subsequent
// requests can be observed against a non-zero baseline.
func withSeededPTZCache() testDaemonOption {
	return func(d *Daemon) {
		d.ptzCache.Set(pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault}, ptzCacheTTL)
	}
}

func TestBehavior_PTZClampingAndMultiplier(t *testing.T) {
	t.Parallel()

	d, v4l2Calls := newPTZCaptureDaemon()

	// When pan is set beyond the maximum (200 → clamp to PanMax)
	resp := d.handlePTZCommand(context.Background(), []string{string(pixy.AxisPan), "200"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, strconv.Itoa(pixy.PanRange.Max*v4l2UnitsPerDegree))

	// When tilt is set beyond minimum (-100 → clamp to TiltMin)
	*v4l2Calls = nil
	resp = d.handlePTZCommand(context.Background(), []string{"tilt", "-100"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, strconv.Itoa(pixy.TiltRange.Min*v4l2UnitsPerDegree))

	// When zoom is set beyond maximum (500 → clamp to ZoomMax, no multiplier)
	*v4l2Calls = nil
	resp = d.handlePTZCommand(context.Background(), []string{string(pixy.AxisZoom), "500"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, strconv.Itoa(pixy.ZoomRange.Max))
}

// TestBehavior_PTZAbsoluteNegativeTilt proves that bare negative values are
// treated as ABSOLUTE positions, not relative offsets. With parsePTZ seeded
// to return tilt=10, an absolute "tilt -90" must produce V4L2 -324000
// (=-90*3600), not -80*3600 which is what relative mode would yield.
func TestBehavior_PTZAbsoluteNegativeTilt(t *testing.T) {
	t.Parallel()

	d, v4l2Calls := newPTZCaptureDaemon(func(d *Daemon) {
		d.deps.parsePTZ = func(_ context.Context, _ string) pixy.PTZValues {
			return pixy.PTZValues{Tilt: 10} // non-zero baseline
		}
	})

	resp := d.handlePTZCommand(context.Background(), []string{"tilt", "-90"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, strconv.Itoa(pixy.TiltRange.Min*v4l2UnitsPerDegree))
}

// TestBehavior_PTZRelativeMath proves that "rel" prefix triggers relative
// mode: current position + relative delta. With parsePTZ seeded to pan=50,
// "pan rel+10" must produce V4L2 216000 (=(50+10)*3600).
func TestBehavior_PTZRelativeMath(t *testing.T) {
	t.Parallel()

	d, v4l2Calls := newPTZCaptureDaemon(func(d *Daemon) {
		d.deps.parsePTZ = func(_ context.Context, _ string) pixy.PTZValues {
			return pixy.PTZValues{Pan: 50}
		}
	})

	resp := d.handlePTZCommand(context.Background(), []string{"pan", "rel+10"})
	notError(t, resp)
	assertV4L2Call(t, *v4l2Calls, strconv.Itoa((50+10)*v4l2UnitsPerDegree))
}

func TestBehavior_PTZWebSliderReflectsUserInput(t *testing.T) {
	t.Parallel()

	// Given a daemon with a device and a web server (cache has stale pan=0)
	d := newTestDaemon(
		pixy.StateTracking,
		"/dev/video0",
		"/dev/hidraw7",
		withNoopV4L2(),
		withSeededPTZCache(),
	)
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)

	server := httptest.NewServer(mux)
	defer server.Close()

	// When user sets pan to 50 via the web interface
	resp, html := postPTZFormValue(t, server, "/api/ptz/pan", "50")
	resp.Body.Close() //nolint:errcheck

	// Then the response slider contains the user's value (50), not the stale cache value (0)
	assertCommandContains(t, html, `value="50"`, "slider response")

	if strings.Contains(html, `value="0"`) {
		t.Error("slider response should NOT contain stale cache value 0")
	}

	// And no success toast is shown (PTZ toasts suppressed to avoid slider spam)
	if strings.Contains(html, "Pan set to 50") {
		t.Error("PTZ success toast should be suppressed")
	}

	// And the PTZ cache is updated with the set value
	assertPTZCacheUpdated(t, d, pixy.AxisPan, 50)
}

func TestBehavior_PTZWebSliderShowsErrorOnFailure(t *testing.T) {
	t.Parallel()

	// Given a daemon with no device
	d := newTestDaemon(pixy.StateOffline, "", "", withNoopV4L2())
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)

	server := httptest.NewServer(mux)
	defer server.Close()

	// When user tries to set pan
	resp, html := postPTZFormValue(t, server, "/api/ptz/pan", "50")
	defer resp.Body.Close() //nolint:errcheck

	assertHTTPStatusOK(t, resp)

	// Then an error toast is shown
	assertCommandContains(t, html, "toast-error", "response")
	assertCommandContains(t, html, "error:", "response")
}

func TestBehavior_PTZWebReachesV4L2Camera(t *testing.T) {
	t.Parallel()

	tests := []struct {
		axis       string
		value      string
		wantCtrl   string
		wantVal    string
		wantSlider string
	}{
		{"pan", "45", "pan_absolute", strconv.Itoa(45 * v4l2UnitsPerDegree), `value="45"`},
		{"tilt", "-30", "tilt_absolute", strconv.Itoa(-30 * v4l2UnitsPerDegree), `value="-30"`},
		{"zoom", "125", "zoom_absolute", "125", `value="125"`},
	}

	for _, tc := range tests {
		t.Run(tc.axis, func(t *testing.T) {
			t.Parallel()

			var v4l2Calls []v4l2Call

			d := newTestDaemon(
				pixy.StateTracking,
				"/dev/video0",
				"/dev/hidraw7",
				withCaptureV4L2(&v4l2Calls),
				withSeededPTZCache(),
				withNoopParsePTZ(),
			)
			webSrv := &webServer{daemon: d}
			mux := newWebMux(webSrv)

			server := httptest.NewServer(mux)
			defer server.Close()

			// When user sets the axis value via the web slider
			resp, html := postPTZFormValue(t, server, "/api/ptz/"+tc.axis, tc.value)
			resp.Body.Close() //nolint:errcheck

			// Then the HTTP response is OK and contains the updated slider
			assertHTTPStatusOK(t, resp)
			assertCommandContains(t, html, tc.wantSlider, "slider response")

			// And v4l2-ctl was called with the correct control and scaled value
			assertV4L2CallFull(t, v4l2Calls, "/dev/video0", tc.wantCtrl, tc.wantVal)

			// And the PTZ cache is updated with the set value
			wantVal, _ := strconv.Atoi(tc.value)
			assertPTZCacheUpdated(t, d, pixy.Axis(tc.axis), wantVal)
		})
	}
}
