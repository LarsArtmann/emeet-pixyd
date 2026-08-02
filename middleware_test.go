//go:build linux

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityMiddleware(t *testing.T) {
	t.Parallel()

	called := false
	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})
	handler := securityHeaderMiddleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}

	headers := []struct {
		key, want string
	}{
		{"Referrer-Policy", "no-referrer"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{
			"Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
		},
	}
	for _, h := range headers {
		got := rec.Header().Get(h.key)
		if got != h.want {
			t.Errorf("%s = %q, want %q", h.key, got, h.want)
		}
	}
}

func runRequestIDMiddleware(t *testing.T, req *http.Request) string {
	t.Helper()

	var capturedID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		capturedID = w.Header().Get("X-Request-ID")
	})
	h := requestIDMW(inner)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return capturedID
}

func TestRequestIDMiddleware_Generated(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	capturedID := runRequestIDMiddleware(t, req)

	if capturedID == "" {
		t.Error("X-Request-ID should be generated when not provided")
	}

	// Our requestIDMiddleware generates a nanosecond timestamp.
	if len(capturedID) == 0 {
		t.Errorf("generated ID is empty")
	}
}

func TestRequestIDMiddleware_Passthrough(t *testing.T) {
	t.Parallel()

	rid := "test-request-id-12345"

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("X-Request-ID", rid)
	capturedID := runRequestIDMiddleware(t, req)

	if capturedID != rid {
		t.Errorf("X-Request-ID = %q, want %q", capturedID, rid)
	}
}

func okHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
}

func TestCachingFS(t *testing.T) {
	t.Parallel()

	cfs := cachingFS{handler: okHandler()}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/static/test.js", nil)
	rec := httptest.NewRecorder()
	cfs.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc != "public, max-age=604800" {
		t.Errorf("Cache-Control = %q, want public max-age 7d", cc)
	}
}

func TestLoggingMiddleware_CapturesStatus(t *testing.T) {
	t.Parallel()

	var capturedStatus int

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		capturedStatus = http.StatusNotFound
	})
	handler := loggingMiddleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedStatus != http.StatusNotFound {
		t.Errorf("inner handler saw status %d, want %d", capturedStatus, http.StatusNotFound)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("recorder status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLoggingMiddleware_DefaultStatusOK(t *testing.T) {
	t.Parallel()

	inner := okHandler()
	handler := loggingMiddleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when WriteHeader not called, got %d", rec.Code)
	}
}

func TestLoggingMiddleware_Flusher(t *testing.T) {
	t.Parallel()

	var flushed bool

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("responseWriter should implement http.Flusher")

			return
		}

		_, _ = w.Write([]byte("chunk"))

		f.Flush()

		flushed = true
	})

	handler := loggingMiddleware(inner)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/stream", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !flushed {
		t.Error("Flush() should delegate to underlying ResponseWriter")
	}
}
