import pytest
from document_schema import EquipmentLabel, ProjectLabel, make_label


EQUIPMENT_DATA = {
    'eq_number': 'EQ001',
    'eq_doc_number': 'DOC-001',
    'eq_doc_title': '유지관리 절차서',
    'eq_doc_count': 3,
    'eq_doc_department': '품질부',
    'eq_doc_year': 2024,
}

PROJECT_DATA = {
    'pjt_number': 'PJT-001',
    'pjt_test_number': 'TEST-001',
    'pjt_doc_title': '시험 절차서',
    'pjt_doc_writer': '홍길동',
    'pjt_doc_count': 2,
}


class TestEquipmentLabel:
    def test_cell_values_has_all_cells(self):
        label = EquipmentLabel(**EQUIPMENT_DATA)
        cells = label.cell_values()
        for addr in ('B2', 'B3', 'B4', 'B5', 'B6', 'B7'):
            assert addr in cells

    def test_cell_b5_has_count_string(self):
        label = EquipmentLabel(**EQUIPMENT_DATA)
        assert label.cell_values()['B5'] == '1/3'

    def test_doc_number_is_eq_doc_number(self):
        label = EquipmentLabel(**EQUIPMENT_DATA)
        assert label.doc_number == 'DOC-001'

    def test_doc_count_is_int(self):
        label = EquipmentLabel(**EQUIPMENT_DATA)
        assert label.doc_count == 3

    def test_qr_payload_sheet1(self):
        label = EquipmentLabel(**EQUIPMENT_DATA)
        result = label.qr_payload(1, 3)
        assert result == 'EQ001|DOC-001|유지관리 절차서|품질부|2024|1/3'

    def test_qr_payload_sheet2(self):
        label = EquipmentLabel(**EQUIPMENT_DATA)
        assert label.qr_payload(2, 3).endswith('|2/3')

    def test_title_cell_is_b4(self):
        assert EquipmentLabel.TITLE_CELL == 'B4'


class TestProjectLabel:
    def test_cell_values_has_secondary_panel(self):
        label = ProjectLabel(**PROJECT_DATA)
        cells = label.cell_values()
        for addr in ('Q21', 'Q22', 'R23', 'S23'):
            assert addr in cells

    def test_q21_combines_number_and_test_number(self):
        label = ProjectLabel(**PROJECT_DATA)
        assert label.cell_values()['Q21'] == '[PJT-001] TEST-001'

    def test_doc_number_is_test_number(self):
        label = ProjectLabel(**PROJECT_DATA)
        assert label.doc_number == 'TEST-001'

    def test_qr_payload_project(self):
        label = ProjectLabel(**PROJECT_DATA)
        result = label.qr_payload(1, 2)
        assert result == 'PJT-001|TEST-001|시험 절차서|홍길동|1/2'

    def test_project_payload_has_5_fields(self):
        label = ProjectLabel(**PROJECT_DATA)
        parts = label.qr_payload(1, 1).split('|')
        assert len(parts) == 5  # no year field


class TestMakeLabel:
    def test_makes_equipment_label(self):
        label = make_label(EQUIPMENT_DATA, '1')
        assert isinstance(label, EquipmentLabel)

    def test_makes_project_label(self):
        label = make_label(PROJECT_DATA, '2')
        assert isinstance(label, ProjectLabel)
