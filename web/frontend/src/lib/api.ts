import type { LabelForm } from './types';
import type { QrItem } from './qrStore';

/**
 * Submit a paste-mode label request to POST /create_label.
 *
 * Reorder contract (matches Flask/Go backend, E_api_contract.md):
 *   - Files are appended in INSERTION order (qrItems insertion order),
 *     each under the `qr_images` key, named qr_{i}.png.
 *   - `qr_order` is a JSON permutation array of length doc_count. The i-th
 *     entry is the INSERTION index of the image shown at DISPLAY position i.
 *     The backend reorders insertion-order files by this permutation.
 *
 * On success the server streams a .pdf binary; we read the
 * Content-Disposition filename and trigger a download via <a download>.
 *
 * @param insertionItems QR items in insertion order (qrStore order).
 * @param displayItems   QR items in display order (dnd order).
 */
export async function submitLabel(
  form: LabelForm,
  insertionItems: QrItem[],
  displayItems: QrItem[],
): Promise<void> {
  const fd = new FormData();
  fd.append('doc_type', form.docType);
  fd.append('binder_size', String(form.binderSize));

  if (form.docType === '1') {
    const e = form.equipment;
    fd.append('eq_number', e.eq_number);
    fd.append('eq_doc_number', e.eq_doc_number);
    fd.append('eq_doc_title', e.eq_doc_title);
    fd.append('eq_doc_count', String(e.eq_doc_count));
    fd.append('eq_doc_department', e.eq_doc_department);
    fd.append('eq_doc_year', String(e.eq_doc_year));
  } else {
    const p = form.project;
    fd.append('pjt_number', p.pjt_number);
    fd.append('pjt_test_number', p.pjt_test_number);
    fd.append('pjt_doc_title', p.pjt_doc_title);
    fd.append('pjt_doc_writer', p.pjt_doc_writer);
    fd.append('pjt_doc_count', String(p.pjt_doc_count));
  }

  // qr_order: display position i -> insertion index of that item.
  const order = displayItems.map((it) => insertionItems.findIndex((x) => x.id === it.id));
  fd.append('qr_order', JSON.stringify(order));

  // Files in insertion order.
  insertionItems.forEach((it, i) => {
    fd.append('qr_images', new File([it.blob], `qr_${i}.png`, { type: 'image/png' }));
  });

  const resp = await fetch('/create_label', { method: 'POST', body: fd });

  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: '서버 오류가 발생했습니다.' }));
    throw new Error(err.error || '서버 오류가 발생했습니다.');
  }

  const disposition = resp.headers.get('Content-Disposition') || '';
  const filename = parseFilename(disposition);

  const blob = await resp.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(a.href);
}

/**
 * Extract the download filename from a Content-Disposition header.
 * Prefers RFC 5987 `filename*=UTF-8''<percent-encoded>` (correct for non-ASCII
 * names like Korean) and falls back to the ASCII `filename="..."` token.
 */
function parseFilename(disposition: string): string {
  const star = disposition.match(/filename\*\s*=\s*UTF-8''([^;]+)/i);
  if (star) {
    try {
      return decodeURIComponent(star[1].trim());
    } catch {
      /* fall through to plain filename */
    }
  }
  const plain = disposition.match(/filename\s*=\s*"?([^";]+)"?/i);
  if (plain) {
    return plain[1].trim();
  }
  return '라벨.pdf';
}
