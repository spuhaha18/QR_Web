import { writable, get } from 'svelte/store';
import { showError } from './toast';

export interface QrItem {
  id: string;
  blob: Blob;
  hash: string;
  url: string;
}

// QR images in *insertion order*. Display order is owned by the thumbnails
// component (svelte-dnd-action); qr_order permutation maps display→insertion.
export const qrItems = writable<QrItem[]>([]);

let nextId = 0;

/**
 * Duplicate-detection fingerprint, ported verbatim from qr_paste.js.
 * djb2-style hash over byte length + first 256 + last 256 bytes.
 * Kept synchronous (no crypto.subtle) to preserve original UX timing.
 */
export function fingerprint(arrayBuffer: ArrayBuffer): string {
  const bytes = new Uint8Array(arrayBuffer);
  const len = bytes.length;
  let h = len;
  const sample = Math.min(256, len);
  for (let i = 0; i < sample; i++) h = (Math.imul(h, 31) + bytes[i]) >>> 0;
  for (let i = Math.max(0, len - sample); i < len; i++) h = (Math.imul(h, 31) + bytes[i]) >>> 0;
  return `${len}_${h.toString(16)}`;
}

/** Add a blob with duplicate detection. Returns true if added. */
export async function addBlob(blob: Blob): Promise<boolean> {
  const buf = await blob.arrayBuffer();
  const hash = fingerprint(buf);
  const current = get(qrItems);
  if (current.some((img) => img.hash === hash)) {
    showError('중복된 QR 이미지입니다.');
    return false;
  }
  const url = URL.createObjectURL(blob);
  const id = `qr_${nextId++}`;
  qrItems.update((list) => [...list, { id, blob, hash, url }]);
  return true;
}

/** Add image files, skipping non-image entries (original addFromFiles behavior). */
export async function addFiles(files: FileList | File[]): Promise<void> {
  let skipped = 0;
  for (const file of Array.from(files)) {
    if (!file.type.startsWith('image/')) {
      skipped++;
      continue;
    }
    await addBlob(file);
  }
  if (skipped > 0) showError(`이미지가 아닌 파일 ${skipped}개는 건너뜁니다.`);
}

/** Parse a data: URI into a Blob (ported from qr_paste.js dataUriToBlob). */
export function dataUriToBlob(dataUri: string): Blob | null {
  const parts = dataUri.trim().split(',');
  if (parts.length < 2) return null;
  const header = parts[0];
  const base64 = parts[1];
  const mimeMatch = header.match(/data:([^;]+)/);
  if (!mimeMatch) return null;
  try {
    const binary = atob(base64);
    const arr = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) arr[i] = binary.charCodeAt(i);
    return new Blob([arr], { type: mimeMatch[1] });
  } catch {
    return null;
  }
}

/** Add an image from a data: URI string (ported from addFromDataUri). */
export async function addDataUri(raw: string): Promise<void> {
  if (!raw.startsWith('data:image/')) {
    showError('data:image/... 형식의 URI만 지원합니다.');
    return;
  }
  const blob = dataUriToBlob(raw);
  if (!blob) {
    showError('유효하지 않은 data URI입니다.');
    return;
  }
  await addBlob(blob);
}

/** Remove an item by id and release its object URL. */
export function removeItem(id: string): void {
  qrItems.update((list) => {
    const item = list.find((i) => i.id === id);
    if (item) URL.revokeObjectURL(item.url);
    return list.filter((i) => i.id !== id);
  });
}

/** Clear all items, releasing object URLs. */
export function clearItems(): void {
  qrItems.update((list) => {
    list.forEach((i) => URL.revokeObjectURL(i.url));
    return [];
  });
}
