//go:build linux

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

var (
	errNoHIDResponse   = errors.New("no HID response")
	errUnrecognizedHID = errors.New("unrecognized HID response")
	errHIDWriteZero    = errors.New("wrote 0 bytes")
)

// HIDDevice abstracts HID communication for testability.
type HIDDevice interface {
	Send(report []byte) error
	SendRecv(ctx context.Context, report []byte) ([]byte, error)
}

type hidrawDevice struct {
	path string
}

const (
	hidByteTracking = 0x01
	hidBytePrivacy  = 0x02
	hidByteIdle     = 0x00
	hidByteNC       = 0x01
	hidByteLive     = 0x02
	hidByteOriginal = 0x03

	hidBufSize     = 32
	hidRespBufSize = 64
	hidMinLen      = 9
	hidDebugLen    = 16

	hidInterfaceTracking = 0x01
	hidInterfaceAudio    = 0x05
	hidInterfaceGesture  = 0x04

	hidResponseMs = 500

	cameraConfigPrefix byte = 0x09
	cameraConfigMarker byte = 0x01
	audioConfigMarker  byte = 0x00
	gestureConfigMark1 byte = 0x02
	gestureConfigMark2 byte = 0x01
	gestureConfigMark3 byte = 0x02
	gestureEnabledByte byte = 0x01

	hidCommandSleepMs = 200

	hidResponseTimeout = hidResponseMs * time.Millisecond
)

type hidResponse struct {
	Tracking pixy.CameraState
	Audio    pixy.AudioMode
	Gesture  bool
	Got      bool
}

//nolint:gochecknoglobals
var cameraHIDBytes = map[pixy.CameraState]byte{
	pixy.StateTracking: hidByteTracking,
	pixy.StatePrivacy:  hidBytePrivacy,
	pixy.StateIdle:     hidByteIdle,
	pixy.StateOffline:  hidByteIdle,
}

func cameraHIDByte(s pixy.CameraState) byte {
	if b, ok := cameraHIDBytes[s]; ok {
		return b
	}

	return hidByteIdle
}

//nolint:gochecknoglobals
var audioHIDBytes = map[pixy.AudioMode]byte{
	pixy.AudioNC:       hidByteNC,
	pixy.AudioLive:     hidByteLive,
	pixy.AudioOriginal: hidByteOriginal,
}

func audioHIDByte(m pixy.AudioMode) byte {
	if b, ok := audioHIDBytes[m]; ok {
		return b
	}

	return hidByteNC
}

func newHIDRawDevice(path string) HIDDevice {
	return &hidrawDevice{path: path}
}

func (h *hidrawDevice) Send(report []byte) (err error) {
	if h.path == "" {
		return fmt.Errorf("hidSend (device not set): %w", pixy.ErrHIDDeviceNotAvailable)
	}

	buf := make([]byte, hidBufSize)
	copy(buf, report)

	hidFile, err := os.OpenFile(h.path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("hidSend open %s: %w", h.path, err)
	}

	defer func() {
		cerr := hidFile.Close()
		if cerr != nil && err == nil {
			err = fmt.Errorf("hidSend close: %w", cerr)
		}
	}()

	_, err = hidFile.Write(buf)
	if err != nil {
		return fmt.Errorf("hidSend write %s: %w", h.path, err)
	}

	return nil
}

func (h *hidrawDevice) SendRecv(ctx context.Context, report []byte) ([]byte, error) {
	if h.path == "" {
		return nil, fmt.Errorf("hidSendRecv (device not set): %w", pixy.ErrHIDDeviceNotAvailable)
	}

	buf := make([]byte, hidBufSize)
	copy(buf, report)

	hidFile, err := os.OpenFile(h.path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open hidraw %s: %w", h.path, err)
	}

	defer func() { _ = hidFile.Close() }()

	written, writeErr := hidFile.Write(buf)
	if writeErr != nil {
		return nil, fmt.Errorf("write hidraw %s: %w", h.path, writeErr)
	}

	if written == 0 {
		return nil, fmt.Errorf("write hidraw %s: %w", h.path, errHIDWriteZero)
	}

	type readResult struct {
		data []byte
		err  error
	}

	resultChan := make(chan readResult, 1)

	go func() {
		resp := make([]byte, hidRespBufSize)

		n, readErr := hidFile.Read(resp)
		resultChan <- readResult{resp[:n], readErr}
	}()

	timeout := time.NewTimer(hidResponseTimeout)
	defer timeout.Stop()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("hidSendRecv %s: %w", h.path, ctx.Err())
	case r := <-resultChan:
		if r.err != nil {
			return nil, fmt.Errorf("hidSendRecv %s read: %w", h.path, r.err)
		}

		return r.data, nil
	case <-timeout.C:
		return nil, nil
	}
}

func newHIDResponse(got bool) hidResponse {
	return hidResponse{
		Tracking: pixy.StateIdle,
		Audio:    pixy.AudioNC,
		Gesture:  false,
		Got:      got,
	}
}

func parseHIDResponse(data []byte) hidResponse {
	// HID response layout (9+ bytes):
	//   data[0] = report prefix (0x09 = camera config)
	//   data[1] = interface (0x01=tracking, 0x04=gesture, 0x05=audio)
	//   data[2..7] = markers/padding
	//   data[8] = mode byte (tracking/audio/gesture state)
	if len(data) < hidMinLen {
		return newHIDResponse(false)
	}

	resp := newHIDResponse(true)

	slog.Debug("HID response", "hex", hex.EncodeToString(data[:min(len(data), hidDebugLen)]))

	switch {
	case data[0] == cameraConfigPrefix && data[1] == hidInterfaceTracking:
		switch data[8] {
		case hidByteTracking:
			resp.Tracking = pixy.StateTracking
		case hidBytePrivacy:
			resp.Tracking = pixy.StatePrivacy
		default:
			resp.Tracking = pixy.StateIdle
		}
	case data[0] == cameraConfigPrefix && data[1] == hidInterfaceAudio:
		switch data[8] {
		case hidByteLive:
			resp.Audio = pixy.AudioLive
		case hidByteOriginal:
			resp.Audio = pixy.AudioOriginal
		default:
			resp.Audio = pixy.AudioNC
		}
	case data[0] == cameraConfigPrefix && data[1] == hidInterfaceGesture:
		resp.Gesture = data[len(data)-1] == gestureEnabledByte
	}

	return resp
}

func pixyConfig(iface, modeByte byte) []byte {
	var buf [hidMinLen]byte

	buf[0] = cameraConfigPrefix
	buf[1] = iface
	buf[2] = cameraConfigMarker
	buf[3] = 0x00
	buf[4] = 0x00
	buf[5] = cameraConfigMarker
	buf[6] = 0x00
	buf[7] = cameraConfigMarker
	buf[8] = modeByte

	return buf[:]
}

func pixyCommit(iface byte) []byte {
	return []byte{0x09, iface, 0x01, iface}
}

func queryHIDState[T any](
	ctx context.Context,
	dev HIDDevice,
	payload []byte,
	extract func(hidResponse) T,
) (T, error) {
	var zero T

	resp, err := dev.SendRecv(ctx, payload)
	if err != nil {
		return zero, fmt.Errorf("queryHIDState: %w", err)
	}

	if resp == nil {
		return zero, fmt.Errorf("queryHIDState: %w", errNoHIDResponse)
	}

	parsed := parseHIDResponse(resp)
	if !parsed.Got {
		return zero, fmt.Errorf("queryHIDState: %w", errUnrecognizedHID)
	}

	return extract(parsed), nil
}
