<script lang="ts">
  import { dndzone } from 'svelte-dnd-action';
  import { flip } from 'svelte/animate';
  import { qrItems, removeItem, type QrItem } from '../lib/qrStore';

  // Display-ordered list (dnd order). Bound out to App for submission.
  export let displayItems: QrItem[] = [];
  // Required volume count (권수) for the counter.
  export let docCount = 1;

  const flipDuration = 150;

  // Reconcile display order with the store (insertion order). New items are
  // appended; removed items are dropped. Existing display order is preserved.
  $: {
    const store = $qrItems;
    const byId = new Map(store.map((it) => [it.id, it]));
    const kept = displayItems.filter((it) => byId.has(it.id));
    const keptIds = new Set(kept.map((it) => it.id));
    const added = store.filter((it) => !keptIds.has(it.id));
    const next = [...kept, ...added];
    // Avoid infinite loop: only assign when changed.
    if (
      next.length !== displayItems.length ||
      next.some((it, i) => it.id !== displayItems[i]?.id)
    ) {
      // Refresh references too (url/blob) from the store.
      displayItems = next.map((it) => byId.get(it.id) ?? it);
    }
  }

  function handleDnd(e: CustomEvent<{ items: QrItem[] }>) {
    displayItems = e.detail.items;
  }

  $: count = displayItems.length;
  $: counterClass =
    count === docCount ? 'ok' : count > docCount ? 'over' : 'under';
</script>

<div class="qr-counter {counterClass}">{count} / {docCount}</div>

<ul
  class="qr-thumbnails"
  use:dndzone={{ items: displayItems, flipDurationMs: flipDuration, type: 'qr' }}
  on:consider={handleDnd}
  on:finalize={handleDnd}
>
  {#each displayItems as item, i (item.id)}
    <li class="qr-thumb-item" animate:flip={{ duration: flipDuration }}>
      <div class="qr-thumb-image">
        <img src={item.url} alt={`QR ${i + 1}`} />
        <button
          type="button"
          class="qr-remove-btn"
          on:click={() => removeItem(item.id)}
          aria-label="삭제"
        >
          ×
        </button>
      </div>
      <span class="qr-thumb-label">{i + 1}권</span>
    </li>
  {/each}
</ul>
