//go:build linux

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	Min        int
	Max        int
	Label      string
	Unit       string
	V4L2Ctrl   string
	Multiplier int
}

//nolint:gochecknoglobals
var ptzAxes = map[pixy.Axis]ptzAxisInfo{
	pixy.AxisPan: {
		Min: pixy.PanRange.Min, Max: pixy.PanRange.Max, Label: "Pan", Unit: "\u00b0",
		V4L2Ctrl: "pan_absolute", Multiplier: v4l2UnitsPerDegree,
	},
	pixy.AxisTilt: {
		Min: pixy.TiltRange.Min, Max: pixy.TiltRange.Max, Label: "Tilt", Unit: "\u00b0",
		V4L2Ctrl: "tilt_absolute", Multiplier: v4l2UnitsPerDegree,
	},
	pixy.AxisZoom: {
		Min: pixy.ZoomRange.Min, Max: pixy.ZoomRange.Max, Label: "Zoom", Unit: "x",
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

func ptzAxisValue(axis pixy.Axis, status webStatus) int {
	return status.Get(axis)
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(hi, v))
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
		val = current.Get(axis) + val
	}

	val = clampInt(val, info.Min, info.Max)

	v4l2Err := d.deps.v4l2Set(
		ctx,
		videoDev,
		info.V4L2Ctrl,
		strconv.Itoa(val*info.Multiplier),
	)
	if v4l2Err != nil {
		return errResult(string(axis), v4l2Err)
	}

	d.ptzCache.Invalidate()
	d.broadcastStateChanged()

	return okResult(fmt.Sprintf("%s set to %d", axis, val))
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
