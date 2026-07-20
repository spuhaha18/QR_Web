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

// packAndDraw lays out the pieces and renders each. No cut guides: the user's
// print comparison showed the dashed rect reads as part of the label and
// confuses size verification — the 5mm gap alone separates pieces, and the
// label's own medium border is the cutting edge.
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
		if err := pieces[i].draw(doc, pl.x, pl.y); err != nil {
			return err
		}
	}
	return doc.Error()
}
