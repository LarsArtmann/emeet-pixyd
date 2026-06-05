//go:build linux

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func unixOpenNetlinkKobjectUevent() (int, error) {
	fd, err := unix.Socket(
		unix.AF_NETLINK,
		unix.SOCK_RAW|unix.SOCK_NONBLOCK,
		unix.NETLINK_KOBJECT_UEVENT,
	)
	if err != nil {
		return -1, fmt.Errorf("netlink socket: %w", err)
	}

	//nolint:exhaustruct
	sa := &unix.SockaddrNetlink{
		//nolint:exhaustruct
		Groups: 1,
	}

	bindErr := unix.Bind(fd, sa)
	if bindErr != nil {
		_ = unix.Close(fd)

		return -1, fmt.Errorf("netlink bind: %w", bindErr)
	}

	return fd, nil
}
