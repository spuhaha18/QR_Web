// Package excel ports excel_generator.py (openpyxl) to Go excelize, reproducing
// the equipment/project label layout with visual parity: cell values, borders,
// fonts, alignment, merges, dimensions, multi-sheet i/N, and the 75x75px QR
// anchor.
//
// styles.go owns the per-cell style synthesis. openpyxl applies border / font /
// alignment as independent, *replaceable* per-cell attributes (each
// `cell.border = Border(...)` fully replaces the prior border). excelize, by
// contrast, attaches a single style ID per cell. We therefore accumulate a
// per-cell border/font state map while replaying the Python layout passes in the
// same order, then flush one synthesized excelize style per cell.
package excel

import (
	"fmt"
	"sort"

	"github.com/xuri/excelize/v2"

	"qrweb/internal/label"
)

// borderStyle maps openpyxl border_style names to excelize Style integers.
//   - "thin"   -> 1
//   - "medium" -> 2
var borderStyleID = map[string]int{
	"thin":   1,
	"medium": 2,
}

// fontKindFor maps a label's domain font intent to the concrete excelize font
// kind below. The label package owns which cell gets which intent; this is the
// renderer's single translation point.
func fontKindFor(cf label.CellFont) fontKind {
	switch cf {
	case label.CellFontTitle:
		return fontTitle
	case label.CellFontHeading:
		return fontQ21
	case label.CellFontSub:
		return fontQ22R23
	default:
		return fontTimes
	}
}

// fontKind identifies which font (if any) a cell carries.
type fontKind int

const (
	fontNone   fontKind = iota // no explicit font (Calibri default, like openpyxl)
	fontTimes                  // FONT_TIMES: Times New Roman 12 bold
	fontTitle                  // FONT_TITLE: Times New Roman 16 bold (B4)
	fontQ21                    // project Q21: Times New Roman 20 bold
	fontQ22R23                 // project Q22/R23: Times New Roman 13 bold
)

// cellState is the accumulated style of a single cell during layout replay.
//
// sides holds the FINAL border side->style assignment. Because openpyxl replaces
// the whole border on each `cell.border = Border(...)`, a border pass that
// touches a cell resets sides entirely (see setBorder).
type cellState struct {
	sides map[string]string // "left"|"right"|"top"|"bottom" -> "thin"|"medium"
	font  fontKind
}

// styleBuilder collects per-cell state and synthesizes excelize style IDs,
// caching by a stable signature so identical cells share one ID.
type styleBuilder struct {
	cells map[string]*cellState
	cache map[string]int // signature -> excelize style ID
	f     *excelize.File
}

func newStyleBuilder(f *excelize.File) *styleBuilder {
	return &styleBuilder{
		cells: map[string]*cellState{},
		cache: map[string]int{},
		f:     f,
	}
}

func (b *styleBuilder) cell(addr string) *cellState {
	cs, ok := b.cells[addr]
	if !ok {
		cs = &cellState{}
		b.cells[addr] = cs
	}
	return cs
}

// setBorderRange replays one openpyxl border pass over a rectangular range.
// `sides` is the side->style map applied (replacing each cell's prior border),
// mirroring `for cell in ws[range]: cell.border = Border(**sides)`.
func (b *styleBuilder) setBorderRange(rng string, sides map[string]string) {
	c1, r1, c2, r2 := mustRange(rng)
	for c := c1; c <= c2; c++ {
		for r := r1; r <= r2; r++ {
			addr, _ := excelize.CoordinatesToCellName(c, r)
			b.setBorderCell(addr, sides)
		}
	}
}

// setBorderCell sets a single cell's border, fully replacing the prior one
// (openpyxl replace semantics).
func (b *styleBuilder) setBorderCell(addr string, sides map[string]string) {
	cs := b.cell(addr)
	cp := make(map[string]string, len(sides))
	for k, v := range sides {
		cp[k] = v
	}
	cs.sides = cp
}

// setFontCell assigns a font kind to a cell.
func (b *styleBuilder) setFontCell(addr string, fk fontKind) {
	b.cell(addr).font = fk
}

// signature builds a stable key for a cell's synthesized style.
func (cs *cellState) signature() string {
	keys := make([]string, 0, len(cs.sides))
	for k := range cs.sides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sig := fmt.Sprintf("f%d|", cs.font)
	for _, k := range keys {
		sig += k + "=" + cs.sides[k] + ";"
	}
	return sig
}

// styleID returns (creating + caching as needed) the excelize style ID for a
// cell's accumulated state. Alignment is center/center/wrap for every styled
// cell (matching the global alignment pass in excel_generator.py).
func (b *styleBuilder) styleID(cs *cellState) (int, error) {
	sig := cs.signature()
	if id, ok := b.cache[sig]; ok {
		return id, nil
	}

	st := &excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	}

	switch cs.font {
	case fontTimes:
		st.Font = &excelize.Font{Family: "times new roman", Size: 12, Bold: true, Color: "000000"}
	case fontTitle:
		st.Font = &excelize.Font{Family: "times new roman", Size: 16, Bold: true, Color: "000000"}
	case fontQ21:
		st.Font = &excelize.Font{Family: "times new roman", Size: 20, Bold: true, Color: "000000"}
	case fontQ22R23:
		st.Font = &excelize.Font{Family: "times new roman", Size: 13, Bold: true, Color: "000000"}
	}

	if len(cs.sides) > 0 {
		// Deterministic order so identical signatures map to identical XML.
		order := []string{"left", "right", "top", "bottom"}
		for _, side := range order {
			style, ok := cs.sides[side]
			if !ok {
				continue
			}
			st.Border = append(st.Border, excelize.Border{
				Type:  side,
				Color: "000000",
				Style: borderStyleID[style],
			})
		}
	}

	id, err := b.f.NewStyle(st)
	if err != nil {
		return 0, err
	}
	b.cache[sig] = id
	return id, nil
}

// flush applies every accumulated cell style to the given sheet.
func (b *styleBuilder) flush(sheet string) error {
	// Sort addresses for deterministic application order.
	addrs := make([]string, 0, len(b.cells))
	for a := range b.cells {
		addrs = append(addrs, a)
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		cs := b.cells[addr]
		if len(cs.sides) == 0 && cs.font == fontNone {
			continue
		}
		id, err := b.styleID(cs)
		if err != nil {
			return err
		}
		if err := b.f.SetCellStyle(sheet, addr, addr, id); err != nil {
			return err
		}
	}
	return nil
}

// mustRange parses an "A1:B2" range into 1-based (col1,row1,col2,row2).
func mustRange(rng string) (int, int, int, int) {
	var a, c string
	for i := 0; i < len(rng); i++ {
		if rng[i] == ':' {
			a, c = rng[:i], rng[i+1:]
			break
		}
	}
	if c == "" { // single cell
		a, c = rng, rng
	}
	c1, r1, _ := excelize.CellNameToCoordinates(a)
	c2, r2, _ := excelize.CellNameToCoordinates(c)
	return c1, r1, c2, r2
}
