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
