//go:build linux

package main

import (
	"errors"
	"strings"
)

// CommandError wraps a command operation error with a descriptive label.
type CommandError struct {
	Op  string // label for the operation that failed
	Err error
}

func (e *CommandError) Error() string { return "error: " + e.Op + ": " + e.Err.Error() }

func (e *CommandError) Unwrap() error { return e.Err }

// IsCommandErrorResponse reports whether s is a command error response string.
func IsCommandErrorResponse(s string) bool { return strings.HasPrefix(s, "error: ") }

var (
	// ErrAudioSourceNotFound is returned when no PIXY audio source is found in PipeWire.
	ErrAudioSourceNotFound = errors.New("PIXY audio source not found")
	// ErrInvalidValue is returned when a PTZ value is out of range.
	ErrInvalidValue = errors.New("invalid value")
)
