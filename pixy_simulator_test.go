//go:build linux

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// pixyProtocolState is the device-side PIXY HID protocol state machine.
// It validates incoming reports against the wire protocol and maintains
// simulated device state. Shared foundation for all simulator layers.
type pixyProtocolState struct {
	mu       sync.Mutex
	tracking pixy.CameraState
	audio    pixy.AudioMode
	gesture  bool
	pending  map[byte]*pendingConfig
}

type pendingConfig struct {
	iface    byte
	modeByte byte
	setTime  time.Time
}

func newPixyProtocolState() *pixyProtocolState {
	return &pixyProtocolState{
		tracking: pixy.StateIdle,
		audio:    pixy.AudioNC,
		gesture:  false,
		pending:  make(map[byte]*pendingConfig),
	}
}

func isValidInterface(iface byte) bool {
	return iface == hidInterfaceTracking ||
		iface == hidInterfaceAudio ||
		iface == hidInterfaceGesture
}

func validateModeByte(iface, mode byte) error {
	switch iface {
	case hidInterfaceTracking:
		if mode == hidByteIdle || mode == hidByteTracking || mode == hidBytePrivacy {
			return nil
		}

		return fmt.Errorf("invalid tracking mode byte 0x%02x", mode)
	case hidInterfaceAudio:
		if mode == hidByteNC || mode == hidByteLive || mode == hidByteOriginal {
			return nil
		}

		return fmt.Errorf("invalid audio mode byte 0x%02x", mode)
	case hidInterfaceGesture:
		if mode == hidByteIdle || mode == gestureEnabledByte {
			return nil
		}

		return fmt.Errorf("invalid gesture mode byte 0x%02x", mode)
	default:
		return fmt.Errorf("unknown interface 0x%02x", iface)
	}
}

func cameraStateFromHIDByte(b byte) pixy.CameraState {
	switch b {
	case hidByteTracking:
		return pixy.StateTracking
	case hidBytePrivacy:
		return pixy.StatePrivacy
	case hidByteIdle:
		return pixy.StateIdle
	default:
		return pixy.StateIdle
	}
}

func audioModeFromHIDByte(b byte) pixy.AudioMode {
	switch b {
	case hidByteNC:
		return pixy.AudioNC
	case hidByteLive:
		return pixy.AudioLive
	case hidByteOriginal:
		return pixy.AudioOriginal
	default:
		return pixy.AudioNC
	}
}

// handleConfig validates a config report and stores it as pending.
// State is NOT changed until the corresponding commit arrives.
func (s *pixyProtocolState) handleConfig(report []byte) error {
	if len(report) < hidMinLen {
		return fmt.Errorf("config report: too short (%d bytes, need %d)", len(report), hidMinLen)
	}

	if report[0] != cameraConfigPrefix {
		return fmt.Errorf("config report: invalid prefix 0x%02x (want 0x%02x)", report[0], cameraConfigPrefix)
	}

	iface := report[1]

	if !isValidInterface(iface) {
		return fmt.Errorf("config report: unknown interface 0x%02x", iface)
	}

	if report[2] != cameraConfigMarker {
		return fmt.Errorf("config report: invalid marker at [2]: 0x%02x (want 0x%02x)", report[2], cameraConfigMarker)
	}

	if err := validateModeByte(iface, report[8]); err != nil {
		return fmt.Errorf("config report: %w", err)
	}

	s.pending[iface] = &pendingConfig{
		iface:    iface,
		modeByte: report[8],
		setTime:  time.Now(),
	}

	return nil
}

// handleCommit validates a commit report and applies the pending config.
// Returns an error if no preceding config exists for the interface.
func (s *pixyProtocolState) handleCommit(report []byte) error {
	if len(report) < 4 {
		return fmt.Errorf("commit report: too short (%d bytes, need 4)", len(report))
	}

	if report[0] != cameraConfigPrefix {
		return fmt.Errorf("commit report: invalid prefix 0x%02x", report[0])
	}

	iface := report[1]

	if !isValidInterface(iface) {
		return fmt.Errorf("commit report: unknown interface 0x%02x", iface)
	}

	if report[2] != cameraConfigMarker {
		return fmt.Errorf("commit report: invalid marker at [2]: 0x%02x", report[2])
	}

	if report[3] != iface {
		return fmt.Errorf("commit report: interface mismatch [1]=0x%02x vs [3]=0x%02x", iface, report[3])
	}

	pending, ok := s.pending[iface]

	if !ok {
		return fmt.Errorf("commit report: no pending config for interface 0x%02x", iface)
	}

	s.applyConfig(pending)

	delete(s.pending, iface)

	return nil
}

func (s *pixyProtocolState) applyConfig(pending *pendingConfig) {
	switch pending.iface {
	case hidInterfaceTracking:
		s.tracking = cameraStateFromHIDByte(pending.modeByte)
	case hidInterfaceAudio:
		s.audio = audioModeFromHIDByte(pending.modeByte)
	case hidInterfaceGesture:
		s.gesture = pending.modeByte == gestureEnabledByte
	}
}

// buildResponse generates a 64-byte protocol-valid response for a query.
// The response reflects the current committed device state (not pending).
func (s *pixyProtocolState) buildResponse(query []byte) ([]byte, error) {
	if len(query) < 2 {
		return nil, fmt.Errorf("query: too short (%d bytes)", len(query))
	}

	if query[0] != cameraConfigPrefix {
		return nil, fmt.Errorf("query: invalid prefix 0x%02x", query[0])
	}

	iface := query[1]

	if !isValidInterface(iface) {
		return nil, fmt.Errorf("query: unknown interface 0x%02x", iface)
	}

	resp := make([]byte, hidRespBufSize)
	resp[0] = cameraConfigPrefix
	resp[1] = iface

	switch iface {
	case hidInterfaceTracking:
		resp[2] = cameraConfigMarker
		resp[5] = cameraConfigMarker
		resp[7] = cameraConfigMarker
		resp[8] = cameraHIDByte(s.tracking)
	case hidInterfaceAudio:
		resp[2] = cameraConfigMarker
		resp[5] = cameraConfigMarker
		resp[7] = cameraConfigMarker
		resp[8] = audioHIDByte(s.audio)
	case hidInterfaceGesture:
		resp[2] = cameraConfigMarker
		resp[5] = cameraConfigMarker
		resp[7] = cameraConfigMarker

		if s.gesture {
			resp[hidRespBufSize-1] = gestureEnabledByte
		}
	}

	return resp, nil
}

// pixySimulator is a protocol-faithful HIDDevice implementation that replaces
// fakeHIDDevice for integration testing. It validates every byte against the
// PIXY HID wire protocol, maintains real device state, and supports failure
// injection.
type pixySimulator struct {
	state pixyProtocolState

	// Failure injection (set before calling Send/SendRecv)
	sendErr     error
	sendRecvErr error
	nilResponse bool
	corruptResp bool

	// Recording
	mu          sync.Mutex
	sentReports [][]byte
	queries     [][]byte
}

func newPixySimulator() *pixySimulator {
	return &pixySimulator{
		state: *newPixyProtocolState(),
	}
}

func (s *pixySimulator) String() string { return "pixy-simulator" }

// isCommitReport distinguishes commit reports from config reports.
// Commit reports have the interface byte repeated at position 3.
func isCommitReport(report []byte) bool {
	return len(report) >= 4 &&
		report[0] == cameraConfigPrefix &&
		report[2] == cameraConfigMarker &&
		report[3] == report[1]
}

func (s *pixySimulator) Send(report []byte) error {
	s.mu.Lock()
	s.sentReports = append(s.sentReports, append([]byte(nil), report...))
	s.mu.Unlock()

	if s.sendErr != nil {
		return s.sendErr
	}

	buf := append([]byte(nil), report...)

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	if isCommitReport(buf) {
		return s.state.handleCommit(buf)
	}

	return s.state.handleConfig(buf)
}

func (s *pixySimulator) SendRecv(_ context.Context, report []byte) ([]byte, error) {
	s.mu.Lock()
	s.queries = append(s.queries, append([]byte(nil), report...))
	s.mu.Unlock()

	if s.sendRecvErr != nil {
		return nil, s.sendRecvErr
	}

	if s.nilResponse {
		return nil, nil
	}

	buf := append([]byte(nil), report...)

	if s.corruptResp {
		garbage := make([]byte, hidRespBufSize)

		for i := range garbage {
			garbage[i] = 0xFF
		}

		return garbage, nil
	}

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	return s.state.buildResponse(buf)
}

// Tracking returns the simulator's committed tracking state.
func (s *pixySimulator) Tracking() pixy.CameraState {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	return s.state.tracking
}

// Audio returns the simulator's committed audio mode.
func (s *pixySimulator) Audio() pixy.AudioMode {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	return s.state.audio
}

// Gesture returns the simulator's committed gesture state.
func (s *pixySimulator) Gesture() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	return s.state.gesture
}

// SentReports returns all reports sent via Send (config + commit).
func (s *pixySimulator) SentReports() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sentReports
}

// Queries returns all query payloads sent via SendRecv.
func (s *pixySimulator) Queries() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queries
}

// withPixySimulator wires a protocol-faithful pixySimulator as the daemon's
// HID device. Unlike withFakeDevices, this keeps the REAL setTracking,
// setAudio, and setGesture methods — so the full setDeviceState → Send →
// protocol validation path is exercised. V4L2 and proc dependencies are
// stubbed since there is no real video device.
//
// Returns the simulator instance for state assertions and failure injection.
func withPixySimulator() (*pixySimulator, testDaemonOption) {
	sim := newPixySimulator()

	return sim, func(d *Daemon) {
		d.hidDev = sim
		d.deps.procInspector = newFakeProcInspector()
		d.deps.ueventListener = noopUeventListener{}
		d.deps.isCameraInUse = func(string) bool { return false }
		d.deps.commander = noopCommandRunner{}
		d.deps.parsePTZ = func(context.Context, string) pixy.PTZValues {
			return pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault}
		}
		d.deps.centerCamera = func(context.Context) error { return nil }
		d.deps.v4l2Set = func(context.Context, string, string, string) error { return nil }
	}
}
