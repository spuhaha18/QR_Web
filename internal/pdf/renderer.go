package pdf

import (
	"bytes"
	"strconv"

	"github.com/go-pdf/fpdf"

	"qrweb/internal/label"
)

const (
	thinMM   = 0.2
	mediumMM = 0.5
)

// sizeFor maps the domain font intent to its base point size (all bold Times).
func sizeFor(cf label.CellFont) float64 {
	switch cf {
	case label.CellFontTitle:
		return 16
	case label.CellFontHeading:
		return 20
	case label.CellFontSub:
		return 13
	default:
		return 12
	}
}

func rect(doc *fpdf.Fpdf, x1, y1, x2, y2, lineW float64) {
	doc.SetLineWidth(lineW)
	doc.Rect(x1, y1, x2-x1, y2-y1, "D")
}

// qrSide returns the QR square side: 19.84mm, shrunk to fit a smaller box
// (the 1cm binder's QR box is only ~15.9mm wide).
func qrSide(boxW, boxH float64) float64 {
	side := qrSizeMM
	if boxW < side {
		side = boxW
	}
	if boxH < side {
		side = boxH
	}
	return side
}

// cellText returns the string form of a CellValues entry (year is an int).
func cellText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

// renderMain draws one main label piece at page position (x, y). marker
// replaces the B5 (i/N) value; qrName must be unique per registered image.
func renderMain(doc *fpdf.Fpdf, x, y float64, dt label.DocType, b label.BinderSize, lbl label.Label, marker string, qrPNG []byte, qrName string) error {
	g := mainGrid(b)
	w, h := mainSize(b)
	doc.SetDrawColor(0, 0, 0)

	// Outer medium frame.
	rect(doc, x, y, x+w, y+h, mediumMM)

	// Value rows: rows 2..6 always, row 7 for equipment. x spans B..M.
	xl, xr := x+g.colX[1], x+g.colX[13]
	layout := dt.Layout()
	lastValueRow := layout.QRBoxTopRow - 1
	for r := 2; r <= lastValueRow; r++ {
		rect(doc, xl, y+g.rowY[r-1], xr, y+g.rowY[r], thinMM)
	}

	// QR box.
	boxTop, boxBot := y+g.rowY[layout.QRBoxTopRow-1], y+g.rowY[layout.QRBoxBottomRow]
	rect(doc, xl, boxTop, xr, boxBot, thinMM)

	// Values. B5 carries the per-piece i/N marker.
	values := lbl.CellValues()
	fonts := lbl.CellFonts()
	rowOf := map[string]int{"B2": 2, "B3": 3, "B4": 4, "B5": 5, "B6": 6, "B7": 7}
	for addr, r := range rowOf {
		v, ok := values[addr]
		if !ok {
			continue
		}
		text := cellText(v)
		if addr == "B5" {
			text = marker
		}
		drawTextBox(doc, xl, y+g.rowY[r-1], xr-xl, g.rowY[r]-g.rowY[r-1], text, sizeFor(fonts[addr]))
	}

	// QR centered in the box, shrunk to fit when the box is smaller than
	// 19.84mm (1cm binder) — grill Q1 decision: shrink, not Excel-style
	// overflow.
	side := qrSide(xr-xl, boxBot-boxTop)
	qx := xl + ((xr - xl) - side) / 2.0
	qy := boxTop + ((boxBot - boxTop) - side) / 2.0
	opts := fpdf.ImageOptions{ImageType: "PNG"}
	doc.RegisterImageOptionsReader(qrName, opts, bytes.NewReader(qrPNG))
	doc.ImageOptions(qrName, qx, qy, side, side, false, opts, 0, "")
	return doc.Error()
}

// renderAux draws the project side table at (x, y). marker replaces S23.
func renderAux(doc *fpdf.Fpdf, x, y float64, lbl label.Label, marker string) {
	g := auxGrid()
	w, h := auxSize()
	doc.SetDrawColor(0, 0, 0)

	rect(doc, x, y, x+w, y+h, thinMM)                                     // outer
	rect(doc, x+g.colX[1], y+g.rowY[1], x+g.colX[4], y+g.rowY[4], thinMM) // inner Q21..S23
	rect(doc, x+g.colX[1], y+g.rowY[2], x+g.colX[4], y+g.rowY[3], thinMM) // Q22:S22

	values := lbl.CellValues()
	fonts := lbl.CellFonts()
	draw := func(addr string, x1, y1, x2, y2 float64, override string) {
		text := override
		if text == "" {
			text = cellText(values[addr])
		}
		drawTextBox(doc, x+x1, y+y1, x2-x1, y2-y1, text, sizeFor(fonts[addr]))
	}
	draw("Q21", g.colX[1], g.rowY[1], g.colX[4], g.rowY[2], "")
	draw("Q22", g.colX[1], g.rowY[2], g.colX[4], g.rowY[3], "")
	draw("R23", g.colX[2], g.rowY[3], g.colX[3], g.rowY[4], "")
	draw("S23", g.colX[3], g.rowY[3], g.colX[4], g.rowY[4], marker)
}
