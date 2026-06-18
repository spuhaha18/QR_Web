# 프론트엔드 UX 개선 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Svelte 라벨 폼에 인라인 필드 검증·QR 카운터 상태·준비상태 패널·초기화를 더해 4개 UX 마찰점을 해소한다.

**Architecture:** 순수 검증 함수(`lib/validation.ts`)를 App이 반응형으로 소비해 `errors`/`isReady`를 파생하고, 필드 컴포넌트는 인라인 에러를 표시하며, 신규 `ReadinessPanel`이 체크리스트+초기화+제출을 담당한다. 기존 단일 페이지 구조 유지, 신규 의존성 없음.

**Tech Stack:** Vite + Svelte + TypeScript, lucide-svelte(기존), svelte-dnd-action(기존). 백엔드/Go 변경 없음.

## Global Constraints

- 브랜치 `feat/go-vite-migration`. 프론트 디렉토리 `web/frontend/`. node/npm: `cd web/frontend`.
- **신규 npm 의존성 금지**(위저드/felte/zod/vitest 없음). 프론트 단위 테스트 러너 없음.
- **검증 게이트 = `npm run build`(svelte-check 0 errors/0 warnings) + `/browse` 인터랙티브 QA.** 각 Task는 빌드 통과 + (런타임 동작은) browse 확인.
- 백엔드는 최종 검증 권위 — 클라 검증은 UX 보조. 한국어 에러 문구 보존.
- 제출 버튼은 `isReady`(필드 에러 0 && QR 수 == 권수) 전까지 비활성.
- 초기화는 문서타입/바인더 유지, 필드·QR만 리셋, 확인창 없음.
- **색은 하드코딩 hex 금지 — 기존 테마 변수 사용**: 성공/일치 `var(--success-text)`, 에러/초과 `var(--error-text)`, 중립 `var(--text-muted)`. (이 변수들은 `[data-theme="dark"]`에서 재정의되어 다크모드 대비 자동 처리됨.)
- `/browse` 바이너리: `B="$HOME/.claude/skills/gstack/browse/dist/browse"`. go 빌드 필요시 `export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" && make build`. 앱 기동 `PORT=5090 ./bin/qrweb &`.

---

### Task 1: 순수 검증 함수 (lib/validation.ts)

**Files:**
- Create: `web/frontend/src/lib/validation.ts`

**Interfaces:**
- Produces:
  - `type FieldErrors = Record<string, string>`
  - `validateEquipment(e: EquipmentForm): FieldErrors` — 빈 맵 = 유효.
  - `validateProject(p: ProjectForm): FieldErrors`

- [ ] **Step 1: validation.ts 작성**

`web/frontend/src/lib/validation.ts`:
```ts
import type { EquipmentForm, ProjectForm } from './types';

export type FieldErrors = Record<string, string>;

const REQUIRED_EQ: Record<string, string> = {
  eq_number: '마스터코드를 입력하세요.',
  eq_doc_number: '문서 번호를 입력하세요.',
  eq_doc_title: '문서 제목을 입력하세요.',
  eq_doc_department: '작성 부서를 입력하세요.',
};

const REQUIRED_PJ: Record<string, string> = {
  pjt_number: '마스터코드를 입력하세요.',
  pjt_test_number: '시험 번호를 입력하세요.',
  pjt_doc_title: '문서 제목을 입력하세요.',
  pjt_doc_writer: '작성자를 입력하세요.',
};

function isPositiveInt(v: unknown): boolean {
  const n = Number(v);
  return Number.isInteger(n) && n >= 1;
}

export function validateEquipment(e: EquipmentForm): FieldErrors {
  const errors: FieldErrors = {};
  for (const [k, msg] of Object.entries(REQUIRED_EQ)) {
    if (!String((e as Record<string, unknown>)[k] ?? '').trim()) errors[k] = msg;
  }
  if (!isPositiveInt(e.eq_doc_count)) errors.eq_doc_count = '권수는 1 이상이어야 합니다.';
  // 연도: 빈 값은 서버 기본 처리(현재년)이라 허용. 값이 있으면 정수 ≥1.
  if (String(e.eq_doc_year ?? '').trim() && !isPositiveInt(e.eq_doc_year)) {
    errors.eq_doc_year = '연도가 올바르지 않습니다.';
  }
  return errors;
}

export function validateProject(p: ProjectForm): FieldErrors {
  const errors: FieldErrors = {};
  for (const [k, msg] of Object.entries(REQUIRED_PJ)) {
    if (!String((p as Record<string, unknown>)[k] ?? '').trim()) errors[k] = msg;
  }
  if (!isPositiveInt(p.pjt_doc_count)) errors.pjt_doc_count = '권수는 1 이상이어야 합니다.';
  return errors;
}
```

> NOTE: `REQUIRED_PJ` 메시지의 필드 라벨이 `ProjectFields.svelte`의 실제 라벨과 일치하는지 확인하라(시험 번호/작성자 등). 다르면 컴포넌트 라벨 기준으로 메시지 문구만 맞춘다.

- [ ] **Step 2: 빌드(타입체크) 통과 확인**

Run: `cd web/frontend && npm run build`
Expected: svelte-check 0 errors/0 warnings, vite build 성공. (validation.ts는 아직 미사용 — import 에러 없어야 함.)

- [ ] **Step 3: 커밋**

```bash
git add web/frontend/src/lib/validation.ts
git commit -m "feat(frontend): pure field validation functions"
```

---

### Task 2: QR 카운터 상태 강화 (QrThumbnails)

**Files:**
- Modify: `web/frontend/src/components/QrThumbnails.svelte:41`
- Modify: `web/frontend/src/styles/style.css` (.qr-counter 상태)

**Interfaces:**
- Consumes: 기존 `count`, `docCount`, `counterClass`(ok/under/over) 반응형 변수.

- [ ] **Step 1: 카운터 마크업에 상태 텍스트 추가**

`QrThumbnails.svelte`의 41행 `<div class="qr-counter {counterClass}">{count} / {docCount}</div>`를 교체:
```svelte
<div class="qr-counter {counterClass}">
  <span class="qr-counter-num">{count} / {docCount}</span>
  {#if count < docCount}
    <span class="qr-counter-hint">{docCount - count}장 더 필요</span>
  {:else if count > docCount}
    <span class="qr-counter-hint">{count - docCount}장 초과</span>
  {:else}
    <span class="qr-counter-hint">준비됨 ✓</span>
  {/if}
</div>
```

- [ ] **Step 2: CSS 상태 스타일 추가/보강**

`web/frontend/src/styles/style.css`에 추가(기존 `.qr-counter` 규칙이 있으면 그 아래에):
```css
.qr-counter { display: flex; align-items: center; gap: 8px; font-weight: 700; }
.qr-counter-hint { font-size: 0.85rem; font-weight: 600; }
.qr-counter.under { color: var(--text-muted); }
.qr-counter.ok    { color: #16a34a; }
.qr-counter.over  { color: #dc2626; }
```

- [ ] **Step 3: 빌드 + browse 확인**

Run: `cd web/frontend && npm run build && cd ../.. && export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH" && make build`
그리고:
```bash
pkill -x qrweb 2>/dev/null; PORT=5090 ./bin/qrweb >/tmp/ux.log 2>&1 &
sleep 2; B="$HOME/.claude/skills/gstack/browse/dist/browse"
$B goto http://localhost:5090/
# QR 0/1 카운터가 'under'(회색)+"1장 더 필요" 표시 확인
$B js "document.querySelector('.qr-counter')?.textContent"
pkill -x qrweb
```
Expected: 빌드 0/0, 카운터에 "0 / 1 1장 더 필요" 류 텍스트.

- [ ] **Step 4: 커밋**

```bash
git add web/frontend/src/components/QrThumbnails.svelte web/frontend/src/styles/style.css
git commit -m "feat(frontend): QR counter shows state hint (under/ok/over)"
```

---

### Task 3: 필드 인라인 에러 (Equipment + Project)

**Files:**
- Modify: `web/frontend/src/components/EquipmentFields.svelte`
- Modify: `web/frontend/src/components/ProjectFields.svelte`
- Modify: `web/frontend/src/styles/style.css` (.field-error, input.invalid, section-title ✓)

**Interfaces:**
- Consumes: `FieldErrors` (Task 1).
- Produces: 두 컴포넌트가 props `errors: Record<string,string>`, `showAll: boolean`를 받음. blur 시 touched 기록, `touched[field] || showAll`일 때 에러 표시.

- [ ] **Step 1: EquipmentFields에 에러 표시 추가**

`web/frontend/src/components/EquipmentFields.svelte` 전체 교체:
```svelte
<script lang="ts">
  import type { EquipmentForm } from '../lib/types';
  import { FileText } from 'lucide-svelte';

  export let data: EquipmentForm;
  export let errors: Record<string, string> = {};
  export let showAll = false;

  let touched: Record<string, boolean> = {};
  const blur = (f: string) => (touched = { ...touched, [f]: true });
  const show = (f: string) => (touched[f] || showAll) && errors[f];

  $: valid = Object.keys(errors).length === 0;
</script>

<div class="form-section">
  <div class="section-title">
    <FileText size={20} /> 기기 문서 정보 {#if valid}<span class="section-check">✓</span>{/if}
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="eq_number">마스터코드</label>
      <input type="text" id="eq_number" bind:value={data.eq_number} placeholder="마스터코드를 입력하세요"
        class:invalid={show('eq_number')} on:blur={() => blur('eq_number')} />
      {#if show('eq_number')}<span class="field-error">{errors.eq_number}</span>{/if}
    </div>
    <div class="form-group">
      <label for="eq_doc_number">문서 번호</label>
      <input type="text" id="eq_doc_number" bind:value={data.eq_doc_number} placeholder="문서 번호를 입력하세요"
        class:invalid={show('eq_doc_number')} on:blur={() => blur('eq_doc_number')} />
      {#if show('eq_doc_number')}<span class="field-error">{errors.eq_doc_number}</span>{/if}
    </div>
  </div>
  <div class="form-group">
    <label for="eq_doc_title">문서 제목</label>
    <input type="text" id="eq_doc_title" bind:value={data.eq_doc_title} placeholder="문서 제목을 입력하세요"
      class:invalid={show('eq_doc_title')} on:blur={() => blur('eq_doc_title')} />
    {#if show('eq_doc_title')}<span class="field-error">{errors.eq_doc_title}</span>{/if}
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="eq_doc_count">총 권수</label>
      <input type="number" id="eq_doc_count" bind:value={data.eq_doc_count} min="1"
        class:invalid={show('eq_doc_count')} on:blur={() => blur('eq_doc_count')} />
      {#if show('eq_doc_count')}<span class="field-error">{errors.eq_doc_count}</span>{/if}
    </div>
    <div class="form-group">
      <label for="eq_doc_department">작성 부서</label>
      <input type="text" id="eq_doc_department" bind:value={data.eq_doc_department} placeholder="작성 부서를 입력하세요"
        class:invalid={show('eq_doc_department')} on:blur={() => blur('eq_doc_department')} />
      {#if show('eq_doc_department')}<span class="field-error">{errors.eq_doc_department}</span>{/if}
    </div>
    <div class="form-group">
      <label for="eq_doc_year">연도</label>
      <input type="number" id="eq_doc_year" bind:value={data.eq_doc_year} min="1900" max="2100"
        class:invalid={show('eq_doc_year')} on:blur={() => blur('eq_doc_year')} />
      {#if show('eq_doc_year')}<span class="field-error">{errors.eq_doc_year}</span>{/if}
    </div>
  </div>
</div>
```

- [ ] **Step 2: ProjectFields에 동일 패턴 적용**

`web/frontend/src/components/ProjectFields.svelte`를 같은 패턴으로 수정: `errors`/`showAll` props 추가, 각 입력에 `class:invalid={show(field)}` + `on:blur={() => blur(field)}` + 아래 `{#if show(field)}<span class="field-error">{errors[field]}</span>{/if}`, 섹션 타이틀에 `{#if valid}<span class="section-check">✓</span>{/if}`. 대상 필드: `pjt_number`,`pjt_test_number`,`pjt_doc_title`,`pjt_doc_writer`,`pjt_doc_count`. (script 블록의 touched/blur/show/valid 로직은 EquipmentFields와 동일하게 복사.)

- [ ] **Step 3: CSS 추가**

`web/frontend/src/styles/style.css`에 추가:
```css
.field-error { display: block; margin-top: 4px; font-size: 0.8rem; color: var(--error-text); font-weight: 600; }
input.invalid { border-color: var(--error-text) !important; }
input.invalid:focus { box-shadow: 0 0 0 3px rgba(225, 29, 72, 0.15) !important; }
.section-check { color: var(--success-text); font-weight: 800; margin-left: 6px; }
```

- [ ] **Step 4: 빌드 확인**

Run: `cd web/frontend && npm run build`
Expected: svelte-check 0 errors/0 warnings. (App이 아직 errors/showAll를 안 넘겨도 기본값으로 컴파일됨.)

- [ ] **Step 5: 커밋**

```bash
git add web/frontend/src/components/EquipmentFields.svelte web/frontend/src/components/ProjectFields.svelte web/frontend/src/styles/style.css
git commit -m "feat(frontend): inline field errors with blur/submit touched display"
```

---

### Task 4: 준비상태 패널 (ReadinessPanel.svelte)

**Files:**
- Create: `web/frontend/src/components/ReadinessPanel.svelte`
- Modify: `web/frontend/src/styles/style.css` (.readiness-panel 등)

**Interfaces:**
- Produces: props `fieldErrorCount: number`, `qrCount: number`, `docCount: number`, `isReady: boolean`, `loading: boolean`, `onReset: () => void`. 내부에 `type="submit"` 버튼(상위 `<form>`이 제출 처리) + `type="button"` 초기화 버튼.

- [ ] **Step 1: ReadinessPanel 작성**

`web/frontend/src/components/ReadinessPanel.svelte`:
```svelte
<script lang="ts">
  import { Printer, Loader2, RotateCcw } from 'lucide-svelte';

  export let fieldErrorCount: number;
  export let qrCount: number;
  export let docCount: number;
  export let isReady: boolean;
  export let loading: boolean;
  export let onReset: () => void;

  $: docOk = fieldErrorCount === 0;
  $: qrOk = qrCount === docCount;
</script>

<div class="readiness-panel">
  <ul class="readiness-checklist">
    <li class:ok={docOk}>
      <span class="rc-mark">{docOk ? '✓' : '✕'}</span> 문서 정보{#if !docOk} — 미입력 {fieldErrorCount}칸{/if}
    </li>
    <li class:ok={qrOk}>
      <span class="rc-mark">{qrOk ? '✓' : '✕'}</span> QR 이미지 {qrCount} / {docCount}
    </li>
  </ul>
  <div class="readiness-actions">
    <button type="button" class="reset-btn" on:click={onReset} disabled={loading}>
      <RotateCcw size={18} /> 초기화
    </button>
    <button type="submit" class="submit-btn" disabled={!isReady || loading}>
      {#if loading}
        <Loader2 size={20} class="submit-icon spin" /> 생성 중...
      {:else}
        <Printer size={20} class="submit-icon" /> 라벨 만들기
      {/if}
    </button>
  </div>
</div>
```

- [ ] **Step 2: CSS 추가**

`web/frontend/src/styles/style.css`에 추가:
```css
.readiness-panel { margin-top: 16px; padding: 16px; border: 1px solid var(--border-color); border-radius: 12px; background: var(--surface-color); }
.readiness-checklist { list-style: none; margin: 0 0 12px; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.readiness-checklist li { font-weight: 600; color: var(--error-text); }
.readiness-checklist li.ok { color: var(--success-text); }
.readiness-checklist .rc-mark { font-weight: 800; margin-right: 6px; }
.readiness-actions { display: flex; gap: 12px; align-items: stretch; }
.readiness-actions .submit-btn { flex: 1; }
.reset-btn { display: inline-flex; align-items: center; gap: 6px; padding: 0 16px; border: 1px solid var(--border-color); border-radius: 10px; background: transparent; color: var(--text-muted); font-weight: 600; cursor: pointer; }
.reset-btn:hover:not(:disabled) { background: var(--border-color); }
.reset-btn:disabled { opacity: 0.5; cursor: not-allowed; }
```

- [ ] **Step 3: 빌드 확인**

Run: `cd web/frontend && npm run build`
Expected: svelte-check 0 errors/0 warnings. (컴포넌트는 아직 미사용 — import 에러 없어야 함.)

- [ ] **Step 4: 커밋**

```bash
git add web/frontend/src/components/ReadinessPanel.svelte web/frontend/src/styles/style.css
git commit -m "feat(frontend): readiness panel component (checklist + reset + submit)"
```

---

### Task 5: App 통합 배선

**Files:**
- Modify: `web/frontend/src/App.svelte`

**Interfaces:**
- Consumes: `validateEquipment`/`validateProject` (Task 1), `EquipmentFields`/`ProjectFields`의 `errors`/`showAll` props (Task 3), `ReadinessPanel` (Task 4), 기존 `clearItems`(이미 import됨).

- [ ] **Step 1: script 블록 배선**

`App.svelte` `<script>`에서:
1. import 추가:
   ```ts
   import { validateEquipment, validateProject } from './lib/validation';
   import ReadinessPanel from './components/ReadinessPanel.svelte';
   import { onMount } from 'svelte';
   ```
2. `validateForm()` 함수(75–103행) **삭제**.
3. 상태/파생 추가(기존 `docCount` 파생 근처):
   ```ts
   let submitAttempted = false;
   $: errors = docType === '1' ? validateEquipment(equipment) : validateProject(project);
   $: fieldErrorCount = Object.keys(errors).length;
   $: qrCount = $qrItems.length;
   $: isReady = fieldErrorCount === 0 && qrCount === docCount;
   ```
4. `handleSubmit` 교체:
   ```ts
   async function handleSubmit(e: Event) {
     e.preventDefault();
     if (loading) return;
     if (!isReady) { submitAttempted = true; return; }
     const insertion = $qrItems;
     const form: LabelForm = { docType, binderSize, equipment, project };
     loading = true;
     try {
       await submitLabel(form, insertion, displayItems);
       showSuccess('라벨이 성공적으로 생성되어 다운로드되었습니다!');
     } catch (err) {
       showError(err instanceof Error ? err.message : '서버 오류가 발생했습니다.');
     } finally {
       loading = false;
     }
   }
   function resetForm() {
     equipment = { eq_number: '', eq_doc_number: '', eq_doc_title: '', eq_doc_count: 1, eq_doc_department: '', eq_doc_year: currentYear };
     project = { pjt_number: '', pjt_test_number: '', pjt_doc_title: '', pjt_doc_writer: '', pjt_doc_count: 1 };
     clearItems();
     submitAttempted = false;
   }
   onMount(() => { (document.getElementById('eq_number') as HTMLInputElement | null)?.focus(); });
   ```

- [ ] **Step 2: 마크업 배선**

`App.svelte` 템플릿에서:
1. 필드 컴포넌트에 props 전달:
   ```svelte
   {#if docType === '1'}
     <EquipmentFields bind:data={equipment} {errors} showAll={submitAttempted} />
   {:else}
     <ProjectFields bind:data={project} {errors} showAll={submitAttempted} />
   {/if}
   ```
2. QR 섹션 타이틀에 ✓ 추가:
   ```svelte
   <div class="section-title"><QrCode size={20} /> QR 이미지 {#if qrCount === docCount}<span class="section-check">✓</span>{/if}</div>
   ```
3. 기존 `<button type="submit" class="submit-btn" ...>...</button>` 블록(184–190행)을 ReadinessPanel로 교체:
   ```svelte
   <ReadinessPanel {fieldErrorCount} {qrCount} {docCount} {isReady} {loading} onReset={resetForm} />
   ```
   (ReadinessPanel은 `<form>` 안에 위치 — submit 버튼이 폼을 제출.)
4. 사용하지 않게 된 import 정리: `Printer`, `Loader2`는 이제 ReadinessPanel에서 import하므로 App의 lucide import에서 제거(App에서 다른 곳에 안 쓰면). 남은 하단 `{#if loading}<div class="loading">...` 스피너 블록은 유지(중복이면 제거 가능하나 동작 무해 — 유지).

- [ ] **Step 3: 빌드 + 미사용 import 점검**

Run: `cd web/frontend && npm run build`
Expected: svelte-check 0 errors/0 warnings. (미사용 import 있으면 svelte-check가 경고 → 제거.)

- [ ] **Step 4: 커밋**

```bash
git add web/frontend/src/App.svelte
git commit -m "feat(frontend): wire inline validation, readiness panel, reset, autofocus"
```

---

### Task 6: 통합 browse QA

**Files:** 없음(검증만). 코드 결함 발견 시 해당 Task 파일 수정 후 커밋.

- [ ] **Step 1: 빌드 + 기동**

```bash
cd /home/spuhaha18/Project/QR_Web && export PATH="$HOME/.local/go/bin:$HOME/go/bin:$PATH"
make build
pkill -x qrweb 2>/dev/null; PORT=5090 ./bin/qrweb >/tmp/ux.log 2>&1 &
sleep 2
```

- [ ] **Step 2: QA 시나리오 7종 (browse)**

```bash
B="$HOME/.claude/skills/gstack/browse/dist/browse"
$B goto http://localhost:5090/
```
확인(스냅샷/js/스크린샷으로):
1. 빈 폼: 제출 버튼 `disabled` (`$B js "document.querySelector('.submit-btn').disabled"` → true).
2. 필드 blur(빈 채로): 해당 입력 아래 `.field-error` 표시 + `input.invalid`.
3. 모든 필수 필드 채움: `.section-check ✓`, 에러 사라짐.
4. QR 업로드(파일 1개, 권수 1): `.qr-counter.ok` + "준비됨 ✓", QR 섹션 ✓.
5. 준비 완료: 제출 버튼 활성(disabled=false), readiness 체크리스트 둘 다 ok.
6. 초기화 클릭: 필드 비워지고 QR 제거, 제출 다시 비활, 에러 미표시(submitAttempted 리셋).
7. 제출: .xlsx 다운로드(네트워크 `POST /create_label → 200`), 성공 토스트.

각 단계 `$B snapshot -i` / `$B js "..."` / `$B screenshot`으로 증거 수집. 스크린샷은 Read로 확인.

- [ ] **Step 3: 다크모드 가독성**

```bash
$B click @e1   # Toggle Dark Mode (snapshot로 ref 확인)
$B screenshot /tmp/ux_dark.png
pkill -x qrweb
```
Read `/tmp/ux_dark.png`: invalid 빨강/준비패널/카운터 색이 다크 배경서 가독.

- [ ] **Step 4: 전체 빌드 회귀 + 정리**

```bash
cd web/frontend && npm run build && cd ../..
go test -count=1 ./... 2>&1 | grep -E 'ok|FAIL'   # 백엔드 회귀(변경 없어 green이어야)
rm -f /tmp/ux.log /tmp/ux_dark.png
```
Expected: svelte-check 0/0, 백엔드 전 패키지 green.

- [ ] **Step 5: (결함 수정 시) 커밋**

QA에서 고친 게 있으면 커밋. 없으면 생략.

---

## Self-Review 메모
- **Spec 커버리지**: ①인라인검증→Task1+3+5, ②QR카운터→Task2, ③준비패널/흐름→Task4+5(섹션✓·포커스·비활), ④초기화→Task4+5. browse QA→Task6. 전부 매핑.
- **타입 일관성**: `FieldErrors`(Task1) ↔ Fields `errors: Record<string,string>`(Task3) ↔ App `errors`/`fieldErrorCount`(Task5) ↔ ReadinessPanel props(Task4) 일치. `showAll`(Fields) ← `submitAttempted`(App). `isReady`/`qrCount`/`docCount` 일관.
- **플레이스홀더**: 없음. ProjectFields는 패턴 동일이라 필드 목록 명시 + EquipmentFields 코드 참조(같은 파일 패턴). 검증 게이트는 vitest 없이 build+browse로 명시(spec YAGNI 준수).
- **주의**: Task5에서 미사용 lucide import(Printer/Loader2) 제거 — svelte-check 경고 0 유지.
