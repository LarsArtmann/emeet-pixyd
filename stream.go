//go:build linux

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"syscall"
	"time"
)

const (
	ffmpegBin             = "ffmpeg"
	ffmpegShutdownTimeout = 2 * time.Second
	streamBufSize         = 64 * 1024
)

var errJPEGMaxIterations = errors.New("max iterations reached scanning for JPEG frame")

const (
	jpegMarker = 0xFF
	jpegSOI    = 0xD8
	jpegEOI    = 0xD9
)

func (s *webServer) handleSnapshot(responseWriter http.ResponseWriter, _ *http.Request) {
	frame := s.daemon.lastFrame.Get()
	if len(frame) == 0 {
		http.Error(responseWriter, "no frame available", http.StatusServiceUnavailable)

		return
	}

	responseWriter.Header().Set("Content-Type", "image/jpeg")
	responseWriter.Header().Set("Cache-Control", "no-store")
	_, _ = responseWriter.Write(frame)
}

func ffmpegStreamCmd(ctx context.Context, device string) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		ffmpegBin,
		"-f", "v4l2",
		"-input_format", "mjpeg",
		"-i", device,
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", "5",
		"-vf", "scale=640:-1",
		"pipe:1",
	)
}

func cleanupFFmpeg(cmd *exec.Cmd) {
	if cmd.Process == nil {
		_ = cmd.Wait()

		return
	}

	_ = cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(ffmpegShutdownTimeout):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func (s *webServer) handleStream(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	ctx := request.Context()

	select {
	case s.daemon.streamSema <- struct{}{}:
	default:
		http.Error(responseWriter, "stream already in use", http.StatusServiceUnavailable)

		return
	}

	defer func() { <-s.daemon.streamSema }()

	result := s.setupStream(responseWriter, ctx)
	if !result.ok {
		return
	}

	defer cleanupFFmpeg(result.cmd)

	streamStart := time.Now()

	defer func() {
		recordStreamDuration(ctx, time.Since(streamStart).Seconds())
	}()

	s.writeFrames(responseWriter, result.reader, result.flusher, ctx)
}

type streamResult struct {
	reader  *bufio.Reader
	cmd     *exec.Cmd
	flusher http.Flusher
	ok      bool
}

//nolint:exhaustruct
func (s *webServer) setupStream(
	responseWriter http.ResponseWriter,
	ctx context.Context,
) streamResult {
	status, ok := s.checkDevice(responseWriter)
	if !ok {
		return streamResult{}
	}

	_, lookErr := exec.LookPath(ffmpegBin)
	if lookErr != nil {
		http.Error(responseWriter, "ffmpeg not available", http.StatusServiceUnavailable)

		return streamResult{}
	}

	flusher, flushOk := responseWriter.(http.Flusher)
	if !flushOk {
		http.Error(responseWriter, "streaming not supported", http.StatusInternalServerError)

		return streamResult{}
	}

	cmd := ffmpegStreamCmd(ctx, status.Device)

	stdOut, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		http.Error(responseWriter, "stream pipe error", http.StatusInternalServerError)

		return streamResult{}
	}

	startErr := cmd.Start()
	if startErr != nil {
		http.Error(responseWriter, "stream start error", http.StatusInternalServerError)

		return streamResult{}
	}

	responseWriter.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	responseWriter.Header().Set("Cache-Control", "no-store")

	return streamResult{
		reader:  bufio.NewReaderSize(stdOut, streamBufSize),
		cmd:     cmd,
		flusher: flusher,
		ok:      true,
	}
}

func (s *webServer) writeFrames(
	responseWriter io.Writer,
	br *bufio.Reader,
	flusher http.Flusher,
	ctx context.Context,
) {
	var buf bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		frame, frameErr := extractJPEGFrame(br, &buf)
		if frameErr != nil {
			slog.Debug("frame extract error", "error", frameErr)

			return
		}

		s.daemon.lastFrame.Set(frame)
		recordFrame(ctx)

		_, headerErr := fmt.Fprintf(
			responseWriter,
			"--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n",
			len(frame),
		)
		if headerErr != nil {
			return
		}

		_, writeErr := responseWriter.Write(frame)
		if writeErr != nil {
			return
		}

		_, sepErr := fmt.Fprint(responseWriter, "\r\n")
		if sepErr != nil {
			return
		}

		flusher.Flush()
	}
}

func extractJPEGFrame(br *bufio.Reader, buf *bytes.Buffer) ([]byte, error) {
	const maxIterations = 10 * 1024 * 1024

	var soiFound bool

	for range maxIterations {
		if buf.Len() > maxStreamBufferSize {
			buf.Reset()
		}

		b, err := br.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("read byte: %w", err)
		}

		if !soiFound {
			if b == jpegMarker {
				next, nextErr := br.ReadByte()
				if nextErr != nil {
					return nil, fmt.Errorf("read soi next: %w", nextErr)
				}

				switch next {
				case jpegSOI:
					buf.Reset()
					buf.Write([]byte{jpegMarker, jpegSOI})

					soiFound = true
				case jpegMarker:
					_ = br.UnreadByte()
				}
			}

			continue
		}

		buf.WriteByte(b)

		if b == jpegMarker {
			next, nextErr := br.ReadByte()
			if nextErr != nil {
				return nil, fmt.Errorf("read eoi next: %w", nextErr)
			}

			buf.WriteByte(next)

			if next == jpegEOI {
				frame := make([]byte, buf.Len())
				copy(frame, buf.Bytes())

				return frame, nil
			}
		}
	}

	return nil, fmt.Errorf(
		"max iterations (%d) reached scanning for JPEG frame: %w",
		maxIterations,
		errJPEGMaxIterations,
	)
}
