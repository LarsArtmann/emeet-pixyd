//go:build linux

package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	testWebAddr   = "127.0.0.1:0"
	audioModeLive = "live"
	audioModeOrg  = "org"
)

func newIntegrationDaemon(t *testing.T) *Daemon {
	t.Helper()

	return newTestDaemon(pixy.StatePrivacy, "", "", withConfig(t.TempDir()))
}

func newDaemonWithDevice(t *testing.T) *Daemon {
	t.Helper()

	return newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, withConfig(t.TempDir()))
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
			t.Parallel()

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

func assertCommandResponse(t *testing.T, cmd, substr, label string) {
	t.Helper()
	d := newDaemonWithDevice(t)
	resp := d.handleCommand(context.Background(), cmd)
	assertCommandContains(t, resp.String(), substr, label)
}

func testPTZEndpoint(t *testing.T, path, body string, expectedStatus int) {
	t.Helper()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := post(t, server.URL+path, "application/x-www-form-urlencoded", strings.NewReader(body))
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, expectedStatus)
}

func testWebEndpointReturnsOK(t *testing.T, endpoint string) {
	t.Helper()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := post(t, server.URL+endpoint, "", nil)
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusOK)
}

func testGETEndpoint503(t *testing.T, path string) {
	t.Helper()
	daemon := newIntegrationDaemon(t)
	server := newTestWebServer(t, daemon)

	resp := get(t, server.URL+path)
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusServiceUnavailable)
	assertResponseContains(t, resp, "no camera device", "503 body")
}
