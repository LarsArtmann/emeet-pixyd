//go:build linux

package main

import (
	"bufio"
	"bytes"
	"io"
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

func TestWriteSSEEvent_DataOnly(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := writeSSEEvent(&buf, SSEEvent{Data: `{"key":"value"}`})
	if err != nil {
		t.Fatalf("writeSSEEvent: %v", err)
	}

	got := buf.String()

	want := `data: {"key":"value"}` + "\n\n"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestWriteSSEEvent_FullEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := writeSSEEvent(&buf, SSEEvent{
		Event: "refresh",
		Data:  "ok",
		ID:    "42",
		Retry: 5000,
	})
	if err != nil {
		t.Fatalf("writeSSEEvent: %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, "event: refresh\n") {
		t.Errorf("missing event line in %q", got)
	}

	if !strings.Contains(got, "data: ok\n") {
		t.Errorf("missing data line in %q", got)
	}

	if !strings.Contains(got, "id: 42\n") {
		t.Errorf("missing id line in %q", got)
	}

	if !strings.Contains(got, "retry: 5000\n") {
		t.Errorf("missing retry line in %q", got)
	}
}

func TestWriteSSEEvent_MultilineData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := writeSSEEvent(&buf, SSEEvent{Data: "line1\nline2\nline3"})
	if err != nil {
		t.Fatalf("writeSSEEvent: %v", err)
	}

	got := buf.String()

	want := "data: line1\ndata: line2\ndata: line3\n\n"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestWriteSSEEvent_EmptyData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := writeSSEEvent(&buf, SSEEvent{Event: "ping"})
	if err != nil {
		t.Fatalf("writeSSEEvent: %v", err)
	}

	got := buf.String()

	want := "event: ping\ndata: \n\n"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestWriteSSEEvent_NoCRLFInjection(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	// Attempt CRLF injection — data with \r\n must not create a new SSE event field.
	// Every line is prefixed with "data: " so "event:" can never appear as a field.
	_ = writeSSEEvent(&buf, SSEEvent{Data: "evil\r\nevent: hacked"})

	got := buf.String()
	for line := range strings.SplitSeq(got, "\n") {
		if strings.HasPrefix(line, "event:") {
			t.Errorf("CRLF injection: line starts with 'event:' — %q in output %q", line, got)
		}
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

func TestSplitSSELines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want []string
	}{
		{name: "empty", s: "", want: []string{""}},
		{name: "single line", s: "hello", want: []string{"hello"}},
		{name: "two lines", s: "a\nb", want: []string{"a", "b"}},
		{name: "trailing newline", s: "a\n", want: []string{"a"}},
		{name: "CRLF preserved", s: "a\r\nb", want: []string{"a\r", "b"}},
		{name: "only newlines", s: "\n\n", want: []string{"", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := splitSSELines(tt.s)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (%q → %v vs %v)", len(got), len(tt.want), tt.s, got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func FuzzWriteSSEEvent(f *testing.F) {
	f.Add("refresh", `{"camera":"tracking"}`)
	f.Add("", "")
	f.Add("event", "multi\nline\ndata")
	f.Add("a", "evil\r\nevent: hacked")

	f.Fuzz(func(t *testing.T, event, data string) {
		var buf bytes.Buffer

		err := writeSSEEvent(&buf, SSEEvent{Event: event, Data: data})
		if err != nil {
			t.Fatalf("writeSSEEvent: %v", err)
		}

		output := buf.String()

		if !strings.HasSuffix(output, "\n\n") {
			t.Errorf("output must end with \\n\\n")
		}
	})
}

func BenchmarkWriteSSEEvent(b *testing.B) {
	event := SSEEvent{Event: "refresh", Data: `{"camera":"tracking","audio":"nc"}`}

	for range b.N {
		_ = writeSSEEvent(io.Discard, event)
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
