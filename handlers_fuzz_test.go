//go:build linux

package main

import (
	"bufio"
	"bytes"
	"testing"
)

func FuzzExtractJPEGFrame(f *testing.F) {
	f.Add([]byte{0xFF, 0xD8, 0xFF, 0xD9})
	f.Add([]byte{0xFF, 0xD8, 0x00, 0xFF, 0xD9})
	f.Add([]byte{})
	f.Add([]byte{0xFF})
	f.Add([]byte{0xFF, 0xFF, 0xD8, 0xFF, 0xD9})
	f.Add([]byte{0x00, 0x00, 0xFF, 0xD8, 0xAA, 0xBB, 0xFF, 0xD9})
	f.Add(bytes.Repeat([]byte{0xFF, 0xD8, 0x42, 0xFF, 0xD9}, 100))
	f.Add(append([]byte{0xFF, 0xD8}, make([]byte, 1024*1024)...))

	f.Fuzz(func(t *testing.T, data []byte) {
		br := bufio.NewReader(bytes.NewReader(data))
		buf := &bytes.Buffer{}

		frame, err := extractJPEGFrame(br, buf)
		if err != nil {
			if frame != nil {
				t.Error("frame should be nil on error")
			}

			return
		}

		if len(frame) < 4 {
			t.Errorf("frame too short: %d bytes", len(frame))
		}

		if frame[0] != 0xFF || frame[1] != 0xD8 {
			t.Errorf("frame must start with JPEG SOI, got %02X %02X", frame[0], frame[1])
		}

		if frame[len(frame)-2] != 0xFF || frame[len(frame)-1] != 0xD9 {
			t.Errorf("frame must end with JPEG EOI, got %02X %02X", frame[len(frame)-2], frame[len(frame)-1])
		}
	})
}
