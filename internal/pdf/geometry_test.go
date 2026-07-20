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

// Snapshot values are the CALIBRATED sizes: nominal Excel-conversion mm times
// calScaleX/calScaleY (anchored to the measured 47×150mm real print of the
// 7cm label — see the calibration comment in geometry.go).
func TestMainSizeSnapshot(t *testing.T) {
	for _, tc := range []struct {
		binder int
		w      float64
	}{{1, 19.14815}, {3, 26.11111}, {5, 33.07407}, {7, 47.0}} {
		b, err := label.ParseBinderSize(itoa(tc.binder), label.DocTypeEquipment)
		if err != nil {
			t.Fatal(err)
		}
		w, h := mainSize(b)
		almost(t, w, tc.w, "width")
		almost(t, h, 150.0, "height")
	}
}

func itoa(n int) string { return string(rune('0' + n)) }

func TestAuxSizeSnapshot(t *testing.T) {
	w, h := auxSize()
	almost(t, w, 105.60494, "aux width")
	almost(t, h, 39.43299, "aux height")
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
	// col A = 3px = 0.79375mm nominal × calScaleX
	almost(t, g.colX[1], 0.87037, "col A right edge")
}
