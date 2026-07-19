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
