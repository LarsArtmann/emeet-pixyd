//go:build linux

package main

import (
	"bufio"
	"bytes"
	"testing"
)

func assertJPEGBytes(t *testing.T, frame, expected []byte) {
	t.Helper()

	if string(frame) != string(expected) {
		t.Errorf("expected %x, got %x", expected, frame)
	}
}

func TestExtractJPEGFrame_MinimalFrame(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(frame) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(frame))
	}

	assertJPEGMarkers(t, frame)
}

func TestExtractJPEGFrame_FrameWithPayload(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0x42, 0x43, 0x44, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertJPEGBytes(t, frame, data)
}

func TestExtractJPEGFrame_GarbageBeforeSOI(t *testing.T) {
	t.Parallel()

	data := []byte{0x00, 0x01, 0x02, 0xFF, 0xD8, 0xAA, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte{0xFF, 0xD8, 0xAA, 0xFF, 0xD9}
	assertJPEGBytes(t, frame, expected)
}

func TestExtractJPEGFrame_DoubleFFBeforeD8(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xFF, 0xD8, 0x42, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte{0xFF, 0xD8, 0x42, 0xFF, 0xD9}
	assertJPEGBytes(t, frame, expected)
}

func TestExtractJPEGFrame_EmptyInput(t *testing.T) {
	t.Parallel()

	br := bufio.NewReader(bytes.NewReader(nil))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestExtractJPEGFrame_NoEOI(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0x42, 0x43}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error when no EOI found")
	}
}

func TestExtractJPEGFrame_NoSOI(t *testing.T) {
	t.Parallel()

	data := []byte{0x42, 0x43, 0x44}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error when no SOI found")
	}
}

func TestExtractJPEGFrame_FFInsidePayload(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0xFF, 0x00, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertJPEGMarkers(t, frame)
}

func TestExtractJPEGFrame_BufferReset(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := bytes.NewBuffer(make([]byte, maxStreamBufferSize+100))

	frame, err := extractJPEGFrame(br, buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertJPEGBytes(t, frame, data)
}

func TestExtractJPEGFrame_FFThenEOF(t *testing.T) {
	t.Parallel()

	data := []byte{0xFF}
	br := bufio.NewReader(bytes.NewReader(data))
	buf := &bytes.Buffer{}

	_, err := extractJPEGFrame(br, buf)
	if err == nil {
		t.Fatal("expected error for truncated input")
	}
}

func assertJPEGMarkers(t *testing.T, frame []byte) {
	t.Helper()

	if frame[0] != 0xFF || frame[1] != 0xD8 {
		t.Errorf("missing SOI")
	}

	if frame[len(frame)-2] != 0xFF || frame[len(frame)-1] != 0xD9 {
		t.Errorf("missing EOI")
	}
}

func BenchmarkExtractJPEGFrame(b *testing.B) {
	data := make([]byte, 0, 104)

	data = append(data, 0xFF, 0xD8)
	for range 100 {
		data = append(data, 0x42)
	}

	data = append(data, 0xFF, 0xD9)

	b.ResetTimer()

	for b.Loop() {
		br := bufio.NewReader(bytes.NewReader(data))
		buf := &bytes.Buffer{}
		_, _ = extractJPEGFrame(br, buf)
	}
}
