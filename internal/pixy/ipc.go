package pixy

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// SetDeadline sets a read/write deadline on the connection relative to now.
func SetDeadline(conn net.Conn, timeout time.Duration) error {
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return Wrapf(err, "ipc.deadline", "setDeadline (timeout=%v)", timeout)
	}

	return nil
}

// SendCommand sends a command string over a Unix socket and returns the response.
func SendCommand(ctx context.Context, socketPath, cmd string) (resp string, err error) {
	//nolint:exhaustruct
	dialer := net.Dialer{Timeout: DefaultSocketTimeout}

	conn, dialErr := dialer.DialContext(ctx, "unix", socketPath)
	if dialErr != nil {
		return "", Wrapf(dialErr, "ipc.dial", "sendCommand dial %s", socketPath)
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			if err != nil {
				err = errorfamily.Compose(err, Wrap(closeErr, "ipc.close", "sendCommand close "+socketPath))
			} else {
				slog.Debug("sendCommand close failed after successful command", "socket", socketPath, "err", closeErr)
			}
		}
	}()

	deadlineErr := SetDeadline(conn, DefaultWriteTimeout)
	if deadlineErr != nil {
		return "", Wrapf(deadlineErr, "ipc.deadline", "sendCommand %s deadline", socketPath)
	}

	_, writeErr := conn.Write([]byte(cmd))
	if writeErr != nil {
		return "", Wrapf(writeErr, "ipc.write", "sendCommand %s write", socketPath)
	}

	buf := make([]byte, ConnBufSize)

	n, readErr := conn.Read(buf)
	if readErr != nil {
		return "", Wrapf(readErr, "ipc.read", "sendCommand %s read", socketPath)
	}

	return strings.TrimSpace(string(buf[:n])), nil
}
