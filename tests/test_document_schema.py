import pytest
from document_schema import parse_label_request, ValidationError, EQUIPMENT_REQUIRED_FIELDS, PROJECT_REQUIRED_FIELDS


VALID_EQUIPMENT = {
    'eq_number': 'EQ001',
    'eq_doc_number': 'DOC-001',
    'eq_doc_title': '장비 유지 관리 절차서',
    'eq_doc_count': '3',
    'eq_doc_department': '품질관리부',
    'eq_doc_year': '2024',
}

VALID_PROJECT = {
    'pjt_number': 'PJT-001',
    'pjt_test_number': 'TEST-001',
    'pjt_doc_title': '시험 절차서',
    'pjt_doc_writer': '홍길동',
    'pjt_doc_count': '2',
}


class TestParseEquipment:
    def test_valid_returns_data_doctype_bindersize(self):
        data, doc_type, binder_size = parse_label_request(VALID_EQUIPMENT, '1', '3')
        assert doc_type == '1'
        assert binder_size == 3
        assert data['eq_number'] == 'EQ001'
        assert data['eq_doc_count'] == 3  # int, not string

    def test_all_binder_sizes_accepted(self):
        for size in [1, 3, 5, 7]:
            data, _, bs = parse_label_request(VALID_EQUIPMENT, '1', str(size))
            assert bs == size

    def test_missing_required_field_raises(self):
        for field in EQUIPMENT_REQUIRED_FIELDS:
            bad = {k: v for k, v in VALID_EQUIPMENT.items() if k != field}
            with pytest.raises(ValidationError):
                parse_label_request(bad, '1', '3')


class TestParseProject:
    def test_valid_returns_data(self):
        data, doc_type, binder_size = parse_label_request(VALID_PROJECT, '2', '3')
        assert doc_type == '2'
        assert data['pjt_doc_count'] == 2

    def test_project_rejects_1cm_binder(self):
        with pytest.raises(ValidationError, match='3cm'):
            parse_label_request(VALID_PROJECT, '2', '1')

    def test_project_accepts_3cm_and_above(self):
        for size in [3, 5, 7]:
            parse_label_request(VALID_PROJECT, '2', str(size))


class TestValidation:
    def test_invalid_doc_type_raises(self):
        with pytest.raises(ValidationError, match='문서 종류'):
            parse_label_request(VALID_EQUIPMENT, '3', '3')

    def test_invalid_binder_size_string_raises(self):
        with pytest.raises(ValidationError, match='바인더'):
            parse_label_request(VALID_EQUIPMENT, '1', 'bad')

    def test_invalid_binder_size_value_raises(self):
        with pytest.raises(ValidationError, match='바인더'):
            parse_label_request(VALID_EQUIPMENT, '1', '2')
