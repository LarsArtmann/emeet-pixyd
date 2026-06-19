//go:build linux

package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func readSSELine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read sse line: %v", err)
	}

	return line
}

// assertSSELine reads the next line from the SSE reader and asserts it equals want.
func assertSSELine(t *testing.T, reader *bufio.Reader, want string) {
	t.Helper()

	if got := readSSELine(t, reader); got != want {
		t.Errorf("sse line = %q, want %q", got, want)
	}
}

// openSSEStream opens a GET connection to /api/events on the given server
// and registers a cleanup hook to close the body.
func openSSEStream(t *testing.T, server *httptest.Server) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func TestSSEEndpoint_SendsConnectedEvent(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()
	server := newTestWebServer(t, d)

	resp := openSSEStream(t, server) //nolint:bodyclose // closed via t.Cleanup in openSSEStream

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	wantContentType := "text/event-stream"
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, wantContentType) {
		t.Errorf("Content-Type = %q, want %q", got, wantContentType)
	}

	reader := bufio.NewReader(resp.Body)

	assertSSELine(t, reader, "event: connected\n")
	assertSSELine(t, reader, "data: {}\n")
}

func TestSSEEndpoint_BroadcastsRefresh(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()
	server := newTestWebServer(t, d)

	resp := openSSEStream(t, server) //nolint:bodyclose // closed via t.Cleanup in openSSEStream
	reader := bufio.NewReader(resp.Body)

	// Consume connected event.
	_ = readSSELine(t, reader)
	_ = readSSELine(t, reader)
	_ = readSSELine(t, reader) // blank line

	d.broadcastStateChanged()

	done := make(chan struct{})

	var refreshLine string

	go func() {
		defer close(done)

		refreshLine = readSSELine(t, reader)
	}()

	select {
	case <-done:
		if refreshLine != "event: refresh\n" {
			t.Errorf("refresh event = %q, want event: refresh", refreshLine)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for refresh event")
	}
}
