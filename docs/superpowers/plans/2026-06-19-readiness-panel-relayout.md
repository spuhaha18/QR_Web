# ReadinessPanel 재배치 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** readiness 패널을 가로 상태 칩 + 헤더 우상단 분리된 초기화 버튼 + 하단 전폭 '라벨 만들기' 구조로 재배치한다.

**Architecture:** 순수 프레젠테이션 변경. `ReadinessPanel.svelte` 마크업을 헤더 행 / 상태 칩 / submit 3블록으로 재구성하고, `style.css`의 readiness 블록(1180–1189)을 교체·확장한다. props 시그니처·`App.svelte`·검증/제출/초기화 로직 모두 불변.

**Tech Stack:** Svelte 4, Vite 5, 바닐라 CSS(토큰 기반), lucide-svelte 아이콘.

## Global Constraints

- props 시그니처 변경 금지: `fieldErrorCount, qrCount, docCount, isReady, loading, onReset`.
- `App.svelte` 호출부 변경 금지 (164번 줄).
- 색상은 기존 CSS 토큰만 사용: `--error-text`, `--success-text`, `--text-muted`, `--border-color`, `--surface-color`. 신규 하드코딩 색 금지.
- 초기화는 `<button type="button">`, submit은 `<button type="submit">` 유지.
- 검증 게이트: `cd web/frontend && npm run build` (svelte-check 타입체크 + vite build) 통과. (프론트 단위 테스트 프레임워크 없음 — 신규 도입 안 함.)
- 로직(검증/제출/초기화/`docOk`/`qrOk`) 변경 금지.

---

### Task 1: ReadinessPanel 마크업 + CSS 재배치

**Files:**
- Modify: `web/frontend/src/components/ReadinessPanel.svelte` (전체 15–36줄 템플릿)
- Modify: `web/frontend/src/styles/style.css:1180-1189` (readiness 블록 교체 + 칩/헤더 클래스 추가)

**Interfaces:**
- Consumes (props, App.svelte에서 주입):
  - `fieldErrorCount: number`, `binderOk: boolean`, `qrCount: number`, `docCount: number`, `isReady: boolean`, `loading: boolean`, `onReset: () => void`
- Produces: 없음 (말단 컴포넌트, 외부 소비자 없음)
- 내부 파생값 (기존 유지): `docOk = fieldErrorCount === 0`, `qrOk = qrCount === docCount`

> 갱신(바인더 필수화 커밋): 바인더 크기 미선택을 패널에서 드러내기 위해 `binderOk` prop과 바인더 상태 칩이 추가됐다. 아래 Step 1 마크업은 현재 출하된 3-칩 계약을 반영한다.

- [ ] **Step 1: 마크업 교체**

`ReadinessPanel.svelte` 전체를 아래로 교체:

```svelte
<script lang="ts">
  import { Printer, Loader2, RotateCcw } from 'lucide-svelte';

  export let fieldErrorCount: number;
  export let binderOk: boolean;
  export let qrCount: number;
  export let docCount: number;
  export let isReady: boolean;
  export let loading: boolean;
  export let onReset: () => void;

  $: docOk = fieldErrorCount === 0;
  $: qrOk = qrCount === docCount;
</script>

<div class="readiness-panel">
  <div class="readiness-header">
    <span class="readiness-title">준비 상태</span>
    <button type="button" class="reset-btn" on:click={onReset} disabled={loading}>
      <RotateCcw size={16} /> 초기화
    </button>
  </div>

  <div class="readiness-chips">
    <span class="status-chip" class:ok={docOk}>
      <span class="chip-mark" aria-hidden="true">{docOk ? '✓' : '✕'}</span>
      {#if docOk}문서 정보{:else}문서 미입력 {fieldErrorCount}칸{/if}
    </span>
    <span class="status-chip" class:ok={binderOk}>
      <span class="chip-mark" aria-hidden="true">{binderOk ? '✓' : '✕'}</span>
      {#if binderOk}바인더 크기{:else}바인더 미선택{/if}
    </span>
    <span class="status-chip" class:ok={qrOk}>
      <span class="chip-mark" aria-hidden="true">{qrOk ? '✓' : '✕'}</span>
      QR {qrCount}/{docCount}
    </span>
  </div>

  <button type="submit" class="submit-btn" disabled={!isReady || loading}>
    {#if loading}
      <Loader2 size={20} class="submit-icon spin" /> 생성 중...
    {:else}
      <Printer size={20} class="submit-icon" /> 라벨 만들기
    {/if}
  </button>
</div>
```

- [ ] **Step 2: CSS 블록 교체**

`style.css`의 1180–1189줄 (`.readiness-panel` ~ `.reset-btn:disabled`) 전체를 아래로 교체:

```css
.readiness-panel { margin-top: 16px; padding: 16px; border: 1px solid var(--border-color); border-radius: 12px; background: var(--surface-color); }

.readiness-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.readiness-title { font-weight: 700; color: var(--text-muted); font-size: 0.95rem; }

.readiness-chips { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 16px; }
.status-chip { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border-radius: 999px; font-weight: 600; font-size: 0.9rem; color: var(--error-text); background: color-mix(in srgb, var(--error-text) 12%, transparent); border: 1px solid color-mix(in srgb, var(--error-text) 30%, transparent); }
.status-chip.ok { color: var(--success-text); background: color-mix(in srgb, var(--success-text) 12%, transparent); border-color: color-mix(in srgb, var(--success-text) 30%, transparent); }
.status-chip .chip-mark { font-weight: 800; }

.reset-btn { display: inline-flex; align-items: center; gap: 6px; padding: 6px 10px; border: none; border-radius: 8px; background: transparent; color: var(--text-muted); font-weight: 600; font-size: 0.9rem; cursor: pointer; font-family: inherit; }
.reset-btn:hover:not(:disabled) { background: var(--border-color); }
.reset-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.readiness-panel .submit-btn { width: 100%; max-width: none; margin: 0; }
```

근거: 기존 `.submit-btn`은 `max-width:320px; margin:40px auto 0`이라 패널 안에서 가운데 좁게 뜸 → 마지막 규칙으로 전폭 override. `.readiness-actions`/`.readiness-checklist` 규칙은 더 이상 참조되지 않으므로 위 교체로 제거됨.

- [ ] **Step 3: 빌드 + 타입체크**

Run: `cd web/frontend && npm run build`
Expected: PASS — svelte-check 0 errors, `vite build`가 `../dist`에 산출. `ReadinessPanel` 관련 미사용/타입 경고 없음.

- [ ] **Step 4: 수동 시각 확인 (dev 서버)**

Run: `cd web/frontend && npm run dev` 후 브라우저로 접속.
확인 항목:
- 초기 상태: 칩 2개 빨강 — `✕ 문서 미입력 N칸`, `✕ QR 0/1`. 헤더 우상단 `↺ 초기화`. submit(`라벨 만들기`) disabled·전폭.
- 문서 필드 채움 → 문서 칩 초록 `✓ 문서 정보`.
- QR 1개 추가 → QR 칩 초록 `✓ QR 1/1`, submit 활성.
- `초기화` 클릭 → 폼·QR 리셋, 칩 빨강 복귀.
- 라이트/다크 토글 시 칩·버튼 색 토큰 정상.
- 창 좁히면 칩 wrap, submit 전폭 유지.

- [ ] **Step 5: Commit**

```bash
git add web/frontend/src/components/ReadinessPanel.svelte web/frontend/src/styles/style.css
git commit -m "feat(frontend): relayout readiness panel — horizontal chips, detached reset"
```

---

## Self-Review

**1. Spec coverage:**
- 전체 순서·구조 변경 → Task 1 Step 1(헤더/칩/submit 3블록) ✓
- 초기화 버튼 분리 → 헤더 우상단 이동, actions 행 제거 ✓
- 가로 상태 칩 + 색 토큰 재사용 → Step 2 `.status-chip` ✓
- submit 전폭 → Step 2 마지막 override 규칙 ✓
- props/App.svelte 불변 → script 블록·시그니처 유지, App.svelte 미수정 ✓
- 접근성(button type, aria-hidden 마크) → Step 1 마크업 ✓
- 반응형 wrap → `.readiness-chips flex-wrap` ✓
- 검증 → Step 3 build, Step 4 수동 ✓

**2. Placeholder scan:** TBD/TODO 없음. 모든 코드 블록 완전 기재.

**3. Type consistency:** props 6개 시그니처 spec·App.svelte와 일치. `docOk`/`qrOk` 명명 기존과 동일. `color-mix` 사용 — 모던 브라우저 지원(타깃 사내 크롬 기준 OK); 미지원 우려 시 fallback은 범위 밖.

범위 밖(YAGNI): 칩 트랜지션, 초기화 확인 다이얼로그, 테스트 프레임워크 도입 — 모두 제외.
