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

var (
	errAudioSourceNotFound = errors.New("PIXY audio source not found")
	errInvalidValue        = errors.New("invalid value")
)

const (
	respAutoModeOff    = "auto mode off"
	respAutoModeOn     = "auto mode on"
	respAudioUsage     = "usage: audio [nc|live|org]"
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

	case cmdAutoOn, "auto-off", cmdToggleAuto:
		return d.handleAutoCommand(parts[0])

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
		return (&CommandError{Ok: label, Err: err}).Error()
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
		return (&CommandError{Ok: "audio " + string(mode), Err: audioErr}).Error()
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
		return (&CommandError{Ok: cmd, Err: err}).Error()
	}

	if enable {
		return "gesture on"
	}

	return "gesture off"
}

func (d *Daemon) handleCenterCommand(ctx context.Context) string {
	if err := d.centerCamera(ctx); err != nil {
		return (&CommandError{Ok: "center", Err: err}).Error()
	}

	return "centered"
}

func (d *Daemon) handleAutoCommand(cmd string) string {
	var mode bool
	switch cmd {
	case cmdAutoOn:
		mode = true
	case "auto-off":
		mode = false
	case "toggle-auto":
		d.mu.RLock()
		mode = !d.state.AutoMode
		d.mu.RUnlock()
	}

	d.mu.Lock()
	d.state.AutoMode = mode
	d.saveStateOrLog("failed to save state")
	d.mu.Unlock()

	if mode {
		return respAutoModeOn
	}

	return respAutoModeOff
}

func (d *Daemon) handlePTZCommand(ctx context.Context, parts []string) string {
	if len(parts) < minCmdParts {
		return fmt.Sprintf("usage: %s <value>", parts[0])
	}

	axis := parts[0]

	lo, hi := ptzLimits(axis)
	val, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Sprintf("error: %s: %v", axis, errInvalidValue)
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
		return (&CommandError{Ok: axis, Err: v4l2Err}).Error()
	}

	return fmt.Sprintf("%s set to %d", axis, val)
}
