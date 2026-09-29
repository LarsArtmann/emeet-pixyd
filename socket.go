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

func (d *Daemon) listenUnix(
	ctx context.Context,
) error { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	socketPath := d.config.SocketPath()
	if removeErr := os.Remove(socketPath); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		slog.Debug("failed to remove stale socket", "path", socketPath, "err", removeErr)
	}

	createErr := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if createErr != nil {
		return errorfamily.Wrapf(
			createErr,
			errorfamily.Classify(createErr),
			"socket.state_dir",
			"create state dir %s",
			d.config.StateDir,
		)
	}

	//nolint:exhaustruct
	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "unix", socketPath)
	if err != nil {
		return errorfamily.Wrapf(err, errorfamily.Classify(err), "socket.listen", "listen on %s", socketPath)
	}

	defer func() {
		closeErr := listener.Close()
		if closeErr != nil {
			slog.Debug("listener close error", "error", closeErr)
		}
	}()

	chmodErr := os.Chmod(socketPath, pixy.PermissionSocket)
	if chmodErr != nil {
		slog.Error("failed to set socket permissions", "error", chmodErr)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			slog.Error("socket accept error", "error", err)

			continue
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
