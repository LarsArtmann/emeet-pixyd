package pixy

// Model identifies a supported EMEET PIXY hardware variant.
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

// ModelFromProductID maps a parsed USB product ID to its Model. The second
// return value reports whether the product ID belongs to a supported model.
func ModelFromProductID(product int64) (Model, bool) {
	switch product {
	case ProductIDOriginal:
		return ModelOriginal, true
	case ProductID2K:
		return Model2K, true
	default:
		return "", false
	}
}
