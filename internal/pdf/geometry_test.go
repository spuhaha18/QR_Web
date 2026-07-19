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
