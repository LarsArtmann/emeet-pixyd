//go:build linux

package pixy

import (
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
)

func TestProperty_RangeClamp_AlwaysInRange(t *testing.T) {
	t.Parallel()

	for _, r := range []Range{PanRange, TiltRange, ZoomRange} {
		prop := func(v int) bool {
			result := r.Clamp(v)

			return result >= r.Min && result <= r.Max
		}

		if err := quick.Check(prop, &quick.Config{MaxCount: 10000}); err != nil {
			t.Errorf("Range%v.Clamp always-in-range failed: %v", r, err)
		}
	}
}

func TestProperty_RangeClamp_BelowMinReturnsMin(t *testing.T) {
	t.Parallel()

	for _, r := range []Range{PanRange, TiltRange, ZoomRange} {
		prop := func(v int) bool {
			if v >= r.Min {
				return true
			}

			return r.Clamp(v) == r.Min
		}

		if err := quick.Check(prop, &quick.Config{MaxCount: 10000}); err != nil {
			t.Errorf("Range%v.Clamp below-min failed: %v", r, err)
		}
	}
}

func TestProperty_RangeClamp_AboveMaxReturnsMax(t *testing.T) {
	t.Parallel()

	for _, r := range []Range{PanRange, TiltRange, ZoomRange} {
		prop := func(v int) bool {
			if v <= r.Max {
				return true
			}

			return r.Clamp(v) == r.Max
		}

		if err := quick.Check(prop, &quick.Config{MaxCount: 10000}); err != nil {
			t.Errorf("Range%v.Clamp above-max failed: %v", r, err)
		}
	}
}

func TestProperty_RangeClamp_IdentityInBounds(t *testing.T) {
	t.Parallel()

	for _, r := range []Range{PanRange, TiltRange, ZoomRange} {
		prop := func(v int) bool {
			if v < r.Min || v > r.Max {
				return true
			}

			return r.Clamp(v) == v
		}

		if err := quick.Check(prop, &quick.Config{MaxCount: 10000}); err != nil {
			t.Errorf("Range%v.Clamp identity failed: %v", r, err)
		}
	}
}

func TestProperty_ValidatePresetName_ValidNamesAccepted(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(42)) //nolint:gosec // test-only deterministic seed

	for range 5000 {
		name := generateValidPresetName(r)

		if err := ValidatePresetName(name); err != nil {
			t.Errorf("expected valid name %q to pass: %v", name, err)
		}
	}
}

func TestProperty_ValidatePresetName_LongNamesRejected(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(99)) //nolint:gosec // test-only deterministic seed

	for range 500 {
		base := generateValidPresetName(r)

		name := base + strings.Repeat("x", MaxPresetNameLength)

		if ValidatePresetName(name) == nil {
			t.Errorf("expected name of length %d to be rejected: %q", len([]rune(name)), name)
		}
	}
}

func TestProperty_ValidatePresetName_PathSeparatorsRejected(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(7)) //nolint:gosec // test-only deterministic seed

	for range 500 {
		base := generateValidPresetName(r)

		for _, sep := range []string{"/", "\\"} {
			name := "a" + sep + base

			if ValidatePresetName(name) == nil {
				t.Errorf("expected name with path separator to be rejected: %q", name)
			}
		}
	}
}

func TestProperty_ValidatePresetName_ControlCharsRejected(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(13)) //nolint:gosec // test-only deterministic seed

	controlChars := []rune{'\n', '\t', '\r', 0, 0x1F, 0x7F}

	for range 500 {
		base := generateValidPresetName(r)

		for _, c := range controlChars {
			name := base + "x" + string(c) + "y"

			if ValidatePresetName(name) == nil {
				t.Errorf("expected name with control char to be rejected: %q", name)
			}
		}
	}
}

// generateValidPresetName creates a random string that satisfies all ValidatePresetName rules.
func generateValidPresetName(r *rand.Rand) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_ "

	length := 1 + r.Intn(MaxPresetNameLength)

	var b strings.Builder

	b.Grow(length)

	for range length {
		b.WriteByte(letters[r.Intn(len(letters))])
	}

	name := strings.TrimSpace(b.String())

	if name == "" {
		return "default"
	}

	return name
}
