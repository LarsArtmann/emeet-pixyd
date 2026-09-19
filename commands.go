//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	respTrackingOn         = "tracking on"
	respPrivacyOn          = "privacy on"
	respTrackingOff        = "tracking off"
	respAutoModeOff        = "auto mode: off"
	respAutoModePrefix     = "auto mode: "
	respAutoUsage          = "usage: auto [off|full|tracking-only|privacy-only]"
	respDeviceNotFound     = "device not found"
	respGestureOn          = "gesture on"
	respGestureOff         = "gesture off"
	respCentered           = "centered"
	respPresetUsage        = "usage: preset <save|load|delete|push|pull [--dry-run]|list> [name]"
	respPresetNotFound     = "preset not found"
	respSpeedUsage         = "usage: speed <pan|tilt|zoom> <value>"
	respTrackingUsage      = "usage: tracking <none|face|halfbody|fullbody>"
	respBatteryUnavailable = "battery: unavailable on this device"

	cmdStatus        = "status"
	cmdGestureOn     = "gesture-on"
	cmdGestureOff    = "gesture-off"
	cmdIdle          = "idle"
	cmdAutoOn        = "auto-on"
	cmdAutoOff       = "auto-off"
	cmdPrivacy       = string(pixy.StatePrivacy)
	cmdTogglePrivacy = "toggle-privacy"
	cmdToggleGesture = "toggle-gesture"
	cmdToggleAuto    = "toggle-auto"
	cmdTrack         = "track"
	cmdAudio         = "audio"
	cmdCenter        = "center"
	cmdSpeed         = "speed"
	cmdBattery       = "battery"
	cmdTracking      = "tracking"
	cmdAuto          = "auto"
	cmdPreset        = "preset"
	cmdVersion       = "version"
	cmdSync          = "sync"
	cmdProbe         = "probe"
	cmdWaybar        = "waybar"
	cmdDevice        = "device"
	minCmdParts      = 2
)

func (d *Daemon) handleCommand(ctx context.Context, cmd string) CommandResult {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return okResult(d.getStatus(ctx))
	}

	var result CommandResult

	switch parts[0] {
	case cmdStatus:
		result = okResult(d.getStatus(ctx))

	case cmdBattery:
		result = d.handleBatteryCommand(ctx)

	case cmdWaybar, cmdVersion, cmdSync, cmdProbe, cmdDevice:
		result = d.handleQueryCommand(ctx, parts)

	case cmdTrack, cmdIdle, cmdPrivacy, cmdTogglePrivacy, cmdAudio,
		cmdGestureOn, cmdGestureOff, cmdToggleGesture, cmdSpeed, cmdTracking:
		d.hidMu.Lock()
		result = d.handleMutatingCommand(ctx, parts)
		d.hidMu.Unlock()

	case cmdCenter:
		d.v4l2Mu.Lock()
		result = d.handleMutatingCommand(ctx, parts)
		d.v4l2Mu.Unlock()

	case cmdAutoOn, cmdAutoOff, cmdToggleAuto, cmdAuto:
		result = d.handleMutatingCommand(ctx, parts)

	case cmdPreset:
		result = d.handlePresetWithLock(ctx, parts)

	default:
		if ptzAxisValid(pixy.Axis(parts[0])) {
			d.v4l2Mu.Lock()
			result = d.handleMutatingCommand(ctx, parts)
			d.v4l2Mu.Unlock()
		} else {
			result = errResultMsg("unknown command: " + parts[0])
		}
	}

	recordCommandMetric(ctx, parts[0], result)

	return result
}

// handlePresetWithLock routes preset subcommands to the correct lock:
// save/load need V4L2 I/O (v4l2Mu); delete/list are state-only (no I/O lock);
// push/pull take hidMu internally (motor-MCU I/O) and never hold v4l2Mu.
func (d *Daemon) handlePresetWithLock(ctx context.Context, parts []string) CommandResult {
	needsV4L2 := len(parts) >= minCmdParts &&
		(parts[1] == presetSave || parts[1] == presetLoad)

	if needsV4L2 {
		d.v4l2Mu.Lock()
		defer d.v4l2Mu.Unlock()
	}

	return d.handleMutatingCommand(ctx, parts)
}

func (d *Daemon) handleMutatingCommand(ctx context.Context, parts []string) CommandResult {
	// PTZ commands are routed by axis name (pan/tilt/zoom). Checked before
	// the string switch because pixy.Axis is a branded type.
	if ptzAxisValid(pixy.Axis(parts[0])) {
		return d.handlePTZCommand(ctx, parts)
	}

	switch parts[0] {
	case cmdTrack:
		return d.handleTrackingCommand(ctx, pixy.StateTracking, cmdTrack)

	case cmdIdle:
		return d.handleTrackingCommand(ctx, pixy.StateIdle, cmdIdle)

	case cmdPrivacy:
		return d.handleTrackingCommand(ctx, pixy.StatePrivacy, cmdPrivacy)

	case cmdTogglePrivacy:
		return d.handleTogglePrivacy(ctx)

	case cmdAudio:
		return d.handleAudioCommand(ctx, parts)

	case cmdGestureOn, cmdGestureOff, cmdToggleGesture:
		return d.handleGestureCommand(ctx, parts[0])

	case cmdCenter:
		return d.handleCenterCommand(ctx)

	case cmdSpeed:
		return d.handleSpeedCommand(ctx, parts)

	case cmdTracking:
		return d.handleTrackingVariantCommand(ctx, parts)

	case cmdAutoOn, cmdAutoOff, cmdToggleAuto, cmdAuto:
		return d.handleAutoCommand(parts)

	case cmdPreset:
		return d.handlePresetCommand(ctx, parts)

	default:
		return errResultMsg("unknown command: " + parts[0])
	}
}

func (d *Daemon) handleQueryCommand(ctx context.Context, parts []string) CommandResult {
	switch parts[0] {
	case cmdWaybar:
		return okResult(d.waybarOutput(ctx))

	case cmdVersion:
		return okResult("emeet-pixyd " + buildVersion)

	case cmdSync:
		return d.syncState(ctx)

	case cmdProbe:
		d.mu.Lock()
		d.applyProbeResultLocked(probeDevices()) //nolint:contextcheck
		dev := d.videoDev
		d.mu.Unlock()
		d.broadcastStateChanged()

		if dev != "" {
			return okResult("device found: " + dev)
		}

		return okResult(respDeviceNotFound)

	case cmdDevice:
		d.mu.RLock()
		dev := d.videoDev
		hid := d.hidrawDev
		model := d.model
		d.mu.RUnlock()

		if dev != "" {
			parts := []string{dev}
			if hid != "" {
				parts = append(parts, hid)
			}

			if model != "" {
				parts = append(parts, string(model))
			}

			// Identity fields (TODO #151) append when the device answers;
			// each head is best-effort so output stays parseable otherwise.
			info, ok := d.identityStatus(ctx)
			parts = append(parts, formatIdentity(info, ok)...)

			return okResult(strings.Join(parts, " "))
		}

		return okResult(respDeviceNotFound)
	}

	return errResultMsg("unknown query command: " + parts[0])
}

func (d *Daemon) handleTogglePrivacy(ctx context.Context) CommandResult {
	d.mu.RLock()
	camera := d.state.Camera
	d.mu.RUnlock()

	if camera == pixy.StatePrivacy {
		return d.handleTrackingCommand(ctx, pixy.StateTracking, cmdTogglePrivacy)
	}

	return d.handleTrackingCommand(ctx, pixy.StatePrivacy, cmdTogglePrivacy)
}

func (d *Daemon) handleTrackingCommand(
	ctx context.Context,
	state pixy.CameraState,
	label string,
) CommandResult {
	err := d.deps.setTracking(ctx, state)
	if err != nil {
		return errResult(label+" "+string(state), err)
	}

	if state == pixy.StateTracking {
		return okResult(respTrackingOn)
	}

	if state == pixy.StatePrivacy {
		return okResult(respPrivacyOn)
	}

	return okResult(respTrackingOff)
}

func (d *Daemon) handleAudioCommand(ctx context.Context, parts []string) CommandResult {
	var mode pixy.AudioMode

	if len(parts) < minCmdParts {
		d.mu.RLock()
		mode = d.state.Audio.Next()
		d.mu.RUnlock()
	} else {
		var parseErr error

		mode, parseErr = pixy.ParseAudioMode(parts[1])
		if parseErr != nil {
			return errResult("audio "+parts[1], parseErr)
		}
	}

	audioErr := d.deps.setAudio(ctx, mode)
	if audioErr != nil {
		return errResult("audio "+string(mode), audioErr)
	}

	return okResult("audio: " + string(mode))
}

func (d *Daemon) handleGestureCommand(ctx context.Context, cmd string) CommandResult {
	var enable bool

	switch cmd {
	case cmdGestureOn:
		enable = true
	case cmdGestureOff:
		enable = false
	case cmdToggleGesture:
		d.mu.RLock()
		enable = !d.state.Gesture
		d.mu.RUnlock()
	}

	err := d.deps.setGesture(ctx, enable)
	if err != nil {
		return errResult(cmd+" enable="+strconv.FormatBool(enable), err)
	}

	if enable {
		return okResult(respGestureOn)
	}

	return okResult(respGestureOff)
}

func (d *Daemon) handleCenterCommand(ctx context.Context) CommandResult {
	// Centering moves every axis: re-assert configured speeds first (TODO
	// #138), best-effort like the single-axis moves.
	d.reassertSpeeds(ctx, pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom)

	err := d.deps.centerCamera(ctx)
	if err != nil {
		return errResult(cmdCenter, err)
	}

	d.ptzCache.Set(
		pixy.PTZValues{Pan: 0, Tilt: 0, Zoom: pixy.ZoomDefault},
		ptzCacheTTL,
	)
	d.broadcastStateChanged()

	return okResult(respCentered)
}

func (d *Daemon) handleAutoCommand(parts []string) CommandResult {
	if len(parts) >= minCmdParts {
		mode, parseErr := pixy.ParseAutoMode(parts[1])
		if parseErr != nil {
			return okResult(respAutoUsage)
		}

		d.mu.Lock()
		d.state.AutoMode = mode
		d.saveStateOrLog("failed to save state")
		d.mu.Unlock()
		d.broadcastStateChanged()

		return okResult(respAutoModePrefix + mode.String())
	}

	cmd := parts[0]

	var mode pixy.AutoMode

	switch cmd {
	case cmdAutoOn:
		mode = pixy.AutoFull
	case cmdAutoOff:
		mode = pixy.AutoOff
	case cmdToggleAuto:
		d.mu.RLock()
		mode = d.state.AutoMode.Toggle()
		d.mu.RUnlock()
	default:
		d.mu.RLock()
		mode = d.state.AutoMode
		d.mu.RUnlock()

		return okResult(respAutoModePrefix + mode.String())
	}

	d.mu.Lock()
	d.state.AutoMode = mode
	d.saveStateOrLog("failed to save state")
	d.mu.Unlock()
	d.broadcastStateChanged()

	if mode.IsOff() {
		return okResult(respAutoModeOff)
	}

	return okResult(respAutoModePrefix + mode.String())
}

const (
	presetSave     = "save"
	presetLoad     = "load"
	presetPush     = "push"
	presetPull     = "pull"
	presetDelete   = "delete"
	presetList     = "list"
	minPresetParts = 3
	flagDryRun     = "--dry-run"
)

func isValidPresetSubcmd(s string) bool {
	switch s {
	case presetSave, presetLoad, presetPush, presetPull, presetDelete, presetList:
		return true
	default:
		return false
	}
}

func (d *Daemon) handlePresetCommand(ctx context.Context, parts []string) CommandResult {
	if len(parts) < minCmdParts || !isValidPresetSubcmd(parts[1]) {
		return okResult(respPresetUsage)
	}

	subcmd := parts[1]

	switch subcmd {
	case presetList:
		return d.handlePresetList()
	case presetSave:
		if len(parts) < minPresetParts {
			return errResultMsg("preset save: missing name")
		}

		return d.handlePresetSave(ctx, parts[2])
	case presetLoad:
		if len(parts) < minPresetParts {
			return errResultMsg("preset load: missing name")
		}

		return d.handlePresetLoad(ctx, parts[2])
	case presetPush:
		if len(parts) < minPresetParts {
			return errResultMsg("preset push: missing name")
		}

		return d.handlePresetPush(ctx, parts[2])
	case presetPull:
		switch {
		case len(parts) == minCmdParts:
			return d.handlePresetPull(ctx, false)
		case len(parts) == minCmdParts+1 && parts[2] == flagDryRun:
			return d.handlePresetPull(ctx, true)
		default:
			return errResultMsg("preset pull: takes no name")
		}
	case presetDelete:
		if len(parts) < minPresetParts {
			return errResultMsg("preset delete: missing name")
		}

		return d.handlePresetDelete(parts[2])
	}

	return okResult(respPresetUsage)
}

func (d *Daemon) handlePresetList() CommandResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.state.Presets) == 0 {
		return okResult("no presets saved")
	}

	var b strings.Builder
	b.WriteString("presets:")

	for _, name := range d.state.Presets.SortedNames() {
		v := d.state.Presets[name]
		fmt.Fprintf(&b, "\n  %s: pan=%d tilt=%d zoom=%d", name, v.Pan, v.Tilt, v.Zoom)
	}

	return okResult(b.String())
}

func (d *Daemon) handlePresetSave(ctx context.Context, name string) CommandResult {
	err := pixy.ValidatePresetName(name)
	if err != nil {
		return errResultMsg(err.Error())
	}

	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	if videoDev == "" {
		return errResultMsg(respDeviceNotFound)
	}

	values := d.deps.parsePTZ(ctx, videoDev)
	clamped := values.Clamp()

	d.mu.Lock()
	if d.state.Presets == nil {
		d.state.Presets = pixy.NewPresetMap()
	}

	if _, exists := d.state.Presets[name]; !exists && d.state.Presets.IsFull() {
		d.mu.Unlock()

		return errResultMsg(fmt.Sprintf("preset limit reached (%d)", pixy.MaxPresets))
	}

	d.state.Presets[name] = clamped
	d.saveStateOrLog("failed to save state")
	d.mu.Unlock()
	d.broadcastStateChanged()

	return okResult(fmt.Sprintf(
		"preset %q saved: pan=%d tilt=%d zoom=%d",
		name, clamped.Pan, clamped.Tilt, clamped.Zoom,
	))
}

func (d *Daemon) handlePresetLoad(ctx context.Context, name string) CommandResult {
	d.mu.RLock()
	videoDev := d.videoDev
	values, ok := d.state.Presets[name]
	d.mu.RUnlock()

	if !ok {
		return errResultMsg(respPresetNotFound)
	}

	if videoDev == "" {
		return errResultMsg(respDeviceNotFound)
	}

	// Wired motor speeds (TODO #138): re-assert all configured speeds before
	// the three-axis recall so the move runs at the user's speeds.
	d.reassertSpeeds(ctx, pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom)

	for _, axis := range ptzAxisOrder {
		info := ptzAxes[axis]
		val, _ := values.Get(axis)

		setErr := d.deps.v4l2Set(ctx, videoDev, info.V4L2Ctrl, strconv.Itoa(val*info.Multiplier))
		if setErr != nil {
			return errResult("preset "+name, setErr)
		}
	}

	d.ptzCache.Set(values.Clamp(), ptzCacheTTL)
	d.schedulePTZReadback(ctx, videoDev)
	d.broadcastStateChanged()

	return okResult(fmt.Sprintf("preset %q loaded: pan=%d tilt=%d zoom=%d", name, values.Pan, values.Tilt, values.Zoom))
}

func (d *Daemon) handlePresetDelete(name string) CommandResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.state.Presets[name]; !ok {
		return errResultMsg(respPresetNotFound)
	}

	delete(d.state.Presets, name)
	d.saveStateOrLog("failed to save state")
	d.broadcastStateChanged()

	return okResult(fmt.Sprintf("preset %q deleted", name))
}

// handlePresetPush mirrors a named software preset into a hardware motor slot
// (TODO #141): the daemon moves each axis to the preset position over the
// official V2 SetMotorPos command, then saves the current position into the
// slot with SetMotorPresetPos. Slots map deterministically from the preset
// list order (alphabetical, 1-based), the same order the CLI and web UI
// present everywhere else.
//
// This command MOVES THE PHYSICAL CAMERA — it is the one preset operation
// with a visible hardware side effect. Slot-count discovery and per-slot
// verification run in the hardware session (plan M27); until then the slot
// guard is the conservative maxHardwarePresetSlots.
func (d *Daemon) handlePresetPush(ctx context.Context, name string) CommandResult {
	if err := pixy.ValidatePresetName(name); err != nil {
		return errResultMsg(err.Error())
	}

	d.mu.RLock()
	values, exists := d.state.Presets[name]
	d.mu.RUnlock()

	if !exists {
		return errResultMsg(respPresetNotFound)
	}

	d.mu.RLock()
	sorted := d.state.Presets.SortedNames()
	d.mu.RUnlock()

	slot := 0

	for i, n := range sorted {
		if n == name {
			slot = i + 1

			break
		}
	}

	if slot == 0 || slot > maxHardwarePresetSlots {
		return errResultMsg(
			fmt.Sprintf("preset push: no hardware slot for %q (slots 1..%d)", name, maxHardwarePresetSlots),
		)
	}

	// The whole move+save sequence is one HID operation: hold hidMu so no
	// tracking/audio command interleaves between the move and the save.
	d.hidMu.Lock()
	defer d.hidMu.Unlock()

	// Wired motor speeds (TODO #138): the push moves every axis over HID —
	// re-assert configured speeds first, best-effort like the V4L2 paths.
	if speedErr := d.reassertSpeedsLocked(ctx, pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom); speedErr != nil {
		slog.Warn("preset push: speed re-assert failed, moving at firmware default", "error", speedErr)
	}

	for _, axis := range []struct {
		motor pixy.MotorType
		val   int
	}{
		{pixy.MotorPan, values.Pan},
		{pixy.MotorTilt, values.Tilt},
		{pixy.MotorZoom, values.Zoom},
	} {
		if err := d.setMotorPos(ctx, axis.motor, float32(axis.val)); err != nil {
			return errResult("preset push "+name, err)
		}
	}

	if err := d.setMotorPresetPos(ctx, byte(slot)); err != nil {
		return errResult("preset push "+name, err)
	}

	return okResult(fmt.Sprintf("preset pushed: %s -> slot %d", name, slot))
}

// presetPullNameFormat is the software preset name for a pulled hardware
// slot. The hw- prefix marks machine-mirrored presets so users can tell them
// apart from named ones; pull never touches any other name.
const presetPullNameFormat = "hw-%d"

// presetPullMaxConsecutiveFailures bounds the pull sweep's worst case: after
// this many back-to-back slot failures the device is treated as
// unresponsive and the sweep stops instead of grinding through the
// remaining slots (~4s of dead air at the assumed cap, TODO #167). The
// threshold matches the HID circuit breaker's consecutive-failure
// semantics; a single success resets the counter, and the summary reports
// the abort so a truncated sweep is never mistaken for a complete one.
const presetPullMaxConsecutiveFailures = 3

// presetPullOutcome is the result of one preset pull sweep: what landed
// (or would land, under dry-run), why the other slots did not, and whether
// the sweep stopped early.
type presetPullOutcome struct {
	pulled   []string
	empty    int
	skipped  int
	occupied int
	failures int
	limitHit bool
	aborted  bool
	dryRun   bool
}

// presetPullSweep is the mutable bookkeeping of one in-flight pull sweep:
// the outcome it accumulates, the first slot failure (the cause reported
// when the whole sweep dies), the consecutive-failure counter behind the
// early abort, and whether the software preset collection was mutated.
type presetPullSweep struct {
	outcome     presetPullOutcome
	firstErr    error
	consecutive int
	changed     bool
}

// recordFailure books one failed slot query and reports whether the
// early-abort threshold (presetPullMaxConsecutiveFailures, TODO #167) is
// now reached.
func (s *presetPullSweep) recordFailure(err error) bool {
	s.outcome.failures++
	s.consecutive++

	if s.firstErr == nil {
		s.firstErr = err
	}

	return s.consecutive >= presetPullMaxConsecutiveFailures
}

// admitReading classifies one occupied-with-position reading against the
// software preset collection: already named → counted as skipped, collection
// full → counted against the limit (stay additive: a pulled slot may never
// push out a user-named preset), dry-run → reported as would-pull, otherwise
// stored as an additive hw-<slot> preset. It returns whether the preset
// limit was reached, which ends the sweep.
func (s *presetPullSweep) admitReading(reading pixy.MotorPresetReading, slot int, d *Daemon) bool {
	name := fmt.Sprintf(presetPullNameFormat, slot)

	d.mu.Lock()
	defer d.mu.Unlock()

	_, exists := d.state.Presets[name]
	full := !exists && d.state.Presets.IsFull()

	switch {
	case exists:
		s.outcome.skipped++
	case full:
	case s.outcome.dryRun:
		s.outcome.pulled = append(s.outcome.pulled, name)
	default:
		if d.state.Presets == nil {
			d.state.Presets = pixy.NewPresetMap()
		}

		d.state.Presets[name] = reading.PTZValues()
		s.outcome.pulled = append(s.outcome.pulled, name)
		s.changed = true
	}

	return full
}

// allSlotsFailed reports whether every attempted slot failed and nothing
// else was observed — the "the whole sweep died" case — together with the
// composed error message (failure count, plus the abort note when the sweep
// stopped early).
func (s *presetPullSweep) allSlotsFailed() (string, bool) {
	outcome := &s.outcome

	if len(outcome.pulled) > 0 || outcome.empty > 0 || outcome.skipped > 0 ||
		outcome.occupied > 0 || outcome.failures == 0 {
		return "", false
	}

	msg := fmt.Sprintf(
		"preset pull: %d/%d slots unreadable",
		outcome.failures, maxHardwarePresetSlots,
	)

	if outcome.aborted {
		msg += fmt.Sprintf(", aborted after %d consecutive failures", presetPullMaxConsecutiveFailures)
	}

	return msg, true
}

// handlePresetPull sweeps the hardware motor preset slots (TODO #141): every
// slot 1..maxHardwarePresetSlots is queried over the official V2
// GetMotorPresetPosMode command and occupied slots (mode byte 1) are stored
// as additive hw-<slot> presets, rounded and clamped to the V4L2 limits.
// Pull is additive — existing preset names are never overwritten — explicit
// (nothing auto-syncs afterward), and read-only on the hardware: no motor
// moves. With dryRun the sweep is report-only: identical queries and
// accounting, but nothing is stored, persisted, or broadcast (TODO #171).
//
// Two response shapes are statically evidenced (Beta.25 x64 parsers, see
// pixy.MotorPresetReading): the full shape carries the position and lands as
// a preset; the mode-only shape proves a slot is set without exposing its
// position, and those slots are counted instead of stored. Slots whose mode
// byte marks them empty/invalid are skipped; per-slot query failures are
// skipped too (one dead slot must not abort the sweep), except an
// unreachable device, which aborts immediately. When EVERY attempted slot
// fails, the first cause is returned prefixed with the failure count ("8/8
// slots unreadable") so the error says how much of the sweep died. After
// presetPullMaxConsecutiveFailures back-to-back failures the sweep aborts
// early (TODO #167) and says so in the outcome. The slot count is the
// assumed maxHardwarePresetSlots cap until the #166 hardware session pins
// the real count with this same sweep.
func (d *Daemon) handlePresetPull(ctx context.Context, dryRun bool) CommandResult {
	// The whole sweep is one HID operation: hold hidMu so no tracking/audio
	// command interleaves between slot queries.
	d.hidMu.Lock()
	defer d.hidMu.Unlock()

	sweep := presetPullSweep{
		outcome: presetPullOutcome{
			pulled:   nil,
			empty:    0,
			skipped:  0,
			occupied: 0,
			failures: 0,
			limitHit: false,
			aborted:  false,
			dryRun:   dryRun,
		},
		firstErr:    nil,
		consecutive: 0,
		changed:     false,
	}

	for slot := 1; slot <= maxHardwarePresetSlots; slot++ {
		reading, err := d.queryMotorPresetPos(ctx, byte(slot))
		if err != nil {
			if errors.Is(err, pixy.ErrPIXYNotConnected) {
				return errResult("preset pull", err)
			}

			if sweep.recordFailure(err) {
				sweep.outcome.aborted = true

				break
			}

			continue
		}

		sweep.consecutive = 0

		if !reading.Occupied() {
			sweep.outcome.empty++

			continue
		}

		if !reading.HasPosition() {
			// Mode-only answer: the slot is set but the GET does not expose
			// its position (Beta.25 GET parser is single-byte).
			sweep.outcome.occupied++

			continue
		}

		if sweep.admitReading(reading, slot, d) {
			sweep.outcome.limitHit = true

			break
		}
	}

	if sweep.changed {
		d.mu.Lock()
		d.saveStateOrLog("failed to save state")
		d.mu.Unlock()
		d.broadcastStateChanged()
	}

	if msg, failed := sweep.allSlotsFailed(); failed {
		return errResult(msg, sweep.firstErr)
	}

	return okResult(sweep.outcome.summary())
}

// summary renders the pull result line: what landed (or would land under
// dry-run), and why the other slots did not. Zero counts are omitted.
func (o presetPullOutcome) summary() string {
	var b strings.Builder

	switch {
	case o.dryRun && len(o.pulled) > 0:
		b.WriteString("preset pull (dry run): would pull: ")
		b.WriteString(strings.Join(o.pulled, ", "))
	case o.dryRun:
		b.WriteString("preset pull (dry run): no slot positions")
	case len(o.pulled) > 0:
		b.WriteString("preset pulled: ")
		b.WriteString(strings.Join(o.pulled, ", "))
	default:
		b.WriteString("preset pull: no slot positions")
	}

	if o.occupied > 0 {
		fmt.Fprintf(&b, ", %d set (position not exposed)", o.occupied)
	}

	if o.empty > 0 {
		fmt.Fprintf(&b, ", %d empty", o.empty)
	}

	if o.skipped > 0 {
		fmt.Fprintf(&b, ", %d already named", o.skipped)
	}

	if o.failures > 0 {
		fmt.Fprintf(&b, ", %d unreadable", o.failures)
	}

	if o.aborted {
		fmt.Fprintf(&b, ", aborted after %d consecutive failures", presetPullMaxConsecutiveFailures)
	}

	if o.limitHit {
		fmt.Fprintf(&b, ", preset limit reached (%d)", pixy.MaxPresets)
	}

	return b.String()
}
