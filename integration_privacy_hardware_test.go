//go:build integration

// Hardware diagnostic for the reported tilt/privacy trap: tilting to -85° or
// below appears to push the camera into privacy on its own, and a direct
// privacy→tracking switch then fails to recover (at least from the web UI).
//
// This test walks the exact scenario on real hardware, logging the mode as
// seen through BOTH query surfaces (our V1 SET-head query 09 01 01 01 and the
// official V2 device-mode query 09 02 01 00) plus the V4L2 tilt readback at
// every step. It is primarily a diagnostic: unexpected findings are logged,
// not failed on — the goal is evidence, not a red test.
//
// Run with: go test -tags=integration -run TestIntegration_TiltPrivacyTrap -v .

package main

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// v1TrackingQuery is our classic SET-head mode query (hid.go queryTracking).
var v1TrackingQuery = []byte{cameraConfigPrefix, hidInterfaceTracking, 0x01, 0x01}

// v2DeviceModeQuery is the official CMD_GET_DEVICE_MODE head (Dev 0x02).
var v2DeviceModeQuery = []byte{0x09, 0x02, 0x01, 0x00}

// motorSettleWait leaves the head time to finish large tilt travels before
// readback. Generous on purpose: the trap may depend on the motor reaching
// the bottom, not on the command being issued.
const motorSettleWait = 3 * time.Second

func logHardwareSnapshot(t *testing.T, d *Daemon, dev HIDDevice, step string) {
	t.Helper()

	raw, rawErr := dev.SendRecv(t.Context(), v1TrackingQuery)
	switch {
	case rawErr != nil:
		t.Logf("[%-34s] v1 query 09 01 01 01: error: %v", step, rawErr)
	case len(raw) == 0:
		t.Logf("[%-34s] v1 query 09 01 01 01: no response (timeout)", step)
	default:
		mode, err := d.queryTracking(t.Context())
		if err != nil {
			t.Logf("[%-34s] v1 query 09 01 01 01: %x → parse error: %v", step, raw, err)
		} else {
			t.Logf("[%-34s] v1 query 09 01 01 01: %x → tracking=%s", step, raw, mode)
		}
	}

	resp, err := dev.SendRecv(t.Context(), v2DeviceModeQuery)
	switch {
	case err != nil:
		t.Logf("[%-34s] v2 device-mode 09 02 01 00: error: %v", step, err)
	case len(resp) == 0:
		t.Logf("[%-34s] v2 device-mode 09 02 01 00: no response (timeout)", step)
	default:
		t.Logf("[%-34s] v2 device-mode 09 02 01 00: %d bytes: %x", step, len(resp), resp)
	}

	values := d.deps.parsePTZ(t.Context(), d.videoDevice())
	tilt, _ := values.Get(pixy.AxisTilt)
	t.Logf("[%-34s] v4l2 readback: tilt=%d (believed=%s)", step, tilt, readCameraState(d))
}

func runDaemonCommand(t *testing.T, d *Daemon, cmd string) {
	t.Helper()

	result := d.handleCommand(t.Context(), cmd)
	if result.IsError() {
		t.Errorf("command %q failed: %v", cmd, result)
		return
	}

	t.Logf("    -> %q: %s", cmd, result.String())
}

func TestIntegration_TiltPrivacyTrap(t *testing.T) {
	probeResult := probeDevices(nil)

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)
	dev := newHIDRawDevice(probeResult.HidrawDev)

	// Baseline: known mode and position.
	runDaemonCommand(t, d, "track")
	runDaemonCommand(t, d, "tilt 0")
	time.Sleep(motorSettleWait)
	logHardwareSnapshot(t, d, dev, "baseline: tracking, tilt 0")

	// Controlled mode flip for query-surface decoding.
	runDaemonCommand(t, d, "privacy")
	time.Sleep(500 * time.Millisecond)
	logHardwareSnapshot(t, d, dev, "privacy commanded")

	runDaemonCommand(t, d, "track")
	time.Sleep(500 * time.Millisecond)
	logHardwareSnapshot(t, d, dev, "tracking commanded")

	// Approach the reported trap zone from above.
	for _, tilt := range []int{-60, -80, -85, -90} {
		runDaemonCommand(t, d, "tilt "+strconv.Itoa(tilt))
		time.Sleep(motorSettleWait)
		logHardwareSnapshot(t, d, dev, "tilt "+strconv.Itoa(tilt)+" (tracking)")
	}

	// The reported failure: direct privacy→tracking switch while parked down.
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	logHardwareSnapshot(t, d, dev, "track commanded while down")

	// Recovery attempt A: explicit privacy first, then tracking.
	runDaemonCommand(t, d, "privacy")
	time.Sleep(motorSettleWait)
	logHardwareSnapshot(t, d, dev, "privacy commanded (recovery A)")

	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	logHardwareSnapshot(t, d, dev, "track after privacy (recovery A)")

	// Recovery attempt B: center (pure V4L2 move) then mode.
	runDaemonCommand(t, d, "center")
	time.Sleep(motorSettleWait)
	logHardwareSnapshot(t, d, dev, "center (recovery B)")

	// Leave the hardware in a safe, consistent state: parked and covered.
	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "privacy")
	time.Sleep(motorSettleWait)
	logHardwareSnapshot(t, d, dev, "cleanup: tilt 0, privacy")
}

// meanFrameLuma grabs one MJPEG frame via v4l2-ctl and returns its mean luma
// (0-255). A covered lens (privacy) reads far darker than an open one — the
// physical ground truth the mode queries cannot provide.
func meanFrameLuma(t *testing.T, videoDev string) (float64, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var buf bytes.Buffer

	cmd := exec.CommandContext(
		ctx, v4l2ctl, "-d", videoDev,
		"--set-fmt-video=width=640,height=360,pixelformat=MJPG",
		"--stream-mmap", "--stream-count=1", "--stream-to=/dev/stdout",
	)
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return 0, err
	}

	img, err := jpeg.Decode(&buf)
	if err != nil {
		return 0, err
	}

	return luma(img), nil
}

func logFrameTruth(t *testing.T, d *Daemon, step string) {
	t.Helper()

	luma, err := meanFrameLuma(t, d.videoDevice())
	if err != nil {
		t.Logf("[%-34s] frame: capture failed: %v", step, err)

		return
	}

	values := d.deps.parsePTZ(t.Context(), d.videoDevice())
	tilt, _ := values.Get(pixy.AxisTilt)
	t.Logf("[%-34s] frame: mean luma=%.1f tilt=%d believed=%s", step, luma, tilt, readCameraState(d))
}

// TestIntegration_PrivacyGroundTruth measures privacy optically: a covered
// lens is dark, an open one is not. This decouples the investigation from the
// mode-query surfaces, which answer constant bytes on the wired firmware.
func TestIntegration_PrivacyGroundTruth(t *testing.T) {
	probeResult := probeDevices(nil)

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)
	dev := newHIDRawDevice(probeResult.HidrawDev)

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "baseline: tracking, tilt 0")

	runDaemonCommand(t, d, "privacy")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "privacy commanded")

	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "track commanded")

	runDaemonCommand(t, d, "tilt -85")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "tilt -85 (the reported trap)")
	logHardwareSnapshot(t, d, dev, "tilt -85 (the reported trap)")

	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "track commanded while trapped")

	runDaemonCommand(t, d, "tilt 0")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "tilt 0 recovery (v4l2 only)")

	runDaemonCommand(t, d, "privacy")
	time.Sleep(motorSettleWait)
	runDaemonCommand(t, d, "tilt 0")
	time.Sleep(motorSettleWait)
	logFrameTruth(t, d, "cleanup: privacy + tilt 0")
}

// luma returns the mean luma (0-255) of an image. YCbCr JPEGs are averaged
// over the Y plane directly; anything else falls back to At() sampling.
func luma(img image.Image) float64 {
	if ycbcr, ok := img.(*image.YCbCr); ok {
		var sum, count uint64

		for _, v := range ycbcr.Y {
			sum += uint64(v)
			count++
		}

		if count > 0 {
			return float64(sum) / float64(count)
		}
	}

	bounds := img.Bounds()

	if bounds.Empty() {
		return 0
	}

	var sum float64

	count := 0.0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / count
}
