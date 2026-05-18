// qr_paste.js — paste-mode QR 이미지 수집 + SortableJS 순서 관리 + FormData fetch 제출

(function () {
  'use strict';

  /** @type {{ id: number, blob: Blob, hash: string, url: string }[]} */
  const state = { images: [], nextId: 0 };

  const dropzone = document.getElementById('qr_dropzone');
  const thumbnailList = document.getElementById('qr_thumbnails');
  const counterEl = document.getElementById('qr_counter');
  const orderInput = document.getElementById('qr_order');
  const form = document.getElementById('label-form');

  // ── SHA-1 해시 (Web Crypto API) ─────────────────────────────────────────
  async function sha1Hex(arrayBuffer) {
    const hashBuf = await crypto.subtle.digest('SHA-1', arrayBuffer);
    return Array.from(new Uint8Array(hashBuf))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  }

  // ── 카운터 갱신 ──────────────────────────────────────────────────────────
  function getDocCount() {
    const docType = document.getElementById('doc_type').value;
    const key = docType === '1' ? 'eq_doc_count' : 'pjt_doc_count';
    return parseInt(document.getElementById(key)?.value || '1', 10);
  }

  function updateCounter() {
    const n = getDocCount();
    const m = state.images.length;
    counterEl.textContent = `${m} / ${n}`;
    counterEl.className = 'qr-counter' + (m === n ? ' ok' : m > n ? ' over' : ' under');
  }

  // ── 썸네일 DOM 동기화 ─────────────────────────────────────────────────────
  function renderThumbnails() {
    thumbnailList.innerHTML = '';
    state.images.forEach(({ id, url }) => {
      const li = document.createElement('li');
      li.className = 'qr-thumb-item';
      li.dataset.id = id;
      li.innerHTML = `
        <img src="${url}" alt="QR ${id}" />
        <button type="button" class="qr-remove-btn" data-id="${id}">×</button>
      `;
      thumbnailList.appendChild(li);
    });
    syncOrder();
    updateCounter();
  }

  // ── qr_order hidden input 동기화 ─────────────────────────────────────────
  function syncOrder() {
    const domIds = [...thumbnailList.querySelectorAll('li[data-id]')]
      .map(el => parseInt(el.dataset.id, 10));
    // DOM 순서 → 원본 state 배열의 인덱스로 변환
    const order = domIds.map(id => state.images.findIndex(img => img.id === id));
    orderInput.value = JSON.stringify(order);
  }

  // ── SortableJS 초기화 ────────────────────────────────────────────────────
  if (window.Sortable) {
    Sortable.create(thumbnailList, {
      animation: 150,
      onEnd: syncOrder,
    });
  }

  // ── 삭제 버튼 이벤트 위임 ────────────────────────────────────────────────
  thumbnailList.addEventListener('click', (e) => {
    const btn = e.target.closest('.qr-remove-btn');
    if (!btn) return;
    const id = parseInt(btn.dataset.id, 10);
    const img = state.images.find(i => i.id === id);
    if (img) URL.revokeObjectURL(img.url);
    state.images = state.images.filter(i => i.id !== id);
    renderThumbnails();
  });

  // ── Blob → state.images 추가 헬퍼 ──────────────────────────────────────
  async function addFromBlob(blob) {
    const arrayBuffer = await blob.arrayBuffer();
    const hash = await sha1Hex(arrayBuffer);
    if (state.images.some(img => img.hash === hash)) {
      showToast('중복된 QR 이미지입니다.');
      return;
    }
    const url = URL.createObjectURL(blob);
    state.images.push({ id: state.nextId++, blob, hash, url });
    renderThumbnails();
  }

  // ── 파일 처리 헬퍼 ────────────────────────────────────────────────────
  async function addFromFiles(files) {
    for (const file of files) {
      if (!file.type.startsWith('image/')) {
        showToast('이미지 파일만 추가할 수 있습니다.');
        continue;
      }
      await addFromBlob(file);
    }
  }

  // ── dropzone drag 이벤트 ─────────────────────────────────────────────
  dropzone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropzone.classList.add('dragover');
  });
  dropzone.addEventListener('dragleave', () => {
    dropzone.classList.remove('dragover');
  });
  dropzone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropzone.classList.remove('dragover');
    addFromFiles(e.dataTransfer.files);
  });

  // ── dropzone 클릭 → 파일 선택 다이얼로그 ────────────────────────────
  const fileInput = document.getElementById('qr_file_input');
  dropzone.addEventListener('click', (e) => {
    if (e.target.closest('input, button')) return;
    fileInput.click();
  });
  fileInput.addEventListener('change', (e) => {
    addFromFiles(e.target.files);
    e.target.value = '';
  });

  // ── data URI 입력 처리 ──────────────────────────────────────────────────
  function dataUriToBlob(dataUri) {
    const parts = dataUri.trim().split(',');
    if (parts.length < 2) return null;
    const header = parts[0];
    const base64 = parts[1];
    const mimeMatch = header.match(/data:([^;]+)/);
    if (!mimeMatch) return null;
    try {
      const binary = atob(base64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      return new Blob([bytes], { type: mimeMatch[1] });
    } catch {
      return null;
    }
  }

  async function addFromDataUri(raw) {
    if (!raw.startsWith('data:image/')) {
      showToast('data:image/... 형식의 URI만 지원합니다.');
      return;
    }
    const blob = dataUriToBlob(raw);
    if (!blob) { showToast('유효하지 않은 data URI입니다.'); return; }
    await addFromBlob(blob);
  }

  const dataUriInput = document.getElementById('qr_data_uri_input');
  const dataUriBtn = document.getElementById('qr_data_uri_btn');

  async function handleDataUriSubmit() {
    const val = dataUriInput.value.trim();
    if (!val) return;
    await addFromDataUri(val);
    dataUriInput.value = '';
  }

  dataUriBtn.addEventListener('click', handleDataUriSubmit);
  dataUriInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { e.preventDefault(); handleDataUriSubmit(); }
  });

  // doc_type 변경 시 카운터 갱신 (index.html의 selectDocType 호출 후)
  document.querySelectorAll('[data-value][onclick*="selectDocType"]').forEach(btn => {
    btn.addEventListener('click', () => setTimeout(updateCounter, 0));
  });
  document.querySelectorAll('#eq_doc_count, #pjt_doc_count').forEach(input => {
    input.addEventListener('input', updateCounter);
  });

  // ── 폼 submit 가로채기 ────────────────────────────────────────────────────
  form.addEventListener('submit', async (e) => {
    e.preventDefault();

    const n = getDocCount();
    if (state.images.length !== n) {
      alert(`QR 이미지 수(${state.images.length})가 권수(${n})와 다릅니다. 맞춰주세요.`);
      return;
    }

    // qr_order를 현재 DOM 순서 기준 인덱스 배열로 최종 갱신
    syncOrder();

    const formData = new FormData(form);
    // paste된 Blob을 File로 변환하여 추가
    state.images.forEach((img, i) => {
      formData.append('qr_images', new File([img.blob], `qr_${i}.png`, { type: 'image/png' }));
    });

    const submitBtn = form.querySelector('.submit-btn');
    if (submitBtn) { submitBtn.disabled = true; submitBtn.textContent = '생성 중...'; }

    try {
      const resp = await fetch('/create_label', { method: 'POST', body: formData });
      if (!resp.ok) {
        const err = await resp.json().catch(() => ({ error: '서버 오류가 발생했습니다.' }));
        alert(err.error || '서버 오류가 발생했습니다.');
        return;
      }
      // 엑셀 다운로드 트리거
      const disposition = resp.headers.get('Content-Disposition') || '';
      const match = disposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
      const filename = match ? match[1].replace(/['"]/g, '') : '라벨.xlsx';
      const blob = await resp.blob();
      const a = document.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(a.href);
    } catch (err) {
      alert('요청 중 오류가 발생했습니다: ' + err.message);
    } finally {
      if (submitBtn) { submitBtn.disabled = false; submitBtn.textContent = '라벨 만들기'; }
    }
  });

  // ── 토스트 메시지 ─────────────────────────────────────────────────────────
  function showToast(msg) {
    const t = document.createElement('div');
    t.className = 'qr-toast';
    t.textContent = msg;
    document.body.appendChild(t);
    setTimeout(() => t.remove(), 2500);
  }

  // 초기 카운터
  updateCounter();
})();
