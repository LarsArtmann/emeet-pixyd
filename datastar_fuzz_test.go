//go:build linux

package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/starfederation/datastar-go/datastar"
)

// FuzzReadSignals verifies that datastar.ReadSignals never panics on
// arbitrary input bodies. It exercises both the POST (body JSON) and GET
// (query param) code paths.
func FuzzReadSignals(f *testing.F) {
	f.Add([]byte(`{"pan":0,"tilt":0,"zoom":100}`), "POST")
	f.Add([]byte(`{"pan":-90,"tilt":45,"zoom":150}`), "POST")
	f.Add([]byte(`{}`), "POST")
	f.Add([]byte(`not json`), "POST")
	f.Add([]byte(`{"pan":"abc"}`), "POST")
	f.Add([]byte(``), "POST")
	f.Add([]byte(`{"pan":99999999999999999}`), "GET")

	// Richer corpus: nested objects, unicode keys, arrays, deep nesting, edge types.
	f.Add([]byte(`{"pan":0,"extra":{"nested":{"deep":true}}}`), "POST")
	f.Add([]byte(`{"\u00f6\u00e4\u00fc":1}`), "POST")
	f.Add([]byte(`[1,2,3]`), "POST")
	f.Add([]byte(`{"pan":1.5,"tilt":-0.1}`), "POST")
	f.Add([]byte(`{"pan":null,"tilt":null,"zoom":null}`), "POST")
	f.Add([]byte(`{"pan":true,"zoom":false}`), "POST")
	f.Add([]byte(`{"pan":0,"tilt":0,"zoom":100,"extra":"x"}`), "POST")
	f.Add([]byte(`{"nested":{"a":{"b":{"c":{"d":1}}}}}`), "POST")

	f.Fuzz(func(t *testing.T, body []byte, method string) {
		if method == "" {
			method = "POST"
		}

		ctx := context.Background()

		var req *http.Request

		var err error

		switch method {
		case "GET", "DELETE":
			req, err = http.NewRequestWithContext(ctx, method, "/?datastar="+string(body), nil)
		default:
			req, err = http.NewRequestWithContext(ctx, method, "/", strings.NewReader(string(body)))
		}

		if err != nil {
			t.Skip()
		}

		var signals pixy.PTZValues

		_ = datastar.ReadSignals(req, &signals) // must not panic
	})
}
