//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const v4l2ctl = "v4l2-ctl"

// parsePTZValueErrStr is the prefix used by parsePTZValue when it returns
// a wrapped error. Lives in ptz.go because that's the only file that
// constructs the error.
const parsePTZValueErrStr = "invalid PTZ value"

// v4l2UnitsPerDegree is the V4L2 internal unit: 1 degree = 3600 V4L2 units.
const v4l2UnitsPerDegree = 3600

type ptzAxisInfo struct {
	Range      pixy.Range
	Label      string
	Unit       string
	V4L2Ctrl   string
	Multiplier int
}

//nolint:gochecknoglobals
var ptzAxes = map[pixy.Axis]ptzAxisInfo{
	pixy.AxisPan: {
		Range: pixy.PanRange, Label: "Pan", Unit: "\u00b0",
		V4L2Ctrl: "pan_absolute", Multiplier: v4l2UnitsPerDegree,
	},
	pixy.AxisTilt: {
		Range: pixy.TiltRange, Label: "Tilt", Unit: "\u00b0",
		V4L2Ctrl: "tilt_absolute", Multiplier: v4l2UnitsPerDegree,
	},
	pixy.AxisZoom: {
		Range: pixy.ZoomRange, Label: "Zoom", Unit: "x",
		V4L2Ctrl: "zoom_absolute", Multiplier: 1,
	},
}

// ptzAxisOrder defines the deterministic order for V4L2 control listing.
//
//nolint:gochecknoglobals
var ptzAxisOrder = []pixy.Axis{pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom}

// v4l2CtrlToAxis maps V4L2 control names back to PTZ axis names.
//
//nolint:gochecknoglobals
var v4l2CtrlToAxis = buildCtrlToAxis()

func buildCtrlToAxis() map[string]pixy.Axis {
	m := make(map[string]pixy.Axis, len(ptzAxes))
	for axis, info := range ptzAxes {
		m[info.V4L2Ctrl] = axis
	}

	return m
}

func ptzAxisValid(axis pixy.Axis) bool {
	_, ok := ptzAxes[axis]

	return ok
}

func (d *Daemon) v4l2Set(ctx context.Context, dev, ctrl, value string) error {
	err := d.deps.commander.Run(ctx, v4l2ctl, "-d", dev, "--set-ctrl="+ctrl+"="+value)
	if err != nil {
		return fmt.Errorf("v4l2Set %s=%s on %s: %w", ctrl, value, dev, err)
	}

	return nil
}

// v4l2GetCtrlList returns the comma-separated list of V4L2 control names for v4l2-ctl --get-ctrl.
func v4l2GetCtrlList() string {
	ctrls := make([]string, 0, len(ptzAxisOrder))
	for _, axis := range ptzAxisOrder {
		ctrls = append(ctrls, ptzAxes[axis].V4L2Ctrl)
	}

	return strings.Join(ctrls, ",")
}

func (d *Daemon) parsePTZValues(ctx context.Context, dev string) pixy.PTZValues {
	out, err := d.deps.commander.Output(
		ctx, v4l2ctl, "-d", dev,
		"--get-ctrl="+v4l2GetCtrlList(),
	)
	if err != nil {
		slog.Warn("parsePTZValues: v4l2-ctl read failed", "device", dev, "error", err)

		//nolint:exhaustruct
		return pixy.PTZValues{}
	}

	var ptz pixy.PTZValues

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		key, rawVal, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		v, parseErr := strconv.Atoi(strings.TrimSpace(rawVal))
		if parseErr != nil {
			continue
		}

		axis, found := v4l2CtrlToAxis[strings.TrimSpace(key)]
		if !found {
			continue
		}

		info := ptzAxes[axis]
		ptz = ptz.Set(axis, v/info.Multiplier)
	}

	return ptz
}

func (d *Daemon) handlePTZCommand(ctx context.Context, parts []string) CommandResult {
	if len(parts) < minCmdParts {
		return okResult(fmt.Sprintf("usage: %s <value>", parts[0]))
	}

	axis := pixy.Axis(parts[0])

	info, ok := ptzAxes[axis]
	if !ok {
		return errResultMsg("unknown PTZ axis: " + string(axis))
	}

	val, relative, parseErr := parsePTZValue(parts[1])
	if parseErr != nil {
		return errResult(string(axis), fmt.Errorf("%w: parse error", ErrInvalidValue))
	}

	d.mu.RLock()
	videoDev := d.videoDev
	d.mu.RUnlock()

	if videoDev == "" {
		return errResult(string(axis), errDeviceNotFound)
	}

	if relative {
		current := d.deps.parsePTZ(ctx, videoDev)
		base, _ := current.Get(axis)
		val = base + val
	}

	val = info.Range.Clamp(val)

	v4l2Err := d.deps.v4l2Set(
		ctx,
		videoDev,
		info.V4L2Ctrl,
		strconv.Itoa(val*info.Multiplier),
	)
	if v4l2Err != nil {
		return errResult(string(axis), v4l2Err)
	}

	// Update cache with known value for immediate accurate readback
	// (avoids stale hardware readback while motor is still moving)
	if cached, valid := d.ptzCache.Get(); valid {
		d.ptzCache.Set(cached.Set(axis, val), ptzCacheTTL)
	} else {
		d.ptzCache.Invalidate()
	}

	// Schedule a delayed hardware readback to correct the cache with
	// the actual motor position (hardware may round or clamp differently).
	d.schedulePTZReadback(ctx, videoDev)

	d.broadcastStateChanged()

	return okResult(fmt.Sprintf("%s set to %d", axis, val))
}

// ptzReadbackDelay is the time to wait after a PTZ set before reading
// back the hardware value. Allows the motor to settle.
const ptzReadbackDelay = 500 * time.Millisecond

func (d *Daemon) schedulePTZReadback(ctx context.Context, videoDev string) {
	readbackCtx := context.WithoutCancel(ctx)

	go func() {
		timer := time.NewTimer(ptzReadbackDelay)
		defer timer.Stop()

		select {
		case <-readbackCtx.Done():
			return
		case <-timer.C:
		}

		actual := d.deps.parsePTZ(readbackCtx, videoDev)
		clamped := actual.Clamp()
		d.ptzCache.Set(clamped, ptzCacheTTL)
		d.broadcastStateChanged()
	}()
}

// parsePTZValue parses a PTZ value string.
// Bare numbers are always absolute (including negatives like "-90").
// Relative mode requires an explicit "rel" prefix (e.g. "rel+10", "rel-5").
// Returns the integer value, whether it's relative, and any parse error.
func parsePTZValue(s string) (int, bool, error) {
	if rest, ok := strings.CutPrefix(s, "rel"); ok {
		v, err := strconv.Atoi(rest)
		if err != nil {
			return 0, false, fmt.Errorf("%s %q: %w", parsePTZValueErrStr, s, err)
		}

		return v, true, nil
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false, fmt.Errorf("%s %q: %w", parsePTZValueErrStr, s, err)
	}

	return v, false, nil
}

// minSpeedCmdParts is the minimum word count of `speed <axis> <value>`.
const minSpeedCmdParts = 3

// maxMotorSpeedSanity is a deliberately wide sanity bound for user-supplied
// motor speeds, NOT a hardware limit: the official protocol transmits the
// value verbatim as float32 and the real per-axis limit (reported by
// GetMotorSpeed) is unknown until the hardware session pins it (plan M27).
// The bound only rejects typos and absurd values before they reach the HID
// device; clamp to the real hardware limit here once it is verified.
const maxMotorSpeedSanity = 10_000

// handleSpeedCommand implements `speed <pan|tilt|zoom> <value>` — the HID
// motor-speed surface (TODO #138). Unlike PTZ, this bypasses V4L2 entirely
// and goes through the official V2 motor command family (motor.go).
//
// The value is passed through verbatim; the physical unit is assumed to be
// degrees/second but is not hardware-verified yet — the CLI response and
// web UI therefore avoid claiming a unit.
func (d *Daemon) handleSpeedCommand(ctx context.Context, parts []string) CommandResult {
	if len(parts) < minSpeedCmdParts {
		return errResultMsg(respSpeedUsage)
	}

	motor, ok := pixy.MotorTypeFromAxis(pixy.Axis(parts[1]))
	if !ok {
		return errResultMsg(respSpeedUsage)
	}

	speed, err := strconv.ParseFloat(parts[2], 32)
	if err != nil || speed < 0 || speed > maxMotorSpeedSanity {
		return errResultMsg(fmt.Sprintf("invalid speed %q (want 0..%d)", parts[2], maxMotorSpeedSanity))
	}

	if err := d.setMotorSpeed(ctx, motor, float32(speed)); err != nil {
		return errResult("speed", err)
	}

	return okResult(fmt.Sprintf("motor speed set: %s %g", motor, speed))
}

// handleTrackingVariantCommand implements `tracking <face|halfbody|fullbody>`
// (TODO #140): the mode-aware tracking layer one level below the binary
// track/idle/privacy switch. Unlike the camera mode, the variant is NOT
// persisted to state.json in v1 — the daemon re-asserts the persisted camera
// mode on device re-appear, but the variant resets (verified M27).
func (d *Daemon) handleTrackingVariantCommand(ctx context.Context, parts []string) CommandResult {
	if len(parts) < minCmdParts {
		return errResultMsg(respTrackingUsage)
	}

	mode, ok := pixy.ParseTargetTrackMode(parts[1])
	if !ok {
		return errResultMsg(respTrackingUsage)
	}

	if err := d.setTargetTrack(ctx, mode); err != nil {
		return errResult("tracking", err)
	}

	return okResult("tracking variant: " + mode.String())
}
