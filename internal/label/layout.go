package label

// QRConfig holds the QR column-width config for a given binder size.
//
// ColumnWidth is applied to columns B–M at QR embed time.
// QR image anchoring and centering is handled by excel.qrCenterAnchor;
// this package only provides ColumnWidth.
//
// Mirrors label_layout.get_qr_config in the Python original (CellPos removed).
type QRConfig struct {
	ColumnWidth float64
}

// binderTable maps binder size (cm) → column width (char units).
// Mirrors _BINDER_QR_CONFIG in label_layout.py.
var binderTable = map[int]float64{
	7: 1.875,
	5: 1.25,
	3: 1.0,
	1: 0.75,
}

// defaultBinder is the fallback binder size for unknown sizes
// (_DEFAULT_BINDER_SIZE in label_layout.py).
const defaultBinder = 3

// GetQRConfig returns the QR config for the given binder size.
//
// Only ColumnWidth is returned; QR anchor placement is handled by
// excel.qrCenterAnchor. Unknown binder sizes fall back to defaultBinder (3).
func GetQRConfig(binder int) QRConfig {
	w, ok := binderTable[binder]
	if !ok {
		w = binderTable[defaultBinder]
	}
	return QRConfig{ColumnWidth: w}
}
