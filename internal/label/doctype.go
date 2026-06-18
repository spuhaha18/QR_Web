package label

import "strings"

// DocType is the document kind: equipment (1) or project (2). It branches the
// required field set, the cell layout, and the QR box geometry.
//
// DocType replaces the bare "1"/"2" strings that previously threaded through
// the label, excel, and httpx layers — primitive obsession that scattered the
// same branch across ParseLabelRequest, the excel generator, and the geometry
// math, with no compiler check that a value was a real doc type. The branch now
// lives behind named methods on this type.
type DocType int

const (
	// DocTypeEquipment is doc_type "1": 장비 라벨.
	DocTypeEquipment DocType = 1
	// DocTypeProject is doc_type "2": 과제 라벨.
	DocTypeProject DocType = 2
)

// ParseDocType validates the raw form value and returns the typed DocType. It
// is the single string→DocType boundary; everything past it speaks DocType.
func ParseDocType(raw string) (DocType, error) {
	switch strings.TrimSpace(raw) {
	case "1":
		return DocTypeEquipment, nil
	case "2":
		return DocTypeProject, nil
	default:
		return 0, validationErr("잘못된 문서 종류입니다.")
	}
}

// Code returns the wire/string form ("1"/"2"), for the few places that still
// serialize the doc type (e.g. log lines, the Python-parity MakeLabel surface).
func (d DocType) Code() string {
	if d == DocTypeProject {
		return "2"
	}
	return "1"
}

// IsProject reports whether this is the project (과제) doc type.
func (d DocType) IsProject() bool { return d == DocTypeProject }

// RequiredFields returns the form fields that must be present for this doc type.
func (d DocType) RequiredFields() []string {
	if d == DocTypeProject {
		return ProjectRequiredFields
	}
	return EquipmentRequiredFields
}

// Layout returns the structural layout facts for this doc type. It is the
// single source of the doc-type geometry that the excel generator (sheet
// construction, i/N override, QR box borders) and the geometry math (QR anchor)
// both consume — previously these were duplicated as magic numbers (8 vs 7) and
// scattered cell lists across generator.go and geometry.go.
func (d DocType) Layout() Layout {
	if d == DocTypeProject {
		return Layout{
			QRBoxTopRow:    7,
			QRBoxBottomRow: 17,
			HasPrintArea:   true,
			CountCells:     []string{"B5", "S23"},
		}
	}
	return Layout{
		QRBoxTopRow:    8,
		QRBoxBottomRow: 17,
		HasPrintArea:   false,
		CountCells:     []string{"B5"},
	}
}

// Layout holds the doc-type-specific structural facts. The QR box columns (B–M)
// are common to both doc types and live as a constant in the excel package.
type Layout struct {
	// QRBoxTopRow is the top row of the lower QR box: 8 for equipment, 7 for
	// project.
	QRBoxTopRow int
	// QRBoxBottomRow is the bottom row of the lower QR box (17 for both); paired
	// with QRBoxTopRow it fully describes the box's vertical extent.
	QRBoxBottomRow int
	// HasPrintArea reports whether an A1:T24 print area is defined per sheet
	// (project only).
	HasPrintArea bool
	// CountCells are the cells that receive the "i/N" volume marker. B5 always;
	// project additionally mirrors it in S23.
	CountCells []string
}
