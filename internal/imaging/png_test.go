package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

// makePNGBytes mirrors tests/conftest.make_png_bytes (50x50 white PNG).
func makePNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// makeJPEGBytes mirrors tests/conftest.make_jpeg_bytes (50x50 gray JPEG).
func makeJPEGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{200, 200, 200, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func TestValidatePNGBytes_ValidPNGReturnsTrue(t *testing.T) {
	if !ValidatePNGBytes(makePNGBytes(t)) {
		t.Error("ValidatePNGBytes(valid png) = false, want true")
	}
}

func TestValidatePNGBytes_JPEGReturnsFalse(t *testing.T) {
	if ValidatePNGBytes(makeJPEGBytes(t)) {
		t.Error("ValidatePNGBytes(jpeg) = true, want false")
	}
}

func TestValidatePNGBytes_GarbageReturnsFalse(t *testing.T) {
	if ValidatePNGBytes([]byte("not an image at all")) {
		t.Error("ValidatePNGBytes(garbage) = true, want false")
	}
}

func TestValidatePNGBytes_EmptyReturnsFalse(t *testing.T) {
	if ValidatePNGBytes([]byte{}) {
		t.Error("ValidatePNGBytes(empty) = true, want false")
	}
	if ValidatePNGBytes(nil) {
		t.Error("ValidatePNGBytes(nil) = true, want false")
	}
}

func TestValidatePNGBytes_TruncatedReturnsFalse(t *testing.T) {
	full := makePNGBytes(t)
	if ValidatePNGBytes(full[:20]) {
		t.Error("ValidatePNGBytes(truncated png) = true, want false")
	}
}

// A real user-uploaded QR PNG with a corrupt IDAT CRC. Pillow's verify()
// rejects it ("bad header checksum in IDAT") and so must we — both Python and
// Go reject the same bytes. Guards the chunk-CRC parity with the legacy app.
func TestValidatePNGBytes_CorruptIDATCRCReturnsFalse(t *testing.T) {
	data, err := os.ReadFile("testdata/broken_idat_crc.png")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if ValidatePNGBytes(data) {
		t.Error("ValidatePNGBytes(corrupt-IDAT-CRC png) = true, want false (Pillow rejects it)")
	}
}

// The same image re-encoded to a structurally valid PNG. Pillow accepts it and
// so must we. Together with the corrupt fixture this pins the validator to
// Pillow.verify() semantics (chunk-CRC check, NOT a full pixel decode), so that
// PNGs carrying extra IDAT pixel data — which Pillow accepts but image/png's
// strict Decode rejects as "too much pixel data" — are not wrongly refused.
func TestValidatePNGBytes_ValidReencodedReturnsTrue(t *testing.T) {
	data, err := os.ReadFile("testdata/valid_reencoded.png")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if !ValidatePNGBytes(data) {
		t.Error("ValidatePNGBytes(valid re-encoded png) = false, want true")
	}
}
