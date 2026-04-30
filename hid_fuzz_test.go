//go:build linux

package main

import (
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func FuzzParseHIDResponse(f *testing.F) {
	seed := [][]byte{
		nil,
		{},
		{0x00},
		make([]byte, 8),
		{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01},
		{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02},
		{0x09, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02},
		{0x09, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x03},
		{
			0x09,
			0x04,
			0x01,
			0x00,
			0x00,
			0x01,
			0x00,
			0x01,
			0xFF,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x00,
			0x01,
		},
		{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		make([]byte, 64),
		make([]byte, 1024),
	}

	for _, s := range seed {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		resp := parseHIDResponse(data)

		if len(data) < hidMinLen {
			if resp.Got {
				t.Error("Got should be false for short data")
			}
			assertTrackingIdle(t, resp.Tracking)
			if resp.Audio != pixy.AudioNC {
				t.Errorf("Audio = %q, want nc", resp.Audio)
			}

			return
		}

		if !resp.Got {
			t.Error("Got should be true for data >= hidMinLen")
		}

		if !resp.Tracking.Valid() {
			t.Errorf("invalid CameraState %q", resp.Tracking)
		}

		if !resp.Audio.Valid() {
			t.Errorf("invalid AudioMode %q", resp.Audio)
		}
	})
}
