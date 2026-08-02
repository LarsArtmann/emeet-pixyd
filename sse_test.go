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

	b.Broadcast(SSEEvent{Event: "test", Data: "hello"})

	select {
	case evt := <-ch:
		if evt.Event != "test" || evt.Data != "hello" {
			t.Errorf("received = %+v, want Event=test Data=hello", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast event")
	}
}

func TestBroadcaster_UnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	b := NewBroadcaster()
	ch := b.Subscribe()

	t.Cleanup(func() { b.Unsubscribe(ch) })

	b.Unsubscribe(ch)

	b.Broadcast(SSEEvent{Event: "noop"})

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
		b.Broadcast(SSEEvent{Data: "flood"})
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

	event := SSEEvent{Event: "refresh", Data: "{}"}

	b.ResetTimer()

	for range b.N {
		broadcaster.Broadcast(event)
	}

	b.StopTimer()
	broadcaster.Unsubscribe(ch)
	<-done
}
