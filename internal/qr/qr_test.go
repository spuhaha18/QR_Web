package qr

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"testing"
)

// cp949Golden holds Python `s.encode('CP949')` output for representative
// strings, generated with:
//
//	.venv/bin/python -c "print(','.join('0x%02x'%b for b in '...'.encode('CP949')))"
//
// Go's golang.org/x/text/encoding/korean.EUCKR implements Windows-949, which is
// byte-identical to Python's CP949 (MS code page 949) for all strings tested
// here — including the rare MS-949 extension syllables 꽸/뷁 that are absent
// from pure EUC-KR. Full parity confirmed; no divergence to document.
var cp949Golden = []struct {
	in   string
	want []byte
}{
	{"품질관리부", []byte{0xc7, 0xb0, 0xc1, 0xfa, 0xb0, 0xfc, 0xb8, 0xae, 0xba, 0xce}},
	{"유지관리 절차서", []byte{0xc0, 0xaf, 0xc1, 0xf6, 0xb0, 0xfc, 0xb8, 0xae, 0x20, 0xc0, 0xfd, 0xc2, 0xf7, 0xbc, 0xad}},
	{"홍길동", []byte{0xc8, 0xab, 0xb1, 0xe6, 0xb5, 0xbf}},
	{"시험 절차서", []byte{0xbd, 0xc3, 0xc7, 0xe8, 0x20, 0xc0, 0xfd, 0xc2, 0xf7, 0xbc, 0xad}},
	{"EQ001|DOC-001|유지관리 절차서|품질부|2024|1/3", []byte{
		0x45, 0x51, 0x30, 0x30, 0x31, 0x7c, 0x44, 0x4f, 0x43, 0x2d, 0x30, 0x30, 0x31, 0x7c,
		0xc0, 0xaf, 0xc1, 0xf6, 0xb0, 0xfc, 0xb8, 0xae, 0x20, 0xc0, 0xfd, 0xc2, 0xf7, 0xbc, 0xad,
		0x7c, 0xc7, 0xb0, 0xc1, 0xfa, 0xba, 0xce, 0x7c, 0x32, 0x30, 0x32, 0x34, 0x7c, 0x31, 0x2f, 0x33,
	}},
	{"한글ABC123", []byte{0xc7, 0xd1, 0xb1, 0xdb, 0x41, 0x42, 0x43, 0x31, 0x32, 0x33}},
	// Rare MS-949 extension syllables (꽸, 뷁) plus middle-dot.
	{"각·꽸·뷁", []byte{0xb0, 0xa2, 0xa1, 0xa4, 0x84, 0xc3, 0xa1, 0xa4, 0x94, 0xee}},
}

func TestEncodeCP949_GoldenParity(t *testing.T) {
	for _, c := range cp949Golden {
		got, err := EncodeCP949(c.in)
		if err != nil {
			t.Errorf("EncodeCP949(%q) error: %v", c.in, err)
			continue
		}
		if !bytes.Equal(got, c.want) {
			t.Errorf("EncodeCP949(%q):\n got  %#v\n want %#v", c.in, got, c.want)
		}
	}
}

func mustText(t *testing.T, s string) QRText {
	t.Helper()
	qt, err := NewQRText(s)
	if err != nil {
		t.Fatalf("NewQRText(%q): %v", s, err)
	}
	return qt
}

func TestNewQRText(t *testing.T) {
	if _, err := NewQRText(""); err == nil {
		t.Error("empty text: expected error")
	} else if err.Error() != "QR 코드 텍스트가 제공되지 않았습니다." {
		t.Errorf("empty msg = %q", err.Error())
	}
	if _, err := NewQRText(string(make([]rune, 501))); err == nil {
		t.Error("501 runes: expected error")
	} else if err.Error() != "QR 코드 텍스트가 너무 깁니다 (최대 500자)." {
		t.Errorf("too-long msg = %q", err.Error())
	}
	if qt, err := NewQRText("MC-001"); err != nil || qt.String() != "MC-001" {
		t.Errorf("valid text: (%q, %v)", qt, err)
	}
}

func TestCreateQRPNG_DecodableAndDeterministic(t *testing.T) {
	const payload = "EQ001|DOC-001|유지관리 절차서|품질부|2024|1/3"
	a, err := CreateQRPNG(mustText(t, payload))
	if err != nil {
		t.Fatalf("CreateQRPNG error: %v", err)
	}
	if len(a) == 0 {
		t.Fatal("CreateQRPNG returned empty bytes")
	}
	img, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		t.Fatalf("output is not a valid PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != pngSize || b.Dy() != pngSize {
		t.Errorf("PNG size = %dx%d, want %dx%d", b.Dx(), b.Dy(), pngSize, pngSize)
	}
	// Same input -> same bytes (no randomness in encoding).
	b, err := CreateQRPNG(mustText(t, payload))
	if err != nil {
		t.Fatalf("CreateQRPNG (2nd) error: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Error("CreateQRPNG is not deterministic for identical input")
	}
}

func TestCreateQRBase64_DecodesToPNG(t *testing.T) {
	s, err := CreateQRBase64(mustText(t, "홍길동"))
	if err != nil {
		t.Fatalf("CreateQRBase64 error: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("base64 decode error: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(raw)); err != nil {
		t.Fatalf("decoded base64 is not a PNG: %v", err)
	}
}
