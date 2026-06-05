//go:build linux

package main

import (
	"errors"
	"fmt"
	"strings"
)

const errorPrefix = "error: "

// CommandError wraps a command operation error with a descriptive label.
type CommandError struct {
	Op  string // label for the operation that failed
	Err error
}

func (e *CommandError) Error() string { return errorPrefix + e.Op + ": " + e.Err.Error() }

func (e *CommandError) Unwrap() error { return e.Err }

// CommandResult carries a command's outcome: success message or structured error,
// plus optional web UI toast information.
type CommandResult struct {
	Message   string
	Err       error
	Toast     string
	ToastType string
}

func okResult(msg string) CommandResult {
	//nolint:exhaustruct
	return CommandResult{Message: msg}
}

func errResult(op string, err error) CommandResult {
	//nolint:exhaustruct
	return CommandResult{Err: &CommandError{Op: op, Err: err}}
}

func errResultMsg(msg string) CommandResult {
	//nolint:exhaustruct
	return CommandResult{Err: fmt.Errorf("%s%s", errorPrefix, msg)}
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

// IsCommandErrorResponse reports whether s is a legacy command error response string.
func IsCommandErrorResponse(s string) bool { return strings.HasPrefix(s, errorPrefix) }

var (
	// ErrAudioSourceNotFound is returned when no PIXY audio source is found in PipeWire.
	ErrAudioSourceNotFound = errors.New("PIXY audio source not found")
	// ErrInvalidValue is returned when a PTZ value is out of range.
	ErrInvalidValue = errors.New("invalid value")
)
