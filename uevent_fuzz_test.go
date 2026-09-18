//go:build linux

package main

import (
	"testing"

	"github.com/LarsArtmann/emeet-pixyd/internal/pixy"
)

// FuzzParseUevent fuzzes the uevent-driven probe parsing: the generic
// key=value parser and pixyModelFromUevent with the two production call
// shapes (video4linux PRODUCT= and hidraw HID_ID=). It closes the gap where
// the model-aware probing shipped without a 0118 fuzz seed corpus (TODO #165).
//
// Invariants: parsing never panics, and a model is only reported together
// with isPixy=true.
func FuzzParseUevent(f *testing.F) {
	seeds := []string{
		// Real-world shapes: original PIXY (00c0) and PIXY 2K (0118).
		"ACTION=add\nSUBSYSTEM=video4linux\nPRODUCT=328f/11c0/100",
		"PRODUCT=328f/00c0/100",
		"PRODUCT=328f/0118/100",
		"HID_ID=0003:0000328F:00000118",
		"HID_ID=0003:0000328F:000000C0",
		"PRODUCT=328f/00c0",
		"DRIVER=usbhid\nHID_ID=0003:0000328F:00000118\nHID_NAME=EMEET Pixy",
		// Degenerate shapes: zeros, truncations, no separator, huge indices.
		"PRODUCT=0000/0000/0000",
		"PRODUCT=328f",
		"PRODUCT=",
		"HID_ID=",
		"HID_ID=:",
		"HID_ID=::",
		"\n\n\n",
		"garbage",
	}

	for _, seed := range seeds {
		f.Add(seed, "PRODUCT=", "/", 0, 1)
		f.Add(seed, "HID_ID=", ":", 1, 2)
	}

	f.Fuzz(func(t *testing.T, data, prefix, sep string, vendorIdx, productIdx int) {
		parsed := parseUevent(data)
		_ = parsed

		model, isPixy := pixyModelFromUevent([]byte(data), prefix, sep, vendorIdx, productIdx)
		if !isPixy && model != "" {
			t.Fatalf("model %q reported with isPixy=false for %q", model, data)
		}

		if isPixy && model != pixy.ModelOriginal && model != pixy.Model2K {
			t.Fatalf("isPixy=true with unknown model %q for %q", model, data)
		}
	})
}
