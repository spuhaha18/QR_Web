---
name: go-backend-build
description: Go(Fiber) 백엔드를 구축할 때 사용. 프로젝트 구조(cmd/internal/web), Flask 라우트→Fiber 핸들러 매핑, .xlsx 스트리밍 응답(임시파일 제거), go-qrcode QR 생성+CP949 인코딩, env 기반 config, PNG 검증, 라벨 스키마 파싱/검증을 다룬다. Go API, Fiber 라우트, 핸들러, QR 생성, config 작업 시 반드시 사용.
---

# Go 백엔드 구축 (Fiber)

현재 `app.py`(12 라우트)+`document_schema.py`+`qr_generator.py`+`config.py`+`utils.py`를 Go로 포팅. 단일 정적 바이너리의 서버 측.

## 프로젝트 구조
```
cmd/qrweb/main.go            # config 로드, 라우터, embed SPA, listen
internal/
  config/config.go           # env 기반 Config, Load()
  label/schema.go            # Label 인터페이스, Equipment/ProjectLabel, ParseLabelRequest, 검증
  label/layout.go            # GetQRConfig (go-excelize-port 스킬 참조)
  excel/                     # excel-parity-engineer 담당 (go-excelize-port 스킬)
  qr/qr.go                   # go-qrcode + CP949, PNG/base64
  imaging/png.go             # PNG 검증 (Pillow 대체)
  httpx/                     # 핸들러 (app.py 대응)
  logging/logging.go
web/embed.go                 # //go:embed dist → fs.FS
web/dist/                    # Vite 빌드 출력 (frontend 담당)
```

## 핵심 결정: 스트리밍 (임시파일 제거)
**.xlsx를 인메모리 생성 후 응답에 직접 스트리밍.** `file_lifecycle.py`/`uploads/`/`/download`/cleanup goroutine 전부 제거 — 큰 단순화 + 버그 클래스 제거.
```go
buf, _ := generator.WriteToBuffer()  // excelize
c.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
return c.Send(buf.Bytes())
```
외부 API용 `download_url` 계약 유지가 꼭 필요할 때만 인메모리 TTL 캐시(`map[string][]byte`+`time.AfterFunc`).

## 라우트 매핑 (상세는 references/routes-map.md)
유지 필수: `GET /`(SPA), 정적 embed, `POST /create_label`(paste, multipart→.xlsx), `POST /api/create_label`(auto, JSON→.xlsx bytes), `GET /api/qr_image/:text`, `POST /api/qr_image_base64`, `GET /api/health`.
드롭 권장(확인): `/api/performance`, `/api/system/*`. 로그뷰어(`/logs`,`/api/logs*`)·매뉴얼은 유지/SPA이전 결정.

## 보존 필수 (테스트 패리티)
- **한국어 에러 문자열·상태코드 원문 보존.** paste 검증: `len==doc_count`, `≤50(MAX_QR_FILES)`, `qr_order` 길이·범위·중복 없음, 각 PNG ≤2MB·유효 PNG. 각 위반에 고유 한국어 메시지 + 400.
- `safe_int_conversion`: `isdigit()`만 변환, 아니면 default, `max(1,result)`. 음수/소수 → default.
- `validate_and_clean_input`: strip 후 `\n`/`\r` 제거.
- 파일명: `{doc_number}_{YYYYMMDDhhmmss}.xlsx` (기기=eq_doc_number, 과제=pjt_test_number).

## QR 생성 (references/qr.md)
go-qrcode(`qrcode.Low`=ERROR_CORRECT_L), **자동 모드 페이로드는 CP949 인코딩**(`korean.EUCKR`, parity-qa가 골든 비교). 페이로드 파이프 구분:
- 기기: `eq_number|eq_doc_number|eq_doc_title|eq_doc_department|eq_doc_year|{i}/{N}`
- 과제: `pjt_number|pjt_test_number|pjt_doc_title|pjt_doc_writer|{i}/{N}`

## config (references/config.md)
stdlib `os.Getenv` + 타입 헬퍼. env: `HOST,PORT,MAX_QR_FILES,MAX_QR_FILE_SIZE,MAX_CONTENT_LENGTH,LOG_LEVEL,QR_BOX_SIZE,QR_BORDER`. 드롭: `SECRET_KEY,DELETE_DELAY,UPLOAD_FOLDER,QR_CACHE_TTL`. 미들웨어: logger, recover(→JSON 500), BodyLimit 16MB.

## Label 인터페이스 (excel-parity-engineer와 합의)
```go
type Label interface {
    CellValues() map[string]any
    QRPayload(i, total int) string
    DocNumber() string
    DocCount() int
    TitleCell() string
}
```
`ParseLabelRequest(form) (Label, docType string, binder int, err error)`. 과제는 binder==1 거부("과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.").

## 참조 파일
- `references/routes-map.md` — Flask 12라우트 → Fiber 핸들러 + 요청/응답 shape
- `references/qr.md` — go-qrcode + CP949 인코딩 상세
- `references/config.md` — Config 구조체 + env 매핑
