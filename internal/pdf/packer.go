package pdf

import "github.com/go-pdf/fpdf"

const (
	pageW    = 210.0
	pageH    = 297.0
	marginMM = 10.0
	gapMM    = 5.0
)

// piece is one cuttable label fragment: its size plus a draw callback bound to
// its content.
type piece struct {
	w, h float64
	draw func(doc *fpdf.Fpdf, x, y float64) error
}

// placed is a piece's resolved position (0-based page, top-left mm).
type placed struct {
	page int
	x, y float64
}

// layoutPieces shelf-packs pieces in order: left-to-right rows, top-to-bottom,
// new page when a row doesn't fit. Pure function so tests need no PDF.
func layoutPieces(sizes [][2]float64) []placed {
	out := make([]placed, len(sizes))
	page, x, y, rowH := 0, marginMM, marginMM, 0.0
	for i, s := range sizes {
		w, h := s[0], s[1]
		if x+w > pageW-marginMM && x > marginMM { // row full
			x = marginMM
			y += rowH + gapMM
			rowH = 0
		}
		if y+h > pageH-marginMM && y > marginMM { // page full
			page++
			x, y, rowH = marginMM, marginMM, 0
		}
		out[i] = placed{page: page, x: x, y: y}
		x += w + gapMM
		if h > rowH {
			rowH = h
		}
	}
	return out
}

// cutGuide draws a dashed gray rectangle 2.5mm outside the piece, clamped to
// the page, marking where to cut (centered in the 5mm gap).
func cutGuide(doc *fpdf.Fpdf, x, y, w, h float64) {
	const off = gapMM / 2.0
	x1, y1 := max(x-off, 1.0), max(y-off, 1.0)
	x2, y2 := min(x+w+off, pageW-1.0), min(y+h+off, pageH-1.0)
	doc.SetDrawColor(153, 153, 153)
	doc.SetLineWidth(0.15)
	doc.SetDashPattern([]float64{2, 1.5}, 0)
	doc.Rect(x1, y1, x2-x1, y2-y1, "D")
	doc.SetDashPattern([]float64{}, 0)
	doc.SetDrawColor(0, 0, 0)
}

// packAndDraw lays out the pieces and renders each with its cut guide.
//
// Single walk: layoutPieces preserves piece order and pages are monotonically
// non-decreasing, so a page is added exactly when a placement's page advances
// past the pages added so far — no need to revisit fpdf's current-page state.
func packAndDraw(doc *fpdf.Fpdf, pieces []piece) error {
	sizes := make([][2]float64, len(pieces))
	for i, p := range pieces {
		sizes[i] = [2]float64{p.w, p.h}
	}
	placements := layoutPieces(sizes)
	pages := 0
	for i, pl := range placements {
		for pl.page >= pages {
			doc.AddPage()
			pages++
		}
		cutGuide(doc, pl.x, pl.y, pieces[i].w, pieces[i].h)
		if err := pieces[i].draw(doc, pl.x, pl.y); err != nil {
			return err
		}
	}
	return doc.Error()
}
