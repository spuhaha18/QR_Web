import pytest
from tests.conftest import make_png_bytes, make_jpeg_bytes
from utils import validate_qr_image_bytes


class TestValidateQrImageBytes:
    def test_valid_png_returns_true(self, valid_png):
        assert validate_qr_image_bytes(valid_png) is True

    def test_jpeg_bytes_returns_false(self, valid_jpeg):
        # PNG만 허용
        assert validate_qr_image_bytes(valid_jpeg) is False

    def test_garbage_bytes_returns_false(self):
        assert validate_qr_image_bytes(b"not an image at all") is False

    def test_empty_bytes_returns_false(self):
        assert validate_qr_image_bytes(b"") is False

    def test_truncated_png_returns_false(self, valid_png):
        # 앞 20바이트만 — 손상된 PNG
        assert validate_qr_image_bytes(valid_png[:20]) is False
