//go:build linux

package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

const socketIOTimeout = 5 * time.Second

// newUnixListener binds the CLI socket and applies its permissions. Startup
// bind failures return an error so Run can fail fast (#191a) — running
// READY=1 with a dead command surface hides the failure from every health
// layer until a human notices.
func (d *Daemon) newUnixListener() (net.Listener, error) {
	socketPath := d.config.SocketPath()
	if removeErr := os.Remove(socketPath); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		slog.Debug("failed to remove stale socket", "path", socketPath, "err", removeErr)
	}

	createErr := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if createErr != nil {
		return nil, errorfamily.Wrapf(
			createErr,
			errorfamily.Classify(createErr),
			"socket.state_dir",
			"create state dir %s",
			d.config.StateDir,
		)
	}

	//x
	lc := net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "unix", socketPath)
	if err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Classify(err), "socket.listen", "listen on %s", socketPath)
	}

	if chmodErr := os.Chmod(socketPath, pixy.PermissionSocket); chmodErr != nil {
		slog.Error("failed to set socket permissions", "error", chmodErr)
	}

	return listener, nil
}

// acceptFailureCeiling is the number of consecutive accept errors that
// marks the daemon degraded (socket-accept). The loop itself never exits —
// a degraded daemon that still serves HTTP can be diagnosed and its state
// read; killing the loop would take away the CLI surface entirely.
const acceptFailureCeiling = 10

// acceptRetryDelay spaces out accept retries so a persistently failing
// listener produces ~10 attempts/second instead of a busy loop (and bounds
// the ceiling episode to roughly one second).
const acceptRetryDelay = 100 * time.Millisecond

// acceptUnixLoop serves CLI connections until the context is cancelled.
// Transient accept errors are logged and retried — the loop never exits on
// its own; hitting the ceiling marks the daemon degraded instead (#191b)
// and one healthy accept clears the mark again.
func (d *Daemon) acceptUnixLoop(ctx context.Context, listener net.Listener) {
	defer func() {
		closeErr := listener.Close()
		if closeErr != nil {
			slog.Debug("listener close error", "error", closeErr)
		}
	}()

	consecutiveFailures := 0

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}

			consecutiveFailures++

			recordSocketAcceptFailure(ctx)

			if consecutiveFailures >= acceptFailureCeiling {
				d.markDegraded(degradedSocketAccept)
				slog.Error("socket accept consecutive failures hit ceiling", "consecutive", consecutiveFailures)

				consecutiveFailures = 0
			} else {
				slog.Error("socket accept error", "error", err, "consecutive", consecutiveFailures)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(acceptRetryDelay):
			}

			continue
		}

		if consecutiveFailures > 0 {
			consecutiveFailures = 0

			d.clearDegraded(degradedSocketAccept)
		}

		d.serveUnixConn(ctx, conn)
	}
}

// serveUnixConn handles a single socket command connection sequentially:
// read one command, respond once, close.
func (d *Daemon) serveUnixConn(
	ctx context.Context,
	conn net.Conn,
) { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Debug("conn close error", "error", closeErr)
		}
	}()

	if deadlineErr := conn.SetReadDeadline(time.Now().Add(socketIOTimeout)); deadlineErr != nil {
		slog.Debug("socket read deadline not set", "err", deadlineErr)
	}

	buf := make([]byte, pixy.SocketBufSize)

	n, readErr := conn.Read(buf)
	if readErr != nil || n == 0 {
		return
	}

	cmd := strings.TrimSpace(string(buf[:n]))

	response := d.handleCommand(ctx, cmd).String() + "\n"

	if deadlineErr := conn.SetWriteDeadline(time.Now().Add(socketIOTimeout)); deadlineErr != nil {
		slog.Debug("socket write deadline not set", "err", deadlineErr)
	}

	if _, writeErr := conn.Write([]byte(response)); writeErr != nil {
		slog.Debug("socket write error", "error", writeErr)
	}
}

func sendCommand(
	cfg pixy.Config,
	cmd string,
) (string, error) { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	resp, err := pixy.SendCommand(context.Background(), cfg.SocketPath(), cmd)
	if err != nil {
		return "", errorfamily.Wrapf(err, errorfamily.Classify(err), "socket.send", "sendCommand %q", cmd)
	}

	return resp, nil
}
