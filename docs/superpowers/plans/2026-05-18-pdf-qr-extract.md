# PDF → QR 추출 입력 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** PDF 파일 1개를 업로드하면 서버가 모든 QR을 추출해 권 순서대로 `state.images`에 채워주는 기능 추가. 기존 image dropzone은 PDF 전용으로 교체.

**Architecture:** 신규 Flask 라우트 `POST /extract_qr_from_pdf`가 PyMuPDF로 페이지를 렌더링하고 pyzbar로 QR을 디코드한다. diff 휴리스틱으로 권 번호를 추론·검증한 뒤 PNG b64 + payload + vol_i JSON을 반환한다. 클라이언트는 응답으로 `state.images`를 교체한다. 기존 `/create_label` 제출 흐름은 무수정.

**Tech Stack:** Python 3.13, Flask 3, PyMuPDF (fitz), pyzbar + libzbar0, Pillow, qrcode (테스트용), pytest

---

## File Map

| 종류 | 경로 | 역할 |
|------|------|------|
| 신규 | `pdf_qr_extractor.py` | PDF→QR 추출 순수 로직 |
| 신규 | `tests/test_pdf_qr_extractor.py` | pdf_qr_extractor 단위·통합 테스트 |
| 신규 | `tests/test_extract_route.py` | /extract_qr_from_pdf 라우트 테스트 |
| 수정 | `requirements.txt` | PyMuPDF, pyzbar 추가 |
| 수정 | `Dockerfile` | libzbar0 apt 패키지 추가 |
| 수정 | `config.py` | MAX_PDF_FILE_SIZE, PDF_RENDER_DPI 추가 |
| 수정 | `utils.py` | validate_pdf_bytes 추가 |
| 수정 | `app.py` | /extract_qr_from_pdf 라우트 추가 |
| 수정 | `static/js/qr_paste.js` | addFromPdf, dropzone PDF 전용 |
| 수정 | `templates/index.html` | dropzone 카피·accept 속성 변경 |
| 수정 | `README.md` | v2.2.0 업데이트 |
| 수정 | `CONTEXT.md` | QR 이미지 입력 경로 정정 |

---

## Task 1: 의존성 추가

**Files:**
- Modify: `requirements.txt`
- Modify: `Dockerfile`

- [ ] **Step 1: requirements.txt 업데이트**

```
qrcode>=7.4.2
openpyxl>=3.1.2
waitress>=3.0.0
flask>=3.0.3
pillow>=10.3.0
PyMuPDF>=1.24.0
pyzbar>=0.1.9
```

- [ ] **Step 2: Dockerfile libzbar0 추가**

`Dockerfile` L11-14 교체:

```dockerfile
RUN apt-get update && apt-get install -y \
    --no-install-recommends \
    gcc \
    libzbar0 \
    && rm -rf /var/lib/apt/lists/*
```

- [ ] **Step 3: 호스트 환경에 libzbar0 설치 (로컬 개발)**

```bash
sudo apt-get install -y libzbar0
```

- [ ] **Step 4: 의존성 설치**

```bash
cd /home/spuhaha18/Project/QR_Web
pip install PyMuPDF pyzbar
```

Expected: 에러 없이 설치 완료. `python -c "import fitz; import pyzbar"` 오류 없음.

- [ ] **Step 5: 커밋**

```bash
git add requirements.txt Dockerfile
git commit -m "feat: add PyMuPDF and pyzbar dependencies for PDF QR extraction"
```

---

## Task 2: config.py 상수 추가

**Files:**
- Modify: `config.py:42-47`

- [ ] **Step 1: Config 클래스에 상수 추가**

`config.py`의 `# QR 이미지 업로드 설정 (paste 모드)` 블록 아래(L48 이후)에 추가:

```python
    # PDF QR 추출 설정
    MAX_PDF_FILE_SIZE = int(os.environ.get('MAX_PDF_FILE_SIZE', 50 * 1024 * 1024))  # 50MB
    PDF_RENDER_DPI = int(os.environ.get('PDF_RENDER_DPI', 300))
```

- [ ] **Step 2: 확인**

```bash
python -c "from config import get_config; c=get_config(); print(c.MAX_PDF_FILE_SIZE, c.PDF_RENDER_DPI)"
```

Expected: `52428800 300`

- [ ] **Step 3: 커밋**

```bash
git add config.py
git commit -m "feat(config): add MAX_PDF_FILE_SIZE and PDF_RENDER_DPI constants"
```

---

## Task 3: utils.py — validate_pdf_bytes

**Files:**
- Modify: `utils.py` (끝에 추가)
- Create: (테스트는 Task 5에서 test_pdf_qr_extractor.py에 통합)

- [ ] **Step 1: validate_pdf_bytes 구현**

`utils.py` 끝에 추가:

```python
def validate_pdf_bytes(data: bytes) -> bool:
    """PDF 바이트 유효성 검사. fitz로 파싱 가능한지 확인."""
    if not data:
        return False
    try:
        import fitz
        doc = fitz.open(stream=data, filetype='pdf')
        _ = doc.page_count  # 파싱 성공 시 0 이상
        return True
    except Exception:
        return False
```

- [ ] **Step 2: import에 validate_pdf_bytes 추가 확인**

`app.py` L17-20의 utils import에 나중에 추가 예정. 지금은 utils.py 구현만.

- [ ] **Step 3: 빠른 연기 테스트**

```bash
python -c "
from utils import validate_pdf_bytes
import fitz
doc = fitz.open()
doc.new_page()
assert validate_pdf_bytes(doc.tobytes()) is True
assert validate_pdf_bytes(b'not pdf') is False
assert validate_pdf_bytes(b'') is False
print('OK')
"
```

Expected: `OK`

- [ ] **Step 4: 커밋**

```bash
git add utils.py
git commit -m "feat(utils): add validate_pdf_bytes for PDF format validation"
```

---

## Task 4: pdf_qr_extractor.py — 뼈대 + _render_pages (TDD)

**Files:**
- Create: `pdf_qr_extractor.py`
- Create: `tests/test_pdf_qr_extractor.py`

- [ ] **Step 1: 실패 테스트 작성 (`tests/test_pdf_qr_extractor.py`)**

```python
# tests/test_pdf_qr_extractor.py
import io
import pytest
from pathlib import Path
from PIL import Image as PILImage


def make_minimal_pdf(page_count: int = 1) -> bytes:
    import fitz
    doc = fitz.open()
    for _ in range(page_count):
        doc.new_page(width=595, height=842)
    return doc.tobytes()


def make_qr_pil(payload: str) -> PILImage.Image:
    import qrcode
    qr = qrcode.QRCode(
        box_size=10, border=2,
        error_correction=qrcode.constants.ERROR_CORRECT_L,
    )
    qr.add_data(payload)
    qr.make(fit=True)
    return qr.make_image(fill_color='black', back_color='white').convert('RGB')


def make_qr_pdf(payloads: list) -> bytes:
    """각 payload QR을 페이지 1개에 1개씩 담은 PDF 반환."""
    import fitz
    doc = fitz.open()
    for payload in payloads:
        qr_img = make_qr_pil(payload)
        buf = io.BytesIO()
        qr_img.save(buf, format='PNG')
        buf.seek(0)
        page = doc.new_page(width=595, height=842)
        rect = fitz.Rect(100, 100, 350, 350)
        page.insert_image(rect, stream=buf.read())
    return doc.tobytes()


class TestRenderPages:
    def test_returns_pil_images(self):
        from pdf_qr_extractor import _render_pages
        pdf = make_minimal_pdf(3)
        imgs = _render_pages(pdf, dpi=72)
        assert len(imgs) == 3
        for img in imgs:
            assert isinstance(img, PILImage.Image)

    def test_empty_pdf_returns_empty_list(self):
        from pdf_qr_extractor import _render_pages
        pdf = make_minimal_pdf(0)
        imgs = _render_pages(pdf, dpi=72)
        assert imgs == []
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestRenderPages -v
```

Expected: `ModuleNotFoundError: No module named 'pdf_qr_extractor'`

- [ ] **Step 3: pdf_qr_extractor.py 뼈대 + _render_pages 구현**

```python
# pdf_qr_extractor.py
"""PDF에서 QR 코드를 추출하고 권 번호를 추론한다."""
import io
import base64
import logging
from PIL import Image as PILImage

logger = logging.getLogger(__name__)


class ExtractionError(Exception):
    def __init__(self, code: str, msg: str):
        super().__init__(msg)
        self.code = code
        self.msg = msg


def _render_pages(pdf_bytes: bytes, dpi: int = 300) -> list:
    """PDF 바이트 → PIL Image 리스트 (페이지 순서)."""
    import fitz
    doc = fitz.open(stream=pdf_bytes, filetype='pdf')
    images = []
    for page in doc:
        pix = page.get_pixmap(dpi=dpi)
        img = PILImage.frombytes('RGB', [pix.width, pix.height], pix.samples)
        images.append(img)
    return images


def _detect_qrs_in_page(img: PILImage.Image) -> list:
    """PIL Image에서 QR 디코드. [(payload_bytes, png_bytes), ...] 반환."""
    raise NotImplementedError


def _longest_common_prefix(strings: list) -> str:
    raise NotImplementedError


def _longest_common_suffix(strings: list) -> str:
    raise NotImplementedError


def _diff_payloads(payloads: list) -> tuple:
    """(prefix, suffix, variants) 반환."""
    raise NotImplementedError


def _infer_volume_indices(variants: list) -> list:
    """variants → int 권 번호 리스트. {1..N} 아니면 ExtractionError."""
    raise NotImplementedError


def extract_qrs_from_pdf(pdf_bytes: bytes) -> dict:
    """PDF bytes → {prefix, total_n, items:[{vol_i, payload, png_b64}]}."""
    raise NotImplementedError
```

- [ ] **Step 4: 테스트 실행 — 통과 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestRenderPages -v
```

Expected: `2 passed`

- [ ] **Step 5: 커밋**

```bash
git add pdf_qr_extractor.py tests/test_pdf_qr_extractor.py
git commit -m "feat(pdf-qr): scaffold extractor + _render_pages (TDD)"
```

---

## Task 5: _detect_qrs_in_page (TDD)

**Files:**
- Modify: `pdf_qr_extractor.py`
- Modify: `tests/test_pdf_qr_extractor.py`

- [ ] **Step 1: 실패 테스트 추가 (`tests/test_pdf_qr_extractor.py` 끝에 추가)**

```python
class TestDetectQrsInPage:
    def test_finds_single_qr(self):
        from pdf_qr_extractor import _detect_qrs_in_page
        img = make_qr_pil('HELLO_QR')
        results = _detect_qrs_in_page(img)
        assert len(results) == 1
        payload_bytes, png_bytes = results[0]
        assert b'HELLO_QR' in payload_bytes
        # PNG 헤더 확인
        assert png_bytes[:8] == b'\x89PNG\r\n\x1a\n'

    def test_no_qr_returns_empty(self):
        from pdf_qr_extractor import _detect_qrs_in_page
        blank = PILImage.new('RGB', (200, 200), color='white')
        results = _detect_qrs_in_page(blank)
        assert results == []
```

- [ ] **Step 2: 실패 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestDetectQrsInPage -v
```

Expected: `NotImplementedError`

- [ ] **Step 3: _detect_qrs_in_page 구현 (`pdf_qr_extractor.py` 교체)**

```python
def _detect_qrs_in_page(img: PILImage.Image) -> list:
    """PIL Image에서 QR 디코드. [(payload_bytes, png_bytes), ...] 반환."""
    from pyzbar.pyzbar import decode, ZBarSymbol
    decoded_list = decode(img, symbols=[ZBarSymbol.QRCODE])
    results = []
    for d in decoded_list:
        left = d.rect.left
        top = d.rect.top
        right = left + d.rect.width
        bottom = top + d.rect.height
        padding = 5
        cropped = img.crop((
            max(0, left - padding),
            max(0, top - padding),
            min(img.width, right + padding),
            min(img.height, bottom + padding),
        ))
        buf = io.BytesIO()
        cropped.save(buf, format='PNG')
        results.append((d.data, buf.getvalue()))
    return results
```

- [ ] **Step 4: 통과 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestDetectQrsInPage -v
```

Expected: `2 passed`

- [ ] **Step 5: 커밋**

```bash
git add pdf_qr_extractor.py tests/test_pdf_qr_extractor.py
git commit -m "feat(pdf-qr): implement _detect_qrs_in_page with pyzbar (TDD)"
```

---

## Task 6: _diff_payloads + _infer_volume_indices (TDD)

**Files:**
- Modify: `pdf_qr_extractor.py`
- Modify: `tests/test_pdf_qr_extractor.py`

- [ ] **Step 1: 실패 테스트 추가**

```python
class TestDiffPayloads:
    def test_extracts_numeric_variant_in_middle(self):
        from pdf_qr_extractor import _diff_payloads
        payloads = ['DOC-001|foo', 'DOC-002|foo', 'DOC-003|foo']
        prefix, suffix, variants = _diff_payloads(payloads)
        assert prefix == 'DOC-00'
        assert suffix == '|foo'
        assert variants == ['1', '2', '3']

    def test_single_payload_returns_empty_variants(self):
        from pdf_qr_extractor import _diff_payloads
        prefix, suffix, variants = _diff_payloads(['ONLY_ONE'])
        assert prefix == 'ONLY_ONE'
        assert suffix == 'ONLY_ONE'
        assert variants == ['']

    def test_no_common_suffix(self):
        from pdf_qr_extractor import _diff_payloads
        payloads = ['A001XYZ', 'A002ABC']
        prefix, suffix, variants = _diff_payloads(payloads)
        assert prefix == 'A00'
        assert suffix == ''
        assert variants == ['1XYZ', '2ABC']


class TestInferVolumeIndices:
    def test_valid_sequence(self):
        from pdf_qr_extractor import _infer_volume_indices
        assert _infer_volume_indices(['1', '2', '3']) == [1, 2, 3]

    def test_zero_padded_valid(self):
        from pdf_qr_extractor import _infer_volume_indices
        assert _infer_volume_indices(['01', '02', '10']) == [1, 2, 10]

    def test_gap_raises(self):
        from pdf_qr_extractor import _infer_volume_indices, ExtractionError
        with pytest.raises(ExtractionError) as exc:
            _infer_volume_indices(['1', '3'])
        assert exc.value.code == 'volume_gap'

    def test_duplicate_raises(self):
        from pdf_qr_extractor import _infer_volume_indices, ExtractionError
        with pytest.raises(ExtractionError) as exc:
            _infer_volume_indices(['1', '1', '2'])
        assert exc.value.code == 'volume_duplicate'

    def test_non_numeric_raises(self):
        from pdf_qr_extractor import _infer_volume_indices, ExtractionError
        with pytest.raises(ExtractionError) as exc:
            _infer_volume_indices(['1', 'abc'])
        assert exc.value.code == 'mixed_document'
```

- [ ] **Step 2: 실패 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestDiffPayloads tests/test_pdf_qr_extractor.py::TestInferVolumeIndices -v
```

Expected: `NotImplementedError`

- [ ] **Step 3: 구현 (`pdf_qr_extractor.py`)**

```python
def _longest_common_prefix(strings: list) -> str:
    if not strings:
        return ''
    s1 = min(strings)
    s2 = max(strings)
    for i, c in enumerate(s1):
        if c != s2[i]:
            return s1[:i]
    return s1


def _longest_common_suffix(strings: list) -> str:
    reversed_strs = [s[::-1] for s in strings]
    return _longest_common_prefix(reversed_strs)[::-1]


def _diff_payloads(payloads: list) -> tuple:
    """(prefix, suffix, variants) 반환."""
    if not payloads:
        return '', '', []
    prefix = _longest_common_prefix(payloads)
    suffix = _longest_common_suffix(payloads)
    suf_len = len(suffix)
    variants = [
        p[len(prefix): len(p) - suf_len if suf_len else len(p)]
        for p in payloads
    ]
    return prefix, suffix, variants


def _infer_volume_indices(variants: list) -> list:
    """variants → int 권 번호 리스트. {1..N} 아니면 ExtractionError."""
    try:
        indices = [int(v) for v in variants]
    except ValueError:
        raise ExtractionError('mixed_document', f'권 번호를 숫자로 해석할 수 없습니다: {variants}')

    if len(set(indices)) < len(indices):
        raise ExtractionError('volume_duplicate', f'중복된 권 번호가 있습니다: {indices}')

    n = len(indices)
    if sorted(indices) != list(range(1, n + 1)):
        raise ExtractionError('volume_gap', f'권 번호가 1..{n} 연속이 아닙니다: {sorted(indices)}')

    return indices
```

- [ ] **Step 4: 통과 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestDiffPayloads tests/test_pdf_qr_extractor.py::TestInferVolumeIndices -v
```

Expected: `8 passed`

- [ ] **Step 5: 커밋**

```bash
git add pdf_qr_extractor.py tests/test_pdf_qr_extractor.py
git commit -m "feat(pdf-qr): implement _diff_payloads and _infer_volume_indices (TDD)"
```

---

## Task 7: extract_qrs_from_pdf 통합 (TDD)

**Files:**
- Modify: `pdf_qr_extractor.py`
- Modify: `tests/test_pdf_qr_extractor.py`

- [ ] **Step 1: 실패 테스트 추가**

```python
class TestExtractQrsFromPdf:
    def test_single_volume(self):
        from pdf_qr_extractor import extract_qrs_from_pdf
        pdf = make_qr_pdf(['SINGLE_DOC'])
        result = extract_qrs_from_pdf(pdf)
        assert result['total_n'] == 1
        assert result['items'][0]['vol_i'] == 1
        assert result['items'][0]['payload'] == 'SINGLE_DOC'
        assert result['items'][0]['png_b64']  # non-empty

    def test_multiple_volumes_ordered(self):
        from pdf_qr_extractor import extract_qrs_from_pdf
        payloads = ['DOC-001|test', 'DOC-002|test', 'DOC-003|test']
        pdf = make_qr_pdf(payloads)
        result = extract_qrs_from_pdf(pdf)
        assert result['total_n'] == 3
        vol_is = [item['vol_i'] for item in result['items']]
        assert sorted(vol_is) == [1, 2, 3]
        assert result['prefix'] == 'DOC-00'

    def test_mixed_document_raises(self):
        from pdf_qr_extractor import extract_qrs_from_pdf, ExtractionError
        payloads = ['DOC-001|foo', 'OTHER-999|bar']
        pdf = make_qr_pdf(payloads)
        with pytest.raises(ExtractionError) as exc:
            extract_qrs_from_pdf(pdf)
        assert exc.value.code == 'mixed_document'

    def test_empty_pdf_raises(self):
        from pdf_qr_extractor import extract_qrs_from_pdf, ExtractionError
        import fitz
        doc = fitz.open()
        with pytest.raises(ExtractionError) as exc:
            extract_qrs_from_pdf(doc.tobytes())
        assert exc.value.code == 'empty_pdf'

    def test_no_qr_in_pdf_raises(self):
        from pdf_qr_extractor import extract_qrs_from_pdf, ExtractionError
        pdf = make_minimal_pdf(1)
        with pytest.raises(ExtractionError) as exc:
            extract_qrs_from_pdf(pdf)
        assert exc.value.code == 'decode_failed'

    def test_real_test_pdf(self):
        """test.pdf(12권)로 통합 검증."""
        from pdf_qr_extractor import extract_qrs_from_pdf
        test_pdf = Path(__file__).parent.parent / 'test.pdf'
        if not test_pdf.exists():
            pytest.skip('test.pdf not found')
        with open(test_pdf, 'rb') as f:
            pdf_bytes = f.read()
        result = extract_qrs_from_pdf(pdf_bytes)
        assert result['total_n'] == 12
        vol_is = sorted(item['vol_i'] for item in result['items'])
        assert vol_is == list(range(1, 13))
        assert result['prefix']
        for item in result['items']:
            assert item['png_b64']
            assert item['payload']
```

- [ ] **Step 2: 실패 확인**

```bash
pytest tests/test_pdf_qr_extractor.py::TestExtractQrsFromPdf -v
```

Expected: `NotImplementedError` (extract_qrs_from_pdf 미구현)

- [ ] **Step 3: extract_qrs_from_pdf 구현**

```python
def extract_qrs_from_pdf(pdf_bytes: bytes) -> dict:
    """PDF bytes → {prefix, total_n, items:[{vol_i, payload, png_b64}]}."""
    pages = _render_pages(pdf_bytes)
    if not pages:
        raise ExtractionError('empty_pdf', 'PDF에 페이지가 없습니다.')

    all_qrs = []
    for page_img in pages:
        qrs = _detect_qrs_in_page(page_img)
        all_qrs.extend(qrs)

    if not all_qrs:
        raise ExtractionError('decode_failed', 'PDF에서 QR 코드를 찾을 수 없습니다.')

    payloads = []
    for payload_bytes, _ in all_qrs:
        for enc in ('cp949', 'utf-8'):
            try:
                payloads.append(payload_bytes.decode(enc))
                break
            except (UnicodeDecodeError, LookupError):
                continue
        else:
            payloads.append(payload_bytes.hex())

    if len(payloads) == 1:
        _, png_bytes = all_qrs[0]
        return {
            'prefix': payloads[0],
            'total_n': 1,
            'items': [{
                'vol_i': 1,
                'payload': payloads[0],
                'png_b64': base64.b64encode(png_bytes).decode(),
            }],
        }

    prefix, suffix, variants = _diff_payloads(payloads)
    vol_indices = _infer_volume_indices(variants)

    items = []
    for vol_i, (_, png_bytes), payload in zip(vol_indices, all_qrs, payloads):
        items.append({
            'vol_i': vol_i,
            'payload': payload,
            'png_b64': base64.b64encode(png_bytes).decode(),
        })

    return {
        'prefix': prefix,
        'total_n': len(items),
        'items': sorted(items, key=lambda x: x['vol_i']),
    }
```

- [ ] **Step 4: 통과 확인 (test.pdf 포함)**

```bash
pytest tests/test_pdf_qr_extractor.py::TestExtractQrsFromPdf -v
```

Expected: `6 passed` (test_real_test_pdf는 test.pdf 구조에 따라 결과 상이할 수 있음 — 실패 시 Step 5 참조)

> **test_real_test_pdf 실패 시:** 먼저 실제 payload 확인:
> ```bash
> python -c "
> from pdf_qr_extractor import _render_pages, _detect_qrs_in_page
> with open('test.pdf','rb') as f: pdf=f.read()
> pages = _render_pages(pdf, dpi=150)
> qrs = _detect_qrs_in_page(pages[0])
> for d,_ in qrs: print(repr(d))
> "
> ```
> 출력된 payload 형식에 맞게 `test_real_test_pdf` assertion 조정.

- [ ] **Step 5: 전체 테스트 회귀 확인**

```bash
pytest tests/ -v
```

Expected: 기존 테스트 포함 모두 통과 (`test_qr_paste.py`, `test_excel_paste_mode.py`, `test_utils_qr.py`).

- [ ] **Step 6: 커밋**

```bash
git add pdf_qr_extractor.py tests/test_pdf_qr_extractor.py
git commit -m "feat(pdf-qr): implement extract_qrs_from_pdf (TDD)"
```

---

## Task 8: Flask 라우트 + 라우트 테스트 (TDD)

**Files:**
- Modify: `app.py`
- Create: `tests/test_extract_route.py`

- [ ] **Step 1: 실패 테스트 작성 (`tests/test_extract_route.py`)**

```python
# tests/test_extract_route.py
import io
import pytest
import fitz
import qrcode
from PIL import Image as PILImage


def make_qr_pil(payload: str) -> PILImage.Image:
    qr = qrcode.QRCode(
        box_size=10, border=2,
        error_correction=qrcode.constants.ERROR_CORRECT_L,
    )
    qr.add_data(payload)
    qr.make(fit=True)
    return qr.make_image(fill_color='black', back_color='white').convert('RGB')


def make_qr_pdf(payloads: list) -> bytes:
    doc = fitz.open()
    for payload in payloads:
        qr_img = make_qr_pil(payload)
        buf = io.BytesIO()
        qr_img.save(buf, format='PNG')
        buf.seek(0)
        page = doc.new_page(width=595, height=842)
        rect = fitz.Rect(100, 100, 350, 350)
        page.insert_image(rect, stream=buf.read())
    return doc.tobytes()


class TestExtractQrFromPdfRoute:
    def test_valid_pdf_returns_200(self, client):
        pdf = make_qr_pdf(['DOC-001|test', 'DOC-002|test'])
        data = {'pdf': (io.BytesIO(pdf), 'test.pdf', 'application/pdf')}
        resp = client.post('/extract_qr_from_pdf',
                           data=data, content_type='multipart/form-data')
        assert resp.status_code == 200
        body = resp.get_json()
        assert body['total_n'] == 2
        assert len(body['items']) == 2
        assert all(item['png_b64'] for item in body['items'])
        assert all(item['vol_i'] in [1, 2] for item in body['items'])

    def test_missing_pdf_returns_400(self, client):
        resp = client.post('/extract_qr_from_pdf',
                           data={}, content_type='multipart/form-data')
        assert resp.status_code == 400
        assert resp.get_json()['error'] == 'missing_pdf'

    def test_invalid_pdf_bytes_returns_400(self, client):
        data = {'pdf': (io.BytesIO(b'not a pdf at all'), 'x.pdf', 'application/pdf')}
        resp = client.post('/extract_qr_from_pdf',
                           data=data, content_type='multipart/form-data')
        assert resp.status_code == 400
        assert resp.get_json()['error'] == 'invalid_pdf'

    def test_pdf_too_large_returns_413(self, client, monkeypatch):
        import app as app_module
        monkeypatch.setattr(app_module.config, 'MAX_PDF_FILE_SIZE', 10)
        pdf = make_qr_pdf(['DOC-001|x'])
        data = {'pdf': (io.BytesIO(pdf), 'big.pdf', 'application/pdf')}
        resp = client.post('/extract_qr_from_pdf',
                           data=data, content_type='multipart/form-data')
        assert resp.status_code == 413
        assert resp.get_json()['error'] == 'pdf_too_large'

    def test_mixed_document_returns_400(self, client):
        pdf = make_qr_pdf(['DOC-001|foo', 'OTHER-999|bar'])
        data = {'pdf': (io.BytesIO(pdf), 'mixed.pdf', 'application/pdf')}
        resp = client.post('/extract_qr_from_pdf',
                           data=data, content_type='multipart/form-data')
        assert resp.status_code == 400
        body = resp.get_json()
        assert body['error'] == 'mixed_document'
```

- [ ] **Step 2: 실패 확인**

```bash
pytest tests/test_extract_route.py -v
```

Expected: `404` 또는 `ConnectionError` (라우트 미존재)

- [ ] **Step 3: app.py에 라우트 추가**

`app.py` 상단 import 블록에 추가:

```python
from utils import (
    get_client_info, log_client_access, create_directory_if_not_exists,
    delete_file_later, validate_and_clean_input, safe_int_conversion,
    validate_qr_image_bytes, delete_dir_later, validate_pdf_bytes
)
import pdf_qr_extractor
```

그리고 `config` 상수 블록(L36-42 근방)에 추가:

```python
MAX_PDF_FILE_SIZE = config.MAX_PDF_FILE_SIZE
```

`api_create_label` 라우트(L248) 바로 앞에 신규 라우트 삽입:

```python
@app.route('/extract_qr_from_pdf', methods=['POST'])
@handle_errors
def extract_qr_from_pdf():
    """PDF 파일에서 QR 코드를 추출하고 권 순서대로 반환한다."""
    client_ip, _ = get_client_info()
    logger.info(f"PDF QR extraction request from {client_ip}")

    pdf_file = request.files.get('pdf')
    if pdf_file is None:
        return jsonify({'error': 'missing_pdf', 'message': 'PDF 파일이 전송되지 않았습니다.'}), 400

    blob = pdf_file.read()

    if len(blob) > MAX_PDF_FILE_SIZE:
        return jsonify({'error': 'pdf_too_large', 'message': f'PDF 파일이 너무 큽니다 (최대 {MAX_PDF_FILE_SIZE // 1024 // 1024} MB).'}), 413

    if not validate_pdf_bytes(blob):
        return jsonify({'error': 'invalid_pdf', 'message': '유효한 PDF 파일이 아닙니다.'}), 400

    try:
        result = pdf_qr_extractor.extract_qrs_from_pdf(blob)
    except pdf_qr_extractor.ExtractionError as e:
        logger.warning(f"PDF QR extraction failed for {client_ip}: {e.code} — {e.msg}")
        return jsonify({'error': e.code, 'message': e.msg}), 400

    logger.info(f"PDF QR extraction success for {client_ip}: {result['total_n']} QRs, prefix={result['prefix']!r}")
    return jsonify(result), 200
```

- [ ] **Step 4: 통과 확인**

```bash
pytest tests/test_extract_route.py -v
```

Expected: `5 passed`

- [ ] **Step 5: 전체 테스트**

```bash
pytest tests/ -v
```

Expected: 전체 통과.

- [ ] **Step 6: 커밋**

```bash
git add app.py tests/test_extract_route.py
git commit -m "feat(api): add /extract_qr_from_pdf route (TDD)"
```

---

## Task 9: 클라이언트 — qr_paste.js

**Files:**
- Modify: `static/js/qr_paste.js`

`qr_paste.js` 전체를 아래 내용으로 교체:

- [ ] **Step 1: `static/js/qr_paste.js` 교체**

```javascript
// qr_paste.js — QR 이미지 수집 (PDF 추출/data URI) + SortableJS 순서 관리 + FormData fetch 제출

(function () {
  'use strict';

  /** @type {{ id: number, blob: Blob, hash: string, url: string }[]} */
  const state = { images: [], nextId: 0 };

  const dropzone = document.getElementById('qr_dropzone');
  const thumbnailList = document.getElementById('qr_thumbnails');
  const counterEl = document.getElementById('qr_counter');
  const orderInput = document.getElementById('qr_order');
  const form = document.getElementById('label-form');

  // ── 중복 체크용 핑거프린트 ──────────────────────────────────────────────
  function fingerprint(arrayBuffer) {
    const bytes = new Uint8Array(arrayBuffer);
    const len = bytes.length;
    let h = len;
    const sample = Math.min(256, len);
    for (let i = 0; i < sample; i++) h = (Math.imul(h, 31) + bytes[i]) >>> 0;
    for (let i = Math.max(0, len - sample); i < len; i++) h = (Math.imul(h, 31) + bytes[i]) >>> 0;
    return `${len}_${h.toString(16)}`;
  }

  // ── 카운터 갱신 ───────────────────────────────────────────────────────────
  function getDocCount() {
    const docType = document.getElementById('doc_type').value;
    const key = docType === '1' ? 'eq_doc_count' : 'pjt_doc_count';
    return parseInt(document.getElementById(key)?.value || '1', 10);
  }

  function updateCounter() {
    const n = getDocCount();
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
        <div class="qr-thumb-image">
          <img src="${url}" alt="QR ${id}" />
          <button type="button" class="qr-remove-btn" data-id="${id}">×</button>
        </div>
        <span class="qr-thumb-label"></span>
      `;
      thumbnailList.appendChild(li);
    });
    syncOrder();
    updateCounter();
  }

  // ── qr_order hidden input 동기화 ─────────────────────────────────────────
  function syncOrder() {
    const lis = [...thumbnailList.querySelectorAll('li[data-id]')];
    const domIds = lis.map(el => parseInt(el.dataset.id, 10));
    const order = domIds.map(id => state.images.findIndex(img => img.id === id));
    orderInput.value = JSON.stringify(order);
    lis.forEach((li, i) => {
      const label = li.querySelector('.qr-thumb-label');
      if (label) label.textContent = `${i + 1}권`;
    });
  }

  // ── SortableJS 초기화 ────────────────────────────────────────────────────
  if (window.Sortable) {
    Sortable.create(thumbnailList, { animation: 150, onEnd: syncOrder });
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

  // ── Blob → state.images 추가 헬퍼 ──────────────────────────────────────
  async function addFromBlob(blob) {
    const arrayBuffer = await blob.arrayBuffer();
    const hash = fingerprint(arrayBuffer);
    if (state.images.some(img => img.hash === hash)) {
      showMessage('중복된 QR 이미지입니다.', 'error');
      return;
    }
    const url = URL.createObjectURL(blob);
    state.images.push({ id: state.nextId++, blob, hash, url });
    renderThumbnails();
  }

  // ── PDF 업로드 → 서버 추출 ───────────────────────────────────────────────
  const ERROR_MESSAGES = {
    empty_pdf: 'PDF에서 QR을 찾을 수 없습니다.',
    decode_failed: '일부 페이지의 QR을 읽을 수 없습니다. 선명한 PDF인지 확인하세요.',
    mixed_document: '다른 문서의 QR이 섞여 있습니다. 동일 문서 PDF만 업로드하세요.',
    volume_gap: '권 번호가 불연속합니다 (예: 1, 2, 4 — 3 누락).',
    volume_duplicate: '중복된 권 번호가 있습니다.',
    invalid_pdf: '유효한 PDF 파일이 아닙니다.',
    pdf_too_large: 'PDF 파일이 너무 큽니다 (최대 50 MB).',
    missing_pdf: 'PDF 파일이 전송되지 않았습니다.',
  };

  async function addFromPdf(file) {
    if (!file || file.type !== 'application/pdf') {
      showMessage('PDF 파일만 업로드할 수 있습니다.', 'error');
      return;
    }
    const formData = new FormData();
    formData.append('pdf', file);
    try {
      const resp = await fetch('/extract_qr_from_pdf', { method: 'POST', body: formData });
      const body = await resp.json().catch(() => ({}));
      if (!resp.ok) {
        showMessage(ERROR_MESSAGES[body.error] || body.message || 'PDF 처리 중 오류가 발생했습니다.', 'error');
        return;
      }
      // state.images 교체
      state.images.forEach(img => URL.revokeObjectURL(img.url));
      state.images = [];
      state.nextId = 0;
      const sorted = [...body.items].sort((a, b) => a.vol_i - b.vol_i);
      for (const item of sorted) {
        const binary = atob(item.png_b64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
        const blob = new Blob([bytes], { type: 'image/png' });
        await addFromBlob(blob);
      }
      showMessage(`PDF에서 QR ${body.total_n}개를 추출했습니다.`, 'success');
    } catch (err) {
      showMessage('요청 중 오류가 발생했습니다: ' + err.message, 'error');
    }
  }

  // ── dropzone drag 이벤트 ─────────────────────────────────────────────────
  dropzone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropzone.classList.add('dragover');
  });
  dropzone.addEventListener('dragleave', (e) => {
    if (!dropzone.contains(e.relatedTarget)) dropzone.classList.remove('dragover');
  });
  dropzone.addEventListener('drop', async (e) => {
    e.preventDefault();
    dropzone.classList.remove('dragover');
    const file = e.dataTransfer.files[0];
    await addFromPdf(file);
  });

  // ── dropzone 클릭 → 파일 선택 ────────────────────────────────────────────
  const fileInput = document.getElementById('qr_file_input');
  dropzone.addEventListener('click', (e) => {
    if (e.target.closest('input, button')) return;
    fileInput.click();
  });
  fileInput.addEventListener('change', async (e) => {
    const file = e.target.files[0];
    if (file) await addFromPdf(file);
    e.target.value = '';
  });

  // ── data URI 입력 처리 ──────────────────────────────────────────────────
  function dataUriToBlob(dataUri) {
    const parts = dataUri.trim().split(',');
    if (parts.length < 2) return null;
    const header = parts[0];
    const b64 = parts[1];
    const mimeMatch = header.match(/data:([^;]+)/);
    if (!mimeMatch) return null;
    try {
      const binary = atob(b64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      return new Blob([bytes], { type: mimeMatch[1] });
    } catch {
      return null;
    }
  }

  async function addFromDataUri(raw) {
    if (!raw.startsWith('data:image/')) {
      showMessage('data:image/... 형식의 URI만 지원합니다.', 'error');
      return;
    }
    const blob = dataUriToBlob(raw);
    if (!blob) { showMessage('유효하지 않은 data URI입니다.', 'error'); return; }
    await addFromBlob(blob);
  }

  const dataUriInput = document.getElementById('qr_data_uri_input');
  const dataUriBtn = document.getElementById('qr_data_uri_btn');

  async function handleDataUriSubmit() {
    const val = dataUriInput.value.trim();
    if (!val) return;
    await addFromDataUri(val);
    dataUriInput.value = '';
  }

  dataUriBtn.addEventListener('click', handleDataUriSubmit);
  dataUriInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { e.preventDefault(); handleDataUriSubmit(); }
  });

  // doc_type 변경 시 카운터 갱신
  document.querySelectorAll('[data-value][onclick*="selectDocType"]').forEach(btn => {
    btn.addEventListener('click', () => setTimeout(updateCounter, 0));
  });
  document.querySelectorAll('#eq_doc_count, #pjt_doc_count').forEach(input => {
    input.addEventListener('input', updateCounter);
  });

  // ── 폼 submit 가로채기 ────────────────────────────────────────────────────
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (typeof validateForm === 'function' && !validateForm()) return;
    const n = getDocCount();
    if (state.images.length !== n) {
      showErrorMessage(`QR 이미지 수(${state.images.length})가 권수(${n})와 다릅니다.`);
      return;
    }
    syncOrder();
    const formData = new FormData(form);
    state.images.forEach((img, i) => {
      formData.append('qr_images', new File([img.blob], `qr_${i}.png`, { type: 'image/png' }));
    });
    showLoading();
    try {
      const resp = await fetch('/create_label', { method: 'POST', body: formData });
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({ error: '서버 오류가 발생했습니다.' }));
        showErrorMessage(err.error || '서버 오류가 발생했습니다.');
        return;
      }
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
      showSuccessMessage();
    } catch (err) {
      showErrorMessage('요청 중 오류가 발생했습니다: ' + err.message);
    } finally {
      hideLoading();
    }
  });

  updateCounter();
})();
```

- [ ] **Step 2: 커밋**

```bash
git add static/js/qr_paste.js
git commit -m "feat(client): replace image dropzone with PDF upload, add addFromPdf"
```

---

## Task 10: templates/index.html UI 업데이트

**Files:**
- Modify: `templates/index.html:327-337` (dropzone 섹션)

- [ ] **Step 1: dropzone 카피·accept 변경**

`index.html` L319-337 (`<div class="form-section" id="qr_section">` 내부 qr-input-group 블록) 교체:

```html
                <div class="form-section" id="qr_section">
                    <div class="section-title">
                        <i data-lucide="qr-code"></i> QR 이미지
                    </div>
                    <p class="section-description">
                        라벨 프린트 PDF를 업로드하면 QR을 자동으로 추출합니다. 추가한 이미지는 드래그로 순서를 바꿀 수 있으며, 캡션의 권 번호가 인쇄 순서가 됩니다.
                    </p>

                    <div class="qr-input-group">
                        <label class="qr-input-label">
                            <i data-lucide="file-up"></i> PDF 업로드
                        </label>
                        <div class="qr-dropzone" id="qr_dropzone" tabindex="0">
                            <i data-lucide="file-up"></i>
                            <p>여기를 클릭하거나 PDF 파일을 끌어다 놓으세요</p>
                            <p class="qr-hint">라벨 프린트 PDF 파일 (1개)</p>
                            <input type="file" id="qr_file_input" accept="application/pdf" hidden>
                        </div>
                    </div>

                    <div class="qr-divider"><span>또는</span></div>

                    <div class="qr-input-group">
                        <label class="qr-input-label" for="qr_data_uri_input">
                            <i data-lucide="link"></i> 이미지 링크 붙여넣기
                        </label>
                        <div class="qr-data-uri-row">
                            <input type="text" id="qr_data_uri_input" class="qr-data-uri-input"
                                   placeholder="QR 이미지 우클릭 → '이미지 주소 복사' → 붙여넣고 Enter" />
                            <button type="button" id="qr_data_uri_btn" class="qr-data-uri-btn">추가</button>
                        </div>
                    </div>

                    <div class="qr-counter" id="qr_counter">0 / 1</div>
                    <ul class="qr-thumbnails sortable" id="qr_thumbnails"></ul>
                    <input type="hidden" id="qr_order" name="qr_order" value="[]">
                </div>
```

- [ ] **Step 2: 커밋**

```bash
git add templates/index.html
git commit -m "feat(ui): switch dropzone to PDF-only with updated copy"
```

---

## Task 11: README.md, CONTEXT.md 업데이트

**Files:**
- Modify: `README.md`
- Modify: `CONTEXT.md`

- [ ] **Step 1: CONTEXT.md QR 이미지 입력 섹션 업데이트**

`CONTEXT.md`의 `**QR 이미지 입력 (QR image input)**` 항목(L19-24) 교체:

```markdown
**QR 이미지 입력 (QR image input)**:
라벨 앱이 QR PNG를 받는 두 가지 입력 경로.
- **PDF 업로드**: 라벨 프린트 PDF 1개를 dropzone에 드래그&드롭 또는 클릭. 서버가 PyMuPDF + pyzbar로 QR을 추출하고 권 순서대로 자동 배치. 동일 문서 검증(diff 휴리스틱, 권 번호 시퀀스 검사) 포함. POST `/extract_qr_from_pdf`.
- **data URI 텍스트 입력**: `data:image/...;base64,...` 텍스트를 입력란에 붙여넣고 Enter 또는 '추가' 버튼 → Blob 복원 후 동일 검사 적용.
두 경로 모두 `state.images = { id, blob, hash, url }[]`에 수렴한다.
_Avoid_: 이미지 클립보드 paste(`clipboardData.items`의 `image/*`), URL fetch, image/* 파일 직접 선택
_이유_: 외부 사내 시스템 보안 정책으로 클립보드 image MIME 차단됨; PDF 업로드로 N회 개별 업로드 문제 해결
```

- [ ] **Step 2: README.md 버전 업데이트**

README 상단 버전 표기 및 기능 목록에서 파일 업로드 설명을 PDF 업로드로 교체. (README 내용에 따라 해당 줄 수정)

- [ ] **Step 3: 커밋**

```bash
git add CONTEXT.md README.md
git commit -m "docs: update CONTEXT.md and README for PDF QR extraction (v2.2.0)"
```

---

## Task 12: E2E 수동 검증

- [ ] **Step 1: 서버 기동**

```bash
cd /home/spuhaha18/Project/QR_Web
python app.py
```

또는:

```bash
python run_waitress.py
```

- [ ] **Step 2: 브라우저에서 test.pdf 드롭**

1. `http://localhost:5000` 접속
2. QR 섹션 dropzone에 `test.pdf` 드롭
3. 썸네일 12개 `<1권>..<12권>` 라벨로 자동 배치 확인

- [ ] **Step 3: SortableJS 재정렬 확인**

드래그로 순서 변경 → 권 번호 라벨이 새 순서 반영하는지 확인

- [ ] **Step 4: 폼 제출**

문서 정보 입력 후 "라벨 만들기" → 다운로드된 xlsx 열어 시트 B5 셀의 "i/N" 값이 권 순서와 일치하는지 확인

- [ ] **Step 5: 에러 케이스 확인**

- 비-PDF 파일 드롭 → "PDF 파일만 업로드할 수 있습니다." 토스트
- data URI 입력 섹션 여전히 동작하는지 확인

- [ ] **Step 6: 전체 pytest 최종 확인**

```bash
pytest tests/ -v
```

Expected: 전체 통과.

---

## Self-Review

**Spec coverage:**
- ✅ 추출 위치: 서버 PyMuPDF + pyzbar (Task 4-7)
- ✅ UI dropzone PDF 전용 (Task 9-10)
- ✅ data URI 유지 (Task 9)
- ✅ 추출 실패 전체 거부 (Task 7-8)
- ✅ state.images replace (Task 9)
- ✅ diff 휴리스틱 동일 문서 검증 (Task 6)
- ✅ doc_count 불일치 기존 검증 재사용 (무수정)
- ✅ 서버 응답 png_b64 + payload + vol_i + prefix (Task 7)
- ✅ 의존성 (Task 1)
- ✅ config 상수 (Task 2)
- ✅ 에러 토스트 한글 매핑 (Task 9)
- ✅ README, CONTEXT.md (Task 11)

**Type/method consistency:**
- `_render_pages` → `list[PIL.Image]` — Task 4 정의, Task 7 사용 ✅
- `_detect_qrs_in_page` → `list[tuple[bytes, bytes]]` — Task 5 정의, Task 7 사용 ✅
- `_diff_payloads` → `tuple[str, str, list[str]]` — Task 6 정의, Task 7 사용 ✅
- `_infer_volume_indices` → `list[int]` — Task 6 정의, Task 7 사용 ✅
- `ExtractionError.code`, `.msg` — Task 4 정의, Task 8 catch ✅
- `addFromBlob(blob)` — Task 9 유지 및 재사용 ✅
- `addFromPdf(file)` — Task 9 정의, dropzone/fileInput 핸들러 사용 ✅
