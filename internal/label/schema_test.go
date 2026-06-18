package label

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

// validEquipment mirrors tests/test_document_schema.py VALID_EQUIPMENT.
func validEquipment() map[string]string {
	return map[string]string{
		"eq_number":        "EQ001",
		"eq_doc_number":    "DOC-001",
		"eq_doc_title":     "장비 유지 관리 절차서",
		"eq_doc_count":     "3",
		"eq_doc_department": "품질관리부",
		"eq_doc_year":      "2024",
	}
}

// validProject mirrors tests/test_document_schema.py VALID_PROJECT.
func validProject() map[string]string {
	return map[string]string{
		"pjt_number":      "PJT-001",
		"pjt_test_number": "TEST-001",
		"pjt_doc_title":   "시험 절차서",
		"pjt_doc_writer":  "홍길동",
		"pjt_doc_count":   "2",
	}
}

// ---- TestParseEquipment ----

func TestParseEquipment_ValidReturnsDataDocTypeBinder(t *testing.T) {
	lbl, docType, binder, err := ParseLabelRequest(validEquipment(), "1", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docType != "1" {
		t.Errorf("docType = %q, want 1", docType)
	}
	if binder != 3 {
		t.Errorf("binder = %d, want 3", binder)
	}
	eq, ok := lbl.(EquipmentLabel)
	if !ok {
		t.Fatalf("label type = %T, want EquipmentLabel", lbl)
	}
	if eq.EqNumber != "EQ001" {
		t.Errorf("EqNumber = %q, want EQ001", eq.EqNumber)
	}
	if eq.EqDocCount != 3 {
		t.Errorf("EqDocCount = %d, want 3 (int)", eq.EqDocCount)
	}
}

func TestParseEquipment_AllBinderSizesAccepted(t *testing.T) {
	for _, size := range []int{1, 3, 5, 7} {
		_, _, bs, err := ParseLabelRequest(validEquipment(), "1", strconv.Itoa(size))
		if err != nil {
			t.Fatalf("size %d: unexpected error: %v", size, err)
		}
		if bs != size {
			t.Errorf("binder = %d, want %d", bs, size)
		}
	}
}

func TestParseEquipment_MissingRequiredFieldRaises(t *testing.T) {
	for _, field := range EquipmentRequiredFields {
		bad := validEquipment()
		delete(bad, field)
		_, _, _, err := ParseLabelRequest(bad, "1", "3")
		if err == nil {
			t.Errorf("missing %q: expected error, got nil", field)
			continue
		}
		if !errors.Is(err, ErrValidation) {
			t.Errorf("missing %q: error %v not ErrValidation", field, err)
		}
	}
}

// ---- TestParseProject ----

func TestParseProject_ValidReturnsData(t *testing.T) {
	lbl, docType, _, err := ParseLabelRequest(validProject(), "2", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docType != "2" {
		t.Errorf("docType = %q, want 2", docType)
	}
	pj, ok := lbl.(ProjectLabel)
	if !ok {
		t.Fatalf("label type = %T, want ProjectLabel", lbl)
	}
	if pj.PjtDocCount != 2 {
		t.Errorf("PjtDocCount = %d, want 2", pj.PjtDocCount)
	}
}

func TestParseProject_Rejects1cmBinder(t *testing.T) {
	_, _, _, err := ParseLabelRequest(validProject(), "2", "1")
	if err == nil {
		t.Fatal("expected error rejecting 1cm binder, got nil")
	}
	if !strings.Contains(err.Error(), "3cm") {
		t.Errorf("error %q does not mention 3cm", err.Error())
	}
	want := "과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다."
	if ValidationMessage(err) != want {
		t.Errorf("message = %q, want %q", ValidationMessage(err), want)
	}
}

func TestParseProject_Accepts3cmAndAbove(t *testing.T) {
	for _, size := range []int{3, 5, 7} {
		if _, _, _, err := ParseLabelRequest(validProject(), "2", strconv.Itoa(size)); err != nil {
			t.Errorf("size %d: unexpected error: %v", size, err)
		}
	}
}

// ---- TestValidation ----

func TestParse_InvalidDocTypeRaises(t *testing.T) {
	_, _, _, err := ParseLabelRequest(validEquipment(), "3", "3")
	if err == nil || !strings.Contains(err.Error(), "문서 종류") {
		t.Fatalf("expected '문서 종류' error, got %v", err)
	}
}

func TestParse_InvalidBinderStringRaises(t *testing.T) {
	_, _, _, err := ParseLabelRequest(validEquipment(), "1", "bad")
	if err == nil || !strings.Contains(err.Error(), "바인더") {
		t.Fatalf("expected '바인더' error, got %v", err)
	}
}

func TestParse_InvalidBinderValueRaises(t *testing.T) {
	_, _, _, err := ParseLabelRequest(validEquipment(), "1", "2")
	if err == nil || !strings.Contains(err.Error(), "바인더") {
		t.Fatalf("expected '바인더' error, got %v", err)
	}
}

func TestParse_EmptyStringFieldRaises(t *testing.T) {
	bad := validEquipment()
	bad["eq_number"] = ""
	_, _, _, err := ParseLabelRequest(bad, "1", "3")
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for empty field, got %v", err)
	}
}

func TestParse_InvalidCountSilentlyDefaults(t *testing.T) {
	form := validEquipment()
	form["eq_doc_count"] = "abc"
	lbl, _, _, err := ParseLabelRequest(form, "1", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	eq := lbl.(EquipmentLabel)
	if eq.EqDocCount != 1 {
		t.Errorf("EqDocCount = %d, want 1 (silent default)", eq.EqDocCount)
	}
}

func TestParse_DefaultYearIsCurrentWhenMissing(t *testing.T) {
	// Python: safe_int_conversion(year, datetime.now().year). The field is
	// required, so an empty year is rejected; but a non-digit year defaults to
	// the current year rather than erroring.
	form := validEquipment()
	form["eq_doc_year"] = "notayear"
	lbl, _, _, err := ParseLabelRequest(form, "1", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	eq := lbl.(EquipmentLabel)
	if eq.EqDocYear != time.Now().Year() {
		t.Errorf("EqDocYear = %d, want %d", eq.EqDocYear, time.Now().Year())
	}
}

// ---- EquipmentLabel (test_label_schema.py) ----

func equipmentLabel() EquipmentLabel {
	return EquipmentLabel{
		EqNumber:        "EQ001",
		EqDocNumber:     "DOC-001",
		EqDocTitle:      "유지관리 절차서",
		EqDocCount:      3,
		EqDocDepartment: "품질부",
		EqDocYear:       2024,
	}
}

func projectLabel() ProjectLabel {
	return ProjectLabel{
		PjtNumber:     "PJT-001",
		PjtTestNumber: "TEST-001",
		PjtDocTitle:   "시험 절차서",
		PjtDocWriter:  "홍길동",
		PjtDocCount:   2,
	}
}

func TestEquipmentLabel_CellValuesHasAllCells(t *testing.T) {
	cells := equipmentLabel().CellValues()
	for _, addr := range []string{"B2", "B3", "B4", "B5", "B6", "B7"} {
		if _, ok := cells[addr]; !ok {
			t.Errorf("missing cell %s", addr)
		}
	}
}

func TestEquipmentLabel_CellB5HasCountString(t *testing.T) {
	if got := equipmentLabel().CellValues()["B5"]; got != "1/3" {
		t.Errorf("B5 = %v, want 1/3", got)
	}
}

func TestEquipmentLabel_B7IsIntNotString(t *testing.T) {
	got := equipmentLabel().CellValues()["B7"]
	if _, ok := got.(int); !ok {
		t.Errorf("B7 = %v (%T), want int", got, got)
	}
}

func TestEquipmentLabel_DocNumber(t *testing.T) {
	if got := equipmentLabel().DocNumber(); got != "DOC-001" {
		t.Errorf("DocNumber = %q, want DOC-001", got)
	}
}

func TestEquipmentLabel_DocCount(t *testing.T) {
	if got := equipmentLabel().DocCount(); got != 3 {
		t.Errorf("DocCount = %d, want 3", got)
	}
}

func TestEquipmentLabel_QRPayloadSheet1(t *testing.T) {
	got := equipmentLabel().QRPayload(1, 3)
	want := "EQ001|DOC-001|유지관리 절차서|품질부|2024|1/3"
	if got != want {
		t.Errorf("QRPayload = %q, want %q", got, want)
	}
}

func TestEquipmentLabel_QRPayloadSheet2(t *testing.T) {
	if got := equipmentLabel().QRPayload(2, 3); !strings.HasSuffix(got, "|2/3") {
		t.Errorf("QRPayload = %q, want suffix |2/3", got)
	}
}

func TestEquipmentLabel_TitleCellB4(t *testing.T) {
	if EquipmentTitleCell != "B4" || equipmentLabel().TitleCell() != "B4" {
		t.Errorf("TitleCell != B4")
	}
}

// ---- ProjectLabel ----

func TestProjectLabel_CellValuesHasSecondaryPanel(t *testing.T) {
	cells := projectLabel().CellValues()
	for _, addr := range []string{"Q21", "Q22", "R23", "S23"} {
		if _, ok := cells[addr]; !ok {
			t.Errorf("missing cell %s", addr)
		}
	}
}

func TestProjectLabel_Q21CombinesNumberAndTestNumber(t *testing.T) {
	if got := projectLabel().CellValues()["Q21"]; got != "[PJT-001] TEST-001" {
		t.Errorf("Q21 = %v, want [PJT-001] TEST-001", got)
	}
}

func TestProjectLabel_DocNumberIsTestNumber(t *testing.T) {
	if got := projectLabel().DocNumber(); got != "TEST-001" {
		t.Errorf("DocNumber = %q, want TEST-001", got)
	}
}

func TestProjectLabel_QRPayload(t *testing.T) {
	got := projectLabel().QRPayload(1, 2)
	want := "PJT-001|TEST-001|시험 절차서|홍길동|1/2"
	if got != want {
		t.Errorf("QRPayload = %q, want %q", got, want)
	}
}

func TestProjectLabel_PayloadHas5Fields(t *testing.T) {
	parts := strings.Split(projectLabel().QRPayload(1, 1), "|")
	if len(parts) != 5 {
		t.Errorf("payload field count = %d, want 5 (no year)", len(parts))
	}
}

func TestProjectLabel_S23EqualsB5Count(t *testing.T) {
	cells := projectLabel().CellValues()
	if cells["S23"] != "1/2" {
		t.Errorf("S23 = %v, want 1/2", cells["S23"])
	}
}

// ---- MakeLabel ----

func TestMakeLabel_Equipment(t *testing.T) {
	data := map[string]any{
		"eq_number": "EQ001", "eq_doc_number": "DOC-001", "eq_doc_title": "t",
		"eq_doc_count": 3, "eq_doc_department": "d", "eq_doc_year": 2024,
	}
	if _, ok := MakeLabel(data, "1").(EquipmentLabel); !ok {
		t.Error("MakeLabel('1') did not return EquipmentLabel")
	}
}

func TestMakeLabel_Project(t *testing.T) {
	data := map[string]any{
		"pjt_number": "PJT-001", "pjt_test_number": "TEST-001", "pjt_doc_title": "t",
		"pjt_doc_writer": "w", "pjt_doc_count": 2,
	}
	if _, ok := MakeLabel(data, "2").(ProjectLabel); !ok {
		t.Error("MakeLabel('2') did not return ProjectLabel")
	}
}

// ---- helpers (utils.py parity) ----

func TestSafeIntConversion(t *testing.T) {
	cases := []struct {
		in  string
		def int
		out int
	}{
		{"3", 1, 3},
		{"", 1, 1},      // empty -> default
		{"abc", 1, 1},   // non-digit -> default
		{"-5", 1, 1},    // sign -> not isdigit -> default
		{"2.5", 1, 1},   // dot -> not isdigit -> default
		{"0", 1, 1},     // isdigit, but max(1, 0) -> 1
		{"10", 1, 10},
		{"abc", 7, 7},   // non-digit -> custom default
	}
	for _, c := range cases {
		if got := SafeIntConversion(c.in, c.def); got != c.out {
			t.Errorf("SafeIntConversion(%q, %d) = %d, want %d", c.in, c.def, got, c.out)
		}
	}
}

func TestValidateAndCleanInput(t *testing.T) {
	cases := []struct{ in, out string }{
		{"  hello  ", "hello"},
		{"line1\nline2", "line1line2"},
		{"a\r\nb", "ab"},
		{"", ""},
		{"  품질부 \n", "품질부"},
	}
	for _, c := range cases {
		if got := ValidateAndCleanInput(c.in); got != c.out {
			t.Errorf("ValidateAndCleanInput(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func TestGenerateTimestampFilename(t *testing.T) {
	got := GenerateTimestampFilename("DOC-001", "xlsx")
	if !strings.HasPrefix(got, "DOC-001_") || !strings.HasSuffix(got, ".xlsx") {
		t.Errorf("filename = %q, want DOC-001_<ts>.xlsx", got)
	}
	// timestamp segment is 14 digits
	mid := strings.TrimSuffix(strings.TrimPrefix(got, "DOC-001_"), ".xlsx")
	if len(mid) != 14 {
		t.Errorf("timestamp segment = %q, want 14 digits", mid)
	}
	for _, r := range mid {
		if r < '0' || r > '9' {
			t.Errorf("timestamp segment %q not all digits", mid)
			break
		}
	}
}

// Compile-time assertions that both label types satisfy Label.
var _ Label = EquipmentLabel{}
var _ Label = ProjectLabel{}
