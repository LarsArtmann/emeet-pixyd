//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const socketIOTimeout = 5 * time.Second

func (d *Daemon) listenUnix(ctx context.Context) error {
	socketPath := d.config.SocketPath()
	_ = os.Remove(socketPath)

	createErr := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if createErr != nil {
		return fmt.Errorf("create state dir: %w", createErr)
	}

	//nolint:exhaustruct
	lc := net.ListenConfig{}

	listener, err := lc.Listen(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
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

		buf := make([]byte, pixy.SocketBufSize)

		_ = conn.SetReadDeadline(time.Now().Add(socketIOTimeout))
		n, readErr := conn.Read(buf)
		if readErr == nil && n > 0 {
			cmd := strings.TrimSpace(string(buf[:n]))

			response := d.handleCommand(ctx, cmd).String() + "\n"

			_ = conn.SetWriteDeadline(time.Now().Add(socketIOTimeout))
			_, writeErr := conn.Write([]byte(response))
			if writeErr != nil {
				slog.Debug("socket write error", "error", writeErr)
			}
		}

		closeErr := conn.Close()
		if closeErr != nil {
			slog.Debug("conn close error", "error", closeErr)
		}
	}
}

func sendCommand(cfg pixy.Config, cmd string) (string, error) {
	resp, err := pixy.SendCommand(context.Background(), cfg.SocketPath(), cmd)
	if err != nil {
		return "", fmt.Errorf("sendCommand %q: %w", cmd, err)
	}

	return resp, nil
}
