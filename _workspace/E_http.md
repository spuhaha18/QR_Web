# E — Phase 4: HTTP 계층 (go-backend-engineer 완료 보고)

## 구현 파일
- `internal/config/config.go` — `Config` + `Load()`. stdlib os.LookupEnv + 타입헬퍼(getEnv/getEnvInt/getEnvInt64). `Addr()`. 드롭: SECRET_KEY/DEBUG/DELETE_DELAY/UPLOAD_FOLDER/QR_CACHE_TTL/perf 플래그. HOST/FLASK_HOST, PORT/FLASK_PORT 둘 다 수용.
- `internal/logging/logging.go` — 파일+stdout 로거. Python format(`%(asctime)s %(levelname)s %(name)s %(threadName)s : %(message)s`) 근사 재현(`MainThread` 고정). `Init(logFile, level)`, `Default()`, Debug/Info/Warn/Error, Close. LOG_LEVEL 필터.
- `internal/httpx/server.go` — Fiber 앱(BodyLimit=MaxContentLength), recover→500 한국어 JSON, requestLogger, 라우트 등록(API/명시 라우트 먼저, `/`·`/logs` 플레이스홀더 마지막).
- `internal/httpx/label_handler.go` — paste(`/create_label`, multipart) + auto(`/api/create_label`, JSON).
- `internal/httpx/qr_handler.go` — `/api/qr_image/:text`, `/api/qr_image_base64`.
- `internal/httpx/health_handler.go` — `/api/health`.
- `internal/httpx/logs_handler.go` — 로그뷰어 4엔드포인트(bufio tail + level/search 필터).
- `internal/httpx/handler_test.go` — test_qr_paste.py 포팅 + 추가 케이스.
- `cmd/qrweb/main.go` — config 로드, 로거 init, 서버 구동.

## 라우트 목록
| 메서드 | 경로 | 핸들러 | 비고 |
|---|---|---|---|
| POST | /create_label | handleCreateLabelPaste | multipart → .xlsx 스트리밍 |
| POST | /api/create_label | handleCreateLabelAuto | JSON → success+file_base64 |
| GET | /api/qr_image/:text | handleQRImage | PNG, ≤500룬 |
| POST | /api/qr_image_base64 | handleQRImageBase64 | {success,image_base64,mime_type} |
| GET | /api/health | handleHealth | {status,timestamp,version} |
| GET | /api/logs | handleGetLogs | tail + level/search |
| POST | /api/logs/clear | handleClearLogs | 백업 후 truncate |
| GET | /api/logs/download | handleDownloadLogs | text/plain 첨부 |
| GET | / | (placeholder) | 204 (Phase 6 SPA) |
| GET | /logs | (placeholder) | 204 (Phase 6 SPA) |

## 핵심 결정
- **임시파일 제거**: paste는 `c.Send(data)` + Content-Disposition으로 .xlsx 직접 스트리밍. file_lifecycle/uploads/`/download` 없음.
- **auto 모드 계약**: app.py가 `{success,...,download_url}` 반환 + test_qr_paste.py가 `success is True` 단언. download_url(임시파일) 제거 불가피 → `file_base64`(인라인 base64 .xlsx)로 대체하여 success 계약·테스트 패리티 둘 다 유지. (routes-map.md "스트리밍 시 bytes 직접 또는 base64" 중 base64 채택 — 기존 JSON shape 보존이 우선.)
- **paste 검증 순서**(routes-map.md/app.py 그대로): 폼파싱→doc_type/binder/필수→qr_order파싱→len==count→≤50→order길이→order순열(중복/범위)→각 ≤2MB·유효PNG→재정렬→generator→스트리밍.
- **qr_order 재정렬**: `ordered[sheetIdx] = fileBytes[qr_order[sheetIdx]]` (Flask `[file_bytes_list[i] for i in qr_order]` 동일).
- **auto QR**: `qr.CreateQRPNG(lbl.QRPayload(i+1, total))` (1-based 시트 인덱스, B_label.md 주의점 5).

## 보존한 한국어 에러 문자열 (app.py 원문, 변형 없음)
- `잘못된 요청 형식입니다.` (multipart 파싱 실패, 신규)
- `잘못된 문서 종류입니다.` / `잘못된 바인더 크기입니다.` (label.ValidationMessage)
- `과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.`
- `필수 필드가 누락되었습니다: {field}`
- `qr_order 형식이 올바르지 않습니다.`
- `QR 이미지 수가 권수와 다릅니다 (받음: {N}, 권수: {C})`
- `QR 이미지는 최대 {MAX_QR_FILES}개까지 허용됩니다.`
- `qr_order 길이가 권수와 다릅니다.`
- `qr_order에 중복이나 범위 초과 인덱스가 있습니다.`
- `QR 이미지 크기가 2MB를 초과합니다: {filename}`
- `유효하지 않은 PNG 이미지입니다: {filename}`
- `잘못된 JSON 데이터입니다.`
- `QR 코드 텍스트가 제공되지 않았습니다.` / `QR 코드 텍스트가 너무 깁니다 (최대 500자).`
- `서버 오류가 발생했습니다.` (recover 500 + 내부오류)
- 로그뷰어: `로그 파일이 아직 생성되지 않았습니다.`, `로그 파일이 초기화되었습니다.`, `초기화할 로그 파일이 없습니다.`, `다운로드할 로그 파일이 없습니다.`

## 검증 결과
- `go build ./...` → OK (전체)
- `go vet ./internal/httpx/ ./internal/config/ ./internal/logging/` → clean
- `gofmt -l` → clean
- `go test -count=1 ./internal/httpx/` → ok. 13 테스트 전부 통과(정확N→.xlsx 200, 개수불일치 400, order 길이/범위/중복 400, 비PNG 400, auto 200 success, 과제 binder1 400, health 200, qr base64 200/길이초과 400, qr_image PNG 200).
- `go test ./...` → 전 패키지 green(excel/imaging/label/qr/httpx).

## 스모크 (PORT=5099 go run ./cmd/qrweb)
- `GET /api/health` → 200 `{"status":"healthy","timestamp":"...","version":"1.0.0"}`
- `GET /` → 204 (플레이스홀더)
- `POST /api/qr_image_base64` → 200 (유효 PNG base64)
- 로그파일 라인 포맷 확인: `2026-06-18 13:48:43,542 INFO app MainThread : ...` (Python format 일치)

## 의존성 변경
- go.mod에 `github.com/gofiber/fiber/v2 v2.52.13` 추가(+ 간접 fasthttp/bytebufferpool 등). `go mod tidy` 적용.

## 막힌 점 / 주의
- 없음(빌드/벳/테스트/스모크 전부 green). 단 auto 모드 응답 shape(`file_base64`)는 app.py `download_url`에서 의도적 변경 — frontend-engineer는 `E_api_contract.md`의 base64→Blob 다운로드 방식 사용 필요. parity-qa는 auto 응답 키 차이(임시파일 제거 결정 §12) 인지 요망.
