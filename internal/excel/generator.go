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
// docType: "1" (equipment) | "2" (project). binder: 1|3|5|7. lbl: parsed label.
// qrPNGs: one PNG per sheet (paste mode), indexed by sheet order; QR i is
// embedded into sheet i.
func (g *Generator) CreateLabelExcel(docType string, binder int, lbl label.Label, qrPNGs [][]byte) (data []byte, filename string, err error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err = f.SetSheetName("Sheet1", sheet1); err != nil {
		return nil, "", err
	}

	docCount := lbl.DocCount()

	// --- Sheet 1: full layout ---
	if err = g.buildSheet1(f, docType, lbl); err != nil {
		return nil, "", err
	}

	// --- additional sheets 2..N (copy of Sheet 1 with i/N overrides) ---
	if docCount > 1 {
		if err = g.createAdditionalSheets(f, docType, docCount); err != nil {
			return nil, "", err
		}
	}

	// --- QR embed: every sheet, after all sheets exist (Python order) ---
	if err = g.applyQRCodes(f, docType, binder, qrPNGs); err != nil {
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
func (g *Generator) buildSheet1(f *excelize.File, docType string, lbl label.Label) error {
	sb := newStyleBuilder(f)

	if err := setupBasicLayout(f); err != nil {
		return err
	}
	applyCommonBorders(sb)

	if docType == "1" {
		if err := setupEquipmentDocument(f, sb, lbl); err != nil {
			return err
		}
	} else {
		if err := setupProjectDocument(f, sb, lbl); err != nil {
			return err
		}
	}

	// Flush synthesized per-cell styles (fonts + borders + global alignment).
	return sb.flush(sheet1)
}

// setupBasicLayout sets row heights, A/N column widths, and the B2..B6 merges
// (mirrors _setup_basic_layout).
func setupBasicLayout(f *excelize.File) error {
	rowHeights := map[int]float64{1: 2.25, 2: 27, 3: 27, 4: 216, 5: 40.5, 6: 27, 7: 27}
	for r, h := range rowHeights {
		if err := f.SetRowHeight(sheet1, r, h); err != nil {
			return err
		}
	}
	for r := 8; r <= 17; r++ {
		if err := f.SetRowHeight(sheet1, r, 6.75); err != nil {
			return err
		}
	}
	if err := f.SetRowHeight(sheet1, 18, 2.25); err != nil {
		return err
	}

	if err := f.SetColWidth(sheet1, "A", "A", 0.375); err != nil {
		return err
	}
	if err := f.SetColWidth(sheet1, "N", "N", 0.375); err != nil {
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
// additional borders, cell values + fonts.
func setupEquipmentDocument(f *excelize.File, sb *styleBuilder, lbl label.Label) error {
	if err := f.MergeCell(sheet1, "B7", "M7"); err != nil {
		return err
	}

	// additional_borders, in Python source order:
	//   B2:M7 thin (full replace) ; B8:M8 top ; B8:B17 left ; M8:M17 right ; B17:M17 bottom
	sb.setBorderRange("B2:M7", map[string]string{"left": "thin", "right": "thin", "top": "thin", "bottom": "thin"})
	sb.setBorderRange("B8:M8", map[string]string{"top": "thin"})
	sb.setBorderRange("B8:B17", map[string]string{"left": "thin"})
	sb.setBorderRange("M8:M17", map[string]string{"right": "thin"})
	sb.setBorderRange("B17:M17", map[string]string{"bottom": "thin"})

	return applyCellValues(f, sb, lbl)
}

// setupProjectDocument mirrors _setup_project_document.
func setupProjectDocument(f *excelize.File, sb *styleBuilder, lbl label.Label) error {
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
	if err := f.SetColWidth(sheet1, "T", "T", 0.375); err != nil {
		return err
	}

	if err := f.MergeCell(sheet1, "Q21", "S21"); err != nil {
		return err
	}
	if err := f.MergeCell(sheet1, "Q22", "S22"); err != nil {
		return err
	}

	applyProjectBorders(f, sb)

	// values
	for addr, v := range lbl.CellValues() {
		if err := setCellValue(f, addr, v); err != nil {
			return err
		}
	}

	// fonts (display concern, applied here in Python)
	sb.setFontCell("B2", fontTimes)
	sb.setFontCell("B3", fontTimes)
	sb.setFontCell("B4", fontTitle)
	sb.setFontCell("B5", fontTimes)
	sb.setFontCell("B6", fontTimes)
	sb.setFontCell("Q21", fontQ21)
	sb.setFontCell("Q22", fontQ22R23)
	sb.setFontCell("R23", fontQ22R23)
	sb.setFontCell("S23", fontTimes)

	// print_area A1:T24
	return setPrintArea(f, sheet1)
}

// applyProjectBorders replays _apply_project_borders in exact source order.
func applyProjectBorders(f *excelize.File, sb *styleBuilder) {
	sb.setBorderRange("B7:M7", map[string]string{"top": "thin"})
	sb.setBorderRange("B7:B17", map[string]string{"left": "thin"})
	sb.setBorderRange("M7:M17", map[string]string{"right": "thin"})
	sb.setBorderRange("B17:M17", map[string]string{"bottom": "thin"})

	sb.setBorderCell("B17", map[string]string{"left": "thin", "bottom": "thin"})
	sb.setBorderCell("M17", map[string]string{"right": "thin", "bottom": "thin"})

	// N,O,P widths 0.375 (Python: columns 14..16). N already set; set O,P too.
	for _, col := range []string{"N", "O", "P"} {
		_ = f.SetColWidth(sheet1, col, col, 0.375)
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

// applyCellValues writes equipment cell values and assigns FONT_TITLE/FONT_TIMES.
func applyCellValues(f *excelize.File, sb *styleBuilder, lbl label.Label) error {
	title := lbl.TitleCell()
	for addr, v := range lbl.CellValues() {
		if err := setCellValue(f, addr, v); err != nil {
			return err
		}
		if addr == title {
			sb.setFontCell(addr, fontTitle)
		} else {
			sb.setFontCell(addr, fontTimes)
		}
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
func (g *Generator) createAdditionalSheets(f *excelize.File, docType string, docCount int) error {
	srcIdx, err := f.GetSheetIndex(sheet1)
	if err != nil {
		return err
	}
	for i := 2; i <= docCount; i++ {
		name := fmt.Sprintf("Sheet %d", i)
		toIdx, nerr := f.NewSheet(name)
		if nerr != nil {
			return nerr
		}
		if cerr := f.CopySheet(srcIdx, toIdx); cerr != nil {
			return cerr
		}

		if err := f.SetCellStr(name, "B5", fmt.Sprintf("%d/%d", i, docCount)); err != nil {
			return err
		}
		if docType == "2" {
			if err := f.SetCellStr(name, "S23", fmt.Sprintf("%d/%d", i, docCount)); err != nil {
				return err
			}
			if err := setPrintArea(f, name); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyQRCodes sets B–M column width and embeds one 75x75px QR per sheet,
// centered in the lower bordered box (B8:M17 equipment, B7:M17 project).
func (g *Generator) applyQRCodes(f *excelize.File, docType string, binder int, qrPNGs [][]byte) error {
	cfg := label.GetQRConfig(binder)
	sheets := f.GetSheetList()

	if qrPNGs != nil && len(qrPNGs) < len(sheets) {
		return fmt.Errorf("qrPNGs has %d entries but %d sheets expected", len(qrPNGs), len(sheets))
	}

	for _, s := range sheets {
		if err := f.SetColWidth(s, "B", "M", cfg.ColumnWidth); err != nil {
			return err
		}
	}

	// Center the QR in the lower bordered box (B8:M17 equipment, B7:M17 project).
	anchorCell, offX, offY := qrCenterAnchor(docType, cfg.ColumnWidth)

	for idx, s := range sheets {
		if qrPNGs == nil {
			continue
		}
		pngBytes := qrPNGs[idx]
		cfgImg, _, derr := image.DecodeConfig(bytes.NewReader(pngBytes))
		if derr != nil {
			return fmt.Errorf("decode QR PNG for %s: %w", s, derr)
		}
		scaleX := 75.0 / float64(cfgImg.Width)
		scaleY := 75.0 / float64(cfgImg.Height)
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
