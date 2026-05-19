import pytest
from label_layout import get_qr_config


class TestGetQrConfig:
    def test_equipment_7cm(self):
        cfg = get_qr_config('1', 7)
        assert cfg['cell_pos'] == 'E9'
        assert cfg['column_width'] == 1.875

    def test_project_7cm(self):
        cfg = get_qr_config('2', 7)
        assert cfg['cell_pos'] == 'E8'
        assert cfg['column_width'] == 1.875

    def test_equipment_5cm(self):
        cfg = get_qr_config('1', 5)
        assert cfg['cell_pos'] == 'D9'

    def test_project_5cm(self):
        cfg = get_qr_config('2', 5)
        assert cfg['cell_pos'] == 'D8'

    def test_equipment_3cm(self):
        cfg = get_qr_config('1', 3)
        assert cfg['cell_pos'] == 'D9'

    def test_1cm_same_for_both_types(self):
        eq_cfg = get_qr_config('1', 1)
        pjt_cfg = get_qr_config('2', 1)
        assert eq_cfg['cell_pos'] == 'B9'
        assert pjt_cfg['cell_pos'] == 'B9'

    def test_unknown_binder_size_falls_back_to_3(self):
        cfg = get_qr_config('1', 99)
        default_cfg = get_qr_config('1', 3)
        assert cfg == default_cfg

    def test_returns_column_width(self):
        cfg = get_qr_config('1', 1)
        assert cfg['column_width'] == 0.75
