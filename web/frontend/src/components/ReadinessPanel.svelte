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
