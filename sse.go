//go:build linux

package main

import (
	"sync"
)

const sseSubscriberBuffer = 8

// SSEEvent represents a broadcast notification that triggers a client refresh.
// The payload is not consumed directly; the /api/events handler re-renders the
// full panel from current daemon state on every received event.
type SSEEvent struct {
	Event string
	Data  string
}

// Broadcaster distributes refresh notifications to all subscribed SSE clients
// via thread-safe, non-blocking fan-out. Slow clients drop events without
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
