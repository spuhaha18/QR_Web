// Package qr generates QR codes for the auto (server-side) label mode.
// Ported from qr_generator.py. The payload is CP949-encoded before QR
// generation; Go's korean.EUCKR implements Windows-949 which is byte-identical
// to Python's CP949 for the inputs this app produces (see qr_test.go golden).
package qr

import (
	"encoding/base64"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/text/encoding/korean"
)

// pngSize is the rendered PNG edge length in pixels. The image is rescaled to
// 75x75 when embedded into Excel, so this only needs to be comfortably large;
// the module density is fixed by the payload, not by this size.
const pngSize = 256

// EncodeCP949 encodes s as CP949/Windows-949 (matching Python str.encode('CP949')).
func EncodeCP949(s string) ([]byte, error) {
	return korean.EUCKR.NewEncoder().Bytes([]byte(s))
}

// CreateQRPNG renders text as a PNG-encoded QR code using error correction
// level Low (== Python qrcode ERROR_CORRECT_L). The payload is CP949-encoded
// first, matching qr_generator.create_qr_image.
func CreateQRPNG(text string) ([]byte, error) {
	payload, err := EncodeCP949(text)
	if err != nil {
		return nil, err
	}
	q, err := qrcode.New(string(payload), qrcode.Low)
	if err != nil {
		return nil, err
	}
	return q.PNG(pngSize)
}

// CreateQRBase64 returns the standard-base64-encoded PNG of the QR code for
// text. Mirrors qr_generator.create_qr_base64.
func CreateQRBase64(text string) (string, error) {
	pngBytes, err := CreateQRPNG(text)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngBytes), nil
}
