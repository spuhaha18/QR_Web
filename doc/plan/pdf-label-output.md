# PDF 라벨 출력 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 라벨 다운로드를 xlsx에서 PDF로 완전 대체 — 라벨 물리 크기 동일(Excel 100% 환산 mm), A4 세로 한 페이지에 여러 라벨 shelf 배치, 재단 점선, 모든 값 셀 자동 폰트 축소(잘림 없음).

**Architecture:** `internal/pdf` 신설(geometry/textfit/renderer/packer/generator). 도메인(`internal/label`)과 QR(`internal/qr`)은 무변경. `internal/excel`은 이번 계획에서 삭제하지 않음(검증 후 후속 정리). 핸들러는 `excel.Generator` 대신 `pdf.Generator`를 호출.

**Tech Stack:** Go 1.26, `github.com/go-pdf/fpdf` (UTF-8 TTF 임베드 + glyph 폴백), 폰트 = TIMES.TTF/TIMESBD.TTF + BATANG.TTC face0 추출본.

## Global Constraints

- Go 툴체인: `~/.local/go/bin/go` (PATH에 없으면 절대경로 사용). 전체 테스트: `~/.local/go/bin/go test ./internal/...`
- 스펙: `doc/spec/pdf-label-output.md` — 결정 사항 표 준수.
- 라벨 크기 환산 공식: 열너비(char)→px는 OOXML `trunc(((256w+trunc(128/7))/256)*7)`, px→mm은 `px*25.4/96`, 행높이 pt→mm은 `pt*25.4/72`. 이 값이 "크기 동일"의 정의.
- 텍스트 잘림(truncation) 절대 금지 — 박스 초과 시 폰트 축소(0.5pt 단위, 하한 2pt).
- 모든 라벨 텍스트는 bold, family "times"(한글 glyph는 batang 폴백). 크기: Body 12 / Title 16 / Heading 20 / Sub 13pt.
- 한국어 에러 메시지·기존 JSON 계약(success/message/filename/file_base64/content_type) 유지.
- 커밋 메시지 끝: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`

## 고정 기대값 (스냅샷 테스트 기준)

| 조각 | 값 |
|------|-----|
| 메인 폭 (binder 1/3/5/7cm) | 17.4625 / 23.8125 / 30.1625 / 42.8625 mm (px: 66/90/114/162) |
| 메인 높이 (공통) | 153.9875 mm (436.5pt) |
| 보조 표 (과제) | 96.3083 × 40.48125 mm (364px × 114.75pt) |
| QR 한 변 | 19.84375 mm (75px) |
| 열 px | A/N/P/T=3, Q=57, R=244, S=57; B–M은 binder별 5/7/9/13 |

---

### Task 1: 폰트 자산 준비 + embed

**Files:**
- Create: `internal/pdf/fonts/` (times.ttf, timesbd.ttf, batang.ttf 복사/추출)
- Create: `internal/pdf/fonts.go`
- Test: `internal/pdf/fonts_test.go`

**Interfaces:**
- Produces: `pdf.fontTimes, pdf.fontTimesBold, pdf.fontBatang []byte` (패키지 내부 변수, go:embed)

- [ ] **Step 1: BATANG.TTC face0 추출 + Times 복사**

```bash
cd /home/spuhaha18/Project/QR_Web
mkdir -p internal/pdf/fonts
uv run --with fonttools python -c "
from fontTools.ttLib import TTFont
f = TTFont('fonts/BATANG.TTC', fontNumber=0)
f.save('internal/pdf/fonts/batang.ttf')
print('faces ok')
"
cp fonts/TIMES.TTF internal/pdf/fonts/times.ttf
cp fonts/TIMESBD.TTF internal/pdf/fonts/timesbd.ttf
ls -la internal/pdf/fonts/
```

Expected: batang.ttf 생성(수 MB), times 2종 복사됨.

- [ ] **Step 2: 실패 테스트 작성** — `internal/pdf/fonts_test.go`

```go
package pdf

import "testing"

// TTF 파일은 sfnt version 0x00010000으로 시작한다.
func TestEmbeddedFontsAreTTF(t *testing.T) {
	for name, b := range map[string][]byte{
		"times": fontTimes, "timesbd": fontTimesBold, "batang": fontBatang,
	} {
		if len(b) < 4 {
			t.Fatalf("%s: empty embed", name)
		}
		if !(b[0] == 0 && b[1] == 1 && b[2] == 0 && b[3] == 0) {
			t.Errorf("%s: not a TTF (magic %x)", name, b[:4])
		}
	}
}
```

- [ ] **Step 3: 실패 확인**

Run: `~/.local/go/bin/go test ./internal/pdf/`
Expected: FAIL (undefined: fontTimes)

- [ ] **Step 4: 구현** — `internal/pdf/fonts.go`

```go
// Package pdf renders label PDFs: same physical label size as the Excel
// 100%-scale printout, packed several-per-A4-page with cut guides.
package pdf

import _ "embed"

// User-supplied MS fonts (see doc/spec/pdf-label-output.md — licensing note):
// Times New Roman regular/bold, plus Batang face 0 extracted from BATANG.TTC.
var (
	//go:embed fonts/times.ttf
	fontTimes []byte
	//go:embed fonts/timesbd.ttf
	fontTimesBold []byte
	//go:embed fonts/batang.ttf
	fontBatang []byte
)
```

- [ ] **Step 5: 통과 확인 후 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS

```bash
git add internal/pdf/ && git commit -m "feat(pdf): embed Times/Batang fonts (Batang face0 from TTC)"
```

---

### Task 2: fpdf 의존성 + 폰트 로딩/폴백 스모크

**Files:**
- Create: `internal/pdf/doc.go` (newDoc 헬퍼)
- Test: `internal/pdf/doc_test.go`

**Interfaces:**
- Produces: `newDoc() *fpdf.Fpdf` — A4 세로 mm 단위, "times" regular/bold 등록 + batang 폴백 설정 완료 상태.
- 이후 모든 태스크는 family `"times"` style `"B"`만 사용.

- [ ] **Step 1: 의존성 추가 + API 확인**

```bash
~/.local/go/bin/go get github.com/go-pdf/fpdf@latest
~/.local/go/bin/go doc github.com/go-pdf/fpdf.Fpdf.SetFallbackFonts
~/.local/go/bin/go doc github.com/go-pdf/fpdf.Fpdf.AddUTF8FontFromBytes
~/.local/go/bin/go doc github.com/go-pdf/fpdf.Fpdf.SplitText
```

Expected: 세 메서드 시그니처 출력. **SetFallbackFonts가 없으면 STOP — 계획 수정 필요(글리프 단위 수동 분할로 전환), 진행 전 보고.**

- [ ] **Step 2: 실패 테스트** — `internal/pdf/doc_test.go`

```go
package pdf

import (
	"bytes"
	"testing"
)

func TestNewDocRendersKoreanAndLatin(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	doc.SetFont(fontFamily, "B", 12)
	doc.SetXY(10, 10)
	doc.CellFormat(100, 10, "바탕체 Times 123", "", 0, "C", false, 0, "")
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Errorf("not a PDF: %x", buf.Bytes()[:8])
	}
}
```

- [ ] **Step 3: 실패 확인** — Run: `~/.local/go/bin/go test ./internal/pdf/` → FAIL (undefined: newDoc, fontFamily)

- [ ] **Step 4: 구현** — `internal/pdf/doc.go`

```go
package pdf

import "github.com/go-pdf/fpdf"

// fontFamily is the single family every label text uses: Times New Roman with
// Batang registered as glyph-level fallback for Korean.
const fontFamily = "times"

const batangFamily = "batang"

// newDoc returns an A4-portrait mm-unit document with fonts registered and
// fallback wired. Auto page break off: the packer owns page boundaries.
func newDoc() *fpdf.Fpdf {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetAutoPageBreak(false, 0)
	doc.AddUTF8FontFromBytes(fontFamily, "", fontTimes)
	doc.AddUTF8FontFromBytes(fontFamily, "B", fontTimesBold)
	// Batang has no bold face; register the same bytes for "B" so bold text
	// falls back to regular-weight Batang glyphs (Excel renders the same way).
	doc.AddUTF8FontFromBytes(batangFamily, "", fontBatang)
	doc.AddUTF8FontFromBytes(batangFamily, "B", fontBatang)
	doc.SetFallbackFonts([]string{batangFamily}, false)
	return doc
}
```

(`SetFallbackFonts` 두 번째 인자는 Step 1의 go doc 출력에 맞춰 조정 — 인자 하나면 제거.)

- [ ] **Step 5: 통과 확인 + 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS

```bash
git add go.mod go.sum internal/pdf/ && git commit -m "feat(pdf): fpdf doc factory with Times+Batang fallback"
```

---

### Task 3: geometry — 조각 크기·그리드 좌표 (mm)

**Files:**
- Create: `internal/pdf/geometry.go`
- Test: `internal/pdf/geometry_test.go`

**Interfaces:**
- Consumes: `label.BinderSize.ColumnWidth()`, `label.DocType.Layout()` (기존)
- Produces:
  - `mainSize(b label.BinderSize) (w, h float64)` — mm
  - `auxSize() (w, h float64)` — mm
  - `mainGrid(b label.BinderSize) grid`, `auxGrid() grid`
  - `type grid struct { colX []float64; rowY []float64 }` — colX[i]=열 i 왼쪽 x(mm, 조각 로컬), 마지막 원소는 오른쪽 끝. rowY 동일 개념. 메인: colX는 A..N(15개 원소), rowY는 행 1..18(19개 원소). 보조: P..T(6개), 행 20..24(6개).
  - `qrSizeMM` 상수 = 19.84375

- [ ] **Step 1: 실패 테스트** — `internal/pdf/geometry_test.go`

```go
package pdf

import (
	"math"
	"testing"

	"qrweb/internal/label"
)

func almost(t *testing.T, got, want float64, msg string) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Errorf("%s = %.5f, want %.5f", msg, got, want)
	}
}

func TestMainSizeSnapshot(t *testing.T) {
	for _, tc := range []struct {
		binder int
		w      float64
	}{{1, 17.4625}, {3, 23.8125}, {5, 30.1625}, {7, 42.8625}} {
		b, err := label.ParseBinderSize(itoa(tc.binder), label.DocTypeEquipment)
		if err != nil {
			t.Fatal(err)
		}
		w, h := mainSize(b)
		almost(t, w, tc.w, "width")
		almost(t, h, 153.9875, "height")
	}
}

func itoa(n int) string { return string(rune('0' + n)) }

func TestAuxSizeSnapshot(t *testing.T) {
	w, h := auxSize()
	almost(t, w, 96.30833333, "aux width")
	almost(t, h, 40.48125, "aux height")
}

func TestMainGridEdges(t *testing.T) {
	b, _ := label.ParseBinderSize("7", label.DocTypeEquipment)
	g := mainGrid(b)
	if len(g.colX) != 15 || len(g.rowY) != 19 {
		t.Fatalf("grid dims: cols %d rows %d", len(g.colX), len(g.rowY))
	}
	w, h := mainSize(b)
	almost(t, g.colX[14], w, "last colX == width")
	almost(t, g.rowY[18], h, "last rowY == height")
	// col A = 3px = 0.79375mm
	almost(t, g.colX[1], 0.79375, "col A right edge")
}
```

- [ ] **Step 2: 실패 확인** — Run: `~/.local/go/bin/go test ./internal/pdf/` → FAIL (undefined: mainSize …)

- [ ] **Step 3: 구현** — `internal/pdf/geometry.go`

```go
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
```

- [ ] **Step 4: 통과 확인 + 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS

```bash
git add internal/pdf/ && git commit -m "feat(pdf): geometry — Excel-unit to mm grids, snapshot-locked sizes"
```

---

### Task 4: textfit — wrap + 자동 폰트 축소

**Files:**
- Create: `internal/pdf/textfit.go`
- Test: `internal/pdf/textfit_test.go`

**Interfaces:**
- Consumes: `newDoc()`, `fontFamily`
- Produces:
  - `fitText(doc *fpdf.Fpdf, text string, baseSizePt, boxW, boxH float64) (sizePt float64, lines []string)` — SplitText로 wrap, 넘치면 0.5pt씩 축소(하한 2pt). 호출 후 doc의 현재 폰트는 (fontFamily, "B", sizePt).
  - `lineHeightMM(sizePt float64) float64` = ptToMM(sizePt)*1.2
  - `drawTextBox(doc *fpdf.Fpdf, x, y, w, h float64, text string, baseSizePt float64)` — fitText 후 수직·수평 중앙 정렬로 각 줄 CellFormat.
  - `textPadMM` 상수 = 0.6 (좌우/상하 합산 여유)

- [ ] **Step 1: 실패 테스트** — `internal/pdf/textfit_test.go`

```go
package pdf

import (
	"strings"
	"testing"
)

func TestFitTextShortKeepsBaseSize(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	size, lines := fitText(doc, "짧은 제목", 16, 40, 70)
	if size != 16 {
		t.Errorf("size = %v, want 16", size)
	}
	if len(lines) == 0 {
		t.Error("no lines")
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
	// no truncation: total rune count preserved (SplitText may trim spaces at breaks)
	joined := strings.ReplaceAll(strings.Join(lines, ""), " ", "")
	orig := strings.ReplaceAll(long, " ", "")
	if joined != orig {
		t.Error("text content lost during wrap")
	}
}
```

- [ ] **Step 2: 실패 확인** — Run: `~/.local/go/bin/go test ./internal/pdf/` → FAIL

- [ ] **Step 3: 구현** — `internal/pdf/textfit.go`

```go
package pdf

import "github.com/go-pdf/fpdf"

// textPadMM keeps a hair of breathing room inside a cell box, like Excel's
// cell padding; subtracted from both the wrap width and the height budget.
const textPadMM = 0.6

// minFontPt is the shrink floor. Below this the text is unreadable anyway and
// the loop must terminate; in practice real inputs never get here.
const minFontPt = 2.0

func lineHeightMM(sizePt float64) float64 { return ptToMM(sizePt) * 1.2 }

// fitText wraps text into the box at baseSizePt and shrinks in 0.5pt steps
// until the wrapped block fits. Never truncates. Leaves the doc font set to
// the returned size.
func fitText(doc *fpdf.Fpdf, text string, baseSizePt, boxW, boxH float64) (float64, []string) {
	availW, availH := boxW-textPadMM, boxH-textPadMM
	size := baseSizePt
	for {
		doc.SetFont(fontFamily, "B", size)
		lines := doc.SplitText(text, availW)
		if float64(len(lines))*lineHeightMM(size) <= availH || size <= minFontPt {
			return size, lines
		}
		size -= 0.5
	}
}

// drawTextBox renders text centered (both axes) inside the box, shrinking to
// fit. Empty text draws nothing.
func drawTextBox(doc *fpdf.Fpdf, x, y, w, h float64, text string, baseSizePt float64) {
	if text == "" {
		return
	}
	size, lines := fitText(doc, text, baseSizePt, w, h)
	lh := lineHeightMM(size)
	top := y + (h-float64(len(lines))*lh)/2.0
	for i, line := range lines {
		doc.SetXY(x, top+float64(i)*lh)
		doc.CellFormat(w, lh, line, "", 0, "C", false, 0, "")
	}
}
```

- [ ] **Step 4: 통과 확인 + 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS
(SplitText의 공백 처리로 rune 보존 assert가 flaky하면 assert를 `이어붙인 길이 >= 원본 길이 - 줄수` 로 완화하되, 잘림 없음 취지는 유지.)

```bash
git add internal/pdf/ && git commit -m "feat(pdf): text wrap + auto-shrink fit, centered box drawing"
```

---

### Task 5: renderer — 메인/보조 라벨 조각 렌더

**Files:**
- Create: `internal/pdf/renderer.go`
- Test: `internal/pdf/renderer_test.go`

**Interfaces:**
- Consumes: `mainGrid/auxGrid/mainSize/auxSize/qrSizeMM`, `drawTextBox`, `label.Label`, `label.DocType.Layout()`, `label.CellFont*`
- Produces:
  - `renderMain(doc *fpdf.Fpdf, x, y float64, dt label.DocType, b label.BinderSize, lbl label.Label, marker string, qrPNG []byte, qrName string) error` — (x,y)=조각 좌상단. marker는 "i/N" (B5 교체값). qrName은 fpdf 이미지 등록용 유니크 키.
  - `renderAux(doc *fpdf.Fpdf, x, y float64, lbl label.Label, marker string)` — 과제 보조 표.
  - 폰트 크기 매핑: `sizeFor(cf label.CellFont) float64` → Body 12 / Title 16 / Heading 20 / Sub 13.
- 선 두께: thin 0.2mm, medium 0.5mm. 색 검정.

**렌더 명세 (메인 라벨, 조각 로컬 mm — g=mainGrid(b), 열 인덱스 A=0..N=13, 행 인덱스 행1=0..행18=17):**
- 외곽 medium 사각형: (0,0)–(W,H)
- 값 행 thin 사각형: 행 2..6 각각 x[g.colX[1]..g.colX[13]] × y[g.rowY[r-1]..g.rowY[r]] (r=2..6). 장비는 행 7도 포함.
- QR 박스 thin 사각형: rows `Layout().QRBoxTopRow..17` → y[g.rowY[top-1]..g.rowY[17]], x는 위와 동일 B..M.
- 값 셀 텍스트: `lbl.CellValues()`의 B2..B7 → 각 행 사각형 안에 drawTextBox. B5는 marker로 교체. B7(int)은 `strconv.Itoa`. 폰트 크기는 `lbl.CellFonts()`→sizeFor.
- QR: 박스 중앙, 19.84375mm 정사각. 박스보다 크면 좌우 대칭 오버플로하되 x는 0 미만으로 클램프(Excel 동작 미러).

**렌더 명세 (보조 표, g=auxGrid(), 열 P=0..T=4, 행 20=0..24=4):**
- 외곽 thin 사각형: (0,0)–(Wa,Ha)
- 내부 thin 사각형: (g.colX[1], g.rowY[1])–(g.colX[4], g.rowY[4])  (Q~T 폭 아님 주의: 오른쪽 경계는 T 왼쪽 = g.colX[4])
- Q22:S22 thin 사각형: (g.colX[1], g.rowY[2])–(g.colX[4], g.rowY[3])
- 값: Q21(Heading 20) → (colX[1],rowY[1])–(colX[4],rowY[2]); Q22(Sub 13) → 행22 동일 폭; R23(Sub 13) → (colX[2],rowY[3])–(colX[3],rowY[4]); S23(Body 12, marker 교체) → (colX[3],rowY[3])–(colX[4],rowY[4]).

- [ ] **Step 1: 실패 테스트** — `internal/pdf/renderer_test.go`

```go
package pdf

import (
	"bytes"
	"testing"

	"qrweb/internal/label"
	"qrweb/internal/qr"
)

func testQRPNG(t *testing.T) []byte {
	t.Helper()
	qt, err := qr.NewQRText("a|b|c")
	if err != nil {
		t.Fatal(err)
	}
	png, err := qr.CreateQRPNG(qt)
	if err != nil {
		t.Fatal(err)
	}
	return png
}

func TestRenderMainEquipmentSmoke(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	lbl := label.EquipmentLabel{
		EqNumber: "EQ-1", EqDocNumber: "DOC-1", EqDocTitle: "장비 문서 제목",
		EqDocCount: 2, EqDocDepartment: "부서", EqDocYear: 2026,
	}
	b, _ := label.ParseBinderSize("7", label.DocTypeEquipment)
	if err := renderMain(doc, 10, 10, label.DocTypeEquipment, b, lbl, "1/2", testQRPNG(t), "qr_0"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Error("not a PDF")
	}
}

func TestQRSideShrinksForNarrowBox(t *testing.T) {
	if got := qrSide(15.875, 23.8125); got != 15.875 {
		t.Errorf("qrSide narrow = %v, want 15.875", got)
	}
	if got := qrSide(22.225, 23.8125); got != qrSizeMM {
		t.Errorf("qrSide normal = %v, want %v", got, qrSizeMM)
	}
}

func TestRenderAuxProjectSmoke(t *testing.T) {
	doc := newDoc()
	doc.AddPage()
	lbl := label.ProjectLabel{
		PjtNumber: "PJ-1", PjtTestNumber: "T-1", PjtDocTitle: "과제 제목",
		PjtDocWriter: "작성자", PjtDocCount: 3,
	}
	renderAux(doc, 10, 10, lbl, "2/3")
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("empty PDF")
	}
}
```

- [ ] **Step 2: 실패 확인** — Run: `~/.local/go/bin/go test ./internal/pdf/` → FAIL

- [ ] **Step 3: 구현** — `internal/pdf/renderer.go`

```go
package pdf

import (
	"bytes"
	"strconv"

	"github.com/go-pdf/fpdf"

	"qrweb/internal/label"
)

const (
	thinMM   = 0.2
	mediumMM = 0.5
)

// sizeFor maps the domain font intent to its base point size (all bold Times).
func sizeFor(cf label.CellFont) float64 {
	switch cf {
	case label.CellFontTitle:
		return 16
	case label.CellFontHeading:
		return 20
	case label.CellFontSub:
		return 13
	default:
		return 12
	}
}

func rect(doc *fpdf.Fpdf, x1, y1, x2, y2, lineW float64) {
	doc.SetLineWidth(lineW)
	doc.Rect(x1, y1, x2-x1, y2-y1, "D")
}

// qrSide returns the QR square side: 19.84mm, shrunk to fit a smaller box
// (the 1cm binder's QR box is only ~15.9mm wide).
func qrSide(boxW, boxH float64) float64 {
	side := qrSizeMM
	if boxW < side {
		side = boxW
	}
	if boxH < side {
		side = boxH
	}
	return side
}

// cellText returns the string form of a CellValues entry (year is an int).
func cellText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

// renderMain draws one main label piece at page position (x, y). marker
// replaces the B5 (i/N) value; qrName must be unique per registered image.
func renderMain(doc *fpdf.Fpdf, x, y float64, dt label.DocType, b label.BinderSize, lbl label.Label, marker string, qrPNG []byte, qrName string) error {
	g := mainGrid(b)
	w, h := mainSize(b)
	doc.SetDrawColor(0, 0, 0)

	// Outer medium frame.
	rect(doc, x, y, x+w, y+h, mediumMM)

	// Value rows: rows 2..6 always, row 7 for equipment. x spans B..M.
	xl, xr := x+g.colX[1], x+g.colX[13]
	lastValueRow := 6
	if dt == label.DocTypeEquipment {
		lastValueRow = 7
	}
	for r := 2; r <= lastValueRow; r++ {
		rect(doc, xl, y+g.rowY[r-1], xr, y+g.rowY[r], thinMM)
	}

	// QR box.
	layout := dt.Layout()
	boxTop, boxBot := y+g.rowY[layout.QRBoxTopRow-1], y+g.rowY[17]
	rect(doc, xl, boxTop, xr, boxBot, thinMM)

	// Values. B5 carries the per-piece i/N marker.
	values := lbl.CellValues()
	fonts := lbl.CellFonts()
	rowOf := map[string]int{"B2": 2, "B3": 3, "B4": 4, "B5": 5, "B6": 6, "B7": 7}
	for addr, r := range rowOf {
		v, ok := values[addr]
		if !ok {
			continue
		}
		text := cellText(v)
		if addr == "B5" {
			text = marker
		}
		drawTextBox(doc, xl, y+g.rowY[r-1], xr-xl, g.rowY[r]-g.rowY[r-1], text, sizeFor(fonts[addr]))
	}

	// QR centered in the box, shrunk to fit when the box is smaller than
	// 19.84mm (1cm binder) — grill Q1 decision: shrink, not Excel-style
	// overflow.
	side := qrSide(xr-xl, boxBot-boxTop)
	qx := xl + ((xr - xl) - side) / 2.0
	qy := boxTop + ((boxBot - boxTop) - side) / 2.0
	opts := fpdf.ImageOptions{ImageType: "PNG"}
	doc.RegisterImageOptionsReader(qrName, opts, bytes.NewReader(qrPNG))
	doc.ImageOptions(qrName, qx, qy, side, side, false, opts, 0, "")
	return doc.Error()
}

// renderAux draws the project side table at (x, y). marker replaces S23.
func renderAux(doc *fpdf.Fpdf, x, y float64, lbl label.Label, marker string) {
	g := auxGrid()
	w, h := auxSize()
	doc.SetDrawColor(0, 0, 0)

	rect(doc, x, y, x+w, y+h, thinMM)                                     // outer
	rect(doc, x+g.colX[1], y+g.rowY[1], x+g.colX[4], y+g.rowY[4], thinMM) // inner Q21..S23
	rect(doc, x+g.colX[1], y+g.rowY[2], x+g.colX[4], y+g.rowY[3], thinMM) // Q22:S22

	values := lbl.CellValues()
	fonts := lbl.CellFonts()
	draw := func(addr string, x1, y1, x2, y2 float64, override string) {
		text := override
		if text == "" {
			text = cellText(values[addr])
		}
		drawTextBox(doc, x+x1, y+y1, x2-x1, y2-y1, text, sizeFor(fonts[addr]))
	}
	draw("Q21", g.colX[1], g.rowY[1], g.colX[4], g.rowY[2], "")
	draw("Q22", g.colX[1], g.rowY[2], g.colX[4], g.rowY[3], "")
	draw("R23", g.colX[2], g.rowY[3], g.colX[3], g.rowY[4], "")
	draw("S23", g.colX[3], g.rowY[3], g.colX[4], g.rowY[4], marker)
}
```

- [ ] **Step 4: 통과 확인 + 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS

```bash
git add internal/pdf/ && git commit -m "feat(pdf): render main/aux label pieces — borders, fitted text, centered QR"
```

---

### Task 6: packer — A4 shelf 배치 + 재단 점선

**Files:**
- Create: `internal/pdf/packer.go`
- Test: `internal/pdf/packer_test.go`

**Interfaces:**
- Consumes: 없음 (순수 기하 + fpdf)
- Produces:
  - `type piece struct { w, h float64; draw func(doc *fpdf.Fpdf, x, y float64) error }`
  - `type placed struct { page int; x, y float64 }`
  - `layoutPieces(sizes [][2]float64) []placed` — 순수 함수: A4(210×297), 여백 10mm, 간격 5mm, 행 우선 shelf. page는 0-base.
  - `packAndDraw(doc *fpdf.Fpdf, pieces []piece) error` — layout 후 페이지 추가·점선·draw 호출.
- 점선: 회색(153,153,153), 폭 0.15mm, dash 2/1.5mm, 각 조각 경계에서 2.5mm 바깥 사각형(페이지 여백 안으로 클램프). 그린 후 dash/색 원복.

- [ ] **Step 1: 실패 테스트** — `internal/pdf/packer_test.go`

```go
package pdf

import "testing"

func TestLayoutNoOverlapWithinBounds(t *testing.T) {
	// 5 main labels (7cm) + 3 aux pieces.
	sizes := [][2]float64{}
	for i := 0; i < 5; i++ {
		sizes = append(sizes, [2]float64{42.8625, 153.9875})
	}
	for i := 0; i < 3; i++ {
		sizes = append(sizes, [2]float64{96.30833333, 40.48125})
	}
	got := layoutPieces(sizes)
	if len(got) != len(sizes) {
		t.Fatalf("placed %d, want %d", len(got), len(sizes))
	}
	for i, p := range got {
		w, h := sizes[i][0], sizes[i][1]
		if p.x < 10 || p.y < 10 || p.x+w > 200 || p.y+h > 287 {
			t.Errorf("piece %d out of margins: %+v", i, p)
		}
		for j := 0; j < i; j++ {
			q := got[j]
			if q.page != p.page {
				continue
			}
			qw, qh := sizes[j][0], sizes[j][1]
			if p.x < q.x+qw && q.x < p.x+w && p.y < q.y+qh && q.y < p.y+h {
				t.Errorf("pieces %d and %d overlap", i, j)
			}
		}
	}
}

func TestLayoutPacksMultiplePerPage(t *testing.T) {
	// 4 narrow labels (3cm) must share one page: 4*23.8125 + 3*5 = 110.25 < 190.
	sizes := [][2]float64{
		{23.8125, 153.9875}, {23.8125, 153.9875}, {23.8125, 153.9875}, {23.8125, 153.9875},
	}
	got := layoutPieces(sizes)
	for i, p := range got {
		if p.page != 0 {
			t.Errorf("piece %d on page %d, want 0 (multi-label per page)", i, p.page)
		}
	}
}
```

- [ ] **Step 2: 실패 확인** — Run: `~/.local/go/bin/go test ./internal/pdf/` → FAIL

- [ ] **Step 3: 구현** — `internal/pdf/packer.go`

```go
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

// cutGuide draws a dashed gray rectangle 2.5mm outside the piece, clamped to
// the page, marking where to cut (centered in the 5mm gap).
func cutGuide(doc *fpdf.Fpdf, x, y, w, h float64) {
	const off = gapMM / 2.0
	x1, y1 := max(x-off, 1.0), max(y-off, 1.0)
	x2, y2 := min(x+w+off, pageW-1.0), min(y+h+off, pageH-1.0)
	doc.SetDrawColor(153, 153, 153)
	doc.SetLineWidth(0.15)
	doc.SetDashPattern([]float64{2, 1.5}, 0)
	doc.Rect(x1, y1, x2-x1, y2-y1, "D")
	doc.SetDashPattern([]float64{}, 0)
	doc.SetDrawColor(0, 0, 0)
}

// packAndDraw lays out the pieces and renders each with its cut guide.
func packAndDraw(doc *fpdf.Fpdf, pieces []piece) error {
	sizes := make([][2]float64, len(pieces))
	for i, p := range pieces {
		sizes[i] = [2]float64{p.w, p.h}
	}
	pages := 0
	for _, pl := range layoutPieces(sizes) {
		if pl.page >= pages {
			doc.AddPage()
			pages = pl.page + 1
		}
		// AddPage leaves the doc on the last page; placements are in page order.
	}
	// Re-walk with explicit page switches.
	placements := layoutPieces(sizes)
	for i, pl := range placements {
		doc.SetPage(pl.page + 1)
		cutGuide(doc, pl.x, pl.y, pieces[i].w, pieces[i].h)
		if err := pieces[i].draw(doc, pl.x, pl.y); err != nil {
			return err
		}
	}
	return doc.Error()
}
```

(`doc.SetPage`가 fpdf에 없으면 — `go doc github.com/go-pdf/fpdf.Fpdf.SetPage` 확인 — placements를 page별로 그룹핑해 페이지 추가 직후 그 페이지 조각을 전부 그리는 순차 루프로 변경. layoutPieces는 순서 보존이라 페이지는 단조 증가 → 단순 루프로 충분.)

- [ ] **Step 4: 통과 확인 + 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS

```bash
git add internal/pdf/ && git commit -m "feat(pdf): shelf packer with cut guides on A4"
```

---

### Task 7: generator — CreateLabelPDF 조립

**Files:**
- Create: `internal/pdf/generator.go`
- Test: `internal/pdf/generator_test.go`

**Interfaces:**
- Consumes: 위 전부 + `label.QRImageSet`, `label.GenerateTimestampFilename`
- Produces: `type Generator struct{}`, `func NewGenerator() *Generator`, `func (g *Generator) CreateLabelPDF(dt label.DocType, b label.BinderSize, lbl label.Label, qrs label.QRImageSet) (data []byte, filename string, err error)` — excel.CreateLabelExcel과 동일 형상, 확장자만 "pdf".

- [ ] **Step 1: 실패 테스트** — `internal/pdf/generator_test.go`

```go
package pdf

import (
	"bytes"
	"strings"
	"testing"

	"qrweb/internal/label"
)

func makeQRSet(t *testing.T, lbl label.Label) label.QRImageSet {
	t.Helper()
	imgs := make([][]byte, lbl.DocCount())
	for i := range imgs {
		imgs[i] = testQRPNG(t)
	}
	set, err := label.NewQRImageSet(imgs, lbl.DocCount())
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestCreateLabelPDFEquipment(t *testing.T) {
	lbl := label.EquipmentLabel{
		EqNumber: "EQ-1", EqDocNumber: "DOC-9", EqDocTitle: "제목",
		EqDocCount: 3, EqDocDepartment: "부서", EqDocYear: 2026,
	}
	b, _ := label.ParseBinderSize("3", label.DocTypeEquipment)
	data, filename, err := NewGenerator().CreateLabelPDF(label.DocTypeEquipment, b, lbl, makeQRSet(t, lbl))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("not a PDF")
	}
	if !strings.HasPrefix(filename, "DOC-9_") || !strings.HasSuffix(filename, ".pdf") {
		t.Errorf("filename = %q", filename)
	}
	// 3 labels of 23.8mm width fit one page.
	if n := bytes.Count(data, []byte("/Type /Page")); n < 1 {
		t.Errorf("page objects = %d", n)
	}
}

func TestCreateLabelPDFProjectHasAuxPieces(t *testing.T) {
	lbl := label.ProjectLabel{
		PjtNumber: "PJ-1", PjtTestNumber: "T-7", PjtDocTitle: "과제 제목",
		PjtDocWriter: "작성자", PjtDocCount: 2,
	}
	b, _ := label.ParseBinderSize("5", label.DocTypeProject)
	data, _, err := NewGenerator().CreateLabelPDF(label.DocTypeProject, b, lbl, makeQRSet(t, lbl))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty PDF")
	}
}
```

- [ ] **Step 2: 실패 확인** — Run: `~/.local/go/bin/go test ./internal/pdf/` → FAIL

- [ ] **Step 3: 구현** — `internal/pdf/generator.go`

```go
package pdf

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"

	"qrweb/internal/label"
)

// Generator builds label PDFs; the drop-in successor to excel.Generator.
type Generator struct{}

// NewGenerator returns a ready-to-use Generator.
func NewGenerator() *Generator { return &Generator{} }

// CreateLabelPDF renders N main label pieces (one per document copy, each with
// its own i/N marker and QR) plus, for project labels, N side-table pieces,
// shelf-packed onto A4 pages with cut guides.
func (g *Generator) CreateLabelPDF(dt label.DocType, b label.BinderSize, lbl label.Label, qrs label.QRImageSet) ([]byte, string, error) {
	doc := newDoc()
	docCount := lbl.DocCount()
	images := qrs.Images()

	mw, mh := mainSize(b)
	pieces := make([]piece, 0, docCount*2)
	for i := 1; i <= docCount; i++ {
		i := i
		marker := fmt.Sprintf("%d/%d", i, docCount)
		pieces = append(pieces, piece{w: mw, h: mh, draw: func(doc *fpdf.Fpdf, x, y float64) error {
			return renderMain(doc, x, y, dt, b, lbl, marker, images[i-1], fmt.Sprintf("qr_%d", i))
		}})
	}
	if dt.IsProject() {
		aw, ah := auxSize()
		for i := 1; i <= docCount; i++ {
			marker := fmt.Sprintf("%d/%d", i, docCount)
			pieces = append(pieces, piece{w: aw, h: ah, draw: func(doc *fpdf.Fpdf, x, y float64) error {
				renderAux(doc, x, y, lbl, marker)
				return doc.Error()
			}})
		}
	}

	if err := packAndDraw(doc, pieces); err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), label.GenerateTimestampFilename(lbl.DocNumber(), "pdf"), nil
}
```

- [ ] **Step 4: 통과 확인 + 커밋**

Run: `~/.local/go/bin/go test ./internal/pdf/` → PASS

```bash
git add internal/pdf/ && git commit -m "feat(pdf): CreateLabelPDF — pieces per copy, project aux tables, packed output"
```

---

### Task 8: 핸들러 전환 + 프론트 폴백명 + 전체 테스트

**Files:**
- Modify: `internal/httpx/server.go:28,46` (gen 타입 excel→pdf)
- Modify: `internal/httpx/label_handler.go` (CreateLabelExcel→CreateLabelPDF, sendXLSX→sendPDF, content type, asciiFallbackName 기본값 "label.pdf")
- Modify: `internal/httpx/handler_test.go` (xlsx 단정 → pdf 단정)
- Modify: `web/frontend/src/lib/api.ts` (주석 .xlsx→.pdf, 폴백 `'라벨.xlsx'`→`'라벨.pdf'`)

**Interfaces:**
- Consumes: `pdf.NewGenerator().CreateLabelPDF(...)` (Task 7 시그니처)
- Produces: HTTP 응답 `Content-Type: application/pdf`, 파일명 `<doc>_<ts>.pdf`, auto 모드 JSON `content_type: "application/pdf"`.

- [ ] **Step 1: 테스트 먼저 수정 (RED)** — `handler_test.go`
  - `TestCreateLabel_CorrectNFiles_ReturnsXLSX` → 이름 `..._ReturnsPDF`, 단정: `ct == "application/pdf"`, 바디 앞 4바이트 `%PDF`, 파일명 `.pdf` 접미.
  - `TestCreateLabel_KoreanDocNumber_RFC5987Filename`의 확장자 기대 `.xlsx`→`.pdf`.
  - `TestAPICreateLabel_AutoGenerates_200`: `content_type` 기대값 `"application/pdf"`, `filename` `.pdf`, `file_base64` 디코드 앞바이트 `%PDF`.

Run: `~/.local/go/bin/go test ./internal/httpx/` → FAIL (아직 xlsx 반환)

- [ ] **Step 2: 핸들러 구현 전환**

`server.go`:

```go
// import 교체: "qrweb/internal/excel" → "qrweb/internal/pdf"
gen *pdf.Generator
// ...
gen: pdf.NewGenerator(),
```

`label_handler.go` 변경점:

```go
// 두 핸들러 공통:
data, filename, err := s.gen.CreateLabelPDF(docType, binderSize, lbl, qrSet)

const pdfContentType = "application/pdf"

// sendXLSX → sendPDF로 개명, xlsxContentType → pdfContentType 사용.
func sendPDF(c *fiber.Ctx, data []byte, filename string) error {
	c.Set("Content-Disposition", contentDisposition(filename))
	c.Set(fiber.HeaderContentType, pdfContentType)
	return c.Send(data)
}

// auto 모드 JSON: "content_type": pdfContentType

// asciiFallbackName 기본 반환: "label.xlsx" → "label.pdf"
```

- [ ] **Step 3: 통과 확인**

Run: `~/.local/go/bin/go test ./internal/...`
Expected: PASS (excel 패키지 자체 테스트는 그대로 존치·통과).

- [ ] **Step 4: 프론트 폴백명**

`web/frontend/src/lib/api.ts`: 14행 주석 `.xlsx binary` → `.pdf binary`, 93행 `return '라벨.xlsx';` → `return '라벨.pdf';`

- [ ] **Step 5: 전체 빌드**

Run: `make build`
Expected: frontend 빌드 + Go 바이너리 생성 성공.

- [ ] **Step 6: 커밋**

```bash
git add internal/httpx/ web/frontend/src/lib/api.ts
git commit -m "feat(label): serve PDF instead of xlsx from both label endpoints"
```

---

### Task 9: 샘플 PDF 생성 + 수동 실측 안내

**Files:**
- Create: `/tmp/claude-1001/-home-spuhaha18-Project-QR-Web/8a929bb9-3cfb-4f2b-842c-2fdef33cb370/scratchpad/sample/` (샘플 출력, 커밋 안 함)

- [ ] **Step 1: 서버 기동 후 샘플 생성** (장비 7cm×3부, 과제 5cm×2부)

```bash
cd /home/spuhaha18/Project/QR_Web && ./bin/qrweb &  # 포트는 config 기본값
sleep 1
SCRATCH=/tmp/claude-1001/-home-spuhaha18-Project-QR-Web/8a929bb9-3cfb-4f2b-842c-2fdef33cb370/scratchpad/sample
mkdir -p "$SCRATCH"
curl -s -X POST http://localhost:8080/api/create_label -H 'Content-Type: application/json' -d '{
  "doc_type":"1","binder_size":"7","eq_number":"EQ-001","eq_doc_number":"DOC-001",
  "eq_doc_title":"밸리데이션 문서 관리 표준 운영 절차서","eq_doc_count":"3",
  "eq_doc_department":"품질보증팀","eq_doc_year":"2026"}' \
  | python3 -c "import sys,json,base64;d=json.load(sys.stdin);open('$SCRATCH/equip.pdf','wb').write(base64.b64decode(d['file_base64']));print(d['filename'])"
curl -s -X POST http://localhost:8080/api/create_label -H 'Content-Type: application/json' -d '{
  "doc_type":"2","binder_size":"5","pjt_number":"PJ-01","pjt_test_number":"T-2026-001",
  "pjt_doc_title":"아주 길게 작성된 과제 문서 제목 자동 축소 확인용 테스트 문자열","pjt_doc_writer":"홍길동","pjt_doc_count":"2"}' \
  | python3 -c "import sys,json,base64;d=json.load(sys.stdin);open('$SCRATCH/project.pdf','wb').write(base64.b64decode(d['file_base64']));print(d['filename'])"
kill %1
```

(포트가 8080이 아니면 `internal/config/config.go` 기본값 확인 후 교체.)

- [ ] **Step 2: 검증 항목 확인**
  - `pdfinfo` 또는 파일 크기로 페이지 수·유효성 확인.
  - 사용자에게 안내: **인쇄 시 "실제 크기(배율 100%)" 설정**으로 출력 후 실측 — 7cm 장비 라벨 42.9×154mm, 점선 재단선, 긴 제목 축소 여부 확인.

- [ ] **Step 3: 사용자 확인 후 후속 정리(별도 결정)**
  - 사용자 시각 승인 후: `internal/excel` 및 xlsx 패리티 자산 삭제는 **별도 커밋/별도 승인**으로 진행 (이 계획 범위 밖).

---

## Self-Review 결과

- 스펙 커버리지: PDF 대체(T7-8), 크기 동일(T3 스냅샷), A4 다중 배치(T6), 보조 표 별도 조각(T5-7), 점선 재단선(T6), 자동 축소 전 셀(T4-5), 폰트 정책(T1-2), 파일명/JSON 계약(T7-8), 수동 실측(T9). 누락 없음.
- excel 삭제는 스펙대로 후속 커밋 — T9 Step 3에 게이트 명시.
- 시그니처 일관성: `CreateLabelPDF(dt, b, lbl, qrs) (data, filename, err)` T7 정의 = T8 사용. `fitText/drawTextBox/renderMain/renderAux/layoutPieces/packAndDraw` 정의·사용 일치.
- 리스크 명시: fpdf `SetFallbackFonts`/`SetPage` API 형상은 T2/T6에 go doc 확인 단계 + 대안 포함.
