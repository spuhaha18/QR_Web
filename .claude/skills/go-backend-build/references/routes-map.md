# 라우트 매핑: Flask(app.py) → Fiber

## 유지 필수
| Flask | Fiber | 요청 | 응답 |
|-------|-------|------|------|
| `GET /` | `app.Get("/", ...)` | — | embed `index.html` (SPA) |
| 정적 | `app.Use(filesystem ...)` | — | `web/dist` embed 서빙 |
| `POST /create_label` | `app.Post("/create_label", ...)` | multipart: doc_type, binder_size, 필드들, qr_order(JSON), N×qr_images | `.xlsx`(attachment) 또는 `{error}` 4xx |
| `POST /api/create_label` | `app.Post("/api/create_label", ...)` | JSON: doc_type, binder_size, 필드 | **스트리밍 시**: `.xlsx` bytes 직접 반환(또는 base64). (현행 `{success,filename,download_url}`에서 변경) |
| `GET /api/qr_image/:text` | `app.Get("/api/qr_image/:text", ...)` | path text(≤500자) | PNG 또는 `{error}` |
| `POST /api/qr_image_base64` | `app.Post(...)` | JSON `{text}` | `{success, image_base64, mime_type}` |
| `GET /api/health` | `app.Get("/api/health", ...)` | — | `{status, timestamp, version}` |

## 드롭/이전 결정
- `GET /download/:filename` — 스트리밍 채택 시 제거.
- `GET /logs`, `GET /api/logs`, `POST /api/logs/clear`, `GET /api/logs/download` — 로그뷰어 유지 시만. 유지하면 `bufio`로 파일 tail + level/search 필터.
- `GET /api/docs`, `GET /manual` — SPA로 이전 권장(엔드포인트 제거).
- `GET /api/performance`, `/api/system/optimize`, `/api/system/status` — **드롭 권장**(performance_monitor/cache_manager 의존, 운영 부가물).

## paste 모드 검증 순서 (POST /create_label)
1. doc_type/binder_size/필수필드 파싱 → 실패 시 400 + 한국어 메시지
2. doc_count 추출
3. `len(qr_files) == doc_count` 아니면 400 ("QR 이미지 개수가 문서 권수와 일치하지 않습니다." 류 — 원문 확인)
4. `len(qr_files) <= 50` 초과 400
5. `qr_order` 파싱, `len == doc_count`, `sorted(qr_order) == range(doc_count)`(중복/범위초과 없음) 아니면 400
6. 각 파일 ≤2MB, 유효 PNG(`imaging.ValidatePNGBytes`) 아니면 400
7. qr_order로 바이트 재정렬 → `excel.CreateLabelExcel(docType, binder, label, qrPNGs)` → 스트리밍

> 정확한 한국어 에러 문자열은 `app.py` 원문을 그대로 복사. 테스트(test_qr_paste.py)가 상태코드 단언.

## 미들웨어
- `logger`(LOG_FORMAT 맞춤), `recover`(panic→`{error:"서버 오류가 발생했습니다."}` 500), `BodyLimit(16MB)`(MAX_CONTENT_LENGTH 대응).
- `handle_errors` 데코레이터 → recover 미들웨어로 통합.

## Fiber 테스트 (test_qr_paste.py 대응)
`app.Test(httptest.NewRequest(...))`로 E2E: 정확 N→.xlsx, 개수불일치 400, qr_order 길이/범위/중복 400, 비PNG 400, api auto 생성. 상태코드 + content-type + 한국어 에러 단언.
