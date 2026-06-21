//go:build linux

package main

import (
	"context"
	"log/slog"
	"os/exec"
	"time"
)

// CommandRunner abstracts subprocess execution for testability and
// centralized logging/metrics.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// realCommandRunner wraps exec.CommandContext with structured logging.
type realCommandRunner struct{}

func (realCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	start := time.Now()

	err := exec.CommandContext(ctx, name, args...).Run()

	logSubprocess(name, args, time.Since(start), err)

	return err //nolint:wrapcheck // caller wraps with context
}

func (realCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	start := time.Now()

	out, err := exec.CommandContext(ctx, name, args...).Output()

	logSubprocess(name, args, time.Since(start), err)

	return out, err //nolint:wrapcheck // caller wraps with context
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
