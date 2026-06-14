//go:build linux

package main

import (
	"context"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func testAutoDaemon(opts ...testDaemonOption) *Daemon {
	return newTestDaemon(pixy.StatePrivacy, testVideoDev, testHIDDev, opts...)
}

func readDebounce(d *Daemon) (_, _ int) {
	d.mu.RLock()
	inUse := d.debounceInUse
	idle := d.debounceIdle
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
			d.deps.setSource = func(_ context.Context, id pixy.SourceID) {
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
			d.deps.setSource = func(_ context.Context, _ pixy.SourceID) { setSourceCalled = true }
		},
	)

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoTrackingOnly)

	if setSourceCalled {
		t.Error("tracking-only should not switch PipeWire source")
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

func TestHandleCallStart_FindSourceErrorSurfacesInAutoError(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(
		func(d *Daemon) {
			d.deps.findSource = func(_ context.Context) (pixy.SourceID, error) {
				return pixy.SourceID{}, ErrAudioSourceNotFound
			}
		},
	)

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoFull)

	d.mu.RLock()
	autoErr := d.autoError
	d.mu.RUnlock()

	if autoErr == nil {
		t.Fatal("expected autoError to be set when findSource fails")
	}
}

func TestHandleCallStart_FindSourceSuccessClearsAutoError(t *testing.T) {
	t.Parallel()

	d := testAutoDaemon(withFindSource("42"))

	d.handleCallStart(context.Background(), pixy.StateTracking, pixy.AutoFull)

	d.mu.RLock()
	autoErr := d.autoError
	d.mu.RUnlock()

	if autoErr != nil {
		t.Errorf("expected autoError to be nil on success, got %v", autoErr)
	}
}
