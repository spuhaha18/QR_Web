package excel

import (
	"math"
	"strconv"
)

// qrSizePx is the fixed QR side length in pixels (75x75).
const qrSizePx = 75.0

// mdw is the max digit width for Calibri 11, used by Excel's column
// width->pixel conversion.
const mdw = 7.0

// colWidthToPx converts an Excel column width (in characters) to pixels using
// the OOXML formula with MDW=7.
func colWidthToPx(w float64) float64 {
	return math.Trunc(((256.0*w + math.Trunc(128.0/mdw)) / 256.0) * mdw)
}

// rowHeightToPx converts an Excel row height (points) to pixels (96/72 DPI).
func rowHeightToPx(h float64) float64 {
	return math.Round(h * 4.0 / 3.0)
}

// rowHeights holds row heights (points) for rows 1..17 (index 1-based; index 0
// unused). Mirrors _setup_basic_layout in excel_generator.py: rows 1-7 explicit,
// rows 8-17 = 6.75.
var rowHeights = func() []float64 {
	h := make([]float64, 18)
	explicit := map[int]float64{1: 2.25, 2: 27, 3: 27, 4: 216, 5: 40.5, 6: 27, 7: 27}
	for r := 1; r <= 17; r++ {
		if v, ok := explicit[r]; ok {
			h[r] = v
		} else {
			h[r] = 6.75
		}
	}
	return h
}()

// colLetters for the columns we may anchor in: A..M.
var colLetters = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M"}

// qrCenterAnchor returns the from-cell and in-cell pixel offsets that place a
// 75x75px QR at the dead center of the lower bordered box. Box = B8:M17 for
// equipment (docType "1"), B7:M17 for project. colW is the B–M column width.
//
// Horizontal: box spans columns B..M (12 cols of width colW); box left is the
// right edge of column A (0.375). Vertical: box top row is 8 (equipment) or 7
// (project), bottom row 17. Targets are clamped to >=0 (a QR wider/taller than
// its box overflows symmetrically until it hits the sheet edge).
func qrCenterAnchor(docType string, colW float64) (cell string, offX, offY int) {
	const colA = 0.375
	// Horizontal geometry.
	boxLeftPx := colWidthToPx(colA)
	boxWidthPx := 12.0 * colWidthToPx(colW)
	targetX := boxLeftPx + (boxWidthPx-qrSizePx)/2.0
	if targetX < 0 {
		targetX = 0
	}
	// Vertical geometry.
	topRow := 8
	if docType != "1" {
		topRow = 7
	}
	boxTopPx := 0.0
	for r := 1; r < topRow; r++ {
		boxTopPx += rowHeightToPx(rowHeights[r])
	}
	boxHeightPx := 0.0
	for r := topRow; r <= 17; r++ {
		boxHeightPx += rowHeightToPx(rowHeights[r])
	}
	targetY := boxTopPx + (boxHeightPx-qrSizePx)/2.0
	if targetY < 0 {
		targetY = 0
	}

	// Resolve targetX -> (column, in-cell offset) by walking A,B,...,M.
	colIdx, accX := 0, 0.0
	for i, letter := range colLetters {
		w := colW
		if letter == "A" {
			w = colA
		}
		wpx := colWidthToPx(w)
		// Stop when the accumulated width reaches or passes targetX (exact match
		// stays in the current cell — the offset becomes 0 or near-0).
		if accX+wpx > targetX || i == len(colLetters)-1 {
			colIdx = i
			break
		}
		accX += wpx
	}
	offX = int(math.Round(targetX - accX))

	// Resolve targetY -> (row, in-cell offset) by walking rows 1..17.
	rowNum, accY := 1, 0.0
	for r := 1; r <= 17; r++ {
		hpx := rowHeightToPx(rowHeights[r])
		// Stop when the accumulated height reaches or passes targetY (exact match
		// stays in the current row — the offset becomes 0 or near-0).
		if accY+hpx > targetY || r == 17 {
			rowNum = r
			break
		}
		accY += hpx
	}
	offY = int(math.Round(targetY - accY))

	return colLetters[colIdx] + strconv.Itoa(rowNum), offX, offY
}
