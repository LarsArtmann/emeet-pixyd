//go:build linux

package main

import (
	"context"
	"testing"
)

func TestHandleCommand_UnknownCommand(t *testing.T) {
	t.Parallel()
	assertCommandResponse(t, "foobar", "unknown command", "response")
}

func TestHandleCommand_StatusFormat(t *testing.T) {
	t.Parallel()

	d := newDaemonWithDevice(t)
	resp := d.handleCommand(context.Background(), "")
	assertCommandContainsAnyOf(t, resp.String(), []string{"camera=", "audio="}, "status response")
}

func TestHandleCommand_AudioUsage(t *testing.T) {
	t.Parallel()
	assertCommandResponse(t, "audio badmode", "error:", "response for bad audio mode")
}

func TestHandleCommand_PTZUsage(t *testing.T) {
	t.Parallel()
	assertCommandResponse(t, "pan", "usage:", "response for pan without value")
}

func TestHandleCommand_Device(t *testing.T) {
	t.Parallel()

	d := newDaemonWithDevice(t)
	resp := d.handleCommand(context.Background(), cmdDevice)
	assertCommandContains(t, resp.String(), "/dev/video", "response")
	assertCommandContains(t, resp.String(), "/dev/hidraw", "response")
}

func TestHandleCommand_DeviceNotFound(t *testing.T) {
	t.Parallel()

	d := newIntegrationDaemon(t)

	resp := d.handleCommand(context.Background(), cmdDevice)
	if resp.String() != respDeviceNotFound {
		t.Errorf("expected device not found, got: %s", resp)
	}
}
