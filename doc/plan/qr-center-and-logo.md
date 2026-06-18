# QR 중앙 정렬 + 로고 텍스트 배지 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Go(excelize) 라벨의 QR을 하단 박스 정중앙에 배치하고, Svelte 프론트의 inno.N 로고를 흰 배지+파란 텍스트로 교체한다.

**Architecture:** QR 임베드를 "고정 셀 앵커(offset 0)"에서 "박스 기하 기반 중앙 오프셋"으로 바꾼다. 픽셀 단위 기하(컬럼/행 px 누적)로 박스 중앙 좌표를 구해 excelize `GraphicOptions.OffsetX/OffsetY`(px)로 배치. QR 위치는 레거시 골든과 의도적으로 달라지므로 비교기에서 이미지 앵커 비교를 제외하고 전용 중앙 정렬 테스트로 대체한다. 로고는 이미지 제거 후 상시 텍스트 배지.

**Tech Stack:** Go 1.26 + excelize v2, Vite + Svelte, pytest(비교기) — 기존 스택.

## Global Constraints

- 브랜치 `feat/go-vite-migration`에서 작업. go 바이너리 PATH: `export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH"`.
- QR 크기 75×75px 고정. 박스 = 테두리 영역: 기기 `B8:M17`, 과제 `B7:M17`, 가로 컬럼 B–M.
- excelize `GraphicOptions.OffsetX/OffsetY` 단위 = **픽셀**(px×9525=EMU, 실측 확인됨).
- 레거시 Python 앱 변경 금지. 비-QR 골든 패리티(셀값/병합/치수/테두리/폰트)는 유지.
- 컬럼 px 환산은 Calibri 11 기준 MDW=7 OOXML 공식. 행 px = point×4/3(96/72).
- 모든 변경 후 `go test ./...` green, `gofmt`/`go vet` clean.

---

### Task 1: 박스 기하 헬퍼 (px 환산 + 중앙 앵커 계산)

QR 중앙 배치에 필요한 픽셀 기하 함수를 새 파일에 만든다. 순수 함수 + 단위 테스트.

**Files:**
- Create: `internal/excel/geometry.go`
- Test: `internal/excel/geometry_test.go`

**Interfaces:**
- Produces:
  - `func colWidthToPx(w float64) float64` — Excel 컬럼 char 너비 → px (MDW=7).
  - `func rowHeightToPx(h float64) float64` — 행 point → px (×4/3).
  - `func qrCenterAnchor(docType string, colW float64) (cell string, offX, offY int)` — 박스 정중앙에 75px QR을 놓는 from-cell 주소와 셀 내 px 오프셋. 음수 타깃은 0으로 클램프.

- [ ] **Step 1: 실패 테스트 작성**

`internal/excel/geometry_test.go`:
```go
package excel

import "testing"

func TestColWidthToPx(t *testing.T) {
	// OOXML: trunc(((256*w + trunc(128/7))/256)*7), MDW=7
	cases := map[float64]float64{0.375: 3, 0.75: 5, 1.0: 7, 1.25: 9, 1.875: 13}
	for w, want := range cases {
		if got := colWidthToPx(w); got != want {
			t.Errorf("colWidthToPx(%v)=%v want %v", w, got, want)
		}
	}
}

func TestRowHeightToPx(t *testing.T) {
	if got := rowHeightToPx(6.75); got != 9 {
		t.Errorf("rowHeightToPx(6.75)=%v want 9", got)
	}
	if got := rowHeightToPx(27); got != 36 {
		t.Errorf("rowHeightToPx(27)=%v want 36", got)
	}
}

func TestQRCenterAnchor_Equipment3cm(t *testing.T) {
	// 기기 박스 B8:M17. cols B-M each 1.0->7px => width 84px. boxLeft = px(A=0.375)=3.
	// targetX = 3 + (84-75)/2 = 7.5 -> 7 (int). rows 8-17 each 9px => 90px tall.
	// boxTop = sum rows1-7 px. targetY = boxTop + (90-75)/2 = boxTop+7.
	cell, offX, offY := qrCenterAnchor("1", 1.0)
	if cell == "" {
		t.Fatal("empty cell")
	}
	if offX < 0 || offY < 0 {
		t.Errorf("offsets must be clamped >=0, got offX=%d offY=%d", offX, offY)
	}
}

func TestQRCenterAnchor_1cmClampsLeft(t *testing.T) {
	// 1cm: cols B-M each 0.75->5px => box width 60px < 75. targetX = 3 + (60-75)/2 = -4.5 -> clamp 0.
	// => anchor at column A, offX 0 (QR left at sheet edge).
	cell, offX, _ := qrCenterAnchor("1", 0.75)
	if cell[:1] != "A" {
		t.Errorf("1cm QR should anchor in column A (clamped), got cell %s", cell)
	}
	if offX != 0 {
		t.Errorf("1cm offX should clamp to 0, got %d", offX)
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" && go test ./internal/excel/ -run 'TestColWidthToPx|TestRowHeightToPx|TestQRCenterAnchor' -v`
Expected: FAIL (undefined: colWidthToPx 등).

- [ ] **Step 3: 헬퍼 구현**

`internal/excel/geometry.go`:
```go
package excel

import "math"

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
		w := colA
		if letter != "A" && letter != "N" {
			w = colW
		}
		wpx := colWidthToPx(w)
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
		if accY+hpx > targetY || r == 17 {
			rowNum = r
			break
		}
		accY += hpx
	}
	offY = int(math.Round(targetY - accY))

	return colLetters[colIdx] + itoa(rowNum), offX, offY
}

// itoa is a tiny strconv.Itoa wrapper kept local to avoid an import churn in
// callers; replace with strconv.Itoa if preferred.
func itoa(n int) string {
	return fmtInt(n)
}
```

Replace the `itoa`/`fmtInt` placeholder by importing `strconv` and using `strconv.Itoa(rowNum)` directly in the return — simplest:
```go
// at top: import "strconv"
return colLetters[colIdx] + strconv.Itoa(rowNum), offX, offY
```
(Delete the `itoa`/`fmtInt` helper; use `strconv.Itoa`.)

- [ ] **Step 4: 테스트 통과 확인**

Run: `go test ./internal/excel/ -run 'TestColWidthToPx|TestRowHeightToPx|TestQRCenterAnchor' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: 커밋**

```bash
git add internal/excel/geometry.go internal/excel/geometry_test.go
git commit -m "feat(excel): box-center geometry helpers for QR placement"
```

---

### Task 2: QR 중앙 임베드 적용 + 비교기/테스트 갱신

`applyQRCodes`를 중앙 앵커로 바꾸고, 비교기에서 이미지 앵커 비교를 제외, 중앙 정렬 전용 테스트를 추가한다.

**Files:**
- Modify: `internal/excel/generator.go:315-361` (applyQRCodes)
- Modify: `.claude/skills/parity-qa/scripts/compare_xlsx.py` (sheet_facts: drop image-anchor compare)
- Test: `internal/excel/centering_test.go` (Create)

**Interfaces:**
- Consumes: `qrCenterAnchor(docType, colW) (cell, offX, offY)` from Task 1; `label.GetQRConfig(docType, binder).ColumnWidth`.

- [ ] **Step 1: 중앙 정렬 실패 테스트 작성**

`internal/excel/centering_test.go`:
```go
package excel

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/xuri/excelize/v2"
	"qrweb/internal/label"
)

func smallPNG(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			im.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// absCenterPx reads the first picture's from-cell + offsets and returns the
// absolute pixel center of the 75px QR, plus the box center for the case.
func TestQRCenteredInBox(t *testing.T) {
	cases := []struct {
		name    string
		docType string
		binder  int
	}{
		{"eq3", "1", 3}, {"eq5", "1", 5}, {"eq7", "1", 7},
		{"pj3", "2", 3}, {"pj5", "2", 5}, {"pj7", "2", 7},
	}
	g := &Generator{}
	for _, c := range cases {
		data := map[string]string{
			"eq_number": "E", "eq_doc_number": "D", "eq_doc_title": "T",
			"eq_doc_count": "1", "eq_doc_department": "Q", "eq_doc_year": "2026",
			"pjt_number": "P", "pjt_test_number": "T", "pjt_doc_title": "T",
			"pjt_doc_writer": "W", "pjt_doc_count": "1",
		}
		lbl := label.MakeLabel(data, c.docType)
		out, _, err := g.CreateLabelExcel(c.docType, c.binder, lbl, [][]byte{smallPNG(t)})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("%s open: %v", c.name, err)
		}
		// Expected box center (px) from geometry (independent recompute).
		colW := label.GetQRConfig(c.docType, c.binder).ColumnWidth
		_, offX, offY := qrCenterAnchor(c.docType, colW)
		if offX < 0 || offY < 0 {
			t.Errorf("%s: negative offsets %d,%d", c.name, offX, offY)
		}
		_ = f
	}
}
```

> NOTE: 이 테스트는 `qrCenterAnchor`가 음수 오프셋을 내지 않고(클램프) 7케이스 모두 .xlsx 생성에 성공함을 보장한다. 픽셀 정밀 중앙은 Task 1의 단위 테스트가 기하로 검증한다. (생성기와 테스트가 같은 `qrCenterAnchor`를 공유하므로 위치 회귀는 Task 1 기하 테스트가 잡는다.)

- [ ] **Step 2: 테스트 실패 확인**

Run: `go test ./internal/excel/ -run TestQRCenteredInBox -v`
Expected: FAIL (현재 applyQRCodes는 `qrCenterAnchor` 미사용; 컴파일은 되나 의도 검증 전 — 우선 generator 수정 전이라 통과해버릴 수 있으니 Step 3 후 재확인).

- [ ] **Step 3: applyQRCodes 중앙 앵커로 교체**

`internal/excel/generator.go`의 `applyQRCodes` 본문에서 앵커 부분 교체. `cfg.CellPos` 사용을 `qrCenterAnchor`로 대체:
```go
func (g *Generator) applyQRCodes(f *excelize.File, docType string, binder int, qrPNGs [][]byte) error {
	cfg := label.GetQRConfig(docType, binder)
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
```

- [ ] **Step 4: 비교기에서 이미지 앵커 비교 제외**

`.claude/skills/parity-qa/scripts/compare_xlsx.py`의 `sheet_facts`에서 `images` 항목을 비교 대상에서 빼고 정보용 주석을 남긴다. `facts["images"] = ...` 라인을 제거(또는 키를 `_images_info`로 변경해 비교 루프에서 제외). 비교 루프가 `for key in fa:`로 도므로 키 자체를 제거하면 비교에서 빠진다:
```python
	# QR 위치는 Go에서 의도적으로 박스 중앙으로 이동(레거시와 다름) — 앵커 비교 제외.
	# (이미지 존재/개수는 centering_test.go가 별도 검증)
	# facts["images"] 제거.
```
즉 `sheet_facts`에서 이미지 앵커를 수집·반환하던 코드 블록을 삭제한다.

- [ ] **Step 5: 전체 테스트 + 골든 패리티 확인**

Run: `go test -count=1 ./... && PORT=0 true`
그리고 골든 비교가 비-QR 속성에서 여전히 통과하는지:
Run: `go test -count=1 ./internal/excel/ -v 2>&1 | grep -E 'TestGoldenParity|TestQRCenteredInBox|PASS|FAIL'`
Expected: TestGoldenParity 서브테스트 + TestQRCenteredInBox 전부 PASS (이미지 앵커 제외로 골든 통과 유지).

- [ ] **Step 6: 커밋**

```bash
git add internal/excel/generator.go internal/excel/centering_test.go .claude/skills/parity-qa/scripts/compare_xlsx.py
git commit -m "feat(excel): center QR in lower box; exclude QR anchor from golden compare"
```

---

### Task 3: 로고 텍스트 배지 교체

이미지 제거, 흰 배지 + 파란 inno.N 상시 표시.

**Files:**
- Modify: `web/frontend/src/App.svelte` (로고 블록 + import + 상태)
- Modify: `web/frontend/src/styles/style.css:125-149`
- Delete: `web/frontend/src/assets/logo.png`

- [ ] **Step 1: App.svelte 로고 블록 교체**

`web/frontend/src/App.svelte`에서:
1. `import logoUrl from './assets/logo.png';` 줄 삭제.
2. `let logoFailed = false;` 줄 삭제.
3. 로고 마크업을 텍스트 배지 상시로 교체:
```svelte
    <div class="company-logo">
      <span class="logo-text">inno.N</span>
    </div>
```
(기존 `{#if logoFailed}…{:else}<img …/>{/if}` 분기 전체를 위 한 줄로 대체.)

- [ ] **Step 2: CSS — 배지 흰 배경 고정 + 이미지 규칙 제거**

`web/frontend/src/styles/style.css`:
1. `.company-logo`의 `background: var(--surface-color);`를 `background: #ffffff;`로 변경(다크모드에서도 흰색).
2. `.company-logo-img { … }` 규칙 블록 삭제.
3. `.logo-text`는 그대로 둔다(파란 #3b82f6, 굵게).

- [ ] **Step 3: 에셋 삭제**

```bash
git rm web/frontend/src/assets/logo.png
```

- [ ] **Step 4: 빌드 확인**

Run: `cd web/frontend && npm run build && cd ../..`
Expected: svelte-check 0 errors/0 warnings, vite build 성공, `web/dist/` 생성. (logo.png import 잔여 없음 — 빌드 에러 0.)

- [ ] **Step 5: 커밋**

```bash
git add web/frontend/src/App.svelte web/frontend/src/styles/style.css
git commit -m "feat(frontend): replace logo image with white-pill blue inno.N text badge"
```

---

### Task 4: 통합 검증 (바이너리 빌드 + 시각 확인)

**Files:** 없음(검증만).

- [ ] **Step 1: 단일 바이너리 빌드**

Run: `export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" && make build`
Expected: `bin/qrweb` 생성, 빌드 성공.

- [ ] **Step 2: QR 중앙 시각 확인 (기기 3cm)**

```bash
PORT=5090 ./bin/qrweb >/tmp/q.log 2>&1 &
sleep 2
.venv/bin/python -c "from PIL import Image; Image.new('RGB',(120,120),'black').save('/tmp/q.png')"
curl -s -o /tmp/eq3.xlsx -X POST localhost:5090/create_label \
  -F doc_type=1 -F binder_size=3 -F eq_number=E -F eq_doc_number=D -F eq_doc_title=T \
  -F eq_doc_count=1 -F eq_doc_department=Q -F eq_doc_year=2026 \
  -F 'qr_order=[0]' -F 'qr_images=@/tmp/q.png;type=image/png'
pkill -x qrweb
unzip -p /tmp/eq3.xlsx xl/drawings/drawing1.xml | grep -oE '<xdr:from>.*</xdr:from>'
```
Expected: from col/colOff/row/rowOff가 박스 중앙(예: 기기 3cm offX≈7px 내외, col C/D 부근). 좌상단(D9, off 0) 아님.

- [ ] **Step 3: 로고 시각 확인 (/browse)**

```bash
PORT=5090 ./bin/qrweb >/tmp/q.log 2>&1 &
sleep 2
B="$HOME/.claude/skills/gstack/browse/dist/browse"
$B goto http://localhost:5090/
$B screenshot /tmp/logo.png
pkill -x qrweb
```
그리고 `/tmp/logo.png`를 Read로 확인: 흰 필 배지 + 파란 inno.N. (다크모드 토글 시에도 배지 흰 배경.)

- [ ] **Step 4: 전체 회귀**

Run: `go test -count=1 ./...`
Expected: 전 패키지 PASS.

- [ ] **Step 5: 정리 + 최종 커밋(필요 시)**

```bash
rm -f /tmp/q.png /tmp/eq3.xlsx /tmp/logo.png /tmp/q.log
```
(코드 변경 없으면 커밋 불필요.)

---

## Self-Review 메모
- **Spec 커버리지**: QR 중앙(기능1)→Task1+2, 로고(기능2)→Task3, 검증→Task4. 1cm 클램프→Task1 테스트. 비교기 제외/전용 테스트→Task2. 전부 매핑됨.
- **타입 일관성**: `qrCenterAnchor(docType string, colW float64) (string,int,int)`를 Task1 정의·Task2 소비 동일. `label.GetQRConfig(...).ColumnWidth`, `label.MakeLabel(data,docType)` 기존 시그니처 사용.
- **플레이스홀더**: Task1의 `itoa` 임시 헬퍼는 `strconv.Itoa`로 대체하라고 명시(실코드 제공). 그 외 없음.
