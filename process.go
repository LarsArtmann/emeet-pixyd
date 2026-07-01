//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

const (
	wpctl      = "wpctl"
	notifySend = "notify-send"
)

// ProcessInspector abstracts /proc filesystem operations for testability.
// The production implementation reads real /proc; tests can inject a fake
// that returns deterministic results without depending on system state.
type ProcessInspector interface {
	// PPIDOf returns the parent PID of the given process, or zero if unknown.
	PPIDOf(pid pixy.PID) pixy.PID
	// IsDescendantOf reports whether pid is a descendant of ancestor.
	IsDescendantOf(pid, ancestor pixy.PID) bool
	// IsCameraInUse reports whether any non-self, non-descendant process
	// has the given video device open.
	IsCameraInUse(videoDev string) bool
}

// procInspector is the production ProcessInspector that reads real /proc.
type procInspector struct{}

func (procInspector) PPIDOf(pid pixy.PID) pixy.PID {
	return ppidOf(pid)
}

func (procInspector) IsDescendantOf(pid, ancestor pixy.PID) bool {
	return isDescendantOf(pid, ancestor)
}

func (procInspector) IsCameraInUse(videoDev string) bool {
	return isCameraInUse(videoDev)
}

func ppidOf(pid pixy.PID) pixy.PID {
	statData, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid.Get()), "stat"))
	if err != nil {
		return pixy.PID{}
	}

	statStr := string(statData)

	lastParen := strings.LastIndex(statStr, ")")
	if lastParen == -1 {
		return pixy.PID{}
	}

	fields := strings.Fields(statStr[lastParen+1:])
	if len(fields) < 2 {
		return pixy.PID{}
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return pixy.PID{}
	}

	return pixy.NewPID(ppid)
}

const maxDescendantDepth = 32

func isDescendantOf(pid, ancestor pixy.PID) bool {
	for range maxDescendantDepth {
		ppid := ppidOf(pid)
		if ppid.IsZero() || ppid.Equal(pid) {
			return false
		}

		if ppid.Equal(ancestor) {
			return true
		}

		pid = ppid
	}

	return false
}

func isCameraInUse(videoDev string) bool {
	if videoDev == "" {
		return false
	}

	myPID := pixy.NewPID(os.Getpid())

	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, proc := range procEntries {
		if !proc.IsDir() {
			continue
		}

		rawPID, parseErr := strconv.Atoi(proc.Name())
		if parseErr != nil {
			continue
		}

		pid := pixy.NewPID(rawPID)
		if pid.Equal(myPID) || isDescendantOf(pid, myPID) {
			continue
		}

		fdPath := filepath.Join("/proc", proc.Name(), "fd")

		fdEntries, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, fd := range fdEntries {
			link, err := os.Readlink(filepath.Join(fdPath, fd.Name()))
			if err != nil {
				continue
			}

			if link == videoDev {
				return true
			}
		}
	}

	return false
}

func (d *Daemon) findPixySource(ctx context.Context) (pixy.SourceID, error) {
	out, err := d.deps.commander.Output(ctx, wpctl, "status")
	if err != nil {
		return pixy.SourceID{}, fmt.Errorf("findPixySource: %w", err)
	}

	for line := range strings.SplitSeq(string(out), "\n") {
		if isPixyName(line) {
			for field := range strings.FieldsSeq(line) {
				field = strings.TrimSuffix(field, ".")

				_, parseErr := strconv.Atoi(field)
				if parseErr == nil {
					return pixy.NewSourceID(field), nil
				}
			}
		}
	}

	return pixy.SourceID{}, fmt.Errorf("findPixySource: %w", ErrAudioSourceNotFound)
}

func (d *Daemon) setDefaultSource(ctx context.Context, sourceID pixy.SourceID) {
	err := d.deps.commander.Run(ctx, wpctl, "set-default", sourceID.Get())
	if err != nil {
		slog.Error("failed to set default audio source", "id", sourceID.Get(), "error", err)
	}
}

func (d *Daemon) notifyCmd(ctx context.Context, title, body string) {
	err := d.deps.commander.Run(ctx, notifySend, "-a", "emeet-pixyd", title, body)
	if err != nil {
		slog.Warn("notification failed", "error", err)
	}
}
