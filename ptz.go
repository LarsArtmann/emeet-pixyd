//go:build linux

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const v4l2ctl = "v4l2-ctl"

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
var ptzAxes = map[string]ptzAxisInfo{
	pixy.AxisPan: {
		Min: pixy.PanMin, Max: pixy.PanMax, Label: "Pan", Unit: "\u00b0",
		V4L2Ctrl: "pan_absolute", Multiplier: v4l2UnitsPerDegree,
	},
	pixy.AxisTilt: {
		Min: pixy.TiltMin, Max: pixy.TiltMax, Label: "Tilt", Unit: "\u00b0",
		V4L2Ctrl: "tilt_absolute", Multiplier: v4l2UnitsPerDegree,
	},
	pixy.AxisZoom: {
		Min: pixy.ZoomMin, Max: pixy.ZoomMax, Label: "Zoom", Unit: "x",
		V4L2Ctrl: "zoom_absolute", Multiplier: 1,
	},
}

// ptzAxisOrder defines the deterministic order for V4L2 control listing.
//
//nolint:gochecknoglobals
var ptzAxisOrder = []string{pixy.AxisPan, pixy.AxisTilt, pixy.AxisZoom}

// v4l2CtrlToAxis maps V4L2 control names back to PTZ axis names.
//
//nolint:gochecknoglobals
var v4l2CtrlToAxis = buildCtrlToAxis()

func buildCtrlToAxis() map[string]string {
	m := make(map[string]string, len(ptzAxes))
	for axis, info := range ptzAxes {
		m[info.V4L2Ctrl] = axis
	}

	return m
}

func ptzAxisValid(axis string) bool {
	_, ok := ptzAxes[axis]

	return ok
}

func ptzAxisValue(axis string, status webStatus) int {
	return status.Get(axis)
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(hi, v))
}

func v4l2Set(ctx context.Context, dev, ctrl, value string) error {
	err := exec.CommandContext(ctx, v4l2ctl, "-d", dev, "--set-ctrl="+ctrl+"="+value).
		Run()
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

func parsePTZValues(ctx context.Context, dev string) pixy.PTZValues {
	out, err := exec.CommandContext(
		ctx, v4l2ctl, "-d", dev,
		"--get-ctrl="+v4l2GetCtrlList(),
	).Output()
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
