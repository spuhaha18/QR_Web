package label

import "testing"

// Ported from tests/test_label_layout.py (TestGetQrConfig).
// CellPos assertions removed: QR anchor is now handled by excel.qrCenterAnchor.

func TestGetQRConfig_7cm(t *testing.T) {
	cfg := GetQRConfig(7)
	if cfg.ColumnWidth != 1.875 {
		t.Errorf("ColumnWidth = %v, want 1.875", cfg.ColumnWidth)
	}
}

func TestGetQRConfig_5cm(t *testing.T) {
	cfg := GetQRConfig(5)
	if cfg.ColumnWidth != 1.25 {
		t.Errorf("ColumnWidth = %v, want 1.25", cfg.ColumnWidth)
	}
}

func TestGetQRConfig_3cm(t *testing.T) {
	cfg := GetQRConfig(3)
	if cfg.ColumnWidth != 1.0 {
		t.Errorf("ColumnWidth = %v, want 1.0", cfg.ColumnWidth)
	}
}

func TestGetQRConfig_1cm(t *testing.T) {
	cfg := GetQRConfig(1)
	if cfg.ColumnWidth != 0.75 {
		t.Errorf("ColumnWidth = %v, want 0.75", cfg.ColumnWidth)
	}
}

func TestGetQRConfig_UnknownBinderFallsBackTo3(t *testing.T) {
	cfg := GetQRConfig(99)
	defaultCfg := GetQRConfig(3)
	if cfg != defaultCfg {
		t.Errorf("unknown binder cfg = %+v, want fallback %+v", cfg, defaultCfg)
	}
}

func TestGetQRConfig_ReturnsColumnWidth(t *testing.T) {
	cfg := GetQRConfig(1)
	if cfg.ColumnWidth != 0.75 {
		t.Errorf("ColumnWidth = %v, want 0.75", cfg.ColumnWidth)
	}
}
