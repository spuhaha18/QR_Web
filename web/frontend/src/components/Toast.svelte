<script lang="ts">
  import { toasts } from '../lib/toast';
  import { AlertCircle, CheckCircle } from 'lucide-svelte';
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as t (t.id)}
      <div class="toast toast-{t.type}" class:hiding={t.hiding}>
        {#if t.type === 'error'}
          <AlertCircle size={20} style="color: var(--error-text);" />
        {:else}
          <CheckCircle size={20} style="color: var(--success-text);" />
        {/if}
        <div>
          <strong style="display:block; font-size:0.9rem;">
            {t.type === 'error' ? '오류' : '성공'}
          </strong>
          <span style="font-size:0.85rem; color: var(--text-muted);">{t.message}</span>
        </div>
      </div>
    {/each}
  </div>
{/if}
