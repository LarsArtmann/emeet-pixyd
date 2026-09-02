package main

import (
	"testing"
	"time"
)

func TestWarnLimiter_FirstCallAllowed(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter := newWarnLimiter(time.Hour)
	limiter.now = func() time.Time { return current }

	if !limiter.allow("/sys/class/video4linux/video0/device/uevent") {
		t.Fatal("first call for a key must be allowed")
	}
}

func TestWarnLimiter_SecondCallWithinIntervalSuppressed(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter := newWarnLimiter(time.Hour)
	limiter.now = func() time.Time { return current }

	limiter.allow("key")

	current = current.Add(59 * time.Minute)
	if limiter.allow("key") {
		t.Fatal("call within the interval must be suppressed")
	}
}

func TestWarnLimiter_AllowedAgainAfterInterval(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter := newWarnLimiter(time.Hour)
	limiter.now = func() time.Time { return current }

	limiter.allow("key")

	current = current.Add(time.Hour)
	if !limiter.allow("key") {
		t.Fatal("call after the interval must be allowed again")
	}
}

func TestWarnLimiter_KeysAreIndependent(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter := newWarnLimiter(time.Hour)
	limiter.now = func() time.Time { return current }

	limiter.allow("video0")

	if !limiter.allow("video1") {
		t.Fatal("a different key must not be suppressed by another key's warning")
	}
}
