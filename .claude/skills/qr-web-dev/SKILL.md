---
name: qr-web-dev
description: QR_Web을 Go(Fiber)+Svelte SPA 단일 바이너리로 개발·마이그레이션·유지할 때 사용하는 오케스트레이터. Go 백엔드/Svelte 프론트/Excel 라벨 생성/QR/마이그레이션 작업을 에이전트 팀(excel-parity-engineer, go-backend-engineer, svelte-frontend-engineer, parity-qa)으로 조율한다. "Go로 바꿔", "마이그레이션", "Excel 라벨", "Svelte 프론트", "백엔드 포팅", "다시/재실행/수정/보완/이어서", "라벨 기능 추가", "패리티 검증" 등 요청 시 반드시 사용.
---

# QR_Web Go+Vite 개발 오케스트레이터

QR/Excel 라벨 앱을 Python Flask → **Go(Fiber)+Svelte SPA 단일 바이너리**로 개발·유지하는 지속 개발 팀을 조율한다.

## 실행 모드: 에이전트 팀 (생성-검증 + 파이프라인 하이브리드)
빌더 3 + QA 1. `TeamCreate`로 팀 구성, `TaskCreate`로 의존성 있는 작업 할당, 팀원이 `SendMessage`로 자체 조율, 산출물은 `_workspace/` 파일 공유.

| 에이전트 | 역할 | 스킬 |
|---------|------|------|
| excel-parity-engineer | Excel 생성 시각 패리티 (최대 위험, 상류) | go-excelize-port |
| go-backend-engineer | Fiber 라우트/QR/config/schema | go-backend-build |
| svelte-frontend-engineer | Vite+Svelte SPA | vite-svelte-spa |
| parity-qa | 점진적 검증 (xlsx diff + 경계면) | parity-qa |

모든 Agent 호출에 `model: "opus"` 명시.

## Phase 0: 컨텍스트 확인 (먼저 실행)
`_workspace/` 존재 여부로 실행 모드 판별:
- **미존재** → 초기 실행(전체 마이그레이션)
- **존재 + 부분 수정 요청**(예: "과제 문서 테두리만 고쳐") → 부분 재실행(해당 에이전트만 재호출, 이전 산출물 읽고 개선)
- **존재 + 새 입력/대규모 변경** → 기존 `_workspace/`를 `_workspace_prev/`로 이동 후 새 실행

## Phase 1: 팀 구성 + 골든 캡처
1. `TeamCreate(qr-web-team, [excel-parity-engineer, go-backend-engineer, svelte-frontend-engineer, parity-qa])`.
2. parity-qa에게 **골든 캡처 먼저** 지시(`scripts/capture_golden.py`로 현재 Python 출력을 `testdata/golden/`에). 모든 Excel 패리티의 오라클.
3. Go 스캐폴딩(`go mod init`, 디렉토리, config/logging).

## Phase 2: 데이터 계층 (병렬)
- go-backend-engineer: `internal/label/schema.go`, `internal/imaging/png.go`, `internal/qr/`.
- excel-parity-engineer: `internal/label/layout.go`.
- 두 에이전트가 **Label 인터페이스 시그니처 합의**(SendMessage). parity-qa: schema/layout/qr/png 테스트 검증.

## Phase 3: Excel 코어 (크리티컬 게이트)
- excel-parity-engineer: `internal/excel/`. 하위 단계(레이아웃→테두리→폰트→멀티시트→QR)마다 parity-qa가 골든 비교.
- **패리티 통과 전 Phase 4 진행 금지.** 위험 #1(스타일 합성)·#2(이미지 앵커)·#3(CopySheet) 여기서 해결.

## Phase 4: HTTP 계층
- go-backend-engineer: `internal/httpx/`, 스트리밍 응답, 미들웨어. **API 계약을 `_workspace/E_api_contract.md`에 명시** → 이게 프론트 선행조건.
- parity-qa: 핸들러 테스트(한국어 에러/상태코드), 기존 templates로 스모크 테스트 가능.

## Phase 5: 프론트 (API 계약 확정 후)
- svelte-frontend-engineer: `web/frontend/` Svelte SPA, dnd/드롭존/제출, dev 프록시.
- parity-qa: 경계면 교차검증(fetch shape vs API 계약).

## Phase 6: Embed + 빌드 + 컷오버
- svelte: vite build → `web/dist`. go-backend: `embed.FS` 연동, Makefile, 단일 바이너리.
- parity-qa: 전체 회귀 + 단일 바이너리 E2E(브라우저 폼→.xlsx, 골든 일치).

## 데이터 전달
- 태스크 기반(`TaskCreate`/`Update`, 의존성) + 파일 기반(`_workspace/{phase}_{agent}_{artifact}.md`, 최종 산출물은 프로젝트 경로) + 메시지 기반(`SendMessage` 실시간 조율).
- 파일명 컨벤션: `B_label.md`, `C_qr.md`, `D_excel_parity.md`, `E_api_contract.md`, `F_frontend.md`, `QA_report.md`.

## 에러 핸들링
- 에이전트 실패 시 1회 재시도. 재실패면 해당 결과 없이 진행하되 `_workspace/QA_report.md`에 누락 명시.
- 상충 데이터(예: 패리티 불일치 원인 이견)는 삭제 말고 출처 병기 후 사용자 판단 요청.
- **크리티컬 게이트(Phase 3 패리티)는 재시도 우회 금지** — 통과까지 반복.

## 테스트 시나리오
**정상 흐름**: "QR_Web을 Go로 마이그레이션해줘" → Phase 0 초기 판별 → 팀 구성 + 골든 캡처 → 데이터 계층(병렬, 테스트 green) → Excel 코어(골든 매트릭스 일치) → HTTP(한국어 에러 보존) → Svelte(경계면 일치) → 단일 바이너리(E2E 골든 일치) → 사용자 피드백 수집.

**에러 흐름**: Phase 3에서 과제 7cm QR 앵커가 골든과 불일치 → parity-qa가 "E8 기대인데 E9에 앵커" 특정 보고 → excel-parity-engineer가 `image-anchor.md` 재확인, one-cell 앵커/스케일 수정 → 재검증 → 통과 후 진행. (게이트라 우회 안 함.)

## 후속/피드백
- 실행 완료 후 사용자에게 "개선할 부분/팀 구성 변경 의견" 1회 질의(강요 안 함).
- 피드백 유형별 반영: 결과 품질→해당 스킬, 역할→에이전트 정의, 순서→이 오케스트레이터, 트리거 누락→description. 모든 변경은 CLAUDE.md 변경 이력에 기록.
