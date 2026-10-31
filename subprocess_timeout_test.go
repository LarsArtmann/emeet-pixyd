//go:build linux

package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

// hangCommander blocks every Run/Output call until its context is done and
// then returns the context error — the faithful model of a wedged subprocess
// (a v4l2-ctl that never answers, wpctl stuck on a dead PipeWire socket).
type hangCommander struct{}

func (hangCommander) Run(ctx context.Context, _ string, _ ...string) error {
	<-ctx.Done()

	return ctx.Err()
}

func (hangCommander) Output(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()

	return nil, ctx.Err()
}

func (hangCommander) LookPath(string) (string, error) {
	return "/bin/true", nil
}

// TestV4L2Set_HungCommanderIsBounded pins the #190 bound end to end at its
// real value: with a caller context that never expires, the wrapper's own
// deadline must fire, the error must classify as Transient, and v4l2Mu must
// be acquirable again afterwards (the move paths hold it across v4l2Set).
func TestV4L2Set_HungCommanderIsBounded(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	d.deps.commander = hangCommander{}

	start := time.Now()

	err := d.v4l2Set(context.Background(), "/dev/video0", "pan_absolute", "1000")
	if err == nil {
		t.Fatal("expected error from hung commander")
	}

	if errorfamily.Classify(err) != errorfamily.Transient {
		t.Errorf("expected Transient classification for a deadline error, got %v", errorfamily.Classify(err))
	}

	if elapsed := time.Since(start); elapsed > v4l2CommandTimeout+2*time.Second {
		t.Errorf("v4l2Set outlived its deadline bound: %v", elapsed)
	}

	if !d.v4l2Mu.TryLock() {
		t.Fatal("v4l2Mu still held after bounded v4l2Set returned")
	}

	d.v4l2Mu.Unlock()
}

// TestParsePTZValues_HungCommanderIsBounded proves deadline propagation with
// a caller deadline shorter than the wrapper bound (the bound itself is
// pinned by TestV4L2Set_HungCommanderIsBounded).
func TestParsePTZValues_HungCommanderIsBounded(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	d.deps.commander = hangCommander{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()

	vals := d.parsePTZValues(ctx, "/dev/video0")

	if vals != (pixy.PTZValues{}) {
		t.Errorf("expected zero PTZValues on read failure, got %+v", vals)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("parsePTZValues outlived the caller deadline: %v", elapsed)
	}
}

// TestAudioAndNotifyPaths_HungCommanderIsBounded covers the remaining exec
// surfaces: findPixySource (error path), setDefaultSource and notifyCmd
// (log-only paths). All must return within the caller deadline, not block
// the auto-manage loop until the systemd watchdog kills the daemon.
func TestAudioAndNotifyPaths_HungCommanderIsBounded(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	d.deps.commander = hangCommander{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()

	_, err := d.findPixySource(ctx)
	if errorfamily.Classify(err) != errorfamily.Transient {
		t.Errorf("expected Transient classification for a deadline error, got %v", errorfamily.Classify(err))
	}

	d.setDefaultSource(ctx, pixy.NewSourceID("42"))
	d.notifyCmd(ctx, "title", "body")

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("audio/notify paths outlived the caller deadline: %v", elapsed)
	}
}

// writeFakeFFmpeg installs an executable ffmpeg script into dir.
func writeFakeFFmpeg(t *testing.T, dir, body string) {
	t.Helper()

	path := filepath.Join(dir, ffmpegBin)

	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
}

// TestStream_SilentFFmpegBoundedAndHealthyStarts pins the #190 stream bound:
// an ffmpeg that starts but never writes a byte must fail setup within the
// first-frame timeout (instead of blocking writeFrames and holding the
// stream semaphore forever), while a fast first frame must pass setup.
func TestStream_SilentFFmpegBoundedAndHealthyStarts(t *testing.T) {
	dir := t.TempDir()
	writeFakeFFmpeg(t, dir, "#!/bin/sh\nexec sleep 60\n")

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	d := newTestDaemon(t, pixy.StateTracking, "/dev/video0", "/dev/hidraw7")
	webSrv := &webServer{daemon: d}

	silent := httptest.NewRecorder()
	start := time.Now()

	result := webSrv.setupStream(silent, context.Background())

	if result.ok {
		t.Fatal("expected setup to fail for an ffmpeg that never writes")
	}

	if silent.Code != errorfamily.HTTPStatus(errStreamSilent) {
		t.Errorf("expected %d for silent ffmpeg, got %d (body %q)",
			errorfamily.HTTPStatus(errStreamSilent), silent.Code, silent.Body.String())
	}

	if elapsed := time.Since(start); elapsed > ffmpegFirstFrameTimeout+3*time.Second {
		t.Errorf("setup outlived the first-frame bound: %v", elapsed)
	}

	writeFakeFFmpeg(t, dir, "#!/bin/sh\nprintf '\\377\\330\\101\\102\\103\\377\\331'\nexec sleep 60\n")

	healthy := httptest.NewRecorder()

	okResult := webSrv.setupStream(healthy, context.Background())
	if !okResult.ok {
		t.Fatalf("expected setup to succeed for a fast first frame, got code %d body %q",
			healthy.Code, healthy.Body.String())
	}

	cleanupFFmpeg(okResult.cmd)
}
