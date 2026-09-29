//go:build linux

package main

import (
	"log/slog"

	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/sys/unix"
)

func unixOpenNetlinkKobjectUevent() (int, error) {
	fd, err := unix.Socket(
		unix.AF_NETLINK,
		unix.SOCK_RAW|unix.SOCK_NONBLOCK,
		unix.NETLINK_KOBJECT_UEVENT,
	)
	if err != nil {
		return -1, errorfamily.WrapTransientf(err, "uevent.netlink_socket", "netlink socket")
	}

	//nolint:exhaustruct
	sa := &unix.SockaddrNetlink{
		//nolint:exhaustruct
		Groups: 1,
	}

	bindErr := unix.Bind(fd, sa)
	if bindErr != nil {
		if closeErr := unix.Close(fd); closeErr != nil {
			slog.Debug("netlink fd close after bind failure failed", "err", closeErr)
		}

		return -1, errorfamily.WrapTransientf(bindErr, "uevent.netlink_bind", "netlink bind")
	}

	return fd, nil
}
