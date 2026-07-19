package pdf

import (
	"strings"
	"unicode"

	"github.com/go-pdf/fpdf"
)

// textPadMM keeps a hair of breathing room inside a cell box, like Excel's
// cell padding; subtracted from both the wrap width and the height budget.
const textPadMM = 0.6

// minFontPt is the shrink floor (grill Q2). Below this the text is unreadable
// anyway and the loop must terminate; real inputs never get here.
const minFontPt = 2.0

func lineHeightMM(sizePt float64) float64 { return ptToMM(sizePt) * 1.2 }

// familyFor routes a rune to its font: Times for Latin/digits/common
// punctuation, Batang for everything from the CJK blocks up (U+2E80+).
// fpdf has no glyph fallback, so this function IS the fallback policy.
func familyFor(r rune) string {
	if r < 0x2E80 {
		return fontFamily
	}
	return batangFamily
}

// run is a maximal same-family span of a single line.
type run struct {
	family string
	text   string
}

// splitRuns splits a line into runs. Spaces are neutral: they stay in the
// current run so runs only break on a real script change.
func splitRuns(s string) []run {
	var out []run
	var cur []rune
	curFam := ""
	for _, r := range s {
		fam := familyFor(r)
		if unicode.IsSpace(r) && curFam != "" {
			fam = curFam
		}
		if fam != curFam && len(cur) > 0 {
			out = append(out, run{curFam, string(cur)})
			cur = cur[:0]
		}
		curFam = fam
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		out = append(out, run{curFam, string(cur)})
	}
	return out
}

// textWidth measures a mixed-script line: per-run font switch + width sum.
func textWidth(doc *fpdf.Fpdf, sizePt float64, s string) float64 {
	w := 0.0
	for _, ru := range splitRuns(s) {
		doc.SetFont(ru.family, "B", sizePt)
		w += doc.GetStringWidth(ru.text)
	}
	return w
}

// runeWidth measures one rune in its own family at sizePt. prevFamily is the
// sticky run family carried from the runes measured so far in the line
// (mirroring splitRuns' curFam) — for whitespace it wins over familyFor so
// the space is measured in the same family splitRuns will render it in
// (see TestSplitRunsMixed). wrapText must budget width identically or lines
// silently overflow their box. Returns the width and the resolved family,
// the latter to be threaded back in as prevFamily for the next rune.
func runeWidth(doc *fpdf.Fpdf, sizePt float64, r rune, prevFamily string) (float64, string) {
	fam := familyFor(r)
	if unicode.IsSpace(r) && prevFamily != "" {
		fam = prevFamily
	}
	doc.SetFont(fam, "B", sizePt)
	return doc.GetStringWidth(string(r)), fam
}

// wrapText greedily breaks s into lines no wider than maxW. Breaks at the
// last space when one exists in the current line, otherwise at the current
// rune (CJK breaks anywhere, like Excel's wrap). Trailing spaces at a break
// are trimmed; no character is ever dropped otherwise.
func wrapText(doc *fpdf.Fpdf, sizePt float64, s string, maxW float64) []string {
	var lines []string
	var line []rune
	lineW := 0.0
	lastSpace := -1
	var prevFamily string
	flush := func(cut int) {
		lines = append(lines, strings.TrimRight(string(line[:cut]), " "))
		line = append([]rune{}, line[cut:]...)
		lineW = 0
		lastSpace = -1
		prevFamily = ""
		for i, lr := range line {
			var w float64
			w, prevFamily = runeWidth(doc, sizePt, lr, prevFamily)
			lineW += w
			if lr == ' ' {
				lastSpace = i
			}
		}
	}
	for _, r := range s {
		rw, fam := runeWidth(doc, sizePt, r, prevFamily)
		// Loop, not if: one flush may leave a non-empty remainder (text
		// after the last space) that still doesn't fit alongside r, so
		// re-check after every flush until it fits or the line is empty.
		for lineW+rw > maxW && len(line) > 0 {
			cut := len(line)
			if lastSpace >= 0 {
				cut = lastSpace + 1
			}
			flush(cut)
			rw, fam = runeWidth(doc, sizePt, r, prevFamily)
		}
		if r == ' ' {
			lastSpace = len(line)
		}
		line = append(line, r)
		lineW += rw
		prevFamily = fam
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// fitText wraps at baseSizePt and shrinks in 0.5pt steps until the block
// fits the box (grill Q2: floor 2pt, never truncate). Acceptance requires
// BOTH the height budget and every line's width to fit — wrapText breaks
// at rune granularity, so a single rune wider than availW (rare, but
// possible at large sizes) can still overflow a line; the explicit width
// check makes the box-fit guarantee direct instead of relying on the
// height shrink loop to incidentally shrink it away too.
func fitText(doc *fpdf.Fpdf, text string, baseSizePt, boxW, boxH float64) (float64, []string) {
	availW, availH := boxW-textPadMM, boxH-textPadMM
	size := baseSizePt
	for {
		lines := wrapText(doc, size, text, availW)
		fits := float64(len(lines))*lineHeightMM(size) <= availH
		if fits {
			for _, l := range lines {
				if textWidth(doc, size, l) > availW {
					fits = false
					break
				}
			}
		}
		if fits || size <= minFontPt {
			return size, lines
		}
		size -= 0.5
	}
}

// drawTextBox renders text centered (both axes) in the box, shrinking to
// fit; each line is drawn run-by-run with per-script font switches. Empty
// text draws nothing.
func drawTextBox(doc *fpdf.Fpdf, x, y, w, h float64, text string, baseSizePt float64) {
	if text == "" {
		return
	}
	size, lines := fitText(doc, text, baseSizePt, w, h)
	lh := lineHeightMM(size)
	top := y + (h-float64(len(lines))*lh)/2.0
	for i, line := range lines {
		cx := x + (w-textWidth(doc, size, line))/2.0
		cy := top + float64(i)*lh
		for _, ru := range splitRuns(line) {
			doc.SetFont(ru.family, "B", size)
			rw := doc.GetStringWidth(ru.text)
			doc.SetXY(cx, cy)
			doc.CellFormat(rw, lh, ru.text, "", 0, "L", false, 0, "")
			cx += rw
		}
	}
}
