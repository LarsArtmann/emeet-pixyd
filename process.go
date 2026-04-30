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
)

func ppidOf(pid int) int {
	statData, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}

	statStr := string(statData)

	lastParen := strings.LastIndex(statStr, ")")
	if lastParen == -1 {
		return 0
	}

	fields := strings.Fields(statStr[lastParen+1:])
	if len(fields) < 2 {
		return 0
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}

	return ppid
}

const maxDescendantDepth = 32

func isDescendantOf(pid, ancestor int) bool {
	for range maxDescendantDepth {
		ppid := ppidOf(pid)
		if ppid == 0 || ppid == pid {
			return false
		}

		if ppid == ancestor {
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

	myPID := os.Getpid()

	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, proc := range procEntries {
		if !proc.IsDir() {
			continue
		}

		pid, parseErr := strconv.Atoi(proc.Name())
		if parseErr != nil {
			continue
		}

		if pid == myPID || isDescendantOf(pid, myPID) {
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

func findPixySource(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "wpctl", "status").Output()
	if err != nil {
		return "", fmt.Errorf("findPixySource: %w", err)
	}

	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.Contains(line, "EMEET") || strings.Contains(line, "Pixy") ||
			strings.Contains(line, "PIXY") {
			for field := range strings.FieldsSeq(line) {
				field = strings.TrimSuffix(field, ".")

				_, parseErr := strconv.Atoi(field)
				if parseErr == nil {
					return field, nil
				}
			}
		}
	}

	return "", fmt.Errorf("findPixySource: %w", errAudioSourceNotFound)
}

func setDefaultSource(ctx context.Context, sourceID string) {
	err := exec.CommandContext(ctx, "wpctl", "set-default", sourceID).Run()
	if err != nil {
		slog.Error("failed to set default audio source", "id", sourceID, "error", err)
	}
}

func notify(ctx context.Context, title, body string) {
	err := exec.CommandContext(ctx, "notify-send", "-a", "emeet-pixyd", title, body).Run()
	if err != nil {
		slog.Debug("notification failed", "error", err)
	}
}
