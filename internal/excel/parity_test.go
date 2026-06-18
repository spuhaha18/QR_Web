package excel

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"qrweb/internal/label"
)

// dummyPNG mirrors capture_golden.dummy_png: a solid black 64x64 RGB PNG.
func dummyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	black := color.RGBA{0, 0, 0, 255}
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, black)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode dummy png: %v", err)
	}
	return buf.Bytes()
}

func dummyPNGs(t *testing.T, n int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		out[i] = dummyPNG(t)
	}
	return out
}

// equipLabel / projLabel mirror capture_golden.EQUIP / PROJ exactly.
func equipLabel(count int) label.Label {
	return label.EquipmentLabel{
		EqNumber:        "EQ-001",
		EqDocNumber:     "DOC-100",
		EqDocTitle:      "장비 검교정 기록",
		EqDocCount:      count,
		EqDocDepartment: "품질관리부",
		EqDocYear:       2026,
	}
}

func projLabel(count int) label.Label {
	return label.ProjectLabel{
		PjtNumber:     "PJT-7",
		PjtTestNumber: "T-42",
		PjtDocTitle:   "안정성 시험 보고서",
		PjtDocWriter:  "홍길동",
		PjtDocCount:   count,
	}
}

type matrixCase struct {
	docType string
	binder  int
	count   int
	lbl     label.Label
}

func parityMatrix() []matrixCase {
	var cases []matrixCase
	for _, binder := range []int{1, 3, 5, 7} {
		for _, count := range []int{1, 3} {
			cases = append(cases, matrixCase{"1", binder, count, equipLabel(count)})
		}
	}
	for _, binder := range []int{3, 5, 7} {
		for _, count := range []int{1, 3} {
			cases = append(cases, matrixCase{"2", binder, count, projLabel(count)})
		}
	}
	return cases
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd() // .../internal/excel
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// TestGoldenParity generates each matrix .xlsx in Go and compares it against the
// Python golden via compare_xlsx.py. Requires .venv python + openpyxl.
func TestGoldenParity(t *testing.T) {
	root := projectRoot(t)
	pyBin := filepath.Join(root, ".venv", "bin", "python")
	if _, err := os.Stat(pyBin); err != nil {
		t.Skipf("python venv not found at %s: %v", pyBin, err)
	}
	compare := filepath.Join(root, ".claude", "skills", "parity-qa", "scripts", "compare_xlsx.py")
	goldenDir := filepath.Join(root, "testdata", "golden")
	outDir := t.TempDir()

	gen := NewGenerator()

	for _, c := range parityMatrix() {
		tag := fmt.Sprintf("t%s_b%d_n%d", c.docType, c.binder, c.count)
		t.Run(tag, func(t *testing.T) {
			data, _, err := gen.CreateLabelExcel(c.docType, c.binder, c.lbl, dummyPNGs(t, c.count))
			if err != nil {
				t.Fatalf("CreateLabelExcel: %v", err)
			}
			candPath := filepath.Join(outDir, tag+".xlsx")
			if err := os.WriteFile(candPath, data, 0o644); err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join(goldenDir, tag+".xlsx")

			cmd := exec.Command(pyBin, compare, goldenPath, candPath)
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(out), "MATCH") || strings.Contains(string(out), "MISMATCH") {
				t.Fatalf("parity MISMATCH for %s:\n%s", tag, out)
			}
		})
	}
}

// TestExcelPasteMode is a structural smoke test (no python dependency).
func TestExcelPasteMode(t *testing.T) {
	gen := NewGenerator()
	data, fn, err := gen.CreateLabelExcel("1", 3, equipLabel(3), dummyPNGs(t, 3))
	if err != nil {
		t.Fatalf("CreateLabelExcel: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty data")
	}
	if !strings.HasPrefix(fn, "DOC-100_") || !strings.HasSuffix(fn, ".xlsx") {
		t.Fatalf("unexpected filename: %s", fn)
	}
}

func TestExcelProjectLabel(t *testing.T) {
	gen := NewGenerator()
	_, fn, err := gen.CreateLabelExcel("2", 5, projLabel(1), dummyPNGs(t, 1))
	if err != nil {
		t.Fatalf("CreateLabelExcel: %v", err)
	}
	if !strings.HasPrefix(fn, "T-42_") {
		t.Fatalf("unexpected filename: %s", fn)
	}
}
