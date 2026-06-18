# Paste 기반 QR 라벨 생성 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 사용자가 외부 프로젝트에서 복사한 QR 이미지를 paste(Ctrl+V)로 입력하면, 바인더 사이즈에 맞는 라벨 엑셀을 생성해 다운로드한다.

**Architecture:** 브라우저 paste 이벤트 → Blob 수집 + SHA-1 중복 차단 + SortableJS 드래그 순서 → FormData fetch POST → 서버 Pillow 검증 + 임시 저장 → `create_label_excel(qr_image_paths=[...])` → openpyxl 시트별 이미지 삽입 → xlsx 다운로드. JSON API(`/api/create_label`)는 자동 QR 생성 흐름을 그대로 유지해 외부 호출자 보호.

**Tech Stack:** Flask 3, openpyxl, Pillow, SortableJS (정적 번들), Web Crypto API (SHA-1), Fetch API, pytest

---

## 파일 맵

| 역할 | 파일 | 변경 |
|---|---|---|
| 상수 | `config.py` | `MAX_QR_FILES`, `MAX_QR_FILE_SIZE` 추가 |
| 검증 헬퍼 | `utils.py` | `validate_qr_image_bytes`, `delete_dir_later` 추가 |
| 폼 라우트 | `app.py` | `create_label()` paste 흐름 대체 (기존 자동 QR 흐름 제거) |
| API 라우트 | `app.py` `/api/create_label` | 변경 없음 |
| 엑셀 생성기 | `excel_generator.py` | `create_label_excel`, `_apply_qr_codes`에 `qr_image_paths=None` 분기 추가 |
| HTML | `templates/index.html` | `enctype` + QR 섹션 + hidden `qr_order` |
| JS | `static/js/qr_paste.js` | 신규 — paste/hash/sort/submit |
| CSS | `static/css/style.css` | dropzone/thumbnail/counter 스타일 추가 |
| SortableJS | `static/vendor/sortablejs/Sortable.min.js` | 신규 정적 번들 |
| 테스트 | `tests/conftest.py` | 신규 — Flask test client fixture |
| 테스트 | `tests/test_qr_paste.py` | 신규 — 서버 검증 테스트 |
| 테스트 | `tests/test_utils_qr.py` | 신규 — validate_qr_image_bytes 단위 테스트 |

---

## Task 1: 상수 + SortableJS 추가

**Files:**
- Modify: `config.py:42-44`
- Create: `static/vendor/sortablejs/Sortable.min.js`

- [ ] **Step 1: SortableJS 다운로드**

```bash
mkdir -p static/vendor/sortablejs
curl -L https://cdn.jsdelivr.net/npm/sortablejs@1.15.6/Sortable.min.js \
  -o static/vendor/sortablejs/Sortable.min.js
# 크기 확인 (예상 ~13KB)
wc -c static/vendor/sortablejs/Sortable.min.js
```

사내 네트워크가 외부 접근 차단 시: [Sortable 1.15.6 릴리즈 페이지](https://github.com/SortableJS/Sortable/releases/tag/1.15.6)에서 `Sortable.min.js` 직접 다운로드 후 경로에 복사.

- [ ] **Step 2: config.py에 상수 추가**

`config.py:42` 의 `ALLOWED_EXTENSIONS` 블록 아래에 추가:

```python
    # QR 이미지 업로드 설정 (paste 모드)
    MAX_QR_FILES = int(os.environ.get('MAX_QR_FILES', 50))
    MAX_QR_FILE_SIZE = int(os.environ.get('MAX_QR_FILE_SIZE', 2 * 1024 * 1024))  # 2MB
```

- [ ] **Step 3: 커밋**

```bash
git add config.py static/vendor/sortablejs/Sortable.min.js
git commit -m "feat: add QR paste constants and vendor SortableJS"
```

---

## Task 2: validate_qr_image_bytes + delete_dir_later 단위 테스트

**Files:**
- Create: `tests/conftest.py`
- Create: `tests/test_utils_qr.py`

- [ ] **Step 1: pytest 설치 확인**

```bash
uv add --dev pytest
python -m pytest --version
```

Expected: `pytest X.Y.Z`

- [ ] **Step 2: `tests/conftest.py` 작성**

```python
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
```

- [ ] **Step 3: 실패하는 테스트 작성 (`tests/test_utils_qr.py`)**

```python
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
```

- [ ] **Step 4: 실패 확인**

```bash
python -m pytest tests/test_utils_qr.py -v
```

Expected: 모두 FAIL (`ImportError: cannot import name 'validate_qr_image_bytes'`)

- [ ] **Step 5: 커밋 (실패 테스트 포함)**

```bash
git add tests/
git commit -m "test: add failing tests for validate_qr_image_bytes"
```

---

## Task 3: validate_qr_image_bytes + delete_dir_later 구현

**Files:**
- Modify: `utils.py`

- [ ] **Step 1: `validate_qr_image_bytes` 구현**

`utils.py` 상단 import에 추가:
```python
import io
from PIL import Image as PILImage
```

`utils.py` 맨 아래에 추가:

```python
def validate_qr_image_bytes(data: bytes) -> bool:
    """PNG 바이트 유효성 검사. Pillow verify + PNG 형식 강제."""
    if not data:
        return False
    try:
        img = PILImage.open(io.BytesIO(data))
        img.verify()          # 파일 손상/위조 검사 (verify 후 img 재사용 불가)
        img2 = PILImage.open(io.BytesIO(data))
        return img2.format == 'PNG'
    except Exception:
        return False


def delete_dir_later(dirpath: str, delay: int = 600):
    """지정된 시간 후 디렉토리를 재귀 삭제한다."""
    import shutil

    def _delete():
        time.sleep(delay)
        if os.path.isdir(dirpath):
            try:
                shutil.rmtree(dirpath)
                logger.info(f"Temp dir deleted: {dirpath}")
            except OSError as e:
                logger.error(f"Failed to delete temp dir {dirpath}: {e}")

    threading.Thread(target=_delete, daemon=True).start()
```

- [ ] **Step 2: 테스트 통과 확인**

```bash
python -m pytest tests/test_utils_qr.py -v
```

Expected: 모두 PASS

- [ ] **Step 3: 커밋**

```bash
git add utils.py
git commit -m "feat: add validate_qr_image_bytes and delete_dir_later"
```

---

## Task 4: excel_generator.py — paste 모드 분기 + 테스트

**Files:**
- Modify: `excel_generator.py:287-341` (`_apply_qr_codes`)
- Modify: `excel_generator.py:357` (`create_label_excel`)
- Create: `tests/test_excel_paste_mode.py`

- [ ] **Step 1: 실패하는 테스트 작성**

```python
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
```

- [ ] **Step 2: 실패 확인**

```bash
python -m pytest tests/test_excel_paste_mode.py -v
```

Expected: FAIL (`create_label_excel() got unexpected keyword argument 'qr_image_paths'`)

- [ ] **Step 3: `_apply_qr_codes` 시그니처 확장 + paste 분기**

`excel_generator.py:287` 의 `def _apply_qr_codes(self, wb, doc_type, binder_size, base_filename):` 를 아래로 교체:

```python
def _apply_qr_codes(self, wb, doc_type, binder_size, base_filename, qr_image_paths=None):
    """QR 코드를 시트에 추가한다. qr_image_paths가 None이면 자동 생성, 아니면 paste 모드."""
    logger.info(f"Applying QR codes to {len(wb.worksheets)} sheets "
                f"({'paste' if qr_image_paths else 'auto'} mode)")
    img_files = []

    binder_configs = {
        7: {'column_width': 1.875, 'cell_pos': 'E9' if doc_type == '1' else 'E8'},
        5: {'column_width': 1.25, 'cell_pos': 'D9' if doc_type == '1' else 'D8'},
        3: {'column_width': 1,    'cell_pos': 'D9' if doc_type == '1' else 'D8'},
        1: {'column_width': 0.75, 'cell_pos': 'B9'},
    }
    config = binder_configs.get(binder_size, binder_configs[3])

    for ws_sheet in wb.worksheets:
        for col in range(ord('B'), ord('N')):
            ws_sheet.column_dimensions[chr(col)].width = config['column_width']

    for idx, sheet in enumerate(wb.worksheets):
        try:
            if qr_image_paths is not None:
                # paste 모드: 전달된 경로 순서대로 삽입
                img_file = qr_image_paths[idx]
            else:
                # 자동 생성 모드 (기존 로직 그대로)
                if doc_type == '1':
                    qr_text = "|".join([
                        str(sheet["B2"].value), str(sheet["B3"].value),
                        str(sheet["B4"].value), str(sheet["B6"].value),
                        str(sheet["B7"].value), str(sheet["B5"].value)
                    ])
                else:
                    qr_text = "|".join([
                        str(sheet["B2"].value), str(sheet["B3"].value),
                        str(sheet["B4"].value), str(sheet["B6"].value),
                        str(sheet["B5"].value)
                    ])
                img_file = self.qr_generator.create_qr_for_excel(
                    qr_text, self.upload_folder, f"{base_filename}_{sheet.title}"
                )
                img_files.append(img_file)

            img_obj = Image(img_file)
            img_obj.width = 75
            img_obj.height = 75
            sheet.add_image(img_obj, config['cell_pos'])

        except Exception as e:
            logger.error(f"Failed to add QR code to sheet {sheet.title}: {e}")
            continue

    return img_files  # auto 모드만 정리 대상 (paste 모드 임시파일은 호출자가 정리)
```

- [ ] **Step 4: `create_label_excel` 시그니처 확장**

`excel_generator.py:357`:
```python
def create_label_excel(self, doc_type, binder_size, data, qr_image_paths=None):
```

같은 함수 내 `_apply_qr_codes` 호출 라인 (`excel_generator.py:388` 근처) 변경:
```python
img_files = self._apply_qr_codes(wb, doc_type, binder_size, base_filename, qr_image_paths)
```

- [ ] **Step 5: 테스트 통과 확인**

```bash
python -m pytest tests/test_excel_paste_mode.py -v
```

Expected: 3개 모두 PASS

- [ ] **Step 6: 커밋**

```bash
git add excel_generator.py tests/test_excel_paste_mode.py
git commit -m "feat: add paste mode to create_label_excel and _apply_qr_codes"
```

---

## Task 5: app.py — create_label() paste 흐름 + 테스트

**Files:**
- Create: `tests/test_qr_paste.py`
- Modify: `app.py:146-206` (`create_label()`)

- [ ] **Step 1: Flask test client conftest 확장**

`tests/conftest.py` 맨 아래에 추가:

```python
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
    """n개 PNG 파일과 qr_order JSON을 반환."""
    import json
    files = []
    for i in range(n):
        path = tmp_path / f"qr_{i}.png"
        img = PILImage.new("RGB", (50, 50), color=(i * 20, 0, 0))
        img.save(path, format="PNG")
        files.append(('qr_images', (f'qr_{i}.png', open(path, 'rb'), 'image/png')))
    order = list(range(n))
    return files, json.dumps(order)
```

- [ ] **Step 2: 실패하는 테스트 작성**

```python
# tests/test_qr_paste.py
import io
import json
import os
import pytest
from tests.conftest import make_png_bytes, make_jpeg_bytes, make_multipart_qr


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
        data = {**FORM_BASE_EQ, 'qr_order': order}
        resp = client.post('/create_label', data={**data, **dict(files)},
                           content_type='multipart/form-data')
        assert resp.status_code == 200
        assert b'PK' in resp.data[:4]  # xlsx는 ZIP 포맷

    def test_file_count_mismatch_returns_400(self, client, tmp_path):
        files, order = make_multipart_qr(1, tmp_path)  # 권수=2인데 파일 1개
        data = {**FORM_BASE_EQ, 'qr_order': json.dumps([0])}
        resp = client.post('/create_label', data={**data, **dict(files)},
                           content_type='multipart/form-data')
        assert resp.status_code == 400
        body = resp.get_json()
        assert '권수' in body.get('error', '')

    def test_qr_order_length_mismatch_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(2, tmp_path)
        data = {**FORM_BASE_EQ, 'qr_order': json.dumps([0])}  # 파일 2개인데 order 1개
        resp = client.post('/create_label', data={**data, **dict(files)},
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_qr_order_out_of_range_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(2, tmp_path)
        data = {**FORM_BASE_EQ, 'qr_order': json.dumps([0, 5])}  # 5는 범위 초과
        resp = client.post('/create_label', data={**data, **dict(files)},
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_qr_order_duplicate_returns_400(self, client, tmp_path):
        files, _ = make_multipart_qr(2, tmp_path)
        data = {**FORM_BASE_EQ, 'qr_order': json.dumps([0, 0])}  # 중복
        resp = client.post('/create_label', data={**data, **dict(files)},
                           content_type='multipart/form-data')
        assert resp.status_code == 400

    def test_non_png_file_returns_400(self, client, tmp_path):
        jpeg_bytes = make_jpeg_bytes()
        files = [('qr_images', ('bad.jpg', io.BytesIO(jpeg_bytes), 'image/jpeg'))]
        data = {**FORM_BASE_EQ, 'eq_doc_count': '1', 'qr_order': json.dumps([0])}
        resp = client.post('/create_label', data={**data, **dict(files)},
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
```

- [ ] **Step 3: 실패 확인**

```bash
python -m pytest tests/test_qr_paste.py -v
```

Expected: 대부분 FAIL (paste 검증 로직 없음)

- [ ] **Step 4: `create_label()` paste 흐름으로 교체**

`app.py` 상단 import에 추가:
```python
import json
import tempfile
import shutil
from utils import validate_qr_image_bytes, delete_dir_later
```

`app.py:146` 의 `create_label()` 함수 전체를 아래로 교체:

```python
@app.route('/create_label', methods=['POST'])
@handle_errors
@monitor_performance("web_label_creation")
def create_label():
    """라벨 생성 (웹 인터페이스) — paste 모드."""
    client_ip, _ = get_client_info()
    logger.info(f"Create label request received from {client_ip}")

    # 기본 폼 필드 검증
    doc_type = request.form.get('doc_type')
    try:
        binder_size = int(request.form.get('binder_size'))
    except (ValueError, TypeError):
        return jsonify({'error': '잘못된 바인더 크기입니다.'}), 400

    if not validate_document_type(doc_type):
        return jsonify({'error': '잘못된 문서 종류입니다.'}), 400

    if not validate_binder_size(binder_size, doc_type):
        return jsonify({'error': '과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.'}), 400

    is_valid, missing_field = validate_required_fields(request.form, doc_type)
    if not is_valid:
        return jsonify({'error': f'필수 필드가 누락되었습니다: {missing_field}'}), 400

    # doc_count 추출
    count_key = 'eq_doc_count' if doc_type == '1' else 'pjt_doc_count'
    try:
        doc_count = int(request.form.get(count_key, 0))
    except (ValueError, TypeError):
        return jsonify({'error': '권수가 올바르지 않습니다.'}), 400

    # QR 이미지 파일 수신
    qr_files = request.files.getlist('qr_images')

    # qr_order 수신 + 파싱
    try:
        qr_order = json.loads(request.form.get('qr_order', '[]'))
        if not isinstance(qr_order, list):
            raise ValueError
    except (ValueError, TypeError):
        return jsonify({'error': 'qr_order 형식이 올바르지 않습니다.'}), 400

    # ── 검증 ──
    if len(qr_files) != doc_count:
        return jsonify({
            'error': f'QR 이미지 수가 권수와 다릅니다 (받음: {len(qr_files)}, 권수: {doc_count})'
        }), 400

    if len(qr_files) > config.MAX_QR_FILES:
        return jsonify({'error': f'QR 이미지는 최대 {config.MAX_QR_FILES}개까지 허용됩니다.'}), 400

    if len(qr_order) != doc_count:
        return jsonify({'error': 'qr_order 길이가 권수와 다릅니다.'}), 400

    if sorted(qr_order) != list(range(doc_count)):
        return jsonify({'error': 'qr_order에 중복이나 범위 초과 인덱스가 있습니다.'}), 400

    # 각 파일 크기 + PNG 검증
    file_bytes_list = []
    for f in qr_files:
        raw = f.read()
        if len(raw) > config.MAX_QR_FILE_SIZE:
            return jsonify({'error': f'QR 이미지 크기가 2MB를 초과합니다: {f.filename}'}), 400
        if not validate_qr_image_bytes(raw):
            return jsonify({'error': f'유효하지 않은 PNG 이미지입니다: {f.filename}'}), 400
        file_bytes_list.append(raw)

    # qr_order 순서대로 재정렬
    ordered_bytes = [file_bytes_list[i] for i in qr_order]

    # 임시 디렉토리에 저장
    tmp_dir = tempfile.mkdtemp(prefix='qr_paste_')
    qr_paths = []
    try:
        for idx, raw in enumerate(ordered_bytes):
            path = os.path.join(tmp_dir, f'qr_{idx}.png')
            with open(path, 'wb') as fh:
                fh.write(raw)
            qr_paths.append(path)

        data = process_form_data(request.form, doc_type)
        filepath, filename = excel_generator.create_label_excel(
            doc_type, binder_size, data, qr_image_paths=qr_paths
        )
    except Exception:
        shutil.rmtree(tmp_dir, ignore_errors=True)
        raise

    # 엑셀 파일 전송 후 임시 디렉토리 정리
    delete_file_later(filepath, DELETE_DELAY)
    delete_dir_later(tmp_dir, delay=60)

    logger.info(f"Paste-mode label generated: {filename} for {client_ip}")
    response = make_response(send_file(filepath, as_attachment=True, download_name=filename))
    response.set_cookie('download_complete', 'true', max_age=10)
    return response
```

- [ ] **Step 5: 테스트 통과 확인**

```bash
python -m pytest tests/test_qr_paste.py -v
```

Expected: 모두 PASS

- [ ] **Step 6: 커밋**

```bash
git add app.py tests/test_qr_paste.py tests/conftest.py
git commit -m "feat: replace create_label() with paste-mode QR flow"
```

---

## Task 6: HTML — paste 섹션 + form 수정

**Files:**
- Modify: `templates/index.html:319` (`<form>` 태그)
- Modify: `templates/index.html:428` (submit 버튼 위)

- [ ] **Step 1: `<form>` 태그 수정**

`templates/index.html:319` 의 `<form action="/create_label" method="post">` 를:

```html
<form action="/create_label" method="post" enctype="multipart/form-data" id="label-form">
```

- [ ] **Step 2: QR 이미지 섹션 추가**

`templates/index.html:429` (현재 `</div>` 닫는 태그, `form-sections` 끝 바로 전) 위에 삽입:

```html
                <div class="form-section" id="qr_section">
                    <div class="section-title">
                        <i data-lucide="qr-code"></i> QR 이미지
                    </div>
                    <div class="qr-dropzone" id="qr_dropzone" tabindex="0">
                        <i data-lucide="clipboard-paste"></i>
                        <p>여기를 클릭한 후 <kbd>Ctrl+V</kbd>로 QR 이미지를 붙여넣으세요</p>
                        <p class="qr-hint">새 프로젝트에서 QR 이미지 우클릭 → "이미지 복사" 후 붙여넣기</p>
                    </div>
                    <div class="qr-counter" id="qr_counter">0 / <span id="qr_total">1</span></div>
                    <ul class="qr-thumbnails sortable" id="qr_thumbnails"></ul>
                    <input type="hidden" id="qr_order" name="qr_order" value="[]">
                </div>
```

- [ ] **Step 3: SortableJS + qr_paste.js 스크립트 로드**

`templates/index.html` 맨 아래 `</body>` 직전에 추가:

```html
    <script src="{{ url_for('static', filename='vendor/sortablejs/Sortable.min.js') }}"></script>
    <script src="{{ url_for('static', filename='js/qr_paste.js') }}"></script>
```

- [ ] **Step 4: 커밋**

```bash
git add templates/index.html
git commit -m "feat: add QR paste section to index.html form"
```

---

## Task 7: qr_paste.js — paste/hash/sort/submit

**Files:**
- Create: `static/js/qr_paste.js`

- [ ] **Step 1: `static/js/qr_paste.js` 작성**

```javascript
// qr_paste.js — paste-mode QR 이미지 수집 + SortableJS 순서 관리 + FormData fetch 제출

(function () {
  'use strict';

  /** @type {{ id: number, blob: Blob, hash: string, url: string }[]} */
  const state = { images: [], nextId: 0 };

  const dropzone = document.getElementById('qr_dropzone');
  const thumbnailList = document.getElementById('qr_thumbnails');
  const counterEl = document.getElementById('qr_counter');
  const totalEl = document.getElementById('qr_total');
  const orderInput = document.getElementById('qr_order');
  const form = document.getElementById('label-form');

  // ── SHA-1 해시 (Web Crypto API) ─────────────────────────────────────────
  async function sha1Hex(arrayBuffer) {
    const hashBuf = await crypto.subtle.digest('SHA-1', arrayBuffer);
    return Array.from(new Uint8Array(hashBuf))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  }

  // ── 카운터 갱신 ──────────────────────────────────────────────────────────
  function getDocCount() {
    const docType = document.getElementById('doc_type').value;
    const key = docType === '1' ? 'eq_doc_count' : 'pjt_doc_count';
    return parseInt(document.getElementById(key)?.value || '1', 10);
  }

  function updateCounter() {
    const n = getDocCount();
    totalEl.textContent = n;
    const m = state.images.length;
    counterEl.textContent = `${m} / ${n}`;
    counterEl.className = 'qr-counter' + (m === n ? ' ok' : m > n ? ' over' : ' under');
  }

  // ── 썸네일 DOM 동기화 ─────────────────────────────────────────────────────
  function renderThumbnails() {
    thumbnailList.innerHTML = '';
    state.images.forEach(({ id, url }) => {
      const li = document.createElement('li');
      li.className = 'qr-thumb-item';
      li.dataset.id = id;
      li.innerHTML = `
        <img src="${url}" alt="QR ${id}" />
        <button type="button" class="qr-remove-btn" data-id="${id}">×</button>
      `;
      thumbnailList.appendChild(li);
    });
    syncOrder();
    updateCounter();
  }

  // ── qr_order hidden input 동기화 ─────────────────────────────────────────
  function syncOrder() {
    const domIds = [...thumbnailList.querySelectorAll('[data-id]')]
      .map(el => parseInt(el.dataset.id, 10));
    // DOM 순서 → 원본 state 배열의 인덱스로 변환
    const order = domIds.map(id => state.images.findIndex(img => img.id === id));
    orderInput.value = JSON.stringify(order);
  }

  // ── SortableJS 초기화 ────────────────────────────────────────────────────
  if (window.Sortable) {
    Sortable.create(thumbnailList, {
      animation: 150,
      onEnd: syncOrder,
    });
  }

  // ── 삭제 버튼 이벤트 위임 ────────────────────────────────────────────────
  thumbnailList.addEventListener('click', (e) => {
    const btn = e.target.closest('.qr-remove-btn');
    if (!btn) return;
    const id = parseInt(btn.dataset.id, 10);
    const img = state.images.find(i => i.id === id);
    if (img) URL.revokeObjectURL(img.url);
    state.images = state.images.filter(i => i.id !== id);
    renderThumbnails();
  });

  // ── paste 핸들러 ─────────────────────────────────────────────────────────
  document.addEventListener('paste', async (e) => {
    // dropzone이 포커스되거나 폼 영역 내부에 paste 이벤트가 오면 처리
    if (!dropzone.matches(':focus') && !form.contains(document.activeElement)) return;

    const items = Array.from(e.clipboardData?.items || []);
    const imageItem = items.find(item => item.type === 'image/png');
    if (!imageItem) return;

    e.preventDefault();
    const blob = imageItem.getAsFile();
    if (!blob) return;

    const arrayBuffer = await blob.arrayBuffer();
    const hash = await sha1Hex(arrayBuffer);

    if (state.images.some(img => img.hash === hash)) {
      showToast('중복된 QR 이미지입니다.');
      return;
    }

    const url = URL.createObjectURL(blob);
    state.images.push({ id: state.nextId++, blob, hash, url });
    renderThumbnails();
  });

  // dropzone 클릭 시 포커스 (paste 핸들러 활성화)
  dropzone.addEventListener('click', () => dropzone.focus());

  // doc_type 변경 시 카운터 갱신 (index.html의 selectDocType 호출 후)
  document.querySelectorAll('[data-value][onclick*="selectDocType"]').forEach(btn => {
    btn.addEventListener('click', () => setTimeout(updateCounter, 0));
  });
  document.querySelectorAll('#eq_doc_count, #pjt_doc_count').forEach(input => {
    input.addEventListener('input', updateCounter);
  });

  // ── 폼 submit 가로채기 ────────────────────────────────────────────────────
  form.addEventListener('submit', async (e) => {
    e.preventDefault();

    const n = getDocCount();
    if (state.images.length !== n) {
      alert(`QR 이미지 수(${state.images.length})가 권수(${n})와 다릅니다. 맞춰주세요.`);
      return;
    }

    // qr_order를 현재 DOM 순서 기준 인덱스 배열로 최종 갱신
    syncOrder();

    const formData = new FormData(form);
    // paste된 Blob을 File로 변환하여 추가
    state.images.forEach((img, i) => {
      formData.append('qr_images', new File([img.blob], `qr_${i}.png`, { type: 'image/png' }));
    });

    const submitBtn = form.querySelector('.submit-btn');
    if (submitBtn) { submitBtn.disabled = true; submitBtn.textContent = '생성 중...'; }

    try {
      const resp = await fetch('/create_label', { method: 'POST', body: formData });
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({ error: '서버 오류가 발생했습니다.' }));
        alert(err.error || '서버 오류가 발생했습니다.');
        return;
      }
      // 엑셀 다운로드 트리거
      const disposition = resp.headers.get('Content-Disposition') || '';
      const match = disposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
      const filename = match ? match[1].replace(/['"]/g, '') : '라벨.xlsx';
      const blob = await resp.blob();
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(a.href);
    } catch (err) {
      alert('요청 중 오류가 발생했습니다: ' + err.message);
    } finally {
      if (submitBtn) { submitBtn.disabled = false; submitBtn.textContent = '라벨 만들기'; }
    }
  });

  // ── 토스트 메시지 ─────────────────────────────────────────────────────────
  function showToast(msg) {
    const t = document.createElement('div');
    t.className = 'qr-toast';
    t.textContent = msg;
    document.body.appendChild(t);
    setTimeout(() => t.remove(), 2500);
  }

  // 초기 카운터
  updateCounter();
})();
```

- [ ] **Step 2: 커밋**

```bash
git add static/js/qr_paste.js
git commit -m "feat: add qr_paste.js with paste/hash/sort/fetch flow"
```

---

## Task 8: CSS — dropzone/thumbnail/counter 스타일

**Files:**
- Modify: `static/css/style.css` (맨 아래에 추가)

- [ ] **Step 1: 스타일 추가**

`static/css/style.css` 맨 아래에 추가:

```css
/* ── QR Paste 영역 ──────────────────────────────────────────────── */

.qr-dropzone {
  border: 2px dashed #999;
  border-radius: 8px;
  padding: 24px;
  text-align: center;
  cursor: pointer;
  color: #666;
  transition: border-color 0.2s, background 0.2s;
  user-select: none;
}

.qr-dropzone:hover,
.qr-dropzone:focus {
  border-color: #4a90d9;
  background: #f0f7ff;
  outline: none;
}

.qr-dropzone p {
  margin: 6px 0;
  font-size: 0.9rem;
}

.qr-dropzone .qr-hint {
  font-size: 0.78rem;
  color: #999;
}

.qr-dropzone kbd {
  background: #eee;
  border: 1px solid #ccc;
  border-radius: 3px;
  padding: 1px 5px;
  font-family: monospace;
  font-size: 0.85em;
}

.qr-counter {
  margin: 8px 0 4px;
  font-size: 0.88rem;
  font-weight: 600;
  color: #888;
}

.qr-counter.ok   { color: #2e7d32; }
.qr-counter.over { color: #c62828; }
.qr-counter.under { color: #e65100; }

.qr-thumbnails {
  list-style: none;
  padding: 0;
  margin: 8px 0;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.qr-thumb-item {
  position: relative;
  width: 90px;
  height: 90px;
  cursor: grab;
  border: 1px solid #ddd;
  border-radius: 6px;
  overflow: hidden;
  background: #fafafa;
}

.qr-thumb-item img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.qr-remove-btn {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-remove-btn:hover { background: #c62828; }

/* SortableJS 드래그 중 고스트 */
.sortable-ghost { opacity: 0.3; }

/* 토스트 */
.qr-toast {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  background: #323232;
  color: #fff;
  padding: 10px 20px;
  border-radius: 6px;
  font-size: 0.88rem;
  z-index: 9999;
  animation: fadeInOut 2.5s forwards;
}

@keyframes fadeInOut {
  0%   { opacity: 0; transform: translateX(-50%) translateY(10px); }
  15%  { opacity: 1; transform: translateX(-50%) translateY(0); }
  75%  { opacity: 1; }
  100% { opacity: 0; }
}
```

- [ ] **Step 2: 커밋**

```bash
git add static/css/style.css
git commit -m "feat: add QR dropzone, thumbnail, counter CSS"
```

---

## Task 9: 수동 E2E 검증 + 사전 체크

**Files:** 없음 (체크리스트만)

- [ ] **Step 1: 사전 — 우클릭 복사 동작 확인**

새 프로젝트 브라우저에서 QR 이미지 우클릭 → "이미지 복사" 실행. 이후 메모장이나 이미지 뷰어에 Ctrl+V 하여 PNG가 붙여넣어지면 OK. 안 되면 pause — grilling 재개 필요.

- [ ] **Step 2: 서버 실행**

```bash
docker compose up --build
```

Expected: `* Running on http://0.0.0.0:5000` (또는 설정된 포트)

- [ ] **Step 3: 테스트 시나리오 A — 기기 문서 2권, 바인더 3cm**

1. 브라우저에서 앱 접속
2. "기기 문서" 선택, 바인더 3cm, 총 권수 = 2, 나머지 필드 임의 입력
3. 폼 맨 아래 "QR 이미지" 영역 클릭 → 포커스 확인
4. 새 프로젝트에서 QR 1 우클릭 → "이미지 복사" → 앱에서 Ctrl+V → 썸네일 1개 표시, 카운터 "1 / 2"
5. QR 2 동일 반복 → 카운터 "2 / 2 ✓" (초록)
6. 썸네일 드래그로 순서 바꾸기
7. "라벨 만들기" 클릭 → xlsx 다운로드
8. 엑셀 열기: 시트 2개, 드래그 순서대로 QR 위치, B5 "1/2"·"2/2", 3cm 바인더 열 너비 정확

- [ ] **Step 4: 테스트 시나리오 B — 중복 paste 거부**

썸네일 1개 있는 상태에서 동일 QR 재복사 후 Ctrl+V → "중복된 QR 이미지입니다." 토스트 + 썸네일 추가 안 됨

- [ ] **Step 5: 테스트 시나리오 C — 수량 불일치 차단**

권수=3, QR 2개만 paste 후 submit → "QR 이미지 수(2)가 권수(3)와 다릅니다." alert + 파일 미다운로드

- [ ] **Step 6: 회귀 — /api/create_label JSON 흐름**

```bash
curl -s -X POST http://localhost:5000/api/create_label \
  -H 'Content-Type: application/json' \
  -d '{"doc_type":"1","binder_size":3,"eq_number":"MC-001","eq_doc_number":"DOC-001","eq_doc_title":"테스트","eq_doc_count":1,"eq_doc_department":"개발","eq_doc_year":2026}' \
  | python3 -m json.tool
```

Expected: `{"success": true, "filename": "...", "download_url": "..."}`

- [ ] **Step 7: 자동화 테스트 전체 실행**

```bash
python -m pytest tests/ -v --tb=short
```

Expected: 전체 PASS

- [ ] **Step 8: 최종 커밋**

```bash
git add .
git commit -m "feat: paste-mode QR label generation complete (Approach D)"
```

---

## 구현 후 체크리스트

- [ ] `static/vendor/sortablejs/Sortable.min.js` 존재 확인
- [ ] `CONTEXT.md` 도메인 용어 최신 (paste 입력 / QR 이미지 / 권 정의 완료)
- [ ] `/api/create_label` JSON 흐름 회귀 없음
- [ ] 임시 QR 디렉토리가 요청별로 격리되고 자동 삭제됨
- [ ] 새 프로젝트 QR 우클릭 복사 동작 사전 확인 완료

---

## 기술 노트

- **paste 이벤트 범위**: `document` 에 전역 부착, `dropzone:focus` 또는 `form 내 activeElement` 조건으로 비의도적 paste 차단
- **`Image.verify()` 주의**: `verify()` 호출 후 동일 `Image` 객체 재사용 불가 → `Image.open()` 두 번 호출
- **`binder_size` 타입**: `app.py`에서 `int()` 변환 후 전달, `excel_generator.py` 내 `binder_configs` 키도 `int` — 일치
- **`doc_type` 타입**: 폼에서 `str('1'/'2')` 로 전달, `binder_configs` 및 `_setup_*document` 내부도 문자열 비교
- **`qr_order` 검증**: `sorted(qr_order) != list(range(doc_count))` 로 범위 초과·중복·누락 한 번에 검사
