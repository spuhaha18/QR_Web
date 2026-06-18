package excel

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"regexp"
	"strconv"
	"testing"

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

// drawingFromAnchor parses the first <xdr:from> element in drawing1.xml of the
// given xlsx bytes, returning (col, colOff, row, rowOff) as integers.
// col/row are 0-based OOXML indices; colOff/rowOff are in EMU.
func drawingFromAnchor(t *testing.T, xlsxBytes []byte) (col, colOff, row, rowOff int) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(xlsxBytes), int64(len(xlsxBytes)))
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	var xmlData []byte
	for _, f := range zr.File {
		if f.Name == "xl/drawings/drawing1.xml" {
			rc, openErr := f.Open()
			if openErr != nil {
				t.Fatalf("open drawing1.xml: %v", openErr)
			}
			buf := new(bytes.Buffer)
			if _, copyErr := buf.ReadFrom(rc); copyErr != nil {
				t.Fatalf("read drawing1.xml: %v", copyErr)
			}
			rc.Close()
			xmlData = buf.Bytes()
			break
		}
	}
	if xmlData == nil {
		t.Fatal("xl/drawings/drawing1.xml not found in xlsx")
	}

	// Extract <xdr:from> block (first occurrence).
	fromRe := regexp.MustCompile(`(?s)<xdr:from>(.*?)</xdr:from>`)
	fromM := fromRe.FindSubmatch(xmlData)
	if fromM == nil {
		t.Fatalf("no <xdr:from> found in drawing1.xml:\n%s", xmlData)
	}
	fromBlock := fromM[1]

	extract := func(tag string) int {
		re := regexp.MustCompile(fmt.Sprintf(`<xdr:%s>(\d+)</xdr:%s>`, tag, tag))
		m := re.FindSubmatch(fromBlock)
		if m == nil {
			t.Fatalf("tag <xdr:%s> not found in from block: %s", tag, fromBlock)
		}
		v, _ := strconv.Atoi(string(m[1]))
		return v
	}
	return extract("col"), extract("colOff"), extract("row"), extract("rowOff")
}

// cellToColRow converts a cell address like "B8" into 0-based OOXML col and
// row indices (col: A=0, B=1, ...; row: row1=0, row8=7).
func cellToColRow(t *testing.T, cell string) (col, row int) {
	t.Helper()
	// Split at the first digit boundary.
	splitRe := regexp.MustCompile(`^([A-Z]+)(\d+)$`)
	m := splitRe.FindStringSubmatch(cell)
	if m == nil {
		t.Fatalf("invalid cell address %q", cell)
	}
	letters, numStr := m[1], m[2]

	colNum := 0
	for _, ch := range letters {
		colNum = colNum*26 + int(ch-'A'+1)
	}
	col = colNum - 1 // 0-based

	rowNum, _ := strconv.Atoi(numStr)
	row = rowNum - 1 // 0-based
	return
}

// TestQRCenteredInBox verifies that the embedded QR anchor in drawing1.xml
// exactly matches the values computed by qrCenterAnchor for all 6 binder
// configurations. Offsets are compared as EMU (px * 9525).
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
		c := c
		t.Run(c.name, func(t *testing.T) {
			data := map[string]any{
				"eq_number": "E", "eq_doc_number": "D", "eq_doc_title": "T",
				"eq_doc_count": "1", "eq_doc_department": "Q", "eq_doc_year": "2026",
				"pjt_number": "P", "pjt_test_number": "T", "pjt_doc_title": "T",
				"pjt_doc_writer": "W", "pjt_doc_count": "1",
			}
			lbl := label.MakeLabel(data, c.docType)
			out, _, err := g.CreateLabelExcel(c.docType, c.binder, lbl, [][]byte{smallPNG(t)})
			if err != nil {
				t.Fatalf("CreateLabelExcel: %v", err)
			}

			// Compute expected anchor from qrCenterAnchor.
			colW := label.GetQRConfig(c.binder).ColumnWidth
			anchorCell, wantOffXpx, wantOffYpx := qrCenterAnchor(c.docType, colW)

			if wantOffXpx < 0 || wantOffYpx < 0 {
				t.Errorf("qrCenterAnchor returned negative offsets: offX=%d offY=%d", wantOffXpx, wantOffYpx)
			}

			wantCol, wantRow := cellToColRow(t, anchorCell)
			wantColOff := wantOffXpx * 9525 // px -> EMU
			wantRowOff := wantOffYpx * 9525

			// Extract actual anchor from drawing1.xml.
			gotCol, gotColOff, gotRow, gotRowOff := drawingFromAnchor(t, out)

			if gotCol != wantCol {
				t.Errorf("col: got %d want %d (cell %s)", gotCol, wantCol, anchorCell)
			}
			if gotColOff != wantColOff {
				t.Errorf("colOff EMU: got %d want %d (offX=%dpx)", gotColOff, wantColOff, wantOffXpx)
			}
			if gotRow != wantRow {
				t.Errorf("row: got %d want %d (cell %s)", gotRow, wantRow, anchorCell)
			}
			if gotRowOff != wantRowOff {
				t.Errorf("rowOff EMU: got %d want %d (offY=%dpx)", gotRowOff, wantRowOff, wantOffYpx)
			}

			t.Logf("%s anchor: cell=%s col=%d colOff=%d row=%d rowOff=%d",
				c.name, anchorCell, gotCol, gotColOff, gotRow, gotRowOff)
		})
	}
}
