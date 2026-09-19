//go:build linux

package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// Tests for the identity read surface (TODO #151): `device` output gains
// sn/ver/devver/func fields when the wired PIXY answers the official GET
// heads, and stays byte-identical when it does not.

func TestDeviceCommand_IdentityWithSimulator(t *testing.T) {
	t.Parallel()

	_, opt := withPixySimulator()
	d := newTestDaemon(t, pixy.StateTracking, testVideoDev, testHIDDev, opt)
	d.videoDev = testVideoDev
	d.hidrawDev = testHIDDev
	d.model = pixy.ModelOriginal

	result := d.handleCommand(t.Context(), "device")
	if result.IsError() {
		t.Fatalf("device failed: %s", result.String())
	}

	for _, want := range []string{
		"sn=PIXY-SIM-0001",
		"ver=0x0203",
		"devver=0x0203",
		"func=0x00000000",
	} {
		if !containsToken(result.String(), want) {
			t.Errorf("device output %q missing %q", result.String(), want)
		}
	}
}

func TestDeviceCommand_IdentityOmittedWithoutDevice(t *testing.T) {
	t.Parallel()

	d := testDaemonNoDevice(t)

	result := d.handleCommand(t.Context(), "device")
	if result.String() != respDeviceNotFound {
		t.Errorf("device = %q, want %q", result.String(), respDeviceNotFound)
	}
}

func TestFormatIdentity_OmitsUnanswered(t *testing.T) {
	t.Parallel()

	info := identityInfo{Serial: "SN1", Version: 0x0203}

	tokens := formatIdentity(info, []bool{true, true, false, false})

	if len(tokens) != 2 || tokens[0] != "sn=SN1" || tokens[1] != "ver=0x0203" {
		t.Errorf("tokens = %v, want [sn=SN1 ver=0x0203]", tokens)
	}
}

func TestParseString_NonPrintableTerminates(t *testing.T) {
	t.Parallel()

	resp := v2Response(pixy.V2GetSN, 'A', 'B', 0x00, 'C')

	got, err := pixy.ParseString(pixy.V2GetSN, resp)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got != "AB" {
		t.Errorf("ParseString = %q, want AB", got)
	}

	garbage := v2Response(pixy.V2GetSN, 0x01, 0x02)

	if _, err := pixy.ParseString(pixy.V2GetSN, garbage); err == nil {
		t.Error("non-printable-only payload accepted")
	}
}

func containsToken(s, token string) bool {
	return slices.Contains(strings.Fields(s), token)
}
