//go:build linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	testVideoDev = "/dev/video0"
	testHIDDev   = "/dev/hidraw7"
)

func testConfig(dir string) pixy.Config {
	//nolint:exhaustruct
	return pixy.Config{
		//nolint:exhaustruct
		StateDir:      dir,
		PollInterval:  2 * time.Second,
		DebounceCount: 3,
		WebAddr:       testWebAddr,
	}
}

type testDaemonOption func(*Daemon)

func withInCall(inCall bool) testDaemonOption {
	return func(d *Daemon) { d.state.InCall = inCall }
}

func withNotifyCalled(called *bool) testDaemonOption {
	return func(d *Daemon) {
		d.notifyFn = func(_ context.Context, _, _ string) { *called = true }
	}
}

func withCameraInUse(inUse bool) testDaemonOption {
	return func(d *Daemon) { d.isCameraInUseFn = func(_ string) bool { return inUse } }
}

func cameraInUseFn(string) bool { return true }

func cameraNotInUseFn(string) bool { return false }

func withNoopV4L2() testDaemonOption {
	return func(d *Daemon) {
		d.v4l2SetFn = func(_ context.Context, _, _, _ string) error { return nil }
	}
}

func withCaptureTracking(captured *pixy.CameraState) testDaemonOption {
	return func(d *Daemon) {
		d.setTrackingFn = func(_ context.Context, s pixy.CameraState) error {
			*captured = s
			return nil
		}
	}
}

func withCaptureAudio(captured *pixy.AudioMode) testDaemonOption {
	return func(d *Daemon) {
		d.setAudioFn = func(_ context.Context, m pixy.AudioMode) error {
			*captured = m
			return nil
		}
	}
}

func withCaptureGesture(called, captured *bool) testDaemonOption {
	return func(d *Daemon) {
		d.setGestureFn = func(_ context.Context, enabled bool) error {
			*called = true
			*captured = enabled
			return nil
		}
	}
}

func withCaptureCenter(calls *int) testDaemonOption {
	return func(d *Daemon) {
		d.centerCameraFn = func(context.Context) error { *calls++; return nil }
	}
}

func withFindSource(id string) testDaemonOption {
	return func(d *Daemon) {
		d.findSourceFn = func(_ context.Context) (string, error) { return id, nil }
	}
}

func withConfig(dir string) testDaemonOption {
	return func(d *Daemon) {
		//nolint:exhaustruct
		d.config = pixy.Config{
			StateDir:      dir,
			PollInterval:  2 * time.Second,
			DebounceCount: 3,
			WebAddr:       "127.0.0.1:0",
		}
	}
}

func assertInCall(t *testing.T, d *Daemon, want bool) {
	t.Helper()
	if got := readState(d, func(s pixy.State) bool { return s.InCall }); got != want {
		t.Errorf("expected InCall=%v, got %v", want, got)
	}
}

func assertHTTPStatusOK(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func assertNotifyContains(t *testing.T, messages []string, substr string) {
	t.Helper()
	if len(messages) == 0 {
		t.Fatalf("expected notification containing %q, but no notifications", substr)
	}
	if !strings.Contains(messages[0], substr) {
		t.Errorf("notification should mention %s, got: %s", substr, messages[0])
	}
}

func newDaemonForStateTest(cfg pixy.Config, state pixy.State) *Daemon {
	//nolint:exhaustruct
	return &Daemon{
		mu:              sync.RWMutex{},
		config:          cfg,
		state:           state,
		streamSema:      make(chan struct{}, 1),
		isCameraInUseFn: func(string) bool { return false },
		findSourceFn:    func(context.Context) (string, error) { return "", nil },
		setSourceFn:     func(context.Context, string) {},
		notifyFn:        func(context.Context, string, string) {},
	}
}

func newTestDaemon(
	camera pixy.CameraState,
	videoDev, hidrawDev string,
	opts ...testDaemonOption,
) *Daemon {
	//nolint:exhaustruct
	d := &Daemon{
		//nolint:exhaustruct
		mu: sync.RWMutex{},
		state: pixy.State{
			Camera:   camera,
			Audio:    pixy.AudioNC,
			Gesture:  false,
			InCall:   false,
			AutoMode: pixy.AutoFull,
		},
		//nolint:exhaustruct
		config: pixy.Config{
			//nolint:exhaustruct
			StateDir:      "/tmp",
			PollInterval:  2 * time.Second,
			DebounceCount: 3,
			WebAddr:       "127.0.0.1:0",
		},
		videoDev:        videoDev,
		hidrawDev:       hidrawDev,
		debounceInUse:   0,
		debounceIdle:    0,
		streamSema:      make(chan struct{}, 1),
		isCameraInUseFn: func(string) bool { return false },
		findSourceFn:    func(context.Context) (string, error) { return "", nil },
		setSourceFn:     func(context.Context, string) {},
		notifyFn:        func(context.Context, string, string) {},
	}
	d.setTrackingFn = d.setTracking
	d.setAudioFn = d.setAudio
	d.setGestureFn = d.setGesture
	d.centerCameraFn = d.centerCamera
	d.v4l2SetFn = v4l2Set
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func assertCameraState(t *testing.T, d *Daemon, expected pixy.CameraState) {
	t.Helper()

	if d.state.Camera != expected {
		t.Errorf("expected camera state %s, got %s", expected, d.state.Camera)
	}
}

func assertErrorPrefix(t *testing.T, result string) {
	t.Helper()

	if !strings.HasPrefix(result, "error:") {
		t.Errorf("expected error prefix, got: %s", result)
	}
}

func assertStatusContains(t *testing.T, result, substr, msg string) {
	t.Helper()

	if !strings.Contains(result, substr) {
		t.Errorf("%s: expected %q in status, got: %s", msg, substr, result)
	}
}

func assertStatusPrefix(t *testing.T, result, prefix, msg string) {
	t.Helper()

	if !strings.HasPrefix(result, prefix) {
		t.Errorf("%s: expected prefix %q, got: %s", msg, prefix, result)
	}
}

func assertAutoMode(t *testing.T, d *Daemon, expected pixy.AutoMode) {
	t.Helper()

	if d.state.AutoMode != expected {
		t.Errorf("expected auto mode=%v, got %v", expected, d.state.AutoMode)
	}
}

func assertGesture(t *testing.T, resp hidResponse, expected bool) {
	t.Helper()

	if !resp.Got || resp.Gesture != expected {
		t.Errorf("expected gesture=%v, got Got=%v Gesture=%v", expected, resp.Got, resp.Gesture)
	}
}

func assertParsedField(t *testing.T, parsed map[string]string, field string) {
	t.Helper()

	if _, ok := parsed[field]; !ok {
		t.Errorf("waybar output missing '%s' field", field)
	}
}

func assertTrackingIdle(t *testing.T, tracking pixy.CameraState) {
	t.Helper()
	if tracking != pixy.StateIdle {
		t.Errorf("Tracking = %q, want idle", tracking)
	}
}

func testDaemonNoDevice() *Daemon {
	return newTestDaemon(pixy.StatePrivacy, "", "")
}

func testDaemonWithDevice(camera pixy.CameraState) *Daemon {
	return newTestDaemon(camera, testVideoDev, testHIDDev)
}

func TestStateDefaults(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()
	assertCameraState(t, d, pixy.StatePrivacy)

	if d.state.Audio != pixy.AudioNC {
		t.Errorf("expected default audio to be nc, got %s", d.state.Audio)
	}

	assertAutoMode(t, d, pixy.AutoFull)

	if d.state.InCall != false {
		t.Error("expected in_call to be false by default")
	}
}

func TestStateSaveLoad(t *testing.T) {
	t.Parallel()

	cfg := testConfig(t.TempDir())

	//nolint:exhaustruct
	d := &Daemon{
		mu:     sync.RWMutex{},
		config: cfg,
		state: pixy.State{
			Camera:   pixy.StateTracking,
			Audio:    pixy.AudioLive,
			Gesture:  true,
			InCall:   true,
			AutoMode: pixy.AutoOff,
		},
		videoDev:      "",
		hidrawDev:     "",
		debounceInUse: 0,
		debounceIdle:  0,
	}

	saveErr := d.saveState()
	if saveErr != nil {
		t.Fatalf("saveState: %v", saveErr)
	}

	//nolint:exhaustruct
	d2 := &Daemon{
		//nolint:exhaustruct
		mu:     sync.RWMutex{},
		config: cfg,
		state: pixy.State{
			Camera:   pixy.StateIdle,
			Audio:    pixy.AudioNC,
			Gesture:  false,
			InCall:   false,
			AutoMode: pixy.AutoFull,
		},
		videoDev:      "",
		hidrawDev:     "",
		debounceInUse: 0,
		debounceIdle:  0,
	}
	d2.loadState()

	if d2.state.Camera != pixy.StateTracking {
		t.Errorf("expected camera=tracking, got %s", d2.state.Camera)
	}

	if d2.state.Audio != pixy.AudioLive {
		t.Errorf("expected audio=live, got %s", d2.state.Audio)
	}

	if d2.state.Gesture != true {
		t.Error("expected gesture=true")
	}

	if d2.state.InCall != true {
		t.Error("expected in_call=true")
	}

	if d2.state.AutoMode != pixy.AutoOff {
		t.Error("expected auto_mode=false")
	}
}

func TestStateFileCorrupt(t *testing.T) {
	t.Parallel()

	cfg := testConfig(t.TempDir())

	err := os.WriteFile(cfg.StateFile(), []byte("not json"), pixy.PermissionStateFile)
	if err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	d := testDaemonNoDevice()
	d.config = cfg
	d.loadState()

	if d.state.Camera != pixy.StatePrivacy {
		t.Errorf("expected state to remain unchanged on corrupt file, got %s", d.state.Camera)
	}
}

func TestStateFileMissing(t *testing.T) {
	t.Parallel()

	cfg := testConfig("/nonexistent")
	d := testDaemonNoDevice()
	d.config = cfg
	d.loadState()

	assertCameraState(t, d, pixy.StatePrivacy)
}

func TestHandleCommandStatus(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice()

	result := d.handleCommand(context.Background(), cmdStatus)
	assertStatusPrefix(t, result, "camera=offline", "offline status")
	assertStatusContains(t, result, "audio=", "offline status")
	assertStatusContains(t, result, "auto=", "offline status")
}

func TestHandleCommandUnknown(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, "/dev/hidraw0")

	result := d.handleCommand(context.Background(), "foobar")
	if result != "unknown command: foobar" {
		t.Errorf("expected unknown command response, got: %s", result)
	}
}

func TestHandleCommandAutoToggle(t *testing.T) {
	t.Parallel()

	d := testDaemonWithDevice(pixy.StatePrivacy)
	d.config = testConfig(t.TempDir())

	result := d.handleCommand(context.Background(), "auto-off")
	if result != respAutoModeOff {
		t.Errorf("expected 'auto mode off', got: %s", result)
	}

	if d.state.AutoMode != pixy.AutoOff {
		t.Error("expected auto mode to be false")
	}

	result = d.handleCommand(context.Background(), "auto-on")
	if result != "auto mode: full" {
		t.Errorf("expected 'auto mode: full', got: %s", result)
	}

	assertAutoMode(t, d, pixy.AutoFull)
}

func TestHandleCommandAudioInvalid(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, "/dev/hidraw0")

	result := d.handleCommand(context.Background(), "audio xyz")
	if result != respAudioUsage {
		t.Errorf("expected usage for invalid mode, got: %s", result)
	}
}

func TestHandleCommandDeviceRequired(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateOffline, "", "")

	for _, cmd := range []string{cmdTrack, cmdIdle, cmdPrivacy, cmdTogglePrivacy, cmdCenter, cmdGestureOn, cmdGestureOff} {
		result := d.handleCommand(context.Background(), cmd)
		if result == "" {
			t.Errorf("expected error response for '%s' with no device", cmd)
		}

		if len(result) < 6 || result[:6] != "error:" {
			t.Errorf("expected error: prefix for '%s' with no device, got: %s", cmd, result)
		}
	}
}

func testDaemonWithState(camera pixy.CameraState, inCall bool) *Daemon {
	return newTestDaemon(camera, "", "", withInCall(inCall))
}

func TestWaybarOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		camera   pixy.CameraState
		inCall   bool
		expected string
	}{
		{pixy.StateTracking, false, "tracking"},
		{pixy.StatePrivacy, false, "privacy"},
		{pixy.StateIdle, false, "idle"},
		{pixy.StateOffline, false, "offline"},
		{pixy.StateTracking, true, "tracking in-call"},
	}

	for _, testCase := range tests {
		d := testDaemonWithState(testCase.camera, testCase.inCall)
		output := d.waybarOutput()

		var parsed map[string]string

		err := json.Unmarshal([]byte(output), &parsed)
		if err != nil {
			t.Fatalf("waybar output is not valid JSON: %s, err: %v", output, err)
		}

		if parsed["class"] != "custom-camera "+testCase.expected {
			t.Errorf(
				"expected class 'custom-camera %s', got '%s'",
				testCase.expected,
				parsed["class"],
			)
		}

		assertParsedField(t, parsed, "text")
		assertParsedField(t, parsed, "tooltip")
	}
}

func TestIsCameraInUseEmptyDevice(t *testing.T) {
	t.Parallel()

	if isCameraInUse("") {
		t.Error("expected false for empty video device")
	}
}

func TestHandleCommandTogglePrivacy(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, testVideoDev, "/dev/hidraw0")

	result := d.handleCommand(context.Background(), cmdTogglePrivacy)
	if result == "" {
		t.Error("expected non-empty response")
	}
}

func TestHandleCommandProbe(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateOffline, "", "")

	result := d.handleCommand(context.Background(), cmdProbe)

	if d.videoDev != "" {
		assertStatusPrefix(t, result, "device found:", "PIXY connected")
	} else if result != respDeviceNotFound {
		t.Errorf("expected 'device not found' when no PIXY connected, got: %s", result)
	}
}

func TestAudioModeNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    pixy.AudioMode
		expected pixy.AudioMode
	}{
		{pixy.AudioNC, pixy.AudioLive},
		{pixy.AudioLive, pixy.AudioOriginal},
		{pixy.AudioOriginal, pixy.AudioNC},
		{pixy.AudioMode("unknown"), pixy.AudioNC},
	}
	for _, testCase := range tests {
		result := testCase.input.Next()
		if result != testCase.expected {
			t.Errorf(
				"pixy.AudioMode(%s).Next() = %s, want %s",
				testCase.input,
				result,
				testCase.expected,
			)
		}
	}
}

func TestAudioModeHIDByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode     pixy.AudioMode
		expected byte
	}{
		{pixy.AudioNC, hidByteNC},
		{pixy.AudioLive, hidByteLive},
		{pixy.AudioOriginal, hidByteOriginal},
		{pixy.AudioMode("unknown"), hidByteNC},
	}
	for _, testCase := range tests {
		result := audioHIDByte(testCase.mode)
		if result != testCase.expected {
			t.Errorf(
				"audioHIDByte(%s) = 0x%02x, want 0x%02x",
				testCase.mode,
				result,
				testCase.expected,
			)
		}
	}
}

func TestCameraStateHIDByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    pixy.CameraState
		expected byte
	}{
		{pixy.StateTracking, hidByteTracking},
		{pixy.StatePrivacy, hidBytePrivacy},
		{pixy.StateIdle, hidByteIdle},
		{pixy.StateOffline, hidByteIdle},
		{pixy.CameraState("unknown"), hidByteIdle},
	}
	for _, testCase := range tests {
		result := cameraHIDByte(testCase.state)
		if result != testCase.expected {
			t.Errorf(
				"cameraHIDByte(%s) = 0x%02x, want 0x%02x",
				testCase.state,
				result,
				testCase.expected,
			)
		}
	}
}

func TestTypeValidation(t *testing.T) {
	t.Parallel()

	if !pixy.AudioNC.Valid() {
		t.Error("pixy.AudioNC should be valid")
	}

	if !pixy.StateTracking.Valid() {
		t.Error("pixy.StateTracking should be valid")
	}

	if pixy.AudioMode("foo").Valid() {
		t.Error("unknown audio mode should not be valid")
	}

	if pixy.CameraState("bar").Valid() {
		t.Error("unknown camera state should not be valid")
	}
}

func TestHandleCommandAudioCycleNoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StatePrivacy, "", "")

	result := d.handleCommand(context.Background(), "audio")
	assertErrorPrefix(t, result)
}

func TestConfigPaths(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	cfg := pixy.Config{
		//nolint:exhaustruct
		StateDir:      "/tmp/test-pixyd",
		PollInterval:  pixy.DefaultPollInterval,
		DebounceCount: pixy.DefaultDebounceCount,
	}
	if cfg.StateFile() != "/tmp/test-pixyd/state.json" {
		t.Errorf("unexpected StateFile: %s", cfg.StateFile())
	}

	if cfg.SocketPath() != "/tmp/test-pixyd/control.sock" {
		t.Errorf("unexpected SocketPath: %s", cfg.SocketPath())
	}
}

func TestParseHIDResponseTracking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data     []byte
		expected pixy.CameraState
	}{
		{[]byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}, pixy.StateTracking},
		{[]byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02}, pixy.StatePrivacy},
		{[]byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00}, pixy.StateIdle},
	}
	for _, testCase := range tests {
		resp := parseHIDResponse(testCase.data)
		if !resp.Got {
			t.Fatal("expected Got=true")
		}

		if resp.Tracking != testCase.expected {
			t.Errorf(
				"tracking from %x = %s, want %s",
				testCase.data,
				resp.Tracking,
				testCase.expected,
			)
		}
	}
}

func TestParseHIDResponseAudio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data     []byte
		expected pixy.AudioMode
	}{
		{[]byte{0x09, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}, pixy.AudioNC},
		{[]byte{0x09, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02}, pixy.AudioLive},
		{[]byte{0x09, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x03}, pixy.AudioOriginal},
	}
	for _, testCase := range tests {
		resp := parseHIDResponse(testCase.data)
		if !resp.Got {
			t.Fatal("expected Got=true")
		}

		if resp.Audio != testCase.expected {
			t.Errorf("audio from %x = %s, want %s", testCase.data, resp.Audio, testCase.expected)
		}
	}
}

func TestParseHIDResponseGesture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte{0x09, 0x04, 0x02, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02, 0x01}, true},
		{[]byte{0x09, 0x04, 0x02, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02, 0x00}, false},
	}
	for _, testCase := range tests {
		assertGesture(t, parseHIDResponse(testCase.data), testCase.expected)
	}
}

func TestParseHIDResponseTooShort(t *testing.T) {
	t.Parallel()

	resp := parseHIDResponse([]byte{0x09, 0x01})
	if resp.Got {
		t.Error("expected Got=false for short response")
	}
}

func TestParseHIDResponseNil(t *testing.T) {
	t.Parallel()

	resp := parseHIDResponse(nil)
	if resp.Got {
		t.Error("expected Got=false for nil response")
	}
}

func TestHandleCommandSyncNoDevice(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(pixy.StateOffline, "", "")
	result := d.handleCommand(context.Background(), cmdSync)
	assertErrorPrefix(t, result)
}

func TestHandleCommandSyncWithDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonWithDevice(pixy.StatePrivacy)
	d.config = testConfig(t.TempDir())

	result := d.handleCommand(context.Background(), cmdSync)
	if !strings.HasPrefix(result, "synced") && !strings.Contains(result, "error") {
		t.Errorf("expected sync result, got: %s", result)
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := pixy.DefaultConfig()
	if cfg.StateDir != pixy.DefaultStateDir {
		t.Errorf("expected StateDir=%s, got %s", pixy.DefaultStateDir, cfg.StateDir)
	}

	if cfg.PollInterval != pixy.DefaultPollInterval {
		t.Errorf("expected PollInterval=%v, got %v", pixy.DefaultPollInterval, cfg.PollInterval)
	}

	if cfg.DebounceCount != pixy.DefaultDebounceCount {
		t.Errorf("expected DebounceCount=%d, got %d", pixy.DefaultDebounceCount, cfg.DebounceCount)
	}
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

func TestParseAudioMode(t *testing.T) {
	t.Parallel()

	tests := []parseTestCase[pixy.AudioMode]{
		{"nc", pixy.AudioNC, false},
		{audioModeLive, pixy.AudioLive, false},
		{audioModeOrg, pixy.AudioOriginal, false},
		{"unknown", "", true},
		{"", "", true},
	}
	runParseTests(t, "pixy.ParseAudioMode", pixy.ParseAudioMode, tests)
}

func TestParseCameraState(t *testing.T) {
	t.Parallel()

	tests := []parseTestCase[pixy.CameraState]{
		{"idle", pixy.StateIdle, false},
		{"tracking", pixy.StateTracking, false},
		{"privacy", pixy.StatePrivacy, false},
		{"offline", pixy.StateOffline, false},
		{"unknown", "", true},
		{"", "", true},
	}
	runParseTests(t, "pixy.ParseCameraState", pixy.ParseCameraState, tests)
}

func TestParseHIDResponseUnknownInterface(t *testing.T) {
	t.Parallel()

	data := []byte{0x09, 0x99, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}

	resp := parseHIDResponse(data)
	if !resp.Got {
		t.Error("expected Got=true for valid-length response with unknown interface")
	}

	assertTrackingIdle(t, resp.Tracking)
}

func TestDefaultStateValues(t *testing.T) {
	t.Parallel()

	s := pixy.DefaultState()
	if s.Camera != pixy.StatePrivacy {
		t.Errorf("expected default camera=privacy, got %s", s.Camera)
	}

	if s.Audio != pixy.AudioNC {
		t.Errorf("expected default audio=nc, got %s", s.Audio)
	}

	if s.Gesture != false {
		t.Error("expected default gesture=false")
	}

	if s.InCall != false {
		t.Error("expected default inCall=false")
	}

	if s.AutoMode != pixy.AutoFull {
		t.Error("expected default autoMode=true")
	}
}

func TestSetDeadlineError(t *testing.T) {
	t.Parallel()

	conn := &mockConn{setDeadlineErr: errors.New("deadline error")}

	err := pixy.SetDeadline(conn, time.Second)
	if err == nil {
		t.Error("expected error from SetDeadline with failing conn")
	}
}

type mockConn struct {
	setDeadlineErr error
}

func (m *mockConn) Read([]byte) (int, error)         { return 0, nil }
func (m *mockConn) Write([]byte) (int, error)        { return 0, nil }
func (m *mockConn) Close() error                     { return nil }
func (m *mockConn) LocalAddr() net.Addr              { return nil }
func (m *mockConn) RemoteAddr() net.Addr             { return nil }
func (m *mockConn) SetDeadline(time.Time) error      { return m.setDeadlineErr }
func (m *mockConn) SetReadDeadline(time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(time.Time) error { return nil }

func writeFakeFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)

	dirErr := os.MkdirAll(dir, 0o755)
	if dirErr != nil {
		t.Fatalf("mkdir %s: %v", dir, dirErr)
	}

	writeErr := os.WriteFile(path, []byte(content), 0o644)
	if writeErr != nil {
		t.Fatalf("write %s: %v", path, writeErr)
	}
}

type fakeVideoDev struct {
	name    string
	vendor  string
	product string
	index   string
}

type fakeHidrawDev struct {
	name    string
	hidID   string
	hidName string
}

func createFakeVideo4linux(t *testing.T, root string, devices []fakeVideoDev) {
	t.Helper()

	for _, dev := range devices {
		base := filepath.Join(root, dev.name)
		writeFakeFile(t, filepath.Join(base, "device/id/vendor"), dev.vendor)
		writeFakeFile(t, filepath.Join(base, "device/id/product"), dev.product)

		if dev.index != "" {
			writeFakeFile(t, filepath.Join(base, "index"), dev.index)
		}
	}
}

func createFakeHidraw(t *testing.T, root string, devices []fakeHidrawDev) {
	t.Helper()

	for _, dev := range devices {
		ueventPath := filepath.Join(root, dev.name, "device/uevent")
		content := "DRIVER=hid-generic\n"
		content += "HID_ID=" + dev.hidID + "\n"
		content += "HID_NAME=" + dev.hidName + "\n"

		writeFakeFile(t, ueventPath, content)
	}
}

const (
	pixyVendor  = "328f"
	pixyProduct = "00c0"
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

func testV4L2ProbesNothing(t *testing.T, devices []fakeVideoDev) {
	t.Helper()

	root := t.TempDir()
	if len(devices) > 0 {
		createFakeVideo4linux(t, root, devices)
	}

	result := probeVideo4linux(root)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeVideo4linux_PIXYFound(t *testing.T) {
	t.Parallel()

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, vendor: pixyVendor, product: pixyProduct, index: "0"},
		{name: testVideoDev2, vendor: pixyVendor, product: pixyProduct, index: "1"},
	})
}

func TestProbeVideo4linux_PIXYOnlyCaptureNode(t *testing.T) {
	t.Parallel()

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, vendor: pixyVendor, product: pixyProduct, index: "0"},
	})
}

func TestProbeVideo4linux_PIXYNoIndexFile(t *testing.T) {
	t.Parallel()

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, vendor: pixyVendor, product: pixyProduct, index: ""},
	})
}

func TestProbeVideo4linux_NonPIXYSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		devices []fakeVideoDev
	}{
		{
			"NoPIXY",
			[]fakeVideoDev{
				{
					name:    "video1",
					vendor:  "1511",
					product: "402d",
					index:   "0",
				},
			},
		},
		{
			"WrongVendorProduct",
			append([]fakeVideoDev{}, fakeVideoDev{
				name:    testVideoDev0,
				vendor:  "1234",
				product: "5678",
				index:   "0",
			}),
		},
		{"EmptyDir", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testV4L2ProbesNothing(t, tc.devices)
		})
	}
}

func TestProbeVideo4linux_NonexistentDir(t *testing.T) {
	t.Parallel()

	result := probeVideo4linux("/nonexistent/path/video4linux")
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeVideo4linux_OBSCamIgnored(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	obsDir := filepath.Join(root, "video1")
	writeFakeFile(t, filepath.Join(obsDir, "name"), "OBS Cam")
	writeFakeFile(t, filepath.Join(obsDir, "index"), "0")

	testV4L2ProbesPIXY(t, []fakeVideoDev{
		{name: testVideoDev0, vendor: pixyVendor, product: pixyProduct, index: "0"},
	})
}

func TestProbeVideo4linux_MetadataNodeSkipped(t *testing.T) {
	t.Parallel()

	testV4L2ProbesNothing(t, []fakeVideoDev{
		{name: testVideoDev2, vendor: pixyVendor, product: pixyProduct, index: "1"},
	})
}

func TestProbeHidraw_PIXYFound(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with a PIXY hidraw device
	root := t.TempDir()
	createFakeHidraw(t, root, []fakeHidrawDev{
		{
			name:    "hidraw7",
			hidID:   "0003:0000328F:000000C0",
			hidName: "EMEET PIXY",
		},
	})

	// When probing
	result := probeHidraw(root)

	// Then the PIXY hidraw is found
	if result != testHIDDev {
		t.Errorf("expected /dev/hidraw7, got %s", result)
	}
}

func TestProbeHidraw_NoPIXY(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with only non-PIXY hidraw devices
	root := t.TempDir()
	createFakeHidraw(t, root, []fakeHidrawDev{
		{
			name:    "hidraw0",
			hidID:   "0003:00003151:0000402D",
			hidName: "2.4G Wireless Mouse",
		},
		{
			name:    "hidraw3",
			hidID:   "0003:00001A2C:00004852",
			hidName: "SEMICO USB Gaming Keyboard",
		},
	})

	// When probing
	result := probeHidraw(root)

	// Then nothing is found
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeHidraw_EmptyDir(t *testing.T) {
	t.Parallel()

	// Given an empty hidraw sysfs directory
	root := t.TempDir()

	// When probing
	result := probeHidraw(root)

	// Then nothing is found
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeHidraw_NonexistentDir(t *testing.T) {
	t.Parallel()

	// Given a nonexistent sysfs path
	result := probeHidraw("/nonexistent/path/hidraw")

	// Then nothing is found
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeHidraw_MixedDevices(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with mouse, keyboard, and PIXY
	root := t.TempDir()
	createFakeHidraw(t, root, []fakeHidrawDev{
		{
			name:    "hidraw0",
			hidID:   "0003:00003151:0000402D",
			hidName: "2.4G Wireless Mouse",
		},
		{
			name:    "hidraw3",
			hidID:   "0003:00001A2C:00004852",
			hidName: "SEMICO USB Gaming Keyboard",
		},
		{
			name:    "hidraw7",
			hidID:   "0003:0000328F:000000C0",
			hidName: "EMEET PIXY",
		},
		{
			name:    "hidraw8",
			hidID:   "0003:0000043E:00009A39",
			hidName: "LG Electronics Inc. LG Monitor Controls",
		},
	})

	// When probing
	result := probeHidraw(root)

	// Then the PIXY is found
	if result != testHIDDev {
		t.Errorf("expected /dev/hidraw7, got %s", result)
	}
}

func TestProbeHidraw_NoUeventFile(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with a directory but no uevent file
	root := t.TempDir()

	dirErr := os.MkdirAll(filepath.Join(root, "hidraw0", "device"), 0o755)
	if dirErr != nil {
		t.Fatalf("mkdir: %v", dirErr)
	}

	// When probing
	result := probeHidraw(root)

	// Then nothing is found (graceful skip)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestProbeDevices_SetsStateToOfflineWhenNoVideo(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		initialCamera pixy.CameraState
	}{
		{"from non-offline state", pixy.StatePrivacy},
		{"from offline state", pixy.StateOffline},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := newTestDaemon(tc.initialCamera, "", "")
			d.probeDevices()

			hasDev := d.videoDev != ""
			isOffline := d.state.Camera == pixy.StateOffline
			if hasDev && isOffline {
				t.Error("camera should not be offline when video device is found")
			}
			if !hasDev && !isOffline {
				t.Errorf("expected offline when no video device, got %s", d.state.Camera)
			}
		})
	}
}

func TestProbeVideo4linux_MultipleCamerasPIXYSecond(t *testing.T) {
	t.Parallel()

	// Given a sysfs tree with another camera first, then PIXY
	root := t.TempDir()

	otherDir := filepath.Join(root, testVideoDev0)
	writeFakeFile(
		t,
		filepath.Join(otherDir, "device/modalias"),
		"usb:v1234p5678d0100dcEFdsc02dp01ic0Eisc01ip00in00",
	)
	writeFakeFile(t, filepath.Join(otherDir, "index"), "0")
	writeFakeFile(t, filepath.Join(otherDir, "name"), "Other Camera")

	createFakeVideo4linux(t, root, []fakeVideoDev{
		{
			name:    testVideoDev2,
			vendor:  pixyVendor,
			product: pixyProduct,
			index:   "0",
		},
	})

	// When probing
	result := probeVideo4linux(root)

	// Then the PIXY is found even though it's not the first device
	if result != "/dev/video2" {
		t.Errorf("expected /dev/video2, got %s", result)
	}
}

func BenchmarkParseHIDResponse(b *testing.B) {
	data := []byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}
	b.ResetTimer()
	for range b.N {
		parseHIDResponse(data)
	}
}

func BenchmarkWaybarOutput(b *testing.B) {
	d := testDaemonWithState(pixy.StateTracking, true)
	b.ResetTimer()
	for range b.N {
		d.waybarOutput()
	}
}
