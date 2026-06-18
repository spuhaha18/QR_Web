<script lang="ts">
  import type { BinderSize, DocType } from '../lib/types';
  import { Ruler, BookOpen, Book, Library } from 'lucide-svelte';

  export let value: BinderSize;
  export let docType: DocType;

  // 과제 문서(doc_type=2)는 3cm 미만(1) 바인더를 숨긴다.
  $: hide1cm = docType === '2';
</script>

<div class="form-group">
  <span class="field-label">바인더 크기</span>
  <div class="option-group binder-size-group">
    {#if !hide1cm}
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
</div>
