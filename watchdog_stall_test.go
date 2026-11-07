//go:build linux

package main

import (
	"context"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TestWatchdogStall_HungV4L2BoundsLoopIteration pins the subprocess timeout
// policy (#190) against the systemd watchdog contract (arch review
// 2026-10-31 c4, TODO #201). WatchdogSec is 30s (modules/nixos.nix) and the
// event loop pings WATCHDOG=1 once per iteration, so the loop must never
// stall for long. The only way a lock is held for a long time is a wedged
// subprocess: a hung v4l2-ctl inside a PTZ write holds v4l2Mu, and the
// uevent re-probe — the loop's only nested v4l2Mu+hidMu site — blocks
// behind it. This test wedges exactly that case and asserts the stall
// stays far below the watchdog budget.
func TestWatchdogStall_HungV4L2BoundsLoopIteration(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.DesiredTracking, testVideoDev, testHIDDev, withCommander(&hungV4L2Commander{}))

	stallStart := time.Now()

	moveDone := make(chan error, 1)

	go func() {
		// The move path as production runs it: v4l2Mu held across the exec
		// (one ctrl write = one v4l2-ctl invocation, the minimal wedge).
		d.v4l2Mu.Lock()
		moveDone <- d.v4l2Set(context.Background(), testVideoDev, "pan_absolute", "0")
		d.v4l2Mu.Unlock()
	}()

	// The wedged move must terminate at the subprocess timeout, not hang
	// until the watchdog kills the unit. Bound = the timeout plus generous
	// scheduler slack; the watchdog invariant it feeds is < 15s (half of
	// WatchdogSec).
	const moveBound = v4l2CommandTimeout + 5*time.Second

	var moveErr error

	select {
	case moveErr = <-moveDone:
	case <-time.After(moveBound):
		t.Fatalf("PTZ write with hung v4l2-ctl did not return within %s (timeout policy broken)", moveBound)
	}

	// The #190 comment's classification claim: the deadline error the
	// commander surfaces classifies as Transient, so callers retry or
	// degrade instead of treating the wedge as a fatal condition.
	if got := errorfamily.Classify(moveErr); got != errorfamily.Transient {
		t.Errorf("Classify(hung v4l2-ctl error) = %v, want Transient (DeadlineExceeded)", got)
	}

	// The loop's uevent re-probe (takes v4l2Mu then hidMu) must complete
	// once the wedged write released the lock.
	probeStart := time.Now()

	d.v4l2Mu.Lock()
	d.hidMu.Lock()
	applyProbe(d, probeResult{})
	d.hidMu.Unlock()
	d.v4l2Mu.Unlock()

	probeWait := time.Since(probeStart)

	if probeWait > 5*time.Second {
		t.Errorf("uevent re-probe blocked %s behind the released lock, want near-instant", probeWait)
	}

	stall := time.Since(stallStart)

	if stall >= 15*time.Second {
		t.Errorf("worst-case loop stall = %s, want < 15s (half of WatchdogSec=30s)", stall)
	}
}

// hungV4L2Commander simulates a wedged v4l2-ctl: the call blocks until the
// commander's deadline cancels the context, then reports DeadlineExceeded —
// the same observable behavior realCommandRunner produces for a killed
// subprocess. Non-v4l2 binaries succeed immediately.
type hungV4L2Commander struct{}

func (c *hungV4L2Commander) Run(ctx context.Context, name string, args ...string) error {
	if name == v4l2ctl {
		<-ctx.Done()

		return ctx.Err()
	}

	return nil
}

func (c *hungV4L2Commander) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == v4l2ctl {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	return nil, nil
}

func (c *hungV4L2Commander) LookPath(name string) (string, error) {
	return "/usr/bin/" + name, nil
}
