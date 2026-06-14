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

func (d *Daemon) listenUevents(ctx context.Context, ch chan<- struct{}) {
	f, err := os.Open("/sys/kernel/uevent_seqnum")
	if err != nil {
		slog.Warn("uevent: cannot open uevent_seqnum, disabling hotplug", "error", err)

		return
	}

	_ = f.Close()

	fd, err := unixSocketUevent()
	if err != nil {
		slog.Warn("uevent: cannot create netlink socket, disabling hotplug", "error", err)

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

		slog.Info("uevent", "action", evt.Action, "subsys", evt.Subsys, "devpath", evt.DevPath)
		recordUevent(evt.Action, evt.Subsys) //nolint:contextcheck // uevent goroutine has no inherited context

		ch <- struct{}{}
	}
}

func unixSocketUevent() (*os.File, error) {
	fd, err := unixOpenNetlinkKobjectUevent()
	if err != nil {
		return nil, fmt.Errorf("uevent socket: %w", err)
	}

	return os.NewFile(uintptr(fd), "uevent"), nil
}
