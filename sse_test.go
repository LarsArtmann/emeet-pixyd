//go:build linux

package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func readSSELine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read sse line: %v", err)
	}

	return line
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

func TestSSEEndpoint_SendsPatchElementsOnConnect(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)
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

	// DataStar sends: event: datastar-patch-elements\ndata: elements <html>\n\n
	eventLine := readSSELine(t, reader)
	if !strings.Contains(eventLine, "datastar-patch-elements") {
		t.Errorf("event line = %q, want datastar-patch-elements", eventLine)
	}

	// Verify the data lines follow the expected DataStar wire format:
	// Each data line starts with "data: elements " followed by HTML.
	dataLine := readSSELine(t, reader)
	if !strings.HasPrefix(dataLine, "data: elements ") {
		t.Errorf("data line prefix = %q, want 'data: elements '", dataLine[:min(len(dataLine), 20)])
	}

	// The data should contain actual panel HTML.
	if !strings.Contains(dataLine, "status-panel") {
		t.Error("data line should contain status-panel HTML")
	}
}

func TestSSEEndpoint_PTZReturnsPatchSignals(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(
		t,
		pixy.StateTracking,
		"/dev/video0",
		"/dev/hidraw7",
		withNoopV4L2(),
		withSeededPTZCache(),
	)
	server := newTestWebServer(t, d)

	// POST to PTZ endpoint with valid signals — should get patch-signals, not patch-elements.
	body := strings.NewReader(`{"pan":50}`)

	resp, err := server.Client().Post(server.URL+"/api/ptz/pan", "application/json", body)
	if err != nil {
		t.Fatalf("POST /api/ptz/pan: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	reader := bufio.NewReader(resp.Body)

	eventLine := readSSELine(t, reader)
	if !strings.Contains(eventLine, "datastar-patch-signals") {
		t.Errorf("event line = %q, want datastar-patch-signals", eventLine)
	}

	// The data line should contain JSON signals, not HTML.
	dataLine := readSSELine(t, reader)
	if !strings.Contains(dataLine, `"pan":50`) {
		t.Errorf("data line = %q, want pan signal", dataLine)
	}

	// Should NOT contain panel HTML (that's the old behavior).
	if strings.Contains(dataLine, "status-panel") {
		t.Error("PTZ success response should use signals, not panel HTML")
	}
}

func TestSSEEndpoint_BroadcastsPatchOnStateChange(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)
	server := newTestWebServer(t, d)

	resp := openSSEStream(t, server) //nolint:bodyclose // closed via t.Cleanup in openSSEStream
	reader := bufio.NewReader(resp.Body)

	// Consume the first patch event line (ignore data lines).
	firstEvent := readSSELine(t, reader)
	if !strings.Contains(firstEvent, "datastar-patch-elements") {
		t.Fatalf("first event = %q, want datastar-patch-elements", firstEvent)
	}

	// Allow the handler goroutine to subscribe to the broadcaster
	// after sending the initial patch.
	time.Sleep(100 * time.Millisecond)

	d.broadcastStateChanged()

	done := make(chan struct{})

	go func() {
		defer close(done)

		// Scan lines until the second datastar-patch-elements event arrives.
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			if strings.Contains(line, "datastar-patch-elements") {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for broadcast patch event")
	}
}

func TestBroadcaster_SubscribeBroadcastReceive(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()
	ch := b.Subscribe()

	t.Cleanup(func() { b.Unsubscribe(ch) })

	b.Broadcast()

	select {
	case _, ok := <-ch:
		if !ok {
			t.Error("channel should not be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast notification")
	}
}

func TestBroadcaster_UnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()
	ch := b.Subscribe()

	t.Cleanup(func() { b.Unsubscribe(ch) })

	b.Unsubscribe(ch)

	b.Broadcast()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: channel not closed after unsubscribe")
	}
}

func TestBroadcaster_NonBlockingDropOnFullBuffer(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()
	ch := b.Subscribe()

	t.Cleanup(func() { b.Unsubscribe(ch) })

	// Flood past the buffer capacity — must not block.
	for range sseSubscriberBuffer + 10 {
		b.Broadcast()
	}

	// Drain at least one event — proves the broadcaster survived.
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout: no event received after flood")
	}
}

func BenchmarkBroadcasterBroadcast(b *testing.B) {
	broadcaster := NewBroadcaster()
	ch := broadcaster.Subscribe()

	b.Cleanup(func() { broadcaster.Unsubscribe(ch) })

	done := make(chan struct{})

	go func() {
		for range ch {
		}

		close(done)
	}()

	for range b.N {
		broadcaster.Broadcast()
	}

	b.StopTimer()
	broadcaster.Unsubscribe(ch)
	<-done
}
