package label

import (
	"fmt"
	"sort"
)

// QRUpload is one raw uploaded QR image with its original filename (used only
// for the per-file error message). The httpx layer reads these from the
// multipart form; the intake rules in BuildQRImageSet are pure domain.
type QRUpload struct {
	Name  string
	Bytes []byte
}

// QRIntakeLimits are the runtime bounds on a paste-mode intake. They are not
// domain invariants — they come from config — so the caller injects them rather
// than the domain hardcoding them.
type QRIntakeLimits struct {
	MaxFiles    int
	MaxFileSize int64
}

// ValidatePNG is the per-image format check, injected so the label package need
// not depend on the imaging layer. Returns true for an acceptable PNG.
type ValidatePNG func(b []byte) bool

// BuildQRImageSet runs the full paste-mode QR image intake: it validates the
// uploads against the 권 = QR 1:1 count invariant and the runtime limits, then
// reorders by the client permutation and returns the QRImageSet that the excel
// generator consumes. This is the cohesive intake module — the validation
// sequence and the reorder used to be a procedural if-chain inside the HTTP
// handler, untestable except through the full transport stack.
//
// order[sheetIdx] = source index into uploads. docCount = lbl.DocCount(). Every
// returned error already carries the exact user-facing Korean message and
// matches ErrValidation; the caller maps that to a 400. Validation order is
// preserved from the original handler so a given request fails the same way.
func BuildQRImageSet(uploads []QRUpload, order []int, docCount int, limits QRIntakeLimits, validPNG ValidatePNG) (QRImageSet, error) {
	if len(uploads) != docCount {
		return QRImageSet{}, validationErr(fmt.Sprintf("QR 이미지 수가 권수와 다릅니다 (받음: %d, 권수: %d)", len(uploads), docCount))
	}
	if len(uploads) > limits.MaxFiles {
		return QRImageSet{}, validationErr(fmt.Sprintf("QR 이미지는 최대 %d개까지 허용됩니다.", limits.MaxFiles))
	}
	if len(order) != docCount {
		return QRImageSet{}, validationErr("qr_order 길이가 권수와 다릅니다.")
	}
	if !isPermutation(order, docCount) {
		return QRImageSet{}, validationErr("qr_order에 중복이나 범위 초과 인덱스가 있습니다.")
	}
	for _, u := range uploads {
		if int64(len(u.Bytes)) > limits.MaxFileSize {
			return QRImageSet{}, validationErr(fmt.Sprintf("QR 이미지 크기가 2MB를 초과합니다: %s", u.Name))
		}
		if !validPNG(u.Bytes) {
			return QRImageSet{}, validationErr(fmt.Sprintf("유효하지 않은 PNG 이미지입니다: %s", u.Name))
		}
	}

	ordered := make([][]byte, docCount)
	for sheetIdx, srcIdx := range order {
		ordered[sheetIdx] = uploads[srcIdx].Bytes
	}
	return QRImageSet{images: ordered}, nil
}

// QRRenderer renders a label's QR payload string to PNG bytes. It is injected
// so the label domain need not depend on the qr package; the httpx layer wires
// in qr.NewQRText + qr.CreateQRPNG.
type QRRenderer func(payload string) ([]byte, error)

// BuildAutoQRImageSet generates the auto-mode QR image set: one QR per sheet,
// rendered from the label's per-sheet payload (1-based sheet index). It mirrors
// BuildQRImageSet (the paste path) so both modes funnel into a QRImageSet, and
// it lifts the per-sheet generation loop out of the HTTP handler into testable
// domain code (inject a fake renderer). A render error propagates unchanged so
// the caller can map it (e.g. a too-long payload from qr.NewQRText → 400).
func BuildAutoQRImageSet(lbl Label, render QRRenderer) (QRImageSet, error) {
	total := lbl.DocCount()
	images := make([][]byte, total)
	for i := 0; i < total; i++ {
		png, err := render(lbl.QRPayload(i+1, total))
		if err != nil {
			return QRImageSet{}, err
		}
		images[i] = png
	}
	return NewQRImageSet(images, total)
}

// isPermutation reports whether order is exactly a permutation of [0, n)
// (sorted(order) == range(n)): correct length, in range, no duplicates.
func isPermutation(order []int, n int) bool {
	if len(order) != n {
		return false
	}
	cp := append([]int(nil), order...)
	sort.Ints(cp)
	for i := 0; i < n; i++ {
		if cp[i] != i {
			return false
		}
	}
	return true
}
