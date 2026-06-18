# DDD 리팩터 전체 계획 — QR_Web

**목표**: 프로젝트 전체(Go 백엔드 + Svelte 프론트)를 도메인 주도로 정리한다. 도메인 규칙을 그것을 소유해야 할 모듈 뒤로 모으고(locality), 작은 인터페이스에 행위를 집중하며(depth), transport/presentation 계층에서 도메인 로직을 걷어낸다. 모든 단계는 **시각 패리티(golden .xlsx 14종) + 전체 테스트 그린**을 게이트로 한다.

**용어**: module/interface/depth/seam/shallow/deletion test (LANGUAGE.md), 권/시트/QR 이미지/QR 이미지 입력/바인더 사이즈/doc_type/Layout/QRImageSet/CellFont (CONTEXT.md).

**진행 상태 (2026-06-19): Phase 1–7 전부 완료.**
| Phase | 커밋 | 상태 |
|-------|------|------|
| 1 값 객체 | `966a8b5` | ✅ |
| 2 도메인 로직 회수 | `1bbddd8` | ✅ |
| 3 QRText | `a850590` | ✅ |
| 4 라벨 생성 유스케이스(BuildAutoQRImageSet) | `a850590` | ✅ |
| 5 에러→HTTP 매핑 중앙화 | `a850590` | ✅ |
| 6 프론트 도메인 모듈 + vitest | `1ae8c71` | ✅ |
| 7 계약 패리티 테스트 | `1ae8c71` | ✅ |
후보4(qr.go CP949) 비적용 → ADR 기록 완료: `docs/adr/0001-cp949-encoding-stays-in-qr-generation.md`.

---

## 완료 (Phase 1–2, 커밋됨)

### Phase 1 — `966a8b5` 도메인 값 객체
- `label.DocType`(장비/과제), `label.BinderSize`(검증+열너비, 폴백 제거), `label.Layout`(topRow/CountCells/HasPrintArea), `label.QRImageSet`(권=QR 개수 불변식), `label.ValidationError`(타입 에러). rowHeights 단일 소스.

### Phase 2 — `1bbddd8` 도메인 로직 회수
- `label.BuildQRImageSet`(paste intake 규칙+reorder), `label.CellFont`+`Label.CellFonts()`(셀→폰트 소유), `Layout.QRBoxBottomRow`+`narrowColWidth`/`qrSizePx` 상수, `logging.LevelOf`(로그 포맷 파싱).

---

## 잔여 계획

### Phase 3 — 백엔드: QRText 값 객체 (🔴 HIGH)
- **Files**: `internal/qr/qr.go`, `internal/httpx/qr_handler.go`, `internal/httpx/label_handler.go`(auto)
- **Problem**: "비어있지 않음 + ≤500자" 규칙이 qr_handler 2곳에 하드코딩(transport). 도메인(qr.CreateQRPNG)은 검증 안 함 → auto 모드(`label_handler` QRPayload 생성)는 무검증. 같은 규칙 분산 + 한 경로 누락.
- **Solution**: QR 텍스트를 값 객체로. 생성 시 비어있음/길이 검증. 두 핸들러와 auto 경로가 동일 타입 경유. 한국어 메시지 도메인으로 이동.
- **Benefits**: deletion test 통과(규칙이 핸들러로 흩어짐). leverage: 모든 QR 생성 경로가 같은 불변식. 순수 단위 테스트.
- **Risk/게이트**: 낮음. 기존 qr_handler 테스트(500자 거부) 유지 + QRText 단위테스트 추가.

### Phase 4 — 백엔드: 라벨 생성 유스케이스 모듈 (🔴 HIGH)
- **Files**: `internal/httpx/label_handler.go`(paste/auto), 신규 application 모듈
- **Problem**: paste/auto 핸들러가 "요청 파싱→QR 집합 구성→Excel 생성→응답"을 각각 인라인 중복. 응집된 라벨 생성 유스케이스 모듈 없음. HTTP 없이 테스트 불가, 재사용(CLI/배치) 불가.
- **Solution**: 라벨 생성 오케스트레이션을 모듈로 추출(파싱된 Label+DocType+BinderSize+QRImageSet → bytes+filename). paste/auto는 QR 소스(업로드 vs 생성)와 응답 형태(스트림 vs base64)만 다르게, 공통 흐름은 모듈 호출. transport는 얇게.
- **Benefits**: locality(라벨 생성 규칙 1곳), 교차 관심사(감사/로깅) 추가 시 1곳. interface가 테스트 표면 — HTTP 없이 검증.
- **Risk/게이트**: 중. 핸들러 동작/응답 형태 불변 — 기존 핸들러 테스트 전수 통과 + golden 패리티.

### Phase 5 — 백엔드: 에러→HTTP 매핑 중앙화 (🔴 HIGH)
- **Files**: `internal/httpx/*` (errJSON 사용처 8곳), `server.go`
- **Problem**: `errors.Is(err, ErrValidation)?400:500` + "서버 오류가 발생했습니다." 8회 반복. 신규 에러 타입(예: 429) 추가 시 전 핸들러 수정.
- **Solution**: 도메인 에러→HTTP 상태/메시지 매핑을 한 곳(헬퍼 또는 에러 핸들링 미들웨어)으로. 핸들러는 도메인 에러만 반환.
- **Benefits**: locality, 신규 에러 타입 1곳 등록. shallow 반복 제거.
- **Risk/게이트**: 낮음. 상태코드/메시지 불변 — 기존 핸들러 테스트 유지.

### Phase 6 — 프론트: 도메인 모듈 응집 (🔴 HIGH / 🟡 MED)
- **Files**: `web/frontend/src/lib/types.ts`, `validation.ts`, `App.svelte`, `DocTypeSelector`/`BinderSizeSelector`/`EquipmentFields`/`ProjectFields`/`QrThumbnails`/`ReadinessPanel`, `qrStore.ts`
- **Problem**:
  - DocType primitive 문자열 `'1'/'2'` 분기가 컴포넌트/App 전반에 산재(F1).
  - 1cm 규칙 UI hide만(BinderSizeSelector), 검증 모듈 부재(F2).
  - 권=QR 카운트 규칙이 QrThumbnails 표시 + App.isReady + 백엔드 3중 구현(F2).
  - 필수필드 목록 4중복(validation.ts + EquipmentFields/ProjectFields FIELDS + 백엔드)(F3).
  - 필드명 문자열 api.ts/컴포넌트 하드코딩(F3).
  - EquipmentFields/ProjectFields 거의 동일 중복(F4).
  - readiness가 App.svelte 프레젠테이션에 계산(F6).
  - `qrStore.hash`는 SHA-1 아닌 djb2 — 이름 오해(F5).
- **Solution**: 프론트 도메인 모듈(`lib/domain/`)에 규칙·상수 응집 — doc_type 타입+헬퍼, binder 규칙(1cm 포함), 필수필드 단일 정의, 권=QR 카운트 판정, readiness 순수 함수. 컴포넌트는 도메인 모듈 소비만. 필드 정의로 Equipment/Project 폼 통합. hash→fingerprint 정명.
- **Benefits**: 프론트 단일 도메인 소스, 컴포넌트 박막화, 순수 함수 테스트.
- **Risk/게이트**: 중. **결정 (2026-06-19): vitest 도입** — 프론트 도메인 모듈 순수 함수에 vitest 단위 테스트. 게이트 = `vite build` 성공 + `vitest run` 그린.

### Phase 7 — 교차경계: 폴리글랏 도메인 계약 단일화 (🔴 HIGH, 설계 결정 필요)
- **Files**: 백엔드 `label/*` ↔ 프론트 `lib/domain/*` 경계
- **Problem**: 백·프론트가 **같은 도메인**(doc_type 값, binder 집합, 1cm 규칙, 필수필드, 권=QR, 필드명, 한국어 메시지)을 독립 재인코딩. 한쪽 변경 시 조용히 드리프트. 폴리글랏 프로젝트의 가장 깊은 friction.
- **Solution 후보** (사용자 결정 필요):
  - **A. 코드젠**: Go가 진실 원천 → TS 타입/상수 생성. 최강 단일소스, 빌드 파이프라인 추가.
  - **B. 응집+패리티 테스트**: 프론트 도메인 모듈 자체 정의, Go 테스트가 프론트 상수↔백엔드 일치 검증(parity-qa 하네스 활용). 중간.
  - **C. 런타임 계약 endpoint**: 백엔드 `/api/contract` 제공, 프론트 fetch. 런타임 결합.
  - **D. 응집만**: 프론트 도메인 모듈로 응집, CONTEXT.md에 계약 문서화, 기존 parity-qa 의존. 최경량.
- **결정 (2026-06-19): B 채택** — 프론트 `lib/domain/`이 단일 소스, Go 패리티 테스트(`contract_parity_test.go`)가 프론트 도메인 상수 파싱→백엔드 `label.*` 일치 검증, 불일치 시 FAIL. 코드젠/런타임 결합 없이 드리프트 차단.
- **게이트**: Go 패리티 테스트 그린.

---

## 시퀀싱 & 원칙
1. 백엔드 먼저(Phase 3→4→5) — golden 패리티 안전망 있음, 저위험순.
2. 프론트(Phase 6) — 빌드+QA 게이트.
3. 교차경계(Phase 7) — Phase 6 응집 완료 후, 선택안 확정 후.
4. 각 단계: 구현 → `go test -count=1 ./...` 그린 + golden 14/14 → 커밋. 단계 독립 커밋.
5. 패리티 직결(Excel/border) 변경은 출력 바이트 동일 보장(Sprintf 동일 문자열) 원칙 유지.
6. 후보4(qr.go CP949 분리)는 의도적 비적용 — ADR 기록됨(`docs/adr/0001`, EncodeCP949 seam 유지가 근거).

## 비범위
- Python 레거시(app.py 등) — 마이그레이션 참조 원천(freeze), DDD 대상 아님.
- 성능/캐시/파일수명 서브시스템 — Go 포트에서 이미 제거됨.
