<script lang="ts">
  import type { EquipmentForm } from '../lib/types';
  import { FileText } from 'lucide-svelte';

  export let data: EquipmentForm;
</script>

<div class="form-section">
  <div class="section-title">
    <FileText size={20} /> 기기 문서 정보
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="eq_number">마스터코드</label>
      <input type="text" id="eq_number" bind:value={data.eq_number} placeholder="마스터코드를 입력하세요" />
    </div>
    <div class="form-group">
      <label for="eq_doc_number">문서 번호</label>
      <input type="text" id="eq_doc_number" bind:value={data.eq_doc_number} placeholder="문서 번호를 입력하세요" />
    </div>
  </div>
  <div class="form-group">
    <label for="eq_doc_title">문서 제목</label>
    <input type="text" id="eq_doc_title" bind:value={data.eq_doc_title} placeholder="문서 제목을 입력하세요" />
  </div>
  <div class="form-row">
    <div class="form-group">
      <label for="eq_doc_count">총 권수</label>
      <input type="number" id="eq_doc_count" bind:value={data.eq_doc_count} min="1" />
    </div>
    <div class="form-group">
      <label for="eq_doc_department">작성 부서</label>
      <input type="text" id="eq_doc_department" bind:value={data.eq_doc_department} placeholder="작성 부서를 입력하세요" />
    </div>
    <div class="form-group">
      <label for="eq_doc_year">연도</label>
      <input type="number" id="eq_doc_year" bind:value={data.eq_doc_year} min="1900" max="2100" />
    </div>
  </div>
</div>
