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
