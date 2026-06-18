package label

import "fmt"

// QRImageSet is a validated, sheet-ordered set of QR PNGs. Its construction is
// the single place the CONTEXT invariant "권 1 = QR 1" is enforced as a count:
// exactly DocCount images, indexed by sheet order (image i embeds into sheet i).
//
// Both intake paths build one — paste mode after reordering uploads by the
// client permutation, auto mode after generating one QR per sheet — so the
// excel generator consumes a type that already guarantees the count and no
// longer needs its own defensive length check. Per-image concerns that depend
// on runtime config (max file size, PNG validity) remain in the HTTP intake
// layer; this type owns only the count-and-order invariant, which is pure
// domain.
type QRImageSet struct {
	images [][]byte
}

// NewQRImageSet builds a set from images already in sheet order. It returns an
// error unless the count matches docCount, enforcing 권 1 = QR 1.
func NewQRImageSet(orderedImages [][]byte, docCount int) (QRImageSet, error) {
	if len(orderedImages) != docCount {
		return QRImageSet{}, fmt.Errorf("qr image count %d does not match doc count %d", len(orderedImages), docCount)
	}
	return QRImageSet{images: orderedImages}, nil
}

// Images returns the PNGs in sheet order.
func (s QRImageSet) Images() [][]byte { return s.images }

// Len returns the number of images, equal to the doc count.
func (s QRImageSet) Len() int { return len(s.images) }
