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
