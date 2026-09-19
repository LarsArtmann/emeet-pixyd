//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the `preset pull` command (TODO #141): the pixy-level response
// parser, the simulator's slot modeling, and the daemon → simulator sweep
// over the official V2 GetMotorPresetPosMode command.

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

// presetPosPayload builds a GetMotorPresetPosMode response payload:
// [mode:u8][pan:f32][tilt:f32][zoom:f32].
func presetPosPayload(mode byte, pan, tilt, zoom float32) []byte {
	out := []byte{mode}
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(pan))
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(tilt))
	out = binary.LittleEndian.AppendUint32(out, math.Float32bits(zoom))

	return out
}

func TestParseMotorPresetPosResponse_Occupied(t *testing.T) {
	t.Parallel()

	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)
	resp := v2Response(head, presetPosPayload(pixy.MotorPresetPositioned, 30.5, -10.25, 120)...)

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse occupied slot: %v", err)
	}

	if !reading.Occupied() {
		t.Error("occupied slot reading = false, want true")
	}

	if reading.Pan != 30.5 || reading.Tilt != -10.25 || reading.Zoom != 120 {
		t.Errorf("position = %+v, want (30.5, -10.25, 120)", reading)
	}

	values := reading.PTZValues()
	if values.Pan != 31 || values.Tilt != -10 || values.Zoom != 120 {
		t.Errorf("PTZValues = %+v, want rounded+clamped (31, -10, 120)", values)
	}
}

func TestParseMotorPresetPosResponse_EmptySlot(t *testing.T) {
	t.Parallel()

	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)
	resp := v2Response(head, 0)

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse empty slot: %v", err)
	}

	if reading.Occupied() {
		t.Error("empty slot reading.Occupied() = true, want false")
	}
}

func TestParseMotorPresetPosResponse_OccupiedTooShort(t *testing.T) {
	t.Parallel()

	head := pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface)
	// mode=1 promises three floats but only one follows.
	resp := v2Response(head, pixy.MotorPresetPositioned, 0, 0, 0, 0)

	if _, err := pixy.ParseMotorPresetPosResponse(head, resp); !errors.Is(err, pixy.ErrV2ResponseShort) {
		t.Errorf("truncated occupied payload err = %v, want ErrV2ResponseShort", err)
	}
}

func TestParseMotorPresetPosResponse_HeadMismatch(t *testing.T) {
	t.Parallel()

	resp := v2Response(pixy.V2GetMotorPresetPosMode.WithIface(pixy.MotorMCUIface), 1)

	if _, err := pixy.ParseMotorPresetPosResponse(pixy.V2GetMotorPresetPosMode, resp); !errors.Is(err, pixy.ErrV2ResponseHeadMismatch) {
		t.Errorf("mismatched echo err = %v, want ErrV2ResponseHeadMismatch", err)
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

	resp, err := sim.SendRecv(t.Context(), append(head.Bytes(), 2))
	if err != nil {
		t.Fatalf("GetMotorPresetPosMode slot 2: %v", err)
	}

	reading, err := pixy.ParseMotorPresetPosResponse(head, resp)
	if err != nil {
		t.Fatalf("parse slot 2: %v", err)
	}

	if got := reading.PTZValues(); got != (pixy.PTZValues{Pan: 30, Tilt: -10, Zoom: 120}) {
		t.Errorf("round-trip values = %+v, want {30 -10 120}", got)
	}

	// Untouched slot: mode byte 0, parseable with framing bytes only.
	emptyResp, err := sim.SendRecv(t.Context(), append(head.Bytes(), 5))
	if err != nil {
		t.Fatalf("GetMotorPresetPosMode slot 5: %v", err)
	}

	empty, err := pixy.ParseMotorPresetPosResponse(head, emptyResp)
	if err != nil {
		t.Fatalf("parse slot 5: %v", err)
	}

	if empty.Occupied() {
		t.Errorf("slot 5 = %+v, want not occupied", empty)
	}

	// The logical 0x03 routing answers with the same shape.
	logicalResp, err := sim.SendRecv(t.Context(), append(pixy.V2GetMotorPresetPosMode.Bytes(), 2))
	if err != nil {
		t.Fatalf("logical-head query: %v", err)
	}

	logical, err := pixy.ParseMotorPresetPosResponse(pixy.V2GetMotorPresetPosMode, logicalResp)
	if err != nil || !logical.Occupied() {
		t.Fatalf("logical-head reading = (%+v, %v), want occupied", logical, err)
	}

	// SetMotorPresetPosMode [slot][mode] flips the slot back to empty.
	if err := sim.Send(append(append(pixy.V2SetMotorPresetPosMode.WithIface(pixy.MotorMCUIface).Bytes(), 2), 0)); err != nil {
		t.Fatalf("SetMotorPresetPosMode: %v", err)
	}

	if entry, _ := sim.MotorPreset(2); entry.mode != 0 {
		t.Errorf("slot 2 mode after SetMotorPresetPosMode 0 = %d, want 0", entry.mode)
	}
}

func TestPresetPull_SimulatorSweep(t *testing.T) {
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

	// Pull is read-only on the hardware: no Send (only SendRecv queries).
	if reports := sim.SentReports(); len(reports) != reportsBeforePull {
		t.Errorf("pull sent %d extra HID reports, want 0 (GET-only sweep)", len(reports)-reportsBeforePull)
	}

	if queries := sim.Queries(); len(queries) != maxHardwarePresetSlots {
		t.Errorf("pull issued %d queries, want %d", len(queries), maxHardwarePresetSlots)
	}
}

func TestPresetPull_NeverOverwritesExistingNames(t *testing.T) {
	t.Parallel()

	sim, opt := withPixySimulator()
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

	sim, opt := withPixySimulator()
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

	if !strings.Contains(result.String(), "injected read failure") {
		t.Errorf("error = %q, want wrapped injected cause", result.String())
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

	sim, opt := withPixySimulator()
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

	sim, opt := withPixySimulator()
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
