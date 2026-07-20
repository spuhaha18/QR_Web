# 로그 기능 개선 설계

날짜: 2026-07-20
상태: 설계 확정 (구현 전)

## 목표

1. **운영 안정성** — 로그 로테이션, 백업 파일 무한 누적 제거
2. **구조화 로깅** — slog 기반 JSON lines, 수집기(Loki 등)·`jq` 호환
3. **요청 추적** — 요청별 request ID, 처리시간 포함 액세스 로그
4. **뷰어 개선** — 한글 메시지 렌더, 필드 기반 필터

Docker 배포(파일 + 볼륨 마운트) 전제. 로그 파일은 JSON lines, 뷰어가 한글로 번역 표시.

## 현재 상태

- `internal/logging/logging.go`: Python `logging.basicConfig` 포팅 커스텀 로거. 텍스트 포맷, 파일+stdout, 로테이션 없음.
- `internal/httpx/logs_handler.go`: `GET /api/logs`(tail/레벨/검색), `POST /api/logs/clear`(작업 디렉토리에 `app_backup_*.log` 생성 후 truncate), `GET /api/logs/download`.
- `internal/httpx/server.go:102`: 전 요청 액세스 로그.
- `web/frontend/src/components/LogsModal.svelte`: 모달 뷰어, 수동 새로고침, 200줄, substring 레벨 색칠.

## 설계 결정

### 1. 로거 코어 (`internal/logging` 재작성)

- `log/slog` + `slog.NewJSONHandler`. 출력 `io.MultiWriter(lumberjack, os.Stdout)`.
- `gopkg.in/natefinch/lumberjack.v2`: `logs/app.log`, MaxSize 10MB, MaxBackups 5, 압축 off, MaxAge 무제한.
- env: 기존 `LOG_LEVEL`, `LOG_FILE` 유지 + `LOG_MAX_SIZE_MB`(기본 10), `LOG_MAX_BACKUPS`(기본 5) 추가.
- 레벨 표기는 slog 기본(`DEBUG`/`INFO`/`WARN`/`ERROR`). `ReplaceAttr` 커스텀 없음.
- `msg`는 안정된 영어 이벤트 키(예: `"label generated"`, `"server started"`, `"logs cleared"`, `"request"`). 가변 데이터는 전부 slog 필드로.
- 기존 printf 스타일 `Logger.Info(format, ...)` 호출부를 slog 필드 스타일로 전환.
- 구 커스텀 로거의 `LevelOf()`, 텍스트 포맷 코드 삭제.

### 2. 요청 추적 (`internal/httpx`)

- Fiber `middleware/requestid` 추가, 기본 설정(클라이언트 `X-Request-ID` 재사용, 없으면 UUID 생성, 응답 헤더 자동).
- 액세스 로그 미들웨어: **`/api/*` 경로만** 기록, 단 **`/api/logs*` 제외**, 정적/SPA 파일 제외.
  - `msg="request"`, 필드: `method`, `path`, `status`, `duration_ms`, `ip`, `request_id`.
- 핸들러 비즈니스 로그(라벨 생성 등)에도 `request_id` 필드 포함 — 요청 단위 상관관계.

### 3. 뷰어 API (`logs_handler.go`)

- `GET /api/logs`: JSON lines 파싱 → 객체 배열 반환.
  - 파라미터: `lines`(기본 100, 최대 1000), `level`(`WARN`/`WARNING` 양쪽 수용), `request_id`(정확 매치). `search` 파라미터 **제거**(클라이언트로 이동).
  - 응답 항목: `{time, level, msg, fields}`. 나머지 필드는 `fields` 객체로 중첩(경계 일관성 유지). JSON 파싱 실패 줄(구 텍스트 포맷 등)은 `{level:"INFO", msg:"<원문>", legacy:true}` — 버리지 않음. 기존 파일 마이그레이션은 이 fallback으로 해결(별도 조치 없음, 로테이션으로 자연 소멸).
- `POST /api/logs/clear`: `lumberjack.Rotate()` 호출 후 로테이션 백업 파일 전부 삭제. `os.Truncate` 및 `app_backup_*.log` 생성 로직 삭제. (외부 truncate는 lumberjack 내부 size 카운터와 불일치 → Rotate API 사용)
- `GET /api/logs/download`: 현재 파일만(변경 없음). 로테이션 백업은 볼륨에서 직접 접근.
- 뷰어/다운로드 모두 현재 파일만 대상. 로테이션 직후 tail이 짧아지는 것 감수(필요 시 추후 확장).

### 4. 뷰어 UI (`LogsModal.svelte` + `logMessages.ts` 신설)

- **한글 메시지 카탈로그** `logMessages.ts`: 이벤트 키 → 한글 템플릿 함수. 프론트 단독 관리.
  - `"label generated"` → `라벨 생성: {file} ({ip})`
  - `"request"` → `{method} {path} → {status} ({duration_ms}ms)`
  - `"logs cleared"` → `로그 초기화됨`
  - 미등록 키·legacy 줄 → 원문 그대로 (fallback이 drift 안전망).
- 레벨 필터 드롭다운: 전체/디버그/정보/경고/오류 (DEBUG 옵션 신설).
- 레벨 뱃지 한글+색상 (substring 색칠 제거, 필드 기반).
- 시간 로컬 포맷 `YYYY-MM-DD HH:mm:ss`.
- request ID: 앞 8자 표시, 클릭 시 해당 ID 필터(`request_id` 파라미터 재조회).
- 검색: 클라이언트에서 한글 렌더 텍스트 + raw 필드값 양쪽 매치.
- raw JSON 토글(디버깅용).

### 5. 테스트

- `internal/logging`: JSON 출력 필드 검증, 레벨 필터, 로테이션 설정 반영.
- `internal/httpx`: 파싱/필터/legacy 줄 핸들러 테스트, clear의 Rotate+백업 삭제, 액세스 로그 경로 필터링.
- 프론트: `logMessages.ts` 키 매핑 단위 테스트.

## 검토한 대안

- **커스텀 로거 확장**(의존성 0): 로테이션 재발명 리스크로 기각.
- **OTel/Loki 스택**: 단일 바이너리 셀프호스팅 규모에 과함, 기각.
- **stdout 전용 로깅**: 뷰어 UI 재설계 필요, 파일+볼륨이 자급자족이라 기각.
- **텍스트 포맷 유지**: 필드 필터·request ID 추적·뷰어 개선 목표와 충돌, 기각.
- **서버측 한글 번역/검색**: 로그 파일 오염 또는 역매핑 복잡도로 기각, 표시 계층(프론트)에서 해결.
