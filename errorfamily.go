//go:build linux

package main

import (
	"sync"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

var errorFamiliesRegistered sync.Once //nolint:gochecknoglobals // lazy init, runs once per process

// registerErrorFamilies maps the daemon's sentinel errors to go-error-family
// behavioral classifications. This enables errorfamily.Classify(err),
// errorfamily.HTTPStatus(err), and errorfamily.ExitCode(err) to derive correct
// HTTP status codes and BSD sysexits exit codes from any error in the chain,
// without per-call-site switch statements.
//
// Called from NewDaemon() alongside registerMetrics(). Idempotent via sync.Once.
func registerErrorFamilies() {
	errorFamiliesRegistered.Do(func() {
		errorfamily.RegisterStdlibDefaults(errorfamily.DefaultRegistry)

		errorfamily.RegisterClassifications(map[error]errorfamily.Family{
			// Infrastructure — system cannot serve (device gone, deps missing).
			pixy.ErrPIXYNotConnected:      errorfamily.Infrastructure,
			pixy.ErrHIDDeviceNotAvailable: errorfamily.Infrastructure,
			errDeviceNotFound:             errorfamily.Infrastructure,
			ErrAudioSourceNotFound:        errorfamily.Infrastructure,

			// Rejection — bad input, unauthorized, or invalid value (user's fault).
			ErrInvalidValue:            errorfamily.Rejection,
			pixy.ErrInvalidCameraState: errorfamily.Rejection,
			pixy.ErrInvalidAudioMode:   errorfamily.Rejection,
			pixy.ErrInvalidPresetName:  errorfamily.Rejection,

			// Rejection — config validation (bad configuration).
			pixy.ErrStateDirEmpty:       errorfamily.Rejection,
			pixy.ErrPollIntervalZero:    errorfamily.Rejection,
			pixy.ErrDebounceCountZero:   errorfamily.Rejection,
			pixy.ErrWebAddrEmpty:        errorfamily.Rejection,
			pixy.ErrInvalidAutoMode:     errorfamily.Rejection,
			pixy.ErrInvalidDefaultAudio: errorfamily.Rejection,

			// Transient — temporary failure, might recover on retry.
			errJPEGMaxIterations: errorfamily.Transient,
			errNoHIDResponse:     errorfamily.Transient,
			errHIDWriteZero:      errorfamily.Transient,
			errUnrecognizedHID:   errorfamily.Transient,
		})
	})
}
