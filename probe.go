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
	pixyVendorID  = "328f"
	pixyProductID = "00c0"
)

func isPixyName(name string) bool {
	return strings.Contains(name, "EMEET") ||
		strings.Contains(name, "Pixy") ||
		strings.Contains(name, "PIXY")
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

		if hasPixyProduct(ueventData) {
			return videoPath
		}
	}

	return ""
}

func hasPixyProduct(ueventData []byte) bool {
	for line := range strings.SplitSeq(string(ueventData), "\n") {
		product, ok := strings.CutPrefix(line, "PRODUCT=")
		if !ok {
			continue
		}

		parts := strings.Split(product, "/")
		if len(parts) < 2 {
			return false
		}

		vendor, vErr := strconv.ParseInt(parts[0], 16, 0)
		prod, pErr := strconv.ParseInt(parts[1], 16, 0)
		expectedVendor, evErr := strconv.ParseInt(pixyVendorID, 16, 0)
		expectedProduct, epErr := strconv.ParseInt(pixyProductID, 16, 0)

		return vErr == nil && pErr == nil && evErr == nil && epErr == nil &&
			vendor == expectedVendor && prod == expectedProduct
	}

	return false
}

func hasPixyVendorProduct(ueventData []byte) bool {
	for line := range strings.SplitSeq(string(ueventData), "\n") {
		hidID, ok := strings.CutPrefix(line, "HID_ID=")
		if !ok {
			continue
		}

		parts := strings.Split(hidID, ":")
		if len(parts) != 3 {
			continue
		}

		vendor, vErr := strconv.ParseInt(parts[1], 16, 0)
		product, pErr := strconv.ParseInt(parts[2], 16, 0)
		expectedVendor, evErr := strconv.ParseInt(pixyVendorID, 16, 0)
		expectedProduct, epErr := strconv.ParseInt(pixyProductID, 16, 0)

		return vErr == nil && pErr == nil && evErr == nil && epErr == nil &&
			vendor == expectedVendor && product == expectedProduct
	}

	return false
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
				if isPixyName(hidName) && hasPixyVendorProduct(ueventData) {
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
	result := probeResult{
		VideoDev:  probeVideo4linux("/sys/class/video4linux"),
		HidrawDev: probeHidraw("/sys/class/hidraw"),
	}
	if result.VideoDev != "" && result.HidrawDev != "" {
		slog.Info("found PIXY device", "video", result.VideoDev, "hidraw", result.HidrawDev)
	}

	return result
}

func (d *Daemon) applyProbeResult(r probeResult) {
	d.videoDev = r.VideoDev
	d.hidrawDev = r.HidrawDev

	if r.VideoDev != "" && r.HidrawDev != "" {
		if d.state.Camera == pixy.StateOffline {
			d.state.Camera = pixy.StatePrivacy
		}
	} else {
		d.state.Camera = pixy.StateOffline
	}
}
