//go:build linux

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	pixyVendorIDInt  = 0x328f
	pixyProductIDInt = 0x00c0
)

func isPixyName(name string) bool {
	return strings.Contains(name, "EMEET") ||
		strings.Contains(name, "Pixy") ||
		strings.Contains(name, "PIXY")
}

func matchesPixyID(ueventData []byte, prefix, sep string, vendorIdx, productIdx int) bool {
	for line := range strings.SplitSeq(string(ueventData), "\n") {
		value, ok := strings.CutPrefix(line, prefix)
		if !ok {
			continue
		}

		parts := strings.Split(value, sep)
		if len(parts) <= max(vendorIdx, productIdx) {
			return false
		}

		vendor, vErr := strconv.ParseInt(parts[vendorIdx], 16, 0)
		product, pErr := strconv.ParseInt(parts[productIdx], 16, 0)

		return vErr == nil && pErr == nil &&
			vendor == int64(pixyVendorIDInt) && product == int64(pixyProductIDInt)
	}

	return false
}

func probeVideo4linux(sysfsPath string) string {
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return ""
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
			slog.Warn("video4linux probe: failed to read uevent", "path", ueventFile, "error", uErr)

			continue
		}

		if matchesPixyID(ueventData, "PRODUCT=", "/", 0, 1) {
			return videoPath
		}
	}

	return ""
}

func probeHidraw(sysfsPath string) string {
	entries, err := os.ReadDir(sysfsPath)
	if err != nil {
		return ""
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
				if isPixyName(hidName) && matchesPixyID(ueventData, "HID_ID=", ":", 1, 2) {
					return hidrawPath
				}
			}
		}
	}

	return ""
}

type probeResult struct {
	VideoDev  string
	HidrawDev string
}

func probeDevices() probeResult {
	recordProbe()

	result := probeResult{
		VideoDev:  probeVideo4linux("/sys/class/video4linux"),
		HidrawDev: probeHidraw("/sys/class/hidraw"),
	}
	switch {
	case result.VideoDev != "" && result.HidrawDev != "":
		slog.Info("found PIXY device", "video", result.VideoDev, "hidraw", result.HidrawDev)
	case result.VideoDev != "" && result.HidrawDev == "":
		slog.Warn("partial PIXY device: video found but no hidraw", "video", result.VideoDev)
	case result.VideoDev == "" && result.HidrawDev != "":
		slog.Warn("partial PIXY device: hidraw found but no video", "hidraw", result.HidrawDev)
	}

	return result
}

func (d *Daemon) applyProbeResult(r probeResult) {
	d.videoDev = r.VideoDev
	d.hidrawDev = r.HidrawDev

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
