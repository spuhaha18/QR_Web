<script lang="ts">
  import type { BinderSize, DocType } from '../lib/types';
  import { allowedBinderSizes } from '../lib/domain';
  import { Ruler, BookOpen, Book, Library } from 'lucide-svelte';

  export let value: BinderSize | null;
  export let docType: DocType;
  export let invalid = false;

  // 허용 바인더 크기는 도메인 규칙(allowedBinderSizes)에서 — 과제는 1cm 제외.
  $: allowed = allowedBinderSizes(docType);
  $: show1cm = allowed.includes(1);
</script>

<div class="form-group">
  <span class="field-label">바인더 크기</span>
  <div class="option-group binder-size-group" class:invalid>
    {#if show1cm}
      <button
        type="button"
        class="option-button"
        class:active={value === 1}
        on:click={() => (value = 1)}
      >
        <span class="option-icon"><Ruler size={20} /></span>
        <span>3cm 미만</span>
      </button>
    {/if}
    <button
      type="button"
      class="option-button"
      class:active={value === 3}
      on:click={() => (value = 3)}
    >
      <span class="option-icon"><BookOpen size={20} /></span>
      <span>3cm</span>
    </button>
    <button
      type="button"
      class="option-button"
      class:active={value === 5}
      on:click={() => (value = 5)}
    >
      <span class="option-icon"><Book size={20} /></span>
      <span>5cm</span>
    </button>
    <button
      type="button"
      class="option-button"
      class:active={value === 7}
      on:click={() => (value = 7)}
    >
      <span class="option-icon"><Library size={20} /></span>
      <span>7cm</span>
    </button>
  </div>
  {#if invalid}<span class="field-error">바인더 크기를 선택하세요.</span>{/if}
</div>
