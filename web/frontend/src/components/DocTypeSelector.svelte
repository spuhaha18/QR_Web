<script lang="ts">
  import type { DocType } from '../lib/types';
  import { Wrench, Microscope } from 'lucide-svelte';

  export let value: DocType;
</script>

<div class="form-group">
  <span class="field-label">문서 종류</span>
  <div class="option-group doc-type-group">
    <button
      type="button"
      class="option-button"
      class:active={value === '1'}
      on:click={() => (value = '1')}
    >
      <span class="option-icon"><Wrench size={20} /></span>
      <span>기기 문서</span>
    </button>
    <button
      type="button"
      class="option-button"
      class:active={value === '2'}
      on:click={() => (value = '2')}
    >
      <span class="option-icon"><Microscope size={20} /></span>
      <span>과제 문서</span>
    </button>
  </div>
</div>
