# 설계문서: 프론트엔드 UX 개선 (향상된 단일 페이지)

> 작성일 2026-06-18 · 상태: 설계 승인(구현 전) · 브랜치 feat/go-vite-migration

## Context

QR_Web Svelte SPA의 라벨 제작 폼은 기능은 완성됐으나 UX 마찰이 있다. 사용자가 4개 마찰점을 지목했다:

1. **검증/에러 피드백** — `App.svelte`의 `validateForm()`이 **제출 시에만** 동작하고 단일 토스트("모든 필드를 채워주세요.")만 띄움. 어느 필드가 빈지 모름. 제출 버튼은 항상 활성(로딩 중만 비활).
2. **QR 개수·순서 혼란** — 권수 대비 업로드 수 불일치가 제출 시 토스트로만 드러남.
3. **입력 흐름/단계 안내** — 한 페이지 긴 폼, 섹션 구분만 있고 진행/완료 인지 수단 없음.
4. **재제출/초기화** — 폼·QR을 비우는 UI 없음(`qrStore.clearItems`는 존재하나 미연결).

**목표**: 기존 단일 페이지 구조를 유지하면서(반복 제작 빠름·익숙함) 인라인 검증·준비상태 안내·초기화를 더해 네 마찰점을 해소한다. 위저드/검증 라이브러리는 과임이라 배제(YAGNI).

**비목표**: 위저드 전환, felte/zod 등 의존성 추가, 백엔드 검증 변경(백엔드는 최종 권위로 유지, 한국어 에러 보존), 레이아웃 전면 재설계.

---

## 기능 ① 인라인 필드 검증

### 현재
`validateForm()`이 제출 시 모든 필수값을 한 번에 체크 → 하나라도 비면 `showError('모든 필드를 채워주세요.')` 단일 토스트.

### 목표
- 필드별 **실시간 검증** + 빈 값이면 입력 아래 인라인 메시지 + invalid 테두리.
- 표시 타이밍: 해당 필드 **blur 후** 또는 **제출 시도 후**(touched/제출플래그 기준). 초기 빈 폼에 빨간색 도배 방지.
- 숫자 필드(`eq_doc_count`/`pjt_doc_count`/`eq_doc_year`): 정수 ≥1. (백엔드 `safe_int_conversion`의 `max(1,...)`과 정합. 연도는 필수(빈 값 허용 시 서버가 현재년으로 조용히 대체하므로 클라에서 막음).)
- 기존 단일 토스트 제거(제출 시 검증 실패는 인라인으로 드러나고 제출 버튼이 비활).

### 검증 규칙 (필드별)
- 기기: `eq_number`,`eq_doc_number`,`eq_doc_title`,`eq_doc_department` 필수(trim 비어있지 않음). `eq_doc_count` 정수 ≥1. `eq_doc_year` 정수 ≥1 (필수 — 빈 값 허용 시 서버가 현재년으로 조용히 대체하므로 클라에서 막음).
- 과제: `pjt_number`,`pjt_test_number`,`pjt_doc_title`,`pjt_doc_writer` 필수. `pjt_doc_count` 정수 ≥1.
- 메시지: 각 필드 placeholder/라벨에 맞춘 한국어("문서 번호를 입력하세요." 등). 필드별 고정 문구.

---

## 기능 ② QR 개수·순서 명료화

### 현재
`QrThumbnails`가 `docCount`를 받아 카운터 표시. 불일치는 제출 토스트.

### 목표
- 카운터 **`N / 권수`**를 눈에 띄게 표시 + 상태색:
  - 부족(`N < 권수`): 중립(회색/주의) "n장 더 필요"
  - 일치(`N === 권수`): 초록 ✓
  - 초과(`N > 권수`): 빨강 "n장 초과"
- 썸네일 캡션 "i권"(표시 순서) 유지 + "드래그로 순서 변경" 힌트 유지.
- 카운터 상태가 준비상태 패널(기능③)에 반영.

---

## 기능 ③ 준비상태 패널 + 흐름 안내

### 목표
- 제출 버튼 영역에 **준비상태 패널**(신규 컴포넌트) — 체크리스트:
  - `문서 정보`: 모든 필수 필드 유효면 ✓, 아니면 ✕ + "미입력 n칸".
  - `QR 이미지`: `N / 권수` + 일치면 ✓, 아니면 ✕.
- **제출 버튼은 `isReady`(필드 에러 0 && QR 수 일치) 전까지 비활성.** 비활 이유를 패널 체크리스트가 설명(왜 막혔는지 항상 보임).
- 로드 시 첫 필수 필드 자동 포커스.
- 섹션 헤더(`기본 설정`/`기기 문서 정보`/`QR 이미지`)에 해당 섹션 완료 시 작은 ✓ 표시(가벼운 단계 인지). 정보 섹션 = 필드 유효, QR 섹션 = 수 일치.

---

## 기능 ④ 초기화 / 재제출

### 목표
- 준비상태 패널에 **초기화 버튼**:
  - 필드 기본값 복귀: 텍스트 빈 값, `*_doc_count = 1`, `eq_doc_year = 현재년`.
  - `clearItems()`로 QR 전체 제거.
  - touched/제출플래그 리셋(인라인 에러 사라짐).
  - 문서타입/바인더는 현재 선택 유지(반복 제작 시 같은 종류 연속 작성 편의). 확인창 없음(폼 리셋은 가역적 작업, 마찰 최소화).
- 제출 성공 후 값 유지(연속 제작 편의), 초기화는 상시 가능.

---

## 아키텍처 (격리·재사용)

| 파일 | 종류 | 책임 |
|------|------|------|
| `web/frontend/src/lib/validation.ts` | 신규 | 순수 함수 `validateEquipment(e): Record<keyof EquipmentForm,string>` / `validateProject(p): Record<...,string>` — 빈 에러맵=유효. 필드 메시지 상수. UI 무관, 테스트 용이. |
| `web/frontend/src/components/ReadinessPanel.svelte` | 신규 | 체크리스트(문서정보/QR) + 초기화 버튼 + 제출 버튼. props: `fieldErrorCount`, `qrCount`, `docCount`, `isReady`, `loading`, `onReset`. 제출은 form submit. |
| `web/frontend/src/components/EquipmentFields.svelte` | 수정 | `errors: Record<string,string>` prop + 필드별 blur→touched + invalid 클래스 + 인라인 메시지. |
| `web/frontend/src/components/ProjectFields.svelte` | 수정 | 동일 패턴. |
| `web/frontend/src/components/QrThumbnails.svelte` | 수정 | 카운터 상태색(부족/일치/초과). |
| `web/frontend/src/App.svelte` | 수정 | 반응형 `errors`/`isReady` 파생, 제출 비활 제어, 초기화 핸들러, 첫 필드 포커스, 섹션 ✓. 기존 단일 토스트 제거. |
| `web/frontend/src/styles/style.css` | 수정 | invalid 입력·인라인 에러·준비패널·카운터 상태·섹션 ✓ 스타일. |

### 데이터 흐름 (Svelte 반응형)
```
$: errors = docType === '1' ? validateEquipment(equipment) : validateProject(project)
$: fieldErrorCount = Object.values(errors).filter(Boolean).length
$: qrCount = $qrItems.length
$: qrOk = qrCount === docCount
$: isReady = fieldErrorCount === 0 && qrOk
```
- `touched: Record<string,boolean>` + `submitAttempted: boolean` → 인라인 에러는 `touched[field] || submitAttempted`일 때만 표시.
- 제출 핸들러: `isReady` 아니면 `submitAttempted = true`로 에러 노출(버튼은 비활이라 정상 경로는 차단됨 — 방어). `isReady`면 기존 `submitLabel` 흐름.

### 에러 핸들링
- 클라이언트 검증은 UX 보조. **백엔드가 최종 권위** — 서버 4xx/5xx 에러는 기존대로 토스트(`submitLabel` catch). 한국어 메시지 보존.
- QR 수 일치 클라 차단 + 서버도 강제(현행 유지) — 이중 방어.

### 검증/테스트
- `validation.ts`는 순수 함수라 테스트 가능. **프론트 단위 테스트 러너(vitest)는 현재 미설치 → 도입하지 않음(YAGNI).** 검증은 `npm run build`(svelte-check 0/0) + `/browse` 인터랙티브 QA로 수행.
- `/browse` QA 시나리오: ①빈 폼 제출버튼 비활 확인 ②필드 blur 시 인라인 에러 ③필드 채우면 ✓·에러 사라짐 ④QR 0/1→1/1 카운터 색·✓ ⑤준비완료 시 제출 활성 ⑥초기화로 폼·QR 비움 ⑦제출→다운로드 정상.

---

## End-to-End 검증
1. `cd web/frontend && npm run build` → svelte-check 0 errors/0 warnings.
2. `make build` → 단일 바이너리, `/browse goto localhost:PORT`로 위 QA 시나리오 7종.
3. 제출 성공(.xlsx 다운로드) + 백엔드 에러 경로(예: QR 수 불일치를 클라가 막는지) 확인.
4. 다크모드에서 invalid/준비패널 스타일 가독성 확인.

## YAGNI (안 함)
- 위저드/다단계 전환. felte/zod. vitest 도입. 백엔드 검증 변경. 자동 저장/최근값 기억(요청 범위 밖).
