//go:build linux

package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	respTrackingOn     = "tracking on"
	respPrivacyOn      = "privacy on"
	respTrackingOff    = "tracking off"
	respAutoModeOff    = "auto mode: off"
	respAutoUsage      = "usage: auto [off|full|tracking-only|privacy-only]"
	respDeviceNotFound = "device not found"
	respGestureOn      = "gesture on"
	respGestureOff     = "gesture off"
	respCentered       = "centered"
	respPresetUsage    = "usage: preset <save|load|delete|list> [name]"
	respPresetNotFound = "preset not found"

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

	case cmdWaybar, cmdVersion, cmdSync, cmdProbe, cmdDevice:
		result = d.handleQueryCommand(ctx, parts)

	case cmdTrack, cmdIdle, cmdPrivacy, cmdTogglePrivacy, cmdAudio,
		cmdGestureOn, cmdGestureOff, cmdToggleGesture:
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
// save/load need V4L2 I/O (v4l2Mu); delete/list are state-only (no I/O lock).
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
		return okResult(d.waybarOutput())

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
		d.mu.RUnlock()

		if dev != "" {
			if hid != "" {
				return okResult(dev + " " + hid)
			}

			return okResult(dev)
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

		return okResult("auto mode: " + mode.String())
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

		return okResult("auto mode: " + mode.String())
	}

	d.mu.Lock()
	d.state.AutoMode = mode
	d.saveStateOrLog("failed to save state")
	d.mu.Unlock()
	d.broadcastStateChanged()

	if mode.IsOff() {
		return okResult(respAutoModeOff)
	}

	return okResult("auto mode: " + mode.String())
}

const maxPresets = 16

const (
	presetSave     = "save"
	presetLoad     = "load"
	presetDelete   = "delete"
	presetList     = "list"
	minPresetParts = 3
)

func isValidPresetSubcmd(s string) bool {
	switch s {
	case presetSave, presetLoad, presetDelete, presetList:
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

	names := make([]string, 0, len(d.state.Presets))
	for name := range d.state.Presets {
		names = append(names, name)
	}

	slices.Sort(names)

	var b strings.Builder
	b.WriteString("presets:")

	for _, name := range names {
		v := d.state.Presets[name]
		fmt.Fprintf(&b, "\n  %s: pan=%d tilt=%d zoom=%d", name, v.Pan, v.Tilt, v.Zoom)
	}

	return okResult(b.String())
}

func (d *Daemon) handlePresetSave(ctx context.Context, name string) CommandResult {
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
		d.state.Presets = make(map[string]pixy.PTZValues)
	}

	if _, exists := d.state.Presets[name]; !exists && len(d.state.Presets) >= maxPresets {
		d.mu.Unlock()

		return errResultMsg(fmt.Sprintf("preset limit reached (%d)", maxPresets))
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
