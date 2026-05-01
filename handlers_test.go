//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func requireGaugeValue(t *testing.T, name string, want float64, attrs ...attribute.KeyValue) {
	t.Helper()
	registerMetrics()

	var rm metricdata.ResourceMetrics
	if err := promExporter.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			gauge, ok := m.Data.(metricdata.Gauge[float64])
			if !ok {
				t.Fatalf("%s: not a float64 gauge", name)
			}
			for _, dp := range gauge.DataPoints {
				if matchAttrs(dp.Attributes, attrs) {
					if dp.Value != want {
						t.Errorf("%s = %v, want %v", name, dp.Value, want)
					}
					return
				}
			}
		}
	}

	t.Errorf("gauge %s with attrs %v not found", name, attrs)
}

func matchAttrs(set attribute.Set, wanted []attribute.KeyValue) bool {
	if len(wanted) == 0 {
		return set.Len() == 0
	}
	for _, w := range wanted {
		v, ok := set.Value(w.Key)
		if !ok || v.AsString() != w.Value.AsString() {
			return false
		}
	}
	return true
}

func assertJPEGBytes(t *testing.T, frame, expected []byte) {
	t.Helper()
	if string(frame) != string(expected) {
		t.Errorf("expected %x, got %x", expected, frame)
	}
}

func TestExtractJPEGFrame_MinimalFrame(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frame) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(frame))
	}
	assertJPEGMarkers(t, frame)
}

func TestExtractJPEGFrame_FrameWithPayload(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0x42, 0x43, 0x44, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertJPEGBytes(t, frame, data)
}

func TestExtractJPEGFrame_GarbageBeforeSOI(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0x01, 0x02, 0xFF, 0xD8, 0xAA, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte{0xFF, 0xD8, 0xAA, 0xFF, 0xD9}
	assertJPEGBytes(t, frame, expected)
}

func TestExtractJPEGFrame_DoubleFFBeforeD8(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xFF, 0xD8, 0x42, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte{0xFF, 0xD8, 0x42, 0xFF, 0xD9}
	assertJPEGBytes(t, frame, expected)
}

func TestExtractJPEGFrame_EmptyInput(t *testing.T) {
	t.Parallel()

	br := bufio.NewReader(bytes.NewReader(nil))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestExtractJPEGFrame_NoEOI(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0x42, 0x43}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error when no EOI found")
	}
}

func TestExtractJPEGFrame_NoSOI(t *testing.T) {
	t.Parallel()

	data := []byte{0x42, 0x43, 0x44}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error when no SOI found")
	}
}

func TestExtractJPEGFrame_FFInsidePayload(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0xFF, 0x00, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertJPEGMarkers(t, frame)
}

func TestExtractJPEGFrame_BufferReset(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := bytes.NewBuffer(make([]byte, maxStreamBufferSize+100))

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertJPEGBytes(t, frame, data)
}

func TestExtractJPEGFrame_FFThenEOF(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error for truncated input")
	}
}

func TestClampInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v, lo, hi, want int
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{0, -170, 170, 0},
		{-200, -170, 170, -170},
		{200, -170, 170, 170},
		{100, 100, 400, 100},
		{250, 100, 400, 250},
		{500, 100, 400, 400},
	}

	for _, tc := range tests {
		got := clampInt(tc.v, tc.lo, tc.hi)
		if got != tc.want {
			t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestPTZLimits(t *testing.T) {
	t.Parallel()

	lo, hi := ptzLimits(axisPan)
	if lo != panMin || hi != panMax {
		t.Errorf("pan limits: got %d,%d, want %d,%d", lo, hi, panMin, panMax)
	}

	lo, hi = ptzLimits(axisTilt)
	if lo != tiltMin || hi != tiltMax {
		t.Errorf("tilt limits: got %d,%d, want %d,%d", lo, hi, tiltMin, tiltMax)
	}

	lo, hi = ptzLimits(axisZoom)
	if lo != zoomMin || hi != zoomMax {
		t.Errorf("zoom limits: got %d,%d, want %d,%d", lo, hi, zoomMin, zoomMax)
	}

	lo, hi = ptzLimits("unknown")
	if lo != 0 || hi != 0 {
		t.Errorf("unknown axis: got %d,%d, want 0,0", lo, hi)
	}
}

func TestPTZAxisValid(t *testing.T) {
	t.Parallel()

	if !ptzAxisValid(axisPan) {
		t.Error("pan should be valid")
	}
	if !ptzAxisValid(axisTilt) {
		t.Error("tilt should be valid")
	}
	if !ptzAxisValid(axisZoom) {
		t.Error("zoom should be valid")
	}
	if ptzAxisValid("unknown") {
		t.Error("unknown should be invalid")
	}
}

func TestFormatLastSynced(t *testing.T) {
	t.Parallel()

	if result := formatLastSynced(time.Time{}); result != "" {
		t.Errorf("zero time should return empty, got %q", result)
	}

	if result := formatLastSynced(time.Now()); result != "just now" {
		t.Errorf("recent time should return 'just now', got %q", result)
	}

	if result := formatLastSynced(time.Now().Add(-2 * time.Minute)); result != "2m ago" {
		t.Errorf("2 min ago should return '2m ago', got %q", result)
	}

	old := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	result := formatLastSynced(old)
	if len(result) != 5 {
		t.Errorf("old time should return HH:MM format, got %q", result)
	}
}

func TestSecurityMiddleware(t *testing.T) {
	t.Parallel()

	called := false
	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})
	handler := securityMiddleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}

	headers := []struct {
		key, want string
	}{
		{"Referrer-Policy", "no-referrer"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{
			"Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
		},
	}
	for _, h := range headers {
		got := rec.Header().Get(h.key)
		if got != h.want {
			t.Errorf("%s = %q, want %q", h.key, got, h.want)
		}
	}
}

func assertJPEGMarkers(t *testing.T, frame []byte) {
	t.Helper()
	if frame[0] != 0xFF || frame[1] != 0xD8 {
		t.Errorf("missing SOI")
	}
	if frame[len(frame)-2] != 0xFF || frame[len(frame)-1] != 0xD9 {
		t.Errorf("missing EOI")
	}
}

func runRequestIDMiddleware(t *testing.T, req *http.Request) string {
	t.Helper()
	var capturedID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		capturedID = w.Header().Get("X-Request-ID")
	})
	h := requestIDMiddleware(inner)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return capturedID
}

func TestRequestIDMiddleware_Generated(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	capturedID := runRequestIDMiddleware(t, req)

	if capturedID == "" {
		t.Error("X-Request-ID should be generated when not provided")
	}
	if len(capturedID) != 8 {
		t.Errorf("generated ID length = %d, want 8", len(capturedID))
	}
}

func TestRequestIDMiddleware_Passthrough(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("X-Request-ID", "abcd1234")
	capturedID := runRequestIDMiddleware(t, req)

	if capturedID != "abcd1234" {
		t.Errorf("X-Request-ID = %q, want %q", capturedID, "abcd1234")
	}
}

func TestCachingFS(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	cfs := cachingFS{handler: inner}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/static/test.js", nil)
	rec := httptest.NewRecorder()
	cfs.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc != "public, max-age=604800" {
		t.Errorf("Cache-Control = %q, want public max-age 7d", cc)
	}
	xcto := rec.Header().Get("X-Content-Type-Options")
	if xcto != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", xcto)
	}
}

func TestUpdateMetrics(t *testing.T) {
	t.Parallel()
	registerMetrics()

	state := pixy.State{
		Camera:   pixy.StateTracking,
		Audio:    pixy.AudioNC,
		InCall:   true,
		AutoMode: false,
	}

	updateMetrics(state)

	requireGaugeValue(t, "emeet_pixyd_in_call", 1)
	requireGaugeValue(t, "emeet_pixyd_auto_mode", 0)
	for _, s := range []pixy.CameraState{pixy.StatePrivacy, pixy.StateTracking, pixy.StateIdle} {
		want := 0.0
		if state.Camera == s {
			want = 1.0
		}
		requireGaugeValue(t, "emeet_pixyd_camera_state", want, attribute.String("state", string(s)))
	}

	updateMetrics(pixy.State{
		Camera:   pixy.StatePrivacy,
		InCall:   false,
		AutoMode: true,
	})

	requireGaugeValue(t, "emeet_pixyd_in_call", 0)
	requireGaugeValue(t, "emeet_pixyd_auto_mode", 1)
}
