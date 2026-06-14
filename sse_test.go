//go:build linux

package main

import (
	"bufio"
	"context"
	"net/http"
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

func TestSSEEndpoint_SendsConnectedEvent(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()
	server := newTestWebServer(t, d)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	wantContentType := "text/event-stream"
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, wantContentType) {
		t.Errorf("Content-Type = %q, want %q", got, wantContentType)
	}

	reader := bufio.NewReader(resp.Body)

	if got := readSSELine(t, reader); got != "event: connected\n" {
		t.Errorf("first event = %q, want event: connected", got)
	}

	if got := readSSELine(t, reader); got != "data: {}\n" {
		t.Errorf("first data = %q, want data: {}", got)
	}
}

func TestSSEEndpoint_BroadcastsRefresh(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()
	server := newTestWebServer(t, d)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}

	t.Cleanup(func() { _ = resp.Body.Close() })

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
