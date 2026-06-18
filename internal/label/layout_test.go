package label

import "testing"

// Ported from tests/test_label_layout.py (TestGetQrConfig).

func TestGetQRConfig_Equipment7cm(t *testing.T) {
	cfg := GetQRConfig("1", 7)
	if cfg.CellPos != "E9" {
		t.Errorf("CellPos = %q, want E9", cfg.CellPos)
	}
	if cfg.ColumnWidth != 1.875 {
		t.Errorf("ColumnWidth = %v, want 1.875", cfg.ColumnWidth)
	}
}

func TestGetQRConfig_Project7cm(t *testing.T) {
	cfg := GetQRConfig("2", 7)
	if cfg.CellPos != "E8" {
		t.Errorf("CellPos = %q, want E8", cfg.CellPos)
	}
	if cfg.ColumnWidth != 1.875 {
		t.Errorf("ColumnWidth = %v, want 1.875", cfg.ColumnWidth)
	}
}

func TestGetQRConfig_Equipment5cm(t *testing.T) {
	cfg := GetQRConfig("1", 5)
	if cfg.CellPos != "D9" {
		t.Errorf("CellPos = %q, want D9", cfg.CellPos)
	}
}

func TestGetQRConfig_Project5cm(t *testing.T) {
	cfg := GetQRConfig("2", 5)
	if cfg.CellPos != "D8" {
		t.Errorf("CellPos = %q, want D8", cfg.CellPos)
	}
}

func TestGetQRConfig_Equipment3cm(t *testing.T) {
	cfg := GetQRConfig("1", 3)
	if cfg.CellPos != "D9" {
		t.Errorf("CellPos = %q, want D9", cfg.CellPos)
	}
}

func TestGetQRConfig_1cmSameForBothTypes(t *testing.T) {
	eqCfg := GetQRConfig("1", 1)
	pjtCfg := GetQRConfig("2", 1)
	if eqCfg.CellPos != "B9" {
		t.Errorf("equipment CellPos = %q, want B9", eqCfg.CellPos)
	}
	if pjtCfg.CellPos != "B9" {
		t.Errorf("project CellPos = %q, want B9", pjtCfg.CellPos)
	}
}

func TestGetQRConfig_UnknownBinderFallsBackTo3(t *testing.T) {
	cfg := GetQRConfig("1", 99)
	defaultCfg := GetQRConfig("1", 3)
	if cfg != defaultCfg {
		t.Errorf("unknown binder cfg = %+v, want fallback %+v", cfg, defaultCfg)
	}
}

func TestGetQRConfig_ReturnsColumnWidth(t *testing.T) {
	cfg := GetQRConfig("1", 1)
	if cfg.ColumnWidth != 0.75 {
		t.Errorf("ColumnWidth = %v, want 0.75", cfg.ColumnWidth)
	}
}
