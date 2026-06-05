//go:build linux

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func newStreamDaemon(t *testing.T) (*Daemon, *webServer, *httptest.Server) {
	t.Helper()

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return d, webSrv, server
}

func getStream(t *testing.T, url string) *http.Response {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new GET request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}

	return resp
}

func TestHandleStream_SemaphoreFull(t *testing.T) {
	t.Parallel()

	d, _, server := newStreamDaemon(t)

	// Fill the semaphore
	d.streamSema <- struct{}{}
	defer func() { <-d.streamSema }()

	resp := getStream(t, server.URL+"/api/stream")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusServiceUnavailable)
}

func TestHandleStream_NoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateOffline, "", "")
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resp := getStream(t, server.URL+"/api/stream")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusServiceUnavailable)
}

func TestHandleStream_NoFFmpeg(t *testing.T) {
	t.Parallel()

	// Daemon with device but ffmpeg likely not in PATH during test
	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// This test passes if ffmpeg is not found (503) or if it is found
	// but the stream fails for another reason (we can't control that in CI)
	resp := getStream(t, server.URL+"/api/stream")
	defer resp.Body.Close() //nolint:errcheck

	// Either 503 (no ffmpeg) or 200 (ffmpeg available) is acceptable
	// We're mainly testing that the handler doesn't panic
	if resp.StatusCode != http.StatusServiceUnavailable && resp.StatusCode != http.StatusOK {
		t.Errorf("expected 503 or 200, got %d", resp.StatusCode)
	}
}

func TestHandleSnapshot_NoFrame(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resp := getStream(t, server.URL+"/api/snapshot")
	defer resp.Body.Close() //nolint:errcheck

	assertStatusCode(t, resp, http.StatusServiceUnavailable)
}

func TestHandleSnapshot_WithFrame(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	d.lastFrame.data = []byte{0xFF, 0xD8, 0x42, 0xFF, 0xD9}
	webSrv := &webServer{daemon: d}
	mux := newWebMux(webSrv)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resp := getStream(t, server.URL+"/api/snapshot")
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}
