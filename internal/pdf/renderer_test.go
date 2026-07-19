package pdf

import (
	"bytes"
	"testing"

	"qrweb/internal/label"
	"qrweb/internal/qr"
)

func testQRPNG(t *testing.T) []byte {
	t.Helper()
	qt, err := qr.NewQRText("a|b|c")
	if err != nil {
		t.Fatal(err)
	}
	png, err := qr.CreateQRPNG(qt)
	if err != nil {
		t.Fatal(err)
	}
	return png
}

func TestRenderMainEquipmentSmoke(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	lbl := label.EquipmentLabel{
		EqNumber: "EQ-1", EqDocNumber: "DOC-1", EqDocTitle: "장비 문서 제목",
		EqDocCount: 2, EqDocDepartment: "부서", EqDocYear: 2026,
	}
	b, _ := label.ParseBinderSize("7", label.DocTypeEquipment)
	if err := renderMain(doc, 10, 10, label.DocTypeEquipment, b, lbl, "1/2", testQRPNG(t), "qr_0"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Error("not a PDF")
	}
}

func TestQRSideShrinksForNarrowBox(t *testing.T) {
	if got := qrSide(15.875, 23.8125); got != 15.875 {
		t.Errorf("qrSide narrow = %v, want 15.875", got)
	}
	if got := qrSide(22.225, 23.8125); got != qrSizeMM {
		t.Errorf("qrSide normal = %v, want %v", got, qrSizeMM)
	}
}

func TestRenderAuxProjectSmoke(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	lbl := label.ProjectLabel{
		PjtNumber: "PJ-1", PjtTestNumber: "T-1", PjtDocTitle: "과제 제목",
		PjtDocWriter: "작성자", PjtDocCount: 3,
	}
	renderAux(doc, 10, 10, lbl, "2/3")
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("empty PDF")
	}
}
