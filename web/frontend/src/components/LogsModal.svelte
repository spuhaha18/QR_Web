<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { ClipboardList, X, RefreshCw, Download, Trash2 } from 'lucide-svelte';
  import { levelLabel, renderMessage, type LogEntry } from '../lib/logMessages';

  export let open = false;
  export let onClose: () => void;

  interface LogsResponse {
    success: boolean;
    logs: LogEntry[];
    total_lines: number;
    message?: string;
  }

  let entries: LogEntry[] = [];
  let totalLines = 0;
  let levelFilter = 'all';
  let searchQuery = '';
  let requestIdFilter = '';
  let showRaw = false;
  let loading = false;
  let errorMsg = '';

  const levelOptions = [
    { value: 'all', label: '전체' },
    { value: 'DEBUG', label: '디버그' },
    { value: 'INFO', label: '정보' },
    { value: 'WARN', label: '경고' },
    { value: 'ERROR', label: '오류' },
  ];

  async function fetchLogs() {
    loading = true;
    errorMsg = '';
    try {
      const params = new URLSearchParams({ lines: '200' });
      if (levelFilter !== 'all') params.set('level', levelFilter);
      if (requestIdFilter) params.set('request_id', requestIdFilter);
      const res = await fetch(`/api/logs?${params.toString()}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: LogsResponse = await res.json();
      entries = data.logs ?? [];
      totalLines = data.total_lines ?? 0;
      if (entries.length === 0 && data.message) errorMsg = data.message;
    } catch (e) {
      errorMsg = e instanceof Error ? e.message : '로그를 불러올 수 없습니다.';
      entries = [];
    } finally {
      loading = false;
    }
  }

  function onLevelChange() {
    fetchLogs();
  }

  // Client-side search: matches the rendered Korean text AND raw msg/fields,
  // so both "라벨" and "label" (or a filename) find the same line.
  function matches(e: LogEntry, q: string): boolean {
    const hay = (renderMessage(e) + ' ' + e.msg + ' ' + JSON.stringify(e.fields)).toLowerCase();
    return hay.includes(q.toLowerCase());
  }
  $: visible = searchQuery.trim()
    ? entries.filter((e) => matches(e, searchQuery.trim()))
    : entries;

  function filterByRequestId(id: string) {
    requestIdFilter = id;
    fetchLogs();
  }
  function clearRequestIdFilter() {
    requestIdFilter = '';
    fetchLogs();
  }

  function shortId(e: LogEntry): string {
    const id = e.fields['request_id'];
    return typeof id === 'string' ? id.slice(0, 8) : '';
  }
  function fullId(e: LogEntry): string {
    const id = e.fields['request_id'];
    return typeof id === 'string' ? id : '';
  }

  function fmtTime(iso: string): string {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d.getTime())) return iso;
    const p = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
  }

  function downloadLogs() {
    window.open('/api/logs/download', '_blank', 'noopener');
  }

  async function clearLogs() {
    if (!confirm('로그를 초기화하시겠습니까?')) return;
    try {
      const res = await fetch('/api/logs/clear', { method: 'POST' });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchLogs();
    } catch (e) {
      errorMsg = e instanceof Error ? e.message : '초기화 실패';
    }
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) onClose();
  }

  $: if (open) {
    document.body.style.overflow = 'hidden';
    fetchLogs();
  } else {
    document.body.style.overflow = '';
    entries = [];
    errorMsg = '';
    searchQuery = '';
    levelFilter = 'all';
    requestIdFilter = '';
    showRaw = false;
  }

  onMount(() => document.addEventListener('keydown', onKey));
  onDestroy(() => {
    document.removeEventListener('keydown', onKey);
    document.body.style.overflow = '';
  });
</script>

<div class="manual-modal logs-modal" class:open role="dialog" aria-modal="true" aria-label="시스템 로그">
  <button type="button" class="manual-backdrop" aria-label="닫기" on:click={onClose}></button>
  <div class="manual-card logs-card">
    <div class="manual-card-head">
      <span class="manual-card-title"><ClipboardList size={20} /> 시스템 로그</span>
      <div class="logs-head-actions">
        <button type="button" class="logs-action-btn" on:click={downloadLogs} title="로그 다운로드">
          <Download size={16} /> 다운로드
        </button>
        <button type="button" class="logs-action-btn logs-action-danger" on:click={clearLogs} title="로그 초기화">
          <Trash2 size={16} /> 초기화
        </button>
        <button type="button" class="logs-action-btn" on:click={fetchLogs} title="새로고침" disabled={loading}>
          <RefreshCw size={16} class={loading ? 'logs-spin' : ''} /> 새로고침
        </button>
        <button type="button" class="manual-close" on:click={onClose} aria-label="닫기">
          <X size={18} />
        </button>
      </div>
    </div>

    <div class="logs-toolbar">
      <select
        class="logs-level-select"
        bind:value={levelFilter}
        on:change={onLevelChange}
        aria-label="로그 레벨 필터"
      >
        {#each levelOptions as opt}
          <option value={opt.value}>{opt.label}</option>
        {/each}
      </select>
      <input
        class="logs-search-input"
        type="text"
        placeholder="검색"
        bind:value={searchQuery}
        aria-label="로그 검색"
      />
      <button type="button" class="logs-action-btn" class:logs-raw-on={showRaw} aria-pressed={showRaw} on:click={() => (showRaw = !showRaw)}>
        원문
      </button>
      {#if requestIdFilter}
        <button type="button" class="logs-reqid-chip" on:click={clearRequestIdFilter} title="요청 필터 해제">
          요청 {requestIdFilter.slice(0, 8)} ✕
        </button>
      {/if}
      {#if totalLines > 0}
        <span class="logs-count">{visible.length} / {totalLines}줄</span>
      {/if}
    </div>

    <div class="logs-body">
      {#if loading}
        <div class="logs-status-msg">불러오는 중...</div>
      {:else if errorMsg}
        <div class="logs-status-msg logs-empty">{errorMsg}</div>
      {:else if entries.length === 0}
        <div class="logs-status-msg logs-empty">로그 없음</div>
      {:else}
        <div class="logs-lines" role="log" aria-live="polite">
          {#each visible as e, i}
            <div class="logs-line" class:logs-line-error={e.level === 'ERROR'} class:logs-line-warn={e.level === 'WARN'}>
              <span class="logs-line-num">{i + 1}</span>
              <span class="logs-time">{fmtTime(e.time)}</span>
              <span class="logs-badge logs-badge-{e.level.toLowerCase()}">{levelLabel(e.level)}</span>
              {#if shortId(e)}
                <button type="button" class="logs-reqid" title={fullId(e)} on:click={() => filterByRequestId(fullId(e))}>
                  {shortId(e)}
                </button>
              {/if}
              <span class="logs-line-text">
                {#if showRaw}{JSON.stringify({ time: e.time, level: e.level, msg: e.msg, ...e.fields })}{:else}{renderMessage(e)}{/if}
              </span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .logs-card {
    max-width: 860px;
    height: 80vh;
  }

  .logs-head-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .logs-action-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 6px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-color);
    background: var(--secondary-bg);
    color: var(--text-muted);
    font-size: 0.85rem;
    font-weight: 600;
    font-family: inherit;
    cursor: pointer;
    transition: var(--transition);
  }

  .logs-action-btn:hover:not(:disabled) {
    background: var(--surface-color);
    color: var(--primary-color);
    border-color: var(--primary-color);
  }

  .logs-action-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .logs-action-danger:hover:not(:disabled) {
    color: var(--error-text);
    border-color: var(--error-text);
    background: var(--error-bg);
  }

  :global(.logs-spin) {
    animation: logsSpin 1s linear infinite;
  }

  @keyframes logsSpin {
    from { transform: rotate(0deg); }
    to   { transform: rotate(360deg); }
  }

  .logs-toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-color);
    flex-shrink: 0;
  }

  .logs-level-select {
    padding: 6px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-color);
    background: var(--surface-color);
    color: var(--text-main);
    font-size: 0.85rem;
    font-family: inherit;
    cursor: pointer;
  }

  .logs-search-input {
    flex: 1;
    padding: 6px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-color);
    background: var(--surface-color);
    color: var(--text-main);
    font-size: 0.85rem;
    font-family: inherit;
    outline: none;
    transition: var(--transition);
  }

  .logs-search-input:focus {
    border-color: var(--border-focus);
    box-shadow: 0 0 0 3px rgba(191, 219, 254, 0.3);
  }

  .logs-count {
    font-size: 0.8rem;
    color: var(--text-muted);
    white-space: nowrap;
  }

  .logs-body {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .logs-status-msg {
    padding: 40px;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.95rem;
  }

  .logs-empty {
    font-style: italic;
  }

  .logs-lines {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
    font-family: 'Consolas', 'Menlo', 'Monaco', monospace;
    font-size: 0.8rem;
  }

  .logs-line {
    display: flex;
    align-items: baseline;
    gap: 0;
    padding: 2px 0;
    line-height: 1.5;
    border-bottom: 1px solid transparent;
  }

  .logs-line:hover {
    background: var(--secondary-bg);
  }

  .logs-line-num {
    flex-shrink: 0;
    width: 52px;
    text-align: right;
    padding-right: 12px;
    color: var(--text-muted);
    user-select: none;
    font-size: 0.75rem;
  }

  .logs-line-text {
    flex: 1;
    white-space: pre-wrap;
    word-break: break-all;
    padding-right: 12px;
    color: var(--text-main);
  }

  .logs-line-error .logs-line-text {
    color: var(--error-text);
  }

  .logs-line-warn .logs-line-text {
    color: #d97706;
  }

  [data-theme="dark"] .logs-line-warn .logs-line-text {
    color: #fbbf24;
  }

  .logs-time {
    flex-shrink: 0;
    color: var(--text-muted);
    padding-right: 10px;
    font-size: 0.75rem;
  }

  .logs-badge {
    flex-shrink: 0;
    padding: 1px 7px;
    margin-right: 8px;
    border-radius: 999px;
    font-size: 0.7rem;
    font-weight: 700;
  }
  .logs-badge-debug { background: var(--secondary-bg); color: var(--text-muted); }
  .logs-badge-info  { background: rgba(59, 130, 246, 0.12); color: #2563eb; }
  .logs-badge-warn  { background: rgba(217, 119, 6, 0.12);  color: #d97706; }
  .logs-badge-error { background: var(--error-bg);           color: var(--error-text); }

  .logs-reqid {
    flex-shrink: 0;
    margin-right: 8px;
    padding: 0 4px;
    border: none;
    background: none;
    color: var(--primary-color);
    font-family: inherit;
    font-size: 0.72rem;
    cursor: pointer;
    text-decoration: underline dotted;
  }

  .logs-reqid-chip {
    padding: 4px 10px;
    border-radius: 999px;
    border: 1px solid var(--primary-color);
    background: var(--surface-color);
    color: var(--primary-color);
    font-size: 0.78rem;
    cursor: pointer;
  }

  .logs-raw-on {
    color: var(--primary-color);
    border-color: var(--primary-color);
  }
</style>
