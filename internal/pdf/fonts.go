// Package pdf renders label PDFs: same physical label size as the Excel
// 100%-scale printout, packed several-per-A4-page with cut guides.
package pdf

import _ "embed"

// User-supplied Microsoft fonts (Times New Roman, Batang face 0 from
// BATANG.TTC); redistribution inside this internal tool's binary was a
// recorded product decision — see doc/spec/pdf-label-output.md 폰트 section.
var (
	//go:embed fonts/times.ttf
	fontTimes []byte
	//go:embed fonts/timesbd.ttf
	fontTimesBold []byte
	//go:embed fonts/batang.ttf
	fontBatang []byte
)
