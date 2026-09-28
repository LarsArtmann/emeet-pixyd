package pixy

import (
	"strings"
	"testing"
)

func TestModelFromProductID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		product int64
		want    Model
		wantOK  bool
	}{
		{"original", ProductIDOriginal, ModelOriginal, true},
		{"2K", ProductID2K, Model2K, true},
		{"fixed C960 is not a supported model", ProductIDC960, "", false},
		{"unknown", 0x1234, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ModelFromProductID(tc.product)
			if ok != tc.wantOK || got != tc.want {
				t.Errorf("ModelFromProductID(0x%04x) = (%q, %v), want (%q, %v)", tc.product, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestResolveProductID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		product  int64
		extra    []int64
		want     DeviceProfile
		wantKnown bool
	}{
		{
			name:     "static PIXY wins over extra list",
			product:  ProductIDOriginal,
			extra:    []int64{ProductIDOriginal},
			want:     DeviceProfile{Model: ModelOriginal, Family: FamilyPIXY},
			wantKnown: true,
		},
		{
			name:     "extra PID becomes PIXY-family",
			product:  0x0119,
			extra:    []int64{0x0119, 0x0120},
			want:     DeviceProfile{Model: "PIXY (PID 0x0119)", Family: FamilyPIXY},
			wantKnown: true,
		},
		{
			name:      "extra PID outranks fixed-UVC classification",
			product:   ProductIDC960,
			extra:     []int64{ProductIDC960},
			want:      DeviceProfile{Model: "PIXY (PID 0x003f)", Family: FamilyPIXY},
			wantKnown: true,
		},
		{
			name:      "fixed C960",
			product:   ProductIDC960,
			want:      DeviceProfile{Model: "EMEET C960", Family: FamilyFixedUVC},
			wantKnown: true,
		},
		{
			name:      "fixed C960 older enumeration",
			product:   ProductIDC960Old,
			want:      DeviceProfile{Model: "EMEET C960", Family: FamilyFixedUVC},
			wantKnown: true,
		},
		{
			name:      "fixed C960 2K",
			product:   ProductIDC9602K,
			want:      DeviceProfile{Model: "EMEET C960 2K", Family: FamilyFixedUVC},
			wantKnown: true,
		},
		{
			name:      "fixed C950",
			product:   ProductIDC950,
			want:      DeviceProfile{Model: "EMEET C950", Family: FamilyFixedUVC},
			wantKnown: true,
		},
		{
			name:      "fixed C970",
			product:   ProductIDC970,
			want:      DeviceProfile{Model: "EMEET C970", Family: FamilyFixedUVC},
			wantKnown: true,
		},
		{
			name:      "fixed S600L",
			product:   ProductIDS600L,
			want:      DeviceProfile{Model: "EMEET S600L", Family: FamilyFixedUVC},
			wantKnown: true,
		},
		{
			name:      "unknown PID",
			product:   0x0abc,
			wantKnown: false,
		},
		{
			name:      "unknown PID with unrelated extra list",
			product:   0x0abc,
			extra:     []int64{0x0119},
			wantKnown: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, known := ResolveProductID(tc.product, tc.extra)
			if known != tc.wantKnown {
				t.Fatalf("ResolveProductID(0x%04x) known = %v, want %v", tc.product, known, tc.wantKnown)
			}

			if known && got != tc.want {
				t.Errorf("ResolveProductID(0x%04x) = %+v, want %+v", tc.product, got, tc.want)
			}
		})
	}
}

func TestUnsupportedDeviceHint(t *testing.T) {
	t.Parallel()

	t.Run("known fixed model names the device and the reason", func(t *testing.T) {
		t.Parallel()

		hint := UnsupportedDeviceHint("EMEET C960", ProductIDC960)
		for _, want := range []string{"EMEET C960", "0x003f", "not controllable", "UVC", "PIXY"} {
			if !strings.Contains(hint, want) {
				t.Errorf("hint %q missing %q", hint, want)
			}
		}
	})

	t.Run("unknown vendor product asks for opt-in or report", func(t *testing.T) {
		t.Parallel()

		hint := UnsupportedDeviceHint("", 0x0abc)
		for _, want := range []string{"unknown EMEET device", "0x0abc", "EMEET_PIXYD_EXTRA_PRODUCT_IDS", "issue"} {
			if !strings.Contains(hint, want) {
				t.Errorf("hint %q missing %q", hint, want)
			}
		}
	})
}
