//go:build linux

package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func assertInCall(t *testing.T, d *Daemon, want bool) {
	t.Helper()

	if got := readState(d, func(s pixy.State) bool { return s.InCall }); got != want {
		t.Errorf("expected InCall=%v, got %v", want, got)
	}
}

func assertHTTPStatusOK(t *testing.T, resp *http.Response) {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func assertNotifyContains(t *testing.T, messages []string, substr string) {
	t.Helper()

	if len(messages) == 0 {
		t.Fatalf("expected notification containing %q, but no notifications", substr)
	}

	if !strings.Contains(messages[0], substr) {
		t.Errorf("notification should mention %s, got: %s", substr, messages[0])
	}
}

func assertCameraState(t *testing.T, d *Daemon, expected pixy.CameraState) {
	t.Helper()

	camera := readCameraState(d)
	if camera != expected {
		t.Errorf("expected camera state %s, got %s", expected, camera)
	}
}

func assertErrorPrefix(t *testing.T, result string) {
	t.Helper()

	if !strings.HasPrefix(result, "error:") {
		t.Errorf("expected error prefix, got: %s", result)
	}
}

func assertStatusContains(t *testing.T, result, substr, msg string) {
	t.Helper()

	if !strings.Contains(result, substr) {
		t.Errorf("%s: expected %q in status, got: %s", msg, substr, result)
	}
}

func assertCommandContains(t *testing.T, resp, substr, label string) {
	t.Helper()

	if !strings.Contains(resp, substr) {
		t.Errorf("expected %q in %s, got: %s", substr, label, resp)
	}
}

func assertStatusPrefix(t *testing.T, result, prefix, msg string) {
	t.Helper()

	if !strings.HasPrefix(result, prefix) {
		t.Errorf("%s: expected prefix %q, got: %s", msg, prefix, result)
	}
}

func assertAutoMode(t *testing.T, d *Daemon, expected pixy.AutoMode) {
	t.Helper()

	if d.state.AutoMode != expected {
		t.Errorf("expected auto mode=%v, got %v", expected, d.state.AutoMode)
	}
}

func assertGesture(t *testing.T, resp hidResponse, expected bool) {
	t.Helper()

	if !resp.Got || resp.Gesture != expected {
		t.Errorf("expected gesture=%v, got Got=%v Gesture=%v", expected, resp.Got, resp.Gesture)
	}
}

func assertParsedField(t *testing.T, parsed map[string]string, field string) {
	t.Helper()

	if _, ok := parsed[field]; !ok {
		t.Errorf("waybar output missing '%s' field", field)
	}
}

func assertTrackingIdle(t *testing.T, tracking pixy.CameraState) {
	t.Helper()

	if tracking != pixy.StateIdle {
		t.Errorf("Tracking = %q, want idle", tracking)
	}
}
