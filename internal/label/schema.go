// Package label owns the equipment/project label domain: field definitions,
// validation rules, request parsing, and the Label abstraction consumed by the
// excel generator. Ported from document_schema.py + utils.py.
package label

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrValidation is the sentinel wrapped by every validation failure. Callers
// use errors.Is(err, ErrValidation) to distinguish a 400-class request error
// from an internal 500. The wrapped message text is the Korean user-facing
// string, preserved byte-for-byte from the Flask original.
var ErrValidation = errors.New("validation error")

// validationErr builds a validation error whose Error() text is exactly the
// Korean message (matching Flask's ValidationError(str)) while still satisfying
// errors.Is(err, ErrValidation).
func validationErr(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

// ValidationMessage returns the bare Korean message for a validation error,
// stripping the sentinel prefix so handlers can emit the exact original string
// (e.g. jsonify({'error': str(e)})).
func ValidationMessage(err error) string {
	if err == nil {
		return ""
	}
	const prefix = "validation error: "
	s := err.Error()
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

var (
	validDocTypes    = map[string]bool{"1": true, "2": true}
	validBinderSizes = map[int]bool{1: true, 3: true, 5: true, 7: true}
)

// EquipmentRequiredFields / ProjectRequiredFields mirror the Python constants
// (exported so tests can iterate them like the pytest suite does).
var (
	EquipmentRequiredFields = []string{
		"eq_number", "eq_doc_number", "eq_doc_title",
		"eq_doc_count", "eq_doc_department", "eq_doc_year",
	}
	ProjectRequiredFields = []string{
		"pjt_number", "pjt_test_number", "pjt_doc_title",
		"pjt_doc_writer", "pjt_doc_count",
	}
)

// Label is the equipment/project common abstraction. The excel generator
// consumes only this interface.
type Label interface {
	// CellValues maps cell address -> value for Sheet 1 (B5 == "1/{count}").
	CellValues() map[string]any
	// QRPayload returns the pipe-delimited payload for sheet i of total (auto mode).
	QRPayload(i, total int) string
	// DocNumber is the filename base (equipment=eq_doc_number, project=pjt_test_number).
	DocNumber() string
	// DocCount is the number of sheets (eq_doc_count / pjt_doc_count).
	DocCount() int
	// TitleCell is the title cell address (both "B4"), the FONT_TITLE target.
	TitleCell() string
}

// EquipmentLabel owns the equipment label's field values, cell mapping, and QR
// payload. Ported from document_schema.EquipmentLabel.
type EquipmentLabel struct {
	EqNumber        string
	EqDocNumber     string
	EqDocTitle      string
	EqDocCount      int
	EqDocDepartment string
	EqDocYear       int
}

// EquipmentTitleCell mirrors EquipmentLabel.TITLE_CELL ('B4').
const EquipmentTitleCell = "B4"

func (l EquipmentLabel) CellValues() map[string]any {
	return map[string]any{
		"B2": l.EqNumber,
		"B3": l.EqDocNumber,
		"B4": l.EqDocTitle,
		"B5": fmt.Sprintf("1/%d", l.EqDocCount),
		"B6": l.EqDocDepartment,
		"B7": l.EqDocYear, // int cell (number format, not text)
	}
}

func (l EquipmentLabel) QRPayload(i, total int) string {
	return strings.Join([]string{
		l.EqNumber,
		l.EqDocNumber,
		l.EqDocTitle,
		l.EqDocDepartment,
		strconv.Itoa(l.EqDocYear),
		fmt.Sprintf("%d/%d", i, total),
	}, "|")
}

func (l EquipmentLabel) DocNumber() string { return l.EqDocNumber }
func (l EquipmentLabel) DocCount() int     { return l.EqDocCount }
func (l EquipmentLabel) TitleCell() string { return EquipmentTitleCell }

// ProjectLabel owns the project label's field values, cell mapping, and QR
// payload. Ported from document_schema.ProjectLabel.
type ProjectLabel struct {
	PjtNumber     string
	PjtTestNumber string
	PjtDocTitle   string
	PjtDocWriter  string
	PjtDocCount   int
}

// ProjectTitleCell mirrors ProjectLabel.TITLE_CELL ('B4').
const ProjectTitleCell = "B4"

func (l ProjectLabel) CellValues() map[string]any {
	countStr := fmt.Sprintf("1/%d", l.PjtDocCount)
	return map[string]any{
		"B2":  l.PjtNumber,
		"B3":  l.PjtTestNumber,
		"B4":  l.PjtDocTitle,
		"B5":  countStr,
		"B6":  l.PjtDocWriter,
		"Q21": fmt.Sprintf("[%s] %s", l.PjtNumber, l.PjtTestNumber),
		"Q22": l.PjtDocTitle,
		"R23": l.PjtDocWriter,
		"S23": countStr,
	}
}

func (l ProjectLabel) QRPayload(i, total int) string {
	return strings.Join([]string{
		l.PjtNumber,
		l.PjtTestNumber,
		l.PjtDocTitle,
		l.PjtDocWriter,
		fmt.Sprintf("%d/%d", i, total),
	}, "|")
}

func (l ProjectLabel) DocNumber() string { return l.PjtTestNumber }
func (l ProjectLabel) DocCount() int     { return l.PjtDocCount }
func (l ProjectLabel) TitleCell() string { return ProjectTitleCell }

// ParseLabelRequest validates and parses a label creation request from a flat
// form map. Returns the concrete Label, the doc type ("1"/"2"), the binder
// size, or a validation error wrapping ErrValidation. Ported from
// document_schema.parse_label_request + make_label.
func ParseLabelRequest(form map[string]string, docType, binderSizeRaw string) (Label, string, int, error) {
	if !validDocTypes[docType] {
		return nil, "", 0, validationErr("잘못된 문서 종류입니다.")
	}

	binderSize, err := strconv.Atoi(strings.TrimSpace(binderSizeRaw))
	if err != nil {
		return nil, "", 0, validationErr("잘못된 바인더 크기입니다.")
	}
	if !validBinderSizes[binderSize] {
		return nil, "", 0, validationErr("잘못된 바인더 크기입니다.")
	}

	if docType == "2" && binderSize == 1 {
		return nil, "", 0, validationErr("과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.")
	}

	required := EquipmentRequiredFields
	if docType == "2" {
		required = ProjectRequiredFields
	}
	for _, field := range required {
		// Python: `if not form_data.get(field)` — empty string is falsy.
		if form[field] == "" {
			return nil, "", 0, validationErr(fmt.Sprintf("필수 필드가 누락되었습니다: %s", field))
		}
	}

	if docType == "1" {
		lbl := EquipmentLabel{
			EqNumber:        ValidateAndCleanInput(form["eq_number"]),
			EqDocNumber:     ValidateAndCleanInput(form["eq_doc_number"]),
			EqDocTitle:      ValidateAndCleanInput(form["eq_doc_title"]),
			EqDocCount:      SafeIntConversion(form["eq_doc_count"], 1),
			EqDocDepartment: ValidateAndCleanInput(form["eq_doc_department"]),
			EqDocYear:       SafeIntConversion(form["eq_doc_year"], time.Now().Year()),
		}
		return lbl, docType, binderSize, nil
	}

	lbl := ProjectLabel{
		PjtNumber:     ValidateAndCleanInput(form["pjt_number"]),
		PjtTestNumber: ValidateAndCleanInput(form["pjt_test_number"]),
		PjtDocTitle:   ValidateAndCleanInput(form["pjt_doc_title"]),
		PjtDocWriter:  ValidateAndCleanInput(form["pjt_doc_writer"]),
		PjtDocCount:   SafeIntConversion(form["pjt_doc_count"], 1),
	}
	return lbl, docType, binderSize, nil
}

// ValidateAndCleanInput strips surrounding whitespace and removes \n / \r.
// Ported from utils.validate_and_clean_input (default="").
func ValidateAndCleanInput(value string) string {
	if value == "" {
		return ""
	}
	s := strings.TrimSpace(value)
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// SafeIntConversion mirrors utils.safe_int_conversion: only Python str.isdigit()
// strings convert, otherwise the default is returned; the result is clamped to a
// minimum of 1. Negative numbers, decimals, and non-numeric strings all yield
// the default. (Python isdigit() is true only for all-ASCII-digit, non-empty
// strings here — no sign, no dot.)
func SafeIntConversion(value string, def int) int {
	if value == "" {
		return def
	}
	if !isPyDigit(value) {
		return def
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	if result < 1 {
		return 1
	}
	return result
}

// isPyDigit replicates Python str.isdigit() for the inputs that matter here:
// non-empty and every rune an ASCII digit 0-9. (Python isdigit also accepts
// superscripts etc., but form numeric inputs are plain ASCII; matching ASCII
// digits preserves the int()-convertible subset the original relies on.)
func isPyDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// GenerateTimestampFilename builds "{base}_{YYYYMMDDhhmmss}.{ext}".
// Ported from utils.generate_timestamp_filename (ext default "xlsx").
func GenerateTimestampFilename(base, ext string) string {
	if ext == "" {
		ext = "xlsx"
	}
	ts := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s_%s.%s", base, ts, ext)
}

// MakeLabel is a factory kept for parity with document_schema.make_label; it
// constructs the appropriate Label from a parsed value map. ParseLabelRequest is
// the primary entry point, but this mirrors the Python test surface.
func MakeLabel(data map[string]any, docType string) Label {
	if docType == "1" {
		return EquipmentLabel{
			EqNumber:        asString(data["eq_number"]),
			EqDocNumber:     asString(data["eq_doc_number"]),
			EqDocTitle:      asString(data["eq_doc_title"]),
			EqDocCount:      asInt(data["eq_doc_count"]),
			EqDocDepartment: asString(data["eq_doc_department"]),
			EqDocYear:       asInt(data["eq_doc_year"]),
		}
	}
	return ProjectLabel{
		PjtNumber:     asString(data["pjt_number"]),
		PjtTestNumber: asString(data["pjt_test_number"]),
		PjtDocTitle:   asString(data["pjt_doc_title"]),
		PjtDocWriter:  asString(data["pjt_doc_writer"]),
		PjtDocCount:   asInt(data["pjt_doc_count"]),
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt(v any) int {
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}
