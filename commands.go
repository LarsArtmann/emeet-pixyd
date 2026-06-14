//go:build linux

package main

import (
	"context"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const parsePTZValueErrStr = "invalid PTZ value"

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

	default:
		d.cmdMu.Lock()
		result = d.handleMutatingCommand(ctx, parts)
		d.cmdMu.Unlock()
	}

	recordCommandMetric(ctx, parts[0], result)

	return result
}

func (d *Daemon) handleMutatingCommand(ctx context.Context, parts []string) CommandResult {
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

	case pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom:
		return d.handlePTZCommand(ctx, parts)

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

	if mode.IsOff() {
		return okResult(respAutoModeOff)
	}

	return okResult("auto mode: " + mode.String())
}
