//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"maps"
	"math"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the `preset pull` command (TODO #141): the pixy-level response
// parser (both statically evidenced shapes), the simulator's slot modeling,
// and the daemon → simulator sweep over the official V2
// GetMotorPresetPosMode command.

// setMotorPosReport builds the official V2 SetMotorPos report with the
// motor-MCU iface routing, the same shape the daemon's push path sends.
func setMotorPosReport(motor pixy.MotorType, pos float32) []byte {
	return append(
		pixy.V2SetMotorPos.WithIface(pixy.MotorMCUIface).Bytes(),
		pixy.MotorSpeedPayload(motor, pos)...,
	)
}

// seedHardwareSlot drives the simulator directly: move the motors and save
// the position into a 1-based slot, exactly like `preset push` does.
func seedHardwareSlot(t *testing.T, sim *pixySimulator, slot byte, pan, tilt, zoom float32) {
	t.Helper()

	for motor, pos := range map[pixy.MotorType]float32{
		pixy.MotorPan:  pan,
		pixy.MotorTilt: tilt,
		pixy.MotorZoom: zoom,
	} {
		if err := sim.Send(setMotorPosReport(motor, pos)); err != nil {
			t.Fatalf("seed slot %d: SetMotorPos %s: %v", slot, motor, err)
		}
	}

	if err := sim.Send(append(pixy.V2SetMotorPresetPos.WithIface(pixy.MotorMCUIface).Bytes(), slot)); err != nil {
		t.Fatalf("seed slot %d: SetMotorPresetPos: %v", slot, err)
	}
}

// fullPresetPayload builds the full (SET-echo) response payload:
// [slot:u8][mode:u8][pan:f32][tilt:f32][zoom:f32].
func fullPresetPayload(slot, mode byte, pan, tilt, zoom float32) []byte {
	out := []byte{slot, mode}
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(pan))
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(tilt))
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(zoom))

	return out
}

func TestParseMotorPresetPosResponse_ModeOnly(t *testing.T) {
	t.Parallel()

	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)

	// The Beta.25 GET parser is single-byte: [mode] at offset 8.
	resp := v2Response(head, pixy.MotorPresetPositioned)

	occupied, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse mode-only occupied: %v", err)
	}

	if !occupied.Occupied() || occupied.HasPosition() {
		t.Errorf(
			"mode-only occupied = (occupied %v, position %v), want (true, false)",
			occupied.Occupied(), occupied.HasPosition(),
		)
	}

	empty, err := pixy.ParseMotorPresetPosResponse(head, v2Response(head, 0))
	if err != nil {
		t.Fatalf("parse mode-only empty: %v", err)
	}

	if empty.Occupied() {
		t.Error("mode-only empty slot = occupied, want not occupied")
	}
}

func TestParseMotorPresetPosResponse_FullShape(t *testing.T) {
	t.Parallel()

	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)
	resp := v2Response(head, fullPresetPayload(2, pixy.MotorPresetPositioned, 30.5, -10.25, 120)...)

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse full occupied: %v", err)
	}

	if !reading.Occupied() || !reading.HasPosition() {
		t.Fatalf(
			"full occupied = (occupied %v, position %v), want (true, true)",
			reading.Occupied(), reading.HasPosition(),
		)
	}

	if reading.Slot != 2 {
		t.Errorf("slot echo = %d, want 2", reading.Slot)
	}

	if reading.Pan != 30.5 || reading.Tilt != -10.25 || reading.Zoom != 120 {
		t.Errorf("position = %+v, want (30.5, -10.25, 120)", reading)
	}

	values := reading.PTZValues()
	if values.Pan != 31 || values.Tilt != -10 || values.Zoom != 120 {
		t.Errorf("PTZValues = %+v, want rounded+clamped (31, -10, 120)", values)
	}

	// Full shape for an unoccupied slot: slot + mode, no floats.
	empty, err := pixy.ParseMotorPresetPosResponse(head, v2Response(head, fullPresetPayload(3, 0, 0, 0, 0)...))
	if err != nil {
		t.Fatalf("parse full empty: %v", err)
	}

	if empty.Occupied() || empty.HasPosition() {
		t.Errorf("full empty = (occupied %v, position %v), want (false, false)", empty.Occupied(), empty.HasPosition())
	}
}

func TestParseMotorPresetPosResponse_TruncatedFullShape(t *testing.T) {
	t.Parallel()

	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)

	// The official SET-echo parser gates the floats on length: a truncated
	// full shape still yields slot + mode but no position.
	resp := v2Response(head, 2, pixy.MotorPresetPositioned, 0, 0)

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse truncated full: %v", err)
	}

	if !reading.Occupied() || reading.HasPosition() {
		t.Errorf(
			"truncated full = (occupied %v, position %v), want (true, false)",
			reading.Occupied(), reading.HasPosition(),
		)
	}
}

func TestParseMotorPresetPosResponse_HeadMismatch(t *testing.T) {
	t.Parallel()

	// Echo a genuinely different command (battery head, masked dev 0x00 vs
	// 0x03) — must be rejected even under the routing mask.
	resp := v2Response(pixy.V2GetBatteryLevel, pixy.MotorPresetPositioned)

	_, err := pixy.ParseMotorPresetPosResponse(pixy.V2GetMotorPresetPosMode, resp)
	if !errors.Is(err, pixy.ErrV2ResponseHeadMismatch) {
		t.Errorf("mismatched echo err = %v, want ErrV2ResponseHeadMismatch", err)
	}
}

func TestParseMotorPresetPosResponse_RoutingMaskEcho(t *testing.T) {
	t.Parallel()

	// Every official parser masks the echo's dev byte with 0x1F, so a device
	// answering with the logical motor dev byte (0x03) instead of the routed
	// 0x63 still validates against the head as sent.
	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)
	resp := v2Response(pixy.V2GetMotorPresetPosMode, pixy.MotorPresetPositioned)

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("logical-dev echo rejected: %v", err)
	}

	if !reading.Occupied() {
		t.Error("logical-dev echo reading = not occupied, want occupied")
	}
}

func TestPixySimulatorV2_PresetSlotRoundTrip(t *testing.T) {
	t.Parallel()

	sim := newPixySimulator()
	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)

	seedHardwareSlot(t, sim, 2, 30, -10, 120)

	entry, ok := sim.MotorPreset(2)
	if !ok || entry.mode != pixy.MotorPresetPositioned {
		t.Fatalf("slot 2 entry = (%+v, %v), want occupied", entry, ok)
	}

	if entry.pos != [3]float32{30, -10, 120} {
		t.Errorf("slot 2 position = %v, want [30 -10 120]", entry.pos)
	}

	// Default: mode-only answers, the Beta.25-evidenced GET shape.
	resp, err := sim.SendRecv(t.Context(), append(head.Bytes(), 2))
	if err != nil {
		t.Fatalf("GetMotorPresetPosMode slot 2: %v", err)
	}

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse slot 2: %v", err)
	}

	if !reading.Occupied() || reading.HasPosition() {
		t.Fatalf(
			"slot 2 reading = (occupied %v, position %v), want (true, false)",
			reading.Occupied(), reading.HasPosition(),
		)
	}

	// Untouched slot: mode byte 0.
	empty, err := pixy.ParseMotorPresetPosResponse(head, mustPresetResponse(t, sim, head, 5))
	if err != nil {
		t.Fatalf("parse slot 5: %v", err)
	}

	if empty.Occupied() {
		t.Errorf("slot 5 = %+v, want not occupied", empty)
	}

	// The logical 0x03 routing answers with the same shape.
	logicalResp := mustPresetResponse(t, sim, pixy.V2GetMotorPresetPosMode, 2)

	if _, err := pixy.ParseMotorPresetPosResponse(pixy.V2GetMotorPresetPosMode, logicalResp); err != nil {
		t.Fatalf("logical-head reading: %v", err)
	}

	// Full-shape knob: the SET-echo shape carries slot + mode + position.
	sim.SetPresetFullResponses(true)

	full, err := pixy.ParseMotorPresetPosResponse(head, mustPresetResponse(t, sim, head, 2))
	if err != nil {
		t.Fatalf("parse full slot 2: %v", err)
	}

	if !full.HasPosition() || full.Slot != 2 {
		t.Fatalf("full slot 2 = (slot %d, position %v), want (2, true)", full.Slot, full.HasPosition())
	}

	if got := full.PTZValues(); got != (pixy.PTZValues{Pan: 30, Tilt: -10, Zoom: 120}) {
		t.Errorf("full round-trip values = %+v, want {30 -10 120}", got)
	}

	// SetMotorPresetPosMode [slot][mode] flips the slot back to empty.
	modeReport := append(
		pixy.V2SetMotorPresetPosMode.WithIface(pixy.MotorMCUIface).Bytes(), 2, 0,
	)

	if err := sim.Send(modeReport); err != nil {
		t.Fatalf("SetMotorPresetPosMode: %v", err)
	}

	if entry, _ := sim.MotorPreset(2); entry.mode != 0 {
		t.Errorf("slot 2 mode after SetMotorPresetPosMode 0 = %d, want 0", entry.mode)
	}
}

// mustPresetResponse queries one preset slot and fails the test on transport
// errors.
func mustPresetResponse(t *testing.T, sim *pixySimulator, head pixy.V2Head, slot byte) []byte {
	t.Helper()

	resp, err := sim.SendRecv(t.Context(), append(head.Bytes(), slot))
	if err != nil {
		t.Fatalf("preset query slot %d: %v", slot, err)
	}

	return resp
}

func TestPresetPull_ModeOnlySweep(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 1, 10, -5, 115)
	seedHardwareSlot(t, sim, 3, 20, 10, 130)

	reportsBeforePull := len(sim.SentReports())

	result := d.handleCommand(t.Context(), "preset pull")
	if result.IsError() {
		t.Fatalf("preset pull failed: %s", result.String())
	}

	// Mode-only answers prove the slots are set but expose no position.
	if want := "preset pull: no slot positions, 2 set (position not exposed), 6 empty"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}

	d.mu.RLock()
	count := len(d.state.Presets)
	d.mu.RUnlock()

	if count != 0 {
		t.Errorf("mode-only pull stored %d presets, want 0 (no positions available)", count)
	}

	// Pull is read-only on the hardware: no Send (only SendRecv queries).
	if reports := sim.SentReports(); len(reports) != reportsBeforePull {
		t.Errorf("pull sent %d extra HID reports, want 0 (GET-only sweep)", len(reports)-reportsBeforePull)
	}

	if queries := sim.Queries(); len(queries) != maxHardwarePresetSlots {
		t.Errorf("pull issued %d queries, want %d", len(queries), maxHardwarePresetSlots)
	}
}

func TestPresetPull_FullShapeSweep(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator(withPresetFullResponses())
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 1, 10, -5, 115)
	seedHardwareSlot(t, sim, 3, 20, 10, 130)

	result := d.handleCommand(t.Context(), "preset pull")
	if result.IsError() {
		t.Fatalf("preset pull failed: %s", result.String())
	}

	if want := "preset pulled: hw-1, hw-3, 6 empty"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}

	d.mu.RLock()
	got := d.state.Presets
	d.mu.RUnlock()

	if v := got["hw-1"]; v != (pixy.PTZValues{Pan: 10, Tilt: -5, Zoom: 115}) {
		t.Errorf("hw-1 = %+v, want {10 -5 115}", v)
	}

	if v := got["hw-3"]; v != (pixy.PTZValues{Pan: 20, Tilt: 10, Zoom: 130}) {
		t.Errorf("hw-3 = %+v, want {20 10 130}", v)
	}

	for _, name := range []string{"hw-2", "hw-4", "hw-5", "hw-6", "hw-7", "hw-8"} {
		if _, exists := got[name]; exists {
			t.Errorf("%s exists, want skipped (empty slot)", name)
		}
	}
}

func TestPresetPull_NeverOverwritesExistingNames(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator(withPresetFullResponses())
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 1, 10, -5, 115)

	// A user-named hw-1 predates the pull: pull must keep it, not clobber.
	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{"hw-1": {Pan: 99, Tilt: 0, Zoom: 140}}
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset pull")
	if result.IsError() {
		t.Fatalf("preset pull failed: %s", result.String())
	}

	if !strings.Contains(result.String(), "1 already named") {
		t.Errorf("response = %q, want kept-slot note", result.String())
	}

	d.mu.RLock()
	kept := d.state.Presets["hw-1"]
	d.mu.RUnlock()

	if kept != (pixy.PTZValues{Pan: 99, Tilt: 0, Zoom: 140}) {
		t.Errorf("hw-1 = %+v, want untouched user values {99 0 140}", kept)
	}
}

func TestPresetPull_LimitReached(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator(withPresetFullResponses())
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 1, 10, -5, 115)

	d.mu.Lock()
	d.state.Presets = pixy.PresetMap{}

	for i := range pixy.MaxPresets {
		d.state.Presets[string(rune('a'+i))] = pixy.PTZValues{Zoom: pixy.ZoomDefault}
	}
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset pull")
	if result.IsError() {
		t.Fatalf("preset pull failed: %s", result.String())
	}

	if !strings.Contains(result.String(), "preset limit reached (16)") {
		t.Errorf("response = %q, want limit note", result.String())
	}

	d.mu.RLock()
	count := len(d.state.Presets)
	d.mu.RUnlock()

	if count != pixy.MaxPresets {
		t.Errorf("preset count after full pull = %d, want %d (additive, no eviction)", count, pixy.MaxPresets)
	}
}

func TestPresetPull_NoDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	result := d.handleCommand(t.Context(), "preset pull")
	if !result.IsError() || !strings.Contains(result.String(), "not connected") {
		t.Errorf("pull without device = %q, want not-connected error", result.String())
	}
}

func TestPresetPull_AllSlotsUnreadable(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	sim.sendRecvErr = errors.New("injected read failure")
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	result := d.handleCommand(t.Context(), "preset pull")
	if !result.IsError() {
		t.Fatalf("pull with failing device = %q, want error", result.String())
	}

	if !strings.Contains(result.String(), "3/8 slots unreadable") {
		t.Errorf("error = %q, want early-aborted failure count %q", result.String(), "3/8 slots unreadable")
	}

	if !strings.Contains(result.String(), "aborted after 3 consecutive failures") {
		t.Errorf("error = %q, want abort note", result.String())
	}

	if !strings.Contains(result.String(), "injected read failure") {
		t.Errorf("error = %q, want wrapped injected cause", result.String())
	}

	if queries := sim.Queries(); len(queries) != presetPullMaxConsecutiveFailures {
		t.Errorf("pull issued %d queries, want %d (abort stops the sweep)", len(queries), presetPullMaxConsecutiveFailures)
	}
}

// flakyQuerySim fails the first N SendRecv queries, then delegates. It models
// a device that answers eventually — impossible to express with the
// simulator's all-or-nothing sendRecvErr — so tests can pin that one
// success between failures resets the early-abort counter.
type flakyQuerySim struct {
	*pixySimulator

	mu        sync.Mutex
	remaining int
}

func (f *flakyQuerySim) SendRecv(ctx context.Context, report []byte) ([]byte, error) {
	f.mu.Lock()
	fail := f.remaining > 0

	if fail {
		f.remaining--
	}

	f.mu.Unlock()

	if fail {
		return nil, errors.New("injected slot failure")
	}

	return f.pixySimulator.SendRecv(ctx, report)
}

func (f *flakyQuerySim) String() string { return "flaky-query-sim" }

func TestPresetPull_EarlyAbortStopsSweep(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)
	d.hidDev = &flakyQuerySim{pixySimulator: sim, remaining: presetPullMaxConsecutiveFailures}

	result := d.handleCommand(t.Context(), "preset pull")
	if !result.IsError() || !strings.Contains(result.String(), "aborted after 3 consecutive failures") {
		t.Fatalf("flaky-start pull = %q, want abort error", result.String())
	}

	if queries := sim.Queries(); len(queries) != presetPullMaxConsecutiveFailures {
		t.Errorf("sweep issued %d queries, want %d", len(queries), presetPullMaxConsecutiveFailures)
	}
}

func TestPresetPull_SuccessResetsAbortCounter(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)
	d.hidDev = &flakyQuerySim{pixySimulator: sim, remaining: 2}

	seedHardwareSlot(t, sim, 3, 20, 10, 130)

	result := d.handleCommand(t.Context(), "preset pull")
	if result.IsError() {
		t.Fatalf("pull with intermittent failures failed: %s", result.String())
	}

	// Two failures, then a success resets the counter, so the sweep runs to
	// completion: 8 queries, 2 unreadable, slot 3 pulled, no abort note.
	if strings.Contains(result.String(), "aborted") {
		t.Errorf("response = %q, want no abort note after a success reset", result.String())
	}

	if !strings.Contains(result.String(), "preset pulled: hw-3") || !strings.Contains(result.String(), "2 unreadable") {
		t.Errorf("response = %q, want hw-3 pulled with 2 unreadable", result.String())
	}

	if queries := sim.Queries(); len(queries) != maxHardwarePresetSlots {
		t.Errorf("sweep issued %d queries, want %d", len(queries), maxHardwarePresetSlots)
	}
}

func TestPresetPull_DryRunDoesNotMutate(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator(withPresetFullResponses())
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 1, 10, -5, 115)
	seedHardwareSlot(t, sim, 3, 20, 10, 130)

	before := pixy.PresetMap{
		"custom": {Pan: 5, Tilt: 5, Zoom: 105},
		"hw-1":   {Pan: 99, Tilt: 0, Zoom: 140},
	}

	d.mu.Lock()
	d.state.Presets = before
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset pull "+flagDryRun)
	if result.IsError() {
		t.Fatalf("dry-run pull failed: %s", result.String())
	}

	// hw-1 already named, hw-3 would land, the rest empty.
	if want := "preset pull (dry run): would pull: hw-3, 6 empty, 1 already named"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}

	d.mu.RLock()
	got := d.state.Presets
	d.mu.RUnlock()

	if !maps.Equal(got, before) {
		t.Errorf("dry run mutated presets: %+v, want unchanged %+v", got, before)
	}

	if reports := sim.SentReports(); len(reports) != 0 {
		t.Errorf("dry run sent %d HID reports, want 0", len(reports))
	}

	if queries := sim.Queries(); len(queries) != maxHardwarePresetSlots {
		t.Errorf("dry run issued %d queries, want %d (full sweep still runs)", len(queries), maxHardwarePresetSlots)
	}
}

func TestPresetPull_DryRunModeOnly(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 1, 10, -5, 115)

	result := d.handleCommand(t.Context(), "preset pull "+flagDryRun)
	if result.IsError() {
		t.Fatalf("dry-run pull failed: %s", result.String())
	}

	if want := "preset pull (dry run): no slot positions, 1 set (position not exposed), 7 empty"; result.String() != want {
		t.Errorf("response = %q, want %q", result.String(), want)
	}
}

func TestPresetPull_DryRunRejectsExtraName(t *testing.T) {
	t.Parallel()

	_, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	result := d.handleCommand(t.Context(), "preset pull "+flagDryRun+" hw-1")
	if !result.IsError() || !strings.Contains(result.String(), "takes no name") {
		t.Errorf("preset pull --dry-run hw-1 = %q, want usage error", result.String())
	}
}

func TestPresetPull_CircuitBreakerOpen(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	d.mu.Lock()
	d.hidFailCount = hidCircuitBreakerThreshold
	d.mu.Unlock()

	result := d.handleCommand(t.Context(), "preset pull")
	if !result.IsError() || !strings.Contains(result.String(), "not connected") {
		t.Errorf("pull with open breaker = %q, want not-connected", result.String())
	}

	if queries := sim.Queries(); len(queries) != 0 {
		t.Errorf("pull issued %d queries despite open breaker, want 0", len(queries))
	}
}

func TestPresetPull_TakesNoName(t *testing.T) {
	t.Parallel()

	_, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	result := d.handleCommand(t.Context(), "preset pull hw-1")
	if !result.IsError() || !strings.Contains(result.String(), "takes no name") {
		t.Errorf("preset pull hw-1 = %q, want usage error", result.String())
	}
}

func TestPresetPull_PersistsInState(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	sim, opt := withPixySimulator(withPresetFullResponses())

	first := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)
	first.config.StateDir = stateDir

	seedHardwareSlot(t, sim, 1, 42, -7, 111)

	if result := first.handleCommand(t.Context(), "preset pull"); result.IsError() {
		t.Fatalf("preset pull failed: %s", result.String())
	}

	second := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev)
	second.config.StateDir = stateDir

	if !second.loadState() {
		t.Fatal("loadState = false, want true")
	}

	second.mu.RLock()
	v, ok := second.state.Presets["hw-1"]
	second.mu.RUnlock()

	if !ok || v != (pixy.PTZValues{Pan: 42, Tilt: -7, Zoom: 111}) {
		t.Errorf("persisted hw-1 = (%+v, %v), want {42 -7 111}", v, ok)
	}
}

func TestWebPresetPullEndpoint(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator(withPresetFullResponses())
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)

	seedHardwareSlot(t, sim, 2, 15, 5, 125)

	server := newTestWebServer(t, d)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"/api/preset/pull", nil)
	if err != nil {
		t.Fatal(err)
	}

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	d.mu.RLock()
	_, exists := d.state.Presets["hw-2"]
	d.mu.RUnlock()

	if !exists {
		t.Error("hw-2 not pulled by the web endpoint")
	}
}

func TestWebPresetPull_RendersHeaderButton(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev)
	server := newTestWebServer(t, d)

	body := getPanelBody(t, server)

	assertContainsAll(t, body, []string{
		"preset-pull-btn",
		`@post('/api/preset/pull')`,
		`aria-label="Pull presets from camera hardware slots"`,
	})
}
