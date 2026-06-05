//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

type stateSetter func(d *Daemon)

func (d *Daemon) loadState() {
	data, err := os.ReadFile(d.config.StateFile())
	if err != nil {
		if !os.IsNotExist(err) {
			_ = os.Remove(d.config.StateFile() + ".tmp")
		}

		return
	}

	var loaded pixy.State

	jsonErr := json.Unmarshal(data, &loaded)
	if jsonErr != nil {
		slog.Warn(
			"failed to parse state file, using defaults",
			"path",
			d.config.StateFile(),
			"error",
			jsonErr,
		)

		return
	}

	if !loaded.Valid() {
		slog.Warn(
			"invalid state values, using defaults",
			"path",
			d.config.StateFile(),
			"camera",
			loaded.Camera,
			"audio",
			loaded.Audio,
			"autoMode",
			loaded.AutoMode,
		)

		return
	}

	d.state = loaded
}

func (d *Daemon) ensureStateDir() error {
	err := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if err != nil {
		return fmt.Errorf("ensure state dir %s: %w", d.config.StateDir, err)
	}

	return nil
}

func (d *Daemon) saveState() error {
	if err := d.ensureStateDir(); err != nil {
		return err
	}

	data, err := json.Marshal(d.state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := d.config.StateFile() + ".tmp"
	writeErr := os.WriteFile(tmp, data, pixy.PermissionStateFile)
	if writeErr != nil {
		return fmt.Errorf("write temp state: %w", writeErr)
	}

	renameErr := os.Rename(tmp, d.config.StateFile())
	if renameErr != nil {
		return fmt.Errorf("rename state: %w", renameErr)
	}

	return nil
}

// saveStateOrLog calls saveState and logs any error with the given message.
// Caller must hold d.mu.
func (d *Daemon) saveStateOrLog(msg string) {
	err := d.saveState()
	if err != nil {
		slog.Error(msg, "error", err)
	}
}
