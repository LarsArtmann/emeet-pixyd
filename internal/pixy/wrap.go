//go:build linux

package pixy

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Wrap returns err wrapped with a machine-readable code and a context
// message, inheriting the error family from the cause. Classification is
// therefore decided exactly once, at the error's root: wraps add code and
// context without ever re-classifying. Nil-safe.
func Wrap(err error, code, message string) error {
	if err == nil {
		return nil
	}

	return errorfamily.Wrap(err, errorfamily.Classify(err), code, message)
}

// Wrapf is the formatted-message variant of Wrap.
func Wrapf(err error, code, format string, args ...any) error {
	if err == nil {
		return nil
	}

	return errorfamily.Wrapf(err, errorfamily.Classify(err), code, format, args...)
}
