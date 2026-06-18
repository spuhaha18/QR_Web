<script lang="ts">
  import type { ProjectForm } from '../lib/types';
  import { FlaskConical } from 'lucide-svelte';

  export let data: ProjectForm;
</script>

<div class="form-section">
  <div class="section-title">
    <FlaskConical size={20} /> 과제 문서 정보
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="pjt_number">과제 코드</label>
      <input type="text" id="pjt_number" bind:value={data.pjt_number} placeholder="과제 코드를 입력하세요" />
    </div>
    <div class="form-group">
      <label for="pjt_test_number">시험 번호</label>
      <input type="text" id="pjt_test_number" bind:value={data.pjt_test_number} placeholder="시험 번호를 입력하세요" />
    </div>
  </div>
  <div class="form-group">
    <label for="pjt_doc_title">문서 제목</label>
    <input type="text" id="pjt_doc_title" bind:value={data.pjt_doc_title} placeholder="문서 제목을 입력하세요" />
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="pjt_doc_writer">연구 담당자</label>
      <input type="text" id="pjt_doc_writer" bind:value={data.pjt_doc_writer} placeholder="연구 담당자를 입력하세요" />
    </div>
    <div class="form-group">
      <label for="pjt_doc_count">총 권수</label>
      <input type="number" id="pjt_doc_count" bind:value={data.pjt_doc_count} min="1" />
    </div>
  </div>
</div>
