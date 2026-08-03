//go:build linux

package main

import (
	"bytes"
	"context"
	"errors"
	"sync"
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

// --- Circuit Breaker Integration Tests ---

func TestSimulator_CircuitBreaker_Accumulation(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	// Pre-load threshold-1 failures so the next failure opens the circuit
	// without triggering a real sysfs re-probe (which would clear hidDev).
	d.mu.Lock()
	d.hidFailCount = hidCircuitBreakerThreshold - 1
	d.mu.Unlock()

	sim.sendErr = errors.New("device disconnected")

	// This call: config Send fails → count increments to threshold → circuit opens.
	err := d.deps.setTracking(context.Background(), pixy.StateTracking)
	if err == nil {
		t.Fatal("expected error from setTracking with sendErr")
	}

	d.mu.RLock()
	count := d.hidFailCount
	d.mu.RUnlock()

	if count != hidCircuitBreakerThreshold {
		t.Fatalf("hidFailCount: got %d, want %d", count, hidCircuitBreakerThreshold)
	}

	// Next call should be blocked by open circuit — no Send at all.
	err = d.deps.setTracking(context.Background(), pixy.StateTracking)

	if !errors.Is(err, pixy.ErrPIXYNotConnected) {
		t.Errorf("circuit-open call: got %v, want ErrPIXYNotConnected", err)
	}

	// Only 1 report from the first failed call (config only, commit never reached).
	reports := sim.SentReports()

	if len(reports) != 1 {
		t.Errorf("expected 1 report (config only), got %d", len(reports))
	}
}

func TestSimulator_CircuitBreaker_ResetOnSuccess(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	// Pre-load a partial failure count.
	d.mu.Lock()
	d.hidFailCount = 1
	d.mu.Unlock()

	// Successful call should reset count to 0.
	err := d.deps.setTracking(context.Background(), pixy.StateTracking)
	if err != nil {
		t.Fatalf("setTracking: %v", err)
	}

	d.mu.RLock()
	count := d.hidFailCount
	d.mu.RUnlock()

	if count != 0 {
		t.Errorf("hidFailCount after success: got %d, want 0", count)
	}

	if sim.Tracking() != pixy.StateTracking {
		t.Errorf("simulator state: tracking=%s, want %s", sim.Tracking(), pixy.StateTracking)
	}
}

func TestSimulator_CircuitBreaker_CommitFailure(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	commitErr := errors.New("commit write failed")
	sim.commitErr = commitErr

	// Config Send succeeds, 200ms sleep, commit Send fails.
	err := d.deps.setTracking(context.Background(), pixy.StateTracking)
	if err == nil {
		t.Fatal("expected error from commit failure")
	}

	if !errors.Is(err, commitErr) {
		t.Errorf("error: got %v, want wrapping %v", err, commitErr)
	}

	d.mu.RLock()
	count := d.hidFailCount
	d.mu.RUnlock()

	if count != 1 {
		t.Errorf("hidFailCount after commit failure: got %d, want 1", count)
	}

	// Both config and commit were attempted (2 reports recorded).
	reports := sim.SentReports()

	if len(reports) != 2 {
		t.Errorf("expected 2 reports (config+commit), got %d", len(reports))
	}

	// State should NOT have changed because the commit failed.
	if d.state.Camera != pixy.StateIdle {
		t.Errorf("daemon state changed after commit failure: %s", d.state.Camera)
	}
}

// --- 200ms Sleep Timing Verification ---

func TestSimulator_200msSleepBetweenConfigAndCommit(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	err := d.deps.setTracking(context.Background(), pixy.StateTracking)
	if err != nil {
		t.Fatalf("setTracking: %v", err)
	}

	timestamps := sim.SentTimestamps()

	if len(timestamps) != 2 {
		t.Fatalf("expected 2 timestamps, got %d", len(timestamps))
	}

	gap := timestamps[1].Sub(timestamps[0])

	// hidCommandSleepMs = 200ms; allow 10ms scheduling slack.
	if gap < 190*time.Millisecond {
		t.Errorf("gap between config and commit: %v, want >= 190ms", gap)
	}
}

// --- Direct pixyConfig() Byte Layout ---

func TestPixyConfig_ByteLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		iface    byte
		modeByte byte
	}{
		{"tracking/idle", hidInterfaceTracking, hidByteIdle},
		{"tracking/tracking", hidInterfaceTracking, hidByteTracking},
		{"tracking/privacy", hidInterfaceTracking, hidBytePrivacy},
		{"audio/nc", hidInterfaceAudio, hidByteNC},
		{"audio/live", hidInterfaceAudio, hidByteLive},
		{"audio/original", hidInterfaceAudio, hidByteOriginal},
		{"gesture/disabled", hidInterfaceGesture, hidByteIdle},
		{"gesture/enabled", hidInterfaceGesture, gestureEnabledByte},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := pixyConfig(tc.iface, tc.modeByte)

			if len(buf) != hidMinLen {
				t.Fatalf("length: got %d, want %d", len(buf), hidMinLen)
			}

			if buf[0] != cameraConfigPrefix {
				t.Errorf("[0]: got 0x%02x, want 0x%02x", buf[0], cameraConfigPrefix)
			}

			if buf[1] != tc.iface {
				t.Errorf("[1]: got 0x%02x, want 0x%02x", buf[1], tc.iface)
			}

			if buf[2] != cameraConfigMarker {
				t.Errorf("[2]: got 0x%02x, want 0x%02x", buf[2], cameraConfigMarker)
			}

			// Position 3 is always 0x00 in config reports (distinguishes from commits).
			if buf[3] != 0x00 {
				t.Errorf("[3]: got 0x%02x, want 0x00", buf[3])
			}

			if buf[5] != cameraConfigMarker {
				t.Errorf("[5]: got 0x%02x, want 0x%02x", buf[5], cameraConfigMarker)
			}

			if buf[7] != cameraConfigMarker {
				t.Errorf("[7]: got 0x%02x, want 0x%02x", buf[7], cameraConfigMarker)
			}

			if buf[8] != tc.modeByte {
				t.Errorf("[8]: got 0x%02x, want 0x%02x", buf[8], tc.modeByte)
			}
		})
	}
}

// --- Direct pixyCommit() Byte Layout ---

func TestPixyCommit_ByteLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iface byte
	}{
		{"tracking", hidInterfaceTracking},
		{"audio", hidInterfaceAudio},
		{"gesture", hidInterfaceGesture},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := pixyCommit(tc.iface)

			if len(buf) != 4 {
				t.Fatalf("length: got %d, want 4", len(buf))
			}

			if buf[0] != cameraConfigPrefix {
				t.Errorf("[0]: got 0x%02x, want 0x%02x", buf[0], cameraConfigPrefix)
			}

			if buf[1] != tc.iface {
				t.Errorf("[1]: got 0x%02x, want 0x%02x", buf[1], tc.iface)
			}

			if buf[2] != cameraConfigMarker {
				t.Errorf("[2]: got 0x%02x, want 0x%02x", buf[2], cameraConfigMarker)
			}

			// Interface byte is repeated at position 3 in commit reports.
			if buf[3] != tc.iface {
				t.Errorf("[3]: got 0x%02x, want 0x%02x (interface repeated)", buf[3], tc.iface)
			}
		})
	}
}

// --- Audio / Gesture Protocol Byte Layout (daemon integration) ---

func TestSimulator_DaemonSetAudio_ProtocolBytesValid(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	_ = d.deps.setAudio(context.Background(), pixy.AudioLive)

	reports := sim.SentReports()

	if len(reports) < 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	configReport := reports[0]

	if configReport[1] != hidInterfaceAudio {
		t.Errorf("config[1]: got 0x%02x, want 0x%02x", configReport[1], hidInterfaceAudio)
	}

	if configReport[8] != hidByteLive {
		t.Errorf("config[8]: got 0x%02x, want 0x%02x", configReport[8], hidByteLive)
	}

	commitReport := reports[1]

	if commitReport[1] != hidInterfaceAudio {
		t.Errorf("commit[1]: got 0x%02x, want 0x%02x", commitReport[1], hidInterfaceAudio)
	}

	if commitReport[3] != hidInterfaceAudio {
		t.Errorf("commit[3]: got 0x%02x, want 0x%02x", commitReport[3], hidInterfaceAudio)
	}
}

func TestSimulator_DaemonSetGesture_ProtocolBytesValid(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	_ = d.deps.setGesture(context.Background(), true)

	reports := sim.SentReports()

	if len(reports) < 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	configReport := reports[0]

	if configReport[1] != hidInterfaceGesture {
		t.Errorf("config[1]: got 0x%02x, want 0x%02x", configReport[1], hidInterfaceGesture)
	}

	if configReport[8] != gestureEnabledByte {
		t.Errorf("config[8]: got 0x%02x, want 0x%02x (gestureEnabledByte)",
			configReport[8], gestureEnabledByte)
	}

	commitReport := reports[1]

	if commitReport[1] != hidInterfaceGesture {
		t.Errorf("commit[1]: got 0x%02x, want 0x%02x", commitReport[1], hidInterfaceGesture)
	}

	if commitReport[3] != hidInterfaceGesture {
		t.Errorf("commit[3]: got 0x%02x, want 0x%02x", commitReport[3], hidInterfaceGesture)
	}
}

// --- Circuit Breaker: Real Accumulation via Commit Failures ---
//
// The commit failure path (device.go:55-65) is the ONLY path that can naturally
// accumulate hidFailCount to the threshold: it increments without calling
// probeDevices(), so hidDev stays intact across failures. Config Send failures
// trigger a re-probe that either resets the counter (device found) or nils
// hidDev (device not found) — neither can reach threshold naturally.

func TestSimulator_CircuitBreaker_RealAccumulationViaCommitFailures(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	commitErr := errors.New("commit write I/O error")
	sim.commitErr = commitErr

	// Drive exactly threshold failures. Config succeeds, 200ms sleep, commit fails.
	// hidDev stays intact because commit failures don't trigger re-probe.
	for i := 1; i <= hidCircuitBreakerThreshold; i++ {
		err := d.deps.setTracking(context.Background(), pixy.StateTracking)
		if err == nil {
			t.Fatalf("call %d: expected commit error, got nil", i)
		}

		if !errors.Is(err, commitErr) {
			t.Fatalf("call %d: error %v, want wrapping %v", i, err, commitErr)
		}

		d.mu.RLock()
		count := d.hidFailCount
		d.mu.RUnlock()

		if count != i {
			t.Fatalf("call %d: hidFailCount=%d, want %d", i, count, i)
		}
	}

	// Circuit is now open — next call returns ErrPIXYNotConnected without Send.
	err := d.deps.setTracking(context.Background(), pixy.StateTracking)

	if !errors.Is(err, pixy.ErrPIXYNotConnected) {
		t.Errorf("circuit-open call: got %v, want ErrPIXYNotConnected", err)
	}

	// Each failed call sent config + commit (2 reports). The circuit-blocked
	// call sent nothing.
	reports := sim.SentReports()

	expected := hidCircuitBreakerThreshold * 2

	if len(reports) != expected {
		t.Errorf("reports: got %d, want %d (2 per failed call, 0 for circuit-blocked)", len(reports), expected)
	}
}

// --- Context Cancellation During 200ms Sleep ---

func TestSimulator_ContextCancellationDuringSleep(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 50ms — config Send will have succeeded, but the 200ms
	// sleep is still in progress. The select in setDeviceState should
	// catch ctx.Done() and return ctx.Err().
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := d.deps.setTracking(ctx, pixy.StateTracking)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error: got %v, want wrapping %v", err, context.Canceled)
	}

	// Only config was sent; commit never reached because ctx was cancelled.
	reports := sim.SentReports()

	if len(reports) != 1 {
		t.Errorf("expected 1 report (config only), got %d", len(reports))
	}

	// Cancellation is NOT a HID failure — counter should stay at 0.
	d.mu.RLock()
	count := d.hidFailCount
	d.mu.RUnlock()

	if count != 0 {
		t.Errorf("hidFailCount: got %d, want 0 (cancellation is not a failure)", count)
	}
}

// --- Concurrent Stress Test ---

func TestSimulator_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	ctx := context.Background()

	const (
		goroutines = 10
		iterations = 20
	)

	var waitGroup sync.WaitGroup

	for range goroutines {
		waitGroup.Go(func() {
			for range iterations {
				_ = sim.Send(pixyConfig(hidInterfaceTracking, hidByteTracking))
				_ = sim.Send(pixyCommit(hidInterfaceTracking))
				_, _ = sim.SendRecv(ctx,
					[]byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01})
			}
		})
	}

	waitGroup.Wait()

	// All reports should be recorded without panic or deadlock.
	reports := sim.SentReports()
	queries := sim.Queries()

	expectedReports := goroutines * iterations * 2 // config + commit per iteration
	expectedQueries := goroutines * iterations

	if len(reports) != expectedReports {
		t.Errorf("reports: got %d, want %d", len(reports), expectedReports)
	}

	if len(queries) != expectedQueries {
		t.Errorf("queries: got %d, want %d", len(queries), expectedQueries)
	}

	// Final state should be consistent — last commit wins.
	if sim.Tracking() != pixy.StateTracking {
		t.Errorf("tracking after concurrent access: %s, want %s", sim.Tracking(), pixy.StateTracking)
	}
}

// --- buildResponse Byte Layout Table Test ---

func TestBuildResponse_ByteLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(s *pixyProtocolState)
		query     []byte
		wantIface byte
		wantMode  byte // resp[8] for tracking/audio; 0 for gesture (checked separately)
	}{
		{
			"tracking/idle",
			func(s *pixyProtocolState) { s.tracking = pixy.StateIdle },
			[]byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, 0x01},
			hidInterfaceTracking,
			hidByteIdle,
		},
		{
			"tracking/tracking",
			func(s *pixyProtocolState) { s.tracking = pixy.StateTracking },
			[]byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, 0x01},
			hidInterfaceTracking,
			hidByteTracking,
		},
		{
			"tracking/privacy",
			func(s *pixyProtocolState) { s.tracking = pixy.StatePrivacy },
			[]byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, 0x01},
			hidInterfaceTracking,
			hidBytePrivacy,
		},
		{
			"audio/nc",
			func(s *pixyProtocolState) { s.audio = pixy.AudioNC },
			[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04},
			hidInterfaceAudio,
			hidByteNC,
		},
		{
			"audio/live",
			func(s *pixyProtocolState) { s.audio = pixy.AudioLive },
			[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04},
			hidInterfaceAudio,
			hidByteLive,
		},
		{
			"audio/original",
			func(s *pixyProtocolState) { s.audio = pixy.AudioOriginal },
			[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04},
			hidInterfaceAudio,
			hidByteOriginal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newPixyProtocolState()
			s.mu.Lock()
			tc.setup(s)
			s.mu.Unlock()

			resp, err := s.buildResponse(tc.query)
			if err != nil {
				t.Fatal(err)
			}

			if len(resp) != hidRespBufSize {
				t.Fatalf("response length: got %d, want %d", len(resp), hidRespBufSize)
			}

			if resp[0] != cameraConfigPrefix {
				t.Errorf("[0]: got 0x%02x, want 0x%02x", resp[0], cameraConfigPrefix)
			}

			if resp[1] != tc.wantIface {
				t.Errorf("[1]: got 0x%02x, want 0x%02x", resp[1], tc.wantIface)
			}

			if resp[2] != cameraConfigMarker {
				t.Errorf("[2]: got 0x%02x, want 0x%02x", resp[2], cameraConfigMarker)
			}

			if resp[5] != cameraConfigMarker {
				t.Errorf("[5]: got 0x%02x, want 0x%02x", resp[5], cameraConfigMarker)
			}

			if resp[7] != cameraConfigMarker {
				t.Errorf("[7]: got 0x%02x, want 0x%02x", resp[7], cameraConfigMarker)
			}

			if resp[8] != tc.wantMode {
				t.Errorf("[8]: got 0x%02x, want 0x%02x", resp[8], tc.wantMode)
			}
		})
	}
}

func TestBuildResponse_GestureLastByte(t *testing.T) {
	t.Parallel()

	query := []byte{cameraConfigPrefix, hidInterfaceGesture, gestureConfigMark1, gestureConfigMark2}

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()

		s := newPixyProtocolState()
		s.mu.Lock()
		s.gesture = true
		s.mu.Unlock()

		resp, err := s.buildResponse(query)
		if err != nil {
			t.Fatal(err)
		}

		if resp[hidRespBufSize-1] != gestureEnabledByte {
			t.Errorf("resp[last]: got 0x%02x, want 0x%02x", resp[hidRespBufSize-1], gestureEnabledByte)
		}

		// Verify markers are present in the response
		if resp[0] != cameraConfigPrefix {
			t.Errorf("[0]: got 0x%02x", resp[0])
		}

		if resp[1] != hidInterfaceGesture {
			t.Errorf("[1]: got 0x%02x, want 0x%02x", resp[1], hidInterfaceGesture)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()

		s := newPixyProtocolState()
		s.mu.Lock()
		s.gesture = false
		s.mu.Unlock()

		resp, err := s.buildResponse(query)
		if err != nil {
			t.Fatal(err)
		}

		if resp[hidRespBufSize-1] != 0x00 {
			t.Errorf("resp[last]: got 0x%02x, want 0x00 (disabled)", resp[hidRespBufSize-1])
		}
	})
}

// --- queryHIDState[T] Generic Wrapper End-to-End ---

func TestSimulator_QueryHIDState_GenericWrapper(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()

	// Set tracking to Privacy via protocol round-trip.
	_ = sim.Send(pixyConfig(hidInterfaceTracking, hidBytePrivacy))
	_ = sim.Send(pixyCommit(hidInterfaceTracking))

	ctx := context.Background()
	query := []byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, 0x01}

	// Type inference: queryHIDState[pixy.CameraState]
	result, err := queryHIDState(
		ctx, sim, query,
		func(r hidResponse) pixy.CameraState { return r.Tracking },
	)
	if err != nil {
		t.Fatal(err)
	}

	if result != pixy.StatePrivacy {
		t.Errorf("queryHIDState tracking: got %s, want %s", result, pixy.StatePrivacy)
	}

	// Also verify the audio query path through the generic wrapper.
	_ = sim.Send(pixyConfig(hidInterfaceAudio, hidByteOriginal))
	_ = sim.Send(pixyCommit(hidInterfaceAudio))

	audioResult, err := queryHIDState(
		ctx, sim,
		[]byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, 0x04},
		func(r hidResponse) pixy.AudioMode { return r.Audio },
	)
	if err != nil {
		t.Fatal(err)
	}

	if audioResult != pixy.AudioOriginal {
		t.Errorf("queryHIDState audio: got %s, want %s", audioResult, pixy.AudioOriginal)
	}

	// Gesture query through the wrapper.
	_ = sim.Send(pixyConfig(hidInterfaceGesture, gestureEnabledByte))
	_ = sim.Send(pixyCommit(hidInterfaceGesture))

	gestureResult, err := queryHIDState(
		ctx, sim,
		[]byte{
			cameraConfigPrefix, hidInterfaceGesture,
			gestureConfigMark1, gestureConfigMark2,
			0x00, cameraConfigMarker,
			0x00, cameraConfigMarker,
			gestureConfigMark3,
		},
		func(r hidResponse) bool { return r.Gesture },
	)
	if err != nil {
		t.Fatal(err)
	}

	if !gestureResult {
		t.Error("queryHIDState gesture: got false, want true")
	}
}

func TestSimulator_QueryHIDState_NilResponse(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	sim.nilResponse = true

	_, err := queryHIDState(
		context.Background(), sim,
		[]byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, 0x01},
		func(r hidResponse) pixy.CameraState { return r.Tracking },
	)

	if !errors.Is(err, errNoHIDResponse) {
		t.Errorf("error: got %v, want wrapping %v", err, errNoHIDResponse)
	}
}

func TestSimulator_QueryHIDState_CorruptResponse(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	sim.corruptResp = true

	_, err := queryHIDState(
		context.Background(), sim,
		[]byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, 0x01},
		func(r hidResponse) pixy.CameraState { return r.Tracking },
	)

	if !errors.Is(err, errUnrecognizedHID) {
		t.Errorf("error: got %v, want wrapping %v", err, errUnrecognizedHID)
	}
}

// --- Write-Then-Read syncState Round-Trip ---

func TestSimulator_SyncState_WriteThenReadRoundTrip(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateIdle, testVideoDev, testHIDDev, opt)

	// Step 1: Set tracking to Privacy via daemon (write path through protocol).
	err := d.deps.setTracking(context.Background(), pixy.StatePrivacy)
	if err != nil {
		t.Fatalf("setTracking: %v", err)
	}

	if sim.Tracking() != pixy.StatePrivacy {
		t.Fatalf("simulator after write: %s, want %s", sim.Tracking(), pixy.StatePrivacy)
	}

	// Step 2: Simulate physical state change on the device (button press).
	sim.state.mu.Lock()
	sim.state.tracking = pixy.StateTracking
	sim.state.audio = pixy.AudioOriginal
	sim.state.gesture = true
	sim.state.mu.Unlock()

	// Step 3: Sync — daemon reads from simulator and detects the drift.
	result := d.syncState(context.Background())
	if result.Err != nil {
		t.Fatalf("syncState: %v", result.Err)
	}

	// Step 4: Daemon state should reflect the simulator's new state.
	if d.state.Camera != pixy.StateTracking {
		t.Errorf("camera after sync: %s, want %s", d.state.Camera, pixy.StateTracking)
	}

	if d.state.Audio != pixy.AudioOriginal {
		t.Errorf("audio after sync: %s, want %s", d.state.Audio, pixy.AudioOriginal)
	}

	if !d.state.Gesture {
		t.Error("gesture after sync: false, want true")
	}
}

// --- Multi-Interface Pending ---

// TestSimulator_MultiInterfacePending verifies that the pending map tracks
// configs for all three interfaces independently. Configs for tracking, audio,
// and gesture can all be queued before any commit; each commit applies only
// its own interface without interfering with the others.
func TestSimulator_MultiInterfacePending(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	// Queue configs for all three interfaces without committing.
	if err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidByteTracking)); err != nil {
		t.Fatalf("config tracking: %v", err)
	}

	if err := s.handleConfig(pixyConfig(hidInterfaceAudio, hidByteLive)); err != nil {
		t.Fatalf("config audio: %v", err)
	}

	if err := s.handleConfig(pixyConfig(hidInterfaceGesture, gestureEnabledByte)); err != nil {
		t.Fatalf("config gesture: %v", err)
	}

	// State must not change until commits arrive.
	if s.tracking != pixy.StateIdle {
		t.Errorf("tracking before commit: %s, want idle", s.tracking)
	}

	if s.audio != pixy.AudioNC {
		t.Errorf("audio before commit: %s, want nc", s.audio)
	}

	if s.gesture {
		t.Error("gesture before commit: true, want false")
	}

	// Commit only tracking — audio and gesture pending must survive.
	if err := s.handleCommit(pixyCommit(hidInterfaceTracking)); err != nil {
		t.Fatalf("commit tracking: %v", err)
	}

	if s.tracking != pixy.StateTracking {
		t.Errorf("tracking after commit: %s, want tracking", s.tracking)
	}

	if s.audio != pixy.AudioNC {
		t.Errorf("audio leaked after tracking commit: %s, want nc", s.audio)
	}

	if s.gesture {
		t.Error("gesture leaked after tracking commit: true, want false")
	}

	// Commit audio.
	if err := s.handleCommit(pixyCommit(hidInterfaceAudio)); err != nil {
		t.Fatalf("commit audio: %v", err)
	}

	if s.audio != pixy.AudioLive {
		t.Errorf("audio after commit: %s, want live", s.audio)
	}

	// Commit gesture.
	if err := s.handleCommit(pixyCommit(hidInterfaceGesture)); err != nil {
		t.Fatalf("commit gesture: %v", err)
	}

	if !s.gesture {
		t.Error("gesture after commit: false, want true")
	}

	// All pending entries must be cleared.
	if len(s.pending) != 0 {
		t.Errorf("pending map not empty after all commits: %d entries", len(s.pending))
	}
}

// TestSimulator_OverwritePendingConfig verifies that sending a second config
// for the same interface (without an intervening commit) overwrites the first
// pending config. The commit then applies the latest value.
func TestSimulator_OverwritePendingConfig(t *testing.T) {
	t.Parallel()

	s := newPixyProtocolState()

	// First config: tracking mode.
	if err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidByteTracking)); err != nil {
		t.Fatalf("first config: %v", err)
	}

	// Second config for the same interface: privacy mode (overwrites pending).
	if err := s.handleConfig(pixyConfig(hidInterfaceTracking, hidBytePrivacy)); err != nil {
		t.Fatalf("second config: %v", err)
	}

	// Commit should apply privacy (the latest pending), not tracking.
	if err := s.handleCommit(pixyCommit(hidInterfaceTracking)); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if s.tracking != pixy.StatePrivacy {
		t.Errorf("after overwrite+commit: tracking=%s, want privacy", s.tracking)
	}
}
