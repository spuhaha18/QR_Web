# 하네스 구성 계획: QR_Web Go+Vite 지속 개발 팀

## Context

QR_Web은 현재 **Python 3.13 + Flask** 기반 웹앱이다. 연구소 기기/과제 문서용 표준 바인더 라벨을 QR 코드 임베드된 Excel(.xlsx)로 생성한다. 사용자는 3가지 목표(프론트 현대화·성능/확장성·배포 단순화)로 **Go + Vite SPA 전면 재작성**을 결정했다.

확정 사항:
- **타겟 스택**: Go(Fiber) + excelize + go-qrcode 백엔드, Vite + **Svelte** SPA 프론트, `embed.FS`로 단일 정적 바이너리
- **하네스 성격**: 일회성 마이그레이션이 아닌 **지속 개발 팀** — 재작성 완료 후에도 Go+Vite 신규 기능·버그수정에 재사용

이 문서는 그 재작성을 수행/유지할 **에이전트 팀 + 스킬 하네스**를 구성하는 계획이다. (현재 plan mode — 실제 `.claude/` 파일 생성은 승인 후 진행)

### 현황 감사 (Phase 0)
- 신규 구축: `.claude/agents/`, `.claude/skills/`, `.claude/commands/` 모두 없음 (`worktrees/`만 존재)
- 프로젝트 루트 CLAUDE.md 없음 (기존 지침은 상위 `~/CLAUDE.md`)

### 마이그레이션 핵심 위험 (왜 이 팀 구조인가)
`excel_generator.py`(373 LOC)의 시각 패리티가 최대 위험. openpyxl의 세밀한 스타일을 excelize로 1:1 재현해야 한다:
- 행 높이(4행=216, 5행=40.5 등)/열 너비(A·N=0.375, B–M=바인더별 가변)
- 셀 병합(B2:M6 등), thin/medium 테두리(외곽 medium, 내부 thin, 모서리 특수처리)
- QR 이미지 앵커 셀(바인더×문서타입별: `label_layout.py`의 7→E9/E8, 5·3→D9/D8, 1→B9), 크기 75×75px
- 멀티시트 i/N 복제(`B5`, 과제는 `S23`도), 과제 문서 우측 영역(Q21:S23) + print_area `A1:T24`

→ Excel 패리티를 **전담 에이전트 + 전담 스킬**로 분리하고, **생성-검증** 패턴으로 QA가 현재 Python 출력과 diff 비교.

## 실행 모드: 에이전트 팀 (생성-검증 + 파이프라인 하이브리드)

빌더 3 + QA 1. 팀원 간 `SendMessage`로 API 계약 합의, `TaskCreate`로 작업/의존성 추적, `_workspace/` 파일로 산출물 공유.

데이터 흐름:
```
excel-parity-engineer (상류: Excel 코어 = 최대 위험, 먼저 착수)
        │  API 계약(JSON shape) 합의
        ├──> go-backend-engineer (Fiber 라우트/QR/config/lifecycle)
        └──> svelte-frontend-engineer (SPA, API 계약 의존)
                        │
        parity-qa (점진적 검증: 각 모듈 완성 직후, xlsx diff + 경계면 shape 교차검증)
```

## 산출물 1: 에이전트 정의 (`.claude/agents/`)

모든 에이전트 `model: "opus"`. 각 파일에 핵심 역할/작업 원칙/입출력 프로토콜/팀 통신 프로토콜/에러 핸들링/재호출 지침(이전 산출물 존재 시 개선) 포함.

| 에이전트 | 타입 | 역할 | 연결 스킬 |
|---------|------|------|----------|
| `excel-parity-engineer.md` | general-purpose | openpyxl→excelize 포팅, 시각 패리티 책임. `excel_generator.py`+`label_layout.py`+`document_schema.py` 셀값 로직 담당 | `go-excelize-port` |
| `go-backend-engineer.md` | general-purpose | Fiber 앱 구조, Flask 12라우트 미러, go-qrcode, env config, 파일 수명주기(스트리밍 권장), 스키마 검증 | `go-backend-build` |
| `svelte-frontend-engineer.md` | general-purpose | Vite+Svelte SPA, 드래그드롭+data-URI 붙여넣기+svelte-dnd-action 재정렬, API 클라이언트, vite build→embed 디렉토리 | `vite-svelte-spa` |
| `parity-qa.md` | general-purpose | 점진적 QA. 생성 .xlsx vs 현재 Python 출력 diff, API↔Svelte store shape 교차검증 (Explore 아님 — 스크립트 실행 필요) | `parity-qa` |

## 산출물 2: 스킬 (`.claude/skills/`)

### 2-1. `go-excelize-port` (최우선 — 패리티 레퍼런스)
openpyxl→excelize 1:1 매핑 표. `references/`에 분리:
- `styling-map.md`: Font/Border/Side/Alignment/merge_cells/row_dimensions/column_dimensions → excelize `SetCellStyle`/`MergeCell`/`SetRowHeight`/`SetColWidth` 매핑. **단위 차이 주의**(openpyxl 너비=문자수, excelize=문자수 동일하나 픽셀 앵커 다름)
- `image-anchor.md`: openpyxl `add_image`(셀 좌상단 앵커, width/height px) → excelize `AddPictureFromBytes`(OneCell/TwoCell 앵커, `GraphicOptions` 스케일링). 75×75px 재현법
- `multisheet.md`: `copy_worksheet` i/N 복제 → excelize `NewSheet`+`CopySheet`, `B5`/`S23` 갱신, print_area `SetDefinedName`
- `binder-layout.go.md`: `label_layout.py` config dict → Go 구조체 (7→E9/E8, 5·3→D9/D8, 1→B9·B9, column_width 1.875/1.25/1/0.75)

### 2-2. `go-backend-build`
Go 프로젝트 구조 + Fiber. `references/`:
- `project-structure.md`: `cmd/server/`, `internal/{label,excel,qr,config,httpapi}/`, `web/dist/`(embed), `embed.go`
- `routes-map.md`: Flask 12라우트 → Fiber 핸들러 매핑 (`/api/create_label`, `/api/qr_image*`, `/api/health`, `/api/logs`, `/download/*` 등), 요청/응답 shape
- `lifecycle.md`: **권장 — 임시파일 제거, .xlsx를 응답에 직접 스트리밍**(`c.SendStream`). 불가피 시 `time.AfterFunc` 레지스트리로 `file_lifecycle.py` 대체
- `config.md`: env var 기반(stdlib `os.Getenv`), dev/prod/test 분기

### 2-3. `vite-svelte-spa`
- `components.md`: Svelte 컴포넌트 트리(문서타입 선택, 바인더 선택, QR 드롭존, 정렬 리스트), store 설계
- `interactions.md`: 드래그드롭 + data-URI 붙여넣기(clipboard paste) + `svelte-dnd-action` 재정렬 (현 `qr_paste.js` 242 LOC 대체)
- `build-embed.md`: vite `outDir`→Go `web/dist`, `base` 설정, Go `//go:embed web/dist` 연동

### 2-4. `parity-qa`
- 점진적 QA(각 모듈 완성 직후), 경계면 교차검증(Fiber 응답 JSON vs Svelte fetch 파싱 shape)
- `xlsx-diff.md`: 현재 Python 앱으로 레퍼런스 .xlsx 생성 → Go 출력과 셀값/스타일/이미지위치 비교 스크립트
- `scripts/`: 공통 xlsx 비교 헬퍼 번들

### 2-5. 오케스트레이터 스킬 `qr-web-dev` (지속 개발 팀)
- description: **적극적 트리거** + 후속 키워드("Go 백엔드", "Svelte 프론트", "Excel 라벨", "다시/재실행/수정/보완", "라벨 기능 추가")
- Phase 0 컨텍스트 확인: `_workspace/` 존재 여부로 초기/후속/부분 재실행 판별
- `TeamCreate`(4 에이전트) → `TaskCreate`(의존성: parity-engineer 상류) → 자체 조율 → parity-qa 점진 검증 → 종합
- 에러 핸들링(1회 재시도 후 누락 명시), 데이터 전달(태스크+파일+메시지 조합)
- 테스트 시나리오(정상 1 + 에러 1)

## 산출물 3: CLAUDE.md 포인터 (프로젝트 루트 신규)
```markdown
## 하네스: QR_Web Go+Vite 개발
**목표:** QR/Excel 라벨 앱을 Go(Fiber)+Svelte SPA 단일 바이너리로 개발·유지
**트리거:** Go 백엔드/Svelte 프론트/Excel 라벨/마이그레이션 작업 시 `qr-web-dev` 스킬 사용. 단순 질문은 직접 응답.
**변경 이력:** | 2026-06-18 | 초기 구성 | 전체 | - |
```

## 마이그레이션 시퀀싱 (하네스가 실행할 순서)
1. **Excel 코어 먼저** (최대 위험 선검증): excelize로 단일 시트 라벨 1종 생성 → 현재 Python 출력과 시각 비교. 패리티 확인 전 진행 금지
2. 멀티시트 i/N + 바인더별 레이아웃 + 과제 문서 우측 영역 + QR 이미지 앵커 패리티
3. API 계약 합의 → Fiber 라우트 + go-qrcode + config + 스트리밍 응답
4. Svelte SPA(폼/드롭존/붙여넣기/재정렬) → API 연동
5. vite build → `embed.FS` → `go build` 단일 바이너리
6. parity-qa 전체 회귀 + 기존 67 pytest 대응 Go 테스트 포팅

## 검증 (Verification)
- **구조 검증**: `.claude/agents/` 4파일 + `.claude/skills/` 5개(오케스트레이터 포함) frontmatter, `.claude/commands/` 비어있음, CLAUDE.md 포인터 등록
- **트리거 검증**: `qr-web-dev` should-trigger 8~10 / near-miss 8~10 (예: "Excel 차트 PNG 추출"은 트리거 안 됨)
- **드라이런**: 오케스트레이터 Phase 순서, 데이터 전달 dead-link 없음, parity-engineer→backend/frontend 입력 매칭
- **하네스 동작 검증**: `qr-web-dev` 트리거 → 팀 생성 → Excel 코어 1종 생성 → 현재 Python `.xlsx`와 diff 0 확인
- **최종 산출물 검증**: 단일 바이너리 실행 → 브라우저 폼 제출 → 생성 .xlsx가 현재 앱 출력과 셀값/스타일/QR위치 일치

## 미생성 확인
- `.claude/commands/` — 아무것도 생성 안 함 (커맨드 아닌 스킬/에이전트만)
