//go:build linux

package main

import (
	"context"
	"log/slog"
	"os/exec"
	"time"
)

// Subprocess timeout policy (#190): every external binary the daemon execs is
// bounded, so a wedged v4l2-ctl/wpctl/notify-send/ffmpeg can never hold
// v4l2Mu, hidMu, or the stream semaphore until the systemd watchdog kills the
// daemon. HID I/O carries its own 500ms deadline (hid.go). When a deadline
// fires the commander kills the subprocess and the resulting
// context.DeadlineExceeded classifies as Transient (stdlib registry entry).
const (
	v4l2CommandTimeout      = 5 * time.Second
	wpctlCommandTimeout     = 5 * time.Second
	notifyCommandTimeout    = 5 * time.Second
	ffmpegFirstFrameTimeout = 5 * time.Second
)

// CommandRunner abstracts subprocess execution for testability and
// centralized logging/metrics.
//
// ffmpeg streaming is intentionally excluded: it needs StdoutPipe +
// Start + long-lived process management, which doesn't fit the
// Run/Output/LookPath pattern. See stream.go's ffmpegStreamCmd.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	LookPath(name string) (string, error)
}

// realCommandRunner wraps exec.CommandContext with structured logging.
type realCommandRunner struct{}

func (realCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	start := time.Now()

	err := exec.CommandContext(ctx, name, args...).Run()

	logSubprocess(name, args, time.Since(start), err)

	return err //x
}

func (realCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	start := time.Now()

	out, err := exec.CommandContext(ctx, name, args...).Output()

	logSubprocess(name, args, time.Since(start), err)

	return out, err //x
}

func (realCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name) //x
}

func logSubprocess(name string, args []string, duration time.Duration, err error) {
	if err != nil {
		slog.Warn("subprocess failed", "cmd", name, "args", args, "duration", duration, "error", err)
	} else {
		slog.Debug("subprocess completed", "cmd", name, "args", args, "duration", duration)
	}
}

// noopCommandRunner is a no-op CommandRunner for tests.
type noopCommandRunner struct{}

func (noopCommandRunner) Run(context.Context, string, ...string) error { return nil }

func (noopCommandRunner) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, nil
}

func (noopCommandRunner) LookPath(string) (string, error) {
	return "", nil
}
