package pixy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// SetDeadline sets a read/write deadline on the connection relative to now.
func SetDeadline(conn net.Conn, timeout time.Duration) error {
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return fmt.Errorf("setDeadline (timeout=%v): %w", timeout, err)
	}

	return nil
}

// SendCommand sends a command string over a Unix socket and returns the response.
func SendCommand(ctx context.Context, socketPath, cmd string) (string, error) {
	//nolint:exhaustruct
	dialer := net.Dialer{Timeout: DefaultSocketTimeout}

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("sendCommand dial %s: %w", socketPath, err)
	}

	defer func() { _ = conn.Close() }()

	deadlineErr := SetDeadline(conn, DefaultWriteTimeout)
	if deadlineErr != nil {
		return "", fmt.Errorf("sendCommand %s deadline: %w", socketPath, deadlineErr)
	}

	_, writeErr := conn.Write([]byte(cmd))
	if writeErr != nil {
		return "", fmt.Errorf("sendCommand %s write: %w", socketPath, writeErr)
	}

	buf := make([]byte, ConnBufSize)

	n, readErr := conn.Read(buf)
	if readErr != nil {
		return "", fmt.Errorf("sendCommand %s read: %w", socketPath, readErr)
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
