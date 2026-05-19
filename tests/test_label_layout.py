import pytest
from label_layout import get_qr_config, encode_qr_payload


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


class TestEncodeQrPayload:
    def test_equipment_format(self):
        data = {
            'eq_number': 'EQ001', 'eq_doc_number': 'DOC-001',
            'eq_doc_title': '유지관리 절차서', 'eq_doc_department': '품질부',
            'eq_doc_year': 2024, 'eq_doc_count': 3,
        }
        result = encode_qr_payload(data, '1', 1, 3)
        assert result == 'EQ001|DOC-001|유지관리 절차서|품질부|2024|1/3'

    def test_equipment_sheet2(self):
        data = {
            'eq_number': 'EQ001', 'eq_doc_number': 'DOC-001',
            'eq_doc_title': '절차서', 'eq_doc_department': '팀',
            'eq_doc_year': 2024, 'eq_doc_count': 5,
        }
        result = encode_qr_payload(data, '1', 2, 5)
        assert result.endswith('|2/5')

    def test_project_format(self):
        data = {
            'pjt_number': 'PJT-001', 'pjt_test_number': 'TEST-001',
            'pjt_doc_title': '시험 절차서', 'pjt_doc_writer': '홍길동',
            'pjt_doc_count': 2,
        }
        result = encode_qr_payload(data, '2', 1, 2)
        assert result == 'PJT-001|TEST-001|시험 절차서|홍길동|1/2'

    def test_project_no_year_field(self):
        data = {
            'pjt_number': 'PJT-001', 'pjt_test_number': 'TEST-001',
            'pjt_doc_title': '절차서', 'pjt_doc_writer': '작성자',
            'pjt_doc_count': 1,
        }
        result = encode_qr_payload(data, '2', 1, 1)
        parts = result.split('|')
        assert len(parts) == 5  # project has 5 fields (no year)
