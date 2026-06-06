//go:build linux

package main

import (
	"errors"
)

const errorPrefix = "error: "

// CommandError wraps a command operation error with a descriptive label.
type CommandError struct {
	Op  string // label for the operation that failed
	Err error
}

func (e *CommandError) Error() string { return errorPrefix + e.Op + ": " + e.Err.Error() }

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

// commandMsgError is a static-error-compatible wrapper for simple error messages.
type commandMsgError struct{ msg string }

func (e *commandMsgError) Error() string { return errorPrefix + e.msg }

func errResultMsg(msg string) CommandResult {
	return CommandResult{Message: "", Err: &commandMsgError{msg: msg}}
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

var (
	// ErrAudioSourceNotFound is returned when no PIXY audio source is found in PipeWire.
	ErrAudioSourceNotFound = errors.New("PIXY audio source not found")
	// ErrInvalidValue is returned when a PTZ value is out of range.
	ErrInvalidValue = errors.New("invalid value")
	// errDeviceNotFound is returned when the video device path is empty.
	errDeviceNotFound = errors.New("device not found")
)
