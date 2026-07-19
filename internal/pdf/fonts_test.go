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
