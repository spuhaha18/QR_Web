package pdf

import "github.com/go-pdf/fpdf"

// fontFamily is the single family every label text uses: Times New Roman with
// Batang registered as glyph-level fallback for Korean.
const fontFamily = "times"

const batangFamily = "batang"

// newDoc returns an A4-portrait mm-unit document with both families
// registered. fpdf has no glyph fallback; Korean-vs-Latin routing happens in
// textfit's run splitting (familyFor), not here. Auto page break off: the
// packer owns page boundaries.
func newDoc() *fpdf.Fpdf {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetAutoPageBreak(false, 0)
	// fpdf's default 1mm cell margin shifts left-aligned CellFormat output
	// right of the given x, so rendered lines land ~1mm right of where
	// textfit measured them — visibly crossing the border on narrow (1cm)
	// labels. drawTextBox does its own centering and padding (textPadMM);
	// the cell margin must be zero so drawn x == measured x.
	doc.SetCellMargin(0)
	doc.AddUTF8FontFromBytes(fontFamily, "", fontTimes)
	doc.AddUTF8FontFromBytes(fontFamily, "B", fontTimesBold)
	// Batang has no bold face; register the same bytes for "B" so bold-styled
	// Korean renders regular-weight Batang glyphs (grill Q4 decision).
	doc.AddUTF8FontFromBytes(batangFamily, "", fontBatang)
	doc.AddUTF8FontFromBytes(batangFamily, "B", fontBatang)
	return doc
}
