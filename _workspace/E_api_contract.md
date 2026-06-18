# E — API 계약 (go-backend-engineer → frontend-engineer)

Go(Fiber) 백엔드 확정 API 계약. **프론트 작업의 선행 조건.** 모든 에러는 `{ "error": "<한국어 메시지>" }` + 4xx/5xx.
한국어 에러 문자열은 app.py 원문 그대로 보존(테스트 패리티). 임의 변경 금지.

---

## POST /create_label  (paste 모드, multipart/form-data)

라벨 .xlsx를 **바이너리로 직접 스트리밍**(임시파일/`/download` 없음).

### 요청 (FormData 키)
공통:
- `doc_type` — `"1"`(기기) | `"2"`(과제)
- `binder_size` — `"1"|"3"|"5"|"7"` (과제는 `"1"` 불가)
- `qr_order` — JSON 문자열. doc_count 길이의 `[0..n)` 순열. 예: `"[1,0,2]"`. 화면 dnd 순서.
- `qr_images` — 같은 키로 N개 파일 첨부. 각 PNG, ≤2MB. N == doc_count.

기기(doc_type=1) 필수 텍스트 필드:
- `eq_number`, `eq_doc_number`, `eq_doc_title`, `eq_doc_count`(권수), `eq_doc_department`, `eq_doc_year`

과제(doc_type=2) 필수 텍스트 필드:
- `pjt_number`, `pjt_test_number`, `pjt_doc_title`, `pjt_doc_writer`, `pjt_doc_count`(권수)

### 응답
- 성공 200:
  - `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - `Content-Disposition: attachment; filename="{doc_number}_{YYYYMMDDhhmmss}.xlsx"`
  - body = .xlsx 바이트 (ZIP, `PK` 시그니처)
- 에러 400: `{ "error": "..." }`

### 검증 순서 + 에러 메시지 (app.py 원문)
1. 폼 파싱 불가 → 400 `잘못된 요청 형식입니다.`
2. doc_type 무효 → 400 `잘못된 문서 종류입니다.`
3. binder_size 무효 → 400 `잘못된 바인더 크기입니다.`
4. 과제 + binder=1 → 400 `과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.`
5. 필수 필드 누락 → 400 `필수 필드가 누락되었습니다: {field}`
6. qr_order 파싱 불가 → 400 `qr_order 형식이 올바르지 않습니다.`
7. 파일 수 ≠ 권수 → 400 `QR 이미지 수가 권수와 다릅니다 (받음: {N}, 권수: {C})`
8. 파일 수 > 50 → 400 `QR 이미지는 최대 50개까지 허용됩니다.`
9. qr_order 길이 ≠ 권수 → 400 `qr_order 길이가 권수와 다릅니다.`
10. qr_order 중복/범위초과 → 400 `qr_order에 중복이나 범위 초과 인덱스가 있습니다.`
11. 파일 >2MB → 400 `QR 이미지 크기가 2MB를 초과합니다: {filename}`
12. 비PNG → 400 `유효하지 않은 PNG 이미지입니다: {filename}`

---

## POST /api/create_label  (auto 모드, application/json)

서버가 시트별 QR을 자동 생성(CP949 인코딩 페이로드)하여 임베드. 기존 `success` JSON 계약 유지하되
`download_url`(임시파일) 제거 → 워크북 바이트를 base64 인라인 반환.

### 요청 (JSON 필드)
`/create_label`의 텍스트 필드와 동일 (`doc_type`, `binder_size`, `eq_*` 또는 `pjt_*`). qr_images/qr_order 없음.
숫자는 number/string 둘 다 허용(서버에서 문자열화).

### 응답
- 성공 200:
  ```json
  {
    "success": true,
    "message": "라벨이 성공적으로 생성되었습니다.",
    "filename": "DOC-001_20260618134843.xlsx",
    "file_base64": "<base64 .xlsx>",
    "content_type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
  }
  ```
  프론트: `file_base64` → Blob → 다운로드.
- 에러 400: `{ "error": "..." }`
  - JSON 파싱 불가/빈 객체 → `잘못된 JSON 데이터입니다.`
  - 검증 실패 → `/create_label`과 동일 한국어 메시지(2~5번)

---

## GET /api/qr_image/:text

QR PNG 직반환. `text`는 URL 경로(퍼센트 인코딩, 서버에서 디코드).

- ≤500자(룬). 응답 200 `Content-Type: image/png`, body=PNG.
- 빈 text → 400 `QR 코드 텍스트가 제공되지 않았습니다.`
- 500자 초과 → 400 `QR 코드 텍스트가 너무 깁니다 (최대 500자).`

## POST /api/qr_image_base64

- 요청 JSON: `{ "text": "..." }`
- 성공 200: `{ "success": true, "image_base64": "<base64 png>", "mime_type": "image/png" }`
- 빈 text → 400 `QR 코드 텍스트가 제공되지 않았습니다.`
- 500자 초과 → 400 `QR 코드 텍스트가 너무 깁니다 (최대 500자).`

## GET /api/health

`{ "status": "healthy", "timestamp": "<RFC3339>", "version": "1.0.0" }` (200)

---

## 로그 뷰어 (SPA 이전 예정, 현재 API만)

- `GET /api/logs?lines=&level=&search=` — `{ success, logs:[...], total_lines, requested_lines, level_filter, search_filter }`.
  `lines` 기본 100, 최대 1000. `level` 기본 `all`(ALL/DEBUG/INFO/WARNING/ERROR). `search` 부분일치(대소문자 무시).
  파일 없으면 `{ success:true, logs:[], message:"로그 파일이 아직 생성되지 않았습니다." }`.
- `POST /api/logs/clear` — `{ success:true, message:"로그 파일이 초기화되었습니다.", backup_file:"..." }`. 파일 없으면 `message:"초기화할 로그 파일이 없습니다."`.
- `GET /api/logs/download` — `text/plain` 첨부(`app_logs_{ts}.log`). 없으면 404 `다운로드할 로그 파일이 없습니다.`

---

## 미구현 (Phase 결정)
- `GET /` → 현재 204 플레이스홀더. Phase 6에서 임베드 SPA.
- `GET /logs` → 204 플레이스홀더(SPA).
- `/api/performance`, `/api/system/*` → 드롭(미구현).
- `/api/docs`, `/manual` → SPA 이전(미구현).
- `/download/:filename` → 스트리밍 채택으로 제거.

## 미들웨어
- BodyLimit 16MB(`MAX_CONTENT_LENGTH`). 초과 시 Fiber 413.
- recover: panic → 500 `{ "error": "서버 오류가 발생했습니다." }`.
- 모든 요청 파일+stdout 로깅.
