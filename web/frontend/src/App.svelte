<script lang="ts">
  import type { DocType, BinderSize, EquipmentForm, ProjectForm, LabelForm } from './lib/types';
  import { qrItems, clearItems, type QrItem } from './lib/qrStore';
  import { showSuccess, showError } from './lib/toast';
  import { submitLabel } from './lib/api';

  import Toast from './components/Toast.svelte';
  import DocTypeSelector from './components/DocTypeSelector.svelte';
  import BinderSizeSelector from './components/BinderSizeSelector.svelte';
  import EquipmentFields from './components/EquipmentFields.svelte';
  import ProjectFields from './components/ProjectFields.svelte';
  import QrDropzone from './components/QrDropzone.svelte';
  import QrThumbnails from './components/QrThumbnails.svelte';
  import ManualModal from './components/ManualModal.svelte';

  import {
    Settings2,
    QrCode,
    Printer,
    Loader2,
    Moon,
    Sun,
    ClipboardList,
    FileJson2,
    BookOpen,
    Info,
    User,
    Phone,
    Mail,
  } from 'lucide-svelte';

  // ── 상태 ──────────────────────────────────────────────
  let docType: DocType = '1';
  let binderSize: BinderSize = 3;

  const currentYear = new Date().getFullYear();
  let equipment: EquipmentForm = {
    eq_number: '',
    eq_doc_number: '',
    eq_doc_title: '',
    eq_doc_count: 1,
    eq_doc_department: '',
    eq_doc_year: currentYear,
  };
  let project: ProjectForm = {
    pjt_number: '',
    pjt_test_number: '',
    pjt_doc_title: '',
    pjt_doc_writer: '',
    pjt_doc_count: 1,
  };

  let displayItems: QrItem[] = [];
  let loading = false;
  let manualOpen = false;

  // 다크모드 토글
  let theme: 'light' | 'dark' =
    (document.documentElement.getAttribute('data-theme') as 'light' | 'dark') || 'light';
  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
  }

  // 과제 문서로 전환 시 binderSize==1이면 3으로 리셋 (updateFormFields 동작)
  $: if (docType === '2' && binderSize === 1) {
    binderSize = 3;
  }

  // 권수 (현재 문서 종류 기준)
  $: docCount =
    docType === '1' ? Number(equipment.eq_doc_count) || 1 : Number(project.pjt_doc_count) || 1;

  function validateForm(): boolean {
    if (docType === '1') {
      const e = equipment;
      if (
        !e.eq_number.trim() ||
        !e.eq_doc_number.trim() ||
        !e.eq_doc_title.trim() ||
        !e.eq_doc_department.trim() ||
        !String(e.eq_doc_count).trim() ||
        !String(e.eq_doc_year).trim()
      ) {
        showError('모든 필드를 채워주세요.');
        return false;
      }
    } else {
      const p = project;
      if (
        !p.pjt_number.trim() ||
        !p.pjt_test_number.trim() ||
        !p.pjt_doc_title.trim() ||
        !p.pjt_doc_writer.trim() ||
        !String(p.pjt_doc_count).trim()
      ) {
        showError('모든 필드를 채워주세요.');
        return false;
      }
    }
    return true;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (loading) return;
    if (!validateForm()) return;

    const insertion = $qrItems;
    const n = docCount;
    if (insertion.length !== n) {
      showError(`QR 이미지 수(${insertion.length})가 권수(${n})와 다릅니다.`);
      return;
    }

    const form: LabelForm = { docType, binderSize, equipment, project };

    loading = true;
    try {
      await submitLabel(form, insertion, displayItems);
      showSuccess('라벨이 성공적으로 생성되어 다운로드되었습니다!');
    } catch (err) {
      showError(err instanceof Error ? err.message : '서버 오류가 발생했습니다.');
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>연구소 라벨 제작 프로그램</title>
</svelte:head>

<div class="container">
  <button class="theme-toggle" on:click={toggleTheme} aria-label="Toggle Dark Mode">
    {#if theme === 'dark'}<Sun size={20} />{:else}<Moon size={20} />{/if}
  </button>

  <div class="header-container">
    <div class="company-logo">
      <span class="logo-text">inno.N</span>
    </div>
    <h1>연구소 바인더 라벨 제작 프로그램</h1>
  </div>

  <div class="header-links">
    <a href="/logs" class="header-link">
      <ClipboardList size={18} /> 시스템 로그
    </a>
    <a href="/api/docs" class="header-link">
      <FileJson2 size={18} /> API 문서
    </a>
    <button type="button" class="header-link" on:click={() => (manualOpen = true)}>
      <BookOpen size={18} /> 사용 설명서
    </button>
  </div>

  <form on:submit={handleSubmit} id="label-form">
    <div class="form-sections">
      <div class="form-section">
        <div class="section-title"><Settings2 size={20} /> 기본 설정</div>
        <DocTypeSelector bind:value={docType} />
        <BinderSizeSelector bind:value={binderSize} {docType} />
      </div>

      {#if docType === '1'}
        <EquipmentFields bind:data={equipment} />
      {:else}
        <ProjectFields bind:data={project} />
      {/if}

      <div class="form-section" id="qr_section">
        <div class="section-title"><QrCode size={20} /> QR 이미지</div>
        <p class="section-description">
          바인더 권수만큼 QR 이미지를 추가하세요. 추가한 이미지는 드래그로 순서를 바꿀 수 있으며, 캡션의 권 번호가 인쇄 순서가 됩니다.
        </p>

        <QrDropzone />
        <QrThumbnails bind:displayItems {docCount} />
      </div>
    </div>

    <button type="submit" class="submit-btn" disabled={loading}>
      {#if loading}
        <Loader2 size={20} class="submit-icon spin" /> 생성 중...
      {:else}
        <Printer size={20} class="submit-icon" /> 라벨 만들기
      {/if}
    </button>
  </form>

  {#if loading}
    <div class="loading" style="display:block;">
      <div class="spinner"></div>
      <p style="color: var(--text-muted); font-weight: 600;">라벨을 생성하는 중...</p>
    </div>
  {/if}

  <div class="contact-info">
    <div class="contact-title"><Info size={18} /> 담당자 정보</div>
    <div class="contact-details">
      <div class="contact-item">
        <div class="contact-icon"><User size={18} /></div>
        <div class="contact-text-wrap">
          <span class="contact-label">담당자</span>
          <span class="contact-value">R&D QA팀 박진기님</span>
        </div>
      </div>
      <div class="contact-item">
        <div class="contact-icon"><Phone size={18} /></div>
        <div class="contact-text-wrap">
          <span class="contact-label">전화번호</span>
          <a href="tel:031-5176-4600" class="contact-value contact-link">031-5176-4600</a>
        </div>
      </div>
      <div class="contact-item">
        <div class="contact-icon"><Mail size={18} /></div>
        <div class="contact-text-wrap">
          <span class="contact-label">이메일</span>
          <a href="mailto:jinki.park@inno-n.com" class="contact-value contact-link">jinki.park@inno-n.com</a>
        </div>
      </div>
    </div>
  </div>
</div>

<ManualModal open={manualOpen} onClose={() => (manualOpen = false)} />
<Toast />
