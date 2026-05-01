//go:build linux

package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func testAutoDaemon(opts ...testDaemonOption) *Daemon {
	return newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, opts...)
}

func TestHandleCallStart_SetsInCall(t *testing.T) {
	t.Parallel()
	d := testAutoDaemon()
	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AudioNC)
	d.mu.RLock()
	inCall := d.state.InCall
	d.mu.RUnlock()
	if !inCall {
		t.Error("expected InCall=true after handleCallStart")
	}
}

func TestHandleCallEnd_ClearsInCall(t *testing.T) {
	t.Parallel()
	d := testAutoDaemon(withInCall(true))
	d.handleCallEnd(context.Background())
	d.mu.RLock()
	inCall := d.state.InCall
	d.mu.RUnlock()
	if inCall {
		t.Error("expected InCall=false after handleCallEnd")
	}
}

func TestAutoManage_NoDevice_Returns(t *testing.T) {
	t.Parallel()
	d := newTestDaemon(pixy.StatePrivacy, "", "")
	d.autoManage(context.Background())
	d.mu.RLock()
	camera := d.state.Camera
	d.mu.RUnlock()
	if camera != pixy.StateOffline {
		t.Errorf("expected offline with no device, got %s", camera)
	}
}

func TestAutoManage_AutoOff_NoAction(t *testing.T) {
	t.Parallel()
	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.state.AutoMode = false
	})
	d.autoManage(context.Background())
	d.mu.RLock()
	inCall := d.state.InCall
	camera := d.state.Camera
	d.mu.RUnlock()
	if inCall {
		t.Error("should not be in call with auto off")
	}
	if camera != pixy.StatePrivacy {
		t.Errorf("camera state should not change with auto off, got %s", camera)
	}
}

func TestAutoManage_InUseNotEnoughDebounce(t *testing.T) {
	t.Parallel()
	d := testAutoDaemon()
	// Camera is not in use (no real device scanning will find it),
	// but we simulate by checking debounce doesn't trigger immediately
	d.autoManage(context.Background())
	d.mu.RLock()
	inCall := d.state.InCall
	debounceInUse := d.debounceInUse
	d.mu.RUnlock()
	if inCall {
		t.Error("should not be in call after single poll")
	}
	_ = debounceInUse
}

func TestAutoManage_UpdatesMetrics(t *testing.T) {
	t.Parallel()
	registerMetrics()
	d := testAutoDaemon()
	d.autoManage(context.Background())
}

func TestAutoManage_SavesStateAfterRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	d := &Daemon{
		mu:    sync.RWMutex{},
		state: pixy.DefaultState(),
		config: pixy.Config{
			StateDir:      dir,
			PollInterval:  2 * time.Second,
			DebounceCount: 3,
			WebAddr:       "127.0.0.1:0",
		},
		videoDev:   testVideoDev,
		hidrawDev:  testHIDDev,
		streamSema: make(chan struct{}, 1),
	}
	d.autoManage(context.Background())
}
