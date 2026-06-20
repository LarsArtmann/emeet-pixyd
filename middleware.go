//go:build linux

package main

import (
	"fmt"
	"log/slog"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
)

type cachingFS struct {
	handler http.Handler
}

func (c cachingFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int64(staticCacheMaxAge.Seconds())))
	c.handler.ServeHTTP(w, r)
}

//nolint:gochecknoglobals
var loggingMiddleware = cqrshtmx.RequestLoggingSlog(slog.Default())

//nolint:gochecknoglobals,exhaustruct
var securityMiddleware = cqrshtmx.SecurityHeadersMiddlewareWithConfig(cqrshtmx.SecurityHeadersConfig{
	ContentTypeOptions: "nosniff",
	FrameOptions:       "DENY",
	ReferrerPolicy:     "no-referrer",
	ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'",
})

//nolint:gochecknoglobals
var requestIDMiddleware = cqrshtmx.ContextEnrichmentMiddleware(nil)
