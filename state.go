//go:build linux

package main

import (
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"os"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// stateMutator is called by setDeviceState after a successful HID commit.
// It runs while d.mu is held and must mutate only d.state (or fields
// protected by that lock). The caller persists the change to disk after
// the mutator returns.
type stateMutator func(d *Daemon)

// loadState reads the persisted state from disk. Returns true if a valid
// state file was found and applied, false if the file was missing, unreadable,
// invalid, or contained invalid enum values (caller should keep its defaults).
func (d *Daemon) loadState() bool {
	data, err := os.ReadFile(d.config.StateFile())
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn(
				"failed to read state file, keeping current state",
				"path",
				d.config.StateFile(),
				"error",
				err,
			)
			_ = os.Remove(d.config.StateFile() + ".tmp")
		}

		return false
	}

	var loaded pixy.State

	jsonErr := json.Unmarshal(data, &loaded, json.RejectUnknownMembers(true))
	if jsonErr != nil {
		slog.Warn(
			"failed to parse state file, using defaults",
			"path",
			d.config.StateFile(),
			"error",
			jsonErr,
		)

		return false
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

		return false
	}

	d.state = loaded

	if loaded.SchemaVersion != pixy.CurrentSchemaVersion {
		slog.Warn(
			"state file schema version mismatch — loading anyway",
			"path",
			d.config.StateFile(),
			"got",
			loaded.SchemaVersion,
			"want",
			pixy.CurrentSchemaVersion,
		)
		d.state.SchemaVersion = pixy.CurrentSchemaVersion
	}

	return true
}

func (d *Daemon) ensureStateDir() error {
	err := os.MkdirAll(d.config.StateDir, pixy.PermissionStateDir)
	if err != nil {
		return fmt.Errorf("ensure state dir %s: %w", d.config.StateDir, err)
	}

	return nil
}

func (d *Daemon) saveState() error {
	err := d.ensureStateDir()
	if err != nil {
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
