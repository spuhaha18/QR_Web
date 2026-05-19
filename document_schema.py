"""
Document schema: field definitions, validation rules, and data parsing
for equipment (doc_type='1') and project (doc_type='2') label requests.
"""
from dataclasses import dataclass
from datetime import datetime
from typing import ClassVar, Union
from utils import validate_and_clean_input, safe_int_conversion


class ValidationError(ValueError):
    """Raised when label request data fails validation."""
    pass


VALID_DOC_TYPES = ('1', '2')
VALID_BINDER_SIZES = (1, 3, 5, 7)

EQUIPMENT_REQUIRED_FIELDS = [
    'eq_number', 'eq_doc_number', 'eq_doc_title',
    'eq_doc_count', 'eq_doc_department', 'eq_doc_year',
]

PROJECT_REQUIRED_FIELDS = [
    'pjt_number', 'pjt_test_number', 'pjt_doc_title',
    'pjt_doc_writer', 'pjt_doc_count',
]


def parse_label_request(form_data, doc_type, binder_size_raw):
    """Validate and parse a label creation request.

    Args:
        form_data: dict-like (ImmutableMultiDict or plain dict)
        doc_type: raw doc_type value from request
        binder_size_raw: raw binder_size value from request

    Returns:
        tuple (data: dict, doc_type: str, binder_size: int)

    Raises:
        ValidationError: on any validation failure
    """
    if doc_type not in VALID_DOC_TYPES:
        raise ValidationError('잘못된 문서 종류입니다.')

    try:
        binder_size = int(binder_size_raw)
    except (ValueError, TypeError):
        raise ValidationError('잘못된 바인더 크기입니다.')

    if binder_size not in VALID_BINDER_SIZES:
        raise ValidationError('잘못된 바인더 크기입니다.')

    if doc_type == '2' and binder_size == 1:
        raise ValidationError('과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.')

    required = EQUIPMENT_REQUIRED_FIELDS if doc_type == '1' else PROJECT_REQUIRED_FIELDS
    for field in required:
        if not form_data.get(field):
            raise ValidationError(f'필수 필드가 누락되었습니다: {field}')

    if doc_type == '1':
        data = {
            'eq_number': validate_and_clean_input(form_data.get('eq_number')),
            'eq_doc_number': validate_and_clean_input(form_data.get('eq_doc_number')),
            'eq_doc_title': validate_and_clean_input(form_data.get('eq_doc_title')),
            'eq_doc_count': safe_int_conversion(form_data.get('eq_doc_count')),
            'eq_doc_department': validate_and_clean_input(form_data.get('eq_doc_department')),
            'eq_doc_year': safe_int_conversion(form_data.get('eq_doc_year'), datetime.now().year),
        }
    else:
        data = {
            'pjt_number': validate_and_clean_input(form_data.get('pjt_number')),
            'pjt_test_number': validate_and_clean_input(form_data.get('pjt_test_number')),
            'pjt_doc_title': validate_and_clean_input(form_data.get('pjt_doc_title')),
            'pjt_doc_writer': validate_and_clean_input(form_data.get('pjt_doc_writer')),
            'pjt_doc_count': safe_int_conversion(form_data.get('pjt_doc_count')),
        }

    return data, doc_type, binder_size


def get_doc_count(data: dict, doc_type: str) -> int:
    """Return the document count (권수) from parsed label data."""
    key = 'eq_doc_count' if doc_type == '1' else 'pjt_doc_count'
    return data[key]


@dataclass
class EquipmentLabel:
    """Owns the equipment label's field values, cell mapping, and QR payload."""

    DOC_TYPE: ClassVar[str] = '1'
    TITLE_CELL: ClassVar[str] = 'B4'

    eq_number: str
    eq_doc_number: str
    eq_doc_title: str
    eq_doc_count: int
    eq_doc_department: str
    eq_doc_year: int

    @property
    def doc_number(self) -> str:
        return self.eq_doc_number

    @property
    def doc_count(self) -> int:
        return self.eq_doc_count

    def cell_values(self) -> dict:
        """Map cell address → value for Sheet 1."""
        return {
            'B2': self.eq_number,
            'B3': self.eq_doc_number,
            'B4': self.eq_doc_title,
            'B5': f"1/{self.eq_doc_count}",
            'B6': self.eq_doc_department,
            'B7': self.eq_doc_year,
        }

    def qr_payload(self, sheet_idx: int, total: int) -> str:
        """Pipe-delimited QR payload for sheet sheet_idx (1-based)."""
        return '|'.join([
            str(self.eq_number),
            str(self.eq_doc_number),
            str(self.eq_doc_title),
            str(self.eq_doc_department),
            str(self.eq_doc_year),
            f"{sheet_idx}/{total}",
        ])


@dataclass
class ProjectLabel:
    """Owns the project label's field values, cell mapping, and QR payload."""

    DOC_TYPE: ClassVar[str] = '2'
    TITLE_CELL: ClassVar[str] = 'B4'

    pjt_number: str
    pjt_test_number: str
    pjt_doc_title: str
    pjt_doc_writer: str
    pjt_doc_count: int

    @property
    def doc_number(self) -> str:
        return self.pjt_test_number

    @property
    def doc_count(self) -> int:
        return self.pjt_doc_count

    def cell_values(self) -> dict:
        """Map cell address → value for Sheet 1 (including secondary panel)."""
        count_str = f"1/{self.pjt_doc_count}"
        return {
            'B2': self.pjt_number,
            'B3': self.pjt_test_number,
            'B4': self.pjt_doc_title,
            'B5': count_str,
            'B6': self.pjt_doc_writer,
            'Q21': f"[{self.pjt_number}] {self.pjt_test_number}",
            'Q22': self.pjt_doc_title,
            'R23': self.pjt_doc_writer,
            'S23': count_str,
        }

    def qr_payload(self, sheet_idx: int, total: int) -> str:
        """Pipe-delimited QR payload for sheet sheet_idx (1-based)."""
        return '|'.join([
            str(self.pjt_number),
            str(self.pjt_test_number),
            str(self.pjt_doc_title),
            str(self.pjt_doc_writer),
            f"{sheet_idx}/{total}",
        ])


def make_label(data: dict, doc_type: str) -> Union[EquipmentLabel, ProjectLabel]:
    """Factory: create the appropriate label object from parsed data."""
    if doc_type == '1':
        return EquipmentLabel(
            eq_number=data['eq_number'],
            eq_doc_number=data['eq_doc_number'],
            eq_doc_title=data['eq_doc_title'],
            eq_doc_count=data['eq_doc_count'],
            eq_doc_department=data['eq_doc_department'],
            eq_doc_year=data['eq_doc_year'],
        )
    return ProjectLabel(
        pjt_number=data['pjt_number'],
        pjt_test_number=data['pjt_test_number'],
        pjt_doc_title=data['pjt_doc_title'],
        pjt_doc_writer=data['pjt_doc_writer'],
        pjt_doc_count=data['pjt_doc_count'],
    )
