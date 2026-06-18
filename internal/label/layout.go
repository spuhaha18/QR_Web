package label

import (
	"strconv"
	"strings"
)

// BinderSize is a validated binder thickness in cm. Only 1, 3, 5, 7 are valid,
// and project documents may not use 1cm. A BinderSize value cannot exist unless
// it is one of the allowed sizes for its doc type.
//
// BinderSize owns the validation that was previously split between
// ParseLabelRequest (which rejected unknown sizes) and the old GetQRConfig
// (which silently fell back to 3cm) — two sources of truth for one domain
// constraint. The fallback is gone: an invalid binder is rejected at the
// parse boundary, everywhere.
type BinderSize int

// binderColumnWidth maps binder size (cm) → column width (char units) applied to
// columns B–M at QR embed time. Mirrors _BINDER_QR_CONFIG in label_layout.py.
var binderColumnWidth = map[BinderSize]float64{
	7: 1.875,
	5: 1.25,
	3: 1.0,
	1: 0.75,
}

// ParseBinderSize validates the raw form value against the doc type. It is the
// single string→BinderSize boundary. Returns a validation error for a
// non-numeric value, an unknown size, or the project+1cm combination.
func ParseBinderSize(raw string, dt DocType) (BinderSize, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, validationErr("잘못된 바인더 크기입니다.")
	}
	b := BinderSize(n)
	if _, ok := binderColumnWidth[b]; !ok {
		return 0, validationErr("잘못된 바인더 크기입니다.")
	}
	if dt.IsProject() && b == 1 {
		return 0, validationErr("과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.")
	}
	return b, nil
}

// ColumnWidth returns the B–M column width (char units) for this binder size.
// A validated BinderSize is always present in the table.
func (b BinderSize) ColumnWidth() float64 {
	return binderColumnWidth[b]
}

// Int returns the numeric cm value.
func (b BinderSize) Int() int { return int(b) }
