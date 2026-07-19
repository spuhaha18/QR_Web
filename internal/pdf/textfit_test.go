package pdf

import (
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
