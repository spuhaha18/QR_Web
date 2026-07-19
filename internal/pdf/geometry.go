package pdf

import (
	"math"

	"qrweb/internal/label"
)

// Unit model (the definition of "same size as Excel 100% print"):
// column width (char units) -> px via the OOXML formula at MDW=7, px -> mm at
// 96dpi; row heights are points, pt -> mm at 72dpi.
const (
	mdw       = 7.0
	pxPerInch = 96.0
	ptPerInch = 72.0
	mmPerInch = 25.4
)

// qrSizeMM is the 75px QR side expressed in mm (75/96*25.4).
const qrSizeMM = 19.84375

func colWidthToPx(w float64) float64 {
	return math.Trunc(((256.0*w + math.Trunc(128.0/mdw)) / 256.0) * mdw)
}

func pxToMM(px float64) float64 { return px * mmPerInch / pxPerInch }
func ptToMM(pt float64) float64 { return pt * mmPerInch / ptPerInch }

// colMM converts a column width in char units straight to mm.
func colMM(w float64) float64 { return pxToMM(colWidthToPx(w)) }

// narrowColWidth mirrors excel.narrowColWidth (spacer cols A/N/P/T).
const narrowColWidth = 0.375

// mainRowHeightsPt mirrors excel.rowHeights rows 1..18 (index 0 unused).
var mainRowHeightsPt = []float64{
	0,
	2.25, 27, 27, 216, 40.5, 27, 27,
	6.75, 6.75, 6.75, 6.75, 6.75,
	6.75, 6.75, 6.75, 6.75, 6.75,
	2.25,
}

// auxRowHeightsPt are project rows 20..24.
var auxRowHeightsPt = []float64{2.25, 48, 34.5, 27.75, 2.25}

// auxColWidths are project cols P,Q,R,S,T in char units.
var auxColWidths = []float64{narrowColWidth, 8.13, 34.88, 8.13, narrowColWidth}

// grid holds cumulative mm offsets, piece-local. colX[0]==0, last element is
// the piece width; rowY likewise for height.
type grid struct {
	colX []float64
	rowY []float64
}

func cumulate(widths []float64) []float64 {
	out := make([]float64, len(widths)+1)
	for i, w := range widths {
		out[i+1] = out[i] + w
	}
	return out
}

// mainGrid returns the main-label grid for cols A..N (14 cols) and rows 1..18.
func mainGrid(b label.BinderSize) grid {
	cols := make([]float64, 0, 14)
	cols = append(cols, colMM(narrowColWidth)) // A
	for i := 0; i < 12; i++ {                  // B..M
		cols = append(cols, colMM(b.ColumnWidth()))
	}
	cols = append(cols, colMM(narrowColWidth)) // N
	rows := make([]float64, 0, 18)
	for r := 1; r <= 18; r++ {
		rows = append(rows, ptToMM(mainRowHeightsPt[r]))
	}
	return grid{colX: cumulate(cols), rowY: cumulate(rows)}
}

// auxGrid returns the project side-table grid, cols P..T and rows 20..24.
func auxGrid() grid {
	cols := make([]float64, len(auxColWidths))
	for i, w := range auxColWidths {
		cols[i] = colMM(w)
	}
	rows := make([]float64, len(auxRowHeightsPt))
	for i, h := range auxRowHeightsPt {
		rows[i] = ptToMM(h)
	}
	return grid{colX: cumulate(cols), rowY: cumulate(rows)}
}

func mainSize(b label.BinderSize) (w, h float64) {
	g := mainGrid(b)
	return g.colX[len(g.colX)-1], g.rowY[len(g.rowY)-1]
}

func auxSize() (w, h float64) {
	g := auxGrid()
	return g.colX[len(g.colX)-1], g.rowY[len(g.rowY)-1]
}
