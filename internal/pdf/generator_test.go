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
	// 2 mains (30.16mm) + 2 aux (96.31mm) on a 5cm binder all fit page 1:
	// exactly one /Type /Pages node plus one /Type /Page node. "/Type /Page"
	// is a prefix of "/Type /Pages", so each Pages node also contributes one
	// match — total 2 for a single-page document.
	if n := bytes.Count(data, []byte("/Type /Page")); n != 2 {
		t.Errorf("page objects = %d, want 2 (1 Pages + 1 Page — single page)", n)
	}

	// Pin the aux-pieces-present composition at the layout level: 2 mains +
	// 2 aux pieces, all placed on page 0.
	mw, mh := mainSize(b)
	aw, ah := auxSize()
	sizes := [][2]float64{{mw, mh}, {mw, mh}, {aw, ah}, {aw, ah}}
	placed := layoutPieces(sizes)
	if len(placed) != 4 {
		t.Fatalf("layoutPieces() = %d pieces, want 4", len(placed))
	}
	for i, p := range placed {
		if p.page != 0 {
			t.Errorf("piece %d on page %d, want page 0", i, p.page)
		}
	}
}
