//go:build linux

package pixy

import (
	"errors"
	"strings"
	"testing"
)

func runValidTests[T any](
	t *testing.T,
	tests []struct {
		input T
		want  bool
	},
	typeName string,
	valid func(T) bool,
) {
	t.Helper()

	for _, tc := range tests {
		got := valid(tc.input)
		if got != tc.want {
			t.Errorf("%s(%v).Valid() = %v, want %v", typeName, tc.input, got, tc.want)
		}
	}
}

func TestCameraState_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input CameraState
		want  bool
	}{
		{StateIdle, true},
		{StateTracking, true},
		{StatePrivacy, true},
		{StateOffline, true},
		{CameraState("unknown"), false},
		{CameraState(""), false},
		{CameraState("IDLE"), false},
	}

	runValidTests(t, tests, "CameraState", func(v CameraState) bool { return v.Valid() })
}

func TestAudioMode_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input AudioMode
		want  bool
	}{
		{AudioNC, true},
		{AudioLive, true},
		{AudioOriginal, true},
		{AudioMode("unknown"), false},
		{AudioMode(""), false},
		{AudioMode("NC"), false},
	}

	runValidTests(t, tests, "AudioMode", func(v AudioMode) bool { return v.Valid() })
}

func TestAudioMode_Next(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input AudioMode
		want  AudioMode
	}{
		{AudioNC, AudioLive},
		{AudioLive, AudioOriginal},
		{AudioOriginal, AudioNC},
		{AudioMode("unknown"), AudioNC},
		{AudioMode(""), AudioNC},
	}

	for _, tc := range tests {
		if got := tc.input.Next(); got != tc.want {
			t.Errorf("AudioMode(%q).Next() = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestAudioMode_Next_CyclesThrough(t *testing.T) {
	t.Parallel()

	mode := AudioNC
	for _, want := range []AudioMode{AudioLive, AudioOriginal, AudioNC} {
		mode = mode.Next()
		if mode != want {
			t.Errorf("Next() = %v, want %v", mode, want)
		}
	}
}

func assertParseAudioMode(t *testing.T, input string, want AudioMode) {
	t.Helper()

	got, err := ParseAudioMode(input)
	if err != nil {
		t.Errorf("ParseAudioMode(%q) unexpected error: %v", input, err)
	}

	if got != want {
		t.Errorf("ParseAudioMode(%q) = %v, want %v", input, got, want)
	}
}

func assertParseCameraState(t *testing.T, input string, want CameraState) {
	t.Helper()

	got, err := ParseCameraState(input)
	if err != nil {
		t.Errorf("ParseCameraState(%q) unexpected error: %v", input, err)
	}

	if got != want {
		t.Errorf("ParseCameraState(%q) = %v, want %v", input, got, want)
	}
}

func expectParseCameraStateError(t *testing.T, input string) {
	t.Helper()

	_, err := ParseCameraState(input)
	if err == nil {
		t.Errorf("ParseCameraState(%q) expected error, got nil", input)
	}

	if !errors.Is(err, ErrInvalidCameraState) {
		t.Errorf("ParseCameraState(%q) error = %v, want ErrInvalidCameraState", input, err)
	}
}

func TestParseAudioMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    AudioMode
		wantErr bool
	}{
		{"nc", AudioNC, false},
		{"live", AudioLive, false},
		{"org", AudioOriginal, false},
		{"unknown", "", true},
		{"", "", true},
		{"NC", AudioNC, false},
		{"ORIGINAL", AudioOriginal, false},
		{"Live", AudioLive, false},
	}

	for _, tc := range tests {
		if tc.wantErr {
			_, err := ParseAudioMode(tc.input)
			if err == nil {
				t.Errorf("ParseAudioMode(%q) expected error, got nil", tc.input)
			}

			if !errors.Is(err, ErrInvalidAudioMode) {
				t.Errorf("ParseAudioMode(%q) error = %v, want ErrInvalidAudioMode", tc.input, err)
			}

			continue
		}

		assertParseAudioMode(t, tc.input, tc.want)
	}
}

func TestParseAudioMode_OrgMapping(t *testing.T) {
	t.Parallel()

	got, err := ParseAudioMode("org")
	if err != nil {
		t.Fatalf("ParseAudioMode(\"org\") unexpected error: %v", err)
	}

	if got != AudioOriginal {
		t.Errorf("ParseAudioMode(\"org\") = %q, want %q", got, AudioOriginal)
	}

	if string(got) != "original" {
		t.Errorf("ParseAudioMode(\"org\").string() = %q, want %q", string(got), "original")
	}
}

func TestParseCameraState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    CameraState
		wantErr bool
	}{
		{"idle", StateIdle, false},
		{"tracking", StateTracking, false},
		{"privacy", StatePrivacy, false},
		{"offline", StateOffline, false},
		{"unknown", "", true},
		{"", "", true},
		{"IDLE", StateIdle, false},
		{"Offline", StateOffline, false},
		{"PRIVACY", StatePrivacy, false},
	}

	for _, tc := range tests {
		if tc.wantErr {
			expectParseCameraStateError(t, tc.input)

			continue
		}

		assertParseCameraState(t, tc.input, tc.want)
	}
}

const testStateDir = "/tmp/test"

func TestConfig_StateFile(t *testing.T) {
	t.Parallel()

	c := Config{StateDir: testStateDir}

	want := testStateDir + "/state.json"
	if got := c.StateFile(); got != want {
		t.Errorf("Config.StateFile() = %v, want %v", got, want)
	}
}

func TestCameraState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input CameraState
		want  string
	}{
		{StateIdle, "idle"},
		{StateTracking, "tracking"},
		{StatePrivacy, "privacy"},
		{StateOffline, "offline"},
	}
	for _, tc := range tests {
		if got := tc.input.String(); got != tc.want {
			t.Errorf("CameraState(%q).String() = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestState_Valid(t *testing.T) {
	t.Parallel()

	if !DefaultState().Valid() {
		t.Error("DefaultState().Valid() = false, want true")
	}

	invalidCamera := DefaultState()

	invalidCamera.Camera = CameraState("bogus")
	if invalidCamera.Valid() {
		t.Error("State with invalid Camera should not be Valid()")
	}

	invalidAudio := DefaultState()

	invalidAudio.Audio = AudioMode("bogus")
	if invalidAudio.Valid() {
		t.Error("State with invalid Audio should not be Valid()")
	}

	invalidAuto := DefaultState()

	invalidAuto.AutoMode = AutoMode("bogus")
	if invalidAuto.Valid() {
		t.Error("State with invalid AutoMode should not be Valid()")
	}
}

func TestSourceIDBrand_Name(t *testing.T) {
	t.Parallel()

	var b sourceIDBrand
	if got := b.Name(); got != "SourceID" {
		t.Errorf("sourceIDBrand.Name() = %q, want %q", got, "SourceID")
	}
}

func TestValidatePresetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"simple", "home", false},
		{"with spaces", "living room", false},
		{"leading trailing whitespace trimmed", "  home  ", false},
		{"slash", "home/base", true},
		{"backslash", "home\\base", true},
		{"control char", "home\x00base", true},
		{"newline", "home\nbase", true},
		{"max length", strings.Repeat("a", MaxPresetNameLength), false},
		{"over max length", strings.Repeat("a", MaxPresetNameLength+1), true},
	}

	for _, tc := range tests {
		err := ValidatePresetName(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidatePresetName(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
		}

		if err != nil && !errors.Is(err, ErrInvalidPresetName) {
			t.Errorf("ValidatePresetName(%q) error does not wrap ErrInvalidPresetName: %v", tc.input, err)
		}
	}
}

func TestStateTrackModeValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		trackMode string
		wantValid bool
	}{
		{"", true}, // never set — valid, reads back as face
		{"none", true},
		{"face", true},
		{"halfbody", true},
		{"fullbody", true},
		{"bogus", false},
	} {
		state := DefaultState()
		state.TrackMode = tc.trackMode

		if got := state.Valid(); got != tc.wantValid {
			t.Errorf("State.Valid() with TrackMode %q = %v, want %v", tc.trackMode, got, tc.wantValid)
		}
	}
}

func TestStateEffectiveTrackMode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		trackMode string
		want      TargetTrackMode
	}{
		{"", TrackFace},
		{"none", TrackNone},
		{"face", TrackFace},
		{"halfbody", TrackHalfBody},
		{"fullbody", TrackFullBody},
		{"bogus", TrackFace}, // defensive: loadState rejects invalid, but reads degrade to the default
	} {
		state := State{TrackMode: tc.trackMode}

		if got := state.EffectiveTrackMode(); got != tc.want {
			t.Errorf("EffectiveTrackMode(%q) = %v, want %v", tc.trackMode, got, tc.want)
		}
	}
}

// TestTargetTrackModeWireValues pins the corrected mode-byte assignment:
// statically evidenced (Beta.25 x64 UI enum registration @0x14027c820) as
// 0=None, 1=Face, 2=HalfBody, 3=FullBody. The numeric pin fails loudly if
// anyone reverts to the disproved 0-based face-first mapping.
func TestTargetTrackModeWireValues(t *testing.T) {
	t.Parallel()

	if TrackNone != 0 || TrackFace != 1 || TrackHalfBody != 2 || TrackFullBody != 3 {
		t.Fatalf(
			"wire values drifted from the official 1-based enum: none=%d face=%d halfbody=%d fullbody=%d",
			TrackNone, TrackFace, TrackHalfBody, TrackFullBody,
		)
	}

	// Numeric aliases encoded the disproved 0-based mapping and were removed;
	// digits must not parse as variants.
	for _, input := range []string{"0", "1", "2", "3"} {
		if _, ok := ParseTargetTrackMode(input); ok {
			t.Errorf("ParseTargetTrackMode(%q) accepted a numeric alias", input)
		}
	}
}

// TestChargeStatusPredicate pins the official consumer-code predicate
// ({1,2} = charging, 0 = discharging; Beta.25 x64 @0x1403ccbf8).
func TestChargeStatusPredicate(t *testing.T) {
	t.Parallel()

	for sta, want := range map[ChargeStatus]bool{
		ChargeDischarging: false,
		ChargeCharging:    true,
		ChargeChargingAlt: true,
		ChargeStatus(3):   false,
		ChargeStatus(255): false,
	} {
		if got := sta.Charging(); got != want {
			t.Errorf("ChargeStatus(%d).Charging() = %v, want %v", sta, got, want)
		}
	}

	if ChargeStatus(3).Valid() {
		t.Error("ChargeStatus(3).Valid() = true, want false outside the official enum")
	}
}

func TestSpeedValuesSetGetZero(t *testing.T) {
	t.Parallel()

	var speeds SpeedValues
	if !speeds.IsZero() {
		t.Error("zero SpeedValues.IsZero() = false, want true")
	}

	speeds = speeds.Set(AxisPan, 40).Set(AxisTilt, 0).Set(AxisZoom, 2.5)
	if speeds.IsZero() {
		t.Error("configured SpeedValues.IsZero() = true, want false")
	}

	for _, tc := range []struct {
		axis Axis
		want float32
	}{
		{AxisPan, 40},
		{AxisTilt, 0},
		{AxisZoom, 2.5},
	} {
		got, ok := speeds.Get(tc.axis)
		if !ok || got != tc.want {
			t.Errorf("Get(%q) = %v, %v; want %v, true", tc.axis, got, ok, tc.want)
		}
	}

	if _, ok := speeds.Get("head"); ok {
		t.Error(`Get("head") = true, want false for unknown axis`)
	}
}
