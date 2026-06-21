//go:build linux

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"sync"
)

const sseSubscriberBuffer = 8

// SSEEvent represents a single Server-Sent Events message.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// Broadcaster distributes SSE events to all subscribed clients via
// thread-safe, non-blocking fan-out. Slow clients drop events without
// stalling the broadcaster.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan SSEEvent]struct{}
}

// NewBroadcaster creates a broadcaster with no subscribers.
//
//nolint:exhaustruct
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{subscribers: make(map[chan SSEEvent]struct{})}
}

// Subscribe returns a channel that receives all broadcast events.
// The channel has a buffer of sseSubscriberBuffer; slow consumers may miss messages.
// Call Unsubscribe when the client disconnects to prevent leaks.
func (b *Broadcaster) Subscribe() <-chan SSEEvent {
	ch := make(chan SSEEvent, sseSubscriberBuffer)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (b *Broadcaster) Unsubscribe(ch <-chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for sender := range b.subscribers {
		if sender == ch {
			delete(b.subscribers, sender)
			close(sender)

			return
		}
	}
}

// Broadcast sends an event to all subscribers. Slow subscribers with
// full buffers have the event dropped — the broadcaster never blocks.
func (b *Broadcaster) Broadcast(event SSEEvent) {
	b.mu.RLock()

	snapshot := make([]chan SSEEvent, 0, len(b.subscribers))
	for ch := range b.subscribers {
		snapshot = append(snapshot, ch)
	}

	b.mu.RUnlock()

	for _, ch := range snapshot {
		select {
		case ch <- event:
		default:
		}
	}
}

// sseStream manages a single Server-Sent Events connection.
//
//nolint:containedctx // SSE streams inherently need to track request context for disconnect detection
type sseStream struct {
	w   io.Writer
	fw  http.Flusher
	ctx context.Context
}

// newSSEStream sets SSE headers and returns a stream for one client.
func newSSEStream(w http.ResponseWriter, r *http.Request) *sseStream {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	fw, _ := w.(http.Flusher)

	return &sseStream{w: w, fw: fw, ctx: r.Context()}
}

// Send writes an SSE event to the stream and flushes.
func (s *sseStream) Send(event SSEEvent) error {
	err := writeSSEEvent(s.w, event)
	if err != nil {
		return fmt.Errorf("send sse event: %w", err)
	}

	if s.fw != nil {
		s.fw.Flush()
	}

	return nil
}

// Context returns the stream's context, cancelled on client disconnect.
func (s *sseStream) Context() context.Context {
	return s.ctx
}

// Close flushes any buffered data.
func (s *sseStream) Close() {
	if s.fw != nil {
		s.fw.Flush()
	}
}

func writeSSEEvent(w io.Writer, event SSEEvent) error {
	var buf []byte

	if event.Event != "" {
		buf = append(buf, "event: "...)
		buf = append(buf, event.Event...)
		buf = append(buf, '\n')
	}

	for _, line := range splitSSELines(event.Data) {
		buf = append(buf, "data: "...)
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if event.ID != "" {
		buf = append(buf, "id: "...)
		buf = append(buf, event.ID...)
		buf = append(buf, '\n')
	}

	if event.Retry > 0 {
		buf = append(buf, "retry: "...)
		buf = append(buf, strconv.Itoa(event.Retry)...)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	_, err := w.Write(buf)

	return err //nolint:wrapcheck // io.Writer.Write is an interface method
}

func splitSSELines(s string) []string {
	if s == "" || !contains(s, '\n') {
		return []string{s}
	}

	var lines []string

	start := 0

	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

func contains(s string, c byte) bool {
	return slices.Index([]byte(s), c) >= 0
}
