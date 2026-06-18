<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { BookOpen, X, Info, ChevronRight, AlertTriangle, Layers } from 'lucide-svelte';

  export let open = false;
  export let onClose: () => void;

  const sections = [
    { id: 'step-0', label: '시작하기' },
    { id: 'step-1', label: 'DSMS 문서 등록' },
    { id: 'step-2', label: 'QR 코드 복사' },
    { id: 'step-3', label: '라벨 정보 입력' },
    { id: 'step-4', label: '라벨 생성' },
  ];

  let activeId = 'step-0';
  let contentEl: HTMLDivElement;

  function scrollTo(id: string, e: MouseEvent) {
    e.preventDefault();
    activeId = id;
    const target = contentEl?.querySelector<HTMLElement>('#' + id);
    if (target && contentEl) {
      contentEl.scrollTo({ top: target.offsetTop - contentEl.offsetTop - 8, behavior: 'smooth' });
    }
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) onClose();
  }

  $: if (open) {
    document.body.style.overflow = 'hidden';
  } else {
    document.body.style.overflow = '';
  }

  onMount(() => document.addEventListener('keydown', onKey));
  onDestroy(() => {
    document.removeEventListener('keydown', onKey);
    document.body.style.overflow = '';
  });

  // Screenshots — bundled via Vite so they get hashed asset URLs.
  import dsmsRegister from '../assets/manual/dsms-register.png';
  import qrList from '../assets/manual/qr-list.png';
  import qrMenu from '../assets/manual/qr-menu.png';
  import appInput from '../assets/manual/app-input.png';
  import appCreate from '../assets/manual/app-create.png';
</script>

<div class="manual-modal" class:open role="dialog" aria-modal="true" aria-label="사용 설명서">
  <!-- Backdrop: clicking outside the card closes the modal. -->
  <button type="button" class="manual-backdrop" aria-label="닫기" on:click={onClose}></button>
  <div class="manual-card">
    <div class="manual-card-head">
      <span class="manual-card-title"><BookOpen size={20} /> 사용 설명서</span>
      <button type="button" class="manual-close" on:click={onClose} aria-label="닫기">
        <X size={18} />
      </button>
    </div>
    <div class="manual-body">
      <div class="manual">
        <nav class="manual-toc">
          <div class="manual-toc-title">목차</div>
          {#each sections as s, i}
            <a
              href={'#' + s.id}
              class:active={activeId === s.id}
              on:click={(e) => scrollTo(s.id, e)}
            >
              <span class="toc-num">
                {#if i === 0}<Info size={14} />{:else}{i}{/if}
              </span>
              {s.label}
            </a>
          {/each}
        </nav>

        <div class="manual-content" bind:this={contentEl}>
          <!-- 0. 시작하기 -->
          <section class="manual-section" id="step-0">
            <div class="manual-cover">
              <span class="manual-cover-kicker">R&amp;D QA</span>
              <h2 class="manual-cover-title">DSMS 문서 등록 및<br />바인더 라벨 생성 매뉴얼</h2>
              <p class="manual-cover-desc">
                DSMS에 문서를 등록하고, QR 코드를 복사해 라벨을 생성하기까지의 전체 과정을 안내합니다.
              </p>
              <ol class="manual-flow">
                <li><span class="flow-step">1</span><span>DSMS 문서 등록</span></li>
                <li><span class="flow-arrow"><ChevronRight size={16} /></span></li>
                <li><span class="flow-step">2</span><span>QR 코드 복사</span></li>
                <li><span class="flow-arrow"><ChevronRight size={16} /></span></li>
                <li><span class="flow-step">3</span><span>라벨 정보 입력</span></li>
                <li><span class="flow-arrow"><ChevronRight size={16} /></span></li>
                <li><span class="flow-step">4</span><span>라벨 생성</span></li>
              </ol>
            </div>
          </section>

          <!-- 1. DSMS 문서 등록 -->
          <section class="manual-section" id="step-1">
            <div class="manual-section-head">
              <span class="manual-section-num">1</span>
              <h3>DSMS 문서 등록</h3>
            </div>
            <p class="manual-lead">DSMS에 접속하여 라벨을 출력할 문서를 등록합니다.</p>

            <figure class="manual-figure">
              <img class="manual-shot" src={dsmsRegister} alt="DSMS 문서 등록 화면" />
              <span class="step-badge" style="top:3%;left:8.5%">1</span>
              <span class="step-badge" style="top:22%;left:93%">2</span>
              <span class="step-badge" style="top:37%;left:3%">3</span>
              <span class="step-badge" style="top:42%;left:29.5%">4</span>
              <span class="step-badge" style="top:55%;left:71%">5</span>
              <span class="step-badge" style="top:78%;left:3.5%">6</span>
              <span class="step-badge" style="top:84%;left:3.5%">7</span>
              <span class="step-badge" style="top:94%;left:93%">8</span>
            </figure>

            <table class="manual-table">
              <thead><tr><th>No.</th><th>Activity</th></tr></thead>
              <tbody>
                <tr><td><span class="row-badge">1</span></td><td>문서 관리 클릭</td></tr>
                <tr><td><span class="row-badge">2</span></td><td>등록 클릭</td></tr>
                <tr><td><span class="row-badge">3</span></td><td>그룹 – <strong>R&amp;D</strong> 선택</td></tr>
                <tr><td><span class="row-badge">4</span></td><td>문서유형 part1.을 수기번호 선택 후 시험번호 입력</td></tr>
                <tr><td><span class="row-badge">5</span></td><td>문서번호 생성 클릭</td></tr>
                <tr><td><span class="row-badge">6</span></td><td>작성일 선택</td></tr>
                <tr><td><span class="row-badge">7</span></td><td>승인일 선택</td></tr>
                <tr><td><span class="row-badge">8</span></td><td>저장</td></tr>
              </tbody>
            </table>

            <div class="manual-note">
              <AlertTriangle size={18} />
              <ul>
                <li><strong>바인더가 여러 개일 경우 각각 등록</strong>해주셔야 합니다.</li>
                <li>그룹을 반드시 <strong>R&amp;D</strong>로 선택해주세요.</li>
                <li>문서번호 생성 클릭 후 생성된 번호의 <strong>0001, 0002</strong>는 바인더 번호에 해당합니다.</li>
              </ul>
            </div>
          </section>

          <!-- 2. QR 코드 복사 -->
          <section class="manual-section" id="step-2">
            <div class="manual-section-head">
              <span class="manual-section-num">2</span>
              <h3>QR 코드 복사</h3>
            </div>
            <p class="manual-lead">등록한 문서의 QR 코드 이미지 링크를 복사합니다.</p>

            <div class="manual-figure-row">
              <figure class="manual-figure">
                <img class="manual-shot" src={qrList} alt="문서 목록에서 QR 보기" />
                <span class="step-badge" style="top:13%;left:72.5%">1</span>
              </figure>
              <figure class="manual-figure">
                <img class="manual-shot" src={qrMenu} alt="QR 우클릭 → 이미지 링크 복사" />
                <span class="step-badge" style="top:82%;left:72%">2</span>
              </figure>
            </div>

            <table class="manual-table">
              <thead><tr><th>No.</th><th>Activity</th></tr></thead>
              <tbody>
                <tr><td><span class="row-badge">1</span></td><td>DSMS에서 라벨 출력할 문서의 <strong>QR보기</strong> 클릭</td></tr>
                <tr><td><span class="row-badge">2</span></td><td>QR 이미지에 마우스 우클릭 후 <strong>‘이미지 링크 복사’</strong> 클릭</td></tr>
              </tbody>
            </table>
          </section>

          <!-- 3. 라벨 정보 입력 -->
          <section class="manual-section" id="step-3">
            <div class="manual-section-head">
              <span class="manual-section-num">3</span>
              <h3>라벨 정보 입력</h3>
            </div>
            <p class="manual-lead">
              <a href="https://label.inno-n.duckdns.org" target="_blank" rel="noopener">https://label.inno-n.duckdns.org</a>
              접속 후 아래 정보를 입력합니다.
            </p>

            <figure class="manual-figure">
              <img class="manual-shot" src={appInput} alt="라벨 정보 입력 화면" />
              <span class="step-badge" style="top:9%;left:7%">1</span>
              <span class="step-badge" style="top:26.5%;left:27%">2</span>
              <span class="step-badge" style="top:48%;left:23%">3</span>
            </figure>

            <table class="manual-table">
              <thead><tr><th>No.</th><th>Activity</th></tr></thead>
              <tbody>
                <tr><td><span class="row-badge">1</span></td><td><strong>문서 종류</strong> 선택 (기기 문서 / 과제 문서)</td></tr>
                <tr><td><span class="row-badge">2</span></td><td><strong>바인더 크기</strong> 선택 (3cm 미만 / 3cm / 5cm / 7cm)</td></tr>
                <tr><td><span class="row-badge">3</span></td><td>마스터코드 · 문서 번호 · 문서 제목 · 총 권수 · 작성 부서 · 연도 입력</td></tr>
              </tbody>
            </table>
          </section>

          <!-- 4. 라벨 생성 -->
          <section class="manual-section" id="step-4">
            <div class="manual-section-head">
              <span class="manual-section-num">4</span>
              <h3>라벨 생성</h3>
            </div>
            <p class="manual-lead">복사한 QR 이미지 링크를 붙여넣고 라벨을 생성합니다.</p>

            <figure class="manual-figure">
              <img class="manual-shot" src={appCreate} alt="라벨 생성 화면" />
              <span class="step-badge" style="top:8%;left:93%">1</span>
              <span class="step-badge" style="top:74%;left:53%">2</span>
            </figure>

            <table class="manual-table">
              <thead><tr><th>No.</th><th>Activity</th></tr></thead>
              <tbody>
                <tr><td><span class="row-badge">1</span></td><td>이미지 링크(데이터 URI)를 붙여넣고 <strong>‘추가’</strong> 클릭</td></tr>
                <tr><td><span class="row-badge">2</span></td><td><strong>‘라벨 만들기’</strong> 클릭</td></tr>
              </tbody>
            </table>

            <div class="manual-note">
              <Layers size={18} />
              <div>
                <strong>바인더가 2개 이상일 경우</strong>
                <ol>
                  <li>총 권수를 바인더 개수만큼 입력</li>
                  <li>이미지 링크를 바인더 수만큼 복사 / 붙여넣기 / 추가</li>
                  <li>‘라벨 만들기’를 하면 한꺼번에 생성됩니다.</li>
                </ol>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </div>
</div>
