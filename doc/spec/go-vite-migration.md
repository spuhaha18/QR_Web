# 설계문서: QR_Web Flask → Go(Fiber) + Vite/Svelte SPA 마이그레이션

> 작성일 2026-06-18 · 상태: 설계 확정(구현 전) · 실행 하네스: `qr-web-dev` 오케스트레이터

## 1. Context — 왜 바꾸는가

QR_Web은 현재 **Python 3.13 + Flask** 웹앱이다. 연구소 기기/과제 문서용 표준 바인더 라벨을 QR 코드 임베드 Excel(.xlsx)로 생성한다(~6,100 LOC, 67 pytest 통과).

사용자가 **전면 재작성**을 결정했고, 동기는 3가지:
1. **프론트엔드 현대화** — 서버렌더 HTML+바닐라 JS → 컴포넌트 프레임워크 + 빌드툴
2. **성능/확장성** — 더 빠른 응답, 동시 처리
3. **배포/설치 단순화** — Docker/런타임 의존 제거

→ 세 목표를 동시에 만족하는 해는 **Go 단일 정적 바이너리**(런타임 불필요=배포 최강, 성능 최강) + **Vite/Svelte SPA를 `embed.FS`로 바이너리 내장**(프론트 현대화). 데스크톱 앱(Tauri/Electron)은 "확장성=멀티유저 서버"와 충돌해 배제.

### 확정 스택
| 영역 | 선택 | 비고 |
|------|------|------|
| 백엔드 | Go + Fiber | go-qrcode, excelize 순수 Go |
| Excel | excelize | 이미지 임베드/멀티시트/스타일 지원 |
| QR | skip2/go-qrcode | CP949 인코딩 |
| 프론트 | Vite + **Svelte** | svelte-dnd-action 재정렬 |
| 배포 | `embed.FS` 단일 바이너리 | Docker 불필요(선택적 `FROM scratch`) |

---

## 2. 보존해야 할 현재 동작 (패리티 스펙 = 오라클)

재작성이 깨면 안 되는 load-bearing 동작. **Excel 시각 패리티가 최대 위험.**

### 2.1 라벨 의미 (document_schema.py)
- `doc_type` 문자열: `'1'`=기기, `'2'`=과제. `binder_size` int ∈ {1,3,5,7}. **과제는 binder==1 거부**("과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.").
- 기기 필수 필드: `eq_number, eq_doc_number, eq_doc_title, eq_doc_count, eq_doc_department, eq_doc_year`(count→int, year→int 기본 현재년).
- 과제 필수 필드: `pjt_number, pjt_test_number, pjt_doc_title, pjt_doc_writer, pjt_doc_count`.
- `safe_int_conversion`: `str(value).isdigit()`만 변환, 아니면 default, `max(1,result)` 클램프(음수/소수→default).
- `validate_and_clean_input`: strip 후 `\n`/`\r` 제거.

### 2.2 셀값 (Sheet 1)
- **기기**: B2=eq_number, B3=eq_doc_number, B4=eq_doc_title, B5="1/{count}", B6=eq_doc_department, B7=eq_doc_year.
- **과제**: B2=pjt_number, B3=pjt_test_number, B4=pjt_doc_title, B5="1/{count}", B6=pjt_doc_writer + 우측 패널 Q21="[{pjt_number}] {pjt_test_number}", Q22=pjt_doc_title, R23=pjt_doc_writer, S23="1/{count}".

### 2.3 QR 페이로드 (자동 모드, 파이프 구분, **CP949 인코딩**)
- 기기: `eq_number|eq_doc_number|eq_doc_title|eq_doc_department|eq_doc_year|{i}/{N}`
- 과제: `pjt_number|pjt_test_number|pjt_doc_title|pjt_doc_writer|{i}/{N}`

### 2.4 Excel 레이아웃 (excel_generator.py — 최고 충실도 영역)
- **행높이**: `1:2.25, 2:27, 3:27, 4:216, 5:40.5, 6:27, 7:27`, 8–17:6.75, 18:2.25. 과제 추가 `20:2.25, 21:48, 22:34.5, 23:27.75, 24:2.25`.
- **열너비**: A=N=0.375, B–M=바인더별 가변. 과제 추가 Q=8.13, R=34.88, S=8.13, T=0.375, N/O/P=0.375.
- **병합**: `B2:M2~B6:M6`; 기기 +`B7:M7`; 과제 +`Q21:S21, Q22:S22`.
- **테두리**: `B2:M6` thin; 외곽 medium(`A1:A18` left, `N1:N18` right, `A1:N1` top, `A18:N18` bottom) + 모서리 2-side medium. 기기/과제 각자 추가 thin 영역(상세 §2.6).
- **폰트**: FONT_TIMES=Times New Roman 12 bold black, FONT_TITLE=TNR 16 bold(B4). 과제 Q21=TNR20 bold center/wrap, Q22/R23=TNR13 bold center/wrap, S23=FONT_TIMES.
- **정렬**: 데이터 입력 후 used range 전 셀 center/vcenter/wrap.
- **멀티시트 i/N**: Sheet 1 완성 후 `copy_worksheet`로 2..N 복제. 각 복제는 B5="{i}/{count}"(과제는 S23도, print_area='A1:T24'). 복제 후 전 셀 정렬 재적용 + B4=FONT_TITLE.
- **QR 이미지**: openpyxl `Image`, **75×75px 강제**, `config['cell_pos']` 좌상단 one-cell 앵커.

### 2.5 바인더 → QR 셀/너비 (label_layout.py)
| binder | column_width | 기기 셀 | 과제 셀 |
|--------|--------------|--------|--------|
| 7 | 1.875 | E9 | E8 |
| 5 | 1.25 | D9 | D8 |
| 3 | 1.0 | D9 | D8 |
| 1 | 0.75 | B9 | B9 |

미지 크기 → binder 3 폴백.

### 2.6 과제 추가 테두리 (`_apply_project_borders`)
`B7:M7` top, `B7:B17` left, `M7:M17` right, `B17:M17` bottom, B17/M17 2-side. `Q20:S20`/`Q24:S24` top+bottom, `P21:P23`/`T21:T23` left+right, 모서리 P20/T20/P24/T24 2-side, `Q22:S22` thin 전체.

### 2.7 paste 모드 vs 자동 모드
- **`/create_label`**(웹폼, paste): 클라이언트가 N개 PNG + `qr_order` JSON 업로드 → 서버가 qr_order로 재정렬, PNG 검증(Pillow verify + format=='PNG', ≤2MB), **그 이미지 그대로 임베드**(서버 QR 생성 안 함).
- **`/api/create_label`**(JSON, auto): 서버가 페이로드로 QR 생성.

### 2.8 검증 규칙 (`/create_label`)
`len(qr_files)==doc_count`; `≤50(MAX_QR_FILES)`; `len(qr_order)==doc_count`; `sorted(qr_order)==range(doc_count)`(중복/범위초과 없음); 각 ≤2MB·유효 PNG. **각 위반에 고유 한국어 에러 + 400.**

### 2.9 파일명/응답
파일명 `{base_filename}_{YYYYMMDDhhmmss}.xlsx`(기기=eq_doc_number, 과제=pjt_test_number). 응답 `send_file(as_attachment, download_name)` + 쿠키 `download_complete=true max_age=10`.

---

## 3. Go 프로젝트 구조

```
qrweb/
├── go.mod · Makefile
├── cmd/qrweb/main.go            # config 로드, 라우터, embed SPA, listen
├── internal/
│   ├── config/config.go         # env 기반 Config, Load()
│   ├── label/
│   │   ├── schema.go            # Label 인터페이스, Equipment/ProjectLabel, ParseLabelRequest, 검증
│   │   └── layout.go            # GetQRConfig (바인더 테이블)
│   ├── excel/
│   │   ├── generator.go         # CreateLabelExcel: 레이아웃/스타일/시트/QR 임베드
│   │   └── styles.go            # 합성 스타일 ID 빌더
│   ├── qr/qr.go                 # go-qrcode + CP949, PNG/base64
│   ├── imaging/png.go           # PNG 검증 (Pillow 대체)
│   ├── httpx/                   # 핸들러 (app.py 대응)
│   │   ├── server.go · label_handler.go · qr_handler.go · health_handler.go · logs_handler.go
│   └── logging/logging.go
└── web/
    ├── embed.go                 # //go:embed all:dist → fs.FS
    ├── dist/                    # Vite 빌드 출력 (gitignore)
    └── frontend/                # Vite + Svelte 소스
        ├── vite.config.ts · package.json · index.html
        └── src/
            ├── App.svelte · main.ts
            └── lib/{api.ts, qrStore.ts, components/*.svelte}
```

---

## 4. 모듈 매핑

| Python | Go | 핵심 |
|--------|-----|------|
| `config.py` | `internal/config/config.go` | `Config` struct, `Load()`. env + 타입 헬퍼. Dev/Prod/Test 계층 → env로 통합 |
| `utils.py` | `internal/label`+`internal/imaging/png.go`+`internal/logging` | `ValidateAndCleanInput`, `SafeIntConversion`, `ValidatePNGBytes`, `GenerateTimestampFilename` |
| `document_schema.py` | `internal/label/schema.go` | `Label` 인터페이스, `EquipmentLabel`/`ProjectLabel`, `ParseLabelRequest`, `ErrValidation` |
| `label_layout.py` | `internal/label/layout.go` | `QRConfig`, `GetQRConfig`, 바인더 테이블 |
| `qr_generator.py` | `internal/qr/qr.go` | `CreateQRPNG`/`CreateQRBase64`. CP949(`korean.EUCKR`), `qrcode.Low` |
| `excel_generator.py` | `internal/excel/generator.go`+`styles.go` | `CreateLabelExcel(...) ([]byte, filename, error)` — **바이트 반환**(§7) |
| `file_lifecycle.py` | **제거**(스트리밍 채택) | — |
| `app.py` | `internal/httpx/*`+`cmd/qrweb/main.go` | Fiber 라우트/미들웨어 |
| `performance_monitor.py`, `cache_manager.py` | **드롭 권장** | 운영 부가물 |

### Label 인터페이스
```go
type Label interface {
    CellValues() map[string]any
    QRPayload(i, total int) string
    DocNumber() string
    DocCount() int
    TitleCell() string
}
```

### excelize 핵심 (openpyxl 차이)
- **시트명 "Sheet 1"(공백)** — 기본 "Sheet1" rename. 테스트가 단언.
- **스타일 모델**: openpyxl=셀별 독립 속성 누적, excelize=셀당 단일 스타일 ID. → 셀별 최종 합성 스타일 맵 만들어 `SetCellStyle` 일괄 flush.
- **멀티시트**: `copy_worksheet` 없음 → `CopySheet(from,to)`. QR은 복제 후 시트별 임베드.
- **이미지 75px**: excelize는 스케일 팩터 → PNG 디코드로 원본 px 구해 `ScaleX=75/srcW`, one-cell 앵커.
- **print_area**: `SetDefinedName(_xlnm.Print_Area)`.
- **저장**: `WriteToBuffer()` → 응답 스트림(임시파일 없음).

---

## 5. Excel 패리티 위험 (최우선)

1. **스타일 모델 불일치** — 셀별 합성 스타일로 해결. 영역 경계 테두리(예: `B7:M7` top-thin과 외곽 medium 공존) 검증.
2. **이미지 사이징/앵커** — 절대 75px·one-cell 앵커 재현. PNG 디코드 후 스케일. **육안 확인 필수**(가장 "어긋나 보이기" 쉬움).
3. **CopySheet 충실도** — 병합/치수/스타일 복제 확인. 이미지 누락 가능 → QR은 복제 후 임베드. 누락 시 폴백(시트별 전체 레이아웃 재실행).
4. **CP949 vs EUC-KR** — Go `korean.EUCKR`(EUC-KR/Win-949) vs Python CP949(MS-949 슈퍼셋). 현대 한글 일치, 희귀 음절 차이 가능 → 골든 바이트 비교. **자동 모드만 영향**(paste는 무관).
5. **QR 모듈 크기** — `fit=False`+`version=None` 실제 출력 캡처. 75px 리사이즈라 시각 고정, 데이터 밀도만 영향.
6. **숫자 포맷** — `eq_doc_year` int 셀(텍스트 아님) 확인.

**전략: 골든 파일 하네스.** 현재 Python 앱으로 (doc_type × binder × 단일/멀티 × paste/auto) 매트릭스 .xlsx 생성 → Go 출력과 **의미 단위** 비교(셀값/병합/치수/시트명/이미지 앵커/셀별 테두리·폰트). XML 완전일치 비대상.

---

## 6. API 표면 (Fiber)

### 유지 필수
- `GET /` → embed `index.html`(SPA). 정적 → `web/dist` embed.
- `POST /create_label` → paste. multipart → `.xlsx`(attachment) 또는 `{error}` 4xx. **한국어 에러/상태코드 원문 보존.**
- `POST /api/create_label` → auto. JSON in → **`.xlsx` bytes 직접 반환**(현 `download_url`에서 변경, §7).
- `GET /api/qr_image/:text`(≤500자) → PNG. `POST /api/qr_image_base64` → `{success, image_base64, mime_type}`.
- `GET /api/health` → `{status, timestamp, version}`.

### 로그뷰어 (유지 — 결정 2b)
- `GET /logs`(SPA 페이지/모달), `GET /api/logs`(`bufio` tail + level/search 필터), `POST /api/logs/clear`, `GET /api/logs/download` 포팅.

### 제거 (확정)
- `GET /download/:filename` — 스트리밍 채택(결정 1)으로 제거.
- `/api/docs`, `/manual` — SPA 이전(결정 3)으로 엔드포인트 제거.
- `/api/performance`, `/api/system/*` — 드롭(결정 2a). performance_monitor/cache_manager 미포팅.

미들웨어: `logger`(LOG_FORMAT), `recover`(panic→`{error:"서버 오류가 발생했습니다."}` 500), `BodyLimit(16MB)`.

---

## 7. 파일 수명주기 — 스트리밍 권장 (임시파일 제거)

현재는 .xlsx를 `uploads/`에 쓰고 `DELETE_DELAY` 후 백그라운드 스레드 삭제 + paste QR PNG 임시 디렉토리도 삭제 예약. 이 서브시스템 전체가 Flask `send_file`이 경로를 원하던 역사적 이유.

**권장: 인메모리 생성(`WriteToBuffer`) 후 응답 스트리밍.** paste QR 바이트는 메모리(`[][]byte`)로 `AddPictureFromBytes`. → `file_lifecycle.py`, `uploads/`, `/download`, cleanup goroutine, 3개 lifecycle 엔드포인트 제거. 큰 단순화 + 버그 클래스 제거 + **배포 단순화 목표 부합**.

영향: `/api/create_label`이 `download_url` 대신 바이트 직접 반환. 외부 API `download_url` 계약 필수 시에만 인메모리 TTL 캐시(`map[string][]byte`+`time.AfterFunc`). **기본안: 스트리밍, `/download` 드롭.**

---

## 8. Config (Go)

stdlib `os.Getenv` + 타입 헬퍼(viper 불필요).
```go
type Config struct {
    Host string; Port int
    LogLevel, LogFile string
    MaxContentLength, MaxQRFileSize int64
    MaxQRFiles, QRBoxSize, QRBorder int
    Version string
}
```
env 보존: `HOST, PORT, MAX_QR_FILES, MAX_QR_FILE_SIZE, MAX_CONTENT_LENGTH, LOG_LEVEL, QR_BOX_SIZE, QR_BORDER`. 드롭: `SECRET_KEY, DELETE_DELAY, UPLOAD_FOLDER, QR_CACHE_TTL`, 성능 플래그. `.env` 유지 원하면 `joho/godotenv`.

---

## 9. 테스트 전략 (현 67 테스트 대응)

| Python 테스트 | 수 | Go 대상 | 비고 |
|--------------|----|---------|------|
| test_document_schema | ~12 | `label/schema_test.go` | 파싱/검증/과제 1cm 거부/int 기본. 1:1 |
| test_label_schema | ~15 | `label/schema_test.go` | cell_values, B5 카운트, qr_payload 5필드, make_label. 1:1 |
| test_label_layout | ~8 | `label/layout_test.go` | GetQRConfig 바인더/타입, 3 폴백. 테이블 테스트 |
| test_utils_qr | 5 | `imaging/png_test.go` | PNG true/JPEG·garbage·empty·truncated false |
| test_excel_paste_mode | 3 | `excel/generator_test.go` | paste .xlsx, 시트수==count, auto |
| test_excel_project_label | ~12 | `excel/generator_test.go` | 과제 2시트, S23=="2/2", 7cm, too-few-paths 에러 |
| test_file_lifecycle | ~6 | **드롭**(스트리밍) | — |
| test_qr_paste | 7 | `httpx/handler_test.go` | `app.Test`로 E2E: N→.xlsx, 개수/order/PNG 400 |

**+ 골든 패리티 테스트(신규)**: 매트릭스 구조 비교. ~55/67 직접 포팅, ~12는 Go 에러경로/드롭, +~10 골든.

---

## 10. 빌드 & 배포 (단일 정적 바이너리)

```
cd web/frontend && npm ci && npm run build   # → web/dist
go build -trimpath -ldflags="-s -w -X main.version=$(cat VERSION)" -o bin/qrweb ./cmd/qrweb
```
- `web/embed.go`: `//go:embed all:dist` → `fs.Sub`.
- `CGO_ENABLED=0` — go-qrcode/excelize 순수 Go → 완전 정적. 크로스컴파일 `GOOS/GOARCH`.
- **Makefile**: `frontend, build, run, test, dev(go run+vite 프록시 동시), clean`.
- 배포: `bin/qrweb` 복사+실행. waitress+Docker 대체. Dockerfile/compose 삭제 가능(선택적 `FROM scratch` 유지).

---

## 11. 마이그레이션 시퀀싱 (단계별 검증)

Excel 코어(최대 위험) 먼저, 패리티 증명 후 프론트.

- **A. 스캐폴딩 + 골든 캡처** — `go mod init`, config/logging. 현재 Python 앱으로 골든 매트릭스 → `testdata/golden/`(오라클). (~0.5d)
- **B. 라벨 도메인**(HTTP/Excel 없음) — schema/layout/png 포팅 + 테스트. 데이터 계층 검증. (~1d)
- **C. QR 생성** — qr.go + CP949. QR 바이트 골든 비교(위험 #4/#5 해결). (~0.5–1d)
- **D. Excel 생성(크럭스)** — generator/styles. 하위 단계(레이아웃→테두리→폰트→멀티시트→QR)마다 골든 비교. **패리티 통과 전 진행 금지**(위험 #1/#2/#3). 실제 Excel/LibreOffice 육안 확인. (**2–4d, 일정 위험**)
- **E. HTTP 계층** — Fiber/라우트/스트리밍. handler 테스트(한국어 에러/상태코드). 기존 templates로 스모크 가능. (~1d)
- **F. 프론트(Vite+Svelte)** — CSS 포팅, 컴포넌트, dnd/드롭존/data-URI/제출. Vite 프록시로 Go 백엔드 연동. (~2–3d)
- **G. Embed + 빌드 + 컷오버** — embed.FS/Makefile/단일 바이너리. 전체 QA. Python/Docker 아카이브. README 갱신. (~1d)

**총 ~8–11 작업일**, Phase D(Excel 패리티)가 일정 위험. 부가 엔드포인트/auto download_url 드롭 시 ~1–1.5d 절감.

---

## 12. 확정 결정 (2026-06-18)

| # | 결정 | 확정 | 결과 |
|---|------|------|------|
| 1 | xlsx 수명주기 | **스트리밍** | 인메모리 생성→응답 직접 전송. `file_lifecycle.py`·`uploads/`·`/download`·cleanup goroutine 제거. `/api/create_label`은 bytes 직반환 |
| 2a | perf/system 엔드포인트 | **드롭** | `/api/performance`, `/api/system/*` 제거. `performance_monitor`/`cache_manager` 미포팅 |
| 2b | 로그뷰어 | **유지** | `GET /logs`, `/api/logs`, `POST /api/logs/clear`, `GET /api/logs/download` 포팅(`bufio` tail + level/search 필터) + SPA 로그 페이지 |
| 3 | 매뉴얼/docs | **SPA 이전** | `/manual`, `/api/docs` 엔드포인트 제거 → Svelte 컴포넌트/모달로 |
| 4 | QR 인코딩 | **EUC-KR 허용** | Go `korean.EUCKR` 그대로. 골든 바이트 비교로 검증(자동 모드만 영향) |
| 5 | dnd 순서 계약 | **순열 배열 전송** | 파일은 삽입 순서 + `qr_order` 순열 배열. 현 Flask 재정렬 로직 유지(핸들러 변경 최소, 테스트 패리티) |

이 결정들은 §6 API 표면·§7 수명주기·§9 테스트에 반영됨.

---

## 13. 실행 방법

이 설계는 `.claude/skills/qr-web-dev` 오케스트레이터(에이전트 팀: excel-parity-engineer, go-backend-engineer, svelte-frontend-engineer, parity-qa)가 실행한다. "마이그레이션 시작해줘" → Phase 0 컨텍스트 확인 → 팀 구성 + 골든 캡처 → §11 시퀀싱 진행.

관련 문서:
- 하네스 구성 계획: `doc/plan/recursive-weaving-tide.md`
- 패리티 매핑 상세: `.claude/skills/go-excelize-port/references/`
