//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
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

func (d *Daemon) handleCommand(ctx context.Context, cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return d.getStatus(ctx)
	}

	switch parts[0] {
	case cmdStatus:
		return d.getStatus(ctx)

	case cmdWaybar, cmdVersion, cmdSync, cmdProbe, cmdDevice:
		return d.handleQueryCommand(ctx, parts)

	default:
		d.cmdMu.Lock()
		defer d.cmdMu.Unlock()
		return d.handleMutatingCommand(ctx, parts)
	}
}

func (d *Daemon) handleMutatingCommand(ctx context.Context, parts []string) string {
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
		return "unknown command: " + parts[0]
	}
}

func (d *Daemon) handleQueryCommand(ctx context.Context, parts []string) string {
	switch parts[0] {
	case cmdWaybar:
		return d.waybarOutput()

	case cmdVersion:
		return "emeet-pixyd " + buildVersion

	case cmdSync:
		return d.syncState(ctx)

	case cmdProbe:
		d.mu.Lock()
		d.applyProbeResult(probeDevices())
		dev := d.videoDev
		d.mu.Unlock()

		if dev != "" {
			return "device found: " + dev
		}

		return respDeviceNotFound

	case cmdDevice:
		d.mu.RLock()
		dev := d.videoDev
		hid := d.hidrawDev
		d.mu.RUnlock()

		if dev != "" {
			if hid != "" {
				return dev + " " + hid
			}
			return dev
		}

		return respDeviceNotFound
	}

	return errorPrefix + "unknown query command: " + parts[0]
}

func (d *Daemon) handleTogglePrivacy(ctx context.Context) string {
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
) string {
	if err := d.setTrackingFn(ctx, state); err != nil {
		return fmt.Errorf("%s%s %s: %w", errorPrefix, label, state, err).Error()
	}

	if state == pixy.StateTracking {
		return respTrackingOn
	}

	if state == pixy.StatePrivacy {
		return respPrivacyOn
	}

	return respTrackingOff
}

func (d *Daemon) handleAudioCommand(ctx context.Context, parts []string) string {
	var mode pixy.AudioMode
	if len(parts) < minCmdParts {
		d.mu.RLock()
		mode = d.state.Audio.Next()
		d.mu.RUnlock()
	} else {
		var parseErr error

		mode, parseErr = pixy.ParseAudioMode(parts[1])
		if parseErr != nil {
			return fmt.Errorf("%saudio %s: %w", errorPrefix, parts[1], parseErr).Error()
		}
	}

	audioErr := d.setAudioFn(ctx, mode)
	if audioErr != nil {
		return fmt.Errorf("%saudio %s: %w", errorPrefix, mode, audioErr).Error()
	}

	return "audio: " + string(mode)
}

func (d *Daemon) handleGestureCommand(ctx context.Context, cmd string) string {
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
	if err := d.setGestureFn(ctx, enable); err != nil {
		return (&CommandError{Op: cmd + " enable=" + strconv.FormatBool(enable), Err: err}).Error()
	}

	if enable {
		return "gesture on"
	}

	return "gesture off"
}

func (d *Daemon) handleCenterCommand(ctx context.Context) string {
	if err := d.centerCameraFn(ctx); err != nil {
		return (&CommandError{Op: cmdCenter, Err: err}).Error()
	}

	return "centered"
}

func (d *Daemon) handleAutoCommand(parts []string) string {
	if len(parts) >= minCmdParts {
		mode, parseErr := pixy.ParseAutoMode(parts[1])
		if parseErr != nil {
			return respAutoUsage
		}

		d.mu.Lock()
		d.state.AutoMode = mode
		d.saveStateOrLog("failed to save state")
		d.mu.Unlock()

		return "auto mode: " + mode.String()
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

		return "auto mode: " + mode.String()
	}

	d.mu.Lock()
	d.state.AutoMode = mode
	d.saveStateOrLog("failed to save state")
	d.mu.Unlock()

	if mode.IsOff() {
		return respAutoModeOff
	}

	return "auto mode: " + mode.String()
}

func (d *Daemon) handlePTZCommand(ctx context.Context, parts []string) string {
	if len(parts) < minCmdParts {
		return fmt.Sprintf("usage: %s <value>", parts[0])
	}

	axis := parts[0]

	info, ok := ptzAxes[axis]
	if !ok {
		return fmt.Sprintf("usage: %s <value>", parts[0])
	}
	val, err := strconv.Atoi(parts[1])
	if err != nil {
		return (&CommandError{Op: axis, Err: fmt.Errorf("%w: parse error", ErrInvalidValue)}).Error()
	}

	val = clampInt(val, info.Min, info.Max)

	multiplier := v4l2DegreesPerUnit
	if axis == pixy.AxisZoom {
		multiplier = 1
	}

	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	if videoDev == "" {
		return (&CommandError{Op: axis, Err: errors.New("device not found")}).Error()
	}

	if v4l2Err := d.v4l2SetFn(
		ctx,
		videoDev,
		axis+"_absolute",
		strconv.Itoa(val*multiplier),
	); v4l2Err != nil {
		return (&CommandError{Op: axis, Err: v4l2Err}).Error()
	}

	return fmt.Sprintf("%s set to %d", axis, val)
}
