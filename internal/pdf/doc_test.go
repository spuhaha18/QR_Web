package pdf

import (
	"bytes"
	"testing"
)

// newDoc must zero fpdf's default 1mm cell margin: drawTextBox positions runs
// at measured x, and any cell margin shifts the rendered text right of the
// measurement — crossing the border on narrow (1cm binder) labels.
func TestNewDocZeroCellMargin(t *testing.T) {
	if m := newDoc().GetCellMargin(); m != 0 {
		t.Errorf("cell margin = %v, want 0", m)
	}
}

func TestNewDocRendersKoreanAndLatin(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	doc.SetFont(fontFamily, "B", 12)
	doc.SetXY(10, 10)
	doc.CellFormat(100, 10, "바탕체 Times 123", "", 0, "C", false, 0, "")
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Errorf("not a PDF: %x", buf.Bytes()[:8])
	}
}
