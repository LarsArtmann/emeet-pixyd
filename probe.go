//go:build linux

package main

import (
	"errors"
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

// parseUeventLine extracts the (vendor, product) pair from one
// "prefix=..." line, or reports not-ok when the line has a different prefix
// or cannot be parsed (uevent files can have spurious continuation lines —
// callers keep scanning instead of treating this as a mismatch).
func parseUeventLine(line, prefix, sep string, vendorIdx, productIdx int) (int64, int64, bool) {
	value, hasPrefix := strings.CutPrefix(line, prefix)
	if !hasPrefix {
		return 0, 0, false
	}

	parts := strings.Split(value, sep)
	if vendorIdx < 0 || productIdx < 0 || len(parts) <= max(vendorIdx, productIdx) {
		return 0, 0, false
	}

	vendor, vErr := strconv.ParseInt(parts[vendorIdx], 16, 0)

	product, pErr := strconv.ParseInt(parts[productIdx], 16, 0)
	if vErr != nil || pErr != nil {
		return 0, 0, false
	}

	return vendor, product, true
}

// pixyModelFromUevent reports which PIXY-family model a "prefix=v/p/..."
// line (separated by sep) in ueventData identifies, where vendor and product
// sit at the given indices. It scans all lines with that prefix; a match
// anywhere counts. extraProductIDs are user-configured PIDs treated as
// PIXY-family variants.
func pixyModelFromUevent(
	ueventData []byte,
	prefix, sep string,
	vendorIdx, productIdx int,
	extraProductIDs []int64,
) (pixy.Model, bool) {
	for line := range strings.SplitSeq(string(ueventData), "\n") {
		vendor, product, ok := parseUeventLine(line, prefix, sep, vendorIdx, productIdx)
		if !ok || vendor != int64(pixyVendorIDInt) {
			continue
		}

		if profile, known := pixy.ResolveProductID(product, extraProductIDs); known &&
			profile.Family == pixy.FamilyPIXY {
			return profile.Model, true
		}
	}

	return "", false
}

// unsupportedEMEETFromUevent describes an EMEET device in a
// "prefix=v/p/..." uevent line that the daemon recognizes but cannot
// control, or "" when the line holds no such device (controllable PIXY
// devices and non-EMEET hardware both yield ""). devicePath is interpolated
// into the hint so users know which node to look at.
func unsupportedEMEETFromUevent(
	ueventData []byte,
	prefix, sep string,
	vendorIdx, productIdx int,
	devicePath string,
) string {
	for line := range strings.SplitSeq(string(ueventData), "\n") {
		vendor, product, ok := parseUeventLine(line, prefix, sep, vendorIdx, productIdx)
		if !ok || vendor != int64(pixyVendorIDInt) {
			continue
		}

		profile, known := pixy.ResolveProductID(product, nil)
		if known && profile.Family == pixy.FamilyPIXY {
			return ""
		}

		model := ""
		if known {
			model = string(profile.Model)
		}

		return pixy.UnsupportedDeviceHint(model, product) + " (at " + devicePath + ")"
	}

	return ""
}

func probeVideo4linux(sysfsPath string, extraProductIDs []int64) (string, pixy.Model, string) {
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return "", "", ""
	}

	unsupported := ""

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

		if model, isPixy := pixyModelFromUevent(ueventData, "PRODUCT=", "/", 0, 1, extraProductIDs); isPixy {
			return videoPath, model, ""
		}

		if unsupported == "" {
			unsupported = unsupportedEMEETFromUevent(ueventData, "PRODUCT=", "/", 0, 1, videoPath)
		}
	}

	return "", "", unsupported
}

func probeHidraw(sysfsPath string, extraProductIDs []int64) (string, pixy.Model) {
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
				if model, isPixy := pixyModelFromUevent(ueventData, "HID_ID=", ":", 1, 2, extraProductIDs); isPixy &&
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

	// UnsupportedHint explains an EMEET device the probe recognized but
	// cannot control. It is only set when no PIXY-family video device was
	// found, so a real PIXY always wins over the hint.
	UnsupportedHint string
}

// unsupportedWarnInterval bounds how often the recognized-but-uncontrolled
// EMEET device hint is repeated. The condition is stable while the device
// stays plugged in, so per-probe logging would flood the journal.
const unsupportedWarnInterval = time.Hour

// unsupportedWarnLimiter rate-limits the unsupported-device hint.
//
//nolint:gochecknoglobals // package-level by design: probes are plain functions
var unsupportedWarnLimiter = newWarnLimiter(unsupportedWarnInterval)

func probeDevices(extraProductIDs []int64) probeResult {
	recordProbe()

	videoDev, videoModel, unsupported := probeVideo4linux("/sys/class/video4linux", extraProductIDs)
	hidrawDev, hidrawModel := probeHidraw("/sys/class/hidraw", extraProductIDs)

	result := probeResult{
		VideoDev:        videoDev,
		HidrawDev:       hidrawDev,
		Model:           hidrawModel,
		UnsupportedHint: unsupported,
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
	case result.UnsupportedHint != "":
		if unsupportedWarnLimiter.allow(result.UnsupportedHint) {
			slog.Info("EMEET device recognized without daemon support", "hint", result.UnsupportedHint)
		}
	}

	return result
}

// warnInaccessibleDevices checks that the probed device nodes are actually
// openable. A device present in sysfs but denied in /dev (missing udev
// rules, wrong group) otherwise only surfaces later as cryptic
// "Permission denied" errors on the first HID command or stream start.
func warnInaccessibleDevices(r probeResult) {
	warnInaccessibleDevicesLimited(r, nil)
}

// deviceAccessWarnInterval bounds repeat accessibility warnings per node on
// the hotplug path: a flapping USB connection re-triggers the uevent-appear
// branch, and the condition persists until the user installs the udev rules.
const deviceAccessWarnInterval = time.Hour

// deviceAccessWarnLimiter rate-limits the hotplug-path accessibility warning.
//
//nolint:gochecknoglobals // package-level by design: probes are plain functions
var deviceAccessWarnLimiter = newWarnLimiter(deviceAccessWarnInterval)

// warnInaccessibleDevicesLimited is warnInaccessibleDevices with an optional
// rate limiter keyed by device node; a nil limiter warns unconditionally.
func warnInaccessibleDevicesLimited(r probeResult, limiter *warnLimiter) {
	check := func(path, kind string) {
		if limiter != nil && !limiter.allow(path) {
			return
		}

		file, err := os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			if closeErr := file.Close(); closeErr != nil {
				slog.Debug("probe file close failed", "path", path, "err", closeErr)
			}

			return
		}

		if errors.Is(err, os.ErrPermission) {
			slog.Warn(
				kind+" device present but not accessible — install the udev rules (NixOS module) or add your user to the required group",
				"device",
				path,
				"error",
				err,
			)
		}
	}

	if r.HidrawDev != "" {
		check(r.HidrawDev, "hidraw")
	}

	if r.VideoDev != "" {
		check(r.VideoDev, "video")
	}
}

// applyProbeResultLocked updates the daemon's view of the PIXY device from a
// probe result. The caller MUST hold d.mu (write lock) for the duration of
// the call; all field writes are unsynchronized. Centralizing the write here
// keeps the lock contract in one place and lets the race detector verify it.
func (d *Daemon) applyProbeResultLocked(r probeResult) {
	d.videoDev = r.VideoDev
	d.hidrawDev = r.HidrawDev
	d.model = r.Model
	d.unsupportedHint = r.UnsupportedHint

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
