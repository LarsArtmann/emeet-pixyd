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
	errorfamily "github.com/larsartmann/go-error-family"
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

// meanFrameLuma grabs a few MJPEG frames via v4l2-ctl and returns the mean
// luma (0-255) of the LAST one. A covered lens (privacy) reads far darker
// than an open one — the physical ground truth the mode queries cannot
// provide. Later frames are used so auto-exposure can settle after a lens
// state change; the buffer's last JPEG is extracted directly.
func meanFrameLuma(t *testing.T, videoDev string) (float64, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var buf bytes.Buffer

	cmd := exec.CommandContext(
		ctx, v4l2ctl, "-d", videoDev,
		"--set-fmt-video=width=640,height=360,pixelformat=MJPG",
		"--stream-mmap", "--stream-count=4", "--stream-to=/dev/stdout",
	)
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return 0, err
	}

	data := buf.Bytes()

	idx := bytes.LastIndex(data, []byte{jpegMarker, jpegSOI, 0xFF})
	if idx < 0 || idx+2 >= len(data) {
		return 0, errNoJPEGFrame
	}

	img, err := jpeg.Decode(bytes.NewReader(data[idx:]))
	if err != nil {
		return 0, err
	}

	return luma(img), nil
}

// errNoJPEGFrame marks a capture that produced no decodable frame.
var errNoJPEGFrame error = errorfamily.NewTransient("test.no_jpeg_frame", "no JPEG frame in capture")

// lumaDark / lumaBright split the optical verdicts: a covered lens reads near
// zero, an open one reads room-light levels (measured 115-126 on the wired
// unit). The gap is wide enough that these bounds are conservative.
const (
	lumaDark   = 30.0
	lumaBright = 60.0
)

// TestIntegration_PrivacyTrapRecovery measures privacy optically (a covered
// lens is dark) and answers the reported bug with clean sequencing: NO HID
// queries run between the trap and the recovery attempts, so nothing pokes
// the mode interface and contaminates the result.
func TestIntegration_PrivacyTrapRecovery(t *testing.T) {
	probeResult := probeDevices(nil)

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)

	frameLuma := func(step string) float64 {
		t.Helper()

		luma, err := meanFrameLuma(t, probeResult.VideoDev)
		if err != nil {
			t.Logf("[%-34s] frame: capture failed: %v", step, err)

			return -1
		}

		values := d.deps.parsePTZ(t.Context(), probeResult.VideoDev)
		tilt, _ := values.Get(pixy.AxisTilt)
		t.Logf("[%-34s] frame: mean luma=%.1f tilt=%d believed=%s", step, luma, tilt, readCameraState(d))

		return luma
	}

	// Sanity: camera open and the room is lit — otherwise luma verdicts are
	// meaningless.
	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)

	if baseline := frameLuma("baseline: tracking, tilt 0"); baseline < lumaBright {
		t.Skipf("baseline luma %.1f too dark to judge privacy optically — light the room", baseline)
	}

	// Confirm the mode write path optically before judging the trap.
	runDaemonCommand(t, d, "privacy")
	time.Sleep(motorSettleWait)

	if dark := frameLuma("privacy commanded"); dark >= lumaBright {
		t.Errorf("privacy commanded but frame stays bright (luma=%.1f) — mode write did not cover the lens", dark)
	}

	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)

	if open := frameLuma("track commanded"); open < lumaBright {
		t.Errorf("track commanded but frame stays dark (luma=%.1f) — mode write did not uncover the lens", open)
	}

	// The reported trap: tilt to -85 while tracking.
	for _, tiltDeg := range []int{-85, -90} {
		runDaemonCommand(t, d, "tilt "+strconv.Itoa(tiltDeg))
		time.Sleep(motorSettleWait)

		if dark := frameLuma("tilt " + strconv.Itoa(tiltDeg) + " (trap)"); dark >= lumaDark {
			t.Logf("tilt %d did NOT engage privacy optically (luma=%.1f) — no trap at this angle on this unit", tiltDeg, dark)

			continue
		}

		// THE reported failure: direct privacy→tracking switch while parked.
		runDaemonCommand(t, d, "track")
		time.Sleep(motorSettleWait)

		recovered := frameLuma("track commanded while trapped")
		switch {
		case recovered >= lumaBright:
			t.Logf("tilt %d: direct track RECOVERED the lens (luma=%.1f)", tiltDeg, recovered)
		case recovered >= lumaDark:
			t.Logf("tilt %d: direct track left the lens HALF recovered (luma=%.1f)", tiltDeg, recovered)
		default:
			t.Errorf("tilt %d: direct track did NOT recover the lens (luma=%.1f) — the reported bug reproduces", tiltDeg, recovered)

			// Recovery fallbacks for the stuck case.
			runDaemonCommand(t, d, "track")
			time.Sleep(motorSettleWait)
			frameLuma("second track")

			runDaemonCommand(t, d, "privacy")
			time.Sleep(motorSettleWait)
			runDaemonCommand(t, d, "track")
			time.Sleep(motorSettleWait)
			frameLuma("privacy then track")

			runDaemonCommand(t, d, "tilt 0")
			time.Sleep(motorSettleWait)
			frameLuma("tilt 0 (v4l2 only)")
			runDaemonCommand(t, d, "track")
			time.Sleep(motorSettleWait)
			frameLuma("tilt 0 then track")
		}
	}

	// Leave the camera open, centered, consistent with the running daemon.
	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	frameLuma("cleanup: tilt 0, tracking")
}

// TestIntegration_PrivacyTrapTimingRace verifies the fix for the reported
// bug on real hardware: a track write landing inside the firmware's trap
// transition is eaten, and the daemon's deferred re-assert (privacy_trap.go)
// must recover the lens within the measurement window. Before the fix every
// delay round below stayed covered (luma 12-34).
func TestIntegration_PrivacyTrapTimingRace(t *testing.T) {
	probeResult := probeDevices(nil)

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)

	frameLuma := func(step string) float64 {
		t.Helper()

		luma, err := meanFrameLuma(t, probeResult.VideoDev)
		if err != nil {
			t.Logf("[%-34s] frame: capture failed: %v", step, err)

			return -1
		}

		t.Logf("[%-34s] frame: mean luma=%.1f believed=%s", step, luma, readCameraState(d))

		return luma
	}

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)

	if baseline := frameLuma("baseline"); baseline < lumaBright {
		t.Skipf("baseline luma %.1f too dark to judge privacy optically — light the room", baseline)
	}

	for _, delay := range []time.Duration{0, 300 * time.Millisecond, 800 * time.Millisecond, 1500 * time.Millisecond} {
		t.Run("delay_"+delay.String(), func(t *testing.T) {
			// Re-arm: open, centered, settled.
			runDaemonCommand(t, d, "tilt 0")
			runDaemonCommand(t, d, "track")
			time.Sleep(motorSettleWait)

			runDaemonCommand(t, d, "tilt -85")
			time.Sleep(delay)
			runDaemonCommand(t, d, "track")

			// Let both the motor travel and the lens mechanism finish.
			time.Sleep(2 * motorSettleWait)

			recovered := frameLuma("track after " + delay.String() + " delay")
			if recovered >= lumaBright {
				t.Logf("delay %s: RECOVERED", delay)

				return
			}

			t.Errorf("delay %s: neither the write nor the trap re-assert recovered the lens (luma=%.1f)", delay, recovered)

			// Cleanup for the next round: settled double-write.
			runDaemonCommand(t, d, "tilt 0")
			time.Sleep(motorSettleWait)
			runDaemonCommand(t, d, "track")
			time.Sleep(motorSettleWait)
			frameLuma("round cleanup")
		})
	}

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	frameLuma("cleanup: tilt 0, tracking")
}

// TestIntegration_TrapRecoveryMatrix finds which write sequence actually
// recovers the lens AFTER a first track write was eaten by the trap race.
// Candidates: (A) a later same-mode write at +5s — distinguishes window
// extension from state latching; (B) privacy then track — a mode bounce.
func TestIntegration_TrapRecoveryMatrix(t *testing.T) {
	probeResult := probeDevices(nil)

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)

	frameLuma := func(step string) float64 {
		t.Helper()

		luma, err := meanFrameLuma(t, probeResult.VideoDev)
		if err != nil {
			t.Logf("[%-34s] frame: capture failed: %v", step, err)

			return -1
		}

		t.Logf("[%-34s] frame: mean luma=%.1f believed=%s", step, luma, readCameraState(d))

		return luma
	}

	rearm := func() {
		t.Helper()

		runDaemonCommand(t, d, "tilt 0")
		runDaemonCommand(t, d, "track")
		time.Sleep(motorSettleWait)

		if open := frameLuma("re-arm check"); open < lumaBright {
			t.Fatalf("re-arm failed: lens not open (luma=%.1f)", open)
		}
	}

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)

	if baseline := frameLuma("baseline"); baseline < lumaBright {
		t.Skipf("baseline luma %.1f too dark to judge privacy optically — light the room", baseline)
	}

	// Round A: eaten write, then same-mode write at +5s.
	rearm()
	runDaemonCommand(t, d, "tilt -85")
	runDaemonCommand(t, d, "track") // eaten by the race
	time.Sleep(5 * time.Second)
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	if a := frameLuma("A: second track at +5s"); a >= lumaBright {
		t.Log("A: late same-mode write RECOVERS — window extension model")
	} else {
		t.Log("A: late same-mode write does NOT recover — state latching model")
	}

	// Round B: eaten write, then privacy→track bounce.
	rearm()
	runDaemonCommand(t, d, "tilt -85")
	runDaemonCommand(t, d, "track") // eaten by the race
	time.Sleep(3 * time.Second)
	runDaemonCommand(t, d, "privacy")
	time.Sleep(500 * time.Millisecond)
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	if b := frameLuma("B: privacy→track bounce"); b >= lumaBright {
		t.Log("B: privacy→track bounce RECOVERS the poisoned state")
	} else {
		t.Log("B: privacy→track bounce does NOT recover")
	}

	// Round C: eaten write, then idle as the second mode.
	rearm()
	runDaemonCommand(t, d, "tilt -85")
	runDaemonCommand(t, d, "track") // eaten by the race
	time.Sleep(3 * time.Second)
	runDaemonCommand(t, d, "idle")
	time.Sleep(motorSettleWait)
	if c := frameLuma("C: idle as recovery"); c >= lumaBright {
		t.Log("C: idle write RECOVERS the poisoned state")
	} else {
		t.Log("C: idle write does NOT recover")
	}

	// Round D: eaten write → idle recovery → track again while still parked.
	rearm()
	runDaemonCommand(t, d, "tilt -85")
	runDaemonCommand(t, d, "track") // eaten by the race
	time.Sleep(3 * time.Second)
	runDaemonCommand(t, d, "idle")
	time.Sleep(motorSettleWait)
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	if dd := frameLuma("D: idle then track"); dd >= lumaBright {
		t.Log("D: idle→track sequence RECOVERS into tracking — the full bounce fix")
	} else {
		t.Log("D: idle→track ends covered — tracking cannot stick in the zone once poisoned")
	}

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	frameLuma("cleanup: tilt 0, tracking")
}

// TestIntegration_PrivacyTrapBoundaries pins the remaining trap semantics:
// does leaving the tilt zone re-open the lens on its own, and how does idle
// interact with the cover. All verdicts optical, all writes settled.
func TestIntegration_PrivacyTrapBoundaries(t *testing.T) {
	probeResult := probeDevices(nil)

	if probeResult.VideoDev == "" || probeResult.HidrawDev == "" {
		t.Skip("no PIXY device found — connect hardware to run this test")
	}

	d := newTestDaemon(t, pixy.StateIdle, probeResult.VideoDev, probeResult.HidrawDev)

	frameLuma := func(step string) float64 {
		t.Helper()

		luma, err := meanFrameLuma(t, probeResult.VideoDev)
		if err != nil {
			t.Logf("[%-34s] frame: capture failed: %v", step, err)

			return -1
		}

		values := d.deps.parsePTZ(t.Context(), probeResult.VideoDev)
		tilt, _ := values.Get(pixy.AxisTilt)
		t.Logf("[%-34s] frame: mean luma=%.1f tilt=%d believed=%s", step, luma, tilt, readCameraState(d))

		return luma
	}

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)

	if baseline := frameLuma("baseline: tracking, tilt 0"); baseline < lumaBright {
		t.Skipf("baseline luma %.1f too dark to judge privacy optically — light the room", baseline)
	}

	// Trap, then leave the zone with NO mode write in between.
	runDaemonCommand(t, d, "tilt -85")
	time.Sleep(motorSettleWait)
	frameLuma("tilt -85 (trap engaged)")

	runDaemonCommand(t, d, "tilt 0")
	time.Sleep(motorSettleWait)
	if up := frameLuma("tilt 0 after trap (no mode write)"); up >= lumaBright {
		t.Log("leaving the tilt zone re-opens the lens on its own")
	} else {
		t.Log("leaving the tilt zone does NOT re-open the lens — a mode write is required")
	}

	// Idle behavior: from open, and from trapped.
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)

	runDaemonCommand(t, d, "idle")
	time.Sleep(motorSettleWait)
	if idle := frameLuma("idle from open"); idle >= lumaBright {
		t.Log("idle leaves the lens open")
	} else {
		t.Log("idle covers the lens")
	}

	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	runDaemonCommand(t, d, "tilt -85")
	time.Sleep(motorSettleWait)
	frameLuma("tilt -85 (trap re-engaged)")

	runDaemonCommand(t, d, "idle")
	time.Sleep(motorSettleWait)
	if idle := frameLuma("idle while trapped"); idle >= lumaBright {
		t.Log("settled idle write recovers the trapped lens")
	} else {
		t.Log("settled idle write does NOT recover the trapped lens")
	}

	runDaemonCommand(t, d, "tilt 0")
	runDaemonCommand(t, d, "track")
	time.Sleep(motorSettleWait)
	frameLuma("cleanup: tilt 0, tracking")
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
