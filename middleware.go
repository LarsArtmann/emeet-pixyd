//go:build linux

package main

import (
	"fmt"
	"net/http"
	"time"
)

type cachingFS struct {
	handler http.Handler
}

func (c cachingFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int64(staticCacheMaxAge.Seconds())))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	c.handler.ServeHTTP(w, r)
}

func ptzAxisValid(axis string) bool {
	switch axis {
	case axisPan, axisTilt, axisZoom:
		return true
	default:
		return false
	}
}

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().
			Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}
