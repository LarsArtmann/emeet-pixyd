//go:build linux

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

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

func findPixySource(ctx context.Context) (pixy.SourceID, error) {
	out, err := exec.CommandContext(ctx, "wpctl", "status").Output()
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

func setDefaultSource(ctx context.Context, sourceID pixy.SourceID) {
	err := exec.CommandContext(ctx, "wpctl", "set-default", sourceID.Get()).Run()
	if err != nil {
		slog.Error("failed to set default audio source", "id", sourceID.Get(), "error", err)
	}
}

func notify(ctx context.Context, title, body string) {
	err := exec.CommandContext(ctx, "notify-send", "-a", "emeet-pixyd", title, body).Run()
	if err != nil {
		slog.Debug("notification failed", "error", err)
	}
}
