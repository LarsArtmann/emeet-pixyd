//go:build linux

package main

import (
	"context"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func BenchmarkParseHIDResponse(b *testing.B) {
	data := []byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}

	b.ResetTimer()

	for b.Loop() {
		parseHIDResponse(data)
	}
}

func BenchmarkWaybarOutput(b *testing.B) {
	d := testDaemonWithState(b, pixy.StateTracking, true)

	b.ResetTimer()

	for b.Loop() {
		d.waybarOutput()
	}
}

func BenchmarkHandleCommand_Query(b *testing.B) {
	d := testDaemonWithDevice(b, pixy.StateTracking)

	b.ResetTimer()

	for b.Loop() {
		d.handleCommand(context.Background(), cmdWaybar)
	}
}

func BenchmarkHandleCommand_Mutating(b *testing.B) {
	d := testDaemonWithDevice(b, pixy.StatePrivacy)
	d.config = defaultTestConfig(b.TempDir())
	b.ResetTimer()

	for b.Loop() {
		d.handleCommand(context.Background(), cmdToggleAuto)
	}
}

func BenchmarkGetWebStatus(b *testing.B) {
	d := testDaemonWithDevice(b, pixy.StateTracking)
	srv := &webServer{daemon: d}

	b.ResetTimer()

	for b.Loop() {
		srv.getWebStatus()
	}
}

// BenchmarkSimulatorRoundTrip measures the overhead of a full simulator
// round-trip (config → commit → query) for all three interfaces. This
// establishes a baseline for comparing future Layer 2 (/dev/uhid) and
// Layer 3 (NixOS VM) test harnesses against the in-process simulator.
func BenchmarkSimulatorRoundTrip(b *testing.B) {
	sim := newPixySimulator()
	ctx := context.Background()

	trackingQuery := []byte{cameraConfigPrefix, hidInterfaceTracking, cameraConfigMarker, hidByteTracking}
	audioQuery := []byte{cameraConfigPrefix, hidInterfaceAudio, audioConfigMarker, hidInterfaceAudio}
	gestureQuery := []byte{cameraConfigPrefix, hidInterfaceGesture, gestureConfigMark1, gestureConfigMark2}

	b.ResetTimer()

	for b.Loop() {
		_ = sim.Send(pixyConfig(hidInterfaceTracking, hidByteTracking))
		_ = sim.Send(pixyCommit(hidInterfaceTracking))
		_, _ = sim.SendRecv(ctx, trackingQuery)

		_ = sim.Send(pixyConfig(hidInterfaceAudio, hidByteLive))
		_ = sim.Send(pixyCommit(hidInterfaceAudio))
		_, _ = sim.SendRecv(ctx, audioQuery)

		_ = sim.Send(pixyConfig(hidInterfaceGesture, gestureEnabledByte))
		_ = sim.Send(pixyCommit(hidInterfaceGesture))
		_, _ = sim.SendRecv(ctx, gestureQuery)
	}
}
