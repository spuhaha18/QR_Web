package label

import (
	"bytes"
	"errors"
	"testing"
)

// alwaysPNG / neverPNG are ValidatePNG stubs; BuildQRImageSet is pure domain and
// needs no real image bytes — the format check is injected.
func alwaysPNG([]byte) bool { return true }
func neverPNG([]byte) bool  { return false }

func uploads(names ...string) []QRUpload {
	out := make([]QRUpload, len(names))
	for i, n := range names {
		out[i] = QRUpload{Name: n, Bytes: []byte(n)}
	}
	return out
}

var noLimits = QRIntakeLimits{MaxFiles: 50, MaxFileSize: 2 * 1024 * 1024}

func TestBuildQRImageSet_ReordersByPermutation(t *testing.T) {
	set, err := BuildQRImageSet(uploads("a", "b", "c"), []int{2, 0, 1}, 3, noLimits, alwaysPNG)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	imgs := set.Images()
	want := []string{"c", "a", "b"} // order[sheetIdx] = source index
	for i, w := range want {
		if !bytes.Equal(imgs[i], []byte(w)) {
			t.Errorf("sheet %d = %q, want %q", i, imgs[i], w)
		}
	}
}

func TestBuildQRImageSet_CountMismatch(t *testing.T) {
	_, err := BuildQRImageSet(uploads("a"), []int{0, 1}, 2, noLimits, alwaysPNG)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
	if ValidationMessage(err) != "QR 이미지 수가 권수와 다릅니다 (받음: 1, 권수: 2)" {
		t.Errorf("msg = %q", ValidationMessage(err))
	}
}

func TestBuildQRImageSet_MaxFiles(t *testing.T) {
	_, err := BuildQRImageSet(uploads("a", "b"), []int{0, 1}, 2, QRIntakeLimits{MaxFiles: 1, MaxFileSize: 1 << 20}, alwaysPNG)
	if ValidationMessage(err) != "QR 이미지는 최대 1개까지 허용됩니다." {
		t.Errorf("msg = %q", ValidationMessage(err))
	}
}

func TestBuildQRImageSet_OrderLengthMismatch(t *testing.T) {
	_, err := BuildQRImageSet(uploads("a", "b"), []int{0}, 2, noLimits, alwaysPNG)
	if ValidationMessage(err) != "qr_order 길이가 권수와 다릅니다." {
		t.Errorf("msg = %q", ValidationMessage(err))
	}
}

func TestBuildQRImageSet_OrderNotPermutation(t *testing.T) {
	_, err := BuildQRImageSet(uploads("a", "b"), []int{0, 0}, 2, noLimits, alwaysPNG)
	if ValidationMessage(err) != "qr_order에 중복이나 범위 초과 인덱스가 있습니다." {
		t.Errorf("msg = %q", ValidationMessage(err))
	}
}

func TestBuildQRImageSet_FileTooLarge(t *testing.T) {
	_, err := BuildQRImageSet(uploads("big.png"), []int{0}, 1, QRIntakeLimits{MaxFiles: 50, MaxFileSize: 1}, alwaysPNG)
	if ValidationMessage(err) != "QR 이미지 크기가 2MB를 초과합니다: big.png" {
		t.Errorf("msg = %q", ValidationMessage(err))
	}
}

func TestBuildQRImageSet_InvalidPNG(t *testing.T) {
	_, err := BuildQRImageSet(uploads("bad.png"), []int{0}, 1, noLimits, neverPNG)
	if ValidationMessage(err) != "유효하지 않은 PNG 이미지입니다: bad.png" {
		t.Errorf("msg = %q", ValidationMessage(err))
	}
}
