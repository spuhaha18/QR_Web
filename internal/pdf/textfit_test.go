package pdf

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"testing"
)

func TestFamilyForRoutesScripts(t *testing.T) {
	for r, want := range map[rune]string{
		'A': fontFamily, '9': fontFamily, '-': fontFamily, '(': fontFamily,
		'한': batangFamily, '글': batangFamily, '漢': batangFamily, '１': batangFamily,
	} {
		if got := familyFor(r); got != want {
			t.Errorf("familyFor(%q) = %s, want %s", r, got, want)
		}
	}
}

func TestSplitRunsMixed(t *testing.T) {
	runs := splitRuns("바탕체 Times 123")
	if len(runs) != 2 {
		t.Fatalf("runs = %+v, want 2 runs", runs)
	}
	if runs[0].family != batangFamily || runs[0].text != "바탕체 " {
		t.Errorf("run0 = %+v", runs[0])
	}
	if runs[1].family != fontFamily || runs[1].text != "Times 123" {
		t.Errorf("run1 = %+v", runs[1])
	}
}

func TestFitTextShortKeepsBaseSize(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	size, lines := fitText(doc, "짧은 제목", 16, 40, 70)
	if size != 16 {
		t.Errorf("size = %v, want 16", size)
	}
	if len(lines) != 1 {
		t.Errorf("lines = %v, want 1", lines)
	}
}

func TestFitTextLongShrinksAndFits(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	long := strings.Repeat("아주 긴 문서 제목 Very Long Title ", 30)
	boxW, boxH := 40.0, 70.0
	size, lines := fitText(doc, long, 16, boxW, boxH)
	if size >= 16 {
		t.Errorf("size = %v, want < 16", size)
	}
	if got := float64(len(lines)) * lineHeightMM(size); got > boxH-textPadMM {
		t.Errorf("wrapped block %vmm exceeds box %vmm", got, boxH)
	}
	// no truncation: content preserved modulo spaces trimmed at line breaks
	joined := strings.ReplaceAll(strings.Join(lines, ""), " ", "")
	orig := strings.ReplaceAll(long, " ", "")
	if joined != orig {
		t.Error("text content lost during wrap")
	}
	// every line fits the width
	for _, line := range lines {
		if w := textWidth(doc, size, line); w > boxW-textPadMM+0.001 {
			t.Errorf("line %q width %v exceeds %v", line, w, boxW-textPadMM)
		}
	}
}

// TestWrapTextMeasuresConsecutiveWhitespaceLikeSplitRuns guards against the
// wrapText measurement loop diverging from splitRuns' sticky-family rule for
// runs of whitespace adjacent to CJK text (double/triple spaces, tabs). Wrap
// at a width that comfortably fits the whole string so no break occurs, then
// the single returned line must be exactly the input and its measured width
// (walking runs the same way drawTextBox/textWidth does) must match — if the
// wrap loop under- or over-measures a run of spaces, this line's accumulated
// lineW would drift from what splitRuns/textWidth compute for the same text,
// and a tight-width wrap (checked below) would place breaks in the wrong
// spot or let a line's rendered width exceed the box.
func TestWrapTextMeasuresConsecutiveWhitespaceLikeSplitRuns(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	const size = 16.0

	// All four inputs: wide enough that wrapText never breaks, so the
	// single returned line must be the untouched input and its width must
	// match textWidth(s) exactly — a basic content/measurement sanity
	// check that holds regardless of where a break would land.
	for _, s := range []string{"한  글", "한   글", "한\t글", "바탕  Times"} {
		lines := wrapText(doc, size, s, 500)
		if len(lines) != 1 || lines[0] != s {
			t.Fatalf("wrapText(%q) at wide width = %v, want single unbroken line", s, lines)
		}
		if w := textWidth(doc, size, lines[0]); w != textWidth(doc, size, s) {
			t.Errorf("textWidth(wrapText(%q)) = %v, want %v", s, w, textWidth(doc, size, s))
		}
	}

	// CJK-adjacent whitespace runs (double/triple space, tab): scan maxW
	// across the whole range up to (and a bit past) the string's real
	// width. The bug this guards against bites right at a break decision —
	// the old code kept measuring a trailing whitespace rune too narrow
	// (using the *previous rune's* family instead of the sticky run
	// family), so it could decide "still fits" for a maxW where the true,
	// splitRuns-consistent width already doesn't. A single hand-picked
	// width can miss that window; scanning finds it wherever it falls.
	// (Excludes "바탕  Times": a bare word with no internal space, like
	// "Times", can force a break with no space to cut at — a separate,
	// pre-existing greedy-wrap limitation unrelated to family stickiness
	// that would confound this scan.)
	for _, s := range []string{"한  글", "한   글", "한\t글"} {
		full := textWidth(doc, size, s)
		// Start above the widest single rune's own width: below that, no
		// wrap algorithm can satisfy the invariant (a lone glyph can't be
		// split further), which is expected ("no character is ever
		// dropped") and not the bug under test.
		maxRuneW := 0.0
		for _, r := range s {
			if w := textWidth(doc, size, string(r)); w > maxRuneW {
				maxRuneW = w
			}
		}
		for maxW := maxRuneW + 0.01; maxW <= full+2; maxW += 0.05 {
			for _, line := range wrapText(doc, size, s, maxW) {
				if w := textWidth(doc, size, line); w > maxW+0.001 {
					t.Fatalf("wrapText(%q, maxW=%.2f) line %q width %v exceeds maxW", s, maxW, line, w)
				}
			}
		}
	}
}

// TestWrapTextRechecksFitAfterFlush guards against the flush-then-append-
// unconditionally bug: after flush() cuts the line at the last space, the
// remainder (text after that space) plus the triggering rune can still
// exceed maxW - the fit check must loop, not run once. Reproduces with an
// ASCII-then-CJK mix (post-flush remainder is non-empty ASCII) and with a
// leading space and no internal break point (post-flush remainder is the
// literal space, forcing a second flush to empty the line). Swept across a
// wide range of maxW so the regression isn't tied to one lucky width.
func TestWrapTextRechecksFitAfterFlush(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	const size = 16.0

	for _, s := range []string{"abc 한국어단어테스트", " 한글", strings.Repeat("가", 40)} {
		// Below the widest single rune's own width, no wrap algorithm can
		// satisfy the invariant (a lone glyph can't be split further); that
		// is the pre-existing "no character is ever dropped" limitation,
		// not the flush-recheck bug under test here (same exclusion as
		// TestWrapTextMeasuresConsecutiveWhitespaceLikeSplitRuns above).
		maxRuneW := 0.0
		for _, r := range s {
			if w := textWidth(doc, size, string(r)); w > maxRuneW {
				maxRuneW = w
			}
		}
		start := 3.0
		if maxRuneW+0.01 > start {
			start = maxRuneW + 0.01
		}
		for maxW := start; maxW <= 40.0; maxW += 0.5 {
			for _, line := range wrapText(doc, size, s, maxW) {
				if w := textWidth(doc, size, line); w > maxW+0.001 {
					t.Errorf("wrapText(%q, maxW=%.1f) line %q width %v exceeds maxW", s, maxW, line, w)
				}
			}
		}
	}
}

// TestFitTextNarrowBoxGuaranteesWidthFit exercises fitText with a very
// tight box so the wrap-then-shrink loop is forced through several
// iterations, and asserts the explicit width guarantee: every returned
// line fits boxW-textPadMM at the returned size (not just the height
// budget).
func TestFitTextNarrowBoxGuaranteesWidthFit(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	boxW, boxH := 8.0, 20.0
	size, lines := fitText(doc, "abc 한국어단어테스트", 16, boxW, boxH)
	availW := boxW - textPadMM
	for _, line := range lines {
		if w := textWidth(doc, size, line); w > availW+0.001 {
			t.Errorf("fitText line %q width %v exceeds availW %v at size %v", line, w, availW, size)
		}
	}
}

// decompressedContent inflates every stream in a PDF and returns the
// concatenated content (test helper for inspecting drawing operators).
func decompressedContent(t *testing.T, pdfBytes []byte) string {
	t.Helper()
	var out []byte
	rest := pdfBytes
	for {
		i := bytes.Index(rest, []byte("stream\n"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream\n"):]
		j := bytes.Index(rest, []byte("endstream"))
		if j < 0 {
			break
		}
		zr, err := zlib.NewReader(bytes.NewReader(rest[:j]))
		if err == nil {
			d, _ := io.ReadAll(zr)
			out = append(out, d...)
			_ = zr.Close()
		}
		rest = rest[j:]
	}
	return string(out)
}

// Batang has no bold face, so Korean runs must be emulated-bold: drawn in
// fill+stroke mode (2 Tr) with the mode reset afterwards. Latin-only text
// uses the real Times bold and must never enable stroke mode.
func TestDrawTextBoxBatangSyntheticBold(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	drawTextBox(doc, 10, 10, 60, 20, "한글", 12)
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	content := decompressedContent(t, buf.Bytes())
	if !strings.Contains(content, "2 Tr") {
		t.Error("Korean text: stroke mode (2 Tr) not enabled")
	}
	if !strings.Contains(content, "0 Tr") {
		t.Error("stroke mode not reset to 0 Tr")
	}

	doc2 := newDoc()
	doc2.AddPage()
	drawTextBox(doc2, 10, 10, 60, 20, "Latin only", 12)
	var buf2 bytes.Buffer
	if err := doc2.Output(&buf2); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(decompressedContent(t, buf2.Bytes()), "2 Tr") {
		t.Error("Latin-only text must not use stroke mode")
	}
}
