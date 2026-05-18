import io
import pytest
from PIL import Image as PILImage


def make_png_bytes(width=50, height=50) -> bytes:
    """유효한 PNG 바이트 생성 (Pillow 사용)."""
    img = PILImage.new("RGB", (width, height), color=(255, 255, 255))
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    return buf.getvalue()


def make_jpeg_bytes(width=50, height=50) -> bytes:
    """유효한 JPEG 바이트 생성."""
    img = PILImage.new("RGB", (width, height), color=(200, 200, 200))
    buf = io.BytesIO()
    img.save(buf, format="JPEG")
    return buf.getvalue()


@pytest.fixture
def valid_png():
    return make_png_bytes()


@pytest.fixture
def valid_jpeg():
    return make_jpeg_bytes()


import sys, os
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))

import pytest
from app import app as flask_app


@pytest.fixture
def client():
    flask_app.config['TESTING'] = True
    flask_app.config['WTF_CSRF_ENABLED'] = False
    with flask_app.test_client() as c:
        yield c


def make_multipart_qr(n: int, tmp_path):
    """n개 PNG 파일과 qr_order JSON을 반환.

    반환값:
      files  — [('qr_images', (fileobj, filename, mimetype)), ...]  (werkzeug 순서)
      order  — JSON 문자열 (예: "[0, 1]")
    """
    import json
    files = []
    for i in range(n):
        path = tmp_path / f"qr_{i}.png"
        img = PILImage.new("RGB", (50, 50), color=(i * 20, 0, 0))
        img.save(path, format="PNG")
        files.append(('qr_images', (open(path, 'rb'), f'qr_{i}.png', 'image/png')))
    order = list(range(n))
    return files, json.dumps(order)


def build_multipart_data(form_fields: dict, files: list):
    """form_fields dict + files list → werkzeug MultiDict (multipart 전송용)."""
    from werkzeug.datastructures import MultiDict
    pairs = list(form_fields.items()) + files
    return MultiDict(pairs)
