//go:build linux

package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/larsartmann/httputil"
)

type cachingFS struct {
	handler http.Handler
}

func (c cachingFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int64(staticCacheMaxAge.Seconds())))
	c.handler.ServeHTTP(w, r)
}

var loggingMiddleware httputil.Middleware = httputil.Logging(slog.Default())

var securityMiddleware httputil.Middleware = httputil.SecurityHeaders(httputil.SecurityHeadersConfig{
	ContentTypeNosniff: true,
	FrameOptions:       "DENY",
	ReferrerPolicy:     "no-referrer",
	ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
})

var requestIDMiddleware httputil.Middleware = httputil.RequestID(httputil.RequestIDConfig{
	HeaderName:    "X-Request-ID",
	ForwardHeader: "X-Request-ID",
	GenerateID: func() string {
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
	},
})
