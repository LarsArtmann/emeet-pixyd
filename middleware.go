//go:build linux

package main

import (
	"fmt"
	"log/slog"
	"net/http"
)

type cachingFS struct {
	handler http.Handler
}

func (c cachingFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int64(staticCacheMaxAge.Seconds())))
	c.handler.ServeHTTP(w, r)
}

//x
var loggingMiddleware = loggingMiddlewareFactory(slog.Default())

//x
var securityHeaderMiddleware = securityHeadersMiddleware

//x
var requestIDMW = requestIDMiddleware
