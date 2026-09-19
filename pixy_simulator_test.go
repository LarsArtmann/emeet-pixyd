//go:build linux

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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

	// V2 command surface state (official protocol families). Motor axes are
	// indexed by pixy.MotorType. Battery/charge are the simulator's fixed
	// fixture values; read paths assert against them.
	motorSpeed   [3]float32
	motorPos     [3]float32
	motorLimit   float32
	motorPresets map[byte]v2MotorPreset
	// presetFullResponses flips preset-slot GET answers from the evidenced
	// Beta.25 mode-only shape to the full SET-echo shape (slot+mode+PTZ),
	// letting tests pin both parser paths until #166 shows which the wired
	// firmware answers.
	presetFullResponses bool
	targetTrack         v2TargetTrack
	batteryLevel        byte
	chargeSta           byte
	funcSta             uint32
	serialNumber        string
	firmwareVer         uint16
}

// v2MotorPreset is one modeled hardware motor preset slot: its position-mode
// byte and, when occupied (pixy.MotorPresetPositioned), the stored position.
type v2MotorPreset struct {
	mode byte
	pos  [3]float32
}

type v2TargetTrack struct {
	mode byte
	args [3]float32
}

type pendingConfig struct {
	iface    byte
	modeByte byte
	setTime  time.Time
}

func newPixyProtocolState() *pixyProtocolState {
	return &pixyProtocolState{
		tracking:     pixy.StateIdle,
		audio:        pixy.AudioNC,
		gesture:      false,
		pending:      make(map[byte]*pendingConfig),
		motorLimit:   100.0,
		motorPresets: make(map[byte]v2MotorPreset),
		batteryLevel: 87,
		serialNumber: "PIXY-SIM-0001",
		firmwareVer:  0x0203,
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
	state *pixyProtocolState

	// Failure injection (set before calling Send/SendRecv)
	sendErr     error
	commitErr   error // fails only commit reports (config still succeeds)
	sendRecvErr error
	nilResponse bool
	corruptResp bool

	// Recording
	mu             sync.Mutex
	sentReports    [][]byte
	sentTimestamps []time.Time
	queries        [][]byte
}

func newPixySimulator() *pixySimulator {
	return &pixySimulator{
		state: newPixyProtocolState(),
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

// v2SetSpecs maps known V2 SET heads to their payload validation. Motor heads
// are registered in both routings (logical 0x03 and motor-MCU 0x63).
var v2SetSpecs = buildV2SetSpecs()

// v2GetHeads maps known V2 GET heads to response payload builders.
var v2GetHeads = buildV2GetHeads()

func buildV2SetSpecs() map[[4]byte]v2SetSpec {
	speedSpec := v2SetSpec{name: "SetMotorSpeed", payloadLen: 5, validate: validateMotorPayload}
	posSpec := v2SetSpec{name: "SetMotorPos", payloadLen: 5, validate: validateMotorPayload}
	presetSpec := v2SetSpec{name: "SetMotorPresetPos", payloadLen: 1, validate: func(payload []byte) error {
		if payload[0] == 0 {
			return errPresetSlotZero
		}

		return nil
	}}

	presetModeSpec := v2SetSpec{name: "SetMotorPresetPosMode", payloadLen: 2, validate: func(payload []byte) error {
		if payload[0] == 0 {
			return errPresetSlotZero
		}

		return nil
	}}

	trackSpec := v2SetSpec{name: "SetTargetTrack", payloadLen: 13, validate: func(payload []byte) error {
		if !pixy.TargetTrackMode(payload[0]).Valid() {
			return fmt.Errorf("target track mode %d out of range (None/Face/HalfBody/FullBody = 0..3): %w",
				payload[0], ErrTrackModeRange)
		}

		return nil
	}}

	specs := make(map[[4]byte]v2SetSpec)

	for _, spec := range []struct {
		head pixy.V2Head
		def  v2SetSpec
	}{
		{pixy.V2SetMotorSpeed, speedSpec},
		{pixy.V2SetMotorPos, posSpec},
		{pixy.V2SetMotorPresetPos, presetSpec},
		{pixy.V2SetMotorPresetPosMode, presetModeSpec},
		{pixy.V2SetTargetTrack, trackSpec},
	} {
		specs[spec.head] = spec.def

		// The motor-MCU iface substitution is a MOTOR-device routing rule
		// (mergeType(3,3)); applying it to other devices would collide —
		// e.g. SetMotorPos(0x63) and SetTargetTrack(0x63) both become
		// [09 63 01 01].
		if spec.head[1] == pixy.V2DevMotor {
			specs[spec.head.WithIface(pixy.MotorMCUIface)] = spec.def
		}
	}

	return specs
}

func buildV2GetHeads() map[[4]byte]bool {
	heads := make(map[[4]byte]bool)

	for _, head := range []pixy.V2Head{
		pixy.V2GetMotorSpeed, pixy.V2GetMotorPos, pixy.V2GetMotorPresetPosMode,
		pixy.V2GetTargetTrack, pixy.V2GetBatteryLevel, pixy.V2GetChargeSta,
		pixy.V2GetFuncSta, pixy.V2GetSN, pixy.V2GetVer, pixy.V2GetDeviceVer,
		pixy.V2GetDeviceMode,
	} {
		heads[head] = true

		// 0x63 substitution is motor-device routing only (see buildV2SetSpecs).
		if head[1] == pixy.V2DevMotor {
			heads[head.WithIface(pixy.MotorMCUIface)] = true
		}
	}

	return heads
}

// errPresetSlotZero and ErrTrackModeRange are the simulator's V2 validation
// errors (wrapped for dynamic context at the call site).
var (
	errPresetSlotZero = errors.New("preset slot 0 is invalid (slots are 1-based)")
	// ErrTrackModeRange is returned when a target-track mode byte exceeds the
	// four official UI variants (statically evidenced, Beta.25 UI enum
	// registration; hardware confirmation pending, #166).
	ErrTrackModeRange = errors.New("track mode out of range")
)

type v2SetSpec struct {
	name       string
	payloadLen int
	validate   func(payload []byte) error
}

func validateMotorPayload(payload []byte) error {
	if !pixy.MotorType(payload[0]).Valid() {
		return fmt.Errorf("invalid motor type %d: %w", payload[0], pixy.ErrInvalidMotorType)
	}

	return nil
}

// isV2SetReport reports whether a Send report is a known V2 SET command.
// Checked BEFORE isCommitReport: motor SET heads collide with the commit
// shape ([2]=0x01, [3]==[1]). Head-only reports (len 4) with a registered V2
// SET head are also routed here so a missing payload is rejected as a V2
// payload error instead of being misread as a legacy commit — no registered
// V2 SET head is a valid legacy commit target (their ifaces are not
// tracking/audio/gesture).
func isV2SetReport(report []byte) bool {
	if len(report) < 4 || report[0] != pixy.V2ReportPrefix {
		return false
	}

	var key [4]byte
	copy(key[:], report[:4])

	_, ok := v2SetSpecs[key]

	return ok
}

// isV2GetQuery reports whether a SendRecv query is a known V2 GET head.
// GET queries carry the bare 4-byte head, plus one motorType byte for the
// per-axis motor queries (framing assumption, pinned at hardware verify).
func isV2GetQuery(query []byte) bool {
	if len(query) < 4 || query[0] != pixy.V2ReportPrefix {
		return false
	}

	var key [4]byte
	copy(key[:], query[:4])

	return v2GetHeads[key]
}

// handleV2Set validates a V2 SET report against its spec and applies it.
// V2 SETs are single reports (head + payload) — there is no config→commit
// two-step and no 200ms sleep in this family.
func (s *pixyProtocolState) handleV2Set(report []byte) error {
	var key [4]byte
	copy(key[:], report[:4])
	spec := v2SetSpecs[key]

	if len(report) < 4+spec.payloadLen {
		return fmt.Errorf("%s: payload too short (%d bytes, need %d)", spec.name, len(report)-4, spec.payloadLen)
	}

	payload := report[4:]
	if err := spec.validate(payload[:spec.payloadLen]); err != nil {
		return fmt.Errorf("%s: %w", spec.name, err)
	}

	switch spec.name {
	case "SetMotorSpeed":
		s.motorSpeed[payload[0]] = f32LE(payload[1:])
	case "SetMotorPos":
		s.motorPos[payload[0]] = f32LE(payload[1:])
	case "SetTargetTrack":
		s.targetTrack = v2TargetTrack{
			mode: payload[0],
			args: [3]float32{f32LE(payload[1:]), f32LE(payload[5:]), f32LE(payload[9:])},
		}
	case "SetMotorPresetPos":
		// Saves the CURRENT position into the slot: it becomes occupied
		// (mode=1) with the live pan/tilt/zoom coordinates.
		s.motorPresets[payload[0]] = v2MotorPreset{
			mode: pixy.MotorPresetPositioned,
			pos:  [3]float32{s.motorPos[0], s.motorPos[1], s.motorPos[2]},
		}
	case "SetMotorPresetPosMode":
		// [slot][mode] flips the slot's mode byte; a slot becoming occupied
		// without stored coordinates snapshots the current position.
		entry := s.motorPresets[payload[0]]
		entry.mode = payload[1]

		if entry.mode == pixy.MotorPresetPositioned && entry.pos == [3]float32{} {
			entry.pos = [3]float32{s.motorPos[0], s.motorPos[1], s.motorPos[2]}
		}

		s.motorPresets[payload[0]] = entry
	}

	return nil
}

// buildV2Response generates a protocol-shaped response for a V2 GET query.
// EVIDENCE (static, Beta.25 x64 parser disassembly; see
// pixy.V2ResponsePayloadOffset): the response echoes the 4-byte request head,
// carries a reserved dword at bytes 4..7 (modeled as zeros), and starts the
// payload at offset 8. If hardware (#166) shows different framing, this
// builder and the internal/pixy parsers change together via the shared
// constant.
func (s *pixyProtocolState) buildV2Response(query []byte) []byte {
	var key [4]byte
	copy(key[:], query[:4])

	resp := make([]byte, hidRespBufSize)
	copy(resp, key[:])
	body := resp[pixy.V2ResponsePayloadOffset:]

	speedKey := [4]byte(pixy.V2GetMotorSpeed)
	speedKey63 := [4]byte(pixy.V2GetMotorSpeed.WithIface(pixy.MotorMCUIface))
	posKey := [4]byte(pixy.V2GetMotorPos)
	posKey63 := [4]byte(pixy.V2GetMotorPos.WithIface(pixy.MotorMCUIface))
	presetModeKey := [4]byte(pixy.V2GetMotorPresetPosMode)
	presetModeKey63 := [4]byte(pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface))

	switch key {
	case speedKey, speedKey63:
		motor := queryMotorType(query)
		body[0] = byte(motor)
		putF32LE(body[1:], s.motorSpeed[motor])
		putF32LE(body[5:], s.motorLimit)
	case posKey, posKey63:
		motor := queryMotorType(query)
		body[0] = byte(motor)
		putF32LE(body[1:], s.motorPos[motor])
	case presetModeKey, presetModeKey63:
		// The queried slot byte rides after the head (like the motorType byte
		// of the per-axis motor GETs). Two evidenced shapes: mode-only is the
		// Beta.25 GET parser's single byte; the full shape mirrors the
		// SET_MOTOR_PRESET_POS_MODE echo ([slot][mode][pan][tilt][zoom],
		// floats only when occupied).
		slot := byte(0)
		if len(query) > 4 {
			slot = query[4]
		}

		entry := s.motorPresets[slot]

		if s.presetFullResponses {
			body[0] = slot
			body[1] = entry.mode

			if entry.mode == pixy.MotorPresetPositioned {
				putF32LE(body[2:], entry.pos[0])
				putF32LE(body[6:], entry.pos[1])
				putF32LE(body[10:], entry.pos[2])
			}
		} else {
			body[0] = entry.mode
		}
	case [4]byte(pixy.V2GetTargetTrack):
		body[0] = s.targetTrack.mode
		putF32LE(body[1:], s.targetTrack.args[0])
		putF32LE(body[5:], s.targetTrack.args[1])
		putF32LE(body[9:], s.targetTrack.args[2])
	case [4]byte(pixy.V2GetBatteryLevel):
		body[0] = s.batteryLevel
	case [4]byte(pixy.V2GetChargeSta):
		body[0] = s.chargeSta
	case [4]byte(pixy.V2GetFuncSta):
		binary.LittleEndian.PutUint32(body, s.funcSta)
	case [4]byte(pixy.V2GetSN):
		copy(body, s.serialNumber)
	case [4]byte(pixy.V2GetVer):
		binary.LittleEndian.PutUint16(body, s.firmwareVer)
	case [4]byte(pixy.V2GetDeviceVer):
		binary.LittleEndian.PutUint16(body, s.firmwareVer)
	case [4]byte(pixy.V2GetDeviceMode):
		body[0] = cameraHIDByte(s.tracking)
	default:
		// Unknown-but-valid V2 head: head echo only (empty payload ACK).
	}

	return resp
}

// queryMotorType reads the per-axis motor byte from a V2 GET query,
// defaulting to pan for bare-head queries.
func queryMotorType(query []byte) pixy.MotorType {
	if len(query) < 5 {
		return pixy.MotorPan
	}

	motor := pixy.MotorType(query[4])

	if !motor.Valid() {
		return pixy.MotorPan
	}

	return motor
}

func f32LE(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

func putF32LE(b []byte, f float32) {
	binary.LittleEndian.PutUint32(b, math.Float32bits(f))
}

func (s *pixySimulator) Send(report []byte) error {
	s.mu.Lock()
	s.sentReports = append(s.sentReports, append([]byte(nil), report...))
	s.sentTimestamps = append(s.sentTimestamps, time.Now())
	s.mu.Unlock()

	if s.commitErr != nil && !isV2SetReport(report) && isCommitReport(report) {
		return s.commitErr
	}

	if s.sendErr != nil {
		return s.sendErr
	}

	buf := append([]byte(nil), report...)

	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	// V2 SETs are single-report commands; checked before the commit shape
	// because motor SET heads collide with it (see isV2SetReport).
	if isV2SetReport(buf) {
		return s.state.handleV2Set(buf)
	}

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

	if isV2GetQuery(buf) {
		return s.state.buildV2Response(buf), nil
	}

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

// MotorSpeed returns the committed speed for a motor axis.
func (s *pixySimulator) MotorSpeed(mt pixy.MotorType) float32 {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	return s.state.motorSpeed[mt]
}

// TargetTrack returns the committed tracking-variant mode and its arguments.
func (s *pixySimulator) TargetTrack() (byte, [3]float32) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	return s.state.targetTrack.mode, s.state.targetTrack.args
}

// MotorPreset returns the modeled contents of a hardware preset slot.
func (s *pixySimulator) MotorPreset(slot byte) (v2MotorPreset, bool) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	entry, ok := s.state.motorPresets[slot]

	return entry, ok
}

// SentReports returns all reports sent via Send (config + commit).
func (s *pixySimulator) SentReports() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sentReports
}

// SentTimestamps returns timestamps for each Send call, paired with SentReports.
func (s *pixySimulator) SentTimestamps() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sentTimestamps
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
