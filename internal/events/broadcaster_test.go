package events

import (
	"sync"
	"testing"
	"time"
)

func TestBroadcaster_PublishToSubscriber(t *testing.T) {
	t.Parallel()

	b := New()
	ch, unsub := b.Subscribe()
	defer unsub()

	ev := Event{Type: TypeState, Body: []byte(`{"camera":"tracking"}`)}
	b.Publish(ev)

	select {
	case got := <-ch:
		if got.Type != TypeState {
			t.Errorf("Type = %q, want %q", got.Type, TypeState)
		}
		if string(got.Body) != string(ev.Body) {
			t.Errorf("Body = %q, want %q", got.Body, ev.Body)
		}
		if got.TS.IsZero() {
			t.Error("TS should be set by Publish when zero")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive event")
	}
}

func TestBroadcaster_FanOutToMultipleSubscribers(t *testing.T) {
	t.Parallel()

	b := New()
	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	if b.SubscriberCount() != 2 {
		t.Fatalf("SubscriberCount = %d, want 2", b.SubscriberCount())
	}

	b.Publish(Event{Type: TypePTZ, Body: []byte(`{"pan":10}`)})

	deadline := time.After(time.Second)
	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case <-ch:
		case <-deadline:
			t.Fatal("subscriber did not receive event")
		}
	}
}

func TestBroadcaster_DropsOnFullBuffer(t *testing.T) {
	t.Parallel()

	b := New()
	ch, unsub := b.Subscribe()
	defer unsub()

	// Fill buffer (subscriberBufferSize) without draining; subsequent publishes
	// must drop without blocking the publisher.
	for i := 0; i < subscriberBufferSize; i++ {
		b.Publish(Event{Type: TypeState})
	}

	done := make(chan struct{})
	go func() {
		b.Publish(Event{Type: TypeState})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked when buffer was full; should drop instead")
	}

	// Drain and assert we got exactly buffer-size events (the extra was dropped).
	count := 0
	timeout := time.After(50 * time.Millisecond)
drain:
	for {
		select {
		case <-ch:
			count++
		case <-timeout:
			break drain
		}
	}
	if count != subscriberBufferSize {
		t.Errorf("delivered = %d, want %d (extras should drop)", count, subscriberBufferSize)
	}
}

func TestBroadcaster_UnsubscribeRemovesSubscriber(t *testing.T) {
	t.Parallel()

	b := New()
	ch, unsub := b.Subscribe()
	if b.SubscriberCount() != 1 {
		t.Fatalf("SubscriberCount = %d, want 1", b.SubscriberCount())
	}

	unsub()
	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount after unsubscribe = %d, want 0", b.SubscriberCount())
	}

	// Channel must be closed.
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after unsubscribe")
	}

	// Idempotent: second unsubscribe must not panic or double-close.
	unsub()
}

func TestBroadcaster_PublishWithNoSubscribersDoesNotBlock(t *testing.T) {
	t.Parallel()

	b := New()
	done := make(chan struct{})
	go func() {
		b.Publish(Event{Type: TypeState})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked with no subscribers")
	}
}

func TestBroadcaster_ConcurrentPublishAndSubscribe(t *testing.T) {
	t.Parallel()

	b := New()
	const subs = 16
	const messages = 64

	var wg sync.WaitGroup
	for i := 0; i < subs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsub := b.Subscribe()
			defer unsub()
			deadline := time.After(2 * time.Second)
			for {
				select {
				case <-ch:
				case <-deadline:
					return
				}
			}
		}()
	}

	for i := 0; i < messages; i++ {
		b.Publish(Event{Type: TypeState})
	}

	wg.Wait()
}
