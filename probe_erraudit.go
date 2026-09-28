//go:build linux

package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errProbeSentinel = errorfamily.NewInfrastructure("probe.sentinel", "probe sentinel")

var errProbeWidened error = errorfamily.NewInfrastructure("probe.widened", "probe widened")

func probeConstructorError() error {
	return errorfamily.NewRejection("probe.rejection", "probe rejection")
}

func probeConcreteReturn() *errorfamily.Error {
	return errorfamily.NewRejection("probe.concrete", "probe concrete return")
}

func probeWrap() error {
	return errorfamily.WrapInfrastructuref(errProbeSentinel, "probe.wrap", "wrapped %d", 42)
}

func probeFmtWrap() error {
	return fmt.Errorf("probe fmt: %w", errProbeSentinel)
}

func probeJoin() error {
	return errors.Join(errProbeSentinel, errProbeWidened)
}

func probeLoggedClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		slog.Debug("probe close failed", "err", err)
	}
}
