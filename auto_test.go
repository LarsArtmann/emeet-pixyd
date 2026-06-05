//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func testAutoDaemon(opts ...testDaemonOption) *Daemon {
	return newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, opts...)
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
	d := testAutoDaemon(withNotifyCalled(&notifyCalled))

	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AutoFull)

	assertInCall(t, d, true)

	if !notifyCalled {
		t.Error("expected notify to be called")
	}
}

func TestHandleCallStart_TracksFromPrivacy(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(withCameraState(pixy.StatePrivacy))

	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AutoFull)

	assertInCall(t, d, true)
}

func TestHandleCallStart_SwitchesAudioToNC(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(withAudioState(pixy.AudioLive))

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoFull)

	assertInCall(t, d, true)
}

func TestHandleCallStart_TrackingOnlyNoAudio(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(withAudioState(pixy.AudioLive))

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoTrackingOnly)

	if audio := readAudioState(d); audio != pixy.AudioLive {
		t.Errorf("tracking-only should not change audio, got %s", audio)
	}
}

func TestHandleCallStart_PrivacyOnlyNoTracking(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(withCameraState(pixy.StatePrivacy))

	d.handleCallStart(context.Background(), pixy.StatePrivacy, pixy.AutoPrivacyOnly)

	if camera := readCameraState(d); camera != pixy.StatePrivacy {
		t.Errorf("privacy-only should not activate tracking, got %s", camera)
	}
}

func TestHandleCallStart_SetsPipeWireSource(t *testing.T) {
	t.Parallel()

	var setSourceCalled bool
	d := testAutoDaemon(
		withFindSource("42"),
		func(d *Daemon) {
			d.setSourceFn = func(_ context.Context, id pixy.SourceID) {
				setSourceCalled = true
				if id.Get() != "42" {
					t.Errorf("expected source id 42, got %s", id.Get())
				}
			}
		},
	)

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoFull)

	if !setSourceCalled {
		t.Error("expected setSource to be called with found source")
	}
}

func TestHandleCallStart_TrackingOnlyNoSourceSwitch(t *testing.T) {
	t.Parallel()

	var setSourceCalled bool
	d := testAutoDaemon(
		withFindSource("42"),
		func(d *Daemon) {
			d.setSourceFn = func(_ context.Context, _ pixy.SourceID) { setSourceCalled = true }
		},
	)

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
		withNotifyCalled(&notifyCalled),
	)

	d.handleCallEnd(context.Background(), pixy.AutoFull)

	assertInCall(t, d, false)

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

	if camera := readCameraState(d); camera != pixy.StateTracking {
		t.Errorf("auto-off should not enter privacy, got %s", camera)
	}
}

func TestAutoManage_NoDevice_Returns(t *testing.T) {
	t.Parallel()

	if probeVideo4linux("/sys/class/video4linux") != "" {
		t.Skip("PIXY device physically connected — test requires no device")
	}

	d := newTestDaemon(pixy.StatePrivacy, "", "")
	d.autoManage(context.Background())

	if camera := readCameraState(d); camera != pixy.StateOffline {
		t.Errorf("expected offline with no device, got %s", camera)
	}
}

func TestAutoManage_AutoOff_NoAction(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, withAutoOff())

	d.autoManage(context.Background())

	assertInCall(t, d, false)
	camera := readCameraState(d)

	if camera != pixy.StatePrivacy {
		t.Errorf("camera state should not change with auto off, got %s", camera)
	}
}

func TestAutoManage_InUseTriggersCallStart(t *testing.T) {
	t.Parallel()

	var notifyCalled bool
	d := testAutoDaemon(
		withCameraInUse(true),
		withNotifyCalled(&notifyCalled),
		withDebounceCount(),
	)

	d.autoManage(context.Background())

	assertInCall(t, d, true)
	debounceInUse, _ := readDebounce(d)

	if debounceInUse != 1 {
		t.Errorf("expected debounceInUse=1, got %d", debounceInUse)
	}

	if !notifyCalled {
		t.Error("expected notify to be called")
	}
}

func TestAutoManage_InUseNotEnoughDebounce(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(withCameraInUse(true))

	d.autoManage(context.Background())

	assertInCall(t, d, false)
	debounceInUse, _ := readDebounce(d)

	if debounceInUse != 1 {
		t.Errorf("expected debounceInUse=1, got %d", debounceInUse)
	}
}

func TestAutoManage_IdleTriggersCallEnd(t *testing.T) {
	t.Parallel()

	var notifyCalled bool
	d := testAutoDaemon(
		withInCall(true),
		withCameraInUse(false),
		withNotifyCalled(&notifyCalled),
		withDebounceCount(),
	)

	d.autoManage(context.Background())

	assertInCall(t, d, false)

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

//nolint:paralleltest
func TestAutoManage_UpdatesMetrics(t *testing.T) {
	d := testAutoDaemon()
	d.autoManage(context.Background())

	requireGaugeValue(t, "emeet_pixyd_auto_mode", 1)
	requireGaugeValue(t, "emeet_pixyd_in_call", 0)
}

func TestAutoManage_SavesStateAfterRun(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	d := newTestDaemon(
		pixy.StatePrivacy, testVideoDev, testHIDDev,
		withConfig(dir), withCameraInUse(true), withDebounceCount(),
	)

	d.autoManage(context.Background())

	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("state file is empty")
	}
}

func withCaptureTrackingSlice(captured *[]pixy.CameraState) testDaemonOption {
	return func(d *Daemon) {
		d.setTrackingFn = func(_ context.Context, s pixy.CameraState) error {
			*captured = append(*captured, s)
			return nil
		}
	}
}

func withDebounceCount() testDaemonOption {
	return func(d *Daemon) { d.config.DebounceCount = 1 }
}

func TestAutoManage_UsesMockedTrackingFn(t *testing.T) {
	t.Parallel()

	var trackingCalls []pixy.CameraState
	d := testAutoDaemon(
		withCameraInUse(true),
		withDebounceCount(),
		withCaptureTrackingSlice(&trackingCalls),
	)

	d.autoManage(context.Background())

	assertInCall(t, d, true)
	if len(trackingCalls) == 0 {
		t.Fatal("expected setTrackingFn to be called via auto-manage")
	}
	if trackingCalls[0] != pixy.StateTracking {
		t.Errorf("expected tracking call with StateTracking, got %s", trackingCalls[0])
	}
}

func TestAutoManage_UsesMockedAudioFn(t *testing.T) {
	t.Parallel()

	var audioCalls []pixy.AudioMode
	d := testAutoDaemon(func(d *Daemon) {
		d.isCameraInUseFn = cameraInUseFn
		d.config.DebounceCount = 1
		d.setAudioFn = func(_ context.Context, m pixy.AudioMode) error {
			audioCalls = append(audioCalls, m)
			return nil
		}
	})

	d.autoManage(context.Background())

	if len(audioCalls) == 0 {
		t.Fatal("expected setAudioFn to be called via auto-manage")
	}
	if audioCalls[0] != pixy.AudioNC {
		t.Errorf("expected audio call with AudioNC, got %s", audioCalls[0])
	}
}

func TestAutoManage_CallEndUsesMockedTrackingFn(t *testing.T) {
	t.Parallel()

	var trackingCalls []pixy.CameraState
	d := testAutoDaemon(
		withInCall(true),
		withCameraInUse(false),
		withDebounceCount(),
		withCaptureTrackingSlice(&trackingCalls),
	)

	d.autoManage(context.Background())

	assertInCall(t, d, false)
	if len(trackingCalls) == 0 {
		t.Fatal("expected setTrackingFn to be called on call end")
	}
	if trackingCalls[0] != pixy.StatePrivacy {
		t.Errorf("expected privacy call on call end, got %s", trackingCalls[0])
	}
}
