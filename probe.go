//go:build linux

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	pixyVendorIDInt = 0x328f

	// ueventWarnInterval bounds how often the absent-uevent probe WARN is
	// repeated per path. One hour: the condition is stable while the device
	// topology is unchanged, so per-probe logging only floods the journal
	// (~160 identical lines/day while the PIXY is absent, 2026-09-02).
	ueventWarnInterval = time.Hour
)

// ueventWarnLimiter rate-limits the video4linux uevent-read warning; see
// warnLimiter for the rationale. Package-level because probes are plain
// functions on sysfs state.
//
//nolint:gochecknoglobals // package-level by design: probes are plain functions
var ueventWarnLimiter = newWarnLimiter(ueventWarnInterval)

func isPixyName(name string) bool {
	return strings.Contains(name, "EMEET") ||
		strings.Contains(name, "Pixy") ||
		strings.Contains(name, "PIXY")
}

// pixyModelFromUevent reports which PIXY model a "prefix=v/p/..." line
// (separated by sep) in ueventData identifies, where vendor and product sit
// at the given indices. It scans all lines with that prefix; a match anywhere
// counts, but lines that don't have enough parts are skipped (not treated
// as a mismatch — uevent files can have spurious continuation lines).
func pixyModelFromUevent(ueventData []byte, prefix, sep string, vendorIdx, productIdx int) (pixy.Model, bool) {
	for line := range strings.SplitSeq(string(ueventData), "\n") {
		value, ok := strings.CutPrefix(line, prefix)
		if !ok {
			continue
		}

		parts := strings.Split(value, sep)
		if len(parts) <= max(vendorIdx, productIdx) {
			continue
		}

		vendor, vErr := strconv.ParseInt(parts[vendorIdx], 16, 0)
		product, pErr := strconv.ParseInt(parts[productIdx], 16, 0)

		if model, isPixy := pixy.ModelFromProductID(product); vErr == nil && pErr == nil &&
			vendor == int64(pixyVendorIDInt) && isPixy {
			return model, true
		}
	}

	return "", false
}

func probeVideo4linux(sysfsPath string) (string, pixy.Model) {
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		name := entry.Name()

		videoPath := "/dev/" + name

		indexFile := fmt.Sprintf("%s/%s/index", sysfsPath, name)

		indexData, iErr := os.ReadFile(indexFile)
		if iErr == nil && strings.TrimSpace(string(indexData)) != "0" {
			continue
		}

		ueventFile := fmt.Sprintf("%s/%s/device/uevent", sysfsPath, name)

		ueventData, uErr := os.ReadFile(ueventFile)
		if uErr != nil {
			if ueventWarnLimiter.allow(ueventFile) {
				slog.Warn("video4linux probe: failed to read uevent", "path", ueventFile, "error", uErr)
			}

			continue
		}

		if model, isPixy := pixyModelFromUevent(ueventData, "PRODUCT=", "/", 0, 1); isPixy {
			return videoPath, model
		}
	}

	return "", ""
}

func probeHidraw(sysfsPath string) (string, pixy.Model) {
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		name := entry.Name()

		hidrawPath := "/dev/" + name

		ueventFile := fmt.Sprintf("%s/%s/device/uevent", sysfsPath, name)

		ueventData, uErr := os.ReadFile(ueventFile)
		if uErr != nil {
			continue
		}

		for line := range strings.SplitSeq(string(ueventData), "\n") {
				if hidName, ok := strings.CutPrefix(line, "HID_NAME="); ok {
					if model, isPixy := pixyModelFromUevent(ueventData, "HID_ID=", ":", 1, 2); isPixy &&
						isPixyName(hidName) {
						return hidrawPath, model
					}
				}
			}
		}

		return "", ""
	}

type probeResult struct {
	VideoDev  string
	HidrawDev string
	Model     pixy.Model
}

func probeDevices() probeResult {
	recordProbe()

	videoDev, videoModel := probeVideo4linux("/sys/class/video4linux")
	hidrawDev, hidrawModel := probeHidraw("/sys/class/hidraw")

	result := probeResult{
		VideoDev:  videoDev,
		HidrawDev: hidrawDev,
		Model:     hidrawModel,
	}
	if result.Model == "" {
		result.Model = videoModel
	}

	switch {
	case result.VideoDev != "" && result.HidrawDev != "":
		slog.Info("found PIXY device", "model", result.Model, "video", result.VideoDev, "hidraw", result.HidrawDev)
	case result.VideoDev != "" && result.HidrawDev == "":
		slog.Warn("partial PIXY device: video found but no hidraw", "model", result.Model, "video", result.VideoDev)
	case result.VideoDev == "" && result.HidrawDev != "":
		slog.Warn("partial PIXY device: hidraw found but no video", "model", result.Model, "hidraw", result.HidrawDev)
	}

	return result
}

// applyProbeResultLocked updates the daemon's view of the PIXY device from a
// probe result. The caller MUST hold d.mu (write lock) for the duration of
// the call; all field writes are unsynchronized. Centralizing the write here
// keeps the lock contract in one place and lets the race detector verify it.
func (d *Daemon) applyProbeResultLocked(r probeResult) {
	d.videoDev = r.VideoDev
	d.hidrawDev = r.HidrawDev
	d.model = r.Model

	if r.HidrawDev != "" {
		d.hidDev = newHIDRawDevice(r.HidrawDev)
	} else {
		d.hidDev = nil
	}

	if r.VideoDev != "" && r.HidrawDev != "" {
		d.hidFailCount = 0
		if d.state.Camera == pixy.StateOffline {
			d.state.Camera = pixy.StatePrivacy
		}
	} else {
		d.state.Camera = pixy.StateOffline
	}
}
