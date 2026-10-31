//go:build linux

package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// TestHandleQueryCommand_Probe_HintTakesPrecedence pins the probe command's
// no-device contract: the probe's unsupported-hint answers in place of the
// generic not-found response. probeDevices is injected so the test stays
// hermetic (the default scans real sysfs).
func TestHandleQueryCommand_Probe_HintTakesPrecedence(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)
	d.deps.probeDevices = func([]int64) probeResult {
		return probeResult{UnsupportedHint: testUnsupportedHint}
	}

	resp := d.handleQueryCommand(context.Background(), []string{cmdProbe})

	if resp.String() == respDeviceNotFound {
		t.Errorf("expected the unsupported hint to take precedence over %q", respDeviceNotFound)
	}

	if !strings.Contains(resp.String(), "EMEET C960") {
		t.Errorf("expected the hint text in the response, got: %s", resp.String())
	}
}

// TestHandleQueryCommand_Probe_DeviceFoundWinsOverHint pins the precedence
// precondition the other way around: a found PIXY answers "device found"
// even when the probe also carried an unsupported hint — a real PIXY always
// wins over the hint (probeResult.UnsupportedHint contract).
func TestHandleQueryCommand_Probe_DeviceFoundWinsOverHint(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)
	d.deps.probeDevices = func([]int64) probeResult {
		return probeResult{
			VideoDev:        "/dev/video9",
			HidrawDev:       "/dev/hidraw9",
			Model:           pixy.Model2K,
			UnsupportedHint: testUnsupportedHint,
		}
	}

	resp := d.handleQueryCommand(context.Background(), []string{cmdProbe})

	assertStatusPrefix(t, resp.String(), "device found: /dev/video9", "probe with device")
	if strings.Contains(resp.String(), "C960") {
		t.Errorf("hint leaked into the found-device response: %s", resp.String())
	}
}

// TestLogUnsupportedHintLimited_RateLimitedPerHint proves the hint log fires
// once per hint text per interval (the condition is stable while the device
// stays plugged; per-probe logging would flood the journal) and that
// different hint texts rate-limit independently.
func TestLogUnsupportedHintLimited_RateLimitedPerHint(t *testing.T) { //nolint:paralleltest // mutates global slog
	var buf bytes.Buffer

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	limiter := newWarnLimiter(unsupportedWarnInterval)
	now := time.Unix(0, 0)
	limiter.now = func() time.Time { return now }

	logUnsupportedHintLimited(testUnsupportedHint, limiter) // first sight: logged

	now = now.Add(time.Minute)

	logUnsupportedHintLimited(testUnsupportedHint, limiter) // within interval: suppressed
	logUnsupportedHintLimited("other EMEET device", limiter) // different key: logged

	now = now.Add(unsupportedWarnInterval)

	logUnsupportedHintLimited(testUnsupportedHint, limiter) // interval elapsed: logged again

	got := strings.Count(buf.String(), "recognized without daemon support")
	if got != 3 {
		t.Fatalf("expected exactly 3 hint logs (first, other key, post-interval), got %d (buffer: %q)", got, buf.String())
	}
}

// TestUnsupportedEMEETFromUevent_HintByFamily table-drives the classifier
// level of the hint: known fixed-UVC models name the device, unknown
// EMEET-vendor PIDs invite the report, and controllable PIXY PIDs yield no
// hint at all. The registry itself is pinned by TestResolveProductID and the
// hint text by TestUnsupportedDeviceHint in internal/pixy.
func TestUnsupportedEMEETFromUevent_HintByFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		productID string
		contains  string
		empty     bool
	}{
		{
			name:      "known fixed webcam names the model",
			productID: "003f",
			contains:  "EMEET C960",
		},
		{
			name:      "unknown EMEET PID invites the report",
			productID: "0abc",
			contains:  "unknown EMEET device",
		},
		{
			name:      "PIXY PID yields no hint",
			productID: "00c0",
			empty:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uevent := "DEVTYPE=usb_interface\nPRODUCT=328f/" + tc.productID + "/2004\n"
			got := unsupportedEMEETFromUevent([]byte(uevent), "PRODUCT=", "/", 0, 1, "/dev/video0")

			if tc.empty {
				if got != "" {
					t.Errorf("expected no hint for a controllable PIXY, got %q", got)
				}

				return
			}

			if !strings.Contains(got, tc.contains) {
				t.Errorf("hint for PID %s = %q, want it to contain %q", tc.productID, got, tc.contains)
			}

			if !strings.Contains(got, "/dev/video0") {
				t.Errorf("hint lacks the device path interpolation: %q", got)
			}
		})
	}
}

// TestProbeVideo4linux_PixyWinsOverUnsupportedHint pins the probe
// precondition end-to-end at the sysfs-scan level: with a fixed-UVC EMEET
// webcam and a PIXY both attached, the scan returns the PIXY (any position)
// and carries no unsupported hint.
func TestProbeVideo4linux_PixyWinsOverUnsupportedHint(t *testing.T) {
	t.Parallel()

	writeVideoNode := func(t *testing.T, root, name, productID string) {
		t.Helper()

		node := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(node, "device"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(node, "index"), []byte("0"), 0o644); err != nil {
			t.Fatal(err)
		}

		uevent := "DEVTYPE=usb_interface\nPRODUCT=328f/" + productID + "/2004\n"
		if err := os.WriteFile(filepath.Join(node, "device", "uevent"), []byte(uevent), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name        string
		fixedUVCDev string
	}{
		{name: "pixy sorts first", fixedUVCDev: "video1"},
		{name: "pixy sorts last", fixedUVCDev: "video0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()

			pixyDev := "video0"
			if tc.fixedUVCDev == "video0" {
				pixyDev = "video1"
			}

			writeVideoNode(t, root, tc.fixedUVCDev, "003f") // EMEET C960
			writeVideoNode(t, root, pixyDev, "00c0")        // PIXY

			dev, model, unsupported := probeVideo4linux(root, nil)

			if dev != "/dev/"+pixyDev {
				t.Errorf("expected the PIXY at /dev/%s, got %q", pixyDev, dev)
			}

			if model != pixy.ModelOriginal {
				t.Errorf("expected model %q, got %q", pixy.ModelOriginal, model)
			}

			if unsupported != "" {
				t.Errorf("expected no unsupported hint alongside a found PIXY, got %q", unsupported)
			}
		})
	}
}
