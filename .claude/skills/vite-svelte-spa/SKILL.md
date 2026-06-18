---
name: vite-svelte-spa
description: Vite+Svelte SPA를 구축할 때 사용. 현재 서버렌더 폼(index.html)+바닐라 JS(qr_paste.js)+CSS를 Svelte 컴포넌트로 재작성, 드래그드롭/data-URI 붙여넣기/svelte-dnd-action 재정렬, API 클라이언트(FormData→.xlsx 다운로드), vite build→Go embed.FS 연동을 다룬다. Svelte 컴포넌트, 프론트 폼, QR 드롭존, 빌드 임베드 작업 시 반드시 사용.
---

# Vite + Svelte SPA 구축

현재 `templates/index.html`(483) + `static/js/qr_paste.js`(242) + `static/css/style.css`(1150)를 Svelte SPA로 재작성. 빌드 산출물 `web/dist`를 Go `embed.FS`가 서빙.

## 선행 조건
**go-backend-engineer의 `_workspace/E_api_contract.md` 확정 후 착수.** FormData 키·응답 형식을 가정하지 말 것.

## 컴포넌트 트리
```
App.svelte                  # 상태 소유: docType, binderSize, 필드, theme
  DocTypeSelector.svelte    # 기기/과제 버튼. 과제 선택 시 binder==1이면 3으로 리셋+1cm 숨김
  BinderSizeSelector.svelte # 1/3/5/7. 과제일 때 1cm 숨김
  EquipmentFields.svelte    # 기기 필드 바인딩 (year 기본=현재년)
  ProjectFields.svelte      # 과제 필드 바인딩
  QrDropzone.svelte         # 드래그드롭 + 파일선택 + data-URI 입력
  QrThumbnails.svelte       # svelte-dnd-action 재정렬 리스트 ("{n}권" 캡션, 삭제 ×)
  Toast.svelte              # 성공/에러 토스트
  (매뉴얼 모달, 연락처)
```
store `qrStore`: `{ id, blob, hash, url }[]` (현 `state.images` 대응).

## 인터랙션 포팅 (qr_paste.js 동작 보존)
- **드롭존**: dragover/drop, click→`<input type=file accept="image/*" multiple>`, data-URI 텍스트 입력 + Enter/버튼. `addFromFiles`/`dataUriToBlob`/`addFromDataUri` 대응.
- **중복 제거**: djb2 핑거프린트 그대로 포팅(`fingerprint()`) — 동기·UX 일치. (crypto.subtle SHA-256은 async라 지양.)
- **재정렬**: `svelte-dnd-action`(SortableJS 대체). 리스트 순서가 제출 순서 결정.
- **제출**: FormData(모든 필드 + doc_type + binder_size + qr_order(JSON) + N개 qr_images) → `POST /create_label` → 응답 `.xlsx` blob → Content-Disposition 파일명 → object URL + `<a download>` 클릭 다운로드.

## 재정렬 계약 (한 번 정하고 문서화)
파일은 **삽입 순서**로 보내고 `qr_order` 순열 배열 전송(현 Flask 로직과 동일). 백엔드가 qr_order로 재정렬하므로 핸들러 로직 유지. (대안: 표시순서로 보내고 identity order — 단 백엔드와 합의 필요.)

## 빌드 + 임베드 (references/build-embed.md)
- `vite.config.ts`: `build.outDir='../dist'`(=web/dist), `base:'./'`(상대 경로), `emptyOutDir:true`, `@sveltejs/vite-plugin-svelte`.
- dev: `server.proxy`로 `/create_label`,`/api`를 Go 백엔드(localhost:5000)에 프록시.
- Go: `//go:embed all:dist` → `fs.Sub`.

## 보존할 기존 기능
다크모드, 반응형, 토스트, 매뉴얼 모달(SPA 내 정적 컴포넌트로 이전 권장), 정렬/삭제 UX. CSS 1150줄 포팅(Pristine Lab 테마).

## 참조 파일
- `references/components.md` — 컴포넌트별 상세 + store 설계
- `references/build-embed.md` — vite 설정 + Go embed 연동 + dev 프록시
