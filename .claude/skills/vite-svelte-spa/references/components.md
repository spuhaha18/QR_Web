# Svelte 컴포넌트 상세 (qr_paste.js + index.html 동작 매핑)

## 상태 (App.svelte)
```ts
let docType: '1' | '2' = '1';
let binderSize: 1 | 3 | 5 | 7 = 3;
let equipment = { eq_number:'', eq_doc_number:'', eq_doc_title:'', eq_doc_count:1, eq_doc_department:'', eq_doc_year: new Date().getFullYear() };
let project = { pjt_number:'', pjt_test_number:'', pjt_doc_title:'', pjt_doc_writer:'', pjt_doc_count:1 };
```
`updateFormFields` 로직: docType 변경 시 표시 필드 전환. 과제로 가면 binderSize==1이면 3으로, 1cm 버튼 숨김.

## qrStore (lib/qrStore.ts)
```ts
import { writable } from 'svelte/store';
export type QrItem = { id: string; blob: Blob; hash: number; url: string };
export const qrItems = writable<QrItem[]>([]);
```
- 추가 시 djb2 hash로 중복 검사. `url = URL.createObjectURL(blob)`(해제 관리).

## djb2 핑거프린트 (qr_paste.js fingerprint 포팅)
```ts
function fingerprint(bytes: Uint8Array): number {
  let h = 5381;
  for (let i = 0; i < bytes.length; i++) h = ((h << 5) + h + bytes[i]) >>> 0;
  return h;
}
```
blob → `arrayBuffer()` → Uint8Array → fingerprint. 같은 hash 존재 시 추가 거부 + 토스트.

## QrDropzone.svelte
- `on:dragover|preventDefault`, `on:drop` → `e.dataTransfer.files` 처리.
- `<input type="file" accept="image/*" multiple>` 숨기고 클릭 위임.
- data-URI 입력: textarea + 버튼/Enter → `dataUriToBlob`:
```ts
function dataUriToBlob(uri: string): Blob {
  const [meta, b64] = uri.split(',');
  const mime = meta.match(/data:(.*?);base64/)?.[1] ?? 'image/png';
  const bin = atob(b64); const arr = new Uint8Array(bin.length);
  for (let i=0;i<bin.length;i++) arr[i]=bin.charCodeAt(i);
  return new Blob([arr], { type: mime });
}
```

## QrThumbnails.svelte (svelte-dnd-action)
```svelte
<script>
  import { dndzone } from 'svelte-dnd-action';
  export let items; // QrItem[]
  function handleSort(e){ items = e.detail.items; }
</script>
<section use:dndzone={{ items }} on:consider={handleSort} on:finalize={handleSort}>
  {#each items as it (it.id)}
    <div class="thumb"><img src={it.url} alt=""/><button on:click={()=>remove(it.id)}>×</button></div>
  {/each}
</section>
```
캡션 "{index+1}권". 리스트 순서 = 제출 순서.

## 제출 (lib/api.ts)
```ts
export async function submitLabel(form, items): Promise<void> {
  const fd = new FormData();
  fd.append('doc_type', form.docType);
  fd.append('binder_size', String(form.binderSize));
  // ...필드들
  fd.append('qr_order', JSON.stringify(items.map((_,i)=>i))); // 재정렬 계약 참조
  items.forEach(it => fd.append('qr_images', it.blob, `${it.id}.png`));
  const res = await fetch('/create_label', { method:'POST', body: fd });
  if (!res.ok) { const {error} = await res.json(); throw new Error(error); }
  const blob = await res.blob();
  const name = res.headers.get('Content-Disposition')?.match(/filename="?(.+?)"?$/)?.[1] ?? 'label.xlsx';
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob); a.download = name; a.click();
}
```
에러 시 Toast로 한국어 메시지 표시.
