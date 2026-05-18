# tests/test_qr_paste.py
import io
import json
import os
import pytest
from tests.conftest import make_png_bytes, make_jpeg_bytes, make_multipart_qr, build_multipart_data


FORM_BASE_EQ = {
    'doc_type': '1',
    'binder_size': '3',
    'eq_number': 'MC-001',
    'eq_doc_number': 'DOC-001',
    'eq_doc_title': '테스트',
    'eq_doc_count': '2',
    'eq_doc_department': '개발',
    'eq_doc_year': '2026',
}


class TestCreateLabelPasteFlow:
    def test_correct_n_files_returns_xlsx(self, client, tmp_path):
        files, order = make_multipart_qr(2, tmp_path)
        form = {**FORM_BASE_EQ, 'qr_order': order}
        data = build_multipart_data(form, files)
        resp = client.post('/create_label', data=data,
                           content_type='multipart/form-data')
        assert resp.status_code == 200
        assert b'PK' in resp.data[:4]  # xlsx는 ZIP 포맷

    def test_file_count_mismatch_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(1, tmp_path)  # 권수=2인데 파일 1개
        form = {**FORM_BASE_EQ, 'qr_order': json.dumps([0])}
        data = build_multipart_data(form, files)
        resp = client.post('/create_label', data=data,
                           content_type='multipart/form-data')
        assert resp.status_code == 400
        body = resp.get_json()
        assert '권수' in body.get('error', '')

    def test_qr_order_length_mismatch_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(2, tmp_path)
        form = {**FORM_BASE_EQ, 'qr_order': json.dumps([0])}  # 파일 2개인데 order 1개
        data = build_multipart_data(form, files)
        resp = client.post('/create_label', data=data,
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_qr_order_out_of_range_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(2, tmp_path)
        form = {**FORM_BASE_EQ, 'qr_order': json.dumps([0, 5])}  # 5는 범위 초과
        data = build_multipart_data(form, files)
        resp = client.post('/create_label', data=data,
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_qr_order_duplicate_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(2, tmp_path)
        form = {**FORM_BASE_EQ, 'qr_order': json.dumps([0, 0])}  # 중복
        data = build_multipart_data(form, files)
        resp = client.post('/create_label', data=data,
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_non_png_file_returns_400(self, client, tmp_path):
        jpeg_bytes = make_jpeg_bytes()
        files = [('qr_images', (io.BytesIO(jpeg_bytes), 'bad.jpg', 'image/jpeg'))]
        form = {**FORM_BASE_EQ, 'eq_doc_count': '1', 'qr_order': json.dumps([0])}
        data = build_multipart_data(form, files)
        resp = client.post('/create_label', data=data,
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_api_create_label_still_auto_generates(self, client):
        # /api/create_label (JSON) 자동 QR 흐름 회귀 방지
        payload = {
            'doc_type': '1',
            'binder_size': 3,
            'eq_number': 'MC-001',
            'eq_doc_number': 'DOC-001',
            'eq_doc_title': '테스트',
            'eq_doc_count': 1,
            'eq_doc_department': '개발',
            'eq_doc_year': 2026,
        }
        resp = client.post('/api/create_label',
                           json=payload,
                           content_type='application/json')
        assert resp.status_code == 200
        body = resp.get_json()
        assert body.get('success') is True
