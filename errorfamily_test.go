//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestErrorFamilies_InfrastructureSentinels(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	cases := []error{
		pixy.ErrPIXYNotConnected,
		pixy.ErrHIDDeviceNotAvailable,
		errDeviceNotFound,
		ErrAudioSourceNotFound,
	}

	for _, err := range cases {
		if got := errorfamily.Classify(err); got != errorfamily.Infrastructure {
			t.Errorf("Classify(%v) = %v, want Infrastructure", err, got)
		}

		if got := errorfamily.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Errorf("HTTPStatus(%v) = %d, want %d", err, got, http.StatusServiceUnavailable)
		}

		if got := errorfamily.ExitCode(err); got != 69 {
			t.Errorf("ExitCode(%v) = %d, want 69", err, got)
		}
	}
}

func TestErrorFamilies_RejectionSentinels(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	cases := []error{
		ErrInvalidValue,
		pixy.ErrInvalidCameraState,
		pixy.ErrInvalidAudioMode,
		pixy.ErrInvalidPresetName,
		pixy.ErrStateDirEmpty,
		pixy.ErrPollIntervalZero,
		pixy.ErrDebounceCountZero,
		pixy.ErrWebAddrEmpty,
		pixy.ErrInvalidAutoMode,
		pixy.ErrInvalidDefaultAudio,
	}

	for _, err := range cases {
		if got := errorfamily.Classify(err); got != errorfamily.Rejection {
			t.Errorf("Classify(%v) = %v, want Rejection", err, got)
		}

		if got := errorfamily.HTTPStatus(err); got != http.StatusBadRequest {
			t.Errorf("HTTPStatus(%v) = %d, want %d", err, got, http.StatusBadRequest)
		}
	}
}

func TestErrorFamilies_TransientSentinels(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	cases := []error{
		errJPEGMaxIterations,
		errNoHIDResponse,
		errHIDWriteZero,
		errUnrecognizedHID,
	}

	for _, err := range cases {
		if got := errorfamily.Classify(err); got != errorfamily.Transient {
			t.Errorf("Classify(%v) = %v, want Transient", err, got)
		}

		if !errorfamily.IsRetryable(err) {
			t.Errorf("IsRetryable(%v) = false, want true", err)
		}
	}
}

func TestErrorFamilies_WrappedErrorsStillClassify(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	wrapped := fmt.Errorf("outer context: %w", pixy.ErrPIXYNotConnected)

	if got := errorfamily.Classify(wrapped); got != errorfamily.Infrastructure {
		t.Errorf("Classify(wrapped ErrPIXYNotConnected) = %v, want Infrastructure", got)
	}

	wrappedRejection := fmt.Errorf("bad input: %w", ErrInvalidValue)

	if got := errorfamily.Classify(wrappedRejection); got != errorfamily.Rejection {
		t.Errorf("Classify(wrapped ErrInvalidValue) = %v, want Rejection", got)
	}
}

func TestErrorFamilies_StdlibDefaults(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	if got := errorfamily.Classify(context.Canceled); got != errorfamily.Rejection {
		t.Errorf("Classify(context.Canceled) = %v, want Rejection", got)
	}

	if got := errorfamily.Classify(context.DeadlineExceeded); got != errorfamily.Transient {
		t.Errorf("Classify(context.DeadlineExceeded) = %v, want Transient", got)
	}
}

func TestErrorFamilies_StreamErrorsAreInfrastructure(t *testing.T) {
	t.Parallel()

	streamErrs := []*errorfamily.Error{
		errStreamNoFrame,
		errStreamInUse,
		errStreamNoDevice,
		errStreamFFmpeg,
		errStreamNotSupported,
		errStreamPipe,
		errStreamStart,
	}

	for _, err := range streamErrs {
		if got := errorfamily.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Errorf("HTTPStatus(%v) = %d, want 503", err.ErrorCode(), got)
		}
	}
}

func TestErrorFamilies_UnknownErrorDefaultsToTransient(t *testing.T) {
	t.Parallel()

	unknown := errors.New("something unexpected")

	if got := errorfamily.Classify(unknown); got != errorfamily.Transient {
		t.Errorf("Classify(unknown) = %v, want Transient (fail-open default)", got)
	}

	if got := errorfamily.HTTPStatus(unknown); got != http.StatusServiceUnavailable {
		t.Errorf("HTTPStatus(unknown) = %d, want 503", got)
	}
}
