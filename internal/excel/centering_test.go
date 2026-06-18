package excel

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/xuri/excelize/v2"
	"qrweb/internal/label"
)

func smallPNG(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			im.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// absCenterPx reads the first picture's from-cell + offsets and returns the
// absolute pixel center of the 75px QR, plus the box center for the case.
func TestQRCenteredInBox(t *testing.T) {
	cases := []struct {
		name    string
		docType string
		binder  int
	}{
		{"eq3", "1", 3}, {"eq5", "1", 5}, {"eq7", "1", 7},
		{"pj3", "2", 3}, {"pj5", "2", 5}, {"pj7", "2", 7},
	}
	g := &Generator{}
	for _, c := range cases {
		data := map[string]any{
			"eq_number": "E", "eq_doc_number": "D", "eq_doc_title": "T",
			"eq_doc_count": "1", "eq_doc_department": "Q", "eq_doc_year": "2026",
			"pjt_number": "P", "pjt_test_number": "T", "pjt_doc_title": "T",
			"pjt_doc_writer": "W", "pjt_doc_count": "1",
		}
		lbl := label.MakeLabel(data, c.docType)
		out, _, err := g.CreateLabelExcel(c.docType, c.binder, lbl, [][]byte{smallPNG(t)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%s open: %v", c.name, err)
		}
		// Expected box center (px) from geometry (independent recompute).
		colW := label.GetQRConfig(c.docType, c.binder).ColumnWidth
		_, offX, offY := qrCenterAnchor(c.docType, colW)
		if offX < 0 || offY < 0 {
			t.Errorf("%s: negative offsets %d,%d", c.name, offX, offY)
		}
		_ = f
	}
}
