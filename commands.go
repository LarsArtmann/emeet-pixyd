//go:build linux

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	respAutoModeOff    = "auto mode: off"
	respAudioUsage     = "usage: audio [nc|live|org]"
	respAutoUsage      = "usage: auto [off|full|tracking-only|privacy-only]"
	respDeviceNotFound = "device not found"

	cmdGestureOn     = "gesture-on"
	cmdIdle          = "idle"
	cmdAutoOn        = "auto-on"
	cmdPrivacy       = string(pixy.StatePrivacy)
	cmdToggleGesture = "toggle-gesture"
	cmdToggleAuto    = "toggle-auto"
	minCmdParts      = 2
)

func (d *Daemon) handleCommand(ctx context.Context, cmd string) string {
	d.cmdMu.Lock()
	defer d.cmdMu.Unlock()

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return d.getStatus(ctx)
	}

	switch parts[0] {
	case "status":
		return d.getStatus(ctx)

	case "track":
		return d.handleTrackingCommand(ctx, pixy.StateTracking, "track")

	case cmdIdle:
		return d.handleTrackingCommand(ctx, pixy.StateIdle, cmdIdle)

	case cmdPrivacy:
		return d.handleTrackingCommand(ctx, pixy.StatePrivacy, cmdPrivacy)

	case "toggle-privacy":
		d.mu.RLock()
		camera := d.state.Camera
		d.mu.RUnlock()

		if camera == pixy.StatePrivacy {
			return d.handleTrackingCommand(ctx, pixy.StateTracking, "toggle-privacy")
		}

		return d.handleTrackingCommand(ctx, pixy.StatePrivacy, "toggle-privacy")

	case "audio":
		return d.handleAudioCommand(ctx, parts)

	case cmdGestureOn, "gesture-off", cmdToggleGesture:
		return d.handleGestureCommand(ctx, parts[0])

	case "center":
		return d.handleCenterCommand(ctx)

	case cmdAutoOn, "auto-off", cmdToggleAuto, "auto":
		return d.handleAutoCommand(parts)

	case "waybar":
		return d.waybarOutput()

	case "sync":
		return d.syncState(ctx)

	case "probe":
		d.mu.Lock()
		d.probeDevices()
		dev := d.videoDev
		d.mu.Unlock()

		if dev != "" {
			return "device found: " + dev
		}

		return respDeviceNotFound

	case axisPan, axisTilt, axisZoom:
		return d.handlePTZCommand(ctx, parts)

	case "device":
		d.mu.RLock()
		dev := d.videoDev
		d.mu.RUnlock()

		if dev != "" {
			return dev
		}

		return respDeviceNotFound

	default:
		return "unknown command: " + parts[0]
	}
}

func (d *Daemon) handleTrackingCommand(
	ctx context.Context,
	state pixy.CameraState,
	label string,
) string {
	if err := d.setTracking(ctx, state); err != nil {
		return (&CommandError{Op: label + " " + string(state), Err: err}).Error()
	}

	if state == pixy.StateTracking {
		return "tracking on"
	}

	if state == pixy.StatePrivacy {
		return "privacy on"
	}

	return "tracking off"
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
			return respAudioUsage
		}
	}

	audioErr := d.setAudio(ctx, mode)
	if audioErr != nil {
		return (&CommandError{Op: "audio " + string(mode), Err: audioErr}).Error()
	}

	return "audio: " + string(mode)
}

func (d *Daemon) handleGestureCommand(ctx context.Context, cmd string) string {
	var enable bool
	switch cmd {
	case cmdGestureOn:
		enable = true
	case "gesture-off":
		enable = false
	case "toggle-gesture":
		d.mu.RLock()
		enable = !d.state.Gesture
		d.mu.RUnlock()
	}
	if err := d.setGesture(ctx, enable); err != nil {
		return (&CommandError{Op: cmd + " enable=" + strconv.FormatBool(enable), Err: err}).Error()
	}

	if enable {
		return "gesture on"
	}

	return "gesture off"
}

func (d *Daemon) handleCenterCommand(ctx context.Context) string {
	if err := d.centerCamera(ctx); err != nil {
		return (&CommandError{Op: "center", Err: err}).Error()
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
	case "auto-off":
		mode = pixy.AutoOff
	case "toggle-auto":
		d.mu.RLock()
		if d.state.AutoMode.IsOff() {
			mode = pixy.AutoFull
		} else {
			mode = pixy.AutoOff
		}
		d.mu.RUnlock()
	default:
		mode = pixy.AutoFull
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

	lo, hi := ptzLimits(axis)
	val, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Sprintf("error: %s: %v", axis, ErrInvalidValue)
	}

	val = clampInt(val, lo, hi)

	multiplier := v4l2DegreesPerUnit
	if axis == axisZoom {
		multiplier = 1
	}

	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	if videoDev == "" {
		return fmt.Sprintf("error: %s: device not found", axis)
	}

	if v4l2Err := v4l2Set(
		ctx,
		videoDev,
		axis+"_absolute",
		strconv.Itoa(val*multiplier),
	); v4l2Err != nil {
		return (&CommandError{Op: axis, Err: v4l2Err}).Error()
	}

	return fmt.Sprintf("%s set to %d", axis, val)
}
