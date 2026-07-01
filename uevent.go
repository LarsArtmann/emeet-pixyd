//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	ueventAdd     = "add"
	ueventRemove  = "remove"
	ueventBufSize = 4096
)

type uevent struct {
	Action  string
	Subsys  string
	DevPath string
}

func parseUevent(data string) uevent {
	//nolint:exhaustruct
	evt := uevent{}

	for line := range strings.SplitSeq(data, "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		switch key {
		case "ACTION":
			evt.Action = val
		case "SUBSYSTEM":
			evt.Subsys = val
		case "DEVPATH":
			evt.DevPath = val
		}
	}

	return evt
}

func isRelevantUevent(evt uevent) bool {
	if evt.Action != ueventAdd && evt.Action != ueventRemove {
		return false
	}

	return evt.Subsys == "video4linux" || evt.Subsys == "hidraw"
}

// UeventListener listens for kernel uevents and signals the provided channel
// when a relevant device event (video4linux/hidraw add/remove) occurs.
// The production implementation uses netlink; tests can inject a fake.
type UeventListener interface {
	Listen(ctx context.Context, ch chan<- struct{})
}

// netlinkUeventListener is the production UeventListener using real netlink.
type netlinkUeventListener struct{}

func (netlinkUeventListener) Listen(ctx context.Context, ch chan<- struct{}) {
	listenNetlinkUevents(ctx, ch)
}

// listenNetlinkUevents connects to the kernel netlink socket and forwards
// relevant device uevents to the provided channel. Returns early if the
// socket cannot be opened (hotplug disabled).
func listenNetlinkUevents(ctx context.Context, ch chan<- struct{}) {
	f, err := os.Open("/sys/kernel/uevent_seqnum")
	if err != nil {
		slog.Error("uevent: cannot open uevent_seqnum, disabling hotplug", "error", err)

		return
	}

	_ = f.Close()

	fd, err := unixSocketUevent()
	if err != nil {
		slog.Error("uevent: cannot create netlink socket, disabling hotplug", "error", err)

		return
	}

	go func() {
		<-ctx.Done()

		_ = fd.Close()
	}()

	buf := make([]byte, ueventBufSize)
	for {
		n, readErr := fd.Read(buf)
		if readErr != nil {
			if ctx.Err() != nil {
				return
			}

			slog.Debug("uevent read error, retrying", "error", readErr)

			continue
		}

		evt := parseUevent(string(buf[:n]))
		if !isRelevantUevent(evt) {
			continue
		}

		slog.Debug("uevent", "action", evt.Action, "subsys", evt.Subsys, "devpath", evt.DevPath)
		recordUevent(evt.Action, evt.Subsys) //nolint:contextcheck // uevent goroutine has no inherited context

		select {
		case ch <- struct{}{}:
		case <-ctx.Done():
			return
		}
	}
}

// noopUeventListener is a no-op UeventListener for tests.
type noopUeventListener struct{}

func (noopUeventListener) Listen(context.Context, chan<- struct{}) {}

func unixSocketUevent() (*os.File, error) {
	fd, err := unixOpenNetlinkKobjectUevent()
	if err != nil {
		return nil, fmt.Errorf("uevent socket: %w", err)
	}

	return os.NewFile(uintptr(fd), "uevent"), nil
}
