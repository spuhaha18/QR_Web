# QR_Web v3.0.0

연구소 바인더 라벨 제작을 위한 웹 애플리케이션입니다. 기기 문서·과제 문서의 라벨을 표준 포맷으로 생성하고, QR 코드를 임베드한 Excel(.xlsx) 파일로 출력합니다.

**v3.0.0부터 Python(Flask) → Go(Fiber) + Svelte SPA로 전면 재작성**되어, 런타임 의존성 없는 **단일 정적 바이너리** 하나로 배포·실행됩니다. (Python·Node·Docker 불필요)

---

## ✨ 주요 기능

- **단일 정적 바이너리**: SPA를 바이너리에 임베드(`embed.FS`). 파일 하나만 복사하면 실행 — 런타임 설치 불필요
- **표준 라벨 생성**: 기기/과제 문서 × 바인더 사이즈(1·3·5·7cm)별 규격 Excel(.xlsx) 생성
- **QR 코드**: 입력 정보 기반 자동 생성(자동 모드) 또는 이미지 업로드(붙여넣기 모드). 라벨 하단 박스 **정중앙** 배치
- **모던 UI/UX**: Vite+Svelte SPA, Pristine Lab 테마, 다크 모드, Toast 알림, Lucide 아이콘
- **인라인 검증 + 준비상태 패널**: 필드별 실시간 검증, QR 개수/권수 카운터(부족·일치·초과), 준비 완료 전 제출 비활성, 폼 초기화
- **드래그앤드롭 순서 재배치**: svelte-dnd-action 기반 QR 순서 조정(인쇄 순서)
- **스트리밍 응답**: Excel을 메모리에서 생성해 직접 전송 — 임시 파일/디렉토리 없음
- **시스템 로그 뷰어**: 인앱 모달에서 로그 조회(레벨/검색 필터·새로고침·다운로드·초기화)

---

## 🏗️ 기술 스택 & 프로젝트 구조

**백엔드** Go + [Fiber](https://gofiber.io) · Excel [excelize](https://github.com/xuri/excelize) · QR [go-qrcode](https://github.com/skip2/go-qrcode) · 인코딩 `golang.org/x/text`(CP949)
**프론트엔드** Vite + Svelte + TypeScript · svelte-dnd-action · lucide-svelte

```
QR_Web/
├── cmd/qrweb/main.go            # 엔트리포인트
├── internal/
│   ├── config/                  # env 기반 설정
│   ├── label/                   # schema.go(라벨/검증), layout.go(바인더→QR 설정)
│   ├── excel/                   # generator.go, styles.go, geometry.go(중앙 배치)
│   ├── qr/                      # go-qrcode + CP949 인코딩
│   ├── imaging/                 # PNG 검증(청크 CRC, Pillow verify 호환)
│   ├── httpx/                   # Fiber 핸들러(라벨/QR/health/logs)
│   └── logging/
├── web/
│   ├── embed.go                 # //go:embed all:dist (SPA 임베드)
│   ├── dist/                    # Vite 빌드 출력 (.gitkeep만 추적, make가 생성)
│   └── frontend/                # Vite + Svelte 소스
│       └── src/{App.svelte, components/, lib/, styles/}
├── Makefile                     # frontend/build/run/dev/test/clean
├── VERSION                      # 3.0.0 (health 응답에 주입)
└── testdata/golden/             # 레거시 Python 출력 골든(.xlsx) — 패리티 검증용
```

> 레거시 Python 앱(`app.py` 등)은 패리티 비교용으로 저장소에 함께 보존되어 있습니다(은퇴 예정).

---

## 🚀 빌드

빌드 머신에는 **Go 1.26+** 와 **Node 18+** 가 필요합니다. (실행 머신은 아무것도 필요 없음)

```bash
make build      # npm ci + vite build → web/dist, 그다음 go build → bin/qrweb
```

`bin/qrweb` 정적 바이너리(약 14MB, `CGO_ENABLED=0`) 하나가 생성됩니다.

**크로스 컴파일**(타겟 OS/아키텍처가 다를 때):
```bash
cd web/frontend && npm ci && npm run build && cd ../..
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags="-s -w -X qrweb/internal/config.defaultVersion=$(cat VERSION)" -o qrweb-linux ./cmd/qrweb
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -X qrweb/internal/config.defaultVersion=$(cat VERSION)" -o qrweb.exe   ./cmd/qrweb
```

---

## ▶️ 실행

```bash
HOST=0.0.0.0 PORT=8080 ./bin/qrweb
```
브라우저에서 `http://localhost:8080` 접속.

**개발 모드**(Vite HMR + Go 백엔드 동시, 프론트 핫리로드):
```bash
make dev
```

---

## 📦 배포

런타임 의존성이 없어 **바이너리 복사 + 실행**이 전부입니다.

**systemd (Linux 상주 권장)** — `/etc/systemd/system/qrweb.service`:
```ini
[Unit]
Description=QR_Web label generator
After=network.target
[Service]
ExecStart=/opt/qrweb/qrweb
Environment=HOST=0.0.0.0
Environment=PORT=8080
WorkingDirectory=/opt/qrweb
Restart=always
[Install]
WantedBy=multi-user.target
```
```bash
sudo systemctl enable --now qrweb     # 부팅 자동시작 + 기동
# 업데이트: 새 바이너리 복사 후
sudo systemctl restart qrweb
```

- **Windows**: `qrweb.exe` 복사 후 실행(작업 스케줄러로 상주 가능)
- **리버스 프록시(선택)**: 도메인·HTTPS 필요 시 nginx/Caddy를 앞단에 두고 `proxy_pass http://127.0.0.1:8080`
- **Docker(선택, 불필요)**: 정적 바이너리라 `FROM scratch`로 초경량 이미지 가능

---

## 📚 API 엔드포인트

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/api/health` | 서버 상태 + 버전 |
| POST | `/create_label` | 웹 폼(multipart) 라벨 생성 → `.xlsx` 스트리밍 |
| POST | `/api/create_label` | JSON 라벨 생성 → `.xlsx`(base64 인라인) |
| GET | `/api/qr_image/*` | QR 코드 PNG (슬래시 포함 텍스트 `1/3` 지원) |
| POST | `/api/qr_image_base64` | Base64 QR 이미지 |
| GET | `/api/logs` | 로그 조회(`lines`/`level`/`search` 쿼리) |
| POST | `/api/logs/clear` · GET `/api/logs/download` | 로그 초기화/다운로드 |

**`POST /api/create_label` 예시(기기 문서):**
```bash
curl -X POST http://localhost:8080/api/create_label \
  -H "Content-Type: application/json" \
  -d '{"doc_type":"1","binder_size":5,"eq_number":"EQ001","eq_doc_number":"DOC001",
       "eq_doc_title":"테스트 문서","eq_doc_count":3,"eq_doc_department":"기술부","eq_doc_year":2026}'
```

---

## ⚙️ 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `HOST` | `0.0.0.0` | 바인딩 호스트 |
| `PORT` | `5000` | 포트 |
| `LOG_LEVEL` | `INFO` | 로그 레벨 |
| `MAX_QR_FILES` | `50` | QR 업로드 최대 개수 |
| `MAX_QR_FILE_SIZE` | `2MB` | QR 파일당 최대 크기 |
| `MAX_CONTENT_LENGTH` | `16MB` | 요청 본문 최대 크기 |

---

## 🧪 테스트

```bash
go test ./...                          # 백엔드(라벨/엑셀/QR/HTTP) + 골든 패리티
cd web/frontend && npm run build       # 프론트 타입체크(svelte-check)
```

Excel 출력은 레거시 Python 앱의 골든 파일(`testdata/golden/`)과 의미 단위(셀값·병합·치수·테두리·폰트)로 비교해 패리티를 검증합니다. (QR 위치는 v3.0에서 의도적으로 박스 중앙으로 이동 — 전용 중앙 정렬 테스트로 검증.)

---

## 📋 버전 히스토리

### v3.0.0 (2026-06-18)
**Go(Fiber) + Svelte SPA 전면 재작성 — 단일 정적 바이너리**
- 🦫 **백엔드 Go 포팅**: Flask → Fiber. excelize(엑셀)·go-qrcode(QR)·CP949 인코딩. 레거시 출력과 14/14 골든 패리티 검증
- 🧩 **프론트 Vite+Svelte SPA**: 서버렌더 HTML+바닐라 JS → 컴포넌트 SPA. `embed.FS`로 바이너리 내장
- 📦 **단일 정적 바이너리 배포**: 런타임·Docker 불필요. 스트리밍 응답으로 임시 파일 제거
- 🎯 **QR 중앙 정렬**: 하단 박스 정중앙 배치(전 바인더·문서타입)
- 🏷 **로고 개선**: 흰 필 배지 + 파란 inno.N 텍스트
- ✅ **프론트 UX 개선**: 인라인 필드 검증, 준비상태 패널(체크리스트·제출 비활), QR 카운터 상태, 폼 초기화
- 🔁 **API 호환**: `/api/qr_image/*` 슬래시 텍스트(`1/3`) 지원, 인앱 로그 뷰어 모달

### v2.1.1.0 (2026-05-19)
**아키텍처 리팩터링 및 보안 수정 (Python)**
- 🏗 모듈 분리(`document_schema.py`), 레이아웃 설정 분리(`label_layout.py`), 관찰 가능한 `FileLifecycleManager`
- 🛡 경로 탐색 취약점 수정(`safe_join`), QR 텍스트 500자 상한, 67개 테스트로 확충

### v2.1.0 (2026-05-18)
**QR 파일 업로드·드롭존·썸네일 UX 개선 (Python)**
- 📂 파일 업로드/드롭존, 썸네일 `<N>권` 라벨, SortableJS 순서 재배치, Toast 테마화, Data URI 입력

### v1.1.0 (2026-05-11)
**UI/UX 리디자인 (Python)**
- ✨ Pristine Lab & Deep Blue 테마, 다크 모드, Pretendard/Lucide, Toast 알림, uv 워크플로우

### v1.0.0 (2025-07-10)
**초기 릴리즈 (Python)**
- ✅ 기본 QR 생성·Excel 출력, 성능 모니터링·캐싱·파일 정리

---

## 📄 라이선스

MIT 라이선스. 자세한 내용은 `LICENSE` 파일을 참조하세요.
