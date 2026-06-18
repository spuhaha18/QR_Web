// Package imaging holds image validation helpers. png.go replaces the Pillow
// verify()+format=='PNG' check from utils.validate_qr_image_bytes.
package imaging

import (
	"bytes"
	"image/png"
)

// pngSignature is the 8-byte PNG magic number. A valid PNG always starts with
// it; we reject early so JPEG/garbage/empty inputs never reach the decoder.
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// ValidatePNGBytes reports whether data is a structurally valid PNG image.
// It mirrors utils.validate_qr_image_bytes: empty -> false, non-PNG (e.g. JPEG)
// -> false, garbage -> false, truncated/corrupt PNG -> false. A full decode
// (image/png.Decode) is used so truncated PNGs that carry the signature but lack
// complete image data are rejected, matching Pillow's verify().
func ValidatePNGBytes(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !bytes.HasPrefix(data, pngSignature) {
		return false
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return false
	}
	return true
}
