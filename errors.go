//go:build linux

package main

import (
	"errors"
)

const errorPrefix = "error: "

// CommandError wraps a command operation error with a descriptive label.
// If Err is nil, Op carries the full error message (leaf error).
type CommandError struct {
	Op  string // label for the operation that failed (or full message when Err is nil)
	Err error
}

func (e *CommandError) Error() string {
	if e.Err != nil {
		return errorPrefix + e.Op + ": " + e.Err.Error()
	}

	return errorPrefix + e.Op
}

func (e *CommandError) Unwrap() error { return e.Err }

// CommandResult carries a command's outcome: success message or structured error.
type CommandResult struct {
	Message string
	Err     error
}

func okResult(msg string) CommandResult {
	return CommandResult{Message: msg, Err: nil}
}

func errResult(op string, err error) CommandResult {
	return CommandResult{Message: "", Err: &CommandError{Op: op, Err: err}}
}

// errResultMsg creates a leaf CommandError carrying a static message.
func errResultMsg(msg string) CommandResult {
	return CommandResult{
		Message: "",
		Err:     &CommandError{Op: msg}, //nolint:exhaustruct // leaf error — Err intentionally nil
	}
}

// String returns the text representation for socket/CLI output.
func (r CommandResult) String() string {
	if r.Err != nil {
		return r.Err.Error()
	}

	return r.Message
}

// IsError reports whether this result represents an error.
func (r CommandResult) IsError() bool {
	return r.Err != nil
}

func errStr(e error) string {
	if e == nil {
		return ""
	}

	return e.Error()
}

var (
	// ErrAudioSourceNotFound is returned when no PIXY audio source is found in PipeWire.
	ErrAudioSourceNotFound = errors.New("PIXY audio source not found")
	// ErrInvalidValue is returned when a PTZ value is out of range.
	ErrInvalidValue = errors.New("invalid value")
	// errDeviceNotFound is returned when the video device path is empty.
	errDeviceNotFound = errors.New("device not found")
)
