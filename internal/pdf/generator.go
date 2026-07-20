package pdf

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"

	"qrweb/internal/label"
)

// Generator builds label PDFs; the drop-in successor to excel.Generator.
type Generator struct{}

// NewGenerator returns a ready-to-use Generator.
func NewGenerator() *Generator { return &Generator{} }

// CreateLabelPDF renders N main label pieces (one per document copy, each with
// its own i/N marker and QR) plus, for project labels, N side-table pieces,
// shelf-packed onto A4 pages with cut guides.
func (g *Generator) CreateLabelPDF(dt label.DocType, b label.BinderSize, lbl label.Label, qrs label.QRImageSet) ([]byte, string, error) {
	doc := newDoc()
	docCount := lbl.DocCount()
	images := qrs.Images()

	mw, mh := mainSize(b)
	pieces := make([]piece, 0, docCount*2)
	for i := 1; i <= docCount; i++ {
		i := i
		marker := fmt.Sprintf("%d/%d", i, docCount)
		pieces = append(pieces, piece{w: mw, h: mh, draw: func(doc *fpdf.Fpdf, x, y float64) error {
			return renderMain(doc, x, y, dt, b, lbl, marker, images[i-1], fmt.Sprintf("qr_%d", i))
		}})
	}
	if dt.IsProject() {
		aw, ah := auxSize()
		for i := 1; i <= docCount; i++ {
			marker := fmt.Sprintf("%d/%d", i, docCount)
			pieces = append(pieces, piece{w: aw, h: ah, draw: func(doc *fpdf.Fpdf, x, y float64) error {
				renderAux(doc, x, y, lbl, marker)
				return doc.Error()
			}})
		}
	}

	if err := packAndDraw(doc, pieces); err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), label.GenerateTimestampFilename(lbl.DocNumber(), "pdf"), nil
}
