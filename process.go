//go:build linux

package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	wpctl      = "wpctl"
	notifySend = "notify-send"
)

func ppidOf(pid pixy.PID) pixy.PID {
	statData, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid.Get()), "stat"))
	if err != nil {
		return pixy.PID{}
	}

	statStr := string(statData)

	_, after, ok := strings.CutLast(statStr, ")")
	if !ok {
		return pixy.PID{}
	}

	fields := strings.Fields(after)
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

		if procFDsOpenDevice(filepath.Join("/proc", proc.Name(), "fd"), videoDev) {
			return true
		}
	}

	return false
}

// procFDsOpenDevice reports whether any fd of the given /proc/<pid>/fd
// directory links to videoDev. A vanished process (ENOENT mid-scan) is an
// expected race and is skipped silently; other failures are debug-logged.
func procFDsOpenDevice(fdPath, videoDev string) bool {
	fdEntries, err := os.ReadDir(fdPath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Debug("proc fd scan failed", "path", fdPath, "err", err)
		}

		return false
	}

	for _, fd := range fdEntries {
		link, err := os.Readlink(filepath.Join(fdPath, fd.Name()))
		if err != nil {
			if !os.IsNotExist(err) {
				slog.Debug("proc fd readlink failed", "path", fdPath, "err", err)
			}

			continue
		}

		if link == videoDev {
			return true
		}
	}

	return false
}

func (d *Daemon) findPixySource(
	ctx context.Context,
) (pixy.SourceID, error) { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	out, err := d.deps.commander.Output(ctx, wpctl, "status")
	if err != nil {
		return pixy.SourceID{}, errorfamily.Wrap(err, errorfamily.Classify(err), "audio.find_source", "findPixySource")
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

	return pixy.SourceID{}, errorfamily.WrapInfrastructuref(
		ErrAudioSourceNotFound,
		"audio.find_source_notfound",
		"findPixySource",
	)
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
