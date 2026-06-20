//go:build linux

package main

import "testing"

// FuzzParsePTZValue ensures parsePTZValue never panics on arbitrary input
// and that the rel-prefix detection is consistent: if it parses without error,
// the returned value must be deterministic for the same input.
func FuzzParsePTZValue(f *testing.F) {
	f.Add("50")
	f.Add("-90")
	f.Add("rel+10")
	f.Add("rel-5")
	f.Add("0")
	f.Add("rel")
	f.Add("")
	f.Add("abc")

	f.Fuzz(func(t *testing.T, input string) {
		val, relative, err := parsePTZValue(input)
		if err != nil {
			return
		}

		// Relative mode requires the "rel" prefix.
		if relative && len(input) < 4 {
			t.Errorf("relative=true for short input %q", input)
		}

		// Absolute mode must not start with "rel".
		if !relative && len(input) >= 3 && input[:3] == "rel" {
			t.Errorf("absolute=true for rel-prefixed input %q", input)
		}

		// Value must be deterministic (re-parse and compare).
		val2, _, err2 := parsePTZValue(input)
		if err2 != nil {
			t.Errorf("non-deterministic: first call succeeded, second failed for %q", input)
		}

		if val != val2 {
			t.Errorf("non-deterministic value: %d vs %d for %q", val, val2, input)
		}
	})
}
