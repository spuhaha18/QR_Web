<script lang="ts">
  import type { EquipmentForm } from '../lib/types';
  import { FileText } from 'lucide-svelte';

  export let data: EquipmentForm;
  export let errors: Record<string, string> = {};
  export let showAll = false;

  let touched: Record<string, boolean> = {};
  const blur = (f: string) => (touched = { ...touched, [f]: true });
  const show = (f: string) => (touched[f] || showAll) && errors[f];

  $: valid = Object.keys(errors).length === 0;
</script>

<div class="form-section">
  <div class="section-title">
    <FileText size={20} /> 기기 문서 정보 {#if valid}<span class="section-check">✓</span>{/if}
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="eq_number">마스터코드</label>
      <input type="text" id="eq_number" bind:value={data.eq_number} placeholder="마스터코드를 입력하세요"
        class:invalid={show('eq_number')} on:blur={() => blur('eq_number')} />
      {#if show('eq_number')}<span class="field-error">{errors.eq_number}</span>{/if}
    </div>
    <div class="form-group">
      <label for="eq_doc_number">문서 번호</label>
      <input type="text" id="eq_doc_number" bind:value={data.eq_doc_number} placeholder="문서 번호를 입력하세요"
        class:invalid={show('eq_doc_number')} on:blur={() => blur('eq_doc_number')} />
      {#if show('eq_doc_number')}<span class="field-error">{errors.eq_doc_number}</span>{/if}
    </div>
  </div>
  <div class="form-group">
    <label for="eq_doc_title">문서 제목</label>
    <input type="text" id="eq_doc_title" bind:value={data.eq_doc_title} placeholder="문서 제목을 입력하세요"
      class:invalid={show('eq_doc_title')} on:blur={() => blur('eq_doc_title')} />
    {#if show('eq_doc_title')}<span class="field-error">{errors.eq_doc_title}</span>{/if}
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="eq_doc_count">총 권수</label>
      <input type="number" id="eq_doc_count" bind:value={data.eq_doc_count} min="1"
        class:invalid={show('eq_doc_count')} on:blur={() => blur('eq_doc_count')} />
      {#if show('eq_doc_count')}<span class="field-error">{errors.eq_doc_count}</span>{/if}
    </div>
    <div class="form-group">
      <label for="eq_doc_department">작성 부서</label>
      <input type="text" id="eq_doc_department" bind:value={data.eq_doc_department} placeholder="작성 부서를 입력하세요"
        class:invalid={show('eq_doc_department')} on:blur={() => blur('eq_doc_department')} />
      {#if show('eq_doc_department')}<span class="field-error">{errors.eq_doc_department}</span>{/if}
    </div>
    <div class="form-group">
      <label for="eq_doc_year">연도</label>
      <input type="number" id="eq_doc_year" bind:value={data.eq_doc_year} min="1900" max="2100"
        class:invalid={show('eq_doc_year')} on:blur={() => blur('eq_doc_year')} />
      {#if show('eq_doc_year')}<span class="field-error">{errors.eq_doc_year}</span>{/if}
    </div>
  </div>
</div>
