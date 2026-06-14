//go:build linux

package pixy

import "testing"

func TestAutoMode_IsOff(t *testing.T) {
	t.Parallel()

	if AutoOff.IsOff() != true {
		t.Error("AutoOff.IsOff() = false, want true")
	}

	if AutoFull.IsOff() != false {
		t.Error("AutoFull.IsOff() = true, want false")
	}

	if AutoTrackingOnly.IsOff() != false {
		t.Error("AutoTrackingOnly.IsOff() = true, want false")
	}
}

func TestAutoMode_Toggle(t *testing.T) {
	t.Parallel()

	if got := AutoOff.Toggle(); got != AutoFull {
		t.Errorf("AutoOff.Toggle() = %q, want %q", got, AutoFull)
	}

	if got := AutoFull.Toggle(); got != AutoOff {
		t.Errorf("AutoFull.Toggle() = %q, want %q", got, AutoOff)
	}

	if got := AutoTrackingOnly.Toggle(); got != AutoOff {
		t.Errorf("AutoTrackingOnly.Toggle() = %q, want %q", got, AutoOff)
	}
}

func TestAutoMode_ActivatesTracking(t *testing.T) {
	t.Parallel()

	if AutoFull.ActivatesTracking() != true {
		t.Error("AutoFull.ActivatesTracking() = false, want true")
	}

	if AutoTrackingOnly.ActivatesTracking() != true {
		t.Error("AutoTrackingOnly.ActivatesTracking() = false, want true")
	}

	if AutoPrivacyOnly.ActivatesTracking() != false {
		t.Error("AutoPrivacyOnly.ActivatesTracking() = true, want false")
	}

	if AutoOff.ActivatesTracking() != false {
		t.Error("AutoOff.ActivatesTracking() = true, want false")
	}
}

func TestAutoMode_ActivatesAudio(t *testing.T) {
	t.Parallel()

	if AutoFull.ActivatesAudio() != true {
		t.Error("AutoFull.ActivatesAudio() = false, want true")
	}

	if AutoTrackingOnly.ActivatesAudio() != false {
		t.Error("AutoTrackingOnly.ActivatesAudio() = true, want false")
	}
}

func TestAutoMode_ActivatesPrivacy(t *testing.T) {
	t.Parallel()

	if AutoFull.ActivatesPrivacy() != true {
		t.Error("AutoFull.ActivatesPrivacy() = false, want true")
	}

	if AutoTrackingOnly.ActivatesPrivacy() != true {
		t.Error("AutoTrackingOnly.ActivatesPrivacy() = false, want true")
	}

	if AutoPrivacyOnly.ActivatesPrivacy() != true {
		t.Error("AutoPrivacyOnly.ActivatesPrivacy() = false, want true")
	}

	if AutoOff.ActivatesPrivacy() != false {
		t.Error("AutoOff.ActivatesPrivacy() = true, want false")
	}
}

func TestAutoMode_SwitchesSource(t *testing.T) {
	t.Parallel()

	if AutoFull.SwitchesSource() != true {
		t.Error("AutoFull.SwitchesSource() = false, want true")
	}

	if AutoTrackingOnly.SwitchesSource() != false {
		t.Error("AutoTrackingOnly.SwitchesSource() = true, want false")
	}

	if AutoPrivacyOnly.SwitchesSource() != false {
		t.Error("AutoPrivacyOnly.SwitchesSource() = true, want false")
	}

	if AutoOff.SwitchesSource() != false {
		t.Error("AutoOff.SwitchesSource() = true, want false")
	}
}
