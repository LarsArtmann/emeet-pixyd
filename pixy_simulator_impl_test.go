//go:build linux

package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// --- Config Report Validation ---

func TestSimulator_HandleConfig_ValidTracking(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidByteTracking))
	if err != nil {
		t.Fatalf("valid tracking config rejected: %v", err)
	}

	// State should NOT change until commit
	if s.tracking != pixy.StateIdle {
		t.Errorf("tracking changed before commit: got %s, want %s", s.tracking, pixy.StateIdle)
	}
}

func TestSimulator_HandleConfig_ValidAudio(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	err := s.handleConfig(pixyConfig(hidInterfaceAudio, hidByteLive))
	if err != nil {
		t.Fatalf("valid audio config rejected: %v", err)
	}
}

func TestSimulator_HandleConfig_ValidGesture(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	err := s.handleConfig(pixyConfig(hidInterfaceGesture, gestureEnabledByte))
	if err != nil {
		t.Fatalf("valid gesture config rejected: %v", err)
	}
}

func TestSimulator_HandleConfig_InvalidPrefix(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	report := pixyConfig(hidInterfaceTracking, hidByteTracking)
	report[0] = 0xFF

	err := s.handleConfig(report)
	if err == nil {
		t.Fatal("expected error for invalid prefix")
	}
}

func TestSimulator_HandleConfig_UnknownInterface(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	report := pixyConfig(hidInterfaceTracking, hidByteTracking)
	report[1] = 0x99

	err := s.handleConfig(report)
	if err == nil {
		t.Fatal("expected error for unknown interface")
	}
}

func TestSimulator_HandleConfig_InvalidModeByte(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	err := s.handleConfig(pixyConfig(hidInterfaceTracking, 0xFF))
	if err == nil {
		t.Fatal("expected error for invalid tracking mode byte 0xFF")
	}

	err = s.handleConfig(pixyConfig(hidInterfaceAudio, 0xFF))
	if err == nil {
		t.Fatal("expected error for invalid audio mode byte 0xFF")
	}
}

func TestSimulator_HandleConfig_TooShort(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	err := s.handleConfig([]byte{0x09, 0x01, 0x01})
	if err == nil {
		t.Fatal("expected error for short report")
	}
}

// --- Commit Report Validation + Sequencing ---

func TestSimulator_Commit_AppliesPendingConfig(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	// Config → Commit → state changes
	if err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidByteTracking)); err != nil {
		t.Fatalf("config failed: %v", err)
	}

	if s.tracking != pixy.StateIdle {
		t.Fatalf("state changed before commit: %s", s.tracking)
	}

	if err := s.handleCommit(pixyCommit(hidInterfaceTracking)); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	if s.tracking != pixy.StateTracking {
		t.Errorf("after commit: tracking=%s, want %s", s.tracking, pixy.StateTracking)
	}
}

func TestSimulator_Commit_NoPendingConfig(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	err := s.handleCommit(pixyCommit(hidInterfaceAudio))
	if err == nil {
		t.Fatal("expected error: commit without preceding config")
	}
}

func TestSimulator_QueryBeforeCommit_ReturnsOldState(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	// Set initial state via config+commit
	if err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidByteTracking)); err != nil {
		t.Fatal(err)
	}

	if err := s.handleCommit(pixyCommit(hidInterfaceTracking)); err != nil {
		t.Fatal(err)
	}

	// Now send a new config (privacy) but DON'T commit
	if err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidBytePrivacy)); err != nil {
		t.Fatal(err)
	}

	// Query should return the committed state (tracking), NOT the pending state (privacy)
	resp, err := s.buildResponse([]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	parsed := parseHIDResponse(resp)
	if !parsed.Got {
		t.Fatal("query response: Got=false")
	}

	if parsed.Tracking != pixy.StateTracking {
		t.Errorf("query before commit: tracking=%s, want %s (committed state)",
			parsed.Tracking, pixy.StateTracking)
	}
}

func TestSimulator_Commit_InterfaceMismatch(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	_ = s.handleConfig(pixyConfig(hidInterfaceTracking, hidByteTracking))

	// Commit with wrong iface byte at position 3
	badCommit := []byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, hidInterfaceAudio}

	err := s.handleCommit(badCommit)
	if err == nil {
		t.Fatal("expected error: interface mismatch in commit")
	}
}

// --- Query Response Round-Trips ---

func TestSimulator_Query_TrackingRoundTrip(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	for _, mode := range []pixy.CameraState{pixy.StateTracking, pixy.StatePrivacy, pixy.StateIdle} {
		_ = s.handleConfig(pixyConfig(hidInterfaceTracking, cameraHIDByte(mode)))
		_ = s.handleCommit(pixyCommit(hidInterfaceTracking))

		resp, err := s.buildResponse([]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
		if err != nil {
			t.Fatalf("query for %s: %v", mode, err)
		}

		parsed := parseHIDResponse(resp)

		if !parsed.Got {
			t.Errorf("query for %s: Got=false", mode)
		}

		if parsed.Tracking != mode {
			t.Errorf("round-trip: set %s, query got %s", mode, parsed.Tracking)
		}
	}
}

func TestSimulator_Query_AudioRoundTrip(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	for _, mode := range []pixy.AudioMode{pixy.AudioNC, pixy.AudioLive, pixy.AudioOriginal} {
		_ = s.handleConfig(pixyConfig(hidInterfaceAudio, audioHIDByte(mode)))
		_ = s.handleCommit(pixyCommit(hidInterfaceAudio))

		resp, err := s.buildResponse([]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04})
		if err != nil {
			t.Fatalf("query for %s: %v", mode, err)
		}

		parsed := parseHIDResponse(resp)

		if !parsed.Got {
			t.Errorf("query for %s: Got=false", mode)
		}

		if parsed.Audio != mode {
			t.Errorf("round-trip: set %s, query got %s", mode, parsed.Audio)
		}
	}
}

func TestSimulator_Query_GestureRoundTrip(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	for _, enabled := range []bool{true, false} {
		mode := byte(hidByteIdle)
		if enabled {
			mode = gestureEnabledByte
		}

		_ = s.handleConfig(pixyConfig(hidInterfaceGesture, mode))
		_ = s.handleCommit(pixyCommit(hidInterfaceGesture))

		query := []byte{
			cameraConfigPrefix, hidInterfaceGesture,
			gestureConfigMark1, gestureConfigMark2,
			0x00, cameraConfigMarker,
			0x00, cameraConfigMarker,
			gestureConfigMark3,
		}

		resp, err := s.buildResponse(query)
		if err != nil {
			t.Fatalf("query for enabled=%v: %v", enabled, err)
		}

		parsed := parseHIDResponse(resp)

		if !parsed.Got {
			t.Errorf("query for enabled=%v: Got=false", enabled)
		}

		if parsed.Gesture != enabled {
			t.Errorf("round-trip: set enabled=%v, query got %v", enabled, parsed.Gesture)
		}
	}
}

func TestSimulator_Query_ResponseParseableByDaemonParser(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()
	_ = s.handleConfig(pixyConfig(hidInterfaceTracking, hidBytePrivacy))
	_ = s.handleCommit(pixyCommit(hidInterfaceTracking))

	resp, err := s.buildResponse([]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp) != hidRespBufSize {
		t.Errorf("response length: got %d, want %d", len(resp), hidRespBufSize)
	}

	parsed := parseHIDResponse(resp)
	if !parsed.Got || parsed.Tracking != pixy.StatePrivacy {
		t.Errorf("daemon parser failed on simulator response: Got=%v Tracking=%s",
			parsed.Got, parsed.Tracking)
	}
}

// --- HIDDevice Implementation ---

func TestSimulator_Send_ConfigThenCommit(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	// Send config report
	if err := sim.Send(pixyConfig(hidInterfaceTracking, hidByteTracking)); err != nil {
		t.Fatalf("config Send: %v", err)
	}

	// State should NOT have changed yet
	if sim.Tracking() != pixy.StateIdle {
		t.Errorf("tracking changed after config only: %s", sim.Tracking())
	}

	// Send commit report
	if err := sim.Send(pixyCommit(hidInterfaceTracking)); err != nil {
		t.Fatalf("commit Send: %v", err)
	}

	if sim.Tracking() != pixy.StateTracking {
		t.Errorf("after commit: tracking=%s, want %s", sim.Tracking(), pixy.StateTracking)
	}
}

func TestSimulator_Send_RecordsAllReports(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	_ = sim.Send(pixyConfig(hidInterfaceTracking, hidByteTracking))
	_ = sim.Send(pixyCommit(hidInterfaceTracking))
	_, _ = sim.SendRecv(context.Background(),
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})

	reports := sim.SentReports()
	if len(reports) != 2 {
		t.Errorf("SentReports: got %d, want 2", len(reports))
	}

	queries := sim.Queries()
	if len(queries) != 1 {
		t.Errorf("Queries: got %d, want 1", len(queries))
	}
}

func TestSimulator_SendRecv_RoundTrip(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	// Set audio to Live via config+commit
	_ = sim.Send(pixyConfig(hidInterfaceAudio, hidByteLive))
	_ = sim.Send(pixyCommit(hidInterfaceAudio))

	// Query audio
	resp, err := sim.SendRecv(context.Background(),
		[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04})
	if err != nil {
		t.Fatalf("SendRecv: %v", err)
	}

	parsed := parseHIDResponse(resp)
	if !parsed.Got || parsed.Audio != pixy.AudioLive {
		t.Errorf("SendRecv round-trip: Got=%v Audio=%s", parsed.Got, parsed.Audio)
	}
}

// --- Failure Injection ---

func TestSimulator_SendErr(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	sim.sendErr = errors.New("device disconnected")

	err := sim.Send(pixyConfig(hidInterfaceTracking, hidByteTracking))
	if !errors.Is(err, sim.sendErr) {
		t.Errorf("Send with sendErr: got %v, want %v", err, sim.sendErr)
	}
}

func TestSimulator_SendRecvErr(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	sim.sendRecvErr = errors.New("read timeout")

	_, err := sim.SendRecv(context.Background(),
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
	if !errors.Is(err, sim.sendRecvErr) {
		t.Errorf("SendRecv with sendRecvErr: got %v, want %v", err, sim.sendRecvErr)
	}
}

func TestSimulator_NilResponse(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	sim.nilResponse = true

	resp, err := sim.SendRecv(context.Background(),
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != nil {
		t.Errorf("expected nil response, got %d bytes", len(resp))
	}
}

func TestSimulator_CorruptResponse(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	sim.corruptResp = true

	resp, err := sim.SendRecv(context.Background(),
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Corrupt response should not parse as valid
	parsed := parseHIDResponse(resp)
	if parsed.Got {
		t.Error("corrupt response parsed as valid — expected Got=false")
	}
}

// --- isCommitReport heuristic ---

func TestIsCommitReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		report []byte
		want   bool
	}{
		{"tracking commit", pixyCommit(hidInterfaceTracking), true},
		{"audio commit", pixyCommit(hidInterfaceAudio), true},
		{"gesture commit", pixyCommit(hidInterfaceGesture), true},
		{"tracking config", pixyConfig(hidInterfaceTracking, hidByteTracking), false},
		{"audio config", pixyConfig(hidInterfaceAudio, hidByteLive), false},
		{"gesture config", pixyConfig(hidInterfaceGesture, gestureEnabledByte), false},
		{"too short", []byte{0x09, 0x01}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isCommitReport(tc.report)
			if got != tc.want {
				t.Errorf("isCommitReport(%x) = %v, want %v", tc.report, got, tc.want)
			}
		})
	}
}

// --- Reverse byte mappings ---

func TestCameraStateFromHIDByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		byteVal  byte
		expected pixy.CameraState
	}{
		{hidByteTracking, pixy.StateTracking},
		{hidBytePrivacy, pixy.StatePrivacy},
		{hidByteIdle, pixy.StateIdle},
		{0xFF, pixy.StateIdle},
	}

	for _, tc := range tests {
		got := cameraStateFromHIDByte(tc.byteVal)
		if got != tc.expected {
			t.Errorf("cameraStateFromHIDByte(0x%02x) = %s, want %s",
				tc.byteVal, got, tc.expected)
		}
	}
}

func TestAudioModeFromHIDByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		byteVal  byte
		expected pixy.AudioMode
	}{
		{hidByteNC, pixy.AudioNC},
		{hidByteLive, pixy.AudioLive},
		{hidByteOriginal, pixy.AudioOriginal},
		{0xFF, pixy.AudioNC},
	}

	for _, tc := range tests {
		got := audioModeFromHIDByte(tc.byteVal)
		if got != tc.expected {
			t.Errorf("audioModeFromHIDByte(0x%02x) = %s, want %s",
				tc.byteVal, got, tc.expected)
		}
	}
}

// --- Full protocol round-trip through daemon's real methods ---

func TestSimulator_DaemonSetTracking_RoundTrip(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	err := d.deps.setTracking(context.Background(), pixy.StateTracking)
	if err != nil {
		t.Fatalf("setTracking: %v", err)
	}

	// Daemon state should be updated
	if d.state.Camera != pixy.StateTracking {
		t.Errorf("daemon state: camera=%s, want %s", d.state.Camera, pixy.StateTracking)
	}

	// Simulator should reflect the same state
	if sim.Tracking() != pixy.StateTracking {
		t.Errorf("simulator state: tracking=%s, want %s", sim.Tracking(), pixy.StateTracking)
	}

	// Should have recorded config + commit
	reports := sim.SentReports()
	if len(reports) != 2 {
		t.Fatalf("expected 2 sent reports (config+commit), got %d", len(reports))
	}

	// First report should be config, second should be commit
	if isCommitReport(reports[0]) {
		t.Error("first report should be config, not commit")
	}

	if !isCommitReport(reports[1]) {
		t.Error("second report should be commit, not config")
	}
}

func TestSimulator_DaemonSetAudio_RoundTrip(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	err := d.deps.setAudio(context.Background(), pixy.AudioLive)
	if err != nil {
		t.Fatalf("setAudio: %v", err)
	}

	if d.state.Audio != pixy.AudioLive {
		t.Errorf("daemon state: audio=%s, want %s", d.state.Audio, pixy.AudioLive)
	}

	if sim.Audio() != pixy.AudioLive {
		t.Errorf("simulator state: audio=%s, want %s", sim.Audio(), pixy.AudioLive)
	}
}

func TestSimulator_DaemonSetGesture_RoundTrip(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	err := d.deps.setGesture(context.Background(), true)
	if err != nil {
		t.Fatalf("setGesture: %v", err)
	}

	if !d.state.Gesture {
		t.Error("daemon state: gesture=false, want true")
	}

	if !sim.Gesture() {
		t.Error("simulator state: gesture=false, want true")
	}
}

func TestSimulator_SyncState_ReadsFromSimulator(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	// Change device state directly on the simulator (simulates physical button press)
	sim.state.mu.Lock()
	sim.state.tracking = pixy.StatePrivacy
	sim.state.audio = pixy.AudioOriginal
	sim.state.gesture = true
	sim.state.mu.Unlock()

	// syncState should read from simulator and update daemon state
	result := d.syncState(context.Background())
	if result.Err != nil {
		t.Fatalf("syncState failed: %v", result.Err)
	}

	if d.state.Camera != pixy.StatePrivacy {
		t.Errorf("after sync: camera=%s, want %s", d.state.Camera, pixy.StatePrivacy)
	}

	if d.state.Audio != pixy.AudioOriginal {
		t.Errorf("after sync: audio=%s, want %s", d.state.Audio, pixy.AudioOriginal)
	}

	if !d.state.Gesture {
		t.Error("after sync: gesture=false, want true")
	}
}

func TestSimulator_DaemonSetTracking_ProtocolBytesValid(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	_ = d.deps.setTracking(context.Background(), pixy.StateTracking)

	reports := sim.SentReports()
	if len(reports) < 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	// Validate config report byte layout
	configReport := reports[0]
	if configReport[0] != cameraConfigPrefix {
		t.Errorf("config[0]: got 0x%02x, want 0x%02x", configReport[0], cameraConfigPrefix)
	}

	if configReport[1] != hidInterfaceTracking {
		t.Errorf("config[1]: got 0x%02x, want 0x%02x", configReport[1], hidInterfaceTracking)
	}

	if configReport[8] != hidByteTracking {
		t.Errorf("config[8]: got 0x%02x, want 0x%02x", configReport[8], hidByteTracking)
	}

	// Validate commit report byte layout
	commitReport := reports[1]
	if commitReport[0] != cameraConfigPrefix {
		t.Errorf("commit[0]: got 0x%02x, want 0x%02x", commitReport[0], cameraConfigPrefix)
	}

	if commitReport[1] != hidInterfaceTracking {
		t.Errorf("commit[1]: got 0x%02x, want 0x%02x", commitReport[1], hidInterfaceTracking)
	}

	if commitReport[3] != hidInterfaceTracking {
		t.Errorf("commit[3]: got 0x%02x, want 0x%02x (interface repeated)", commitReport[3], hidInterfaceTracking)
	}
}

func TestSimulator_SendRecvResponse_ProducesCorrectBytes(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	_ = sim.Send(pixyConfig(hidInterfaceTracking, hidBytePrivacy))
	_ = sim.Send(pixyCommit(hidInterfaceTracking))

	resp, err := sim.SendRecv(context.Background(),
		[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
	if err != nil {
		t.Fatal(err)
	}

	// Verify response uses the same byte layout the daemon expects
	expected := make([]byte, hidRespBufSize)
	expected[0] = cameraConfigPrefix
	expected[1] = hidInterfaceTracking
	expected[2] = cameraConfigMarker
	expected[5] = cameraConfigMarker
	expected[7] = cameraConfigMarker
	expected[8] = hidBytePrivacy

	// Only check the meaningful bytes (rest is zero-padded)
	if !bytes.Equal(resp[:9], expected[:9]) {
		t.Errorf("response bytes mismatch:\n got: %x\nwant: %x", resp[:9], expected[:9])
	}
}
