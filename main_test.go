//go:build linux

package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	testVideoDev = "/dev/video0"
	testHIDDev   = "/dev/hidraw7"
)

const (
	testStrPrivacy  = "privacy"
	testStrTracking = "tracking"
	testStrUnknown  = "unknown"
)

func defaultTestConfig(dir string) pixy.Config {
	return pixy.Config{
		StateDir:      dir,
		PollInterval:  2 * time.Second,
		DebounceCount: 3,
		WebAddr:       testWebAddr,
		AutoMode:      pixy.AutoFull,
		DefaultAudio:  pixy.AudioNC,
		Debug:         false,
	}
}

type testDaemonOption func(*Daemon)

func withInCall(inCall bool) testDaemonOption {
	return func(d *Daemon) { d.state.InCall = inCall }
}

func withNotifyCalled(called *bool) testDaemonOption {
	return func(d *Daemon) {
		d.deps.notify = func(_ context.Context, _, _ string) { *called = true }
	}
}

func withCameraInUse(inUse bool) testDaemonOption {
	return func(d *Daemon) { d.deps.isCameraInUse = func(_ string) bool { return inUse } }
}

func cameraInUseFn(string) bool { return true }

func cameraNotInUseFn(string) bool { return false }

// Named noop stubs shared by withNoop* builders and noopDependencies().
// Eliminates closure duplication across test daemon wiring paths.
func noopV4L2Set(_ context.Context, _, _, _ string) error { return nil }

func noopSetTracking(_ context.Context, _ pixy.CameraState) error { return nil }

func noopSetAudio(_ context.Context, _ pixy.AudioMode) error { return nil }

func noopSetGesture(_ context.Context, _ bool) error { return nil }

func noopCenterCamera(_ context.Context) error { return nil }

func noopParsePTZ(_ context.Context, _ string) pixy.PTZValues {
	return pixy.PTZValues{}
}

func withNoopV4L2() testDaemonOption {
	return func(d *Daemon) { d.deps.v4l2Set = noopV4L2Set }
}

// withNoopParsePTZ wires a deterministic parsePTZ stub so relative-mode PTZ
// commands (e.g. "tilt -30") don't read real /dev/video0 hardware state.
func withNoopParsePTZ() testDaemonOption {
	return func(d *Daemon) { d.deps.parsePTZ = noopParsePTZ }
}

func withNoopTracking() testDaemonOption {
	return func(d *Daemon) { d.deps.setTracking = noopSetTracking }
}

func withNoopAudio() testDaemonOption {
	return func(d *Daemon) { d.deps.setAudio = noopSetAudio }
}

type v4l2Call struct {
	dev, ctrl, val string
}

func withCaptureV4L2(calls *[]v4l2Call) testDaemonOption {
	return func(d *Daemon) {
		d.deps.v4l2Set = func(_ context.Context, dev, ctrl, val string) error {
			*calls = append(*calls, v4l2Call{dev: dev, ctrl: ctrl, val: val})

			return nil
		}
	}
}

func withCaptureTracking(captured *pixy.CameraState) testDaemonOption {
	return func(d *Daemon) {
		d.deps.setTracking = func(_ context.Context, s pixy.CameraState) error {
			*captured = s

			return nil
		}
	}
}

func withCaptureAudio(captured *pixy.AudioMode) testDaemonOption {
	return func(d *Daemon) {
		d.deps.setAudio = func(_ context.Context, m pixy.AudioMode) error {
			*captured = m

			return nil
		}
	}
}

func withCaptureGesture(called, captured *bool) testDaemonOption {
	return func(d *Daemon) {
		d.deps.setGesture = func(_ context.Context, enabled bool) error {
			*called = true
			*captured = enabled

			return nil
		}
	}
}

func withCaptureGestureArg(captured *bool) testDaemonOption {
	return func(d *Daemon) {
		d.deps.setGesture = func(_ context.Context, enabled bool) error {
			*captured = enabled

			return nil
		}
	}
}

func withNotifyMessages(captured *[]string) testDaemonOption {
	return func(d *Daemon) {
		d.deps.notify = func(_ context.Context, _, body string) {
			*captured = append(*captured, body)
		}
	}
}

func withCaptureCenter(calls *int) testDaemonOption {
	return func(d *Daemon) {
		d.deps.centerCamera = func(context.Context) error {
			*calls++

			return nil
		}
	}
}

func withAutoOff() testDaemonOption {
	return func(d *Daemon) { d.state.AutoMode = pixy.AutoOff }
}

func ptr[T any](v T) *T { return new(v) }

func withFindSource(id string) testDaemonOption {
	return func(d *Daemon) {
		d.deps.findSource = func(_ context.Context) (pixy.SourceID, error) { return pixy.NewSourceID(id), nil }
	}
}

func withConfig(dir string) testDaemonOption {
	return func(d *Daemon) {
		d.config = defaultTestConfig(dir)
	}
}

func withCameraState(state pixy.CameraState) testDaemonOption {
	return func(d *Daemon) { d.state.Camera = state }
}

func withAudioState(mode pixy.AudioMode) testDaemonOption {
	return func(d *Daemon) { d.state.Audio = mode }
}

func noopFindSourceFn(context.Context) (pixy.SourceID, error) { return pixy.SourceID{}, nil }

func noopSetSourceFn(context.Context, pixy.SourceID) {}

func noopNotifyFn(context.Context, string, string) {}

func readState[T any](d *Daemon, fn func(pixy.State) T) T {
	d.mu.RLock()
	v := fn(d.state)
	d.mu.RUnlock()

	return v
}

func readCameraState(d *Daemon) pixy.CameraState {
	return readState(d, func(s pixy.State) pixy.CameraState { return s.Camera })
}

func readAudioState(d *Daemon) pixy.AudioMode {
	return readState(d, func(s pixy.State) pixy.AudioMode { return s.Audio })
}

// noopDependencies returns a Dependencies struct where every function is a
// no-op stub. Shared by test daemon builders that need a fully populated
// Dependencies without any real HID/V4L2 side effects.
func noopDependencies() Dependencies {
	return Dependencies{
		commander:     noopCommandRunner{},
		isCameraInUse: cameraNotInUseFn,
		findSource:    noopFindSourceFn,
		setSource:     noopSetSourceFn,
		notify:        noopNotifyFn,
		setTracking:   noopSetTracking,
		setAudio:      noopSetAudio,
		setGesture:    noopSetGesture,
		centerCamera:  noopCenterCamera,
		v4l2Set:       noopV4L2Set,
		parsePTZ:      noopParsePTZ,
	}
}

func newDaemonForStateTest(cfg pixy.Config, state pixy.State) *Daemon {
	return &Daemon{
		mu:         sync.RWMutex{},
		config:     cfg,
		state:      state,
		streamSema: make(chan struct{}, 1),
		deps:       noopDependencies(),
	}
}

func newTestDaemon(
	tb testing.TB,
	camera pixy.CameraState,
	videoDev, hidrawDev string,
	opts ...testDaemonOption,
) *Daemon {
	tb.Helper()

	d := &Daemon{
		mu: sync.RWMutex{},
		state: pixy.State{
			Camera:   camera,
			Audio:    pixy.AudioNC,
			Gesture:  false,
			InCall:   false,
			AutoMode: pixy.AutoFull,
		},

		config: pixy.Config{
			StateDir:      tb.TempDir(),
			PollInterval:  2 * time.Second,
			DebounceCount: 3,
			WebAddr:       "127.0.0.1:0",
			AutoMode:      pixy.AutoFull,
			DefaultAudio:  pixy.AudioNC,
			Debug:         false,
		},
		videoDev:      videoDev,
		hidrawDev:     hidrawDev,
		debounceInUse: 0,
		debounceIdle:  0,
		streamSema:    make(chan struct{}, 1),
		broadcaster:   NewBroadcaster(),
		deps: Dependencies{
			commander:     realCommandRunner{},
			isCameraInUse: func(string) bool { return false },
			findSource:    noopFindSourceFn,
			setSource:     noopSetSourceFn,
			notify:        noopNotifyFn,
		},
	}
	d.deps.setTracking = d.setTracking
	d.deps.setAudio = d.setAudio
	d.deps.setGesture = d.setGesture
	d.deps.centerCamera = d.centerCamera
	d.deps.v4l2Set = d.v4l2Set

	d.deps.parsePTZ = d.parsePTZValues
	if hidrawDev != "" {
		d.hidDev = newHIDRawDevice(hidrawDev)
	}

	registerMetrics()

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func testDaemonNoDevice(tb testing.TB) *Daemon {
	tb.Helper()

	return newTestDaemon(tb, pixy.StatePrivacy, "", "")
}

func testDaemonWithDevice(tb testing.TB, camera pixy.CameraState) *Daemon {
	tb.Helper()

	return newTestDaemon(tb, camera, testVideoDev, testHIDDev)
}

func testDaemonWithState(tb testing.TB, camera pixy.CameraState, inCall bool) *Daemon {
	tb.Helper()

	return newTestDaemon(tb, camera, "", "", withInCall(inCall))
}

type parseTestCase[T comparable] struct {
	input    string
	expected T
	wantErr  bool
}

func runParseTests[T comparable](
	t *testing.T,
	name string,
	parse func(string) (T, error),
	tests []parseTestCase[T],
) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("%s(%q): expected error, got nil", name, tc.input)
				}

				return
			}

			if err != nil {
				t.Errorf("%s(%q): unexpected error: %v", name, tc.input, err)

				return
			}

			if got != tc.expected {
				t.Errorf("%s(%q) = %v, want %v", name, tc.input, got, tc.expected)
			}
		})
	}
}

const (
	pixyUeventProduct = "328f/c0/2004"
	pixyVendor        = "328f"
	pixyProduct       = "00c0"
)

const (
	testVideoDev0 = "video0"
	testVideoDev2 = "video2"
)

func testV4L2ProbesPIXY(t *testing.T, devices []fakeVideoDev) {
	t.Helper()
	root := t.TempDir()
	createFakeVideo4linux(t, root, devices)

	result := probeVideo4linux(root)
	if result != testVideoDev {
		t.Errorf("expected /dev/video0, got %s", result)
	}
}

type failingHID struct {
	err error
}

func (f *failingHID) String() string { return "failing-hid" }

func (f *failingHID) Send(_ []byte) error { return f.err }

func (f *failingHID) SendRecv(_ context.Context, _ []byte) ([]byte, error) {
	return nil, f.err
}
