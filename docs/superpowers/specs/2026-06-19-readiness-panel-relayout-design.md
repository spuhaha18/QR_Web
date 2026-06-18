# ReadinessPanel 재배치 설계

날짜: 2026-06-19
대상: `web/frontend/src/components/ReadinessPanel.svelte`, `web/frontend/src/styles/style.css`

## 배경

라벨 폼 하단 readiness 패널이 세 요소를 담는다: 문서 정보 미입력 칸 수, QR 이미지 개수(N/1), 초기화 버튼, 라벨 만들기 버튼. 현재 배치 불만 두 가지:

1. **전체 순서·구조** — 체크리스트가 세로 목록이라 시선 무게가 약함.
2. **초기화 버튼 위치** — `초기화`가 `라벨 만들기` 바로 옆(같은 actions 행)이라 오조작 위험, 위계 불명확.

## 현재 구조 (변경 전)

```
┌─ readiness-panel ───────────────────────┐
│  ✕ 문서 정보 — 미입력 N칸   (세로 목록)
│  ✕ QR 이미지 N / 1
│  [초기화]  [──── 라벨 만들기 ────]   (가로 행: 초기화 + submit flex:1)
└──────────────────────────────────────────┘
```

CSS: `style.css` 1180–1189 (`.readiness-panel`, `.readiness-checklist`, `.readiness-actions`, `.reset-btn`).

## 목표 구조 (변경 후) — 안 C

```
┌─ 준비 상태 ───────────────────[↺ 초기화]┐   헤더 행: 좌 제목(muted) + 우 보조버튼
│                                          │
│  ( ✕ 문서 미입력 N칸 )  ( ✕ QR N/1 )    │   가로 상태 칩 2개 (wrap 가능)
│                                          │
│  [────────── 라벨 만들기 ──────────]     │   전폭 primary
└──────────────────────────────────────────┘
```

## 컴포넌트 단위

### 1. 헤더 행 (`readiness-header`)
- `display:flex; justify-content:space-between; align-items:center`.
- 좌: `준비 상태` — 작은 muted 라벨(`text-muted`).
- 우: 초기화 버튼 — `<button type="button">`, 아이콘(`RotateCcw`) + `초기화` 텍스트. 보조 텍스트버튼 스타일(`text-muted`, 투명 배경, hover 시 옅은 배경). `disabled={loading}` 유지.
- 기존 `.readiness-actions` 행에서 초기화 제거.

### 2. 상태 칩 (`readiness-chips`)
- `display:flex; gap; flex-wrap:wrap`.
- 칩 = pill(둥근 모서리, 옅은 bg 틴트, 좌측 마크 `✓`/`✕`).
- 색상은 기존 토큰 재사용: 미충족 `--error-text`, 충족 `--success-text`. 배경은 동일 색 저투명 틴트.
- 문서 칩:
  - 미입력: `✕ 문서 미입력 N칸` (N = `fieldErrorCount`)
  - 완료: `✓ 문서 정보`
- QR 칩:
  - 미충족: `✕ QR N/1` (N=`qrCount`, 1=`docCount`)
  - 충족: `✓ QR N/1`
- `docOk = fieldErrorCount === 0`, `qrOk = qrCount === docCount` (기존 로직 그대로).

### 3. Submit 버튼
- 전폭(`width:100%`). 기존 `.submit-btn` 스타일 유지, `flex:1` 의존 제거(actions 행 사라지므로 직접 전폭).
- `disabled={!isReady || loading}`, 로딩 스피너/문구 기존 그대로.

## Props / 데이터 흐름
- props 시그니처 **변경 없음**: `fieldErrorCount, qrCount, docCount, isReady, loading, onReset`.
- `App.svelte` **변경 없음** (호출부 동일).

## 접근성
- 초기화는 링크 아닌 `<button type="button">` 유지.
- submit `type="submit"` 유지.
- 칩의 `✓`/`✕`는 시각 마크. 의미는 텍스트(`문서 미입력 N칸`, `QR N/1`)에 이미 담겨 별도 aria 불필요. 단, 마크 span에 `aria-hidden="true"` 부여.

## 반응형
- 좁은 폭: 칩 `flex-wrap`으로 2줄 가능. submit 전폭 그대로. 헤더 행 유지.

## 에러 처리
- UI 전용 변경. 신규 에러 경로 없음. 로직(검증/제출/초기화) 불변.

## 테스트 / 검증
- 빌드: `cd web/frontend && npm run build` (또는 `make build`) 통과.
- 수동 확인:
  - 초기 상태: 두 칩 빨강(`✕ 문서 미입력 N칸`, `✕ QR 0/1`), submit disabled.
  - 문서 채움 → 문서 칩 초록 `✓ 문서 정보`.
  - QR 1개 추가 → QR 칩 초록 `✓ QR 1/1`, submit 활성.
  - 초기화 클릭 → 폼/QR 리셋, 칩 빨강 복귀.
  - 로딩 중 초기화·submit 둘 다 disabled.
- 패리티 영향 없음(Excel 출력 무관).

## 범위 밖 (YAGNI)
- 칩 애니메이션/트랜지션 추가 안 함.
- 다른 섹션(QR 섹션 제목의 `✓` 등) 변경 안 함.
- 초기화 확인 다이얼로그 추가 안 함(현 동작 유지).
