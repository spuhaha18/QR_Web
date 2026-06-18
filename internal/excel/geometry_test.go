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
