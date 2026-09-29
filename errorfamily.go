//go:build linux

package main

import (
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errorFamiliesRegistered sync.Once //nolint:gochecknoglobals // lazy init, runs once per process

// registerErrorFamilies registers stdlib defaults with go-error-family so
// errorfamily.Classify(err), errorfamily.HTTPStatus(err), and
// errorfamily.ExitCode(err) classify non-project errors (context.Canceled,
// fs.ErrNotExist, ...) from any point in the chain. Project sentinels carry
// their family at construction (errorfamily.New*/pixy.Wrap) and need no
// registry entry.
//
// Called from NewDaemon() alongside registerMetrics(). Idempotent via sync.Once.
func registerErrorFamilies() {
	errorFamiliesRegistered.Do(func() {
		errorfamily.RegisterStdlibDefaults(errorfamily.DefaultRegistry)
	})
}
