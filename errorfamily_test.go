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

	if got := errorfamily.ExitCode(wrapped); got != 69 {
		t.Errorf("ExitCode(wrapped ErrPIXYNotConnected) = %d, want 69", got)
	}

	wrappedRejection := fmt.Errorf("bad input: %w", ErrInvalidValue)

	if got := errorfamily.Classify(wrappedRejection); got != errorfamily.Rejection {
		t.Errorf("Classify(wrapped ErrInvalidValue) = %v, want Rejection", got)
	}

	if got := errorfamily.ExitCode(wrappedRejection); got != 1 {
		t.Errorf("ExitCode(wrapped ErrInvalidValue) = %d, want 1", got)
	}

	wrappedTransient := fmt.Errorf("hid timeout: %w", errNoHIDResponse)

	if got := errorfamily.Classify(wrappedTransient); got != errorfamily.Transient {
		t.Errorf("Classify(wrapped errNoHIDResponse) = %v, want Transient", got)
	}

	if got := errorfamily.ExitCode(wrappedTransient); got != 75 {
		t.Errorf("ExitCode(wrapped errNoHIDResponse) = %d, want 75", got)
	}
}

func TestErrorFamilies_CLIExitCodes(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	cases := []struct {
		name string
		err  error
		want int
	}{
		{"config validation (rejection)", pixy.ErrInvalidAutoMode, 1},
		{"device not connected (infrastructure)", pixy.ErrPIXYNotConnected, 69},
		{"HID timeout (transient)", errNoHIDResponse, 75},
		{"unknown error (transient default)", errors.New("unexpected"), 75},
		{"wrapped config error", fmt.Errorf("config bad: %w", pixy.ErrWebAddrEmpty), 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := errorfamily.ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
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

	streamErrs := []error{
		errStreamNoFrame,
		errStreamInUse,
		errStreamNoDevice,
		errStreamFFmpeg,
		errStreamNotSupported,
		errStreamPipe,
		errStreamStart,
	}

	for _, err := range streamErrs {
		code := errorfamily.Code(err)

		if got := errorfamily.HTTPStatus(err); got != http.StatusServiceUnavailable {
			t.Errorf("HTTPStatus(%v) = %d, want 503", code, got)
		}

		if got := errorfamily.ExitCode(err); got != 69 {
			t.Errorf("ExitCode(%v) = %d, want 69 (Infrastructure)", code, got)
		}

		if got := errorfamily.Classify(err); got != errorfamily.Infrastructure {
			t.Errorf("Classify(%v) = %v, want Infrastructure", code, got)
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

// TestErrorFamilies_ErrorsIsThroughComposeTrees pins sentinel matching and
// family resolution through errorfamily.Compose trees (erraudit triage
// 2026-09-29 b3/f5). Compose is live at real call sites (hid.go recv-close
// join, auto.go per-step joins, pixy ipc close join) — only single-wrap
// chains were pinned before. Compose is errors.Join: every branch must stay
// reachable by errors.Is and the family must resolve from the tree (first
// non-Transient branch wins).
func TestErrorFamilies_ErrorsIsThroughComposeTrees(t *testing.T) {
	t.Parallel()
	registerErrorFamilies()

	cause := pixy.ErrPIXYNotConnected
	primary := errorfamily.Wrapf(cause, errorfamily.Infrastructure, "hid.recv_write", "write hidraw")
	closeErr := errorfamily.Wrap(errors.New("close failed"), errorfamily.Transient, "hid.recv_close", "close hidraw")

	flat := errorfamily.Compose(primary, closeErr)

	if !errors.Is(flat, cause) {
		t.Error("errors.Is(flat, cause) = false, want true (Compose must preserve every Unwrap branch)")
	}

	if !errors.Is(flat, closeErr) {
		t.Error("errors.Is(flat, closeErr) = false, want true")
	}

	if got := errorfamily.Classify(flat); got != errorfamily.Infrastructure {
		t.Errorf("Classify(flat) = %v, want Infrastructure (first non-Transient branch)", got)
	}

	if got := errorfamily.HTTPStatus(flat); got != http.StatusServiceUnavailable {
		t.Errorf("HTTPStatus(flat) = %d, want 503", got)
	}

	// A join whose branches are themselves joins (auto.go shape: per-step
	// results composed into one autoError).
	nested := errorfamily.Compose(
		errorfamily.Compose(primary),
		errorfamily.Compose(closeErr, errors.New("extra context")),
	)

	if !errors.Is(nested, cause) || !errors.Is(nested, closeErr) {
		t.Error("errors.Is through nested Compose = false, want true for both leaves")
	}

	if got := errorfamily.Code(flat); got != "hid.recv_write" {
		t.Errorf("Code(flat) = %q, want hid.recv_write (Coded survives the join)", got)
	}

	// The wrap-depth case: cause two wraps below the join surface.
	deep := errorfamily.Compose(fmt.Errorf("outer: %w", primary), closeErr)

	if !errors.Is(deep, cause) {
		t.Error("errors.Is(deep, cause) = false, want true through wrap + join depth")
	}
}
