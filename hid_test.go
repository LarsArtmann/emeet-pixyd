//go:build linux

package main

import (
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

func TestAudioModeNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    pixy.AudioMode
		expected pixy.AudioMode
	}{
		{pixy.AudioNC, pixy.AudioLive},
		{pixy.AudioLive, pixy.AudioOriginal},
		{pixy.AudioOriginal, pixy.AudioNC},
		{pixy.AudioMode(testStrUnknown), pixy.AudioNC},
	}
	for _, testCase := range tests {
		result := testCase.input.Next()
		if result != testCase.expected {
			t.Errorf(
				"pixy.AudioMode(%s).Next() = %s, want %s",
				testCase.input,
				result,
				testCase.expected,
			)
		}
	}
}

func TestAudioModeHIDByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode     pixy.AudioMode
		expected byte
	}{
		{pixy.AudioNC, hidByteNC},
		{pixy.AudioLive, hidByteLive},
		{pixy.AudioOriginal, hidByteOriginal},
		{pixy.AudioMode(testStrUnknown), hidByteNC},
	}
	for _, testCase := range tests {
		result := audioHIDByte(testCase.mode)
		if result != testCase.expected {
			t.Errorf(
				"audioHIDByte(%s) = 0x%02x, want 0x%02x",
				testCase.mode,
				result,
				testCase.expected,
			)
		}
	}
}

func TestCameraStateHIDByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    pixy.CameraState
		expected byte
	}{
		{pixy.StateTracking, hidByteTracking},
		{pixy.StatePrivacy, hidBytePrivacy},
		{pixy.StateIdle, hidByteIdle},
		{pixy.StateOffline, hidByteIdle},
		{pixy.CameraState(testStrUnknown), hidByteIdle},
	}
	for _, testCase := range tests {
		result := cameraHIDByte(testCase.state)
		if result != testCase.expected {
			t.Errorf(
				"cameraHIDByte(%s) = 0x%02x, want 0x%02x",
				testCase.state,
				result,
				testCase.expected,
			)
		}
	}
}

func TestTypeValidation(t *testing.T) {
	t.Parallel()

	if !pixy.AudioNC.Valid() {
		t.Error("pixy.AudioNC should be valid")
	}

	if !pixy.StateTracking.Valid() {
		t.Error("pixy.StateTracking should be valid")
	}

	if pixy.AudioMode("foo").Valid() {
		t.Error("unknown audio mode should not be valid")
	}

	if pixy.CameraState("bar").Valid() {
		t.Error("unknown camera state should not be valid")
	}
}

func TestParseHIDResponseTracking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data     []byte
		expected pixy.CameraState
	}{
		{[]byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}, pixy.StateTracking},
		{[]byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02}, pixy.StatePrivacy},
		{[]byte{0x09, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00}, pixy.StateIdle},
	}
	for _, testCase := range tests {
		resp := parseHIDResponse(testCase.data)
		if !resp.Got {
			t.Fatal("expected Got=true")
		}

		if resp.Tracking != testCase.expected {
			t.Errorf(
				"tracking from %x = %s, want %s",
				testCase.data,
				resp.Tracking,
				testCase.expected,
			)
		}
	}
}

func TestParseHIDResponseAudio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data     []byte
		expected pixy.AudioMode
	}{
		{[]byte{0x09, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}, pixy.AudioNC},
		{[]byte{0x09, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02}, pixy.AudioLive},
		{[]byte{0x09, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x03}, pixy.AudioOriginal},
	}
	for _, testCase := range tests {
		resp := parseHIDResponse(testCase.data)
		if !resp.Got {
			t.Fatal("expected Got=true")
		}

		if resp.Audio != testCase.expected {
			t.Errorf("audio from %x = %s, want %s", testCase.data, resp.Audio, testCase.expected)
		}
	}
}

func TestParseHIDResponseGesture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte{0x09, 0x04, 0x02, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02, 0x01}, true},
		{[]byte{0x09, 0x04, 0x02, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02, 0x00}, false},
	}
	for _, testCase := range tests {
		assertGesture(t, parseHIDResponse(testCase.data), testCase.expected)
	}
}

func TestParseHIDResponseTooShort(t *testing.T) {
	t.Parallel()

	resp := parseHIDResponse([]byte{0x09, 0x01})
	if resp.Got {
		t.Error("expected Got=false for short response")
	}
}

func TestParseHIDResponseNil(t *testing.T) {
	t.Parallel()

	resp := parseHIDResponse(nil)
	if resp.Got {
		t.Error("expected Got=false for nil response")
	}
}

func TestParseHIDResponseUnknownInterface(t *testing.T) {
	t.Parallel()

	data := []byte{0x09, 0x99, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x01}

	resp := parseHIDResponse(data)
	if !resp.Got {
		t.Error("expected Got=true for valid-length response with unknown interface")
	}

	assertTrackingIdle(t, resp.Tracking)
}
