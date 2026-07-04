//go:build linux

package main

import (
	"context"
	"sync"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// fakeHIDDevice is a test double for HIDDevice that records all interactions
// and returns configurable responses. Enables testing HID code paths without
// real /dev/hidraw* hardware.
type fakeHIDDevice struct {
	mu         sync.Mutex
	sentBytes  [][]byte
	sendErr    error
	sendRecvFn func(report []byte) ([]byte, error)
}

func newFakeHIDDevice() *fakeHIDDevice {
	return &fakeHIDDevice{}
}

func (f *fakeHIDDevice) String() string { return "fake-hid" }

func (f *fakeHIDDevice) Send(report []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sentBytes = append(f.sentBytes, append([]byte(nil), report...))

	return f.sendErr
}

func (f *fakeHIDDevice) SendRecv(_ context.Context, report []byte) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sentBytes = append(f.sentBytes, append([]byte(nil), report...))

	if f.sendRecvFn != nil {
		return f.sendRecvFn(report)
	}

	resp := make([]byte, hidRespBufSize)

	return resp, nil
}

// fakeProcInspector is a configurable ProcessInspector for testing
// call-detection and process-tree traversal without real /proc access.
type fakeProcInspector struct {
	cameraInUse bool
	ppidMap     map[int]int // pid → ppid
}

func newFakeProcInspector() *fakeProcInspector {
	return &fakeProcInspector{
		ppidMap: make(map[int]int),
	}
}

func (f *fakeProcInspector) PPIDOf(pid pixy.PID) pixy.PID {
	if ppid, ok := f.ppidMap[pid.Get()]; ok {
		return pixy.NewPID(ppid)
	}

	return pixy.PID{}
}

func (f *fakeProcInspector) IsDescendantOf(pid, ancestor pixy.PID) bool {
	visited := make(map[int]bool)

	for range maxDescendantDepth {
		id := pid.Get()
		if visited[id] || pid.IsZero() || pid.Equal(ancestor) {
			break
		}

		visited[id] = true

		ppid := f.PPIDOf(pid)
		if ppid.Equal(ancestor) {
			return true
		}

		if ppid.IsZero() || ppid.Equal(pid) {
			return false
		}

		pid = ppid
	}

	return false
}

func (f *fakeProcInspector) IsCameraInUse(_ string) bool {
	return f.cameraInUse
}

// fakeUeventListener is a test UeventListener that can be triggered manually.
type fakeUeventListener struct {
	ch chan<- struct{}
}

func (f *fakeUeventListener) Listen(_ context.Context, ch chan<- struct{}) {
	f.ch = ch
}

// withFakeDevices wires all dependency interfaces to fake implementations,
// enabling tests to exercise real code paths (setTracking, setAudio, autoManage)
// without touching /dev/hidraw*, /dev/video*, or real /proc.
func withFakeDevices() testDaemonOption {
	return func(d *Daemon) {
		fakeHID := newFakeHIDDevice()
		fakeProc := newFakeProcInspector()

		d.hidDev = fakeHID
		d.deps.procInspector = fakeProc
		d.deps.ueventListener = noopUeventListener{}
		d.deps.isCameraInUse = fakeProc.IsCameraInUse
		d.deps.commander = noopCommandRunner{}
		d.deps.parsePTZ = func(context.Context, string) pixy.PTZValues {
			return pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault}
		}
		d.deps.setTracking = func(_ context.Context, state pixy.CameraState) error {
			d.mu.Lock()
			d.state.Camera = state
			d.mu.Unlock()

			return nil
		}
		d.deps.setAudio = func(_ context.Context, mode pixy.AudioMode) error {
			d.mu.Lock()
			d.state.Audio = mode
			d.mu.Unlock()

			return nil
		}
		d.deps.setGesture = func(_ context.Context, enabled bool) error {
			d.mu.Lock()
			d.state.Gesture = enabled
			d.mu.Unlock()

			return nil
		}
		d.deps.centerCamera = func(context.Context) error { return nil }
		d.deps.v4l2Set = func(context.Context, string, string, string) error { return nil }
	}
}

// withFakeParsePTZ sets a deterministic PTZ readback value for the fake harness.
func withFakeParsePTZ(pan, tilt, zoom int) testDaemonOption {
	return func(d *Daemon) {
		d.deps.parsePTZ = func(context.Context, string) pixy.PTZValues {
			return pixy.PTZValues{Pan: pan, Tilt: tilt, Zoom: zoom}
		}
	}
}
