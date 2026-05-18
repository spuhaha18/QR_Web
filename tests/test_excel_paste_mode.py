# tests/test_excel_paste_mode.py
import io
import os
import tempfile
import pytest
from PIL import Image as PILImage
from excel_generator import ExcelLabelGenerator


def make_png_file(tmp_dir: str, name: str = "qr.png") -> str:
    """PNG 임시 파일 경로 반환."""
    path = os.path.join(tmp_dir, name)
    img = PILImage.new("RGB", (75, 75), color=(0, 0, 0))
    img.save(path, format="PNG")
    return path


@pytest.fixture
def generator(tmp_path):
    upload = str(tmp_path / "uploads")
    os.makedirs(upload)
    return ExcelLabelGenerator(upload_folder=upload)


@pytest.fixture
def eq_data():
    return {
        'eq_number': 'MC-001',
        'eq_doc_number': 'DOC-001',
        'eq_doc_title': '테스트 문서',
        'eq_doc_count': 2,
        'eq_doc_department': '개발팀',
        'eq_doc_year': 2026,
    }


class TestCreateLabelExcelPasteMode:
    def test_paste_mode_creates_xlsx(self, generator, eq_data, tmp_path):
        qr_dir = str(tmp_path / "qr")
        os.makedirs(qr_dir)
        paths = [make_png_file(qr_dir, f"qr_{i}.png") for i in range(2)]

        filepath, filename = generator.create_label_excel(
            doc_type='1', binder_size=3, data=eq_data, qr_image_paths=paths
        )
        assert os.path.exists(filepath)
        assert filename.endswith('.xlsx')

    def test_auto_mode_still_works(self, generator, eq_data):
        # qr_image_paths 생략 시 기존 자동 QR 경로 사용
        filepath, filename = generator.create_label_excel(
            doc_type='1', binder_size=3, data=eq_data
        )
        assert os.path.exists(filepath)

    def test_paste_mode_sheet_count_matches_doc_count(self, generator, eq_data, tmp_path):
        qr_dir = str(tmp_path / "qr")
        os.makedirs(qr_dir)
        paths = [make_png_file(qr_dir, f"qr_{i}.png") for i in range(2)]

        filepath, _ = generator.create_label_excel(
            doc_type='1', binder_size=3, data=eq_data, qr_image_paths=paths
        )
        import openpyxl
        wb = openpyxl.load_workbook(filepath)
        assert len(wb.worksheets) == 2
