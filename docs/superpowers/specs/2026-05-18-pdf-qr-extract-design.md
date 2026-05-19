# PDF → QR 추출 입력 — 설계 문서

**날짜**: 2026-05-18  
**상태**: 승인됨

## Context

현재 QR_Web 라벨 흐름은 사용자가 외부 "새 프로젝트"(라벨 프린트 도구)에서
QR 이미지를 권당 1장씩 PNG로 저장하거나 우클릭 → "이미지 링크 복사"로
data URI를 추출해 dropzone/입력란에 넣어야 한다. 권수가 많으면(예: 20권)
이 단계가 N회 반복되어 사용성이 크게 떨어진다.

새 프로젝트는 라벨을 PDF로 한 번에 출력할 수 있다(예: `test.pdf` — 12권,
4×3 그리드 × 2페이지, 권당 `TIC1505-ST03-NNNN` 일련번호와 QR 1개씩).
**PDF 한 장을 업로드하면 시스템이 QR을 모두 추출해 권 순서대로 폼에 채워주는
입력 경로를 추가**해, N회 반복 다운로드/복붙을 1회 업로드로 대체한다.

## 결정 사항

1. **추출 위치**: 서버 (Flask + PyMuPDF + pyzbar).
2. **UI 통합**: 기존 image 파일/드롭존 입력 제거 → 같은 영역을 PDF 전용 dropzone으로 교체. data URI 입력 섹션은 유지(클립보드 차단 환경 fallback).
3. **추출 실패 정책**: 단 하나라도 디코드 실패 / 다른 문서 섞임 / 권 시퀀스 불연속 시 **전체 거부** + 에러 토스트.
4. **추출 후 흐름**: 기존 `state.images` **완전 교체(replace)**. 사용자는 기존 SortableJS 그리드에서 드래그로 최종 순서 조정 가능.
5. **동일 문서 검증**: payload 문자열의 **자동 diff 휴리스틱** — longest common prefix/suffix 제외한 variant 부가 `{1..N}` 정수 시퀀스를 형성하면 동일 문서 인정. variant = 권 번호(i).
6. **doc_count 불일치**: 기존 서버 검증 재사용 (`/create_label` 제출 시 `len(qr_files) != doc_count` 에러). 추출 단계에서는 토스트 경고만.
7. **서버 응답**: PNG bytes(b64) + payload raw + 권 번호(i) + 추론 prefix.

## 아키텍처 / 데이터 흐름

```
[PDF dropzone]
   ↓ file
POST /extract_qr_from_pdf  (multipart, pdf 1개)
   ↓
서버: pdf_qr_extractor
  ├─ PyMuPDF(fitz): 페이지 N개 → PIL.Image (300 dpi)
  ├─ pyzbar.decode: 각 페이지 QR 다중 디코드 + bbox crop → PNG bytes
  ├─ payload 문자열 diff: longest common prefix·suffix 추출
  ├─ variant 부 = 권번호 시퀀스 (정수 파싱, {1..N} 연속·중복 검사)
  └─ 검증 통과 → JSON 응답
   ↓
JSON { prefix, total_n, items: [{png_b64, payload, vol_i}] }
   ↓
클라이언트(qr_paste.js): state.images 클리어 → vol_i 오름차순 push
   ↓
사용자 SortableJS 재정렬 가능 → 기존 /create_label 제출 흐름 그대로
```

## 컴포넌트 상세

### 신규 모듈 — `pdf_qr_extractor.py`

```
extract_qrs_from_pdf(pdf_bytes: bytes) -> dict
  _render_pages(pdf_bytes, dpi=300) -> list[PIL.Image]
      # fitz.open(stream=pdf_bytes) → page.get_pixmap(dpi=dpi) → PIL.Image
  _detect_qrs_in_page(img) -> list[tuple[bytes, bytes]]
      # pyzbar.decode(img, symbols=[ZBarSymbol.QRCODE])
      # decoded.polygon bbox crop → PNG 인코딩
      # 반환: (payload_bytes, png_bytes)
  _diff_payloads(payloads: list[str]) -> tuple[str, str, list[str]]
      # (prefix, suffix, variants)
  _infer_volume_indices(variants: list[str]) -> list[int]
      # 모두 int 파싱 + zero-pad 허용 + 결과 {1..N} 일치 검사
```

**단권 분기**: `len(payloads) == 1` 시 diff 생략, `vol_i = 1`, `prefix = payload`.

**페이로드 인코딩**: payload bytes 디코드 시 CP949 → UTF-8 순서로 시도. 모두 실패 시 raw hex fallback.

**ExtractionError codes**:

| code | 의미 |
|------|------|
| `empty_pdf` | fitz 파싱 성공했으나 페이지 수 0 |
| `decode_failed` | 전체 추출 QR 0개, 또는 pyzbar 픽셀 디코드 실패 |
| `mixed_document` | variant가 숫자로 해석 안 됨 |
| `volume_gap` | 권 번호 1..N 연속 아님 |
| `volume_duplicate` | 같은 권 번호 중복 |

### 신규 라우트 — `app.py`

```python
@app.route('/extract_qr_from_pdf', methods=['POST'])
def extract_qr_from_pdf():
    pdf = request.files.get('pdf')
    if pdf is None:
        return jsonify({'error': 'missing_pdf'}), 400
    blob = pdf.read()
    if len(blob) > MAX_PDF_FILE_SIZE:
        return jsonify({'error': 'pdf_too_large'}), 413
    if not utils.validate_pdf_bytes(blob):
        return jsonify({'error': 'invalid_pdf'}), 400
    try:
        result = pdf_qr_extractor.extract_qrs_from_pdf(blob)
    except pdf_qr_extractor.ExtractionError as e:
        return jsonify({'error': e.code, 'message': e.msg}), 400
    return jsonify(result), 200
```

응답 예시:

```json
{
  "prefix": "TIC1505-ST03-",
  "total_n": 12,
  "items": [
    {"vol_i": 1,  "payload": "TIC1505-ST03-0001|...|1/12",  "png_b64": "..."},
    {"vol_i": 12, "payload": "TIC1505-ST03-0012|...|12/12", "png_b64": "..."}
  ]
}
```

### 클라이언트 — `static/js/qr_paste.js`, `templates/index.html`

**index.html**: 기존 파일/드롭존 섹션 → PDF 전용 dropzone으로 교체.  
`<input type="file" accept="application/pdf">`, 드롭존 카피: "PDF 파일을 드래그&드롭 또는 클릭하여 업로드". data URI 섹션 변경 없음.

**qr_paste.js**:
- `addFromFiles`의 `image/*` 분기 제거. dropzone drop/change 핸들러 → `addFromPdf(file)` 위임.
- 신규 `addFromPdf(file)`:
  1. FormData에 `pdf` key로 file 첨부 → `fetch('/extract_qr_from_pdf')`
  2. 200: `state.images.length = 0` → items를 `vol_i` asc 정렬 → 각 item의 `png_b64` → `atob` → `Uint8Array` → `Blob('image/png')` → 기존 `addFromBlob` 재사용
  3. 400/413: `showMessage` 토스트 (코드별 한글 메시지)
- `syncOrder`, `<N>권` 라벨, SortableJS, 제출 흐름 무수정.

**에러 토스트 한글 매핑**:

| code | 메시지 |
|------|--------|
| `empty_pdf` | PDF에서 QR을 찾을 수 없습니다 |
| `decode_failed` | 일부 페이지의 QR을 읽을 수 없습니다. 선명한 PDF인지 확인하세요 |
| `mixed_document` | 다른 문서의 QR이 섞여 있습니다. 동일 문서 PDF만 업로드하세요 |
| `volume_gap` | 권 번호가 불연속합니다 (예: 1,2,4 — 3 누락) |
| `volume_duplicate` | 중복된 권 번호가 있습니다 |
| `invalid_pdf` | 유효한 PDF 파일이 아닙니다 |
| `pdf_too_large` | PDF 파일이 너무 큽니다 (최대 50 MB) |
| `missing_pdf` | PDF 파일이 전송되지 않았습니다 |

## 검증 알고리즘 (diff 휴리스틱)

```python
prefix = longest_common_prefix(payloads)
suffix = longest_common_suffix(payloads)
variants = [p[len(prefix) : len(p) - len(suffix) if suffix else len(p)]
            for p in payloads]

try:
    vol_indices = [int(v) for v in variants]
except ValueError:
    raise ExtractionError('mixed_document', ...)

if sorted(vol_indices) != list(range(1, len(vol_indices) + 1)):
    if len(set(vol_indices)) < len(vol_indices):
        raise ExtractionError('volume_duplicate', ...)
    raise ExtractionError('volume_gap', ...)
```

`suffix` 빈 문자열 처리 주의: `p[len(prefix):]` 사용 (슬라이싱 버그 방지).

## 의존성·환경

- `requirements.txt`: `PyMuPDF`, `pyzbar`
- `Dockerfile`: `apt-get install -y libzbar0`
- `config.py`: `MAX_PDF_FILE_SIZE = 50 * 1024 * 1024`, `PDF_RENDER_DPI = 300`

## 변경 파일

| 종류 | 경로 |
|------|------|
| 신규 | `pdf_qr_extractor.py` |
| 신규 | `tests/test_pdf_qr_extractor.py` |
| 신규 | `tests/test_extract_route.py` |
| 수정 | `app.py` |
| 수정 | `utils.py` (`validate_pdf_bytes` 추가) |
| 수정 | `config.py` (`MAX_PDF_FILE_SIZE`, `PDF_RENDER_DPI`) |
| 수정 | `static/js/qr_paste.js` |
| 수정 | `templates/index.html` |
| 수정 | `requirements.txt` |
| 수정 | `Dockerfile` |
| 수정 | `README.md` (v2.2.0) |
| 수정 | `CONTEXT.md` |

## 재사용 기존 함수

- `qr_paste.js:94 addFromBlob` — b64→Blob push 흐름 (해시·URL·썸네일).
- `qr_paste.js:62 syncOrder`, SortableJS L75 — 재정렬·`<N>권` 라벨.
- `utils.py:88 validate_qr_image_bytes` — PNG 검증 (서버가 PNG 반환하므로 그대로).
- `app.py:150 create_label` paste 모드 — 무수정.
- `excel_generator.py:287 _apply_qr_codes` paste 분기 — 무수정.

## E2E 검증 방법

1. `pytest` — `tests/test_pdf_qr_extractor.py`, `tests/test_extract_route.py` 통과.
2. 브라우저: `test.pdf` 드롭 → 썸네일 12개 `<1권>..<12권>` 자동 배치 확인.
3. SortableJS 재정렬 → 제출 → xlsx 시트 i = vol_i 일치 확인.
4. 다른 문서 섞인 PDF → `mixed_document` 토스트 확인.
5. PDF >50MB → 413 토스트 확인.
