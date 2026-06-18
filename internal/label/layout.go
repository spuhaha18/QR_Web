package label

// QRConfig is the QR placement config for a given doc type and binder size.
//
// ColumnWidth is applied to columns B–M at QR embed time; CellPos is the
// Excel cell address used as the QR image anchor.
//
// Mirrors label_layout.get_qr_config in the Python original.
type QRConfig struct {
	ColumnWidth float64
	CellPos     string
}

// binderEntry holds the per-binder-size layout values from _BINDER_QR_CONFIG.
type binderEntry struct {
	columnWidth   float64
	equipmentCell string
	projectCell   string
}

// binderTable mirrors _BINDER_QR_CONFIG in label_layout.py.
var binderTable = map[int]binderEntry{
	7: {1.875, "E9", "E8"},
	5: {1.25, "D9", "D8"},
	3: {1.0, "D9", "D8"},
	1: {0.75, "B9", "B9"},
}

// defaultBinder is the fallback binder size for unknown sizes
// (_DEFAULT_BINDER_SIZE in label_layout.py).
const defaultBinder = 3

// GetQRConfig returns the QR placement config for the given docType and binder.
//
// docType "1" selects the equipment cell; any other value selects the project
// cell. Unknown binder sizes fall back to defaultBinder (3).
func GetQRConfig(docType string, binder int) QRConfig {
	e, ok := binderTable[binder]
	if !ok {
		e = binderTable[defaultBinder]
	}
	cell := e.equipmentCell
	if docType != "1" {
		cell = e.projectCell
	}
	return QRConfig{ColumnWidth: e.columnWidth, CellPos: cell}
}
