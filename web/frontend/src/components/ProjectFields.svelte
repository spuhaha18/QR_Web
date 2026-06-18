<script lang="ts">
  import type { ProjectForm } from '../lib/types';
  import { REQUIRED_PROJECT_FIELDS as FIELDS } from '../lib/domain';
  import { FlaskConical } from 'lucide-svelte';

  export let data: ProjectForm;
  export let errors: Record<string, string> = {};
  export let showAll = false;
  let touched: Record<string, boolean> = {};
  const blur = (f: string) => (touched = { ...touched, [f]: true });
  $: vis = Object.fromEntries(
    FIELDS.map((f) => [f, (touched[f] || showAll) && errors[f] ? errors[f] : ''])
  ) as Record<string, string>;

  $: valid = Object.keys(errors).length === 0;
</script>

<div class="form-section">
  <div class="section-title">
    <FlaskConical size={20} /> 과제 문서 정보 {#if valid}<span class="section-check">✓</span>{/if}
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="pjt_number">과제 코드</label>
      <input type="text" id="pjt_number" bind:value={data.pjt_number} placeholder="과제 코드를 입력하세요"
        class:invalid={vis.pjt_number} on:blur={() => blur('pjt_number')} />
      {#if vis.pjt_number}<span class="field-error">{vis.pjt_number}</span>{/if}
    </div>
    <div class="form-group">
      <label for="pjt_test_number">시험 번호</label>
      <input type="text" id="pjt_test_number" bind:value={data.pjt_test_number} placeholder="시험 번호를 입력하세요"
        class:invalid={vis.pjt_test_number} on:blur={() => blur('pjt_test_number')} />
      {#if vis.pjt_test_number}<span class="field-error">{vis.pjt_test_number}</span>{/if}
    </div>
  </div>
  <div class="form-group">
    <label for="pjt_doc_title">문서 제목</label>
    <input type="text" id="pjt_doc_title" bind:value={data.pjt_doc_title} placeholder="문서 제목을 입력하세요"
      class:invalid={vis.pjt_doc_title} on:blur={() => blur('pjt_doc_title')} />
    {#if vis.pjt_doc_title}<span class="field-error">{vis.pjt_doc_title}</span>{/if}
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="pjt_doc_writer">연구 담당자</label>
      <input type="text" id="pjt_doc_writer" bind:value={data.pjt_doc_writer} placeholder="연구 담당자를 입력하세요"
        class:invalid={vis.pjt_doc_writer} on:blur={() => blur('pjt_doc_writer')} />
      {#if vis.pjt_doc_writer}<span class="field-error">{vis.pjt_doc_writer}</span>{/if}
    </div>
    <div class="form-group">
      <label for="pjt_doc_count">총 권수</label>
      <input type="number" id="pjt_doc_count" bind:value={data.pjt_doc_count} min="1"
        class:invalid={vis.pjt_doc_count} on:blur={() => blur('pjt_doc_count')} />
      {#if vis.pjt_doc_count}<span class="field-error">{vis.pjt_doc_count}</span>{/if}
    </div>
  </div>
</div>
