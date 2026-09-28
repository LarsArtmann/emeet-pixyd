//go:build linux

package main

import (
	"sync"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// ptzCacheTTL is the time-to-live for PTZ cache entries.
const ptzCacheTTL = 2 * time.Second

type lastFrameCache struct {
	mu   sync.RWMutex
	data []byte
}

func (f *lastFrameCache) Get() []byte {
	f.mu.RLock()
	data := make([]byte, len(f.data))
	copy(data, f.data)
	f.mu.RUnlock()

	return data
}

func (f *lastFrameCache) Set(data []byte) {
	f.mu.Lock()
	f.data = data
	f.mu.Unlock()
}

// ttlCache is a tiny generic TTL cache: Get reports whether the entry is
// still fresh, Set stores a value with a fresh deadline, Invalidate forces a
// miss. The zero value is ready to use.
type ttlCache[T any] struct {
	mu        sync.RWMutex
	value     T
	expiresAt time.Time
}

func (c *ttlCache[T]) Get() (T, bool) {
	now := time.Now()

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.value, now.Before(c.expiresAt)
}

func (c *ttlCache[T]) Set(value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value = value
	c.expiresAt = time.Now().Add(ttl)
}

func (c *ttlCache[T]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.expiresAt = time.Time{}
}

// ptzCache memoizes the last-synced PTZ position so panel renders don't hit
// v4l2-ctl on every keystroke.
type ptzCache = ttlCache[pixy.PTZValues]
