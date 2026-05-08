// Package events provides a small in-process pub/sub broadcaster used to
// notify connected web clients of daemon state changes via Server-Sent Events.
package events

import (
	"sync"
	"time"
)

// Type is the SSE event type. Clients subscribe to specific names via
// EventSource.addEventListener.
type Type string

const (
	// TypeState indicates the camera/audio/gesture/auto/in-call state changed.
	TypeState Type = "state"
	// TypePTZ indicates the pan/tilt/zoom values changed.
	TypePTZ Type = "ptz"
	// TypeOnline indicates the device became online or offline.
	TypeOnline Type = "online"
)

// Event is a single fan-out message. Body is opaque JSON the handler will
// inline into the SSE data: line.
type Event struct {
	Type Type
	Body []byte
	TS   time.Time
}

// subscriberBufferSize is the per-subscriber channel buffer. Slow subscribers
// drop events instead of stalling publishers; this is acceptable because the
// SSE handler always sends an initial state snapshot on connect, so a
// reconnecting client catches up regardless of dropped events.
const subscriberBufferSize = 8

// Broadcaster fans Events out to all current subscribers.
type Broadcaster struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

// New returns an empty broadcaster.
func New() *Broadcaster {
	return &Broadcaster{
		subs: make(map[chan Event]struct{}),
	}
}

// Subscribe returns a buffered receive-only channel and an unsubscribe func.
// The caller must invoke unsubscribe exactly once when finished; it is safe
// to call from any goroutine.
func (b *Broadcaster) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, subscriberBufferSize)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, ch)
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

// Publish delivers ev to every current subscriber. Subscribers whose buffer
// is full have ev dropped to keep publishers from blocking.
func (b *Broadcaster) Publish(ev Event) {
	if ev.TS.IsZero() {
		ev.TS = time.Now()
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- ev:
		default:
			// Slow subscriber; drop event. SSE clients receive a fresh
			// snapshot on reconnect, so this is recoverable.
		}
	}
}

// SubscriberCount returns the number of active subscribers. Mostly useful for
// tests and metrics.
func (b *Broadcaster) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}
