package excel

import (
	"math"
	"testing"
)

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
	// 기기 박스 B8:M17, colW=1.0 (3cm 설정).
	//
	// 수평 계산:
	//   colA=0.375 -> 3px, colW=1.0 -> 7px
	//   boxLeft=3px, boxWidth=12*7=84px
	//   targetX = 3 + (84-75)/2 = 7.5  => col B (accX=3 + 7px > 7.5), offX=round(7.5-3)=5
	//
	// 수직 계산:
	//   boxTop = sum rows1..7 = 3+36+36+288+54+36+36 = 489px
	//   boxHeight = 10*9 = 90px (rows 8..17 each 6.75pt -> 9px)
	//   targetY = 489 + (90-75)/2 = 496.5  => row 8 (accY=489 + 9 > 496.5), offY=round(496.5-489)=8
	//
	// QR 절대 중앙: (3+5+37.5=45.5, 489+8+37.5=534.5)
	// 박스 중앙:    (3+84/2=45.0,  489+90/2=534.0)
	// 오차: |dx|=0.5px, |dy|=0.5px  => 모두 ±1px 이내
	cell, offX, offY := qrCenterAnchor("1", 1.0)

	// 앵커 셀 단언 — 위치 회귀 시 실패해야 함
	if cell != "B8" {
		t.Fatalf("equipment 3cm QR anchor cell: got %s, want B8", cell)
	}
	if offX != 5 {
		t.Errorf("equipment 3cm offX: got %d, want 5", offX)
	}
	if offY != 8 {
		t.Errorf("equipment 3cm offY: got %d, want 8", offY)
	}

	// QR 중앙이 박스 중앙과 ±1px 이내인지 검증.
	// QR 절대 중앙 계산: 앵커 셀까지의 누적 px + offset + 37.5(=75/2)
	const colA = 0.375
	colW := 1.0

	// 앵커 셀(B=index 1)까지 col 누적
	colLetters := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M"}
	anchorColIdx := 1 // B
	absX := 0.0
	for i, letter := range colLetters {
		if i == anchorColIdx {
			break
		}
		w := colW
		if letter == "A" {
			w = colA
		}
		absX += colWidthToPx(w)
	}

	// 앵커 행(8)까지 row 누적
	anchorRow := 8
	absY := 0.0
	for r := 1; r < anchorRow; r++ {
		absY += rowHeightToPx(rowHeights[r])
	}

	qrCenterX := absX + float64(offX) + qrSizePx/2.0
	qrCenterY := absY + float64(offY) + qrSizePx/2.0

	// 박스 중앙
	boxLeft := colWidthToPx(colA)
	boxWidth := 12.0 * colWidthToPx(colW)
	boxTop := absY
	boxHeight := 0.0
	for r := 8; r <= 17; r++ {
		boxHeight += rowHeightToPx(rowHeights[r])
	}
	boxCenterX := boxLeft + boxWidth/2.0
	boxCenterY := boxTop + boxHeight/2.0

	dx := math.Abs(qrCenterX - boxCenterX)
	dy := math.Abs(qrCenterY - boxCenterY)
	if dx > 1.0 {
		t.Errorf("QR center X off by %.2fpx (got %.1f, want %.1f), exceeds ±1px tolerance", dx, qrCenterX, boxCenterX)
	}
	if dy > 1.0 {
		t.Errorf("QR center Y off by %.2fpx (got %.1f, want %.1f), exceeds ±1px tolerance", dy, qrCenterY, boxCenterY)
	}
}

func TestQRCenterAnchor_1cmClampsLeft(t *testing.T) {
	// 1cm: cols B-M each 0.75->5px => box width 60px < 75. targetX = 3 + (60-75)/2 = -4.5 -> clamp 0.
	// => anchor at column A, offX 0 (QR left at sheet edge).
	cell, offX, offY := qrCenterAnchor("1", 0.75)
	if cell[:1] != "A" {
		t.Errorf("1cm QR should anchor in column A (clamped), got cell %s", cell)
	}
	if offX != 0 {
		t.Errorf("1cm offX should clamp to 0, got %d", offX)
	}
	// 클램프 상황에서도 offY는 음수이면 안 됨
	if offY < 0 {
		t.Errorf("1cm offY must be >=0 (clamped), got %d", offY)
	}
}
