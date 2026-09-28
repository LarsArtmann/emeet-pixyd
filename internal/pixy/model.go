package pixy

import "fmt"

// Model identifies an EMEET device known to emeet-pixyd.
type Model string

const (
	// ModelOriginal is the original EMEET PIXY (USB product ID 0x00c0).
	ModelOriginal Model = "PIXY"

	// Model2K is the PIXY 2K variant (USB product ID 0x0118, issue #6).
	Model2K Model = "PIXY 2K"
)

// USB product IDs of the supported PIXY models. Both expose the same HID
// control interface and V4L2 controls.
const (
	ProductIDOriginal = 0x00c0
	ProductID2K       = 0x0118
)

// USB product IDs of recognized fixed-lens EMEET webcams. Sourced from
// public Linux hardware probes (linux-hardware.org device pages, raw-fetched
// 2026-09-28) and the rakhbari/emeet-control README (S600L).
const (
	ProductIDC960    = 0x003f // HD Webcam eMeet C960 (31 probes)
	ProductIDC960Old = 0x2013 // HD Webcam eMeet C960, older enumeration (33 probes)
	ProductIDC9602K  = 0x007c // EMEET SmartCam C960 2K (5 probes)
	ProductIDC950    = 0x0073 // HD Webcam eMeet C950 (23 probes, enumerates as "USB 2.0 Camera")
	ProductIDC970    = 0x002d // HD Webcam C970 (3 probes)
	ProductIDS600L   = 0x00ef // EMEET SmartCam S600L (rakhbari/emeet-control)
)

// Family classifies what emeet-pixyd can do with a device line.
type Family string

const (
	// FamilyPIXY devices speak the reverse-engineered PIXY HID protocol:
	// tracking, privacy, audio, gesture, PTZ, presets, battery. These get
	// full daemon support.
	FamilyPIXY Family = "pixy"

	// FamilyFixedUVC devices are fixed-lens EMEET webcams driven entirely by
	// the kernel's standard UVC driver. They have no vendor control surface
	// the daemon could automate, so emeet-pixyd recognizes them (for clear
	// user messaging) but never sends them vendor HID bytes.
	FamilyFixedUVC Family = "fixed-uvc"
)

// DeviceProfile is everything the daemon knows about an EMEET product.
type DeviceProfile struct {
	Model  Model
	Family Family
}

// pixyFamilyProfiles are the controllable PIXY-family products.
//
//nolint:gochecknoglobals // static device registry
var pixyFamilyProfiles = map[int64]DeviceProfile{
	ProductIDOriginal: {Model: ModelOriginal, Family: FamilyPIXY},
	ProductID2K:       {Model: Model2K, Family: FamilyPIXY},
}

// fixedUVCProfiles are recognized fixed-lens EMEET webcams.
//
//nolint:gochecknoglobals // static device registry
var fixedUVCProfiles = map[int64]DeviceProfile{
	ProductIDC960:    {Model: "EMEET C960", Family: FamilyFixedUVC},
	ProductIDC960Old: {Model: "EMEET C960", Family: FamilyFixedUVC},
	ProductIDC9602K:  {Model: "EMEET C960 2K", Family: FamilyFixedUVC},
	ProductIDC950:    {Model: "EMEET C950", Family: FamilyFixedUVC},
	ProductIDC970:    {Model: "EMEET C970", Family: FamilyFixedUVC},
	ProductIDS600L:   {Model: "EMEET S600L", Family: FamilyFixedUVC},
}

// ModelFromProductID maps a parsed USB product ID to its Model. The second
// return value reports whether the product ID belongs to a supported
// (PIXY-family) model.
func ModelFromProductID(product int64) (Model, bool) {
	profile, ok := pixyFamilyProfiles[product]
	if !ok {
		return "", false
	}

	return profile.Model, true
}

// ResolveProductID classifies a USB product ID. PIXY-family products win,
// then user-configured extra IDs (EMEET_PIXYD_EXTRA_PRODUCT_IDS, treated as
// PIXY-family variants — an explicit user opt-in outranks the fixed-UVC
// classification), then the fixed-UVC registry. The second return value
// reports whether the product ID is known at all.
func ResolveProductID(product int64, extraProductIDs []int64) (DeviceProfile, bool) {
	if profile, ok := pixyFamilyProfiles[product]; ok {
		return profile, true
	}

	for _, extra := range extraProductIDs {
		if extra == product {
			return DeviceProfile{Model: ExtraModelName(product), Family: FamilyPIXY}, true
		}
	}

	if profile, ok := fixedUVCProfiles[product]; ok {
		return profile, true
	}

	var zero DeviceProfile

	return zero, false
}

// ExtraModelName is the display model for a user-configured extra product
// ID. It keeps the custom PID visible in logs and the UI so misclassified
// devices are easy to spot.
func ExtraModelName(product int64) Model {
	return Model(fmt.Sprintf("PIXY (PID 0x%04x)", product))
}

// UnsupportedDeviceHint explains an EMEET device the daemon found but cannot
// control. An empty model means the EMEET vendor was recognized but the
// product ID is unknown; the hint then offers the explicit opt-in (PIXY-family
// variants) or asks for a report so the device can be classified.
func UnsupportedDeviceHint(model string, product int64) string {
	if model == "" {
		return fmt.Sprintf(
			"unknown EMEET device (USB product ID 0x%04x) is not supported yet: "+
				"if it is a PIXY-family variant, set EMEET_PIXYD_EXTRA_PRODUCT_IDS=0x%04x to control it as a PIXY, "+
				"otherwise open an issue with this ID so the device can be classified",
			product, product,
		)
	}

	return fmt.Sprintf(
		"%s (USB product ID 0x%04x) is recognized but not controllable: fixed EMEET webcams "+
			"are fully driven by the standard Linux UVC driver, and emeet-pixyd automates only "+
			"the PIXY family's vendor features (tracking, privacy shutter, audio)",
		model, product,
	)
}
