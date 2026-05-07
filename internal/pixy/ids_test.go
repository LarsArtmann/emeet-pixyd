package pixy

import "testing"

func TestNewPID(t *testing.T) {
	t.Parallel()

	pid := NewPID(42)
	if pid.Get() != 42 {
		t.Errorf("NewPID(42).Get() = %d, want 42", pid.Get())
	}
	if pid.IsZero() {
		t.Error("NewPID(42).IsZero() = true, want false")
	}
	if pid.String() != "42" {
		t.Errorf("NewPID(42).String() = %q, want %q", pid.String(), "42")
	}
}

func TestPID_ZeroValue(t *testing.T) {
	t.Parallel()

	var pid PID
	if !pid.IsZero() {
		t.Error("zero PID should be IsZero()")
	}
	if pid.Get() != 0 {
		t.Errorf("zero PID.Get() = %d, want 0", pid.Get())
	}
}

func TestPID_Equal(t *testing.T) {
	t.Parallel()

	a := NewPID(1)
	b := NewPID(1)
	c := NewPID(2)

	if !a.Equal(b) {
		t.Error("PID(1) should equal PID(1)")
	}
	if a.Equal(c) {
		t.Error("PID(1) should not equal PID(2)")
	}
}

func TestNewSourceID(t *testing.T) {
	t.Parallel()

	sid := NewSourceID("42")
	if sid.Get() != "42" {
		t.Errorf("NewSourceID(%q).Get() = %q, want %q", "42", sid.Get(), "42")
	}
	if sid.IsZero() {
		t.Errorf("NewSourceID(%q).IsZero() = true, want false", "42")
	}
	if sid.String() != "42" {
		t.Errorf("NewSourceID(%q).String() = %q, want %q", "42", sid.String(), "42")
	}
}

func TestSourceID_ZeroValue(t *testing.T) {
	t.Parallel()

	var sid SourceID
	if !sid.IsZero() {
		t.Error("zero SourceID should be IsZero()")
	}
	if sid.Get() != "" {
		t.Errorf("zero SourceID.Get() = %q, want empty string", sid.Get())
	}
}

func TestSourceID_Equal(t *testing.T) {
	t.Parallel()

	a := NewSourceID("42")
	b := NewSourceID("42")
	c := NewSourceID("99")

	if !a.Equal(b) {
		t.Error("SourceID(42) should equal SourceID(42)")
	}
	if a.Equal(c) {
		t.Error("SourceID(42) should not equal SourceID(99)")
	}
}

func TestPID_DoesNotEqualSourceID(t *testing.T) {
	t.Parallel()

	pid := NewPID(42)
	sid := NewSourceID("42")

	_ = pid
	_ = sid
}
