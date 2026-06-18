<script lang="ts">
  import { addFiles, addDataUri } from '../lib/qrStore';
  import { FolderOpen, Upload, Link } from 'lucide-svelte';

  let dragover = false;
  let fileInput: HTMLInputElement;
  let dataUri = '';

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    dragover = true;
  }
  function onDragLeave(e: DragEvent) {
    const related = e.relatedTarget as Node | null;
    if (!related || !(e.currentTarget as HTMLElement).contains(related)) {
      dragover = false;
    }
  }
  async function onDrop(e: DragEvent) {
    e.preventDefault();
    dragover = false;
    if (e.dataTransfer?.files) await addFiles(e.dataTransfer.files);
  }
  function onZoneClick(e: MouseEvent) {
    if ((e.target as HTMLElement).closest('input, button')) return;
    fileInput.click();
  }
  function onZoneKey(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      fileInput.click();
    }
  }
  async function onFileChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) await addFiles(input.files);
    input.value = '';
  }
  async function submitDataUri() {
    const val = dataUri.trim();
    if (!val) return;
    await addDataUri(val);
    dataUri = '';
  }
  function onDataUriKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      submitDataUri();
    }
  }
</script>

<div class="qr-input-group">
  <span class="qr-input-label"><Upload size={16} /> 파일 업로드</span>
  <div
    class="qr-dropzone"
    class:dragover
    role="button"
    tabindex="0"
    on:dragover={onDragOver}
    on:dragleave={onDragLeave}
    on:drop={onDrop}
    on:click={onZoneClick}
    on:keydown={onZoneKey}
  >
    <FolderOpen size={28} />
    <p>여기를 클릭하거나 파일을 끌어다 놓으세요</p>
    <p class="qr-hint">PNG · JPG 등 이미지 파일 (여러 개 선택 가능)</p>
    <input
      type="file"
      accept="image/*"
      multiple
      hidden
      bind:this={fileInput}
      on:change={onFileChange}
    />
  </div>
</div>

<div class="qr-divider"><span>또는</span></div>

<div class="qr-input-group">
  <label class="qr-input-label" for="qr_data_uri_input">
    <Link size={16} /> 이미지 데이터 URI 붙여넣기
  </label>
  <div class="qr-data-uri-row">
    <input
      type="text"
      id="qr_data_uri_input"
      class="qr-data-uri-input"
      placeholder="data:image/... 형식의 URI를 붙여넣고 Enter"
      bind:value={dataUri}
      on:keydown={onDataUriKey}
    />
    <button type="button" class="qr-data-uri-btn" on:click={submitDataUri}>추가</button>
  </div>
</div>
