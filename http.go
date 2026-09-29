//go:build linux

package main

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// writeJSON encodes v as JSON and writes it with the given status and
// Content-Type header. Buffers before writing headers so a failed encode
// doesn't commit a success status.
//
func writeJSON(w http.ResponseWriter, status int, v any) error { //nolint:erraudit // family-inheriting errorfamily.Wrap; a per-function concrete error type would add no errors.AsType consumer
	var buf bytes.Buffer

	err := json.MarshalWrite(&buf, v)
	if err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Classify(err),
			"http.encode_json",
			"encode JSON response (status %d)",
			status,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, writeErr := buf.WriteTo(w); writeErr != nil {
		slog.Debug("JSON response write failed", "status", status, "err", writeErr)
	}

	return nil
}

// chain applies multiple middleware left-to-right: chain(a, b, c) wraps as a(b(c(handler))).
func chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for _, mw := range slices.Backward(middlewares) {
			final = mw(final)
		}

		return final
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter

	status int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}

	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}

	return pusher.Push(target, opts)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// loggingMiddlewareFactory returns middleware that logs each request via slog.
func loggingMiddlewareFactory(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			recorder := &statusRecorder{ResponseWriter: w} //nolint:exhaustruct // zero values are correct

			next.ServeHTTP(recorder, r)

			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration", time.Since(start),
			)
		})
	}
}

// securityHeadersMiddleware returns middleware that sets security headers.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; img-src 'self' data:; "+
				"connect-src 'self'; frame-ancestors 'none'",
		)

		next.ServeHTTP(w, r)
	})
}

// requestIDMiddleware returns middleware that generates a request ID
// and sets it in the response header.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = strconv.FormatInt(time.Now().UnixNano(), 10)
		}

		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r)
	})
}
