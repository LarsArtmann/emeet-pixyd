//go:build linux

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	ueventAdd    = "add"
	ueventRemove = "remove"
)

type uevent struct {
	Action  string
	Subsys  string
	DevPath string
}

func parseUevent(data string) uevent {
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

func (d *Daemon) listenUevents(ch chan<- struct{}) {
	f, err := os.Open("/sys/kernel/uevent_seqnum")
	if err != nil {
		slog.Debug("uevent: cannot open uevent_seqnum, disabling hotplug", "error", err)

		return
	}
	_ = f.Close()

	fd, err := unixSocketUevent()
	if err != nil {
		slog.Debug("uevent: cannot create netlink socket, disabling hotplug", "error", err)

		return
	}
	defer fd.Close()

	buf := make([]byte, 4096)
	for {
		n, readErr := fd.Read(buf)
		if readErr != nil {
			slog.Debug("uevent read error", "error", readErr)

			return
		}

		evt := parseUevent(string(buf[:n]))
		if !isRelevantUevent(evt) {
			continue
		}

		slog.Info("uevent", "action", evt.Action, "subsys", evt.Subsys, "devpath", evt.DevPath)
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
