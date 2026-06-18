# F — Frontend (Vite + Svelte SPA)

svelte-frontend-engineer 산출물. 소스 `web/frontend/`, 빌드 출력 `web/dist/` (Go embed 대상).
선행: `_workspace/E_api_contract.md` 계약 준수. 동작 오라클: `templates/index.html` + `static/js/qr_paste.js` + `static/css/style.css`.

## 빌드 결과
- `cd web/frontend && npm install` → 성공 (81 packages).
- `npm run build` → `svelte-check 0 errors / 0 warnings` + `vite build` 성공.
- 출력: `web/dist/index.html` (상대 base `./assets/...`) + `assets/`(JS/CSS 해시 번들 + 매뉴얼 스크린샷 PNG 5종).
- Svelte 4 + vite 5 + TS strict. lucide-svelte(아이콘), svelte-dnd-action(재정렬).

## 컴포넌트 트리
```
App.svelte                  # 상태 소유: docType, binderSize, equipment, project, displayItems, theme, loading, manualOpen
  DocTypeSelector.svelte    # 기기('1')/과제('2') 버튼. bind:value
  BinderSizeSelector.svelte # 1/3/5/7. docType==='2'면 1cm(=1) 버튼 숨김(미렌더)
  EquipmentFields.svelte    # 기기 필드 bind:data (eq_doc_year 기본=현재년)
  ProjectFields.svelte      # 과제 필드 bind:data
  QrDropzone.svelte         # 드래그드롭 + 클릭→파일선택 + data-URI 입력(Enter/추가버튼)
  QrThumbnails.svelte       # svelte-dnd-action 재정렬, "{i+1}권" 캡션, × 삭제, "{m}/{n}" 카운터(ok/over/under)
  ManualModal.svelte        # 매뉴얼 5섹션 정적 내장(스크린샷 src/assets/manual/*.png 번들), 목차 스크롤, ESC/백드롭 닫기
  Toast.svelte              # 성공/에러 토스트 (toast 스토어 구독)
lib/qrStore.ts              # qrItems writable<QrItem[]> (삽입 순서), djb2 fingerprint 중복제거
lib/api.ts                  # submitLabel — FormData → POST /create_label → .xlsx blob 다운로드
lib/toast.ts                # toasts 스토어 + showSuccess/showError (4s 표시 + 0.3s slide-out)
lib/types.ts                # DocType/BinderSize/EquipmentForm/ProjectForm/LabelForm
```

## 상태 전환 (updateFormFields 포팅)
- `docType==='2' && binderSize===1` → reactive로 `binderSize=3` 리셋. BinderSizeSelector가 1cm 버튼 미렌더(숨김).
- docType 변경 시 EquipmentFields ↔ ProjectFields `{#if}` 전환.
- `docCount` = docType 기준 eq_doc_count 또는 pjt_doc_count (카운터/제출 검증에 사용).

## QR 수집 (qr_paste.js 포팅)
- **fingerprint**: djb2 변형(`len + 앞256 + 뒤256` 바이트, `Math.imul(h,31)+byte`) → `"{len}_{hex}"`. 원본과 동일 문자열. 동기(crypto.subtle 미사용).
- **중복**: 같은 hash 존재 시 추가 거부 + `중복된 QR 이미지입니다.` 토스트.
- **dataUriToBlob / addDataUri**: 원본 그대로. `data:image/` 접두 아니면 거부.
- **addFiles**: 비이미지 스킵 + `이미지가 아닌 파일 {n}개는 건너뜁니다.` 토스트.
- 삭제 시 `URL.revokeObjectURL`.

## dnd 순열 계약 구현 (★ parity-qa 핵심 경계면)
원본 Flask `syncOrder` 로직과 **동일**:
- **삽입 순서** = `qrStore.qrItems` 순서. 파일을 이 순서대로 `qr_images` 키로 N개 append, 이름 `qr_{i}.png`.
- **표시 순서** = `QrThumbnails`의 `displayItems`(dnd 결과). App에 `bind:displayItems`로 전달.
- `qr_order` = `displayItems.map(it => insertionItems.findIndex(x => x.id === it.id))`.
  즉 **표시 위치 i번째 항목의 삽입 인덱스**를 담은 `[0..n)` 순열. (원본: `domIds.map(id => state.images.findIndex(...))` 와 동일 의미.)
- 백엔드는 삽입 순서 파일을 `qr_order`로 재정렬 → 핸들러 로직 유지.
- 재정렬 안 하면 `qr_order = [0,1,...,n-1]` (identity).

## API 계약 사용 방식 (E_api_contract.md 대조)
### POST /create_label (multipart, paste 모드) — 유일하게 사용하는 제출 경로
FormData 키 (E 계약 §요청과 1:1):
- `doc_type`: `'1'|'2'`
- `binder_size`: `String(1|3|5|7)`
- 기기: `eq_number, eq_doc_number, eq_doc_title, eq_doc_count, eq_doc_department, eq_doc_year` (숫자는 String 변환)
- 과제: `pjt_number, pjt_test_number, pjt_doc_title, pjt_doc_writer, pjt_doc_count`
- `qr_order`: `JSON.stringify(순열)`
- `qr_images`: N개 File(`image/png`, `qr_{i}.png`), 삽입 순서

응답 처리 (E 계약 §응답):
- `res.ok` false → `await res.json()` 의 `{error}` → throw → 에러 토스트(한국어 메시지 그대로 노출).
- 성공 → `Content-Disposition` 정규식 `filename[^;=\n]*=((['"]).*?\2|[^;\n]*)`로 파일명 추출(따옴표 제거, 기본 `라벨.xlsx`).
- `res.blob()` → object URL → `<a download>` 클릭 → revoke. 성공 토스트.

### 프론트 자체 검증 (서버 검증 이전, UX)
- 빈 필드 → `모든 필드를 채워주세요.` (서버엔 안 보냄)
- `qrItems.length !== docCount` → `QR 이미지 수({m})가 권수({n})와 다릅니다.` (서버 전송 차단)
- 그 외(2MB/PNG/순열 검증)는 서버 위임 → 400 `{error}` 토스트로 표시.

### 미사용 (이 화면에서 호출 안 함)
- `/api/create_label`(auto/json), `/api/qr_image*`, `/api/health`, `/api/logs*` — 현 폼은 paste 모드 전용.
  (로그 뷰어/auto 모드는 Phase 추후 SPA 라우트.)

## 보존된 기존 기능
- 다크모드: `data-theme` 속성 + localStorage. index.html 인라인 스크립트가 paint 전 적용(FOUC 방지). 토글 버튼 Moon/Sun 전환.
- 반응형: style.css `@media (max-width:768px)` 전부 포팅.
- 토스트: 4s 표시 + 0.3s slide-out, success/error. 원본 애니메이션 CSS 그대로.
- 매뉴얼 모달: `/manual` 엔드포인트 없음 → manual.html 내용 ManualModal에 정적 내장. 스크린샷은 Vite 번들. 목차 스무스 스크롤, ESC/백드롭 닫기, body scroll-lock.
- 담당자 정보, 헤더 링크(시스템 로그 `/logs`, API 문서 `/api/docs`, 사용 설명서 버튼).

## CSS 포팅
- `static/css/style.css`(1150줄) → `src/styles/style.css` 전량 포팅(Pristine Lab + Deep Blue, 다크모드 변수, 매뉴얼/QR/토스트 스타일).
- 변경점:
  - `.option-group:has([onclick*="selectDocType"])` → `.option-group.doc-type-group`, `selectBinderSize` → `.binder-size-group` (onclick 속성 제거에 따른 셀렉터 교체. 반응형 포함).
  - `.option-button` div→button 전환 위해 `font-family/font-size/width` 리셋 추가.
  - `label` 규칙에 `.field-label` 추가(a11y: 컨트롤 없는 라벨을 span으로).
  - `.spinner`/`@keyframes spin`/`.submit-icon.spin` 추가(원본 index.html이 참조했으나 CSS 미정의였던 로딩 스피너 구현).
  - `.manual-backdrop` 추가(모달 백드롭을 button으로, a11y).
- `data-light/data-dark` 스크린샷 다크 스왑은 원본에 다크 이미지 자산이 없어 단일 이미지 사용(원본도 동일 src였음).

## 막힌 점 / 주의
- 없음. 빌드/타입체크 0/0.
- lucide-svelte 0.453.0 deprecated 경고(설치만, 동작 영향 없음) — 추후 `@lucide/svelte`로 교체 가능.
- E2E(Go 임베드 서빙 후 실제 제출)는 Phase 6. 여기선 빌드 성공 + 구조 완전성까지.

## parity-qa 확인 요청 (경계면)
1. `qr_order` 순열 정의가 백엔드 재정렬 로직과 일치하는지(표시위치→삽입인덱스). 위 §dnd 계약 참조.
2. `qr_images` 파일이 **삽입 순서**로 도착하는지(이름 `qr_{i}.png`는 식별용일 뿐, 순서는 append 순).
3. 에러 응답 `{error}` 한국어 문자열이 토스트에 그대로 노출되는지(E 계약 §검증 순서 2~12).
4. Content-Disposition 파일명 파싱(`filename="{doc_number}_{ts}.xlsx"`).
