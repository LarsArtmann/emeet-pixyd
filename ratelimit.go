package main

import (
	"sync"
	"time"
)

// warnLimiter rate-limits repeated identical log warnings by key.
//
// Motivation: while the PIXY is absent, the event loop's ticker calls
// autoManage every PollInterval, which re-probes; every non-PIXY video4linux
// entry whose device/uevent symlink does not exist re-logs the SAME ENOENT
// WARN — measured ~160 identical lines/day in production (4,809 in 30 days,
// 2026-09-02). Once per key per interval is plenty to notice a real
// regression while keeping the journal readable.
type warnLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     map[string]time.Time
	now      func() time.Time // injectable for tests
}

func newWarnLimiter(interval time.Duration) *warnLimiter {
	return &warnLimiter{
		interval: interval,
		last:     make(map[string]time.Time),
		now:      time.Now,
	}
}

// allow reports whether the warning keyed by key should be logged now,
// recording the decision so consecutive calls within the interval are
// suppressed.
func (l *warnLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	if prev, ok := l.last[key]; ok && now.Sub(prev) < l.interval {
		return false
	}

	l.last[key] = now

	return true
}
