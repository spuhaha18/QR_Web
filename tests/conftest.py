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
