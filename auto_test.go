//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func testAutoDaemon(opts ...testDaemonOption) *Daemon {
	return newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, opts...)
}

func readState[T any](d *Daemon, fn func(pixy.State) T) T {
	d.mu.RLock()
	v := fn(d.state)
	d.mu.RUnlock()
	return v
}

func readDebounce(d *Daemon) (inUse, idle int) {
	d.mu.RLock()
	inUse = d.debounceInUse
	idle = d.debounceIdle
	d.mu.RUnlock()
	return inUse, idle
}

func TestHandleCallStart_SetsInCall(t *testing.T) {
	t.Parallel()

	var notifyCalled bool
	d := testAutoDaemon(func(d *Daemon) {
		d.notifyFn = func(_ context.Context, _, _ string) { notifyCalled = true }
	})

	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AutoFull)

	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=true after handleCallStart")
	}

	if !notifyCalled {
		t.Error("expected notify to be called")
	}
}

func TestHandleCallStart_TracksFromPrivacy(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(func(d *Daemon) {
		d.state.Camera = pixy.StatePrivacy
	})

	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AutoFull)

	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=true")
	}
}

func TestHandleCallStart_SwitchesAudioToNC(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(func(d *Daemon) {
		d.state.Audio = pixy.AudioLive
	})

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoFull)

	if !readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=true")
	}
}

func TestHandleCallStart_TrackingOnlyNoAudio(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(func(d *Daemon) {
		d.state.Audio = pixy.AudioLive
	})

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoTrackingOnly)

	if audio := readState(d, func(s pixy.State) pixy.AudioMode {
		return s.Audio
	}); audio != pixy.AudioLive {
		t.Errorf("tracking-only should not change audio, got %s", audio)
	}
}

func TestHandleCallStart_PrivacyOnlyNoTracking(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(func(d *Daemon) {
		d.state.Camera = pixy.StatePrivacy
	})

	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AutoPrivacyOnly)

	if camera := readState(d, func(s pixy.State) pixy.CameraState {
		return s.Camera
	}); camera != pixy.StatePrivacy {
		t.Errorf("privacy-only should not activate tracking, got %s", camera)
	}
}

func TestHandleCallStart_SetsPipeWireSource(t *testing.T) {
	t.Parallel()

	var setSourceCalled bool
	d := testAutoDaemon(func(d *Daemon) {
		d.findSourceFn = func(_ context.Context) (string, error) { return "42", nil }
		d.setSourceFn = func(_ context.Context, id string) {
			setSourceCalled = true
			if id != "42" {
				t.Errorf("expected source id 42, got %s", id)
			}
		}
	})

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoFull)

	if !setSourceCalled {
		t.Error("expected setSource to be called with found source")
	}
}

func TestHandleCallStart_TrackingOnlyNoSourceSwitch(t *testing.T) {
	t.Parallel()

	var setSourceCalled bool
	d := testAutoDaemon(func(d *Daemon) {
		d.findSourceFn = func(_ context.Context) (string, error) { return "42", nil }
		d.setSourceFn = func(_ context.Context, _ string) { setSourceCalled = true }
	})

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoTrackingOnly)

	if setSourceCalled {
		t.Error("tracking-only should not switch PipeWire source")
	}
}

func TestHandleCallEnd_ClearsInCall(t *testing.T) {
	t.Parallel()

	var notifyCalled bool
	d := testAutoDaemon(
		withInCall(true),
		func(d *Daemon) {
			d.notifyFn = func(_ context.Context, _, _ string) { notifyCalled = true }
		},
	)

	d.handleCallEnd(context.Background(), pixy.AutoFull)

	if readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=false after handleCallEnd")
	}

	if !notifyCalled {
		t.Error("expected notify to be called")
	}
}

func TestHandleCallEnd_PrivacyOnlyNoPrivacy(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(
		withInCall(true),
		func(d *Daemon) {
			d.state.Camera = pixy.StateTracking
		},
	)

	d.handleCallEnd(context.Background(), pixy.AutoOff)

	if camera := readState(d, func(s pixy.State) pixy.CameraState {
		return s.Camera
	}); camera != pixy.StateTracking {
		t.Errorf("auto-off should not enter privacy, got %s", camera)
	}
}

func TestAutoManage_NoDevice_Returns(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")
	d.autoManage(context.Background())

	if camera := readState(d, func(s pixy.State) pixy.CameraState {
		return s.Camera
	}); camera != pixy.StateOffline {
		t.Errorf("expected offline with no device, got %s", camera)
	}
}

func TestAutoManage_AutoOff_NoAction(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.state.AutoMode = pixy.AutoOff
	})

	d.autoManage(context.Background())

	inCall := readState(d, func(s pixy.State) bool { return s.InCall })
	camera := readState(d, func(s pixy.State) pixy.CameraState { return s.Camera })

	if inCall {
		t.Error("should not be in call with auto off")
	}

	if camera != pixy.StatePrivacy {
		t.Errorf("camera state should not change with auto off, got %s", camera)
	}
}

func TestAutoManage_InUseTriggersCallStart(t *testing.T) {
	t.Parallel()

	var notifyCalled bool
	d := testAutoDaemon(func(d *Daemon) {
		d.isCameraInUseFn = func(_ string) bool { return true }
		d.notifyFn = func(_ context.Context, _, _ string) { notifyCalled = true }
		d.config.DebounceCount = 1
	})

	d.autoManage(context.Background())

	inCall := readState(d, func(s pixy.State) bool { return s.InCall })
	debounceInUse, _ := readDebounce(d)

	if !inCall {
		t.Error("expected InCall=true after camera in use with debounce=1")
	}

	if debounceInUse != 1 {
		t.Errorf("expected debounceInUse=1, got %d", debounceInUse)
	}

	if !notifyCalled {
		t.Error("expected notify to be called")
	}
}

func TestAutoManage_InUseNotEnoughDebounce(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(func(d *Daemon) {
		d.isCameraInUseFn = func(_ string) bool { return true }
	})

	d.autoManage(context.Background())

	inCall := readState(d, func(s pixy.State) bool { return s.InCall })
	debounceInUse, _ := readDebounce(d)

	if inCall {
		t.Error("should not be in call after single poll with debounce=3")
	}

	if debounceInUse != 1 {
		t.Errorf("expected debounceInUse=1, got %d", debounceInUse)
	}
}

func TestAutoManage_IdleTriggersCallEnd(t *testing.T) {
	t.Parallel()

	var notifyCalled bool
	d := testAutoDaemon(
		withInCall(true),
		func(d *Daemon) {
			d.isCameraInUseFn = func(_ string) bool { return false }
			d.notifyFn = func(_ context.Context, _, _ string) { notifyCalled = true }
			d.config.DebounceCount = 1
		},
	)

	d.autoManage(context.Background())

	if readState(d, func(s pixy.State) bool { return s.InCall }) {
		t.Error("expected InCall=false after camera idle with debounce=1")
	}

	if !notifyCalled {
		t.Error("expected notify to be called")
	}
}

func TestAutoManage_DebounceResetsOnStateChange(t *testing.T) {
	t.Parallel()

	callCount := 0
	d := testAutoDaemon(func(d *Daemon) {
		d.isCameraInUseFn = func(_ string) bool {
			callCount++
			return callCount <= 2
		}
		d.config.DebounceCount = 3
	})

	d.autoManage(context.Background())
	d.autoManage(context.Background())

	debounceInUse, debounceIdle := readDebounce(d)

	if debounceInUse != 2 {
		t.Errorf("expected debounceInUse=2, got %d", debounceInUse)
	}

	if debounceIdle != 0 {
		t.Errorf("expected debounceIdle=0, got %d", debounceIdle)
	}

	d.autoManage(context.Background())

	debounceInUse, debounceIdle = readDebounce(d)

	if debounceInUse != 0 {
		t.Errorf("expected debounceInUse=0 after flip, got %d", debounceInUse)
	}

	if debounceIdle != 1 {
		t.Errorf("expected debounceIdle=1 after flip, got %d", debounceIdle)
	}
}

func TestAutoManage_UpdatesMetrics(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon()
	d.autoManage(context.Background())

	requireGaugeValue(t, "emeet_pixyd_auto_mode", 1)
	requireGaugeValue(t, "emeet_pixyd_in_call", 0)
}

func TestAutoManage_SavesStateAfterRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, func(d *Daemon) {
		d.config = pixy.Config{
			StateDir:      dir,
			PollInterval:  2 * time.Second,
			DebounceCount: 3,
			WebAddr:       "127.0.0.1:0",
		}
	})

	d.autoManage(context.Background())

	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("state file is empty")
	}
}
