package excel

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/xuri/excelize/v2"

	"qrweb/internal/label"
)

// Generator builds label .xlsx workbooks. Ported from
// excel_generator.ExcelLabelGenerator.
type Generator struct{}

// NewGenerator returns a ready-to-use Generator.
func NewGenerator() *Generator { return &Generator{} }

const sheet1 = "Sheet 1"

// CreateLabelExcel builds the label workbook and returns its bytes plus the
// timestamped filename.
//
// dt: equipment or project. binder: validated thickness. lbl: parsed label.
// qrs: one validated PNG per sheet, in sheet order (QR i embeds into sheet i);
// the QRImageSet type guarantees the count matches lbl.DocCount().
func (g *Generator) CreateLabelExcel(dt label.DocType, binder label.BinderSize, lbl label.Label, qrs label.QRImageSet) (data []byte, filename string, err error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err = f.SetSheetName("Sheet1", sheet1); err != nil {
		return nil, "", err
	}

	docCount := lbl.DocCount()

	// --- Sheet 1: full layout ---
	if err = g.buildSheet1(f, dt, lbl); err != nil {
		return nil, "", err
	}

	// --- additional sheets 2..N (copy of Sheet 1 with i/N overrides) ---
	if docCount > 1 {
		if err = g.createAdditionalSheets(f, dt, docCount); err != nil {
			return nil, "", err
		}
	}

	// --- QR embed: every sheet, after all sheets exist (Python order) ---
	if err = g.applyQRCodes(f, dt, binder, qrs); err != nil {
		return nil, "", err
	}

	buf, werr := f.WriteToBuffer()
	if werr != nil {
		return nil, "", werr
	}
	// excelize's natural output coalesces equal-width adjacent columns into a
	// single <col min=a max=b> span; this is visually identical to per-column
	// elements. The comparator (compare_xlsx.py) normalizes both sides by
	// expanding column_dimensions over their min..max range, so no XML
	// post-processing is needed here.
	filename = label.GenerateTimestampFilename(lbl.DocNumber(), "xlsx")
	return buf.Bytes(), filename, nil
}

// buildSheet1 constructs Sheet 1: basic layout, common borders, then
// equipment- or project-specific layout, then flushes synthesized styles.
func (g *Generator) buildSheet1(f *excelize.File, dt label.DocType, lbl label.Label) error {
	sb := newStyleBuilder(f)

	if err := setupBasicLayout(f); err != nil {
		return err
	}
	applyCommonBorders(sb)

	if dt == label.DocTypeEquipment {
		if err := setupEquipmentDocument(f, sb, dt, lbl); err != nil {
			return err
		}
	} else {
		if err := setupProjectDocument(f, sb, dt, lbl); err != nil {
			return err
		}
	}

	// Flush synthesized per-cell styles (fonts + borders + global alignment).
	return sb.flush(sheet1)
}

// setupBasicLayout sets row heights, A/N column widths, and the B2..B6 merges
// (mirrors _setup_basic_layout). Row heights come from the package-shared
// rowHeights table (also consumed by the geometry math), so the two never drift.
func setupBasicLayout(f *excelize.File) error {
	for r := 1; r <= 18; r++ {
		if err := f.SetRowHeight(sheet1, r, rowHeights[r]); err != nil {
			return err
		}
	}

	if err := f.SetColWidth(sheet1, "A", "A", narrowColWidth); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet1, "N", "N", narrowColWidth); err != nil {
		return err
	}

	for _, m := range [][2]string{{"B2", "M2"}, {"B3", "M3"}, {"B4", "M4"}, {"B5", "M5"}, {"B6", "M6"}} {
		if err := f.MergeCell(sheet1, m[0], m[1]); err != nil {
			return err
		}
	}
	return nil
}

// applyCommonBorders replays _apply_borders: B2:M6 thin, outer medium edges,
// then 2-side medium corners (replace semantics, in source order).
func applyCommonBorders(sb *styleBuilder) {
	thin := map[string]string{"left": "thin", "right": "thin", "top": "thin", "bottom": "thin"}
	sb.setBorderRange("B2:M6", thin)

	// Outer medium edges (each replaces the cell border).
	sb.setBorderRange("A1:A18", map[string]string{"left": "medium"})
	sb.setBorderRange("N1:N18", map[string]string{"right": "medium"})
	sb.setBorderRange("A1:N1", map[string]string{"top": "medium"})
	sb.setBorderRange("A18:N18", map[string]string{"bottom": "medium"})

	// 2-side medium corners.
	sb.setBorderCell("A1", map[string]string{"left": "medium", "top": "medium"})
	sb.setBorderCell("N1", map[string]string{"right": "medium", "top": "medium"})
	sb.setBorderCell("A18", map[string]string{"left": "medium", "bottom": "medium"})
	sb.setBorderCell("N18", map[string]string{"right": "medium", "bottom": "medium"})
}

// setupEquipmentDocument mirrors _setup_equipment_document: B7:M7 merge,
// additional borders, cell values + fonts. The QR box top row comes from the
// doc type's Layout (8 for equipment), the single source also read by the
// geometry math — the literal cell strings are derived from it, not hardcoded.
func setupEquipmentDocument(f *excelize.File, sb *styleBuilder, dt label.DocType, lbl label.Label) error {
	if err := f.MergeCell(sheet1, "B7", "M7"); err != nil {
		return err
	}

	layout := dt.Layout()
	top, bottom := layout.QRBoxTopRow, layout.QRBoxBottomRow
	// additional_borders, in Python source order:
	//   B2:M7 thin (full replace) ; box top ; box left ; box right ; box bottom
	sb.setBorderRange("B2:M7", map[string]string{"left": "thin", "right": "thin", "top": "thin", "bottom": "thin"})
	sb.setBorderRange(fmt.Sprintf("B%d:M%d", top, top), map[string]string{"top": "thin"})
	sb.setBorderRange(fmt.Sprintf("B%d:B%d", top, bottom), map[string]string{"left": "thin"})
	sb.setBorderRange(fmt.Sprintf("M%d:M%d", top, bottom), map[string]string{"right": "thin"})
	sb.setBorderRange(fmt.Sprintf("B%d:M%d", bottom, bottom), map[string]string{"bottom": "thin"})

	return applyCellValuesAndFonts(f, sb, lbl)
}

// setupProjectDocument mirrors _setup_project_document.
func setupProjectDocument(f *excelize.File, sb *styleBuilder, dt label.DocType, lbl label.Label) error {
	// additional rows
	for r, h := range map[int]float64{20: 2.25, 21: 48, 22: 34.5, 23: 27.75, 24: 2.25} {
		if err := f.SetRowHeight(sheet1, r, h); err != nil {
			return err
		}
	}
	// additional columns
	if err := f.SetColWidth(sheet1, "Q", "Q", 8.13); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet1, "R", "R", 34.88); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet1, "S", "S", 8.13); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet1, "T", "T", narrowColWidth); err != nil {
		return err
	}

	if err := f.MergeCell(sheet1, "Q21", "S21"); err != nil {
		return err
	}
	if err := f.MergeCell(sheet1, "Q22", "S22"); err != nil {
		return err
	}

	applyProjectBorders(f, sb, dt)

	if err := applyCellValuesAndFonts(f, sb, lbl); err != nil {
		return err
	}

	// print_area A1:T24
	return setPrintArea(f, sheet1)
}

// applyProjectBorders replays _apply_project_borders in exact source order. The
// QR box top row comes from the doc type's Layout (7 for project), the single
// source also read by the geometry math.
func applyProjectBorders(f *excelize.File, sb *styleBuilder, dt label.DocType) {
	layout := dt.Layout()
	top, bottom := layout.QRBoxTopRow, layout.QRBoxBottomRow
	sb.setBorderRange(fmt.Sprintf("B%d:M%d", top, top), map[string]string{"top": "thin"})
	sb.setBorderRange(fmt.Sprintf("B%d:B%d", top, bottom), map[string]string{"left": "thin"})
	sb.setBorderRange(fmt.Sprintf("M%d:M%d", top, bottom), map[string]string{"right": "thin"})
	sb.setBorderRange(fmt.Sprintf("B%d:M%d", bottom, bottom), map[string]string{"bottom": "thin"})

	sb.setBorderCell(fmt.Sprintf("B%d", bottom), map[string]string{"left": "thin", "bottom": "thin"})
	sb.setBorderCell(fmt.Sprintf("M%d", bottom), map[string]string{"right": "thin", "bottom": "thin"})

	// N,O,P spacer widths (Python: columns 14..16). N already set; set O,P too.
	for _, col := range []string{"N", "O", "P"} {
		_ = f.SetColWidth(sheet1, col, col, narrowColWidth)
	}

	// Q20:S20, Q24:S24 top+bottom thin
	sb.setBorderRange("Q20:S20", map[string]string{"top": "thin", "bottom": "thin"})
	sb.setBorderRange("Q24:S24", map[string]string{"top": "thin", "bottom": "thin"})

	// P21:P23, T21:T23 left+right thin
	sb.setBorderRange("P21:P23", map[string]string{"left": "thin", "right": "thin"})
	sb.setBorderRange("T21:T23", map[string]string{"left": "thin", "right": "thin"})

	// corner cells P20/T20/P24/T24
	sb.setBorderCell("P20", map[string]string{"left": "thin", "top": "thin"})
	sb.setBorderCell("T20", map[string]string{"right": "thin", "top": "thin"})
	sb.setBorderCell("P24", map[string]string{"left": "thin", "bottom": "thin"})
	sb.setBorderCell("T24", map[string]string{"right": "thin", "bottom": "thin"})

	// Q22:S22 thin full
	sb.setBorderRange("Q22:S22", map[string]string{"left": "thin", "right": "thin", "top": "thin", "bottom": "thin"})
}

// applyCellValuesAndFonts writes the label's cell values and applies its
// declared per-cell fonts. Both the addresses/values and the font intents are
// owned by the Label (CellValues + CellFonts); the renderer only translates
// each intent to an excelize font. Equipment and project share this one path —
// the doc-type-specific font block is gone.
func applyCellValuesAndFonts(f *excelize.File, sb *styleBuilder, lbl label.Label) error {
	for addr, v := range lbl.CellValues() {
		if err := setCellValue(f, addr, v); err != nil {
			return err
		}
	}
	for addr, cf := range lbl.CellFonts() {
		sb.setFontCell(addr, fontKindFor(cf))
	}
	return nil
}

// setCellValue writes a value with the correct type (int year stays numeric).
func setCellValue(f *excelize.File, addr string, v any) error {
	switch val := v.(type) {
	case int:
		return f.SetCellInt(sheet1, addr, int64(val))
	case string:
		return f.SetCellStr(sheet1, addr, val)
	default:
		return f.SetCellValue(sheet1, addr, v)
	}
}

// createAdditionalSheets copies Sheet 1 to Sheet 2..N, overriding B5 (and S23
// for project) with i/N, and re-applying B4 title font + global alignment.
//
// CopySheet preserves merges, dimensions, and cell styles; QR images are added
// later per sheet, so image fidelity is not relied upon here.
func (g *Generator) createAdditionalSheets(f *excelize.File, dt label.DocType, docCount int) error {
	srcIdx, err := f.GetSheetIndex(sheet1)
	if err != nil {
		return err
	}
	layout := dt.Layout()
	for i := 2; i <= docCount; i++ {
		name := fmt.Sprintf("Sheet %d", i)
		toIdx, nerr := f.NewSheet(name)
		if nerr != nil {
			return nerr
		}
		if cerr := f.CopySheet(srcIdx, toIdx); cerr != nil {
			return cerr
		}

		// Override the volume marker in every cell the layout declares (B5
		// always; project mirrors it in S23).
		marker := fmt.Sprintf("%d/%d", i, docCount)
		for _, cell := range layout.CountCells {
			if err := f.SetCellStr(name, cell, marker); err != nil {
				return err
			}
		}
		if layout.HasPrintArea {
			if err := setPrintArea(f, name); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyQRCodes sets B–M column width and embeds one 75x75px QR per sheet,
// centered in the lower bordered box (B8:M17 equipment, B7:M17 project). The
// QRImageSet guarantees one image per sheet, so no length check is needed here.
func (g *Generator) applyQRCodes(f *excelize.File, dt label.DocType, binder label.BinderSize, qrs label.QRImageSet) error {
	colWidth := binder.ColumnWidth()
	sheets := f.GetSheetList()
	images := qrs.Images()

	for _, s := range sheets {
		if err := f.SetColWidth(s, "B", "M", colWidth); err != nil {
			return err
		}
	}

	// Center the QR in the lower bordered box (B8:M17 equipment, B7:M17 project).
	anchorCell, offX, offY := qrCenterAnchor(dt, colWidth)

	for idx, s := range sheets {
		pngBytes := images[idx]
		cfgImg, _, derr := image.DecodeConfig(bytes.NewReader(pngBytes))
		if derr != nil {
			return fmt.Errorf("decode QR PNG for %s: %w", s, derr)
		}
		scaleX := qrSizePx / float64(cfgImg.Width)
		scaleY := qrSizePx / float64(cfgImg.Height)
		if err := f.AddPictureFromBytes(s, anchorCell, &excelize.Picture{
			Extension: ".png",
			File:      pngBytes,
			Format: &excelize.GraphicOptions{
				ScaleX:          scaleX,
				ScaleY:          scaleY,
				OffsetX:         offX,
				OffsetY:         offY,
				Positioning:     "oneCell",
				AutoFit:         false,
				LockAspectRatio: false,
			},
		}); err != nil {
			return fmt.Errorf("add QR to %s: %w", s, err)
		}
	}
	return nil
}

// setPrintArea sets the print area A1:T24 for the given sheet, scoped to it.
func setPrintArea(f *excelize.File, sheet string) error {
	return f.SetDefinedName(&excelize.DefinedName{
		Name:     "_xlnm.Print_Area",
		RefersTo: fmt.Sprintf("'%s'!$A$1:$T$24", sheet),
		Scope:    sheet,
	})
}
