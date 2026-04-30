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

	sa := &unix.SockaddrNetlink{
		Groups: 1,
	}
	if bindErr := unix.Bind(fd, sa); bindErr != nil {
		_ = unix.Close(fd)

		return -1, fmt.Errorf("netlink bind: %w", bindErr)
	}

	return fd, nil
}
