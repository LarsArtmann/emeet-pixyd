package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn with os.Stdout redirected into a pipe and returns
// everything written during fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w

	t.Cleanup(func() { os.Stdout = orig })

	fn()

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("close stdout pipe: %v", closeErr)
	}

	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}

	return string(out)
}

// runFlag invokes handleFlag with the given command-line args (argv[0] is
// filled in) and returns the captured stdout plus whether the flag was
// handled. Not parallel: it swaps os.Stdout and os.Args.
func runFlag(t *testing.T, args ...string) (string, bool) {
	t.Helper()

	var handled bool

	origArgs := os.Args

	os.Args = append([]string{"emeet-pixyd"}, args...)

	t.Cleanup(func() { os.Args = origArgs })

	out := captureStdout(t, func() { handled = handleFlag() })

	return out, handled
}

// TestVersionFlag pins the `--version` contract: exact output shape
// ("emeet-pixyd <buildVersion>", single line) with the default "dev" build
// version, the -v alias, and that real commands still fall through to the
// socket path. The ldflags-injected round-trip (buildVersion == release tag)
// is pinned end to end by the release workflow's smoke job, which downloads
// the published artifact and asserts `--version` equals the tag — mutating
// the package-global here would race the parallel tests that read it.
//
//nolint:paralleltest // swaps os.Stdout and os.Args process-globals; the subtests must stay sequential
func TestVersionFlag(t *testing.T) {
	t.Run("dev default shape", func(t *testing.T) {
		out, handled := runFlag(t, "--version")

		if !handled {
			t.Fatal("handleFlag handled --version = false, want true")
		}

		if want := "emeet-pixyd " + buildVersion + "\n"; out != want {
			t.Errorf("version output = %q, want %q", out, want)
		}
	})

	t.Run("short alias", func(t *testing.T) {
		out, handled := runFlag(t, "-v")

		if !handled {
			t.Fatal("handleFlag handled -v = false, want true")
		}

		if want := "emeet-pixyd " + buildVersion + "\n"; out != want {
			t.Errorf("-v output = %q, want %q", out, want)
		}
	})

	t.Run("commands fall through to socket path", func(t *testing.T) {
		for _, cmd := range []string{"status", "version", "privacy", "preset list"} {
			if _, handled := runFlag(t, cmd); handled {
				t.Errorf("handleFlag handled %q, want fall-through", cmd)
			}
		}
	})

	t.Run("no args falls through", func(t *testing.T) {
		if _, handled := runFlag(t); handled {
			t.Error("handleFlag handled no args, want fall-through (daemon start)")
		}
	})

	t.Run("single line output", func(t *testing.T) {
		out, _ := runFlag(t, "--version")

		if lines := strings.Count(out, "\n"); lines != 1 {
			t.Errorf("version output has %d newlines, want exactly 1: %q", lines, out)
		}
	})
}
