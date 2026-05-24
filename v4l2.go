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

const (
	v4l2DegreesPerUnit = 3600

	axisPan  = "pan"
	axisTilt = "tilt"
	axisZoom = "zoom"
)

func v4l2Set(ctx context.Context, dev, ctrl, value string) error {
	err := exec.CommandContext(ctx, "v4l2-ctl", "-d", dev, "--set-ctrl="+ctrl+"="+value).
		Run()
	if err != nil {
		return fmt.Errorf("v4l2Set %s=%s on %s: %w", ctrl, value, dev, err)
	}

	return nil
}

func parsePTZValues(ctx context.Context, dev string) pixy.PTZValues {
	out, err := exec.CommandContext(
		ctx, "v4l2-ctl", "-d", dev,
		"--get-ctrl=pan_absolute,tilt_absolute,zoom_absolute",
	).Output()
	if err != nil {
		//nolint:exhaustruct
		return pixy.PTZValues{}
	}

	var ptz pixy.PTZValues

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		v, parseErr := strconv.Atoi(strings.TrimSpace(val))
		if parseErr != nil {
			continue
		}

		switch strings.TrimSpace(key) {
		case "pan_absolute":
			ptz.Pan = v / v4l2DegreesPerUnit
		case "tilt_absolute":
			ptz.Tilt = v / v4l2DegreesPerUnit
		case "zoom_absolute":
			ptz.Zoom = v
		}
	}

	return ptz
}
