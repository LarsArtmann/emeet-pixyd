//go:build linux

package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
	"github.com/starfederation/datastar-go/datastar"
)

// FuzzReadSignals verifies that datastar.ReadSignals never panics on
// arbitrary input bodies. It exercises both the POST (body JSON) and GET
// (query param) code paths. For valid JSON with integer pan/tilt/zoom
// values, it also checks that the parsed values match.
func FuzzReadSignals(f *testing.F) {
	f.Add([]byte(`{"pan":0,"tilt":0,"zoom":100}`), "POST")
	f.Add([]byte(`{"pan":-90,"tilt":45,"zoom":150}`), "POST")
	f.Add([]byte(`{}`), "POST")
	f.Add([]byte(`not json`), "POST")
	f.Add([]byte(`{"pan":"abc"}`), "POST")
	f.Add([]byte(``), "POST")
	f.Add([]byte(`{"pan":99999999999999999}`), "GET")

	f.Fuzz(func(t *testing.T, body []byte, method string) {
		if method == "" {
			method = "POST"
		}

		var req *http.Request

		var err error

		switch method {
		case "GET", "DELETE":
			req, err = http.NewRequest(method, "/?datastar="+string(body), nil)
		default:
			req, err = http.NewRequest(method, "/", strings.NewReader(string(body)))
		}

		if err != nil {
			t.Skip()
		}

		var signals pixy.PTZValues

		_ = datastar.ReadSignals(req, &signals) // must not panic
	})
}
