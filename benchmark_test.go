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
