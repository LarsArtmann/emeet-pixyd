//go:build linux

package main

import (
	"context"
	"sync"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// fakeHIDDevice is a test double for HIDDevice that records all interactions
// and returns configurable responses. Enables testing HID code paths without
// real /dev/hidraw* hardware.
type fakeHIDDevice struct {
	mu         sync.Mutex
	sentBytes  [][]byte
	recvBytes  [][]byte
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
	f.recvBytes = append(f.recvBytes, resp)

	return resp, nil
}

func (f *fakeHIDDevice) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.sentBytes)
}

func (f *fakeHIDDevice) lastSent() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.sentBytes) == 0 {
		return nil
	}

	return f.sentBytes[len(f.sentBytes)-1]
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

func (f *fakeUeventListener) trigger() {
	if f.ch != nil {
		select {
		case f.ch <- struct{}{}:
		default:
		}
	}
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

// withFakeCameraInUse configures the fake process inspector to report
// the camera as in-use (or not). Requires withFakeDevices().
func withFakeCameraInUse(inUse bool) testDaemonOption {
	return func(d *Daemon) {
		if fake, ok := d.deps.procInspector.(*fakeProcInspector); ok {
			fake.cameraInUse = inUse
		}
	}
}

// withFakeHIDResponse sets a custom SendRecv handler on the fake HID device.
// Requires withFakeDevices().
func withFakeHIDResponse(fn func(report []byte) ([]byte, error)) testDaemonOption {
	return func(d *Daemon) {
		if fake, ok := d.hidDev.(*fakeHIDDevice); ok {
			fake.sendRecvFn = fn
		}
	}
}

// withFakePPID sets a parent-child relationship in the fake process inspector.
// Requires withFakeDevices().
func withFakePPID(child, parent int) testDaemonOption {
	return func(d *Daemon) {
		if fake, ok := d.deps.procInspector.(*fakeProcInspector); ok {
			fake.ppidMap[child] = parent
		}
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

// fakeSleepDuration is a short sleep for fake-device tests that need to wait
// for async operations (e.g., PTZ readback goroutine) to complete.
const fakeSleepDuration = 600 * time.Millisecond
