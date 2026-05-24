//go:build linux

package main

import (
	"sync"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

type lastFrameCache struct {
	mu   sync.RWMutex
	data []byte
}

func (f *lastFrameCache) Get() []byte {
	f.mu.RLock()
	data := f.data
	f.mu.RUnlock()
	return data
}

func (f *lastFrameCache) Set(data []byte) {
	f.mu.Lock()
	f.data = data
	f.mu.Unlock()
}

type ptzCache struct {
	mu        sync.RWMutex
	values    pixy.PTZValues
	expiresAt time.Time
}

func (c *ptzCache) Get() (pixy.PTZValues, bool) {
	now := time.Now()
	c.mu.RLock()
	valid := now.Before(c.expiresAt)
	values := c.values
	c.mu.RUnlock()
	return values, valid
}

func (c *ptzCache) Set(values pixy.PTZValues, ttl time.Duration) {
	c.mu.Lock()
	c.values = values
	c.expiresAt = time.Now().Add(ttl)
	c.mu.Unlock()
}

func (c *ptzCache) Invalidate() {
	c.mu.Lock()
	c.expiresAt = time.Time{}
	c.mu.Unlock()
}
