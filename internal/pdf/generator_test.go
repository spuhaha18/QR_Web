package pdf

import (
	"bytes"
	"strings"
	"testing"

	"qrweb/internal/label"
)

func makeQRSet(t *testing.T, lbl label.Label) label.QRImageSet {
	t.Helper()
	imgs := make([][]byte, lbl.DocCount())
	for i := range imgs {
		imgs[i] = testQRPNG(t)
	}
	set, err := label.NewQRImageSet(imgs, lbl.DocCount())
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestCreateLabelPDFEquipment(t *testing.T) {
	lbl := label.EquipmentLabel{
		EqNumber: "EQ-1", EqDocNumber: "DOC-9", EqDocTitle: "제목",
		EqDocCount: 3, EqDocDepartment: "부서", EqDocYear: 2026,
	}
	b, _ := label.ParseBinderSize("3", label.DocTypeEquipment)
	data, filename, err := NewGenerator().CreateLabelPDF(label.DocTypeEquipment, b, lbl, makeQRSet(t, lbl))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("not a PDF")
	}
	if !strings.HasPrefix(filename, "DOC-9_") || !strings.HasSuffix(filename, ".pdf") {
		t.Errorf("filename = %q", filename)
	}
	// 3 labels of 23.8mm width fit one page.
	if n := bytes.Count(data, []byte("/Type /Page")); n < 1 {
		t.Errorf("page objects = %d", n)
	}
}

func TestCreateLabelPDFProjectHasAuxPieces(t *testing.T) {
	lbl := label.ProjectLabel{
		PjtNumber: "PJ-1", PjtTestNumber: "T-7", PjtDocTitle: "과제 제목",
		PjtDocWriter: "작성자", PjtDocCount: 2,
	}
	b, _ := label.ParseBinderSize("5", label.DocTypeProject)
	data, _, err := NewGenerator().CreateLabelPDF(label.DocTypeProject, b, lbl, makeQRSet(t, lbl))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty PDF")
	}
}
